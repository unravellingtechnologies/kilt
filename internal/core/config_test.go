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
	configPath := filepath.Join(tmpDir, "config.toml")

	// Write a valid config file
	// Note: Root-level arrays must come before array-of-tables to avoid TOML parsing issues
	configContent := `
data_file = "data.toml"
template_engine = "go"

directories = [
    "~/dev/personal",
    "~/dev/work"
]

run_once = [
    "scripts/install.sh"
]

on_change = [
    "brew bundle"
]

[[files]]
source = "zsh/zshrc"
target = "~/.zshrc"
template = false
mode = "0644"

[[files]]
source = "gitconfig.tmpl"
target = "~/.gitconfig"
template = true
mode = "0644"

[[extra_repos]]
url = "https://github.com/test/repo"
path = "~/.test/repo"
branch = "main"
sparse = false

[plugins.test]
key = "value"
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

	if len(cfg.Files) != 2 {
		t.Errorf("LoadConfig() Files length = %d, want 2", len(cfg.Files))
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

	// Check that paths were expanded
	homeDir, _ := os.UserHomeDir()
	expectedTarget := filepath.Join(homeDir, ".zshrc")
	if cfg.Files[0].Target != expectedTarget {
		t.Errorf("LoadConfig() Files[0].Target = %v, want %v", cfg.Files[0].Target, expectedTarget)
	}
}

func TestLoadConfig_InvalidFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.toml")
	if err == nil {
		t.Error("LoadConfig() should return error for nonexistent file")
	}
}

func TestLoadConfig_InvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	invalidContent := `
[invalid toml
`
	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("LoadConfig() should return error for invalid TOML")
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

	configPath := filepath.Join(kiltDir, "config.toml")
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

	configPath := filepath.Join(kiltDir, "config.toml")
	if err := os.WriteFile(configPath, []byte("# test config"), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	defer os.Remove(configPath)

	// Change to a directory without .kilt/config.toml
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
	homeConfig := filepath.Join(homeDir, ".kilt", "config.toml")
	os.Remove(homeConfig)

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
		DataFile: "~/data.toml",
		Files: []FileMapping{
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

	if cfg.DataFile != filepath.Join(homeDir, "data.toml") {
		t.Errorf("ExpandPaths() DataFile = %v, want %v", cfg.DataFile, filepath.Join(homeDir, "data.toml"))
	}

	if cfg.Files[0].Target != filepath.Join(homeDir, ".target") {
		t.Errorf("ExpandPaths() Files[0].Target = %v, want %v", cfg.Files[0].Target, filepath.Join(homeDir, ".target"))
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
			name: "valid config",
			cfg: &Config{
				TemplateEngine: "go",
				Files: []FileMapping{
					{Source: "source", Target: "target"},
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
			name: "invalid template engine",
			cfg: &Config{
				TemplateEngine: "invalid",
			},
			wantErr: true,
		},
		{
			name: "empty file source",
			cfg: &Config{
				Files: []FileMapping{
					{Source: "", Target: "target"},
				},
			},
			wantErr: true,
		},
		{
			name: "empty file target",
			cfg: &Config{
				Files: []FileMapping{
					{Source: "source", Target: ""},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid file mode",
			cfg: &Config{
				Files: []FileMapping{
					{Source: "source", Target: "target", Mode: "invalid"},
				},
			},
			wantErr: true,
		},
		{
			name: "valid file mode",
			cfg: &Config{
				Files: []FileMapping{
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
		{
			name: "circular dependency",
			cfg: &Config{
				Files: []FileMapping{
					{Source: "source1", Target: "target1"},
					{Source: "target1", Target: "target2"},
				},
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
		DataFile:       "base_data.toml",
		TemplateEngine: "go",
		Files: []FileMapping{
			{Source: "base1", Target: "target1"},
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
		DataFile:       "override_data.toml",
		TemplateEngine: "",
		Files: []FileMapping{
			{Source: "override1", Target: "override_target1"},
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

	if merged.DataFile != "override_data.toml" {
		t.Errorf("MergeConfigs() DataFile = %v, want 'override_data.toml'", merged.DataFile)
	}

	if merged.TemplateEngine != "go" {
		t.Errorf("MergeConfigs() TemplateEngine = %v, want 'go'", merged.TemplateEngine)
	}

	if len(merged.Files) != 2 {
		t.Errorf("MergeConfigs() Files length = %d, want 2", len(merged.Files))
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
		"## File Mappings",
		"## Extra Repositories",
		"## Directories",
		"## Run Once Scripts",
		"## On Change Commands",
		"## Plugin Configuration",
		"## Path Expansion",
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
				"path1": "~/plugin_path1",
				"path2": "${HOME}/plugin_path2",
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

func TestCheckCircularDependencies(t *testing.T) {
	tests := []struct {
		name    string
		files   []FileMapping
		wantErr bool
	}{
		{
			name: "no circular dependency",
			files: []FileMapping{
				{Source: "source1", Target: "target1"},
				{Source: "source2", Target: "target2"},
			},
			wantErr: false,
		},
		{
			name: "circular dependency",
			files: []FileMapping{
				{Source: "source1", Target: "target1"},
				{Source: "target1", Target: "target2"},
			},
			wantErr: true,
		},
		{
			name:    "empty files",
			files:   []FileMapping{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkCircularDependencies(tt.files)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkCircularDependencies() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

