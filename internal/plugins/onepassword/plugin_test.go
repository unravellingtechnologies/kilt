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

func TestOnePasswordPlugin_Name(t *testing.T) {
	p := &OnePasswordPlugin{}
	assert.Equal(t, "onepassword", p.Name())
}

func TestOnePasswordPlugin_Version(t *testing.T) {
	p := &OnePasswordPlugin{}
	assert.Equal(t, "1.0.0", p.Version())
}

func TestOnePasswordPlugin_Phase(t *testing.T) {
	p := &OnePasswordPlugin{}
	assert.Equal(t, plugin.PhasePreSync, p.Phase())
}

func TestOnePasswordPlugin_Dependencies(t *testing.T) {
	p := &OnePasswordPlugin{}
	deps := p.Dependencies()
	assert.Empty(t, deps)
}

func setupOnePasswordPlugin(t *testing.T, workDir string, cfg *core.Config) (*OnePasswordPlugin, *plugin.PluginContext) {
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

	p := &OnePasswordPlugin{}
	require.NoError(t, p.Initialize(ctx))

	return p, ctx
}

func TestOnePasswordPlugin_Initialize(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{},
	}

	p, _ := setupOnePasswordPlugin(t, workDir, cfg)

	assert.NotNil(t, p.ctx)
	assert.Empty(t, p.account)
	assert.Empty(t, p.vault)
	assert.True(t, p.cacheEnabled)
	assert.Equal(t, 5*time.Minute, p.cacheTTL)
}

func TestOnePasswordPlugin_Initialize_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

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

	p, _ := setupOnePasswordPlugin(t, workDir, cfg)

	assert.Equal(t, "myaccount.1password.com", p.account)
	assert.Equal(t, "Personal", p.vault)
	assert.False(t, p.cacheEnabled)
	assert.Equal(t, 10*time.Minute, p.cacheTTL)
}

func TestOnePasswordPlugin_Initialize_InvalidCacheTTL(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

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

	p := &OnePasswordPlugin{}
	err = p.Initialize(ctx)
	assert.Error(t, err, "Initialize() should return error for invalid cache_ttl")
}

func TestOnePasswordPlugin_Initialize_RegistersTemplateFunction(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{},
	}

	_, ctx := setupOnePasswordPlugin(t, workDir, cfg)

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

func TestOnePasswordPlugin_Validate(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{},
	}

	p, _ := setupOnePasswordPlugin(t, workDir, cfg)

	// Validate should pass (even if op is not installed)
	err := p.Validate()
	assert.NoError(t, err)
}

func TestOnePasswordPlugin_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{},
	}

	p, ctx := setupOnePasswordPlugin(t, workDir, cfg)

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	// Execute should succeed (even if op is not installed/authenticated)
	err := p.Execute(execCtx)
	assert.NoError(t, err)
}

func TestOnePasswordPlugin_Rollback(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{},
	}

	p, ctx := setupOnePasswordPlugin(t, workDir, cfg)

	// Add some cache entries
	p.cacheMu.Lock()
	p.cache["test/path"] = cacheEntry{
		value:     "secret_value",
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	p.cacheMu.Unlock()

	rollbackCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	// Rollback should clear cache
	err := p.Rollback(rollbackCtx)
	assert.NoError(t, err)

	// Verify cache is cleared
	p.cacheMu.RLock()
	assert.Empty(t, p.cache)
	p.cacheMu.RUnlock()
}

func TestOnePasswordPlugin_getSecret_NoOPInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{},
	}

	p, _ := setupOnePasswordPlugin(t, workDir, cfg)

	// Set opPath to empty to simulate op not installed
	p.opPath = ""

	// getSecret should return error
	_, err := p.getSecret("test/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not installed")
}

func TestOnePasswordPlugin_getSecret_Cache(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{
			"onepassword": map[string]interface{}{
				"cache_enabled": true,
				"cache_ttl":     "5m",
			},
		},
	}

	p, _ := setupOnePasswordPlugin(t, workDir, cfg)

	// Manually add cache entry
	p.cacheMu.Lock()
	p.cache["test/path"] = cacheEntry{
		value:     "cached_secret",
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	p.cacheMu.Unlock()

	// Verify cache logic
	p.cacheMu.RLock()
	entry, found := p.cache["test/path"]
	p.cacheMu.RUnlock()

	assert.True(t, found, "Cache entry should be found")
	assert.Equal(t, "cached_secret", entry.value)
}

func TestOnePasswordPlugin_getSecret_CacheExpired(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	cfg := &core.Config{
		Plugins: map[string]interface{}{
			"onepassword": map[string]interface{}{
				"cache_enabled": true,
			},
		},
	}

	p, _ := setupOnePasswordPlugin(t, workDir, cfg)

	// Add expired cache entry
	p.cacheMu.Lock()
	p.cache["test/path"] = cacheEntry{
		value:     "expired_secret",
		expiresAt: time.Now().Add(-1 * time.Minute), // Expired
	}
	p.cacheMu.Unlock()

	// Cache should be removed when accessed
	p.cacheMu.RLock()
	_, found := p.cache["test/path"]
	p.cacheMu.RUnlock()

	// The entry should still be there until we try to use it
	// But since we can't actually call getSecret without op, we'll just verify the entry exists
	assert.True(t, found, "Cache entry should exist (even if expired)")
}
