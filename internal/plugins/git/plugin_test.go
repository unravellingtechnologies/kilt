package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func setupGitPlugin(t *testing.T, workDir string, cfg *core.Config) (*GitPlugin, *plugin.PluginContext) {
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

	p := &GitPlugin{}
	require.NoError(t, p.Initialize(ctx))

	return p, ctx
}

func TestGitPlugin_Name(t *testing.T) {
	p := &GitPlugin{}
	assert.Equal(t, "git", p.Name())
}

func TestGitPlugin_Version(t *testing.T) {
	p := &GitPlugin{}
	assert.Equal(t, "1.0.0", p.Version())
}

func TestGitPlugin_Phase(t *testing.T) {
	p := &GitPlugin{}
	assert.Equal(t, plugin.PhasePreSync, p.Phase())
}

func TestGitPlugin_Dependencies(t *testing.T) {
	p := &GitPlugin{}
	deps := p.Dependencies()
	assert.Empty(t, deps)
}

func TestGitPlugin_Initialize(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	cfg := &core.Config{
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
	}

	p, _ := setupGitPlugin(t, workDir, cfg)

	assert.NotNil(t, p.ctx)
	assert.Equal(t, true, p.autoPull)
}

func TestGitPlugin_Initialize_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	sshKey := filepath.Join(tmpDir, "id_ed25519")
	require.NoError(t, os.WriteFile(sshKey, []byte("test key"), 0600))

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

	p, _ := setupGitPlugin(t, workDir, cfg)

	assert.Equal(t, false, p.autoPull)
	assert.Equal(t, sshKey, p.sshKey)
	assert.Equal(t, "test_token", p.gitToken)
}

func TestGitPlugin_Initialize_DefaultPath(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	cfg := &core.Config{
		DotfilesPath: "", // Empty, should use default
		ExtraRepos:   []core.Repository{},
	}

	p, _ := setupGitPlugin(t, workDir, cfg)

	expectedPath := core.DefaultDotfilesPath()
	expandedPath, _ := core.ExpandPath(expectedPath)
	assert.Equal(t, expandedPath, p.repoPath)
}

func TestGitPlugin_Validate_RepositoryNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")

	cfg := &core.Config{
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
	}

	p, _ := setupGitPlugin(t, workDir, cfg)

	err := p.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dotfiles repository not found")
}

func TestGitPlugin_Validate_NotGitRepository(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	cfg := &core.Config{
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
	}

	p, _ := setupGitPlugin(t, workDir, cfg)

	err := p.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository")
}

func TestGitPlugin_Validate_ValidRepository(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	cfg := &core.Config{
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
	}

	p, _ := setupGitPlugin(t, workDir, cfg)

	err := p.Validate()
	assert.NoError(t, err)
}

func TestGitPlugin_Execute_AutoPullDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	cfg := &core.Config{
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
		Plugins: map[string]interface{}{
			"git": map[string]interface{}{
				"auto_pull": false,
			},
		},
	}

	p, ctx := setupGitPlugin(t, workDir, cfg)
	require.NoError(t, p.Validate())

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	err := p.Execute(execCtx)
	assert.NoError(t, err)
}

func TestGitPlugin_Execute_ExtraRepos_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

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

	p, ctx := setupGitPlugin(t, workDir, cfg)
	ctx.DryRun = true

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	err := p.Execute(execCtx)
	assert.NoError(t, err)

	// Should record dry-run changes
	found := false
	for _, change := range execCtx.Changes {
		if change.Type == "git_clone" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestGitPlugin_HasUncommittedChanges(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	// Configure git user for commit
	exec.Command("git", "config", "user.email", "test@example.com").Run()
	exec.Command("git", "config", "user.name", "Test User").Run()

	cfg := &core.Config{
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
	}

	p, _ := setupGitPlugin(t, workDir, cfg)

	// Initially no changes
	hasChanges, err := p.hasUncommittedChanges()
	assert.NoError(t, err)
	assert.False(t, hasChanges)

	// Create a file
	testFile := filepath.Join(workDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

	// Now should have changes
	hasChanges, err = p.hasUncommittedChanges()
	assert.NoError(t, err)
	assert.True(t, hasChanges)
}

func TestGitPlugin_HasRemoteChanges_NoRemote(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	cfg := &core.Config{
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
	}

	p, _ := setupGitPlugin(t, workDir, cfg)

	// No remote configured
	hasChanges, err := p.hasRemoteChanges()
	assert.NoError(t, err)
	assert.False(t, hasChanges)
}

func TestGitPlugin_GitFetch_NoRemote(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	cfg := &core.Config{
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
	}

	p, _ := setupGitPlugin(t, workDir, cfg)

	// Fetch with no remote should not error
	err := p.gitFetch(workDir)
	assert.NoError(t, err)
}

func TestGitPlugin_AuthenticateURL(t *testing.T) {
	p := &GitPlugin{}

	// Test HTTPS URL without token
	url := "https://github.com/user/repo.git"
	result := p.authenticateURL(url)
	assert.Equal(t, url, result)

	// Test HTTPS URL with token
	p.gitToken = "test_token"
	result = p.authenticateURL(url)
	expected := "https://test_token@github.com/user/repo.git"
	assert.Equal(t, expected, result)

	// Test SSH URL (should not be modified)
	sshURL := "git@github.com:user/repo.git"
	result = p.authenticateURL(sshURL)
	assert.Equal(t, sshURL, result)
}

func TestGitPlugin_HandleExtraRepository_ExistingRepo_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	// Create existing extra repo
	extraRepoPath := filepath.Join(tmpDir, "extra_repo")
	require.NoError(t, os.MkdirAll(extraRepoPath, 0755))
	cmd = exec.Command("git", "init")
	cmd.Dir = extraRepoPath
	require.NoError(t, cmd.Run())

	cfg := &core.Config{
		DotfilesPath: workDir,
		ExtraRepos: []core.Repository{
			{
				URL:  "https://github.com/test/repo.git",
				Path: extraRepoPath,
			},
		},
		Plugins: map[string]interface{}{
			"git": map[string]interface{}{
				"auto_pull": false,
			},
		},
	}

	p, ctx := setupGitPlugin(t, workDir, cfg)
	ctx.DryRun = true

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	err := p.handleExtraRepository(execCtx, cfg.ExtraRepos[0])
	assert.NoError(t, err)

	// Should record dry-run change
	found := false
	for _, change := range execCtx.Changes {
		if change.Type == "git_pull" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestGitPlugin_Execute_SyncMainRepo_NoChanges(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())
	
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	// Make initial commit to create a branch
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "README.md"), []byte("# Test"), 0644))
	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())
	
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	cfg := &core.Config{
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
		Plugins: map[string]interface{}{
			"git": map[string]interface{}{
				"auto_pull": true,
			},
		},
	}

	p, ctx := setupGitPlugin(t, workDir, cfg)
	require.NoError(t, p.Validate())

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	err := p.syncMainRepository(execCtx)
	assert.NoError(t, err)

	// Should record sync change
	found := false
	for _, change := range execCtx.Changes {
		if change.Type == "git_sync" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestGitPlugin_Rollback(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	cfg := &core.Config{
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
	}

	p, ctx := setupGitPlugin(t, workDir, cfg)

	// Simulate cloned repos
	clonedRepo := filepath.Join(tmpDir, "cloned_repo")
	require.NoError(t, os.MkdirAll(clonedRepo, 0755))
	p.clonedRepos = []string{clonedRepo}

	rollbackCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	err := p.Rollback(rollbackCtx)
	assert.NoError(t, err)

	// Verify cloned repo was removed
	_, err = os.Stat(clonedRepo)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))

	// Verify rollback changes were recorded
	assert.NotEmpty(t, rollbackCtx.Changes)

	// Verify cloned repos list was cleared
	assert.Empty(t, p.clonedRepos)
}

func TestGitPlugin_SetupGitAuth_SSHKey(t *testing.T) {
	tmpDir := t.TempDir()
	sshKey := filepath.Join(tmpDir, "id_ed25519")
	require.NoError(t, os.WriteFile(sshKey, []byte("test key"), 0600))

	p := &GitPlugin{
		sshKey: sshKey,
	}

	cmd := exec.Command("git", "status")
	p.setupGitAuth(cmd)

	// Check that GIT_SSH_COMMAND is set
	found := false
	for _, env := range cmd.Env {
		if strings.Contains(env, "GIT_SSH_COMMAND") {
			found = true
			assert.Contains(t, env, sshKey)
			break
		}
	}
	assert.True(t, found)
}

func TestGitPlugin_SetupGitAuth_Token(t *testing.T) {
	p := &GitPlugin{
		gitToken: "test_token",
	}

	cmd := exec.Command("git", "status")
	p.setupGitAuth(cmd)

	// Check that GIT_ASKPASS is set
	foundAskPass := false
	foundTerminal := false
	for _, env := range cmd.Env {
		if strings.Contains(env, "GIT_ASKPASS") {
			foundAskPass = true
		}
		if strings.Contains(env, "GIT_TERMINAL_PROMPT") {
			foundTerminal = true
		}
	}
	assert.True(t, foundAskPass || foundTerminal)
}
