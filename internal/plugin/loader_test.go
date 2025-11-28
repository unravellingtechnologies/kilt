package plugin

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	mockPlugin := &mockPlugin{
		name:        "test-plugin",
		version:     "1.0.0",
		description: "Test plugin",
		phase:       PhaseCore,
	}

	registry.Register(mockPlugin)

	ctx := mockPluginContext()

	err := loader.LoadPlugins(ctx)
	require.NoError(t, err)

	assert.True(t, mockPlugin.initialized, "LoadPlugins() plugin.Initialize() was not called")
	assert.True(t, mockPlugin.validated, "LoadPlugins() plugin.Validate() was not called")
}

func TestPluginLoader_LoadPlugins_InitializationError(t *testing.T) {
	registry := NewRegistry()
	loader := NewLoader(registry)

	// Create a plugin that fails to initialize
	mockPlugin := &failingMockPlugin{
		mockPlugin: mockPlugin{
			name:        "failing-plugin",
			version:     "1.0.0",
			description: "Failing plugin",
			phase:       PhaseCore,
		},
		initError: errors.New("initialization failed"),
	}

	registry.Register(mockPlugin)

	ctx := mockPluginContext()

	err := loader.LoadPlugins(ctx)
	assert.Error(t, err)
}

func TestPluginLoader_LoadPlugins_ValidationError(t *testing.T) {
	registry := NewRegistry()
	loader := NewLoader(registry)

	// Create a plugin that fails validation
	mockPlugin := &failingMockPlugin{
		mockPlugin: mockPlugin{
			name:        "invalid-plugin",
			version:     "1.0.0",
			description: "Invalid plugin",
			phase:       PhaseCore,
		},
		validateError: errors.New("validation failed"),
	}

	registry.Register(mockPlugin)

	ctx := mockPluginContext()

	err := loader.LoadPlugins(ctx)
	assert.Error(t, err)
}

func TestRegisterPlugin(t *testing.T) {
	// Reset default registry
	defaultRegistry = NewRegistry()

	mockPlugin := &mockPlugin{
		name:    "default-plugin",
		version: "1.0.0",
		phase:   PhaseCore,
	}

	err := RegisterPlugin(mockPlugin)
	require.NoError(t, err)

	// Verify it's in the default registry
	reg := GetDefaultRegistry()
	got, err := reg.Get("default-plugin")
	require.NoError(t, err)
	assert.Equal(t, "default-plugin", got.Name())
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
