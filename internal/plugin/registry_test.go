package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPlugin is a test implementation of the Plugin interface
type mockPlugin struct {
	name         string
	version      string
	description  string
	dependencies []string
	phase        ExecutionPhase
	initialised  bool
	validated    bool
	executed     bool
	rollbacked   bool
}

func (m *mockPlugin) Name() string                         { return m.name }
func (m *mockPlugin) Version() string                      { return m.version }
func (m *mockPlugin) Description() string                  { return m.description }
func (m *mockPlugin) Dependencies() []string               { return m.dependencies }
func (m *mockPlugin) Phase() ExecutionPhase                { return m.phase }
func (m *mockPlugin) Initialise(ctx *Context) error        { m.initialised = true; return nil }
func (m *mockPlugin) Validate() error                      { m.validated = true; return nil }
func (m *mockPlugin) Execute(ctx *ExecutionContext) error  { m.executed = true; return nil }
func (m *mockPlugin) Rollback(ctx *ExecutionContext) error { m.rollbacked = true; return nil }

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	tests := []struct {
		name    string
		plugin  Plugin
		wantErr bool
		errType error
	}{
		{
			name: "register valid plugin",
			plugin: &mockPlugin{
				name:    "test-plugin",
				version: "1.0.0",
				phase:   PhaseCore,
			},
			wantErr: false,
		},
		{
			name:    "register nil plugin",
			plugin:  nil,
			wantErr: true,
		},
		{
			name: "register plugin with empty name",
			plugin: &mockPlugin{
				name:    "",
				version: "1.0.0",
				phase:   PhaseCore,
			},
			wantErr: true,
		},
		{
			name: "register duplicate plugin",
			plugin: &mockPlugin{
				name:    "test-plugin",
				version: "1.0.0",
				phase:   PhaseCore,
			},
			wantErr: true,
			errType: ErrPluginAlreadyRegistered,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.Register(tt.plugin)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()
	plugin := &mockPlugin{name: "test-plugin", version: "1.0.0", phase: PhaseCore}
	require.NoError(t, registry.Register(plugin))

	tests := []struct {
		name    string
		lookup  string
		wantErr bool
	}{
		{
			name:    "get existing plugin",
			lookup:  "test-plugin",
			wantErr: false,
		},
		{
			name:    "get non-existent plugin",
			lookup:  "non-existent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := registry.Get(tt.lookup)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.lookup, got.Name())
			}
		})
	}
}

func TestRegistry_Unregister(t *testing.T) {
	registry := NewRegistry()
	plugin1 := &mockPlugin{name: "plugin1", version: "1.0.0", phase: PhaseCore}
	plugin2 := &mockPlugin{name: "plugin2", version: "1.0.0", phase: PhaseCore, dependencies: []string{"plugin1"}}

	require.NoError(t, registry.Register(plugin1))
	require.NoError(t, registry.Register(plugin2))

	// Try to unregister plugin1 which plugin2 depends on
	err := registry.Unregister("plugin1")
	assert.Error(t, err, "Unregister() should fail when plugin has dependencies")

	// Unregister plugin2 first (no dependencies)
	err = registry.Unregister("plugin2")
	require.NoError(t, err)

	// Now unregister plugin1 should work
	err = registry.Unregister("plugin1")
	require.NoError(t, err)
}

func TestRegistry_GetOrderedPlugins(t *testing.T) {
	registry := NewRegistry()

	// Create plugins with dependencies
	pluginA := &mockPlugin{name: "plugin-a", version: "1.0.0", phase: PhaseCore}
	pluginB := &mockPlugin{name: "plugin-b", version: "1.0.0", phase: PhaseCore, dependencies: []string{"plugin-a"}}
	pluginC := &mockPlugin{name: "plugin-c", version: "1.0.0", phase: PhaseCore, dependencies: []string{"plugin-b"}}

	// Register in reverse order to test ordering
	require.NoError(t, registry.Register(pluginC))
	require.NoError(t, registry.Register(pluginB))
	require.NoError(t, registry.Register(pluginA))

	ordered, err := registry.GetOrderedPlugins()
	require.NoError(t, err)

	// Verify order: A -> B -> C
	require.Len(t, ordered, 3)

	assert.Equal(t, "plugin-a", ordered[0].Name())
	assert.Equal(t, "plugin-b", ordered[1].Name())
	assert.Equal(t, "plugin-c", ordered[2].Name())
}

func TestRegistry_GetOrderedPlugins_CircularDependency(t *testing.T) {
	registry := NewRegistry()

	// Create circular dependency: A -> B -> C -> A
	pluginA := &mockPlugin{name: "plugin-a", version: "1.0.0", phase: PhaseCore, dependencies: []string{"plugin-c"}}
	pluginB := &mockPlugin{name: "plugin-b", version: "1.0.0", phase: PhaseCore, dependencies: []string{"plugin-a"}}
	pluginC := &mockPlugin{name: "plugin-c", version: "1.0.0", phase: PhaseCore, dependencies: []string{"plugin-b"}}

	require.NoError(t, registry.Register(pluginA))
	require.NoError(t, registry.Register(pluginB))
	require.NoError(t, registry.Register(pluginC))

	_, err := registry.GetOrderedPlugins()
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrCircularDependency)
}

func TestRegistry_GetOrderedPlugins_MissingDependency(t *testing.T) {
	registry := NewRegistry()

	plugin := &mockPlugin{name: "plugin", version: "1.0.0", phase: PhaseCore, dependencies: []string{"non-existent"}}
	require.NoError(t, registry.Register(plugin))

	_, err := registry.GetOrderedPlugins()
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrMissingDependency)
}

func TestRegistry_GetOrderedPlugins_PhaseOrdering(t *testing.T) {
	registry := NewRegistry()

	// Create plugins in different phases
	plugin1 := &mockPlugin{name: "plugin1", version: "1.0.0", phase: PhasePostSync}
	plugin2 := &mockPlugin{name: "plugin2", version: "1.0.0", phase: PhasePreSync}
	plugin3 := &mockPlugin{name: "plugin3", version: "1.0.0", phase: PhaseCore}

	require.NoError(t, registry.Register(plugin1))
	require.NoError(t, registry.Register(plugin2))
	require.NoError(t, registry.Register(plugin3))

	ordered, err := registry.GetOrderedPlugins()
	require.NoError(t, err)

	// Verify phase order: PreSync -> Core -> PostSync
	require.Len(t, ordered, 3)

	assert.Equal(t, PhasePreSync, ordered[0].Phase())
	assert.Equal(t, PhaseCore, ordered[1].Phase())
	assert.Equal(t, PhasePostSync, ordered[2].Phase())
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	require.NoError(t, registry.Register(&mockPlugin{name: "plugin-c", version: "1.0.0", phase: PhaseCore}))
	require.NoError(t, registry.Register(&mockPlugin{name: "plugin-a", version: "1.0.0", phase: PhaseCore}))
	require.NoError(t, registry.Register(&mockPlugin{name: "plugin-b", version: "1.0.0", phase: PhaseCore}))

	list := registry.List()
	require.Len(t, list, 3)

	// Should be sorted
	assert.Equal(t, "plugin-a", list[0])
	assert.Equal(t, "plugin-b", list[1])
	assert.Equal(t, "plugin-c", list[2])
}

func TestRegistry_Count(t *testing.T) {
	registry := NewRegistry()

	assert.Zero(t, registry.Count())

	require.NoError(t, registry.Register(&mockPlugin{name: "plugin1", version: "1.0.0", phase: PhaseCore}))
	assert.Equal(t, 1, registry.Count())

	require.NoError(t, registry.Register(&mockPlugin{name: "plugin2", version: "1.0.0", phase: PhaseCore}))
	assert.Equal(t, 2, registry.Count())
}
