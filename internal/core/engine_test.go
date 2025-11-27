package core

import (
	"errors"
	"testing"

	"github.com/unravelling/kilt/internal/plugin"
)

// MockPlugin is a test implementation of the Plugin interface
type MockPlugin struct {
	name         string
	version      string
	description  string
	dependencies []string
	phase        plugin.ExecutionPhase
	initError    error
	validateError error
	executeError error
	initialized  bool
	executed     bool
}

func (m *MockPlugin) Name() string                                   { return m.name }
func (m *MockPlugin) Version() string                                { return m.version }
func (m *MockPlugin) Description() string                            { return m.description }
func (m *MockPlugin) Dependencies() []string                         { return m.dependencies }
func (m *MockPlugin) Phase() plugin.ExecutionPhase                   { return m.phase }
func (m *MockPlugin) Initialize(ctx *plugin.PluginContext) error     { m.initialized = true; return m.initError }
func (m *MockPlugin) Validate() error                                { return m.validateError }
func (m *MockPlugin) Execute(ctx *plugin.ExecutionContext) error     { m.executed = true; return m.executeError }
func (m *MockPlugin) Rollback(ctx *plugin.ExecutionContext) error    { return nil }

func TestNewEngine(t *testing.T) {
	// Create a minimal config
	cfg := &Config{
		TemplateEngine: "go",
		Files:          []FileMapping{},
	}

	// Create registry
	registry := plugin.NewRegistry()

	engine, err := NewEngine(cfg, registry)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	if engine == nil {
		t.Fatal("Engine should not be nil")
	}

	if engine.config != cfg {
		t.Error("Engine config should match provided config")
	}

	if engine.state == nil {
		t.Error("Engine state manager should not be nil")
	}

	if engine.backup == nil {
		t.Error("Engine backup manager should not be nil")
	}

	if engine.template == nil {
		t.Error("Engine template engine should not be nil")
	}
}

func TestNewEngine_NilConfig(t *testing.T) {
	registry := plugin.NewRegistry()
	_, err := NewEngine(nil, registry)
	if err == nil {
		t.Error("Expected error when config is nil")
	}
}

func TestNewEngine_NilRegistry(t *testing.T) {
	cfg := &Config{}
	_, err := NewEngine(cfg, nil)
	if err == nil {
		t.Error("Expected error when registry is nil")
	}
}

func TestEngine_PreflightChecks(t *testing.T) {
	cfg := &Config{
		TemplateEngine: "go",
		Files:          []FileMapping{},
	}
	registry := plugin.NewRegistry()

	engine, err := NewEngine(cfg, registry)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Pre-flight checks should pass with minimal config
	if err := engine.PreflightChecks(); err != nil {
		t.Fatalf("Pre-flight checks should pass: %v", err)
	}
}

func TestEngine_Execute_NoPlugins(t *testing.T) {
	cfg := &Config{
		TemplateEngine: "go",
		Files:          []FileMapping{},
	}
	registry := plugin.NewRegistry()

	engine, err := NewEngine(cfg, registry)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute should succeed with no plugins: %v", err)
	}

	if !result.Success {
		t.Error("Result should be successful with no plugins")
	}

	if result.PluginsRun != 0 {
		t.Errorf("Expected 0 plugins run, got %d", result.PluginsRun)
	}
}

func TestEngine_Execute_WithPlugins(t *testing.T) {
	cfg := &Config{
		TemplateEngine: "go",
		Files:          []FileMapping{},
	}
	registry := plugin.NewRegistry()

	// Register a mock plugin
	plugin1 := &MockPlugin{
		name:        "test-plugin",
		version:     "1.0.0",
		description: "Test plugin",
		phase:       plugin.PhaseCore,
	}
	registry.Register(plugin1)

	engine, err := NewEngine(cfg, registry)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute should succeed: %v", err)
	}

	if !result.Success {
		t.Error("Result should be successful")
	}

	if result.PluginsRun != 1 {
		t.Errorf("Expected 1 plugin run, got %d", result.PluginsRun)
	}

	if !plugin1.initialized {
		t.Error("Plugin should have been initialized")
	}

	if !plugin1.executed {
		t.Error("Plugin should have been executed")
	}
}

func TestEngine_Execute_PluginValidationFailure(t *testing.T) {
	cfg := &Config{
		TemplateEngine: "go",
		Files:          []FileMapping{},
	}
	registry := plugin.NewRegistry()

	// Register a plugin that fails validation
	plugin1 := &MockPlugin{
		name:          "test-plugin",
		version:       "1.0.0",
		description:   "Test plugin",
		phase:         plugin.PhaseCore,
		validateError: errors.New("validation failed"),
	}
	registry.Register(plugin1)

	engine, err := NewEngine(cfg, registry)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	result, err := engine.Execute()
	if err != nil {
		// Execution may continue with errors
		t.Logf("Execute returned error (may be expected): %v", err)
	}

	// Plugin should still be initialized
	if !plugin1.initialized {
		t.Error("Plugin should have been initialized")
	}

	// Plugin should not be executed if validation fails
	// (The engine continues execution, but records errors)
	if result != nil && len(result.Errors) == 0 {
		t.Error("Expected errors from validation failure")
	}
}

func TestEngine_Execute_DryRun(t *testing.T) {
	cfg := &Config{
		TemplateEngine: "go",
		Files:          []FileMapping{},
	}
	registry := plugin.NewRegistry()

	plugin1 := &MockPlugin{
		name:        "test-plugin",
		version:     "1.0.0",
		description: "Test plugin",
		phase:       plugin.PhaseCore,
	}
	registry.Register(plugin1)

	engine, err := NewEngine(cfg, registry)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	engine.SetDryRun(true)

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute should succeed in dry-run: %v", err)
	}

	if result.PluginsRun != 1 {
		t.Errorf("Expected 1 plugin reported in dry-run, got %d", result.PluginsRun)
	}

	// Plugin should be initialized but not executed in dry-run
	if !plugin1.initialized {
		t.Error("Plugin should have been initialized")
	}

	if plugin1.executed {
		t.Error("Plugin should NOT have been executed in dry-run mode")
	}
}

func TestEngine_GetExecutionPlan(t *testing.T) {
	cfg := &Config{
		TemplateEngine: "go",
		Files:          []FileMapping{},
	}
	registry := plugin.NewRegistry()

	plugin1 := &MockPlugin{
		name:        "plugin1",
		phase:       plugin.PhaseCore,
	}
	plugin2 := &MockPlugin{
		name:        "plugin2",
		phase:       plugin.PhaseIntegration,
	}
	registry.Register(plugin1)
	registry.Register(plugin2)

	engine, err := NewEngine(cfg, registry)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	plan, err := engine.GetExecutionPlan()
	if err != nil {
		t.Fatalf("GetExecutionPlan failed: %v", err)
	}

	if len(plan) != 2 {
		t.Errorf("Expected 2 plugins in plan, got %d", len(plan))
	}

	// PhaseCore should come before PhaseIntegration
	if plan[0].Phase() != plugin.PhaseCore {
		t.Errorf("Expected first plugin to be PhaseCore, got %v", plan[0].Phase())
	}
}

func TestEngine_Cancel(t *testing.T) {
	cfg := &Config{
		TemplateEngine: "go",
		Files:          []FileMapping{},
	}
	registry := plugin.NewRegistry()

	engine, err := NewEngine(cfg, registry)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Cancel should not panic
	engine.Cancel()

	// Verify context is cancelled
	select {
	case <-engine.executionCtx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled")
	}
}

func TestEngine_SetDryRun(t *testing.T) {
	cfg := &Config{
		TemplateEngine: "go",
		Files:          []FileMapping{},
	}
	registry := plugin.NewRegistry()

	engine, err := NewEngine(cfg, registry)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	if engine.dryRun {
		t.Error("Engine should not be in dry-run mode by default")
	}

	engine.SetDryRun(true)
	if !engine.dryRun {
		t.Error("Engine should be in dry-run mode after SetDryRun(true)")
	}

	engine.SetDryRun(false)
	if engine.dryRun {
		t.Error("Engine should not be in dry-run mode after SetDryRun(false)")
	}
}

func TestEngine_SetVerbose(t *testing.T) {
	cfg := &Config{
		TemplateEngine: "go",
		Files:          []FileMapping{},
	}
	registry := plugin.NewRegistry()

	engine, err := NewEngine(cfg, registry)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	if engine.verbose {
		t.Error("Engine should not be verbose by default")
	}

	engine.SetVerbose(true)
	if !engine.verbose {
		t.Error("Engine should be verbose after SetVerbose(true)")
	}
}

