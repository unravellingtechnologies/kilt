package runonce

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
	assert.Equal(t, "runonce", p.Name())
}

func TestPlugin_Version(t *testing.T) {
	p := &Plugin{}
	assert.Equal(t, "1.0.0", p.Version())
}

func TestPlugin_Phase(t *testing.T) {
	p := &Plugin{}
	assert.Equal(t, plugin.PhaseRunOnce, p.Phase())
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
		RunOnce: []string{},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	assert.NotNil(t, p.ctx)
	assert.Equal(t, 5*time.Minute, p.defaultTimeout)
	assert.False(t, p.forceRun)
}

func TestPlugin_Initialise_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		RunOnce: []string{},
		Plugins: map[string]interface{}{
			"runonce": map[string]interface{}{
				"timeout":   "10m",
				"force_run": true,
			},
		},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	assert.Equal(t, 10*time.Minute, p.defaultTimeout)
	assert.True(t, p.forceRun)
}

func TestPlugin_Execute_SimpleScript(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Create a simple test script
	scriptPath := filepath.Join(workDir, "test_script.sh")
	scriptContent := `#!/bin/sh
echo "Hello from test script"
exit 0
`
	require.NoError(t, os.WriteFile(scriptPath, []byte(scriptContent), 0o755))

	cfg := &core.Config{
		RunOnce: []string{"test_script.sh"},
	}

	p, ctx := setupPlugin(t, workDir, cfg)

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	require.NoError(t, p.Execute(execCtx))

	// Verify script was executed (check state)
	taskID := p.generateTaskID(scriptPath)
	assert.True(t, ctx.State.IsTaskCompleted(taskID), "Script execution should be recorded in state")
	assert.Len(t, execCtx.Changes, 1)

	// Verify the change type
	change := execCtx.Changes[0]
	assert.Equal(t, "runonce_execute", change.Type)
}

func TestPlugin_Execute_SkipAlreadyExecuted(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Create a simple test script
	scriptPath := filepath.Join(workDir, "test_script.sh")
	scriptContent := `#!/bin/sh
echo "Hello from test script"
exit 0
`
	require.NoError(t, os.WriteFile(scriptPath, []byte(scriptContent), 0o755))

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	// Mark script as already executed
	taskID := generateTaskIDForTest(scriptPath)
	record := core.RunOnceRecord{
		TaskID:     taskID,
		ExecutedAt: time.Now(),
		ExitCode:   0,
		Output:     "Previous execution",
	}
	require.NoError(t, state.MarkTaskCompleted(taskID, record))

	cfg := &core.Config{
		RunOnce: []string{"test_script.sh"},
	}

	p, ctx := setupPlugin(t, workDir, cfg)
	ctx.State = state

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	require.NoError(t, p.Execute(execCtx))

	// Verify script was skipped
	assert.Len(t, execCtx.Changes, 1)

	change := execCtx.Changes[0]
	assert.Equal(t, "runonce_skipped", change.Type)
}

func TestPlugin_Execute_ForceRun(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Create a simple test script
	scriptPath := filepath.Join(workDir, "test_script.sh")
	scriptContent := `#!/bin/sh
echo "Hello from test script"
exit 0
`
	require.NoError(t, os.WriteFile(scriptPath, []byte(scriptContent), 0o755))

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	// Mark script as already executed
	taskID := generateTaskIDForTest(scriptPath)
	record := core.RunOnceRecord{
		TaskID:     taskID,
		ExecutedAt: time.Now(),
		ExitCode:   0,
		Output:     "Previous execution",
	}
	require.NoError(t, state.MarkTaskCompleted(taskID, record))

	cfg := &core.Config{
		RunOnce: []string{"test_script.sh"},
		Plugins: map[string]interface{}{
			"runonce": map[string]interface{}{
				"force_run": true,
			},
		},
	}

	p, ctx := setupPlugin(t, workDir, cfg)
	ctx.State = state

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	require.NoError(t, p.Execute(execCtx))

	// Verify script was executed (not skipped)
	assert.Len(t, execCtx.Changes, 1)

	change := execCtx.Changes[0]
	assert.Equal(t, "runonce_execute", change.Type, "should not be skipped")
}

func TestPlugin_Execute_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Create a simple test script
	scriptPath := filepath.Join(workDir, "test_script.sh")
	scriptContent := `#!/bin/sh
echo "Hello from test script"
exit 0
`
	require.NoError(t, os.WriteFile(scriptPath, []byte(scriptContent), 0o755))

	cfg := &core.Config{
		RunOnce: []string{"test_script.sh"},
	}

	p, ctx := setupPlugin(t, workDir, cfg)
	ctx.DryRun = true

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	require.NoError(t, p.Execute(execCtx))

	// Verify script was NOT executed (dry-run)
	taskID := generateTaskIDForTest(scriptPath)
	assert.False(t, ctx.State.IsTaskCompleted(taskID), "Script should not be executed in dry-run mode")
	assert.Len(t, execCtx.Changes, 1)

	change := execCtx.Changes[0]
	assert.Equal(t, "runonce_execute", change.Type)
}

func TestPlugin_Execute_ScriptFailure(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Create a script that fails
	scriptPath := filepath.Join(workDir, "failing_script.sh")
	scriptContent := `#!/bin/sh
echo "This script fails"
exit 1
`
	require.NoError(t, os.WriteFile(scriptPath, []byte(scriptContent), 0o755))

	cfg := &core.Config{
		RunOnce: []string{"failing_script.sh"},
	}

	p, ctx := setupPlugin(t, workDir, cfg)

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	// Execute should fail
	err := p.Execute(execCtx)
	assert.Error(t, err, "Execute() should return error when script fails")

	// Verify failure was recorded in state
	taskID := generateTaskIDForTest(scriptPath)
	assert.True(t, ctx.State.IsTaskCompleted(taskID), "Script failure should be recorded in state")

	// Verify the record shows failure
	recordInterface, exists := ctx.State.GetRunOnceRecord(taskID)
	require.True(t, exists, "Record should exist")
	record, ok := recordInterface.(*core.RunOnceRecord)
	require.True(t, ok, "Record should be RunOnceRecord type")
	assert.Equal(t, 1, record.ExitCode)
}

func TestPlugin_Execute_MultipleScripts(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Create multiple test scripts
	script1 := filepath.Join(workDir, "script1.sh")
	script2 := filepath.Join(workDir, "script2.sh")
	script3 := filepath.Join(workDir, "script3.sh")

	scripts := []struct {
		path    string
		content string
	}{
		{script1, "#!/bin/sh\necho 'Script 1'\nexit 0\n"},
		{script2, "#!/bin/sh\necho 'Script 2'\nexit 0\n"},
		{script3, "#!/bin/sh\necho 'Script 3'\nexit 0\n"},
	}

	for _, s := range scripts {
		require.NoError(t, os.WriteFile(s.path, []byte(s.content), 0o755))
	}

	cfg := &core.Config{
		RunOnce: []string{"script1.sh", "script2.sh", "script3.sh"},
	}

	p, ctx := setupPlugin(t, workDir, cfg)

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	require.NoError(t, p.Execute(execCtx))

	// Verify all scripts were executed
	assert.Len(t, execCtx.Changes, 3)

	// Verify all scripts are recorded in state
	for _, s := range scripts {
		taskID := generateTaskIDForTest(s.path)
		assert.True(t, ctx.State.IsTaskCompleted(taskID), "Script should be recorded in state: %s", s.path)
	}
}

func TestPlugin_Validate_MissingScript(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		RunOnce: []string{"nonexistent_script.sh"},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	// Validate should fail
	err := p.Validate()
	assert.Error(t, err, "Validate() should return error when script doesn't exist")
}

func TestPlugin_Rollback(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Create a simple test script
	scriptPath := filepath.Join(workDir, "test_script.sh")
	scriptContent := `#!/bin/sh
echo "Hello from test script"
exit 0
`
	require.NoError(t, os.WriteFile(scriptPath, []byte(scriptContent), 0o755))

	cfg := &core.Config{
		RunOnce: []string{"test_script.sh"},
	}

	p, ctx := setupPlugin(t, workDir, cfg)

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	// Execute script
	require.NoError(t, p.Execute(execCtx))

	// Verify script is recorded
	taskID := generateTaskIDForTest(scriptPath)
	assert.True(t, ctx.State.IsTaskCompleted(taskID), "Script should be recorded in state")

	// Rollback
	rollbackCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	require.NoError(t, p.Rollback(rollbackCtx))

	// Verify script record was removed (allows re-execution)
	assert.False(t, ctx.State.IsTaskCompleted(taskID), "Script record should be removed after rollback")
	assert.Len(t, rollbackCtx.Changes, 1)
}

// Helper function to generate task ID for testing (matches plugin's method)
func generateTaskIDForTest(scriptPath string) string {
	absPath, err := filepath.Abs(scriptPath)
	if err != nil {
		absPath = scriptPath
	}
	// Use the same logic as the plugin
	p := &Plugin{}
	return p.generateTaskID(absPath)
}
