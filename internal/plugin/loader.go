// Package plugin provides the plugin system architecture for Kilt.
// It defines the Plugin interface, execution phases, plugin contexts,
// and provides utilities for plugin registration, discovery, and execution.
package plugin

import (
	"fmt"
)

// PluginLoader handles plugin discovery, registration, and initialization
type PluginLoader struct {
	registry *PluginRegistry
}

// NewLoader creates a new plugin loader
func NewLoader(registry *PluginRegistry) *PluginLoader {
	return &PluginLoader{
		registry: registry,
	}
}

// LoadPlugins initializes all registered plugins with the given context
// Plugins should already be registered in the registry (via init() functions in plugin packages)
func (l *PluginLoader) LoadPlugins(ctx *PluginContext) error {
	plugins, err := l.registry.GetOrderedPlugins()
	if err != nil {
		return fmt.Errorf("failed to get ordered plugins: %w", err)
	}

	// Initialize all plugins in dependency order
	for _, plugin := range plugins {
		if err := plugin.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize plugin %s: %w", plugin.Name(), err)
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

// RegisterPlugin is a convenience function that registers a plugin with the global registry
// This should be called from plugin init() functions
// Note: For now, this is a simple wrapper. In a full implementation, you might want
// a global registry or use dependency injection
var defaultRegistry *PluginRegistry

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
func GetDefaultRegistry() *PluginRegistry {
	if defaultRegistry == nil {
		defaultRegistry = NewRegistry()
	}
	return defaultRegistry
}

