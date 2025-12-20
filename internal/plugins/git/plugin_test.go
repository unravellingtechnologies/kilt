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

func TestPlugin_Name(t *testing.T) {
	p := &Plugin{}
	assert.Equal(t, "git", p.Name())
}

func TestPlugin_Version(t *testing.T) {
	p := &Plugin{}
	assert.Equal(t, "1.0.0", p.Version())
}

func TestPlugin_Phase(t *testing.T) {
	p := &Plugin{}
	assert.Equal(t, plugin.PhasePreSync, p.Phase())
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
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	assert.NotNil(t, p.ctx)
	assert.Equal(t, true, p.autoPull)
}

func TestPlugin_Initialise_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	sshKey := filepath.Join(tmpDir, "id_ed25519")
	require.NoError(t, os.WriteFile(sshKey, []byte("test key"), 0o600))

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

	p, _ := setupPlugin(t, workDir, cfg)

	assert.Equal(t, false, p.autoPull)
	assert.Equal(t, sshKey, p.sshKey)
	assert.Equal(t, "test_token", p.gitToken)
}

func TestPlugin_Initialise_DefaultPath(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		DotfilesPath: "", // Empty, should use default
		ExtraRepos:   []core.Repository{},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	expectedPath := core.DefaultDotfilesPath()
	expandedPath, _ := core.ExpandPath(expectedPath)
	assert.Equal(t, expandedPath, p.repoPath)
}

func TestPlugin_Validate_RepositoryNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")

	cfg := &core.Config{
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	err := p.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dotfiles repository not found")
}

func TestPlugin_Validate_NotGitRepository(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	err := p.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository")
}

func TestPlugin_Validate_ValidRepository(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Initialise git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	cfg := &core.Config{
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	err := p.Validate()
	assert.NoError(t, err)
}

func TestPlugin_Execute_AutoPullDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Initialise git repository
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

	p, ctx := setupPlugin(t, workDir, cfg)
	require.NoError(t, p.Validate())

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	err := p.Execute(execCtx)
	assert.NoError(t, err)
}

func TestPlugin_Execute_ExtraRepos_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Initialise git repository
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

	p, ctx := setupPlugin(t, workDir, cfg)
	ctx.DryRun = true

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
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

func TestPlugin_HasUncommittedChanges(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Initialise git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	// Configure git user for commit
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	cfg := &core.Config{
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	// Initially no changes
	hasChanges, err := p.hasUncommittedChanges()
	assert.NoError(t, err)
	assert.False(t, hasChanges)

	// Create a file
	testFile := filepath.Join(workDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0o644))

	// Now should have changes
	hasChanges, err = p.hasUncommittedChanges()
	assert.NoError(t, err)
	assert.True(t, hasChanges)
}

func TestPlugin_HasRemoteChanges_NoRemote(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Initialise git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	// Configure git user locally in the repo
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	// Create initial commit to establish HEAD and a branch
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "README.md"), []byte("# Test"), 0o644))
	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	cfg := &core.Config{
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	// No remote configured
	hasChanges, err := p.hasRemoteChanges()
	assert.NoError(t, err)
	assert.False(t, hasChanges)
}

func TestPlugin_GitFetch_NoRemote(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Initialise git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	cfg := &core.Config{
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	// Fetch with no remote should not error
	err := p.gitFetch(workDir)
	assert.NoError(t, err)
}

func TestPlugin_HandleExtraRepository_ExistingRepo_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Initialise git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	// Create existing extra repo
	extraRepoPath := filepath.Join(tmpDir, "extra_repo")
	require.NoError(t, os.MkdirAll(extraRepoPath, 0o755))
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

	p, ctx := setupPlugin(t, workDir, cfg)
	ctx.DryRun = true

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
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

func TestPlugin_Execute_SyncMainRepo_NoChanges(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Initialise git repository
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
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "README.md"), []byte("# Test"), 0o644))
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

	p, ctx := setupPlugin(t, workDir, cfg)
	require.NoError(t, p.Validate())

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
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

func TestPlugin_Rollback(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Initialise git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	cfg := &core.Config{
		DotfilesPath: workDir,
		ExtraRepos:   []core.Repository{},
	}

	p, ctx := setupPlugin(t, workDir, cfg)

	// Simulate cloned repos
	clonedRepo := filepath.Join(tmpDir, "cloned_repo")
	require.NoError(t, os.MkdirAll(clonedRepo, 0o755))
	p.clonedRepos = []string{clonedRepo}

	rollbackCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
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

func TestPlugin_SetupGitAuth_SSHKey(t *testing.T) {
	tmpDir := t.TempDir()
	sshKey := filepath.Join(tmpDir, "id_ed25519")
	require.NoError(t, os.WriteFile(sshKey, []byte("test key"), 0o600))

	p := &Plugin{
		sshKey: sshKey,
	}

	cmd := exec.Command("git", "status")
	_ = p.setupGitAuth(cmd) //nolint:errcheck // Test setup - errors handled by test framework

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

func TestPlugin_SetupGitAuth_Token(t *testing.T) {
	p := &Plugin{
		gitToken: "test_token",
	}

	cmd := exec.Command("git", "status")
	_ = p.setupGitAuth(cmd) //nolint:errcheck // Test setup - errors handled by test framework

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
