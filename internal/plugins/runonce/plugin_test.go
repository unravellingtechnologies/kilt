package runonce

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/unravelling/kilt/internal/core"
	"github.com/unravelling/kilt/internal/plugin"
)

// mockLogger is a simple mock logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, fields ...interface{}) {}
func (m *mockLogger) Info(msg string, fields ...interface{})  {}
func (m *mockLogger) Warn(msg string, fields ...interface{})  {}
func (m *mockLogger) Error(msg string, fields ...interface{}) {}

func TestRunOncePlugin_Name(t *testing.T) {
	p := &RunOncePlugin{}
	if p.Name() != "runonce" {
		t.Errorf("Name() = %v, want 'runonce'", p.Name())
	}
}

func TestRunOncePlugin_Version(t *testing.T) {
	p := &RunOncePlugin{}
	if p.Version() != "1.0.0" {
		t.Errorf("Version() = %v, want '1.0.0'", p.Version())
	}
}

func TestRunOncePlugin_Phase(t *testing.T) {
	p := &RunOncePlugin{}
	if p.Phase() != plugin.PhaseRunOnce {
		t.Errorf("Phase() = %v, want PhaseRunOnce", p.Phase())
	}
}

func TestRunOncePlugin_Dependencies(t *testing.T) {
	p := &RunOncePlugin{}
	deps := p.Dependencies()
	if len(deps) != 0 {
		t.Errorf("Dependencies() = %v, want empty slice", deps)
	}
}

func TestRunOncePlugin_Initialize(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create backup manager: %v", err)
	}

	template := core.NewTemplateEngine()
	homeDir, _ := os.UserHomeDir()

	cfg := &core.Config{
		RunOnce: []string{},
	}

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

	p := &RunOncePlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	if p.ctx == nil {
		t.Error("Plugin context should be set")
	}

	// Check default timeout
	if p.defaultTimeout != 5*time.Minute {
		t.Errorf("defaultTimeout = %v, want 5m", p.defaultTimeout)
	}

	// Check default forceRun
	if p.forceRun != false {
		t.Errorf("forceRun = %v, want false", p.forceRun)
	}
}

func TestRunOncePlugin_Initialize_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create backup manager: %v", err)
	}

	template := core.NewTemplateEngine()
	homeDir, _ := os.UserHomeDir()

	cfg := &core.Config{
		RunOnce: []string{},
		Plugins: map[string]interface{}{
			"runonce": map[string]interface{}{
				"timeout":   "10m",
				"force_run": true,
			},
		},
	}

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

	p := &RunOncePlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	if p.defaultTimeout != 10*time.Minute {
		t.Errorf("defaultTimeout = %v, want 10m", p.defaultTimeout)
	}

	if p.forceRun != true {
		t.Errorf("forceRun = %v, want true", p.forceRun)
	}
}

func TestRunOncePlugin_Execute_SimpleScript(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	// Create a simple test script
	scriptPath := filepath.Join(workDir, "test_script.sh")
	scriptContent := `#!/bin/sh
echo "Hello from test script"
exit 0
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create backup manager: %v", err)
	}

	template := core.NewTemplateEngine()
	homeDir, _ := os.UserHomeDir()

	cfg := &core.Config{
		RunOnce: []string{"test_script.sh"},
	}

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

	p := &RunOncePlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	if err := p.Execute(execCtx); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// Verify script was executed (check state)
	taskID := p.generateTaskID(scriptPath)
	if !state.IsTaskCompleted(taskID) {
		t.Error("Script execution should be recorded in state")
	}

	// Verify changes were recorded
	if len(execCtx.Changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(execCtx.Changes))
	}

	// Verify the change type
	change := execCtx.Changes[0]
	if change.Type != "runonce_execute" {
		t.Errorf("Change type = %v, want 'runonce_execute'", change.Type)
	}
}

func TestRunOncePlugin_Execute_SkipAlreadyExecuted(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	// Create a simple test script
	scriptPath := filepath.Join(workDir, "test_script.sh")
	scriptContent := `#!/bin/sh
echo "Hello from test script"
exit 0
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	// Mark script as already executed
	taskID := generateTaskIDForTest(scriptPath)
	record := core.RunOnceRecord{
		TaskID:     taskID,
		ExecutedAt: time.Now(),
		ExitCode:   0,
		Output:     "Previous execution",
	}
	if err := state.MarkTaskCompleted(taskID, record); err != nil {
		t.Fatalf("Failed to mark task as completed: %v", err)
	}

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create backup manager: %v", err)
	}

	template := core.NewTemplateEngine()
	homeDir, _ := os.UserHomeDir()

	cfg := &core.Config{
		RunOnce: []string{"test_script.sh"},
	}

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

	p := &RunOncePlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	if err := p.Execute(execCtx); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// Verify script was skipped
	if len(execCtx.Changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(execCtx.Changes))
	}

	change := execCtx.Changes[0]
	if change.Type != "runonce_skipped" {
		t.Errorf("Change type = %v, want 'runonce_skipped'", change.Type)
	}
}

func TestRunOncePlugin_Execute_ForceRun(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	// Create a simple test script
	scriptPath := filepath.Join(workDir, "test_script.sh")
	scriptContent := `#!/bin/sh
echo "Hello from test script"
exit 0
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	// Mark script as already executed
	taskID := generateTaskIDForTest(scriptPath)
	record := core.RunOnceRecord{
		TaskID:     taskID,
		ExecutedAt: time.Now(),
		ExitCode:   0,
		Output:     "Previous execution",
	}
	if err := state.MarkTaskCompleted(taskID, record); err != nil {
		t.Fatalf("Failed to mark task as completed: %v", err)
	}

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create backup manager: %v", err)
	}

	template := core.NewTemplateEngine()
	homeDir, _ := os.UserHomeDir()

	cfg := &core.Config{
		RunOnce: []string{"test_script.sh"},
		Plugins: map[string]interface{}{
			"runonce": map[string]interface{}{
				"force_run": true,
			},
		},
	}

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

	p := &RunOncePlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	if err := p.Execute(execCtx); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// Verify script was executed (not skipped)
	if len(execCtx.Changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(execCtx.Changes))
	}

	change := execCtx.Changes[0]
	if change.Type != "runonce_execute" {
		t.Errorf("Change type = %v, want 'runonce_execute' (not skipped)", change.Type)
	}
}

func TestRunOncePlugin_Execute_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	// Create a simple test script
	scriptPath := filepath.Join(workDir, "test_script.sh")
	scriptContent := `#!/bin/sh
echo "Hello from test script"
exit 0
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create backup manager: %v", err)
	}

	template := core.NewTemplateEngine()
	homeDir, _ := os.UserHomeDir()

	cfg := &core.Config{
		RunOnce: []string{"test_script.sh"},
	}

	ctx := &plugin.PluginContext{
		Config:   cfg,
		State:    state,
		Backup:   backup,
		Template: template,
		Logger:   &mockLogger{},
		DryRun:   true,
		WorkDir:  workDir,
		HomeDir:  homeDir,
	}

	p := &RunOncePlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	if err := p.Execute(execCtx); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// Verify script was NOT executed (dry-run)
	taskID := generateTaskIDForTest(scriptPath)
	if state.IsTaskCompleted(taskID) {
		t.Error("Script should not be executed in dry-run mode")
	}

	// Verify changes were recorded
	if len(execCtx.Changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(execCtx.Changes))
	}

	change := execCtx.Changes[0]
	if change.Type != "runonce_execute" {
		t.Errorf("Change type = %v, want 'runonce_execute'", change.Type)
	}
}

func TestRunOncePlugin_Execute_ScriptFailure(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	// Create a script that fails
	scriptPath := filepath.Join(workDir, "failing_script.sh")
	scriptContent := `#!/bin/sh
echo "This script fails"
exit 1
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create backup manager: %v", err)
	}

	template := core.NewTemplateEngine()
	homeDir, _ := os.UserHomeDir()

	cfg := &core.Config{
		RunOnce: []string{"failing_script.sh"},
	}

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

	p := &RunOncePlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	// Execute should fail
	if err := p.Execute(execCtx); err == nil {
		t.Error("Execute() should return error when script fails")
	}

	// Verify failure was recorded in state
	taskID := generateTaskIDForTest(scriptPath)
	if !state.IsTaskCompleted(taskID) {
		t.Error("Script failure should be recorded in state")
	}

	// Verify the record shows failure
	record, exists := state.GetRunOnceRecordTyped(taskID)
	if !exists {
		t.Fatal("Record should exist")
	}
	if record.ExitCode != 1 {
		t.Errorf("ExitCode = %v, want 1", record.ExitCode)
	}
}

func TestRunOncePlugin_Execute_MultipleScripts(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

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
		if err := os.WriteFile(s.path, []byte(s.content), 0755); err != nil {
			t.Fatalf("Failed to create test script: %v", err)
		}
	}

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create backup manager: %v", err)
	}

	template := core.NewTemplateEngine()
	homeDir, _ := os.UserHomeDir()

	cfg := &core.Config{
		RunOnce: []string{"script1.sh", "script2.sh", "script3.sh"},
	}

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

	p := &RunOncePlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	if err := p.Execute(execCtx); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// Verify all scripts were executed
	if len(execCtx.Changes) != 3 {
		t.Errorf("Expected 3 changes, got %d", len(execCtx.Changes))
	}

	// Verify all scripts are recorded in state
	for _, s := range scripts {
		taskID := generateTaskIDForTest(s.path)
		if !state.IsTaskCompleted(taskID) {
			t.Errorf("Script should be recorded in state: %s", s.path)
		}
	}
}

func TestRunOncePlugin_Validate_MissingScript(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create backup manager: %v", err)
	}

	template := core.NewTemplateEngine()
	homeDir, _ := os.UserHomeDir()

	cfg := &core.Config{
		RunOnce: []string{"nonexistent_script.sh"},
	}

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

	p := &RunOncePlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Validate should fail
	if err := p.Validate(); err == nil {
		t.Error("Validate() should return error when script doesn't exist")
	}
}

func TestRunOncePlugin_Rollback(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	// Create a simple test script
	scriptPath := filepath.Join(workDir, "test_script.sh")
	scriptContent := `#!/bin/sh
echo "Hello from test script"
exit 0
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create backup manager: %v", err)
	}

	template := core.NewTemplateEngine()
	homeDir, _ := os.UserHomeDir()

	cfg := &core.Config{
		RunOnce: []string{"test_script.sh"},
	}

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

	p := &RunOncePlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	// Execute script
	if err := p.Execute(execCtx); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// Verify script is recorded
	taskID := generateTaskIDForTest(scriptPath)
	if !state.IsTaskCompleted(taskID) {
		t.Error("Script should be recorded in state")
	}

	// Rollback
	rollbackCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	if err := p.Rollback(rollbackCtx); err != nil {
		t.Fatalf("Rollback() error = %v, want nil", err)
	}

	// Verify script record was removed (allows re-execution)
	if state.IsTaskCompleted(taskID) {
		t.Error("Script record should be removed after rollback")
	}

	// Verify rollback change was recorded
	if len(rollbackCtx.Changes) != 1 {
		t.Errorf("Expected 1 rollback change, got %d", len(rollbackCtx.Changes))
	}
}

// Helper function to generate task ID for testing (matches plugin's method)
func generateTaskIDForTest(scriptPath string) string {
	absPath, err := filepath.Abs(scriptPath)
	if err != nil {
		absPath = scriptPath
	}
	// Use the same logic as the plugin
	p := &RunOncePlugin{}
	return p.generateTaskID(absPath)
}
