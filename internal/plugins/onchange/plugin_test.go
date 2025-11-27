package onchange

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/unravelling/kilt/internal/core"
	"github.com/unravelling/kilt/internal/plugin"
)

// mockLogger is a simple mock logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, fields ...interface{}) {}
func (m *mockLogger) Info(msg string, fields ...interface{})   {}
func (m *mockLogger) Warn(msg string, fields ...interface{})  {}
func (m *mockLogger) Error(msg string, fields ...interface{}) {}

func TestOnChangePlugin_Name(t *testing.T) {
	p := &OnChangePlugin{}
	if p.Name() != "onchange" {
		t.Errorf("Name() = %v, want 'onchange'", p.Name())
	}
}

func TestOnChangePlugin_Version(t *testing.T) {
	p := &OnChangePlugin{}
	if p.Version() != "1.0.0" {
		t.Errorf("Version() = %v, want '1.0.0'", p.Version())
	}
}

func TestOnChangePlugin_Phase(t *testing.T) {
	p := &OnChangePlugin{}
	if p.Phase() != plugin.PhaseOnChange {
		t.Errorf("Phase() = %v, want PhaseOnChange", p.Phase())
	}
}

func TestOnChangePlugin_Dependencies(t *testing.T) {
	p := &OnChangePlugin{}
	deps := p.Dependencies()
	if len(deps) != 0 {
		t.Errorf("Dependencies() = %v, want empty slice", deps)
	}
}

func TestOnChangePlugin_Initialize(t *testing.T) {
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
		OnChange: []string{},
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

	p := &OnChangePlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	if p.ctx == nil {
		t.Error("Plugin context should be set")
	}

	// Check defaults
	if p.defaultTimeout != 5*time.Minute {
		t.Errorf("defaultTimeout = %v, want 5m", p.defaultTimeout)
	}

	if p.detectionMode != "global" {
		t.Errorf("detectionMode = %v, want 'global'", p.detectionMode)
	}
}

func TestOnChangePlugin_Initialize_WithConfig(t *testing.T) {
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

	watchFile := filepath.Join(tmpDir, "watch.txt")
	if err := os.WriteFile(watchFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create watch file: %v", err)
	}

	cfg := &core.Config{
		OnChange: []string{},
		Plugins: map[string]interface{}{
			"onchange": map[string]interface{}{
				"timeout":        "10m",
				"detection_mode": "conditional",
				"watch_files":   []interface{}{watchFile},
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

	p := &OnChangePlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	if p.defaultTimeout != 10*time.Minute {
		t.Errorf("defaultTimeout = %v, want 10m", p.defaultTimeout)
	}

	if p.detectionMode != "conditional" {
		t.Errorf("detectionMode = %v, want 'conditional'", p.detectionMode)
	}

	if len(p.watchFiles) != 1 {
		t.Errorf("watchFiles length = %v, want 1", len(p.watchFiles))
	}
}

func TestOnChangePlugin_Execute_GlobalMode_NoChanges(t *testing.T) {
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
		OnChange: []string{"echo 'test'"},
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

	p := &OnChangePlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	// Execute with no changes
	if err := p.Execute(execCtx); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// Should not execute commands when no changes
	if len(execCtx.Changes) != 0 {
		t.Errorf("Expected 0 changes, got %d", len(execCtx.Changes))
	}
}

func TestOnChangePlugin_Execute_GlobalMode_WithChanges(t *testing.T) {
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
		OnChange: []string{"echo 'test'"},
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

	p := &OnChangePlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Add a change to the execution context
	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
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
	if err := p.Execute(execCtx); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// Should execute commands when changes detected
	if len(execCtx.Changes) < 2 {
		t.Errorf("Expected at least 2 changes (original + command execution), got %d", len(execCtx.Changes))
	}

	// Check that command execution was recorded
	found := false
	for _, change := range execCtx.Changes {
		if change.Type == "onchange_execute" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected onchange_execute change to be recorded")
	}
}

func TestOnChangePlugin_Execute_ConditionalMode_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	watchFile := filepath.Join(tmpDir, "watch.txt")
	if err := os.WriteFile(watchFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create watch file: %v", err)
	}

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	// Record the watch file in state (so it's not considered changed)
	if err := state.UpdateFileRecord("", watchFile); err != nil {
		t.Fatalf("Failed to record file: %v", err)
	}

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create backup manager: %v", err)
	}

	template := core.NewTemplateEngine()
	homeDir, _ := os.UserHomeDir()

	cfg := &core.Config{
		OnChange: []string{"echo 'test'"},
		Plugins: map[string]interface{}{
			"onchange": map[string]interface{}{
				"detection_mode": "conditional",
				"watch_files":   []interface{}{watchFile},
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

	p := &OnChangePlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Add a change to a different file
	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
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
	if err := p.Execute(execCtx); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// Should not execute commands when watched file didn't change
	// Count only onchange_execute changes
	onChangeCount := 0
	for _, change := range execCtx.Changes {
		if change.Type == "onchange_execute" {
			onChangeCount++
		}
	}
	if onChangeCount != 0 {
		t.Errorf("Expected 0 onchange_execute changes, got %d", onChangeCount)
	}
}

func TestOnChangePlugin_Execute_ConditionalMode_WithMatch(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	watchFile := filepath.Join(tmpDir, "watch.txt")
	if err := os.WriteFile(watchFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create watch file: %v", err)
	}

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	// Record the watch file in state
	if err := state.UpdateFileRecord("", watchFile); err != nil {
		t.Fatalf("Failed to record file: %v", err)
	}

	// Modify the file to trigger change detection
	if err := os.WriteFile(watchFile, []byte("modified"), 0644); err != nil {
		t.Fatalf("Failed to modify watch file: %v", err)
	}

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create backup manager: %v", err)
	}

	template := core.NewTemplateEngine()
	homeDir, _ := os.UserHomeDir()

	cfg := &core.Config{
		OnChange: []string{"echo 'test'"},
		Plugins: map[string]interface{}{
			"onchange": map[string]interface{}{
				"detection_mode": "conditional",
				"watch_files":   []interface{}{watchFile},
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

	p := &OnChangePlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	// Execute - should run because watched file changed
	if err := p.Execute(execCtx); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// Should execute commands when watched file changed
	onChangeCount := 0
	for _, change := range execCtx.Changes {
		if change.Type == "onchange_execute" {
			onChangeCount++
		}
	}
	if onChangeCount == 0 {
		t.Error("Expected onchange_execute change to be recorded")
	}
}

func TestOnChangePlugin_Execute_DryRun(t *testing.T) {
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
		OnChange: []string{"echo 'test'"},
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

	p := &OnChangePlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Add a change to trigger execution
	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
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
	if err := p.Execute(execCtx); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// Should record dry-run execution
	found := false
	for _, change := range execCtx.Changes {
		if change.Type == "onchange_execute" {
			found = true
			if !strings.Contains(change.Description, "Would execute") {
				t.Error("Dry-run change should indicate 'Would execute'")
			}
			break
		}
	}
	if !found {
		t.Error("Expected onchange_execute change to be recorded in dry-run")
	}
}

func TestOnChangePlugin_Execute_MultipleCommands(t *testing.T) {
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
		OnChange: []string{"echo 'command1'", "echo 'command2'", "echo 'command3'"},
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

	p := &OnChangePlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Add a change to trigger execution
	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
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
	if err := p.Execute(execCtx); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// Should execute all commands
	onChangeCount := 0
	for _, change := range execCtx.Changes {
		if change.Type == "onchange_execute" {
			onChangeCount++
		}
	}
	if onChangeCount != 3 {
		t.Errorf("Expected 3 onchange_execute changes, got %d", onChangeCount)
	}
}

func TestOnChangePlugin_Validate(t *testing.T) {
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
		OnChange: []string{"echo 'test'"},
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

	p := &OnChangePlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Validate should pass
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestOnChangePlugin_Validate_EmptyCommand(t *testing.T) {
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
		OnChange: []string{""}, // Empty command
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

	p := &OnChangePlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Validate should fail
	if err := p.Validate(); err == nil {
		t.Error("Validate() should return error for empty command")
	}
}

func TestOnChangePlugin_Validate_ConditionalMode_NoWatchFiles(t *testing.T) {
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
		OnChange: []string{"echo 'test'"},
		Plugins: map[string]interface{}{
			"onchange": map[string]interface{}{
				"detection_mode": "conditional",
				// No watch_files specified
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

	p := &OnChangePlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Validate should fail
	if err := p.Validate(); err == nil {
		t.Error("Validate() should return error when detection_mode is conditional but no watch_files specified")
	}
}

func TestOnChangePlugin_Rollback(t *testing.T) {
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
		OnChange: []string{"echo 'test'"},
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

	p := &OnChangePlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Add a change and execute
	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes: []plugin.Change{
			{
				Type:        "file_write",
				Files:       []string{filepath.Join(tmpDir, "test.txt")},
				Description: "Test file change",
			},
		},
		Errors: make([]error, 0),
	}

	if err := p.Execute(execCtx); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
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

	// Verify rollback change was recorded
	if len(rollbackCtx.Changes) == 0 {
		t.Error("Expected rollback changes to be recorded")
	}
}

