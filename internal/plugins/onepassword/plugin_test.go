package onepassword

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
	assert.Equal(t, "onepassword", p.Name())
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
		Plugins: map[string]interface{}{},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	assert.NotNil(t, p.ctx)
	assert.Empty(t, p.account)
	assert.Empty(t, p.vault)
	assert.True(t, p.cacheEnabled)
	assert.Equal(t, 5*time.Minute, p.cacheTTL)
}

func TestPlugin_Initialise_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{
			"onepassword": map[string]interface{}{
				"account":       "myaccount.1password.com",
				"vault":         "Personal",
				"cache_enabled": false,
				"cache_ttl":     "10m",
			},
		},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	assert.Equal(t, "myaccount.1password.com", p.account)
	assert.Equal(t, "Personal", p.vault)
	assert.False(t, p.cacheEnabled)
	assert.Equal(t, 10*time.Minute, p.cacheTTL)
}

func TestPlugin_Initialise_InvalidCacheTTL(t *testing.T) {
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
		Plugins: map[string]interface{}{
			"onepassword": map[string]interface{}{
				"cache_ttl": "invalid",
			},
		},
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
	assert.Error(t, err, "Initialise() should return error for invalid cache_ttl")
}

func TestPlugin_Initialise_RegistersTemplateFunction(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{},
	}

	_, ctx := setupPlugin(t, workDir, cfg)

	// Verify template function is registered by trying to use it
	// Since op might not be installed, we expect an error, but the function should be registered
	templateStr := "Secret: {{ op \"path/to/secret\" }}"
	_, err := ctx.Template.RenderString(templateStr)
	// Error is expected if op is not installed/authenticated, but function should be registered
	if err != nil {
		// Error should NOT mention "function not registered" - that would mean it wasn't registered
		assert.NotContains(t, err.Error(), "1Password function not registered")
	}
	// If no error, that's fine - op might be working
}

func TestPlugin_Validate(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	// Validate should pass (even if op is not installed)
	err := p.Validate()
	assert.NoError(t, err)
}

func TestPlugin_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{},
	}

	p, ctx := setupPlugin(t, workDir, cfg)

	execCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	// Execute should succeed (even if op is not installed/authenticated)
	err := p.Execute(execCtx)
	assert.NoError(t, err)
}

func TestPlugin_Rollback(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{},
	}

	p, ctx := setupPlugin(t, workDir, cfg)

	// Add some cache entries
	p.cacheMu.Lock()
	p.cache["test/path"] = cacheEntry{
		value:     "secret_value",
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	p.cacheMu.Unlock()

	rollbackCtx := &plugin.ExecutionContext{
		Context: ctx,
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	// Rollback should clear cache
	err := p.Rollback(rollbackCtx)
	assert.NoError(t, err)

	// Verify cache is cleared
	p.cacheMu.RLock()
	assert.Empty(t, p.cache)
	p.cacheMu.RUnlock()
}

func TestPlugin_getSecret_NoOPInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	// Set opPath to empty to simulate op not installed
	p.opPath = ""

	// getSecret should return error
	_, err := p.getSecret("test/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not installed")
}

func TestPlugin_getSecret_Cache(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{
			"onepassword": map[string]interface{}{
				"cache_enabled": true,
				"cache_ttl":     "5m",
			},
		},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	// Manually add cache entry
	p.cacheMu.Lock()
	p.cache["test/path"] = cacheEntry{
		value:     "cached_secret",
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	p.cacheMu.Unlock()

	// Ensure getSecret returns cached value without calling op
	p.opPath = "op" // non-empty to bypass missing CLI check
	p.authenticated.Store(true)

	val, err := p.getSecret("test/path")
	assert.NoError(t, err)
	assert.Equal(t, "cached_secret", val)
}

func TestPlugin_CacheEntryCanBeExpired(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{
			"onepassword": map[string]interface{}{
				"cache_enabled": true,
			},
		},
	}

	p, _ := setupPlugin(t, workDir, cfg)

	// Add expired cache entry
	p.cacheMu.Lock()
	p.cache["test/path"] = cacheEntry{
		value:     "expired_secret",
		expiresAt: time.Now().Add(-1 * time.Minute), // Expired
	}
	p.cacheMu.Unlock()

	// Expired entry can exist in map; actual removal requires op invocation
	p.cacheMu.RLock()
	_, found := p.cache["test/path"]
	p.cacheMu.RUnlock()

	// TODO: add a cache-expiry behaviour test once op command can be mocked
	assert.True(t, found, "Expired cache entry should remain stored until accessed")
}
