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

func TestPlugin_Name(t *testing.T) {
	p := &Plugin{}
	assert.Equal(t, "alternates", p.Name())
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
		Dotfiles: []core.DotfileEntry{},
	}

	p, _ := setupPlugin(t, workDir, cfg)
	assert.NotNil(t, p.ctx)
}

func TestPlugin_Execute_OSMatch(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Create test files
	testDir := filepath.Join(workDir, "config")
	require.NoError(t, os.MkdirAll(testDir, 0o755))

	// Create base file
	baseFile := filepath.Join(testDir, "config.txt")
	require.NoError(t, os.WriteFile(baseFile, []byte("base"), 0o644))

	// Create OS-specific alternate
	var osFile string
	if runtime.GOOS == "darwin" {
		osFile = filepath.Join(testDir, "config.mac.txt")
	} else {
		osFile = filepath.Join(testDir, "config.linux.txt")
	}
	require.NoError(t, os.WriteFile(osFile, []byte("os-specific"), 0o644))

	cfg := &core.Config{
		Dotfiles: []core.DotfileEntry{
			{
				Source: "config/config.txt",
				Target: filepath.Join(tmpDir, "target.txt"),
			},
		},
	}

	p, ctx := setupPlugin(t, workDir, cfg)

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	require.NoError(t, p.Execute(execCtx))

	// Verify source was updated to OS-specific file
	expectedSource := "config/config.mac.txt"
	if runtime.GOOS != "darwin" {
		expectedSource = "config/config.linux.txt"
	}

	assert.Equal(t, expectedSource, cfg.Dotfiles[0].Source)
	assert.Len(t, execCtx.Changes, 1)
}

func TestPlugin_Execute_HostnameMatch(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	hostname, _ := os.Hostname()

	// Create test files
	testDir := filepath.Join(workDir, "config")
	require.NoError(t, os.MkdirAll(testDir, 0o755))

	// Create base file
	baseFile := filepath.Join(testDir, "config.txt")
	require.NoError(t, os.WriteFile(baseFile, []byte("base"), 0o644))

	// Create hostname-specific alternate
	hostnameFile := filepath.Join(testDir, "config."+hostname+"@work.txt")
	require.NoError(t, os.WriteFile(hostnameFile, []byte("hostname-specific"), 0o644))

	cfg := &core.Config{
		Dotfiles: []core.DotfileEntry{
			{
				Source: "config/config.txt",
				Target: filepath.Join(tmpDir, "target.txt"),
			},
		},
	}

	p, ctx := setupPlugin(t, workDir, cfg)

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	require.NoError(t, p.Execute(execCtx))

	// Verify source was updated to hostname-specific file
	expectedSource := "config/config." + hostname + "@work.txt"
	assert.Equal(t, expectedSource, cfg.Dotfiles[0].Source)
}

func TestPlugin_Execute_Priority(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	hostname, _ := os.Hostname()

	// Create test files
	testDir := filepath.Join(workDir, "config")
	require.NoError(t, os.MkdirAll(testDir, 0o755))

	// Create base file
	baseFile := filepath.Join(testDir, "config.txt")
	require.NoError(t, os.WriteFile(baseFile, []byte("base"), 0o644))

	// Create OS-specific alternate
	var osFile string
	if runtime.GOOS == "darwin" {
		osFile = filepath.Join(testDir, "config.mac.txt")
	} else {
		osFile = filepath.Join(testDir, "config.linux.txt")
	}
	require.NoError(t, os.WriteFile(osFile, []byte("os-specific"), 0o644))

	// Create hostname-specific alternate (should win due to higher priority)
	hostnameFile := filepath.Join(testDir, "config."+hostname+"@work.txt")
	require.NoError(t, os.WriteFile(hostnameFile, []byte("hostname-specific"), 0o644))

	cfg := &core.Config{
		Dotfiles: []core.DotfileEntry{
			{
				Source: "config/config.txt",
				Target: filepath.Join(tmpDir, "target.txt"),
			},
		},
	}

	p, ctx := setupPlugin(t, workDir, cfg)

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	require.NoError(t, p.Execute(execCtx))

	// Verify hostname-specific file was selected (higher priority)
	expectedSource := "config/config." + hostname + "@work.txt"
	assert.Equal(t, expectedSource, cfg.Dotfiles[0].Source, "hostname should win over OS")
}

func TestPlugin_Execute_NoAlternates(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Create test files
	testDir := filepath.Join(workDir, "config")
	require.NoError(t, os.MkdirAll(testDir, 0o755))

	// Create only base file (no alternates)
	baseFile := filepath.Join(testDir, "config.txt")
	require.NoError(t, os.WriteFile(baseFile, []byte("base"), 0o644))

	originalSource := "config/config.txt"
	cfg := &core.Config{
		Dotfiles: []core.DotfileEntry{
			{
				Source: originalSource,
				Target: filepath.Join(tmpDir, "target.txt"),
			},
		},
	}

	p, ctx := setupPlugin(t, workDir, cfg)

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	require.NoError(t, p.Execute(execCtx))

	// Verify source was not changed (no alternates found)
	assert.Equal(t, originalSource, cfg.Dotfiles[0].Source, "should remain unchanged")
	assert.Empty(t, execCtx.Changes)
}

func TestPlugin_Execute_ArchitectureMatch(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Create test files
	testDir := filepath.Join(workDir, "config")
	require.NoError(t, os.MkdirAll(testDir, 0o755))

	// Create base file
	baseFile := filepath.Join(testDir, "config.txt")
	require.NoError(t, os.WriteFile(baseFile, []byte("base"), 0o644))

	// Create architecture-specific alternate
	archFile := filepath.Join(testDir, "config."+runtime.GOARCH+".txt")
	require.NoError(t, os.WriteFile(archFile, []byte("arch-specific"), 0o644))

	cfg := &core.Config{
		Dotfiles: []core.DotfileEntry{
			{
				Source: "config/config.txt",
				Target: filepath.Join(tmpDir, "target.txt"),
			},
		},
	}

	p, ctx := setupPlugin(t, workDir, cfg)

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	require.NoError(t, p.Execute(execCtx))

	// Verify source was updated to architecture-specific file
	expectedSource := "config/config." + runtime.GOARCH + ".txt"
	assert.Equal(t, expectedSource, cfg.Dotfiles[0].Source)
}

func TestPlugin_Execute_SkipsDirectoryMode(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Use directory mode entry (no Source set)
	cfg := &core.Config{
		Dotfiles: []core.DotfileEntry{
			{Directory: "zsh"},
			{Directory: "git"},
		},
	}

	p, ctx := setupPlugin(t, workDir, cfg)

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	require.NoError(t, p.Execute(execCtx))

	// Verify no changes were made (directory entries are skipped)
	assert.Empty(t, execCtx.Changes, "directory-mode entries should be skipped")

	// Verify entries remain unchanged
	assert.Equal(t, "zsh", cfg.Dotfiles[0].Directory)
}
