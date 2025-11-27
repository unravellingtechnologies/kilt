package git

import (
	"os"
	"os/exec"
	"path/filepath"
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

func TestGitPlugin_Name(t *testing.T) {
	p := &GitPlugin{}
	if p.Name() != "git" {
		t.Errorf("Name() = %v, want 'git'", p.Name())
	}
}

func TestGitPlugin_Version(t *testing.T) {
	p := &GitPlugin{}
	if p.Version() != "1.0.0" {
		t.Errorf("Version() = %v, want '1.0.0'", p.Version())
	}
}

func TestGitPlugin_Phase(t *testing.T) {
	p := &GitPlugin{}
	if p.Phase() != plugin.PhasePreSync {
		t.Errorf("Phase() = %v, want PhasePreSync", p.Phase())
	}
}

func TestGitPlugin_Dependencies(t *testing.T) {
	p := &GitPlugin{}
	deps := p.Dependencies()
	if len(deps) != 0 {
		t.Errorf("Dependencies() = %v, want empty slice", deps)
	}
}

func TestGitPlugin_Initialize(t *testing.T) {
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
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
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

	p := &GitPlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	if p.ctx == nil {
		t.Error("Plugin context should be set")
	}

	// Check defaults
	if p.autoPull != true {
		t.Errorf("autoPull = %v, want true", p.autoPull)
	}
}

func TestGitPlugin_Initialize_WithConfig(t *testing.T) {
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

	sshKey := filepath.Join(tmpDir, "id_ed25519")
	if err := os.WriteFile(sshKey, []byte("test key"), 0600); err != nil {
		t.Fatalf("Failed to create test SSH key: %v", err)
	}

	cfg := &core.Config{
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
		Plugins: map[string]interface{}{
			"git": map[string]interface{}{
				"auto_pull": false,
				"ssh_key":   sshKey,
				"git_token": "test_token",
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

	p := &GitPlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	if p.autoPull != false {
		t.Errorf("autoPull = %v, want false", p.autoPull)
	}

	if p.sshKey != sshKey {
		t.Errorf("sshKey = %v, want %v", p.sshKey, sshKey)
	}

	if p.gitToken != "test_token" {
		t.Errorf("gitToken = %v, want 'test_token'", p.gitToken)
	}
}

func TestGitPlugin_Validate_RepositoryNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")

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
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
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

	p := &GitPlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Validate should fail
	if err := p.Validate(); err == nil {
		t.Error("Validate() should return error when repository doesn't exist")
	}
}

func TestGitPlugin_Validate_NotGitRepository(t *testing.T) {
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
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
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

	p := &GitPlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Validate should fail (not a git repo)
	if err := p.Validate(); err == nil {
		t.Error("Validate() should return error when path is not a git repository")
	}
}

func TestGitPlugin_Validate_ValidRepository(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = workDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
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
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
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

	p := &GitPlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Validate should pass
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestGitPlugin_Execute_ExtraRepos_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = workDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Disable auto_pull to avoid remote fetch issues
	cfg := &core.Config{
		DotfilesPath: workDir,
		ExtraRepos: []core.Repository{
			{
				URL:    "https://github.com/test/repo.git",
				Path:   filepath.Join(tmpDir, "extra_repo"),
				Branch: "main",
				Sparse: false,
			},
		},
		Plugins: map[string]interface{}{
			"git": map[string]interface{}{
				"auto_pull": false,
			},
		},
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

	p := &GitPlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

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
		if change.Type == "git_clone" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected git_clone change to be recorded in dry-run")
	}
}

func TestGitPlugin_authenticateURL(t *testing.T) {
	p := &GitPlugin{}

	// Test HTTPS URL without token
	url := "https://github.com/user/repo.git"
	result := p.authenticateURL(url)
	if result != url {
		t.Errorf("authenticateURL() = %v, want %v", result, url)
	}

	// Test HTTPS URL with token
	p.gitToken = "test_token"
	result = p.authenticateURL(url)
	expected := "https://test_token@github.com/user/repo.git"
	if result != expected {
		t.Errorf("authenticateURL() = %v, want %v", result, expected)
	}

	// Test SSH URL (should not be modified)
	sshURL := "git@github.com:user/repo.git"
	result = p.authenticateURL(sshURL)
	if result != sshURL {
		t.Errorf("authenticateURL() = %v, want %v", result, sshURL)
	}
}

func TestGitPlugin_Rollback(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = workDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
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
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
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

	p := &GitPlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Simulate cloned repos
	clonedRepo := filepath.Join(tmpDir, "cloned_repo")
	if err := os.MkdirAll(clonedRepo, 0755); err != nil {
		t.Fatalf("Failed to create cloned repo: %v", err)
	}
	p.clonedRepos = []string{clonedRepo}

	rollbackCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	// Rollback
	if err := p.Rollback(rollbackCtx); err != nil {
		t.Fatalf("Rollback() error = %v, want nil", err)
	}

	// Verify cloned repo was removed
	if _, err := os.Stat(clonedRepo); err == nil {
		t.Error("Cloned repository should be removed after rollback")
	}

	// Verify rollback changes were recorded
	if len(rollbackCtx.Changes) == 0 {
		t.Error("Expected rollback changes to be recorded")
	}
}

