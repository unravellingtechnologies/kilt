package core

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// TemplateEngine handles template rendering with built-in functions and custom data
type TemplateEngine struct {
	funcMap    template.FuncMap
	customData map[string]interface{}
	leftDelim  string
	rightDelim string
	opFunction func(string) (string, error) // 1Password function (optional, set by plugin)
}

// TemplateData contains data available to templates
type TemplateData struct {
	Hostname string
	OS       string
	Arch     string
	User     string
	Home     string
	Custom   map[string]interface{}
}

// NewTemplateEngine creates a new template engine with default settings
func NewTemplateEngine() *TemplateEngine {
	te := &TemplateEngine{
		funcMap:    make(template.FuncMap),
		customData: make(map[string]interface{}),
		leftDelim:  "{{",
		rightDelim: "}}",
	}

	// Register built-in functions
	te.registerBuiltinFunctions()

	return te
}

// registerBuiltinFunctions registers all built-in template functions
func (te *TemplateEngine) registerBuiltinFunctions() {
	te.funcMap["env"] = func(key string) string {
		return os.Getenv(key)
	}

	te.funcMap["op"] = func(path string) (string, error) {
		if te.opFunction == nil {
			return "", fmt.Errorf("1Password function not registered. Install and configure the 1Password plugin")
		}
		return te.opFunction(path)
	}
}

// SetDelimiters sets custom template delimiters
func (te *TemplateEngine) SetDelimiters(left, right string) {
	te.leftDelim = left
	te.rightDelim = right
}

// RegisterFunction registers a custom template function
func (te *TemplateEngine) RegisterFunction(name string, fn interface{}) {
	te.funcMap[name] = fn
}

// RegisterOPFunction registers the 1Password function (called by 1Password plugin)
func (te *TemplateEngine) RegisterOPFunction(fn func(string) (string, error)) {
	te.opFunction = fn
}

// LoadCustomData loads custom data from a YAML file
func (te *TemplateEngine) LoadCustomData(dataPath string) error {
	if dataPath == "" {
		return nil
	}

	// Expand path if needed
	expandedPath, err := ExpandPath(dataPath)
	if err != nil {
		return fmt.Errorf("failed to expand data file path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		return fmt.Errorf("data file not found: %s", expandedPath)
	}

	// Read file
	data, err := os.ReadFile(expandedPath)
	if err != nil {
		return fmt.Errorf("failed to read data file: %w", err)
	}

	// Determine file type by extension
	ext := strings.ToLower(filepath.Ext(expandedPath))
	switch ext {
	case ".yaml", ".yml":
		return te.loadYAMLData(data)
	default:
		return fmt.Errorf("unsupported data file format: %s (supported: .yaml, .yml)", ext)
	}
}

// loadYAMLData loads data from YAML format
func (te *TemplateEngine) loadYAMLData(data []byte) error {
	var customData map[string]interface{}
	if err := yaml.Unmarshal(data, &customData); err != nil {
		return fmt.Errorf("failed to parse YAML data: %w", err)
	}

	// Merge with existing custom data
	for k, v := range customData {
		te.customData[k] = v
	}

	return nil
}

// buildTemplateData builds the complete template data structure
func (te *TemplateEngine) buildTemplateData() (*TemplateData, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}

	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	return &TemplateData{
		Hostname: hostname,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		User:     currentUser.Username,
		Home:     homeDir,
		Custom:   te.customData,
	}, nil
}

// RenderString renders a template string with the provided data
func (te *TemplateEngine) RenderString(templateStr string) (string, error) {
	// Build template data
	data, err := te.buildTemplateData()
	if err != nil {
		return "", fmt.Errorf("failed to build template data: %w", err)
	}

	// Parse template
	tmpl, err := template.New("template").
		Delims(te.leftDelim, te.rightDelim).
		Funcs(te.funcMap).
		Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w\nHint: Check template syntax and ensure all variables are defined", err)
	}

	// Execute template
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w\nHint: Check that all template variables and functions are valid", err)
	}

	return buf.String(), nil
}

// Render renders a template file with the provided data
func (te *TemplateEngine) Render(templatePath string) (string, error) {
	// Expand path if needed
	expandedPath, err := ExpandPath(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to expand template path: %w", err)
	}

	// Read template file
	templateStr, err := os.ReadFile(expandedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file: %w", err)
	}

	// Use RenderString to render
	return te.RenderString(string(templateStr))
}

// ValidateTemplate validates a template string without executing it
func (te *TemplateEngine) ValidateTemplate(templateStr string) error {
	// Build template data (for validation, we just need the structure)
	data, err := te.buildTemplateData()
	if err != nil {
		return fmt.Errorf("failed to build template data: %w", err)
	}

	// Parse template
	_, err = template.New("template").
		Delims(te.leftDelim, te.rightDelim).
		Funcs(te.funcMap).
		Parse(templateStr)
	if err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	// Try to execute with empty data to catch runtime errors
	// (Note: This won't catch all errors, but will catch syntax issues)
	testTmpl, _ := template.New("test").
		Delims(te.leftDelim, te.rightDelim).
		Funcs(te.funcMap).
		Parse(templateStr)

	var buf strings.Builder
	if err := testTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("template execution test failed: %w\nHint: Check that all template variables exist and functions are called correctly", err)
	}

	return nil
}

// GetCustomData returns the current custom data
func (te *TemplateEngine) GetCustomData() map[string]interface{} {
	// Return a copy to prevent external modification
	result := make(map[string]interface{})
	for k, v := range te.customData {
		result[k] = v
	}
	return result
}
