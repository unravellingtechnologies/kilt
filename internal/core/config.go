package core

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config represents the main configuration structure
type Config struct {
	DataFile       string                 `toml:"data_file"`
	TemplateEngine string                 `toml:"template_engine"`
	Files          []FileMapping          `toml:"files"`
	ExtraRepos     []Repository           `toml:"extra_repos"`
	Directories    []string               `toml:"directories"`
	RunOnce        []string               `toml:"run_once"`
	OnChange       []string               `toml:"on_change"`
	Plugins        map[string]interface{} `toml:"plugins"`
}

// FileMapping represents a source to target file mapping
type FileMapping struct {
	Source   string `toml:"source"`
	Target   string `toml:"target"`
	Template bool   `toml:"template"`
	Mode     string `toml:"mode"` // file permissions (e.g., "0644")
}

// Repository represents an extra Git repository to clone
type Repository struct {
	URL    string `toml:"url"`
	Path   string `toml:"path"`
	Branch string `toml:"branch"`
	Sparse bool   `toml:"sparse"`
}

// LoadConfig loads and parses a TOML configuration file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var cfg Config
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse TOML config: %w", err)
	}

	// Expand paths after loading
	if err := ExpandPaths(&cfg); err != nil {
		return nil, fmt.Errorf("failed to expand paths: %w", err)
	}

	// Validate configuration
	if err := ValidateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// FindConfigFile discovers the configuration file in standard locations
// Returns the path to the config file if found, or an error if not found
func FindConfigFile() (string, error) {
	// Check current directory first: .kilt/config.toml
	cwd, err := os.Getwd()
	if err == nil {
		localConfig := filepath.Join(cwd, ".kilt", "config.toml")
		if _, err := os.Stat(localConfig); err == nil {
			return localConfig, nil
		}
	}

	// Check home directory: ~/.kilt/config.toml
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	homeConfig := filepath.Join(homeDir, ".kilt", "config.toml")
	if _, err := os.Stat(homeConfig); err == nil {
		return homeConfig, nil
	}

	return "", fmt.Errorf("config file not found in .kilt/config.toml or ~/.kilt/config.toml")
}

// ExpandPaths expands ~ and environment variables in all path fields
func ExpandPaths(cfg *Config) error {
	// Expand data file path
	if cfg.DataFile != "" {
		expanded, err := ExpandPath(cfg.DataFile)
		if err != nil {
			return fmt.Errorf("failed to expand data_file path: %w", err)
		}
		cfg.DataFile = expanded
	}

	// Expand file mappings
	for i := range cfg.Files {
		expanded, err := ExpandPath(cfg.Files[i].Source)
		if err != nil {
			return fmt.Errorf("failed to expand file source path: %w", err)
		}
		cfg.Files[i].Source = expanded

		expanded, err = ExpandPath(cfg.Files[i].Target)
		if err != nil {
			return fmt.Errorf("failed to expand file target path: %w", err)
		}
		cfg.Files[i].Target = expanded
	}

	// Expand repository paths
	for i := range cfg.ExtraRepos {
		expanded, err := ExpandPath(cfg.ExtraRepos[i].Path)
		if err != nil {
			return fmt.Errorf("failed to expand repository path: %w", err)
		}
		cfg.ExtraRepos[i].Path = expanded
	}

	// Expand directory paths
	for i := range cfg.Directories {
		expanded, err := ExpandPath(cfg.Directories[i])
		if err != nil {
			return fmt.Errorf("failed to expand directory path: %w", err)
		}
		cfg.Directories[i] = expanded
	}

	// Expand run_once script paths
	for i := range cfg.RunOnce {
		expanded, err := ExpandPath(cfg.RunOnce[i])
		if err != nil {
			return fmt.Errorf("failed to expand run_once path: %w", err)
		}
		cfg.RunOnce[i] = expanded
	}

	// Expand plugin config paths (recursively)
	if err := expandPluginPaths(cfg.Plugins); err != nil {
		return fmt.Errorf("failed to expand plugin paths: %w", err)
	}

	return nil
}

// expandPluginPaths recursively expands paths in plugin configuration
func expandPluginPaths(plugins map[string]interface{}) error {
	for pluginName, pluginConfig := range plugins {
		if pluginMap, ok := pluginConfig.(map[string]interface{}); ok {
			for key, value := range pluginMap {
				if strValue, ok := value.(string); ok {
					// Check if it looks like a path
					if strings.HasPrefix(strValue, "~") || strings.Contains(strValue, "${") || strings.Contains(strValue, "$(") {
						expanded, err := ExpandPath(strValue)
						if err != nil {
							return fmt.Errorf("failed to expand plugin %s path %s: %w", pluginName, key, err)
						}
						pluginMap[key] = expanded
					}
				} else if nestedMap, ok := value.(map[string]interface{}); ok {
					// Recursively handle nested maps
					if err := expandPluginPaths(map[string]interface{}{"nested": nestedMap}); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// ExpandPath expands ~ and environment variables in a path string
func ExpandPath(path string) (string, error) {
	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(homeDir, path[2:])
	} else if path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = homeDir
	}

	// Expand ${VAR} and $VAR environment variables
	expanded := os.ExpandEnv(path)

	// Also handle $(VAR) syntax
	re := regexp.MustCompile(`\$\(([^)]+)\)`)
	expanded = re.ReplaceAllStringFunc(expanded, func(match string) string {
		varName := match[2 : len(match)-1] // Remove $( and )
		if val := os.Getenv(varName); val != "" {
			return val
		}
		return match // Return original if not found
	})

	return expanded, nil
}

// ValidateConfig validates the configuration structure
func ValidateConfig(cfg *Config) error {
	// Validate template engine
	if cfg.TemplateEngine != "" && cfg.TemplateEngine != "go" {
		return fmt.Errorf("unsupported template_engine: %s (only 'go' is supported)", cfg.TemplateEngine)
	}

	// Validate file mappings
	for i, file := range cfg.Files {
		if file.Source == "" {
			return fmt.Errorf("files[%d]: source is required", i)
		}
		if file.Target == "" {
			return fmt.Errorf("files[%d]: target is required", i)
		}

		// Validate file mode if provided
		if file.Mode != "" {
			if !isValidFileMode(file.Mode) {
				return fmt.Errorf("files[%d]: invalid file mode: %s (must be octal like 0644)", i, file.Mode)
			}
		}
	}

	// Validate repositories
	for i, repo := range cfg.ExtraRepos {
		if repo.URL == "" {
			return fmt.Errorf("extra_repos[%d]: url is required", i)
		}
		if repo.Path == "" {
			return fmt.Errorf("extra_repos[%d]: path is required", i)
		}
	}

	// Validate directories
	for i, dir := range cfg.Directories {
		if dir == "" {
			return fmt.Errorf("directories[%d]: path cannot be empty", i)
		}
	}

	// Validate run_once scripts
	for i, script := range cfg.RunOnce {
		if script == "" {
			return fmt.Errorf("run_once[%d]: script path cannot be empty", i)
		}
	}

	// Validate on_change commands
	for i, cmd := range cfg.OnChange {
		if cmd == "" {
			return fmt.Errorf("on_change[%d]: command cannot be empty", i)
		}
	}

	// Check for circular dependencies in file mappings
	if err := checkCircularDependencies(cfg.Files); err != nil {
		return fmt.Errorf("circular dependency detected: %w", err)
	}

	return nil
}

// isValidFileMode checks if a file mode string is valid (octal format)
func isValidFileMode(mode string) bool {
	// Should be 3-4 digits in octal format (e.g., "0644", "755")
	// 3-digit modes should not start with 0 (e.g., "755", "644")
	// 4-digit modes should start with 0 (e.g., "0644", "0755")
	if len(mode) == 3 {
		// 3 digits, first digit should be 1-7 (not 0)
		matched, _ := regexp.MatchString(`^[1-7][0-7]{2}$`, mode)
		return matched
	}
	if len(mode) == 4 {
		// 4 digits, must start with 0
		matched, _ := regexp.MatchString(`^0[0-7]{3}$`, mode)
		return matched
	}
	return false
}

// checkCircularDependencies checks for circular dependencies in file mappings
// This is a simplified check - in a real scenario, we might need to check
// if any file's target is another file's source, creating a cycle
func checkCircularDependencies(files []FileMapping) error {
	// Build a map of targets
	targets := make(map[string]int)
	for i, file := range files {
		targets[file.Target] = i
	}

	// Check if any source is also a target (potential cycle)
	for i, file := range files {
		if targetIdx, exists := targets[file.Source]; exists {
			return fmt.Errorf("file[%d].source (%s) conflicts with file[%d].target", i, file.Source, targetIdx)
		}
	}

	return nil
}

// GetPluginConfig returns plugin-specific configuration
// This implements the plugin.Config interface
func (c *Config) GetPluginConfig(pluginName string) map[string]interface{} {
	if c.Plugins == nil {
		return nil
	}

	if pluginConfig, ok := c.Plugins[pluginName]; ok {
		if configMap, ok := pluginConfig.(map[string]interface{}); ok {
			return configMap
		}
		// If it's not a map, return nil
		return nil
	}

	return nil
}

// MergeConfigs merges two configurations, with override taking precedence
func MergeConfigs(base, override *Config) *Config {
	merged := &Config{}

	// Merge simple fields (override takes precedence)
	if override.DataFile != "" {
		merged.DataFile = override.DataFile
	} else {
		merged.DataFile = base.DataFile
	}

	if override.TemplateEngine != "" {
		merged.TemplateEngine = override.TemplateEngine
	} else {
		merged.TemplateEngine = base.TemplateEngine
	}

	// Merge slices (append override to base)
	merged.Files = append(base.Files, override.Files...)
	merged.ExtraRepos = append(base.ExtraRepos, override.ExtraRepos...)
	merged.Directories = append(base.Directories, override.Directories...)
	merged.RunOnce = append(base.RunOnce, override.RunOnce...)
	merged.OnChange = append(base.OnChange, override.OnChange...)

	// Merge plugin configs (override takes precedence for each plugin)
	merged.Plugins = make(map[string]interface{})
	if base.Plugins != nil {
		for k, v := range base.Plugins {
			merged.Plugins[k] = v
		}
	}
	if override.Plugins != nil {
		for k, v := range override.Plugins {
			merged.Plugins[k] = v
		}
	}

	return merged
}

// GenerateSchemaDoc generates a markdown documentation of the configuration schema
func GenerateSchemaDoc() string {
	var sb strings.Builder

	sb.WriteString("# Kilt Configuration Schema\n\n")
	sb.WriteString("This document describes the complete TOML configuration schema for Kilt.\n\n")

	sb.WriteString("## Global Configuration\n\n")
	sb.WriteString("| Field | Type | Required | Description |\n")
	sb.WriteString("|-------|------|----------|-------------|\n")
	sb.WriteString("| `data_file` | string | No | Path to custom data file for templates |\n")
	sb.WriteString("| `template_engine` | string | No | Template engine to use (currently only `\"go\"`) |\n\n")

	sb.WriteString("## File Mappings\n\n")
	sb.WriteString("File mappings are defined using `[[files]]` array of tables.\n\n")
	sb.WriteString("| Field | Type | Required | Description |\n")
	sb.WriteString("|-------|------|----------|-------------|\n")
	sb.WriteString("| `source` | string | Yes | Source file path (relative to repo root) |\n")
	sb.WriteString("| `target` | string | Yes | Target file path (supports `~` expansion) |\n")
	sb.WriteString("| `template` | boolean | No | Whether to render as template (default: `false`) |\n")
	sb.WriteString("| `mode` | string | No | File permissions in octal format (e.g., `\"0644\"`) |\n\n")

	sb.WriteString("## Extra Repositories\n\n")
	sb.WriteString("Extra repositories are defined using `[[extra_repos]]` array of tables.\n\n")
	sb.WriteString("| Field | Type | Required | Description |\n")
	sb.WriteString("|-------|------|----------|-------------|\n")
	sb.WriteString("| `url` | string | Yes | Git repository URL |\n")
	sb.WriteString("| `path` | string | Yes | Local path to clone to (supports `~` expansion) |\n")
	sb.WriteString("| `branch` | string | No | Branch to checkout (default: default branch) |\n")
	sb.WriteString("| `sparse` | boolean | No | Enable sparse checkout (default: `false`) |\n\n")

	sb.WriteString("## Directories\n\n")
	sb.WriteString("Directories are defined as an array of strings. Each directory will be created if it doesn't exist.\n\n")
	sb.WriteString("```toml\ndirectories = [\n    \"~/dev/personal\",\n    \"~/dev/work\"\n]\n```\n\n")

	sb.WriteString("## Run Once Scripts\n\n")
	sb.WriteString("Run once scripts are executed exactly once per machine. Defined as an array of script paths.\n\n")
	sb.WriteString("```toml\nrun_once = [\n    \"scripts/install_homebrew.sh\",\n    \"scripts/setup_ssh_keys.sh\"\n]\n```\n\n")

	sb.WriteString("## On Change Commands\n\n")
	sb.WriteString("On change commands are executed when configuration or files change. Defined as an array of command strings.\n\n")
	sb.WriteString("```toml\non_change = [\n    \"brew bundle --file=Brewfile\",\n    \"mise install --yes\"\n]\n```\n\n")

	sb.WriteString("## Plugin Configuration\n\n")
	sb.WriteString("Plugin-specific configuration is defined using `[plugins.<name>]` sections.\n\n")
	sb.WriteString("### Example Plugin Configurations\n\n")
	sb.WriteString("```toml\n[plugins.git]\nbare_repo_path = \"~/.kilt/repo\"\nauto_pull = true\n\n[plugins.brew]\nbrewfile = \"Brewfile\"\nauto_update = false\n\n[plugins.onepassword]\naccount = \"my.1password.com\"\ncache_ttl = 3600\n```\n\n")

	sb.WriteString("## Path Expansion\n\n")
	sb.WriteString("Kilt supports path expansion in the following ways:\n\n")
	sb.WriteString("1. **Home directory expansion**: Use `~` or `~/` prefix\n")
	sb.WriteString("   - `~/.zshrc` expands to `/home/user/.zshrc`\n")
	sb.WriteString("2. **Environment variables**: Use `${VAR}` or `$VAR` syntax\n")
	sb.WriteString("   - `${HOME}/config` expands using the `HOME` environment variable\n")
	sb.WriteString("3. **Combined**: `~/${DEV_DIR}/project` combines both expansions\n\n")

	return sb.String()
}

