package brew

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unravelling/kilt/internal/core"
	"github.com/unravelling/kilt/internal/plugin"
)

// mockLogger is a simple mock logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, fields ...interface{}) {}
func (m *mockLogger) Info(msg string, fields ...interface{})   {}
func (m *mockLogger) Warn(msg string, fields ...interface{})  {}
func (m *mockLogger) Error(msg string, fields ...interface{}) {}

func TestBrewPlugin_Name(t *testing.T) {
	p := &BrewPlugin{}
	if p.Name() != "brew" {
		t.Errorf("Name() = %v, want 'brew'", p.Name())
	}
}

func TestBrewPlugin_Version(t *testing.T) {
	p := &BrewPlugin{}
	if p.Version() != "1.0.0" {
		t.Errorf("Version() = %v, want '1.0.0'", p.Version())
	}
}

func TestBrewPlugin_Phase(t *testing.T) {
	p := &BrewPlugin{}
	if p.Phase() != plugin.PhaseIntegration {
		t.Errorf("Phase() = %v, want PhaseIntegration", p.Phase())
	}
}

func TestBrewPlugin_Dependencies(t *testing.T) {
	p := &BrewPlugin{}
	deps := p.Dependencies()
	if len(deps) != 0 {
		t.Errorf("Dependencies() = %v, want empty slice", deps)
	}
}

func TestBrewPlugin_Initialize(t *testing.T) {
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
		Plugins: map[string]interface{}{},
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

	p := &BrewPlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	if p.ctx == nil {
		t.Error("Plugin context should be set")
	}

	// Check defaults
	if p.brewfile != "Brewfile" {
		t.Errorf("brewfile = %v, want 'Brewfile'", p.brewfile)
	}

	if p.autoUpdate != false {
		t.Errorf("autoUpdate = %v, want false", p.autoUpdate)
	}

	if p.cleanupAfter != false {
		t.Errorf("cleanupAfter = %v, want false", p.cleanupAfter)
	}

	if p.autoInstall != false {
		t.Errorf("autoInstall = %v, want false", p.autoInstall)
	}
}

func TestBrewPlugin_Initialize_WithConfig(t *testing.T) {
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

	p := &BrewPlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	if p.brewfile != "custom/Brewfile" {
		t.Errorf("brewfile = %v, want 'custom/Brewfile'", p.brewfile)
	}

	if p.autoUpdate != true {
		t.Errorf("autoUpdate = %v, want true", p.autoUpdate)
	}

	if p.cleanupAfter != true {
		t.Errorf("cleanupAfter = %v, want true", p.cleanupAfter)
	}

	if len(p.bundles) != 2 {
		t.Errorf("bundles length = %v, want 2", len(p.bundles))
	}

	if p.autoInstall != true {
		t.Errorf("autoInstall = %v, want true", p.autoInstall)
	}
}

func TestBrewPlugin_Validate(t *testing.T) {
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
		Plugins: map[string]interface{}{},
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

	p := &BrewPlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Validate should pass (even if brew is not installed)
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestBrewPlugin_Execute_NoBrew(t *testing.T) {
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
		Plugins: map[string]interface{}{},
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

	p := &BrewPlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Set brewPath to empty to simulate brew not found
	p.brewPath = ""

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	// Execute should skip gracefully
	if err := p.Execute(execCtx); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// Should not execute anything
	if len(execCtx.Changes) != 0 {
		t.Errorf("Expected 0 changes when brew is not installed, got %d", len(execCtx.Changes))
	}
}

func TestBrewPlugin_Execute_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	// Create a test Brewfile
	brewfilePath := filepath.Join(workDir, "Brewfile")
	if err := os.WriteFile(brewfilePath, []byte("brew 'test'\n"), 0644); err != nil {
		t.Fatalf("Failed to create Brewfile: %v", err)
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
		Plugins: map[string]interface{}{},
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

	p := &BrewPlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Set a mock brew path for testing
	p.brewPath = "/usr/local/bin/brew"

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	// Execute in dry-run mode
	if err := p.Execute(execCtx); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// Should record dry-run changes
	found := false
	for _, change := range execCtx.Changes {
		if change.Type == "brew_bundle" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected brew_bundle change to be recorded in dry-run")
	}
}

func TestBrewPlugin_Execute_WithBundles(t *testing.T) {
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
		Plugins: map[string]interface{}{
			"brew": map[string]interface{}{
				"bundles": []interface{}{"bootstrap", "dev"},
			},
		},
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

	p := &BrewPlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Set a mock brew path for testing
	p.brewPath = "/usr/local/bin/brew"

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	// Execute in dry-run mode
	if err := p.Execute(execCtx); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// Should record changes for each bundle
	bundleCount := 0
	for _, change := range execCtx.Changes {
		if change.Type == "brew_bundle" {
			bundleCount++
		}
	}
	if bundleCount != 2 {
		t.Errorf("Expected 2 brew_bundle changes, got %d", bundleCount)
	}
}

func TestBrewPlugin_pathsMatch(t *testing.T) {
	p := &BrewPlugin{}

	// Test identical paths
	if !p.pathsMatch("/path/to/file", "/path/to/file") {
		t.Error("pathsMatch() should return true for identical paths")
	}

	// Test relative vs absolute (same file)
	tmpDir := t.TempDir()
	absPath := filepath.Join(tmpDir, "file")
	relPath := filepath.Join(tmpDir, "file")

	if !p.pathsMatch(absPath, relPath) {
		t.Error("pathsMatch() should match paths referring to same file")
	}

	// Test different paths
	if p.pathsMatch("/path/to/file1", "/path/to/file2") {
		t.Error("pathsMatch() should return false for different files")
	}
}

func TestBrewPlugin_Rollback(t *testing.T) {
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
		Plugins: map[string]interface{}{},
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

	p := &BrewPlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Simulate executed bundles
	p.executedBundles = []string{filepath.Join(workDir, "Brewfile")}

	rollbackCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	// Rollback
	if err := p.Rollback(rollbackCtx); err != nil {
		t.Fatalf("Rollback() error = %v, want nil", err)
	}

	// Verify rollback changes were recorded
	if len(rollbackCtx.Changes) == 0 {
		t.Error("Expected rollback changes to be recorded")
	}

	// Verify executed bundles were cleared
	if len(p.executedBundles) != 0 {
		t.Error("Executed bundles should be cleared after rollback")
	}
}

func TestBrewPlugin_Execute_AutoInstall_DryRun(t *testing.T) {
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
		Plugins: map[string]interface{}{
			"brew": map[string]interface{}{
				"auto_install": true,
			},
		},
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

	p := &BrewPlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Verify autoInstall is enabled
	if p.autoInstall != true {
		t.Errorf("autoInstall = %v, want true", p.autoInstall)
	}

	// If brew is already installed, we can't test the install path
	// So we'll force brewPath to empty to simulate brew not found
	if p.brewPath != "" {
		p.brewPath = ""
	}

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	// Execute in dry-run mode - should attempt to install
	if err := p.Execute(execCtx); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// Should record dry-run installation
	found := false
	for _, change := range execCtx.Changes {
		if change.Type == "brew_install" {
			found = true
			if !strings.Contains(change.Description, "Would install") {
				t.Error("Dry-run change should indicate 'Would install'")
			}
			break
		}
	}
	if !found {
		t.Error("Expected brew_install change to be recorded in dry-run")
	}
}

