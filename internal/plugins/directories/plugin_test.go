package directories

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

func TestDirectoriesPlugin_Name(t *testing.T) {
	p := &DirectoriesPlugin{}
	assert.Equal(t, "directories", p.Name())
}

func TestDirectoriesPlugin_Version(t *testing.T) {
	p := &DirectoriesPlugin{}
	assert.Equal(t, "1.0.0", p.Version())
}

func TestDirectoriesPlugin_Phase(t *testing.T) {
	p := &DirectoriesPlugin{}
	assert.Equal(t, plugin.PhaseCore, p.Phase())
}

func TestDirectoriesPlugin_Dependencies(t *testing.T) {
	p := &DirectoriesPlugin{}
	deps := p.Dependencies()
	assert.Empty(t, deps)
}

func setupDirectoriesPlugin(t *testing.T, workDir string, cfg *core.Config) (*DirectoriesPlugin, *plugin.PluginContext) {
	stateDir := filepath.Join(filepath.Dir(workDir), ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	backupDir := filepath.Join(filepath.Dir(workDir), ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	require.NoError(t, err)

	template := core.NewTemplateEngine()
	homeDir, _ := os.UserHomeDir()

	ctx := &plugin.PluginContext{
		Config:   cfg,
		State:    state,
		Backup:   backup,
		Template: template,
		Logger:   &mockLogger{},
		DryRun:   false,
		WorkDir:  workDir,
		HomeDir:  homeDir,
	}

	p := &DirectoriesPlugin{}
	require.NoError(t, p.Initialize(ctx))

	return p, ctx
}

func TestDirectoriesPlugin_Initialize(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	cfg := &core.Config{
		Directories: []string{},
	}

	p, _ := setupDirectoriesPlugin(t, workDir, cfg)

	assert.NotNil(t, p.ctx)
	assert.Equal(t, os.FileMode(0755), p.defaultMode)
}

func TestDirectoriesPlugin_Initialize_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	cfg := &core.Config{
		Directories: []string{},
		Plugins: map[string]interface{}{
			"directories": map[string]interface{}{
				"default_mode": "0700",
			},
		},
	}

	p, _ := setupDirectoriesPlugin(t, workDir, cfg)

	assert.Equal(t, os.FileMode(0700), p.defaultMode)
}

func TestDirectoriesPlugin_Execute_CreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	targetDir := filepath.Join(tmpDir, "new", "nested", "directory")

	cfg := &core.Config{
		Directories: []string{targetDir},
	}

	p, ctx := setupDirectoriesPlugin(t, workDir, cfg)

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	require.NoError(t, p.Execute(execCtx))

	// Verify directory was created
	info, err := os.Stat(targetDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
	assert.Len(t, execCtx.Changes, 1)
}

func TestDirectoriesPlugin_Execute_ExistingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	targetDir := filepath.Join(tmpDir, "existing")
	require.NoError(t, os.MkdirAll(targetDir, 0700))

	cfg := &core.Config{
		Directories: []string{targetDir},
	}

	p, ctx := setupDirectoriesPlugin(t, workDir, cfg)

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	require.NoError(t, p.Execute(execCtx))

	// Verify directory still exists and permissions were updated
	info, err := os.Stat(targetDir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
}

func TestDirectoriesPlugin_Execute_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	targetDir := filepath.Join(tmpDir, "would", "be", "created")

	cfg := &core.Config{
		Directories: []string{targetDir},
	}

	p, ctx := setupDirectoriesPlugin(t, workDir, cfg)
	ctx.DryRun = true

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	require.NoError(t, p.Execute(execCtx))

	// Verify directory was NOT created
	_, err := os.Stat(targetDir)
	assert.True(t, os.IsNotExist(err), "Directory should not exist in dry-run mode")
	assert.Len(t, execCtx.Changes, 1)
}

func TestDirectoriesPlugin_Execute_NonDirectoryError(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Create a file (not a directory)
	targetFile := filepath.Join(tmpDir, "file.txt")
	require.NoError(t, os.WriteFile(targetFile, []byte("content"), 0644))

	cfg := &core.Config{
		Directories: []string{targetFile}, // Trying to create a directory where a file exists
	}

	p, ctx := setupDirectoriesPlugin(t, workDir, cfg)

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	err := p.Execute(execCtx)
	assert.Error(t, err, "Execute() should return error when path exists but is not a directory")
}

func TestDirectoriesPlugin_Rollback(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	targetDir1 := filepath.Join(tmpDir, "dir1")
	targetDir2 := filepath.Join(tmpDir, "dir2")

	cfg := &core.Config{
		Directories: []string{targetDir1, targetDir2},
	}

	p, ctx := setupDirectoriesPlugin(t, workDir, cfg)

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	// Execute to create directories
	require.NoError(t, p.Execute(execCtx))

	// Verify directories were created
	_, err := os.Stat(targetDir1)
	require.NoError(t, err)
	_, err = os.Stat(targetDir2)
	require.NoError(t, err)

	// Rollback
	rollbackCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	require.NoError(t, p.Rollback(rollbackCtx))

	// Verify directories were removed
	_, err = os.Stat(targetDir1)
	assert.True(t, os.IsNotExist(err), "Directory 1 should be removed after rollback")
	_, err = os.Stat(targetDir2)
	assert.True(t, os.IsNotExist(err), "Directory 2 should be removed after rollback")
}

func TestDirectoriesPlugin_Execute_MultipleDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	dir1 := filepath.Join(tmpDir, "dir1")
	dir2 := filepath.Join(tmpDir, "dir2")
	dir3 := filepath.Join(tmpDir, "nested", "dir3")

	cfg := &core.Config{
		Directories: []string{dir1, dir2, dir3},
	}

	p, ctx := setupDirectoriesPlugin(t, workDir, cfg)

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	require.NoError(t, p.Execute(execCtx))

	// Verify all directories were created
	for _, dir := range []string{dir1, dir2, dir3} {
		_, err := os.Stat(dir)
		assert.NoError(t, err, "Directory should exist: %s", dir)
	}

	// Verify changes were recorded
	assert.Len(t, execCtx.Changes, 3)
}
