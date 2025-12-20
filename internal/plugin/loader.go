// Package plugin provides the plugin system architecture for Kilt.
// It defines the Plugin interface, execution phases, plugin contexts,
// and provides utilities for plugin registration, discovery, and execution.
package plugin

import (
	"fmt"
)

// Loader handles plugin discovery, registration, and initialization
type Loader struct {
	registry *Registry
}

// NewLoader creates a new plugin loader
func NewLoader(registry *Registry) *Loader {
	return &Loader{
		registry: registry,
	}
}

// LoadPlugins initialises all registered plugins with the given context
// Plugins should already be registered in the registry (via init() functions in plugin packages)
func (l *Loader) LoadPlugins(ctx *Context) error {
	plugins, err := l.registry.GetOrderedPlugins()
	if err != nil {
		return fmt.Errorf("failed to get ordered plugins: %w", err)
	}

	// Initialise all plugins in dependency order
	for _, plugin := range plugins {
		if err := plugin.Initialise(ctx); err != nil {
			return fmt.Errorf("failed to initialise plugin %s: %w", plugin.Name(), err)
		}

		if err := plugin.Validate(); err != nil {
			return fmt.Errorf("plugin %s validation failed: %w", plugin.Name(), err)
		}
	}

	return nil
}

// GetPluginConfig extracts plugin-specific configuration from the main config
// This is a helper method that plugins can use to get their configuration
func GetPluginConfig(config Config, pluginName string) map[string]interface{} {
	if config == nil {
		return make(map[string]interface{})
	}
	return config.GetPluginConfig(pluginName)
}

// defaultRegistry is the global plugin registry used for automatic plugin registration.
//
// ARCHITECTURAL DECISION: We intentionally use a global registry here rather than
// dependency injection for the following reasons:
//
//  1. Plugin Auto-Registration: Plugins register themselves via init() functions when
//     their packages are imported. This is idiomatic Go (similar to database/sql drivers)
//     and requires zero configuration from users.
//
//  2. Compile-Time Static: Kilt's plugin system is compile-time static - no dynamic
//     plugin loading. The set of plugins is fixed at build time, making a global
//     registry safe and predictable.
//
//  3. Simplicity: Dependency injection would require explicit wiring in main() for
//     every plugin, adding boilerplate with no real benefit for our use case.
//
//  4. Testing: Tests can create fresh registries via NewRegistry() when isolation
//     is needed. Integration tests use the global registry to test real behaviour.
//
// Trade-off: Global state makes unit testing slightly harder, but the pattern is
// well-established in Go and provides excellent ergonomics for plugin authors.
var defaultRegistry *Registry

// init initializes the default plugin registry
func init() {
	defaultRegistry = NewRegistry()
}

// RegisterPlugin registers a plugin with the default registry
// Plugins should call this in their init() functions
func RegisterPlugin(p Plugin) error {
	if defaultRegistry == nil {
		defaultRegistry = NewRegistry()
	}
	return defaultRegistry.Register(p)
}

// GetDefaultRegistry returns the default plugin registry
// This allows plugins to register themselves during init()
func GetDefaultRegistry() *Registry {
	if defaultRegistry == nil {
		defaultRegistry = NewRegistry()
	}
	return defaultRegistry
}
