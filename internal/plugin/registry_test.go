package plugin

import (
	"errors"
	"testing"
)

// mockPlugin is a test implementation of the Plugin interface
type mockPlugin struct {
	name         string
	version      string
	description  string
	dependencies []string
	phase        ExecutionPhase
	initialized  bool
	validated    bool
	executed     bool
	rollbacked   bool
}

func (m *mockPlugin) Name() string                          { return m.name }
func (m *mockPlugin) Version() string                       { return m.version }
func (m *mockPlugin) Description() string                   { return m.description }
func (m *mockPlugin) Dependencies() []string                { return m.dependencies }
func (m *mockPlugin) Phase() ExecutionPhase                 { return m.phase }
func (m *mockPlugin) Initialize(ctx *PluginContext) error   { m.initialized = true; return nil }
func (m *mockPlugin) Validate() error                       { m.validated = true; return nil }
func (m *mockPlugin) Execute(ctx *ExecutionContext) error   { m.executed = true; return nil }
func (m *mockPlugin) Rollback(ctx *ExecutionContext) error  { m.rollbacked = true; return nil }

func TestPluginRegistry_Register(t *testing.T) {
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
			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.errType != nil && err != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("Register() error = %v, wantErr type %v", err, tt.errType)
				}
			}
		})
	}
}

func TestPluginRegistry_Get(t *testing.T) {
	registry := NewRegistry()
	plugin := &mockPlugin{name: "test-plugin", version: "1.0.0", phase: PhaseCore}
	registry.Register(plugin)

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
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Name() != tt.lookup {
				t.Errorf("Get() = %v, want %v", got.Name(), tt.lookup)
			}
		})
	}
}

func TestPluginRegistry_Unregister(t *testing.T) {
	registry := NewRegistry()
	plugin1 := &mockPlugin{name: "plugin1", version: "1.0.0", phase: PhaseCore}
	plugin2 := &mockPlugin{name: "plugin2", version: "1.0.0", phase: PhaseCore, dependencies: []string{"plugin1"}}

	registry.Register(plugin1)
	registry.Register(plugin2)

	// Try to unregister plugin1 which plugin2 depends on
	err := registry.Unregister("plugin1")
	if err == nil {
		t.Error("Unregister() should fail when plugin has dependencies")
	}

	// Unregister plugin2 first (no dependencies)
	err = registry.Unregister("plugin2")
	if err != nil {
		t.Errorf("Unregister() error = %v, want nil", err)
	}

	// Now unregister plugin1 should work
	err = registry.Unregister("plugin1")
	if err != nil {
		t.Errorf("Unregister() error = %v, want nil", err)
	}
}

func TestPluginRegistry_GetOrderedPlugins(t *testing.T) {
	registry := NewRegistry()

	// Create plugins with dependencies
	pluginA := &mockPlugin{name: "plugin-a", version: "1.0.0", phase: PhaseCore}
	pluginB := &mockPlugin{name: "plugin-b", version: "1.0.0", phase: PhaseCore, dependencies: []string{"plugin-a"}}
	pluginC := &mockPlugin{name: "plugin-c", version: "1.0.0", phase: PhaseCore, dependencies: []string{"plugin-b"}}

	// Register in reverse order to test ordering
	registry.Register(pluginC)
	registry.Register(pluginB)
	registry.Register(pluginA)

	ordered, err := registry.GetOrderedPlugins()
	if err != nil {
		t.Fatalf("GetOrderedPlugins() error = %v", err)
	}

	// Verify order: A -> B -> C
	if len(ordered) != 3 {
		t.Fatalf("GetOrderedPlugins() returned %d plugins, want 3", len(ordered))
	}

	if ordered[0].Name() != "plugin-a" {
		t.Errorf("GetOrderedPlugins() first plugin = %v, want plugin-a", ordered[0].Name())
	}
	if ordered[1].Name() != "plugin-b" {
		t.Errorf("GetOrderedPlugins() second plugin = %v, want plugin-b", ordered[1].Name())
	}
	if ordered[2].Name() != "plugin-c" {
		t.Errorf("GetOrderedPlugins() third plugin = %v, want plugin-c", ordered[2].Name())
	}
}

func TestPluginRegistry_GetOrderedPlugins_CircularDependency(t *testing.T) {
	registry := NewRegistry()

	// Create circular dependency: A -> B -> C -> A
	pluginA := &mockPlugin{name: "plugin-a", version: "1.0.0", phase: PhaseCore, dependencies: []string{"plugin-c"}}
	pluginB := &mockPlugin{name: "plugin-b", version: "1.0.0", phase: PhaseCore, dependencies: []string{"plugin-a"}}
	pluginC := &mockPlugin{name: "plugin-c", version: "1.0.0", phase: PhaseCore, dependencies: []string{"plugin-b"}}

	registry.Register(pluginA)
	registry.Register(pluginB)
	registry.Register(pluginC)

	_, err := registry.GetOrderedPlugins()
	if err == nil {
		t.Error("GetOrderedPlugins() should fail with circular dependency")
	}
	if !errors.Is(err, ErrCircularDependency) {
		t.Errorf("GetOrderedPlugins() error = %v, want %v", err, ErrCircularDependency)
	}
}

func TestPluginRegistry_GetOrderedPlugins_MissingDependency(t *testing.T) {
	registry := NewRegistry()

	plugin := &mockPlugin{name: "plugin", version: "1.0.0", phase: PhaseCore, dependencies: []string{"non-existent"}}
	registry.Register(plugin)

	_, err := registry.GetOrderedPlugins()
	if err == nil {
		t.Error("GetOrderedPlugins() should fail with missing dependency")
	}
	if !errors.Is(err, ErrMissingDependency) {
		t.Errorf("GetOrderedPlugins() error = %v, want %v", err, ErrMissingDependency)
	}
}

func TestPluginRegistry_GetOrderedPlugins_PhaseOrdering(t *testing.T) {
	registry := NewRegistry()

	// Create plugins in different phases
	plugin1 := &mockPlugin{name: "plugin1", version: "1.0.0", phase: PhasePostSync}
	plugin2 := &mockPlugin{name: "plugin2", version: "1.0.0", phase: PhasePreSync}
	plugin3 := &mockPlugin{name: "plugin3", version: "1.0.0", phase: PhaseCore}

	registry.Register(plugin1)
	registry.Register(plugin2)
	registry.Register(plugin3)

	ordered, err := registry.GetOrderedPlugins()
	if err != nil {
		t.Fatalf("GetOrderedPlugins() error = %v", err)
	}

	// Verify phase order: PreSync -> Core -> PostSync
	if len(ordered) != 3 {
		t.Fatalf("GetOrderedPlugins() returned %d plugins, want 3", len(ordered))
	}

	if ordered[0].Phase() != PhasePreSync {
		t.Errorf("GetOrderedPlugins() first phase = %v, want PhasePreSync", ordered[0].Phase())
	}
	if ordered[1].Phase() != PhaseCore {
		t.Errorf("GetOrderedPlugins() second phase = %v, want PhaseCore", ordered[1].Phase())
	}
	if ordered[2].Phase() != PhasePostSync {
		t.Errorf("GetOrderedPlugins() third phase = %v, want PhasePostSync", ordered[2].Phase())
	}
}

func TestPluginRegistry_List(t *testing.T) {
	registry := NewRegistry()

	registry.Register(&mockPlugin{name: "plugin-c", version: "1.0.0", phase: PhaseCore})
	registry.Register(&mockPlugin{name: "plugin-a", version: "1.0.0", phase: PhaseCore})
	registry.Register(&mockPlugin{name: "plugin-b", version: "1.0.0", phase: PhaseCore})

	list := registry.List()
	if len(list) != 3 {
		t.Fatalf("List() returned %d plugins, want 3", len(list))
	}

	// Should be sorted
	if list[0] != "plugin-a" || list[1] != "plugin-b" || list[2] != "plugin-c" {
		t.Errorf("List() = %v, want [plugin-a, plugin-b, plugin-c]", list)
	}
}

func TestPluginRegistry_Count(t *testing.T) {
	registry := NewRegistry()

	if registry.Count() != 0 {
		t.Errorf("Count() = %d, want 0", registry.Count())
	}

	registry.Register(&mockPlugin{name: "plugin1", version: "1.0.0", phase: PhaseCore})
	if registry.Count() != 1 {
		t.Errorf("Count() = %d, want 1", registry.Count())
	}

	registry.Register(&mockPlugin{name: "plugin2", version: "1.0.0", phase: PhaseCore})
	if registry.Count() != 2 {
		t.Errorf("Count() = %d, want 2", registry.Count())
	}
}

