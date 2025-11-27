package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary directory for test config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write a valid YAML config file
	configContent := `
data_file: "data.yaml"
template_engine: "go"
dotfiles_repo: "https://github.com/test/dotfiles"
dotfiles_path: "~/.dotfiles"

dotfiles:
  - zsh
  - git
  - source: ssh/config
    target: ~/.ssh/config
    mode: "0600"
    template: true

extra_repos:
  - url: "https://github.com/test/repo"
    path: "~/.test/repo"
    branch: "main"

directories:
  - "~/dev/personal"
  - "~/dev/work"

run_once:
  - "scripts/install.sh"

on_change:
  - "brew bundle"

plugins:
  test:
    key: "value"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}

	if cfg.DataFile == "" {
		t.Error("LoadConfig() DataFile should not be empty")
	}

	if cfg.TemplateEngine != "go" {
		t.Errorf("LoadConfig() TemplateEngine = %v, want 'go'", cfg.TemplateEngine)
	}

	if cfg.DotfilesRepo != "https://github.com/test/dotfiles" {
		t.Errorf("LoadConfig() DotfilesRepo = %v, want 'https://github.com/test/dotfiles'", cfg.DotfilesRepo)
	}

	if len(cfg.Dotfiles) != 3 {
		t.Errorf("LoadConfig() Dotfiles length = %d, want 3", len(cfg.Dotfiles))
	}

	// Check simple form (directory name)
	if cfg.Dotfiles[0].Directory != "zsh" {
		t.Errorf("LoadConfig() Dotfiles[0].Directory = %v, want 'zsh'", cfg.Dotfiles[0].Directory)
	}

	// Check complex form (source/target mapping)
	homeDir, _ := os.UserHomeDir()
	expectedTarget := filepath.Join(homeDir, ".ssh", "config")
	if cfg.Dotfiles[2].Target != expectedTarget {
		t.Errorf("LoadConfig() Dotfiles[2].Target = %v, want %v", cfg.Dotfiles[2].Target, expectedTarget)
	}

	if len(cfg.ExtraRepos) != 1 {
		t.Errorf("LoadConfig() ExtraRepos length = %d, want 1", len(cfg.ExtraRepos))
	}

	if len(cfg.Directories) != 2 {
		t.Errorf("LoadConfig() Directories length = %d, want 2", len(cfg.Directories))
	}

	if len(cfg.RunOnce) != 1 {
		t.Errorf("LoadConfig() RunOnce length = %d, want 1", len(cfg.RunOnce))
	}

	if len(cfg.OnChange) != 1 {
		t.Errorf("LoadConfig() OnChange length = %d, want 1", len(cfg.OnChange))
	}
}

func TestLoadConfig_InvalidFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("LoadConfig() should return error for nonexistent file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	invalidContent := `
invalid: yaml: content: [
`
	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("LoadConfig() should return error for invalid YAML")
	}
}

func TestFindConfigFile(t *testing.T) {
	// Test with local config
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir("/")

	// Create .kilt directory
	kiltDir := filepath.Join(tmpDir, ".kilt")
	if err := os.MkdirAll(kiltDir, 0755); err != nil {
		t.Fatalf("Failed to create .kilt directory: %v", err)
	}

	configPath := filepath.Join(kiltDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("# test config"), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	found, err := FindConfigFile()
	if err != nil {
		t.Fatalf("FindConfigFile() error = %v, want nil", err)
	}

	// Resolve symlinks for comparison (macOS uses /private/var for /var)
	foundResolved, _ := filepath.EvalSymlinks(found)
	configPathResolved, _ := filepath.EvalSymlinks(configPath)
	if foundResolved != configPathResolved {
		t.Errorf("FindConfigFile() = %v (resolved: %v), want %v (resolved: %v)", found, foundResolved, configPath, configPathResolved)
	}
}

func TestFindConfigFile_HomeDir(t *testing.T) {
	// Test with home directory config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	kiltDir := filepath.Join(homeDir, ".kilt")
	if err := os.MkdirAll(kiltDir, 0755); err != nil {
		t.Fatalf("Failed to create .kilt directory: %v", err)
	}

	configPath := filepath.Join(kiltDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("# test config"), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	defer os.Remove(configPath)

	// Change to a directory without .kilt/config.yaml
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir("/")

	found, err := FindConfigFile()
	if err != nil {
		t.Fatalf("FindConfigFile() error = %v, want nil", err)
	}

	if found != configPath {
		t.Errorf("FindConfigFile() = %v, want %v", found, configPath)
	}
}

func TestFindConfigFile_NotFound(t *testing.T) {
	// Change to a directory without config
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir("/")

	// Remove home config if it exists
	homeDir, _ := os.UserHomeDir()
	homeConfig := filepath.Join(homeDir, ".kilt", "config.yaml")
	os.Remove(homeConfig)
	dotfilesConfig := filepath.Join(homeDir, ".dotfiles", ".kilt", "config.yaml")
	os.Remove(dotfilesConfig)

	_, err := FindConfigFile()
	if err == nil {
		t.Error("FindConfigFile() should return error when config not found")
	}
}

func TestExpandPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		want     string
		wantErr  bool
		setupEnv func()
		cleanup  func()
	}{
		{
			name:  "expand ~/path",
			input: "~/.zshrc",
			want:  filepath.Join(homeDir, ".zshrc"),
		},
		{
			name:  "expand ~",
			input: "~",
			want:  homeDir,
		},
		{
			name:     "expand ${VAR}",
			input:    "${HOME}/test",
			want:     filepath.Join(homeDir, "test"),
			setupEnv: func() {},
		},
		{
			name:     "expand $VAR",
			input:    "$HOME/test",
			want:     filepath.Join(homeDir, "test"),
			setupEnv: func() {},
		},
		{
			name:     "expand $(VAR)",
			input:    "$(HOME)/test",
			want:     filepath.Join(homeDir, "test"),
			setupEnv: func() {},
		},
		{
			name:  "no expansion needed",
			input: "/absolute/path",
			want:  "/absolute/path",
		},
		{
			name:  "relative path",
			input: "relative/path",
			want:  "relative/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv()
			}
			if tt.cleanup != nil {
				defer tt.cleanup()
			}

			got, err := ExpandPath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExpandPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExpandPaths(t *testing.T) {
	homeDir, _ := os.UserHomeDir()

	cfg := &Config{
		DotfilesPath: "~/.dotfiles",
		DataFile:     "~/data.yaml",
		Dotfiles: []DotfileEntry{
			{Source: "source", Target: "~/.target"},
		},
		ExtraRepos: []Repository{
			{URL: "https://test.com", Path: "~/repo"},
		},
		Directories: []string{"~/dir1", "~/dir2"},
		RunOnce:     []string{"~/script.sh"},
	}

	err := ExpandPaths(cfg)
	if err != nil {
		t.Fatalf("ExpandPaths() error = %v, want nil", err)
	}

	if cfg.DotfilesPath != filepath.Join(homeDir, ".dotfiles") {
		t.Errorf("ExpandPaths() DotfilesPath = %v, want %v", cfg.DotfilesPath, filepath.Join(homeDir, ".dotfiles"))
	}

	if cfg.DataFile != filepath.Join(homeDir, "data.yaml") {
		t.Errorf("ExpandPaths() DataFile = %v, want %v", cfg.DataFile, filepath.Join(homeDir, "data.yaml"))
	}

	if cfg.Dotfiles[0].Target != filepath.Join(homeDir, ".target") {
		t.Errorf("ExpandPaths() Dotfiles[0].Target = %v, want %v", cfg.Dotfiles[0].Target, filepath.Join(homeDir, ".target"))
	}

	if cfg.ExtraRepos[0].Path != filepath.Join(homeDir, "repo") {
		t.Errorf("ExpandPaths() ExtraRepos[0].Path = %v, want %v", cfg.ExtraRepos[0].Path, filepath.Join(homeDir, "repo"))
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config with directory entries",
			cfg: &Config{
				TemplateEngine: "go",
				Dotfiles: []DotfileEntry{
					{Directory: "zsh"},
					{Directory: "git"},
				},
				ExtraRepos: []Repository{
					{URL: "https://test.com", Path: "path"},
				},
				Directories: []string{"dir1"},
				RunOnce:     []string{"script.sh"},
				OnChange:    []string{"command"},
			},
			wantErr: false,
		},
		{
			name: "valid config with source/target entries",
			cfg: &Config{
				Dotfiles: []DotfileEntry{
					{Source: "ssh/config", Target: "~/.ssh/config"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid template engine",
			cfg: &Config{
				TemplateEngine: "invalid",
			},
			wantErr: true,
		},
		{
			name: "empty dotfile entry",
			cfg: &Config{
				Dotfiles: []DotfileEntry{
					{}, // Neither Directory nor Source set
				},
			},
			wantErr: true,
		},
		{
			name: "source without target",
			cfg: &Config{
				Dotfiles: []DotfileEntry{
					{Source: "source/file"}, // Target missing
				},
			},
			wantErr: true,
		},
		{
			name: "invalid file mode",
			cfg: &Config{
				Dotfiles: []DotfileEntry{
					{Source: "source", Target: "target", Mode: "invalid"},
				},
			},
			wantErr: true,
		},
		{
			name: "valid file mode",
			cfg: &Config{
				Dotfiles: []DotfileEntry{
					{Source: "source", Target: "target", Mode: "0644"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty repo URL",
			cfg: &Config{
				ExtraRepos: []Repository{
					{URL: "", Path: "path"},
				},
			},
			wantErr: true,
		},
		{
			name: "empty repo path",
			cfg: &Config{
				ExtraRepos: []Repository{
					{URL: "https://test.com", Path: ""},
				},
			},
			wantErr: true,
		},
		{
			name: "empty directory",
			cfg: &Config{
				Directories: []string{""},
			},
			wantErr: true,
		},
		{
			name: "empty run_once",
			cfg: &Config{
				RunOnce: []string{""},
			},
			wantErr: true,
		},
		{
			name: "empty on_change",
			cfg: &Config{
				OnChange: []string{""},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidFileMode(t *testing.T) {
	tests := []struct {
		mode string
		want bool
	}{
		{"0644", true},
		{"0755", true},
		{"0600", true},
		{"755", true},
		{"644", true},
		{"invalid", false},
		{"9999", false},
		{"", false},
		{"064", false},
		{"06444", false},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			got := isValidFileMode(tt.mode)
			if got != tt.want {
				t.Errorf("isValidFileMode(%q) = %v, want %v", tt.mode, got, tt.want)
			}
		})
	}
}

func TestGetPluginConfig(t *testing.T) {
	cfg := &Config{
		Plugins: map[string]interface{}{
			"test": map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			},
			"other": "not a map",
		},
	}

	// Test existing plugin config
	pluginConfig := cfg.GetPluginConfig("test")
	if pluginConfig == nil {
		t.Fatal("GetPluginConfig() returned nil for existing plugin")
	}

	if pluginConfig["key1"] != "value1" {
		t.Errorf("GetPluginConfig() key1 = %v, want 'value1'", pluginConfig["key1"])
	}

	// Test non-existent plugin
	pluginConfig = cfg.GetPluginConfig("nonexistent")
	if pluginConfig != nil {
		t.Error("GetPluginConfig() should return nil for non-existent plugin")
	}

	// Test plugin with non-map value
	pluginConfig = cfg.GetPluginConfig("other")
	if pluginConfig != nil {
		t.Error("GetPluginConfig() should return nil for non-map plugin config")
	}

	// Test nil plugins map
	cfg.Plugins = nil
	pluginConfig = cfg.GetPluginConfig("test")
	if pluginConfig != nil {
		t.Error("GetPluginConfig() should return nil when Plugins is nil")
	}
}

func TestMergeConfigs(t *testing.T) {
	base := &Config{
		DotfilesRepo:   "https://github.com/base/dotfiles",
		DotfilesPath:   "~/.base_dotfiles",
		DataFile:       "base_data.yaml",
		TemplateEngine: "go",
		Dotfiles: []DotfileEntry{
			{Directory: "base_zsh"},
		},
		ExtraRepos: []Repository{
			{URL: "https://base.com", Path: "base_path"},
		},
		Directories: []string{"base_dir"},
		RunOnce:     []string{"base_script.sh"},
		OnChange:    []string{"base_cmd"},
		Plugins: map[string]interface{}{
			"base": map[string]interface{}{"key": "base_value"},
		},
	}

	override := &Config{
		DotfilesRepo:   "https://github.com/override/dotfiles",
		DataFile:       "override_data.yaml",
		TemplateEngine: "",
		Dotfiles: []DotfileEntry{
			{Directory: "override_git"},
		},
		ExtraRepos: []Repository{
			{URL: "https://override.com", Path: "override_path"},
		},
		Directories: []string{"override_dir"},
		RunOnce:     []string{"override_script.sh"},
		OnChange:    []string{"override_cmd"},
		Plugins: map[string]interface{}{
			"override": map[string]interface{}{"key": "override_value"},
		},
	}

	merged := MergeConfigs(base, override)

	if merged.DotfilesRepo != "https://github.com/override/dotfiles" {
		t.Errorf("MergeConfigs() DotfilesRepo = %v, want 'https://github.com/override/dotfiles'", merged.DotfilesRepo)
	}

	if merged.DotfilesPath != "~/.base_dotfiles" {
		t.Errorf("MergeConfigs() DotfilesPath = %v, want '~/.base_dotfiles'", merged.DotfilesPath)
	}

	if merged.DataFile != "override_data.yaml" {
		t.Errorf("MergeConfigs() DataFile = %v, want 'override_data.yaml'", merged.DataFile)
	}

	if merged.TemplateEngine != "go" {
		t.Errorf("MergeConfigs() TemplateEngine = %v, want 'go'", merged.TemplateEngine)
	}

	if len(merged.Dotfiles) != 2 {
		t.Errorf("MergeConfigs() Dotfiles length = %d, want 2", len(merged.Dotfiles))
	}

	if len(merged.ExtraRepos) != 2 {
		t.Errorf("MergeConfigs() ExtraRepos length = %d, want 2", len(merged.ExtraRepos))
	}

	if len(merged.Directories) != 2 {
		t.Errorf("MergeConfigs() Directories length = %d, want 2", len(merged.Directories))
	}

	if len(merged.RunOnce) != 2 {
		t.Errorf("MergeConfigs() RunOnce length = %d, want 2", len(merged.RunOnce))
	}

	if len(merged.OnChange) != 2 {
		t.Errorf("MergeConfigs() OnChange length = %d, want 2", len(merged.OnChange))
	}

	// Check plugin merging
	if merged.Plugins["base"] == nil {
		t.Error("MergeConfigs() should preserve base plugin config")
	}
	if merged.Plugins["override"] == nil {
		t.Error("MergeConfigs() should include override plugin config")
	}
}

func TestGenerateSchemaDoc(t *testing.T) {
	doc := GenerateSchemaDoc()

	if doc == "" {
		t.Error("GenerateSchemaDoc() returned empty string")
	}

	// Check for key sections
	expectedSections := []string{
		"# Kilt Configuration Schema",
		"## Global Configuration",
		"## Dotfiles",
		"## Extra Repositories",
		"## Directories",
		"## Run Once Scripts",
		"## On Change Commands",
		"## Plugin Configuration",
	}

	for _, section := range expectedSections {
		if !strings.Contains(doc, section) {
			t.Errorf("GenerateSchemaDoc() missing section: %s", section)
		}
	}
}

func TestExpandPaths_PluginPaths(t *testing.T) {
	homeDir, _ := os.UserHomeDir()

	cfg := &Config{
		Plugins: map[string]interface{}{
			"test": map[string]interface{}{
				"path1":    "~/plugin_path1",
				"path2":    "${HOME}/plugin_path2",
				"non_path": "regular_value",
			},
		},
	}

	err := ExpandPaths(cfg)
	if err != nil {
		t.Fatalf("ExpandPaths() error = %v, want nil", err)
	}

	pluginConfig := cfg.Plugins["test"].(map[string]interface{})
	if pluginConfig["path1"] != filepath.Join(homeDir, "plugin_path1") {
		t.Errorf("ExpandPaths() plugin path1 = %v, want %v", pluginConfig["path1"], filepath.Join(homeDir, "plugin_path1"))
	}

	if pluginConfig["non_path"] != "regular_value" {
		t.Errorf("ExpandPaths() should not modify non-path values")
	}
}

func TestDotfileEntry_UnmarshalYAML(t *testing.T) {
	tmpDir := t.TempDir()

	// Test simple string form
	configPath := filepath.Join(tmpDir, "config1.yaml")
	configContent := `
dotfiles:
  - zsh
  - git
  - vim
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(cfg.Dotfiles) != 3 {
		t.Fatalf("Expected 3 dotfiles, got %d", len(cfg.Dotfiles))
	}

	if cfg.Dotfiles[0].Directory != "zsh" {
		t.Errorf("Dotfiles[0].Directory = %v, want 'zsh'", cfg.Dotfiles[0].Directory)
	}

	// Test map form
	configPath2 := filepath.Join(tmpDir, "config2.yaml")
	configContent2 := `
dotfiles:
  - source: ssh/config
    target: ~/.ssh/config
    mode: "0600"
    template: true
`
	if err := os.WriteFile(configPath2, []byte(configContent2), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg2, err := LoadConfig(configPath2)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(cfg2.Dotfiles) != 1 {
		t.Fatalf("Expected 1 dotfile, got %d", len(cfg2.Dotfiles))
	}

	if cfg2.Dotfiles[0].Source != "ssh/config" {
		t.Errorf("Dotfiles[0].Source = %v, want 'ssh/config'", cfg2.Dotfiles[0].Source)
	}

	if cfg2.Dotfiles[0].Mode != "0600" {
		t.Errorf("Dotfiles[0].Mode = %v, want '0600'", cfg2.Dotfiles[0].Mode)
	}

	if !cfg2.Dotfiles[0].Template {
		t.Error("Dotfiles[0].Template should be true")
	}
}

func TestDefaultDotfilesPath(t *testing.T) {
	path := DefaultDotfilesPath()
	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".dotfiles")

	if path != expected {
		t.Errorf("DefaultDotfilesPath() = %v, want %v", path, expected)
	}
}
