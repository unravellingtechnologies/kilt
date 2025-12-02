package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewTemplateEngine(t *testing.T) {
	te := NewTemplateEngine()
	if te == nil {
		t.Fatal("NewTemplateEngine returned nil")
	}

	if te.funcMap == nil {
		t.Error("funcMap should not be nil")
	}

	if te.customData == nil {
		t.Error("customData should not be nil")
	}

	if te.leftDelim != "{{" || te.rightDelim != "}}" {
		t.Errorf("Expected default delimiters {{ }}, got %s %s", te.leftDelim, te.rightDelim)
	}
}

func TestTemplateEngine_RenderString_Basic(t *testing.T) {
	te := NewTemplateEngine()

	templateStr := "Hello, {{ .User }}!"
	result, err := te.RenderString(templateStr)
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}

	if !strings.Contains(result, "Hello") {
		t.Errorf("Expected result to contain 'Hello', got: %s", result)
	}
}

func TestTemplateEngine_RenderString_SystemVariables(t *testing.T) {
	te := NewTemplateEngine()

	tests := []struct {
		template string
		check    func(string) bool
		name     string
	}{
		{"{{ .Hostname }}", func(s string) bool { return s != "" }, "hostname"},
		{"{{ .OS }}", func(s string) bool { return s == "darwin" || s == "linux" || s == "windows" }, "os"},
		{"{{ .Arch }}", func(s string) bool { return s == "amd64" || s == "arm64" }, "arch"},
		{"{{ .User }}", func(s string) bool { return s != "" }, "user"},
		{"{{ .Home }}", func(s string) bool { return strings.HasPrefix(s, "/") || strings.Contains(s, "\\") }, "home"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := te.RenderString(tt.template)
			if err != nil {
				t.Fatalf("RenderString failed: %v", err)
			}
			if !tt.check(result) {
				t.Errorf("Template %q produced unexpected result: %s", tt.template, result)
			}
		})
	}
}

func TestTemplateEngine_RenderString_EnvFunction(t *testing.T) {
	te := NewTemplateEngine()

	// Set a test environment variable
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	templateStr := "Value: {{ env \"TEST_VAR\" }}"
	result, err := te.RenderString(templateStr)
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}

	if !strings.Contains(result, "test_value") {
		t.Errorf("Expected result to contain 'test_value', got: %s", result)
	}
}

func TestTemplateEngine_RenderString_CustomData(t *testing.T) {
	te := NewTemplateEngine()
	te.customData["name"] = "TestUser"
	te.customData["email"] = "test@example.com"

	templateStr := "Name: {{ .Custom.name }}, Email: {{ .Custom.email }}"
	result, err := te.RenderString(templateStr)
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}

	if !strings.Contains(result, "TestUser") {
		t.Errorf("Expected result to contain 'TestUser', got: %s", result)
	}
	if !strings.Contains(result, "test@example.com") {
		t.Errorf("Expected result to contain 'test@example.com', got: %s", result)
	}
}

func TestTemplateEngine_LoadCustomData_YAML(t *testing.T) {
	te := NewTemplateEngine()

	// Create temporary YAML file
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "data.yaml")
	yamlContent := `name: TestUser
email: test@example.com
settings:
  theme: dark
`
	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("Failed to write YAML file: %v", err)
	}

	// Load custom data
	if err := te.LoadCustomData(yamlFile); err != nil {
		t.Fatalf("LoadCustomData failed: %v", err)
	}

	// Verify data was loaded
	if te.customData["name"] != "TestUser" {
		t.Errorf("Expected name to be 'TestUser', got: %v", te.customData["name"])
	}
}

func TestTemplateEngine_LoadCustomData_InvalidFormat(t *testing.T) {
	te := NewTemplateEngine()

	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "data.json")

	if err := te.LoadCustomData(invalidFile); err == nil {
		t.Error("Expected error for unsupported file format")
	}
}

func TestTemplateEngine_LoadCustomData_NonExistentFile(t *testing.T) {
	te := NewTemplateEngine()

	if err := te.LoadCustomData("/nonexistent/file.yaml"); err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestTemplateEngine_Render_File(t *testing.T) {
	te := NewTemplateEngine()

	// Create temporary template file
	tmpDir := t.TempDir()
	templateFile := filepath.Join(tmpDir, "template.txt")
	templateContent := "Hello, {{ .User }}! Your home is {{ .Home }}."
	if err := os.WriteFile(templateFile, []byte(templateContent), 0o644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	result, err := te.Render(templateFile)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "Hello") {
		t.Errorf("Expected result to contain 'Hello', got: %s", result)
	}
}

func TestTemplateEngine_SetDelimiters(t *testing.T) {
	te := NewTemplateEngine()
	te.SetDelimiters("[[", "]]")

	if te.leftDelim != "[[" || te.rightDelim != "]]" {
		t.Errorf("Expected delimiters [[ ]], got %s %s", te.leftDelim, te.rightDelim)
	}

	// Test rendering with custom delimiters
	templateStr := "Hello, [[ .User ]]!"
	result, err := te.RenderString(templateStr)
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}

	if !strings.Contains(result, "Hello") {
		t.Errorf("Expected result to contain 'Hello', got: %s", result)
	}
}

func TestTemplateEngine_RegisterFunction(t *testing.T) {
	te := NewTemplateEngine()

	// Register custom function
	te.RegisterFunction("uppercase", func(s string) string {
		return strings.ToUpper(s)
	})

	templateStr := "{{ uppercase \"hello\" }}"
	result, err := te.RenderString(templateStr)
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}

	if result != "HELLO" {
		t.Errorf("Expected 'HELLO', got: %s", result)
	}
}

func TestTemplateEngine_ValidateTemplate_Valid(t *testing.T) {
	te := NewTemplateEngine()

	templateStr := "Hello, {{ .User }}!"
	if err := te.ValidateTemplate(templateStr); err != nil {
		t.Errorf("ValidateTemplate should pass for valid template, got error: %v", err)
	}
}

func TestTemplateEngine_ValidateTemplate_Invalid(t *testing.T) {
	te := NewTemplateEngine()

	// Invalid template syntax
	templateStr := "Hello, {{ .User }"
	if err := te.ValidateTemplate(templateStr); err == nil {
		t.Error("ValidateTemplate should fail for invalid template")
	}
}

func TestTemplateEngine_ValidateTemplate_UndefinedVariable(t *testing.T) {
	te := NewTemplateEngine()

	// Template with undefined variable (this will pass validation but fail execution)
	// Note: Go templates don't fail on undefined variables at parse time
	templateStr := "Hello, {{ .UndefinedVar }}!"
	// This should not error during validation (Go templates are lenient)
	err := te.ValidateTemplate(templateStr)
	// Validation might pass, but execution will show the issue
	if err != nil {
		// If validation catches it, that's fine
		t.Logf("Validation caught undefined variable (expected): %v", err)
	}
}

func TestTemplateEngine_OPFunction_NotRegistered(t *testing.T) {
	te := NewTemplateEngine()

	templateStr := "Secret: {{ op \"path/to/secret\" }}"
	_, err := te.RenderString(templateStr)
	if err == nil {
		t.Error("Expected error when op function is not registered")
	}
	if !strings.Contains(err.Error(), "1Password function not registered") {
		t.Errorf("Expected error about 1Password function, got: %v", err)
	}
}

func TestTemplateEngine_OPFunction_Registered(t *testing.T) {
	te := NewTemplateEngine()

	// Register mock OP function
	te.RegisterOPFunction(func(path string) (string, error) {
		return "secret_value", nil
	})

	templateStr := "Secret: {{ op \"path/to/secret\" }}"
	result, err := te.RenderString(templateStr)
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}

	if !strings.Contains(result, "secret_value") {
		t.Errorf("Expected result to contain 'secret_value', got: %s", result)
	}
}

func TestTemplateEngine_ComplexTemplate(t *testing.T) {
	te := NewTemplateEngine()

	// Load custom data
	te.customData["project"] = "Kilt"
	te.customData["version"] = "1.0.0"

	templateStr := `Project: {{ .Custom.project }}
Version: {{ .Custom.version }}
User: {{ .User }}
OS: {{ .OS }}
Arch: {{ .Arch }}
Home: {{ .Home }}
Env PATH: {{ env "PATH" }}`

	result, err := te.RenderString(templateStr)
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}

	// Check multiple expected values
	checks := []string{"Kilt", "1.0.0", "OS:", "Arch:"}
	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("Expected result to contain %q, got: %s", check, result)
		}
	}
}

func TestTemplateEngine_GetCustomData(t *testing.T) {
	te := NewTemplateEngine()
	te.customData["key1"] = "value1"
	te.customData["key2"] = "value2"

	data := te.GetCustomData()

	// Verify it's a copy (modifying returned data shouldn't affect original)
	data["key3"] = "value3"
	if te.customData["key3"] != nil {
		t.Error("GetCustomData should return a copy, not the original map")
	}

	// Verify original values are present
	if data["key1"] != "value1" || data["key2"] != "value2" {
		t.Error("GetCustomData should return all custom data")
	}
}
