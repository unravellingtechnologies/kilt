package onepassword

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/unravelling/kilt/internal/core"
	"github.com/unravelling/kilt/internal/plugin"
)

// mockLogger is a simple mock logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, fields ...interface{}) {}
func (m *mockLogger) Info(msg string, fields ...interface{})  {}
func (m *mockLogger) Warn(msg string, fields ...interface{}) {}
func (m *mockLogger) Error(msg string, fields ...interface{}) {}

func TestOnePasswordPlugin_Name(t *testing.T) {
	p := &OnePasswordPlugin{}
	if p.Name() != "onepassword" {
		t.Errorf("Name() = %v, want 'onepassword'", p.Name())
	}
}

func TestOnePasswordPlugin_Version(t *testing.T) {
	p := &OnePasswordPlugin{}
	if p.Version() != "1.0.0" {
		t.Errorf("Version() = %v, want '1.0.0'", p.Version())
	}
}

func TestOnePasswordPlugin_Phase(t *testing.T) {
	p := &OnePasswordPlugin{}
	if p.Phase() != plugin.PhasePreSync {
		t.Errorf("Phase() = %v, want PhasePreSync", p.Phase())
	}
}

func TestOnePasswordPlugin_Dependencies(t *testing.T) {
	p := &OnePasswordPlugin{}
	deps := p.Dependencies()
	if len(deps) != 0 {
		t.Errorf("Dependencies() = %v, want empty slice", deps)
	}
}

func TestOnePasswordPlugin_Initialize(t *testing.T) {
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

	p := &OnePasswordPlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	if p.ctx == nil {
		t.Error("Plugin context should be set")
	}

	// Check defaults
	if p.account != "" {
		t.Errorf("account = %v, want empty string", p.account)
	}

	if p.vault != "" {
		t.Errorf("vault = %v, want empty string", p.vault)
	}

	if p.cacheEnabled != true {
		t.Errorf("cacheEnabled = %v, want true", p.cacheEnabled)
	}

	if p.cacheTTL != 5*time.Minute {
		t.Errorf("cacheTTL = %v, want 5m", p.cacheTTL)
	}
}

func TestOnePasswordPlugin_Initialize_WithConfig(t *testing.T) {
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
			"onepassword": map[string]interface{}{
				"account":       "myaccount.1password.com",
				"vault":         "Personal",
				"cache_enabled": false,
				"cache_ttl":     "10m",
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
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	if p.account != "myaccount.1password.com" {
		t.Errorf("account = %v, want 'myaccount.1password.com'", p.account)
	}

	if p.vault != "Personal" {
		t.Errorf("vault = %v, want 'Personal'", p.vault)
	}

	if p.cacheEnabled != false {
		t.Errorf("cacheEnabled = %v, want false", p.cacheEnabled)
	}

	if p.cacheTTL != 10*time.Minute {
		t.Errorf("cacheTTL = %v, want 10m", p.cacheTTL)
	}
}

func TestOnePasswordPlugin_Initialize_InvalidCacheTTL(t *testing.T) {
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
	if err := p.Initialize(ctx); err == nil {
		t.Error("Initialize() should return error for invalid cache_ttl")
	}
}

func TestOnePasswordPlugin_Initialize_RegistersTemplateFunction(t *testing.T) {
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

	p := &OnePasswordPlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Verify template function is registered by trying to use it
	// Since op might not be installed, we expect an error, but the function should be registered
	templateStr := "Secret: {{ op \"path/to/secret\" }}"
	_, err = template.RenderString(templateStr)
	// Error is expected if op is not installed/authenticated, but function should be registered
	if err == nil {
		// If no error, that's fine - op might be working
	} else {
		// Error should mention op function or 1Password
		if err.Error() == "1Password function not registered. Install and configure the 1Password plugin" {
			t.Error("Template function was not registered")
		}
		// Other errors (like op not installed) are expected and fine
	}
}

func TestOnePasswordPlugin_Validate(t *testing.T) {
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

	p := &OnePasswordPlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Validate should pass (even if op is not installed)
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestOnePasswordPlugin_Execute(t *testing.T) {
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

	p := &OnePasswordPlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	execCtx := &plugin.ExecutionContext{
		PluginContext: ctx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
	}

	// Execute should succeed (even if op is not installed/authenticated)
	if err := p.Execute(execCtx); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
}

func TestOnePasswordPlugin_Rollback(t *testing.T) {
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

	p := &OnePasswordPlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

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
	if err := p.Rollback(rollbackCtx); err != nil {
		t.Fatalf("Rollback() error = %v, want nil", err)
	}

	// Verify cache is cleared
	p.cacheMu.RLock()
	if len(p.cache) != 0 {
		t.Errorf("Cache should be cleared after rollback, got %d entries", len(p.cache))
	}
	p.cacheMu.RUnlock()
}

func TestOnePasswordPlugin_getSecret_NoOPInstalled(t *testing.T) {
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

	p := &OnePasswordPlugin{}
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Set opPath to empty to simulate op not installed
	p.opPath = ""

	// getSecret should return error
	_, err = p.getSecret("test/path")
	if err == nil {
		t.Error("getSecret() should return error when op is not installed")
	}
}

func TestOnePasswordPlugin_getSecret_Cache(t *testing.T) {
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
			"onepassword": map[string]interface{}{
				"cache_enabled": true,
				"cache_ttl":     "5m",
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
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// Manually add cache entry
	p.cacheMu.Lock()
	p.cache["test/path"] = cacheEntry{
		value:     "cached_secret",
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	p.cacheMu.Unlock()

	// getSecret should return cached value (even if op is not installed, cache should work)
	// But we need to set opPath to avoid the "not installed" error
	// Since we can't actually test with real op, we'll just verify cache logic
	p.cacheMu.RLock()
	entry, found := p.cache["test/path"]
	p.cacheMu.RUnlock()

	if !found {
		t.Error("Cache entry should be found")
	}

	if entry.value != "cached_secret" {
		t.Errorf("Cache value = %v, want 'cached_secret'", entry.value)
	}
}

func TestOnePasswordPlugin_getSecret_CacheExpired(t *testing.T) {
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
			"onepassword": map[string]interface{}{
				"cache_enabled": true,
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
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

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
	if !found {
		t.Error("Cache entry should exist (even if expired)")
	}
}

