package onchange

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestPlugin_Name(t *testing.T) {
	p := &Plugin{}
	assert.Equal(t, "onchange", p.Name())
}

func TestPlugin_Version(t *testing.T) {
	p := &Plugin{}
	assert.Equal(t, "1.0.0", p.Version())
}

func TestPlugin_Phase(t *testing.T) {
	p := &Plugin{}
	assert.Equal(t, plugin.PhaseOnChange, p.Phase())
}

func TestPlugin_Dependencies(t *testing.T) {
	p := &Plugin{}
	deps := p.Dependencies()
	assert.Empty(t, deps)
}

func setupPlugin(t *testing.T, workDir string, cfg *core.Config) (*Plugin, *plugin.Context) {
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

func TestPlugin_Initialise(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		OnChange: []string{},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	assert.NotNil(t, p.ctx)
	assert.Equal(t, 5*time.Minute, p.defaultTimeout)
	assert.Equal(t, "global", p.detectionMode)
}

func TestPlugin_Initialise_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	watchFile := filepath.Join(tmpDir, "watch.txt")
	require.NoError(t, os.WriteFile(watchFile, []byte("test"), 0o644))

	cfg := &core.Config{
		OnChange: []string{},
		Plugins: map[string]interface{}{
			"onchange": map[string]interface{}{
				"timeout":        "10m",
				"detection_mode": "conditional",
				"watch_files":    []interface{}{watchFile},
			},
		},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	assert.Equal(t, 10*time.Minute, p.defaultTimeout)
	assert.Equal(t, "conditional", p.detectionMode)
	assert.Len(t, p.watchFiles, 1)
}

func TestPlugin_Execute_GlobalMode_NoChanges(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		OnChange: []string{"echo 'test'"},
	}

	p, ctx := setupPlugin(t, workDir, cfg)

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	// Execute with no changes
	require.NoError(t, p.Execute(execCtx))

	// Should not execute commands when no changes
	assert.Empty(t, execCtx.Changes)
}

func TestPlugin_Execute_GlobalMode_WithChanges(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		OnChange: []string{"echo 'test'"},
	}

	p, ctx := setupPlugin(t, workDir, cfg)

	// Add a change to the execution context
	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: []plugin.Change{
			{
				Type:        "file_write",
				Files:       []string{filepath.Join(tmpDir, "test.txt")},
				Description: "Test file change",
			},
		},
		Errors: make([]error, 0),
	}

	// Execute with changes
	require.NoError(t, p.Execute(execCtx))

	// Should execute commands when changes detected
	assert.GreaterOrEqual(t, len(execCtx.Changes), 2, "Expected at least 2 changes (original + command execution)")

	// Check that command execution was recorded
	found := false
	for _, change := range execCtx.Changes {
		if change.Type == "onchange_execute" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected onchange_execute change to be recorded")
}

func TestPlugin_Execute_ConditionalMode_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	watchFile := filepath.Join(tmpDir, "watch.txt")
	require.NoError(t, os.WriteFile(watchFile, []byte("test"), 0o644))

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	// Record the watch file in state (so it's not considered changed)
	require.NoError(t, state.UpdateFileRecord("", watchFile))

	cfg := &core.Config{
		OnChange: []string{"echo 'test'"},
		Plugins: map[string]interface{}{
			"onchange": map[string]interface{}{
				"detection_mode": "conditional",
				"watch_files":    []interface{}{watchFile},
			},
		},
	}

	p, ctx := setupPlugin(t, workDir, cfg)
	// Override state with the one that has the file recorded
	ctx.State = state

	// Add a change to a different file
	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: []plugin.Change{
			{
				Type:        "file_write",
				Files:       []string{filepath.Join(tmpDir, "other.txt")},
				Description: "Test file change",
			},
		},
		Errors: make([]error, 0),
	}

	// Execute - should not run because watched file didn't change
	require.NoError(t, p.Execute(execCtx))

	// Should not execute commands when watched file didn't change
	onChangeCount := 0
	for _, change := range execCtx.Changes {
		if change.Type == "onchange_execute" {
			onChangeCount++
		}
	}
	assert.Zero(t, onChangeCount, "Expected 0 onchange_execute changes")
}

func TestPlugin_Execute_ConditionalMode_WithMatch(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	watchFile := filepath.Join(tmpDir, "watch.txt")
	require.NoError(t, os.WriteFile(watchFile, []byte("test"), 0o644))

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	// Record the watch file in state
	require.NoError(t, state.UpdateFileRecord("", watchFile))

	// Modify the file to trigger change detection
	require.NoError(t, os.WriteFile(watchFile, []byte("modified"), 0o644))

	cfg := &core.Config{
		OnChange: []string{"echo 'test'"},
		Plugins: map[string]interface{}{
			"onchange": map[string]interface{}{
				"detection_mode": "conditional",
				"watch_files":    []interface{}{watchFile},
			},
		},
	}

	p, ctx := setupPlugin(t, workDir, cfg)
	// Override state with the one that has the file recorded
	ctx.State = state

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	// Execute - should run because watched file changed
	require.NoError(t, p.Execute(execCtx))

	// Should execute commands when watched file changed
	onChangeCount := 0
	for _, change := range execCtx.Changes {
		if change.Type == "onchange_execute" {
			onChangeCount++
		}
	}
	assert.Greater(t, onChangeCount, 0, "Expected onchange_execute change to be recorded")
}

func TestPlugin_Execute_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		OnChange: []string{"echo 'test'"},
	}

	p, ctx := setupPlugin(t, workDir, cfg)
	ctx.DryRun = true

	// Add a change to trigger execution
	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: []plugin.Change{
			{
				Type:        "file_write",
				Files:       []string{filepath.Join(tmpDir, "test.txt")},
				Description: "Test file change",
			},
		},
		Errors: make([]error, 0),
	}

	// Execute in dry-run mode
	require.NoError(t, p.Execute(execCtx))

	// Should record dry-run execution
	found := false
	for _, change := range execCtx.Changes {
		if change.Type == "onchange_execute" {
			found = true
			assert.Contains(t, change.Description, "Would execute", "Dry-run change should indicate 'Would execute'")
			break
		}
	}
	assert.True(t, found, "Expected onchange_execute change to be recorded in dry-run")
}

func TestPlugin_Execute_MultipleCommands(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		OnChange: []string{"echo 'command1'", "echo 'command2'", "echo 'command3'"},
	}

	p, ctx := setupPlugin(t, workDir, cfg)

	// Add a change to trigger execution
	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: []plugin.Change{
			{
				Type:        "file_write",
				Files:       []string{filepath.Join(tmpDir, "test.txt")},
				Description: "Test file change",
			},
		},
		Errors: make([]error, 0),
	}

	// Execute
	require.NoError(t, p.Execute(execCtx))

	// Should execute all commands
	onChangeCount := 0
	for _, change := range execCtx.Changes {
		if change.Type == "onchange_execute" {
			onChangeCount++
		}
	}
	assert.Equal(t, 3, onChangeCount)
}

func TestPlugin_Validate(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		OnChange: []string{"echo 'test'"},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	// Validate should pass
	err := p.Validate()
	assert.NoError(t, err)
}

func TestPlugin_Validate_EmptyCommand(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		OnChange: []string{""}, // Empty command
	}

	p, _ := setupPlugin(t, workDir, cfg)

	// Validate should fail
	err := p.Validate()
	assert.Error(t, err, "Validate() should return error for empty command")
}

func TestPlugin_Validate_ConditionalMode_NoWatchFiles(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		OnChange: []string{"echo 'test'"},
		Plugins: map[string]interface{}{
			"onchange": map[string]interface{}{
				"detection_mode": "conditional",
				// No watch_files specified
			},
		},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	// Validate should fail
	err := p.Validate()
	assert.Error(t, err, "Validate() should return error when detection_mode is conditional but no watch_files specified")
}

func TestPlugin_Rollback(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		OnChange: []string{"echo 'test'"},
	}

	p, ctx := setupPlugin(t, workDir, cfg)

	// Add a change and execute
	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: []plugin.Change{
			{
				Type:        "file_write",
				Files:       []string{filepath.Join(tmpDir, "test.txt")},
				Description: "Test file change",
			},
		},
		Errors: make([]error, 0),
	}

	require.NoError(t, p.Execute(execCtx))

	// Rollback
	rollbackCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	require.NoError(t, p.Rollback(rollbackCtx))

	// Verify rollback change was recorded
	assert.NotEmpty(t, rollbackCtx.Changes)
}
