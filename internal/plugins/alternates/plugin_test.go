package alternates

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestAlternatesPlugin_Name(t *testing.T) {
	p := &AlternatesPlugin{}
	assert.Equal(t, "alternates", p.Name())
}

func TestAlternatesPlugin_Version(t *testing.T) {
	p := &AlternatesPlugin{}
	assert.Equal(t, "1.0.0", p.Version())
}

func TestAlternatesPlugin_Phase(t *testing.T) {
	p := &AlternatesPlugin{}
	assert.Equal(t, plugin.PhasePreSync, p.Phase())
}

func TestAlternatesPlugin_Dependencies(t *testing.T) {
	p := &AlternatesPlugin{}
	deps := p.Dependencies()
	assert.Empty(t, deps)
}

func TestAlternatesPlugin_Initialize(t *testing.T) {
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
		Dotfiles: []core.DotfileEntry{},
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

	p := &AlternatesPlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	if p.ctx == nil {
		t.Error("Plugin context should be set")
	}
}

func TestAlternatesPlugin_Execute_OSMatch(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	// Create test files
	testDir := filepath.Join(workDir, "config")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	// Create base file
	baseFile := filepath.Join(testDir, "config.txt")
	if err := os.WriteFile(baseFile, []byte("base"), 0644); err != nil {
		t.Fatalf("Failed to create base file: %v", err)
	}

	// Create OS-specific alternate
	var osFile string
	if runtime.GOOS == "darwin" {
		osFile = filepath.Join(testDir, "config.mac.txt")
	} else {
		osFile = filepath.Join(testDir, "config.linux.txt")
	}
	if err := os.WriteFile(osFile, []byte("os-specific"), 0644); err != nil {
		t.Fatalf("Failed to create OS file: %v", err)
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
		Dotfiles: []core.DotfileEntry{
			{
				Source: "config/config.txt",
				Target: filepath.Join(tmpDir, "target.txt"),
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

	p := &AlternatesPlugin{}
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

	// Verify source was updated to OS-specific file
	expectedSource := "config/config.mac.txt"
	if runtime.GOOS != "darwin" {
		expectedSource = "config/config.linux.txt"
	}

	if cfg.Dotfiles[0].Source != expectedSource {
		t.Errorf("Source = %v, want %v", cfg.Dotfiles[0].Source, expectedSource)
	}

	// Verify change was recorded
	if len(execCtx.Changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(execCtx.Changes))
	}
}

func TestAlternatesPlugin_Execute_HostnameMatch(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	hostname, _ := os.Hostname()

	// Create test files
	testDir := filepath.Join(workDir, "config")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	// Create base file
	baseFile := filepath.Join(testDir, "config.txt")
	if err := os.WriteFile(baseFile, []byte("base"), 0644); err != nil {
		t.Fatalf("Failed to create base file: %v", err)
	}

	// Create hostname-specific alternate
	hostnameFile := filepath.Join(testDir, "config."+hostname+"@work.txt")
	if err := os.WriteFile(hostnameFile, []byte("hostname-specific"), 0644); err != nil {
		t.Fatalf("Failed to create hostname file: %v", err)
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
		Dotfiles: []core.DotfileEntry{
			{
				Source: "config/config.txt",
				Target: filepath.Join(tmpDir, "target.txt"),
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

	p := &AlternatesPlugin{}
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

	// Verify source was updated to hostname-specific file
	expectedSource := "config/config." + hostname + "@work.txt"
	if cfg.Dotfiles[0].Source != expectedSource {
		t.Errorf("Source = %v, want %v", cfg.Dotfiles[0].Source, expectedSource)
	}
}

func TestAlternatesPlugin_Execute_Priority(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	hostname, _ := os.Hostname()

	// Create test files
	testDir := filepath.Join(workDir, "config")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	// Create base file
	baseFile := filepath.Join(testDir, "config.txt")
	if err := os.WriteFile(baseFile, []byte("base"), 0644); err != nil {
		t.Fatalf("Failed to create base file: %v", err)
	}

	// Create OS-specific alternate
	var osFile string
	if runtime.GOOS == "darwin" {
		osFile = filepath.Join(testDir, "config.mac.txt")
	} else {
		osFile = filepath.Join(testDir, "config.linux.txt")
	}
	if err := os.WriteFile(osFile, []byte("os-specific"), 0644); err != nil {
		t.Fatalf("Failed to create OS file: %v", err)
	}

	// Create hostname-specific alternate (should win due to higher priority)
	hostnameFile := filepath.Join(testDir, "config."+hostname+"@work.txt")
	if err := os.WriteFile(hostnameFile, []byte("hostname-specific"), 0644); err != nil {
		t.Fatalf("Failed to create hostname file: %v", err)
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
		Dotfiles: []core.DotfileEntry{
			{
				Source: "config/config.txt",
				Target: filepath.Join(tmpDir, "target.txt"),
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

	p := &AlternatesPlugin{}
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

	// Verify hostname-specific file was selected (higher priority)
	expectedSource := "config/config." + hostname + "@work.txt"
	if cfg.Dotfiles[0].Source != expectedSource {
		t.Errorf("Source = %v, want %v (hostname should win over OS)", cfg.Dotfiles[0].Source, expectedSource)
	}
}

func TestAlternatesPlugin_Execute_NoAlternates(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	// Create test files
	testDir := filepath.Join(workDir, "config")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	// Create only base file (no alternates)
	baseFile := filepath.Join(testDir, "config.txt")
	if err := os.WriteFile(baseFile, []byte("base"), 0644); err != nil {
		t.Fatalf("Failed to create base file: %v", err)
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

	originalSource := "config/config.txt"
	cfg := &core.Config{
		Dotfiles: []core.DotfileEntry{
			{
				Source: originalSource,
				Target: filepath.Join(tmpDir, "target.txt"),
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

	p := &AlternatesPlugin{}
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

	// Verify source was not changed (no alternates found)
	if cfg.Dotfiles[0].Source != originalSource {
		t.Errorf("Source = %v, want %v (should remain unchanged)", cfg.Dotfiles[0].Source, originalSource)
	}

	// Verify no changes were recorded
	if len(execCtx.Changes) != 0 {
		t.Errorf("Expected 0 changes, got %d", len(execCtx.Changes))
	}
}

func TestAlternatesPlugin_Execute_ArchitectureMatch(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	// Create test files
	testDir := filepath.Join(workDir, "config")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	// Create base file
	baseFile := filepath.Join(testDir, "config.txt")
	if err := os.WriteFile(baseFile, []byte("base"), 0644); err != nil {
		t.Fatalf("Failed to create base file: %v", err)
	}

	// Create architecture-specific alternate
	archFile := filepath.Join(testDir, "config."+runtime.GOARCH+".txt")
	if err := os.WriteFile(archFile, []byte("arch-specific"), 0644); err != nil {
		t.Fatalf("Failed to create arch file: %v", err)
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
		Dotfiles: []core.DotfileEntry{
			{
				Source: "config/config.txt",
				Target: filepath.Join(tmpDir, "target.txt"),
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

	p := &AlternatesPlugin{}
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

	// Verify source was updated to architecture-specific file
	expectedSource := "config/config." + runtime.GOARCH + ".txt"
	if cfg.Dotfiles[0].Source != expectedSource {
		t.Errorf("Source = %v, want %v", cfg.Dotfiles[0].Source, expectedSource)
	}
}

func TestAlternatesPlugin_Execute_SkipsDirectoryMode(t *testing.T) {
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

	// Use directory mode entry (no Source set)
	cfg := &core.Config{
		Dotfiles: []core.DotfileEntry{
			{Directory: "zsh"},
			{Directory: "git"},
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

	p := &AlternatesPlugin{}
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

	// Verify no changes were made (directory entries are skipped)
	if len(execCtx.Changes) != 0 {
		t.Errorf("Expected 0 changes for directory-mode entries, got %d", len(execCtx.Changes))
	}

	// Verify entries remain unchanged
	if cfg.Dotfiles[0].Directory != "zsh" {
		t.Errorf("Directory[0] = %v, want 'zsh'", cfg.Dotfiles[0].Directory)
	}
}
