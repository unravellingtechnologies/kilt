package plugin

import (
	"errors"
	"testing"
)

// mockPluginContext creates a minimal PluginContext for testing
func mockPluginContext() *PluginContext {
	return &PluginContext{
		Config:   &mockConfig{pluginConfigs: make(map[string]map[string]interface{})},
		State:    nil,
		Backup:   nil,
		Template: nil,
		Logger:   &mockLogger{},
		DryRun:   false,
		WorkDir:  "/tmp/test",
		HomeDir:  "/home/test",
	}
}

func TestPluginLoader_LoadPlugins(t *testing.T) {
	registry := NewRegistry()
	loader := NewLoader(registry)

	// Create a mock plugin that initializes successfully
	plugin := &mockPlugin{
		name:        "test-plugin",
		version:     "1.0.0",
		description: "Test plugin",
		phase:       PhaseCore,
	}

	registry.Register(plugin)

	ctx := mockPluginContext()

	err := loader.LoadPlugins(ctx)
	if err != nil {
		t.Fatalf("LoadPlugins() error = %v, want nil", err)
	}

	if !plugin.initialized {
		t.Error("LoadPlugins() plugin.Initialize() was not called")
	}

	if !plugin.validated {
		t.Error("LoadPlugins() plugin.Validate() was not called")
	}
}

func TestPluginLoader_LoadPlugins_InitializationError(t *testing.T) {
	registry := NewRegistry()
	loader := NewLoader(registry)

	// Create a plugin that fails to initialize
	plugin := &failingMockPlugin{
		mockPlugin: mockPlugin{
			name:        "failing-plugin",
			version:     "1.0.0",
			description: "Failing plugin",
			phase:       PhaseCore,
		},
		initError: errors.New("initialization failed"),
	}

	registry.Register(plugin)

	ctx := mockPluginContext()

	err := loader.LoadPlugins(ctx)
	if err == nil {
		t.Error("LoadPlugins() error = nil, want error")
	}
}

func TestPluginLoader_LoadPlugins_ValidationError(t *testing.T) {
	registry := NewRegistry()
	loader := NewLoader(registry)

	// Create a plugin that fails validation
	plugin := &failingMockPlugin{
		mockPlugin: mockPlugin{
			name:        "invalid-plugin",
			version:     "1.0.0",
			description: "Invalid plugin",
			phase:       PhaseCore,
		},
		validateError: errors.New("validation failed"),
	}

	registry.Register(plugin)

	ctx := mockPluginContext()

	err := loader.LoadPlugins(ctx)
	if err == nil {
		t.Error("LoadPlugins() error = nil, want error")
	}
}

func TestRegisterPlugin(t *testing.T) {
	// Reset default registry
	defaultRegistry = NewRegistry()

	plugin := &mockPlugin{
		name:    "default-plugin",
		version: "1.0.0",
		phase:   PhaseCore,
	}

	err := RegisterPlugin(plugin)
	if err != nil {
		t.Fatalf("RegisterPlugin() error = %v, want nil", err)
	}

	// Verify it's in the default registry
	reg := GetDefaultRegistry()
	got, err := reg.Get("default-plugin")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if got.Name() != "default-plugin" {
		t.Errorf("Get() = %v, want default-plugin", got.Name())
	}
}

// failingMockPlugin is a plugin that can fail initialization or validation
type failingMockPlugin struct {
	mockPlugin
	initError     error
	validateError error
}

func (f *failingMockPlugin) Initialize(ctx *PluginContext) error {
	if f.initError != nil {
		return f.initError
	}
	return f.mockPlugin.Initialize(ctx)
}

func (f *failingMockPlugin) Validate() error {
	if f.validateError != nil {
		return f.validateError
	}
	return f.mockPlugin.Validate()
}

