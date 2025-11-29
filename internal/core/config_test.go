package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)

	assert.NotEmpty(t, cfg.DataFile)
	assert.Equal(t, "go", cfg.TemplateEngine)
	assert.Equal(t, "https://github.com/test/dotfiles", cfg.DotfilesRepo)
	assert.Len(t, cfg.Dotfiles, 3)

	// Check simple form (directory name)
	assert.Equal(t, "zsh", cfg.Dotfiles[0].Directory)

	// Check complex form (source/target mapping)
	homeDir, _ := os.UserHomeDir()
	expectedTarget := filepath.Join(homeDir, ".ssh", "config")
	assert.Equal(t, expectedTarget, cfg.Dotfiles[2].Target)

	assert.Len(t, cfg.ExtraRepos, 1)
	assert.Len(t, cfg.Directories, 2)
	assert.Len(t, cfg.RunOnce, 1)
	assert.Len(t, cfg.OnChange, 1)
}

func TestLoadConfig_InvalidFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	assert.Error(t, err, "LoadConfig() should return error for nonexistent file")
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	invalidContent := `
invalid: yaml: content: [
`
	require.NoError(t, os.WriteFile(configPath, []byte(invalidContent), 0644))

	_, err := LoadConfig(configPath)
	assert.Error(t, err, "LoadConfig() should return error for invalid YAML")
}

func TestFindConfigFile(t *testing.T) {
	// Test with local config
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir("/")

	// Create .kilt directory
	kiltDir := filepath.Join(tmpDir, ".kilt")
	require.NoError(t, os.MkdirAll(kiltDir, 0755))

	configPath := filepath.Join(kiltDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("# test config"), 0644))

	found, err := FindConfigFile()
	require.NoError(t, err)

	// Resolve symlinks for comparison (macOS uses /private/var for /var)
	foundResolved, _ := filepath.EvalSymlinks(found)
	configPathResolved, _ := filepath.EvalSymlinks(configPath)
	assert.Equal(t, configPathResolved, foundResolved)
}

func TestFindConfigFile_HomeDir(t *testing.T) {
	// Save original working directory and HOME
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	// Create a temporary fake home directory
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	// Create .kilt directory and config file under fake home
	kiltDir := filepath.Join(fakeHome, ".kilt")
	require.NoError(t, os.MkdirAll(kiltDir, 0755))

	configPath := filepath.Join(kiltDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("# test config"), 0644))

	// Change to a separate temporary working directory (not the fake home)
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	found, err := FindConfigFile()
	require.NoError(t, err)
	assert.Equal(t, configPath, found)
}

func TestFindConfigFile_NotFound(t *testing.T) {
	// Save original working directory and HOME
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	// Create a temporary fake home directory (isolated environment)
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	// Change to a directory without config
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	// Verify no config files exist in fake home
	// (they shouldn't exist since we just created the temp dir)
	_, err = FindConfigFile()
	assert.Error(t, err, "FindConfigFile() should return error when config not found")
}

func TestExpandPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

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
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
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
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(homeDir, ".dotfiles"), cfg.DotfilesPath)
	assert.Equal(t, filepath.Join(homeDir, "data.yaml"), cfg.DataFile)
	assert.Equal(t, filepath.Join(homeDir, ".target"), cfg.Dotfiles[0].Target)
	assert.Equal(t, filepath.Join(homeDir, "repo"), cfg.ExtraRepos[0].Path)
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
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
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
			assert.Equal(t, tt.want, got)
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
	require.NotNil(t, pluginConfig, "GetPluginConfig() returned nil for existing plugin")
	assert.Equal(t, "value1", pluginConfig["key1"])

	// Test non-existent plugin
	pluginConfig = cfg.GetPluginConfig("nonexistent")
	assert.Nil(t, pluginConfig, "GetPluginConfig() should return nil for non-existent plugin")

	// Test plugin with non-map value
	pluginConfig = cfg.GetPluginConfig("other")
	assert.Nil(t, pluginConfig, "GetPluginConfig() should return nil for non-map plugin config")

	// Test nil plugins map
	cfg.Plugins = nil
	pluginConfig = cfg.GetPluginConfig("test")
	assert.Nil(t, pluginConfig, "GetPluginConfig() should return nil when Plugins is nil")
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

	assert.Equal(t, "https://github.com/override/dotfiles", merged.DotfilesRepo)
	assert.Equal(t, "~/.base_dotfiles", merged.DotfilesPath)
	assert.Equal(t, "override_data.yaml", merged.DataFile)
	assert.Equal(t, "go", merged.TemplateEngine)
	assert.Len(t, merged.Dotfiles, 2)
	assert.Len(t, merged.ExtraRepos, 2)
	assert.Len(t, merged.Directories, 2)
	assert.Len(t, merged.RunOnce, 2)
	assert.Len(t, merged.OnChange, 2)

	// Check plugin merging
	assert.NotNil(t, merged.Plugins["base"], "MergeConfigs() should preserve base plugin config")
	assert.NotNil(t, merged.Plugins["override"], "MergeConfigs() should include override plugin config")
}

func TestGenerateSchemaDoc(t *testing.T) {
	doc := GenerateSchemaDoc()

	assert.NotEmpty(t, doc, "GenerateSchemaDoc() returned empty string")

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
		assert.Contains(t, doc, section, "GenerateSchemaDoc() missing section: %s", section)
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
	require.NoError(t, err)

	pluginConfig := cfg.Plugins["test"].(map[string]interface{})
	assert.Equal(t, filepath.Join(homeDir, "plugin_path1"), pluginConfig["path1"])
	assert.Equal(t, "regular_value", pluginConfig["non_path"], "ExpandPaths() should not modify non-path values")
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
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)

	require.Len(t, cfg.Dotfiles, 3)
	assert.Equal(t, "zsh", cfg.Dotfiles[0].Directory)

	// Test map form
	configPath2 := filepath.Join(tmpDir, "config2.yaml")
	configContent2 := `
dotfiles:
  - source: ssh/config
    target: ~/.ssh/config
    mode: "0600"
    template: true
`
	require.NoError(t, os.WriteFile(configPath2, []byte(configContent2), 0644))

	cfg2, err := LoadConfig(configPath2)
	require.NoError(t, err)

	require.Len(t, cfg2.Dotfiles, 1)
	assert.Equal(t, "ssh/config", cfg2.Dotfiles[0].Source)
	assert.Equal(t, "0600", cfg2.Dotfiles[0].Mode)
	assert.True(t, cfg2.Dotfiles[0].Template)
}

func TestDefaultDotfilesPath(t *testing.T) {
	path := DefaultDotfilesPath()
	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".dotfiles")

	assert.Equal(t, expected, path)
}
