// Package core provides the core engine and configuration management
// for the Kilt dotfiles manager. It includes configuration parsing,
// state management, backup operations, template rendering, and the
// main orchestration engine.
package core

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	DotfilesRepo   string                 `yaml:"dotfiles_repo"`
	DotfilesPath   string                 `yaml:"dotfiles_path"`
	DataFile       string                 `yaml:"data_file"`
	TemplateEngine string                 `yaml:"template_engine"`
	Dotfiles       []DotfileEntry         `yaml:"dotfiles"`
	ExtraRepos     []Repository           `yaml:"extra_repos"`
	Directories    []string               `yaml:"directories"`
	RunOnce        []string               `yaml:"run_once"`
	OnChange       []string               `yaml:"on_change"`
	Plugins        map[string]interface{} `yaml:"plugins"`
}

// DotfileEntry represents a dotfile or directory to sync
// Supports both simple form (just directory name) and complex form (explicit mapping)
type DotfileEntry struct {
	// Directory is the simple form: just a directory name (e.g., "zsh", "git")
	// When set, all files in this directory are linked to home
	Directory string `yaml:"directory,omitempty"`

	// Source is for explicit source path mapping
	Source string `yaml:"source,omitempty"`

	// Target is the explicit target path (default: ~ for directory mode)
	Target string `yaml:"target,omitempty"`

	// Template indicates if the file should be rendered as a template
	Template bool `yaml:"template,omitempty"`

	// Mode is the file permissions in octal format (e.g., "0644")
	Mode string `yaml:"mode,omitempty"`
}

// UnmarshalYAML implements custom YAML unmarshaling to support both string and map forms
func (d *DotfileEntry) UnmarshalYAML(node *yaml.Node) error {
	// Handle simple string form: "- zsh"
	if node.Kind == yaml.ScalarNode {
		d.Directory = node.Value
		return nil
	}

	// Handle map form: "- source: ..., target: ..."
	if node.Kind == yaml.MappingNode {
		type rawDotfileEntry DotfileEntry
		var raw rawDotfileEntry
		if err := node.Decode(&raw); err != nil {
			return err
		}
		*d = DotfileEntry(raw)
		return nil
	}

	return fmt.Errorf("invalid dotfile entry: expected string or map")
}

// Repository represents an extra Git repository to clone
type Repository struct {
	URL    string `yaml:"url"`
	Path   string `yaml:"path"`
	Branch string `yaml:"branch,omitempty"`
	Sparse bool   `yaml:"sparse,omitempty"`
}

// DefaultDotfilesPath returns the default dotfiles repository path
func DefaultDotfilesPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "~/.dotfiles"
	}
	return filepath.Join(homeDir, ".dotfiles")
}

// LoadConfig loads and parses a YAML configuration file
func LoadConfig(path string) (*Config, error) {
	//nolint:gosec // G304: path comes from user config file location (validated), not arbitrary user input
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Set defaults
	if cfg.DotfilesPath == "" {
		cfg.DotfilesPath = "~/.dotfiles"
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
	// Check current directory first: .kilt/config.yaml
	cwd, err := os.Getwd()
	if err == nil {
		localConfig := filepath.Join(cwd, ".kilt", "config.yaml")
		if _, err := os.Stat(localConfig); err == nil {
			return localConfig, nil
		}
	}

	// Check home directory: ~/.kilt/config.yaml
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	homeConfig := filepath.Join(homeDir, ".kilt", "config.yaml")
	if _, err := os.Stat(homeConfig); err == nil {
		return homeConfig, nil
	}

	// Check dotfiles directory: ~/.dotfiles/.kilt/config.yaml
	dotfilesConfig := filepath.Join(homeDir, ".dotfiles", ".kilt", "config.yaml")
	if _, err := os.Stat(dotfilesConfig); err == nil {
		return dotfilesConfig, nil
	}

	return "", fmt.Errorf("config file not found in .kilt/config.yaml, ~/.kilt/config.yaml, or ~/.dotfiles/.kilt/config.yaml")
}

// ExpandPaths expands ~ and environment variables in all path fields
func ExpandPaths(cfg *Config) error {
	// Expand dotfiles path
	if cfg.DotfilesPath != "" {
		expanded, err := ExpandPath(cfg.DotfilesPath)
		if err != nil {
			return fmt.Errorf("failed to expand dotfiles_path: %w", err)
		}
		cfg.DotfilesPath = expanded
	}

	// Expand data file path
	if cfg.DataFile != "" {
		expanded, err := ExpandPath(cfg.DataFile)
		if err != nil {
			return fmt.Errorf("failed to expand data_file path: %w", err)
		}
		cfg.DataFile = expanded
	}

	// Expand dotfile entries
	for i := range cfg.Dotfiles {
		if cfg.Dotfiles[i].Source != "" {
			expanded, err := ExpandPath(cfg.Dotfiles[i].Source)
			if err != nil {
				return fmt.Errorf("failed to expand dotfile source path: %w", err)
			}
			cfg.Dotfiles[i].Source = expanded
		}

		if cfg.Dotfiles[i].Target != "" {
			expanded, err := ExpandPath(cfg.Dotfiles[i].Target)
			if err != nil {
				return fmt.Errorf("failed to expand dotfile target path: %w", err)
			}
			cfg.Dotfiles[i].Target = expanded
		}
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

	// Validate dotfile entries
	for i, dotfile := range cfg.Dotfiles {
		// Must have either Directory or Source set
		if dotfile.Directory == "" && dotfile.Source == "" {
			return fmt.Errorf("dotfiles[%d]: must have either directory or source set", i)
		}

		// If Source is set, Target is required for explicit mappings
		if dotfile.Source != "" && dotfile.Target == "" {
			return fmt.Errorf("dotfiles[%d]: target is required when source is specified", i)
		}

		// Validate file mode if provided
		if dotfile.Mode != "" {
			if !isValidFileMode(dotfile.Mode) {
				return fmt.Errorf("dotfiles[%d]: invalid file mode: %s (must be octal like 0644)", i, dotfile.Mode)
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
	if override.DotfilesRepo != "" {
		merged.DotfilesRepo = override.DotfilesRepo
	} else {
		merged.DotfilesRepo = base.DotfilesRepo
	}

	if override.DotfilesPath != "" {
		merged.DotfilesPath = override.DotfilesPath
	} else {
		merged.DotfilesPath = base.DotfilesPath
	}

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
	merged.Dotfiles = append(base.Dotfiles, override.Dotfiles...)
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
	sb.WriteString("This document describes the complete YAML configuration schema for Kilt.\n\n")

	sb.WriteString("## Global Configuration\n\n")
	sb.WriteString("| Field | Type | Required | Description |\n")
	sb.WriteString("|-------|------|----------|-------------|\n")
	sb.WriteString("| `dotfiles_repo` | string | Yes (for init) | Git repository URL for dotfiles |\n")
	sb.WriteString("| `dotfiles_path` | string | No | Local path to clone dotfiles (default: `~/.dotfiles`) |\n")
	sb.WriteString("| `data_file` | string | No | Path to custom data file for templates |\n")
	sb.WriteString("| `template_engine` | string | No | Template engine to use (currently only `\"go\"`) |\n\n")

	sb.WriteString("## Dotfiles\n\n")
	sb.WriteString("Dotfiles to sync are defined as a list. Each entry can be:\n")
	sb.WriteString("- A simple string (directory name): All files in that directory are linked to home\n")
	sb.WriteString("- A map with explicit source/target mapping\n\n")
	sb.WriteString("```yaml\ndotfiles:\n  - zsh              # Links all files in zsh/ to ~/\n  - git\n  - source: ssh/config\n    target: ~/.ssh/config\n    mode: \"0600\"\n```\n\n")

	sb.WriteString("### Dotfile Entry Fields\n\n")
	sb.WriteString("| Field | Type | Required | Description |\n")
	sb.WriteString("|-------|------|----------|-------------|\n")
	sb.WriteString("| `directory` | string | No | Directory name to sync (simple form) |\n")
	sb.WriteString("| `source` | string | No | Explicit source file path |\n")
	sb.WriteString("| `target` | string | No | Explicit target path (required if source is set) |\n")
	sb.WriteString("| `template` | boolean | No | Whether to render as template (default: `false`) |\n")
	sb.WriteString("| `mode` | string | No | File permissions in octal format (e.g., `\"0644\"`) |\n\n")

	sb.WriteString("## Extra Repositories\n\n")
	sb.WriteString("Extra repositories to clone.\n\n")
	sb.WriteString("```yaml\nextra_repos:\n  - url: https://github.com/user/repo\n    path: ~/projects/repo\n    branch: main\n```\n\n")

	sb.WriteString("## Directories\n\n")
	sb.WriteString("Directories to create if they don't exist.\n\n")
	sb.WriteString("```yaml\ndirectories:\n  - ~/dev/personal\n  - ~/dev/work\n```\n\n")

	sb.WriteString("## Run Once Scripts\n\n")
	sb.WriteString("Scripts executed exactly once per machine (order preserved).\n\n")
	sb.WriteString("```yaml\nrun_once:\n  - scripts/install_homebrew.sh\n  - scripts/setup_ssh_keys.sh\n```\n\n")

	sb.WriteString("## On Change Commands\n\n")
	sb.WriteString("Commands executed when files change.\n\n")
	sb.WriteString("```yaml\non_change:\n  - brew bundle --file=Brewfile\n  - mise install --yes\n```\n\n")

	sb.WriteString("## Plugin Configuration\n\n")
	sb.WriteString("Plugin-specific configuration.\n\n")
	sb.WriteString("```yaml\nplugins:\n  git:\n    auto_pull: true\n  brew:\n    bundles:\n      - bootstrap\n      - dev\n```\n\n")

	return sb.String()
}
