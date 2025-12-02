package dotfiles

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

func TestPlugin_Name(t *testing.T) {
	p := &Plugin{}
	assert.Equal(t, "dotfiles", p.Name())
}

func TestPlugin_Version(t *testing.T) {
	p := &Plugin{}
	assert.Equal(t, "1.0.0", p.Version())
}

func TestPlugin_Description(t *testing.T) {
	p := &Plugin{}
	assert.Equal(t, "Symlink-based dotfile synchronisation", p.Description())
}

func TestPlugin_Dependencies(t *testing.T) {
	p := &Plugin{}
	deps := p.Dependencies()
	assert.Equal(t, []string{"alternates"}, deps)
}

func TestPlugin_Phase(t *testing.T) {
	p := &Plugin{}
	assert.Equal(t, plugin.PhaseCore, p.Phase())
}

func TestPlugin_Initialise(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	require.NoError(t, err)

	template := core.NewTemplateEngine()
	homeDir, _ := os.UserHomeDir()

	cfg := &core.Config{
		DotfilesPath: "~/.dotfiles",
		Plugins:      map[string]interface{}{},
	}

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
	err = p.Initialise(ctx)
	require.NoError(t, err)

	assert.NotNil(t, p.ctx)
	assert.Equal(t, filepath.Join(homeDir, ".dotfiles"), p.dotfilesPath)
	assert.NotNil(t, p.linkedFiles)
}

func TestPlugin_Initialise_DefaultPath(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	require.NoError(t, err)

	template := core.NewTemplateEngine()
	homeDir, _ := os.UserHomeDir()

	cfg := &core.Config{
		DotfilesPath: "", // Empty, should use default
		Plugins:      map[string]interface{}{},
	}

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
	err = p.Initialise(ctx)
	require.NoError(t, err)

	expectedPath := core.DefaultDotfilesPath()
	expandedPath, _ := core.ExpandPath(expectedPath)
	assert.Equal(t, expandedPath, p.dotfilesPath)
}

func TestPlugin_Validate_Success(t *testing.T) {
	tmpDir := t.TempDir()
	dotfilesPath := filepath.Join(tmpDir, ".dotfiles")
	require.NoError(t, os.MkdirAll(dotfilesPath, 0o755))

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	require.NoError(t, err)

	template := core.NewTemplateEngine()
	homeDir, _ := os.UserHomeDir()

	cfg := &core.Config{
		DotfilesPath: dotfilesPath,
		Plugins:      map[string]interface{}{},
	}

	ctx := &plugin.Context{
		Config:   cfg,
		State:    state,
		Backup:   backup,
		Template: template,
		Logger:   &mockLogger{},
		DryRun:   false,
		WorkDir:  tmpDir,
		HomeDir:  homeDir,
	}

	p := &Plugin{}
	require.NoError(t, p.Initialise(ctx))
	err = p.Validate()
	assert.NoError(t, err)
}

func TestPlugin_Validate_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	dotfilesPath := filepath.Join(tmpDir, "nonexistent")

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	require.NoError(t, err)

	template := core.NewTemplateEngine()
	homeDir, _ := os.UserHomeDir()

	cfg := &core.Config{
		DotfilesPath: dotfilesPath,
		Plugins:      map[string]interface{}{},
	}

	ctx := &plugin.Context{
		Config:   cfg,
		State:    state,
		Backup:   backup,
		Template: template,
		Logger:   &mockLogger{},
		DryRun:   false,
		WorkDir:  tmpDir,
		HomeDir:  homeDir,
	}

	p := &Plugin{}
	require.NoError(t, p.Initialise(ctx))
	err = p.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dotfiles repository not found")
}

func TestPlugin_Execute_DirectoryMode(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	dotfilesPath := filepath.Join(tmpDir, ".dotfiles")
	zshDir := filepath.Join(dotfilesPath, "zsh")
	require.NoError(t, os.MkdirAll(zshDir, 0o755))

	// Create files in zsh directory
	zshrcFile := filepath.Join(zshDir, "zshrc")
	require.NoError(t, os.WriteFile(zshrcFile, []byte("# zsh config"), 0o644))

	targetPath := filepath.Join(homeDir, ".zshrc")

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	require.NoError(t, err)

	template := core.NewTemplateEngine()

	cfg := &core.Config{
		DotfilesPath: dotfilesPath,
		Dotfiles: []core.DotfileEntry{
			{Directory: "zsh"},
		},
		Plugins: map[string]interface{}{},
	}

	ctx := &plugin.Context{
		Config:   cfg,
		State:    state,
		Backup:   backup,
		Template: template,
		Logger:   &mockLogger{},
		DryRun:   false,
		WorkDir:  tmpDir,
		HomeDir:  homeDir,
	}

	p := &Plugin{}
	require.NoError(t, p.Initialise(ctx))

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	err = p.Execute(execCtx)
	require.NoError(t, err)

	// Verify symlink was created
	linkTarget, err := os.Readlink(targetPath)
	assert.NoError(t, err)
	absZshrc, _ := filepath.Abs(zshrcFile)
	absLink, _ := filepath.Abs(linkTarget)
	assert.Equal(t, absZshrc, absLink)

	// Verify change was recorded
	assert.Greater(t, len(execCtx.Changes), 0)
	found := false
	for _, change := range execCtx.Changes {
		if change.Type == "symlink" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestPlugin_Execute_DirectoryMode_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	dotfilesPath := filepath.Join(tmpDir, ".dotfiles")
	zshDir := filepath.Join(dotfilesPath, "zsh")
	require.NoError(t, os.MkdirAll(zshDir, 0o755))

	zshrcFile := filepath.Join(zshDir, "zshrc")
	require.NoError(t, os.WriteFile(zshrcFile, []byte("# zsh config"), 0o644))

	targetPath := filepath.Join(homeDir, ".zshrc")

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	require.NoError(t, err)

	template := core.NewTemplateEngine()

	cfg := &core.Config{
		DotfilesPath: dotfilesPath,
		Dotfiles: []core.DotfileEntry{
			{Directory: "zsh"},
		},
		Plugins: map[string]interface{}{},
	}

	ctx := &plugin.Context{
		Config:   cfg,
		State:    state,
		Backup:   backup,
		Template: template,
		Logger:   &mockLogger{},
		DryRun:   true,
		WorkDir:  tmpDir,
		HomeDir:  homeDir,
	}

	p := &Plugin{}
	require.NoError(t, p.Initialise(ctx))

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	err = p.Execute(execCtx)
	require.NoError(t, err)

	// Verify symlink was NOT created in dry-run
	_, err = os.Readlink(targetPath)
	assert.Error(t, err)

	// Verify change was recorded
	assert.Greater(t, len(execCtx.Changes), 0)
}

func TestPlugin_Execute_ExplicitMapping(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	dotfilesPath := filepath.Join(tmpDir, ".dotfiles")
	require.NoError(t, os.MkdirAll(dotfilesPath, 0o755))

	sourceFile := filepath.Join(dotfilesPath, "ssh", "config")
	require.NoError(t, os.MkdirAll(filepath.Dir(sourceFile), 0o755))
	require.NoError(t, os.WriteFile(sourceFile, []byte("Host *\n"), 0o644))

	targetPath := filepath.Join(homeDir, ".ssh", "config")

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	require.NoError(t, err)

	template := core.NewTemplateEngine()

	cfg := &core.Config{
		DotfilesPath: dotfilesPath,
		Dotfiles: []core.DotfileEntry{
			{
				Source: "ssh/config",
				Target: filepath.Join(homeDir, ".ssh", "config"),
			},
		},
		Plugins: map[string]interface{}{},
	}

	ctx := &plugin.Context{
		Config:   cfg,
		State:    state,
		Backup:   backup,
		Template: template,
		Logger:   &mockLogger{},
		DryRun:   false,
		WorkDir:  tmpDir,
		HomeDir:  homeDir,
	}

	p := &Plugin{}
	require.NoError(t, p.Initialise(ctx))

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	err = p.Execute(execCtx)
	require.NoError(t, err)

	// Verify symlink was created
	linkTarget, err := os.Readlink(targetPath)
	assert.NoError(t, err)
	absSource, _ := filepath.Abs(sourceFile)
	absLink, _ := filepath.Abs(linkTarget)
	assert.Equal(t, absSource, absLink)
}

func TestPlugin_Execute_TemplateFile(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	dotfilesPath := filepath.Join(tmpDir, ".dotfiles")
	require.NoError(t, os.MkdirAll(dotfilesPath, 0o755))

	sourceFile := filepath.Join(dotfilesPath, "config", "template.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(sourceFile), 0o755))
	templateContent := "Hello {{ .User }}, your home is {{ .Home }}"
	require.NoError(t, os.WriteFile(sourceFile, []byte(templateContent), 0o644))

	targetPath := filepath.Join(homeDir, ".config", "rendered.txt")

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	require.NoError(t, err)

	template := core.NewTemplateEngine()

	cfg := &core.Config{
		DotfilesPath: dotfilesPath,
		Dotfiles: []core.DotfileEntry{
			{
				Source:   "config/template.txt",
				Target:   filepath.Join(homeDir, ".config", "rendered.txt"),
				Template: true,
			},
		},
		Plugins: map[string]interface{}{},
	}

	ctx := &plugin.Context{
		Config:   cfg,
		State:    state,
		Backup:   backup,
		Template: template,
		Logger:   &mockLogger{},
		DryRun:   false,
		WorkDir:  tmpDir,
		HomeDir:  homeDir,
	}

	p := &Plugin{}
	require.NoError(t, p.Initialise(ctx))

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	err = p.Execute(execCtx)
	require.NoError(t, err)

	// Verify file was created (not symlink for templates)
	content, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	assert.NotEmpty(t, content)
	// Should not be a symlink
	_, err = os.Readlink(targetPath)
	assert.Error(t, err)

	// Verify change was recorded
	assert.Greater(t, len(execCtx.Changes), 0)
	found := false
	for _, change := range execCtx.Changes {
		if change.Type == "template_file" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestPlugin_Execute_TemplateFile_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	dotfilesPath := filepath.Join(tmpDir, ".dotfiles")
	require.NoError(t, os.MkdirAll(dotfilesPath, 0o755))

	sourceFile := filepath.Join(dotfilesPath, "template.txt")
	require.NoError(t, os.WriteFile(sourceFile, []byte("Template content"), 0o644))

	targetPath := filepath.Join(homeDir, ".template.txt")

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	require.NoError(t, err)

	template := core.NewTemplateEngine()

	cfg := &core.Config{
		DotfilesPath: dotfilesPath,
		Dotfiles: []core.DotfileEntry{
			{
				Source:   "template.txt",
				Target:   filepath.Join(homeDir, ".template.txt"),
				Template: true,
			},
		},
		Plugins: map[string]interface{}{},
	}

	ctx := &plugin.Context{
		Config:   cfg,
		State:    state,
		Backup:   backup,
		Template: template,
		Logger:   &mockLogger{},
		DryRun:   true,
		WorkDir:  tmpDir,
		HomeDir:  homeDir,
	}

	p := &Plugin{}
	require.NoError(t, p.Initialise(ctx))

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	err = p.Execute(execCtx)
	require.NoError(t, err)

	// Verify file was NOT created in dry-run
	_, err = os.ReadFile(targetPath)
	assert.Error(t, err)
}

func TestPlugin_Execute_SourceNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	dotfilesPath := filepath.Join(tmpDir, ".dotfiles")
	require.NoError(t, os.MkdirAll(dotfilesPath, 0o755))

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	require.NoError(t, err)

	template := core.NewTemplateEngine()

	cfg := &core.Config{
		DotfilesPath: dotfilesPath,
		Dotfiles: []core.DotfileEntry{
			{
				Source: "nonexistent/file",
				Target: filepath.Join(homeDir, ".target"),
			},
		},
		Plugins: map[string]interface{}{},
	}

	ctx := &plugin.Context{
		Config:   cfg,
		State:    state,
		Backup:   backup,
		Template: template,
		Logger:   &mockLogger{},
		DryRun:   false,
		WorkDir:  tmpDir,
		HomeDir:  homeDir,
	}

	p := &Plugin{}
	require.NoError(t, p.Initialise(ctx))

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	err = p.Execute(execCtx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source file not found")
}

func TestPlugin_Execute_InvalidEntry(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	dotfilesPath := filepath.Join(tmpDir, ".dotfiles")
	require.NoError(t, os.MkdirAll(dotfilesPath, 0o755))

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	require.NoError(t, err)

	template := core.NewTemplateEngine()

	cfg := &core.Config{
		DotfilesPath: dotfilesPath,
		Dotfiles: []core.DotfileEntry{
			{}, // Invalid: neither Directory nor Source/Target
		},
		Plugins: map[string]interface{}{},
	}

	ctx := &plugin.Context{
		Config:   cfg,
		State:    state,
		Backup:   backup,
		Template: template,
		Logger:   &mockLogger{},
		DryRun:   false,
		WorkDir:  tmpDir,
		HomeDir:  homeDir,
	}

	p := &Plugin{}
	require.NoError(t, p.Initialise(ctx))

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	err = p.Execute(execCtx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid dotfile entry")
}

func TestPlugin_Execute_ExistingFileBackup(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	dotfilesPath := filepath.Join(tmpDir, ".dotfiles")
	require.NoError(t, os.MkdirAll(dotfilesPath, 0o755))

	sourceFile := filepath.Join(dotfilesPath, "test.txt")
	require.NoError(t, os.WriteFile(sourceFile, []byte("new content"), 0o644))

	targetPath := filepath.Join(homeDir, ".test.txt")
	existingContent := []byte("existing content")
	require.NoError(t, os.WriteFile(targetPath, existingContent, 0o644))

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	require.NoError(t, err)

	template := core.NewTemplateEngine()

	cfg := &core.Config{
		DotfilesPath: dotfilesPath,
		Dotfiles: []core.DotfileEntry{
			{
				Source: "test.txt",
				Target: filepath.Join(homeDir, ".test.txt"),
			},
		},
		Plugins: map[string]interface{}{},
	}

	ctx := &plugin.Context{
		Config:   cfg,
		State:    state,
		Backup:   backup,
		Template: template,
		Logger:   &mockLogger{},
		DryRun:   false,
		WorkDir:  tmpDir,
		HomeDir:  homeDir,
	}

	p := &Plugin{}
	require.NoError(t, p.Initialise(ctx))

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	err = p.Execute(execCtx)
	require.NoError(t, err)

	// Verify symlink was created
	linkTarget, err := os.Readlink(targetPath)
	assert.NoError(t, err)
	absSource, _ := filepath.Abs(sourceFile)
	absLink, _ := filepath.Abs(linkTarget)
	assert.Equal(t, absSource, absLink)

	// Verify backup was created
	backups, err := backup.ListBackups()
	assert.NoError(t, err)
	assert.Greater(t, len(backups), 0)
}

func TestPlugin_Execute_ExistingSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	dotfilesPath := filepath.Join(tmpDir, ".dotfiles")
	require.NoError(t, os.MkdirAll(dotfilesPath, 0o755))

	sourceFile := filepath.Join(dotfilesPath, "test.txt")
	require.NoError(t, os.WriteFile(sourceFile, []byte("content"), 0o644))

	targetPath := filepath.Join(homeDir, ".test.txt")

	// Create existing symlink pointing to the same file
	absSource, _ := filepath.Abs(sourceFile)
	require.NoError(t, os.Symlink(absSource, targetPath))

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	require.NoError(t, err)

	template := core.NewTemplateEngine()

	cfg := &core.Config{
		DotfilesPath: dotfilesPath,
		Dotfiles: []core.DotfileEntry{
			{
				Source: "test.txt",
				Target: filepath.Join(homeDir, ".test.txt"),
			},
		},
		Plugins: map[string]interface{}{},
	}

	ctx := &plugin.Context{
		Config:   cfg,
		State:    state,
		Backup:   backup,
		Template: template,
		Logger:   &mockLogger{},
		DryRun:   false,
		WorkDir:  tmpDir,
		HomeDir:  homeDir,
	}

	p := &Plugin{}
	require.NoError(t, p.Initialise(ctx))

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	err = p.Execute(execCtx)
	require.NoError(t, err)

	// Verify symlink still exists and points to correct location
	linkTarget, err := os.Readlink(targetPath)
	assert.NoError(t, err)
	absLink, _ := filepath.Abs(linkTarget)
	assert.Equal(t, absSource, absLink)

	// Should not create additional symlinks
	assert.Equal(t, 0, len(execCtx.Changes))
}

func TestPlugin_Rollback(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	dotfilesPath := filepath.Join(tmpDir, ".dotfiles")
	require.NoError(t, os.MkdirAll(dotfilesPath, 0o755))

	sourceFile := filepath.Join(dotfilesPath, "test.txt")
	require.NoError(t, os.WriteFile(sourceFile, []byte("content"), 0o644))

	targetPath := filepath.Join(homeDir, ".test.txt")

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	require.NoError(t, err)

	template := core.NewTemplateEngine()

	cfg := &core.Config{
		DotfilesPath: dotfilesPath,
		Dotfiles: []core.DotfileEntry{
			{
				Source: "test.txt",
				Target: filepath.Join(homeDir, ".test.txt"),
			},
		},
		Plugins: map[string]interface{}{},
	}

	ctx := &plugin.Context{
		Config:   cfg,
		State:    state,
		Backup:   backup,
		Template: template,
		Logger:   &mockLogger{},
		DryRun:   false,
		WorkDir:  tmpDir,
		HomeDir:  homeDir,
	}

	p := &Plugin{}
	require.NoError(t, p.Initialise(ctx))

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	// Execute to create symlink
	err = p.Execute(execCtx)
	require.NoError(t, err)

	// Verify symlink exists
	_, err = os.Readlink(targetPath)
	assert.NoError(t, err)

	// Rollback
	err = p.Rollback(execCtx)
	require.NoError(t, err)

	// Verify symlink was removed
	_, err = os.Readlink(targetPath)
	assert.Error(t, err)

	// Verify linkedFiles was cleared
	assert.Equal(t, 0, len(p.linkedFiles))
}

func TestPlugin_DetermineTargetPath(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	dotfilesPath := filepath.Join(tmpDir, ".dotfiles")
	require.NoError(t, os.MkdirAll(dotfilesPath, 0o755))

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	require.NoError(t, err)

	template := core.NewTemplateEngine()

	cfg := &core.Config{
		DotfilesPath: dotfilesPath,
		Plugins:      map[string]interface{}{},
	}

	ctx := &plugin.Context{
		Config:   cfg,
		State:    state,
		Backup:   backup,
		Template: template,
		Logger:   &mockLogger{},
		DryRun:   false,
		WorkDir:  tmpDir,
		HomeDir:  homeDir,
	}

	p := &Plugin{}
	require.NoError(t, p.Initialise(ctx))

	tests := []struct {
		name     string
		dirName  string
		relPath  string
		expected string
	}{
		{
			name:     "common dotfile (zshrc)",
			dirName:  "zsh",
			relPath:  "zshrc",
			expected: filepath.Join(homeDir, ".zshrc"),
		},
		{
			name:     "already dot-prefixed",
			dirName:  "zsh",
			relPath:  ".zshrc",
			expected: filepath.Join(homeDir, ".zshrc"),
		},
		{
			name:     "nested path",
			dirName:  "config",
			relPath:  "nvim/init.vim",
			expected: filepath.Join(homeDir, ".nvim", "init.vim"),
		},
		{
			name:     "common dotfile in nested",
			dirName:  "zsh",
			relPath:  ".config/zsh/aliases",
			expected: filepath.Join(homeDir, ".config", "zsh", "aliases"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.determineTargetPath(tt.dirName, tt.relPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPlugin_GetFileMode(t *testing.T) {
	p := &Plugin{}

	tests := []struct {
		name     string
		modeStr  string
		expected os.FileMode
	}{
		{
			name:     "valid octal mode",
			modeStr:  "0644",
			expected: 0o644,
		},
		{
			name:     "empty string defaults",
			modeStr:  "",
			expected: 0o644,
		},
		{
			name:     "invalid mode defaults",
			modeStr:  "invalid",
			expected: 0o644,
		},
		{
			name:     "restrictive mode",
			modeStr:  "0600",
			expected: 0o600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.getFileMode(tt.modeStr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPlugin_ProcessDirectory_MissingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	dotfilesPath := filepath.Join(tmpDir, ".dotfiles")
	require.NoError(t, os.MkdirAll(dotfilesPath, 0o755))

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	require.NoError(t, err)

	template := core.NewTemplateEngine()

	cfg := &core.Config{
		DotfilesPath: dotfilesPath,
		Plugins:      map[string]interface{}{},
	}

	ctx := &plugin.Context{
		Config:   cfg,
		State:    state,
		Backup:   backup,
		Template: template,
		Logger:   &mockLogger{},
		DryRun:   false,
		WorkDir:  tmpDir,
		HomeDir:  homeDir,
	}

	p := &Plugin{}
	require.NoError(t, p.Initialise(ctx))

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	// Process non-existent directory
	err = p.processDirectory("nonexistent", execCtx)
	// Should not error, just skip
	assert.NoError(t, err)
}

func TestPlugin_Execute_DirectoryWithNestedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	dotfilesPath := filepath.Join(tmpDir, ".dotfiles")
	configDir := filepath.Join(dotfilesPath, "config")
	require.NoError(t, os.MkdirAll(configDir, 0o755))

	// Create nested structure
	nvimDir := filepath.Join(configDir, "nvim")
	require.NoError(t, os.MkdirAll(nvimDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "init.vim"), []byte("nvim config"), 0o644))

	stateDir := filepath.Join(tmpDir, ".kilt", "state")
	state, err := core.NewStateManager(stateDir)
	require.NoError(t, err)

	backupDir := filepath.Join(tmpDir, ".kilt", "backup")
	backup, err := core.NewBackupManager(backupDir)
	require.NoError(t, err)

	template := core.NewTemplateEngine()

	cfg := &core.Config{
		DotfilesPath: dotfilesPath,
		Dotfiles: []core.DotfileEntry{
			{Directory: "config"},
		},
		Plugins: map[string]interface{}{},
	}

	ctx := &plugin.Context{
		Config:   cfg,
		State:    state,
		Backup:   backup,
		Template: template,
		Logger:   &mockLogger{},
		DryRun:   false,
		WorkDir:  tmpDir,
		HomeDir:  homeDir,
	}

	p := &Plugin{}
	require.NoError(t, p.Initialise(ctx))

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	err = p.Execute(execCtx)
	require.NoError(t, err)

	// Verify nested file was linked
	// Note: determineTargetPath adds dot prefix to first part of relPath, not dirName
	expectedTarget := filepath.Join(homeDir, ".nvim", "init.vim")
	_, err = os.Readlink(expectedTarget)
	assert.NoError(t, err)
}
