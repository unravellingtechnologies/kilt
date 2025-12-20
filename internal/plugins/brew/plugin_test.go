package brew

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unravelling/kilt/internal/core"
	"github.com/unravelling/kilt/internal/plugin"
)

// mockLogger is a simple mock logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, fields ...interface{}) {}
func (m *mockLogger) Info(msg string, fields ...interface{})  {}
func (m *mockLogger) Warn(msg string, fields ...interface{})  {}
func (m *mockLogger) Error(msg string, fields ...interface{}) {}

func setupPlugin(t *testing.T, workDir string, cfg *core.Config) (*Plugin, *plugin.Context) {
	t.Helper()
	stateDir := filepath.Join(filepath.Dir(workDir), ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	backupDir := filepath.Join(filepath.Dir(workDir), ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	require.NoError(t, err)

	template := core.NewTemplateEngine()
	homeDir, _ := os.UserHomeDir()

	ctx := &plugin.Context{
		Config:   cfg,
		State:    state,
		Backup:   backup,
		Template: template,
		Logger:   &mockLogger{},
		DryRun:   false,
		WorkDir:  workDir,
		HomeDir:  homeDir,
	}

	p := &Plugin{}
	require.NoError(t, p.Initialise(ctx))

	return p, ctx
}

func TestPlugin_Name(t *testing.T) {
	p := &Plugin{}
	assert.Equal(t, "brew", p.Name())
}

func TestPlugin_Version(t *testing.T) {
	p := &Plugin{}
	assert.Equal(t, "1.0.0", p.Version())
}

func TestPlugin_Phase(t *testing.T) {
	p := &Plugin{}
	assert.Equal(t, plugin.PhaseIntegration, p.Phase())
}

func TestPlugin_Dependencies(t *testing.T) {
	p := &Plugin{}
	deps := p.Dependencies()
	assert.Empty(t, deps)
}

func TestPlugin_Initialise(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	assert.NotNil(t, p.ctx)
	assert.Equal(t, "Brewfile", p.brewfile)
	assert.False(t, p.autoUpdate)
	assert.False(t, p.cleanupAfter)
	assert.False(t, p.autoInstall)
}

func TestPlugin_Initialise_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{
			"brew": map[string]interface{}{
				"brewfile":      "custom/Brewfile",
				"auto_update":   true,
				"cleanup_after": true,
				"auto_install":  true,
				"bundles":       []interface{}{"bootstrap", "dev"},
			},
		},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	assert.Equal(t, "custom/Brewfile", p.brewfile)
	assert.True(t, p.autoUpdate)
	assert.True(t, p.cleanupAfter)
	assert.Len(t, p.bundles, 2)
	assert.True(t, p.autoInstall)
}

func TestPlugin_Validate(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	// Validate should pass (even if brew is not installed)
	err := p.Validate()
	assert.NoError(t, err)
}

func TestPlugin_Execute_NoBrew(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{},
	}

	p, ctx := setupPlugin(t, workDir, cfg)

	// Set brewPath to empty to simulate brew not found
	p.brewPath = ""

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	// Execute should skip gracefully
	err := p.Execute(execCtx)
	assert.NoError(t, err)
	assert.Empty(t, execCtx.Changes)
}

func TestPlugin_Execute_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Create a test Brewfile
	brewfilePath := filepath.Join(workDir, "Brewfile")
	require.NoError(t, os.WriteFile(brewfilePath, []byte("brew 'test'\n"), 0o644))

	cfg := &core.Config{
		Plugins: map[string]interface{}{},
	}

	p, ctx := setupPlugin(t, workDir, cfg)
	ctx.DryRun = true

	// Set a mock brew path for testing
	p.brewPath = "/usr/local/bin/brew"

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	// Execute in dry-run mode
	err := p.Execute(execCtx)
	assert.NoError(t, err)

	// Should record dry-run changes
	found := false
	for _, change := range execCtx.Changes {
		if change.Type == "brew_bundle" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected brew_bundle change to be recorded in dry-run")
}

func TestPlugin_Execute_WithBundles(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{
			"brew": map[string]interface{}{
				"bundles": []interface{}{"bootstrap", "dev"},
			},
		},
	}

	p, ctx := setupPlugin(t, workDir, cfg)
	ctx.DryRun = true

	// Set a mock brew path for testing
	p.brewPath = "/usr/local/bin/brew"

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	// Execute in dry-run mode
	err := p.Execute(execCtx)
	assert.NoError(t, err)

	// Should record changes for each bundle
	bundleCount := 0
	for _, change := range execCtx.Changes {
		if change.Type == "brew_bundle" {
			bundleCount++
		}
	}
	assert.Equal(t, 2, bundleCount)
}

func TestPathsMatch(t *testing.T) {
	// Test identical paths
	assert.True(t, core.PathsMatch("/path/to/file", "/path/to/file"))

	// Test paths resolving to same file
	tmpDir := t.TempDir()
	absPath := filepath.Join(tmpDir, "file")
	// Create the file so we can test from within its directory
	require.NoError(t, os.WriteFile(absPath, []byte{}, 0o644))

	// Test same path comparison
	assert.True(t, core.PathsMatch(absPath, absPath))

	// Test different paths
	assert.False(t, core.PathsMatch("/path/to/file1", "/path/to/file2"))
}

func TestPlugin_Rollback(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{},
	}

	p, ctx := setupPlugin(t, workDir, cfg)

	// Simulate executed bundles
	p.executedBundles = []string{filepath.Join(workDir, "Brewfile")}

	rollbackCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	// Rollback
	err := p.Rollback(rollbackCtx)
	assert.NoError(t, err)
	assert.NotEmpty(t, rollbackCtx.Changes)
	assert.Empty(t, p.executedBundles)
}

func TestPlugin_Execute_AutoInstall_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{
			"brew": map[string]interface{}{
				"auto_install": true,
			},
		},
	}

	p, ctx := setupPlugin(t, workDir, cfg)
	ctx.DryRun = true

	// Verify autoInstall is enabled
	assert.True(t, p.autoInstall)

	// If brew is already installed, we can't test the install path
	// So we'll force brewPath to empty to simulate brew not found
	p.brewPath = ""

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	// Execute in dry-run mode - should attempt to install
	err := p.Execute(execCtx)
	assert.NoError(t, err)

	// Should record dry-run installation
	found := false
	for _, change := range execCtx.Changes {
		if change.Type == "brew_install" {
			found = true
			assert.Contains(t, change.Description, "Would install")
			break
		}
	}
	assert.True(t, found, "Expected brew_install change to be recorded in dry-run")
}
