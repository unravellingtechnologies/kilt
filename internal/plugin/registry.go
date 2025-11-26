package plugin

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

var (
	// ErrPluginNotFound is returned when a plugin is not found in the registry
	ErrPluginNotFound = errors.New("plugin not found")
	// ErrPluginAlreadyRegistered is returned when trying to register a duplicate plugin
	ErrPluginAlreadyRegistered = errors.New("plugin already registered")
	// ErrCircularDependency is returned when plugins have circular dependencies
	ErrCircularDependency = errors.New("circular dependency detected")
	// ErrMissingDependency is returned when a plugin depends on a non-existent plugin
	ErrMissingDependency = errors.New("missing dependency")
)

// PluginRegistry manages plugin registration and execution order
type PluginRegistry struct {
	plugins map[string]Plugin
	mu      sync.RWMutex
}

// NewRegistry creates a new plugin registry
func NewRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string]Plugin),
	}
}

// Register registers a plugin in the registry
func (r *PluginRegistry) Register(p Plugin) error {
	if p == nil {
		return errors.New("cannot register nil plugin")
	}

	name := p.Name()
	if name == "" {
		return errors.New("plugin name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if plugin is already registered
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("%w: %s", ErrPluginAlreadyRegistered, name)
	}

	// Validate dependencies exist (can't check circular deps here, that's done in GetOrderedPlugins)
	deps := p.Dependencies()
	for _, dep := range deps {
		if dep == name {
			return fmt.Errorf("plugin %s cannot depend on itself", name)
		}
		// Note: We check if dependencies exist when getting ordered plugins
		// to allow registering plugins in any order
	}

	r.plugins[name] = p
	return nil
}

// Unregister removes a plugin from the registry
func (r *PluginRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[name]; !exists {
		return fmt.Errorf("%w: %s", ErrPluginNotFound, name)
	}

	// Check if any other plugin depends on this one
	for _, plugin := range r.plugins {
		for _, dep := range plugin.Dependencies() {
			if dep == name {
				return fmt.Errorf("cannot unregister %s: plugin %s depends on it", name, plugin.Name())
			}
		}
	}

	delete(r.plugins, name)
	return nil
}

// Get retrieves a plugin by name
func (r *PluginRegistry) Get(name string) (Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.plugins[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrPluginNotFound, name)
	}

	return plugin, nil
}

// List returns all registered plugin names
func (r *PluginRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// Count returns the number of registered plugins
func (r *PluginRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.plugins)
}

// GetOrderedPlugins returns plugins sorted by execution phase and dependencies
// Plugins are grouped by phase and topologically sorted within each phase
func (r *PluginRegistry) GetOrderedPlugins() ([]Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// First, validate all dependencies exist
	if err := r.validateDependencies(); err != nil {
		return nil, err
	}

	// Group plugins by phase
	pluginsByPhase := make(map[ExecutionPhase][]Plugin)
	for _, plugin := range r.plugins {
		phase := plugin.Phase()
		pluginsByPhase[phase] = append(pluginsByPhase[phase], plugin)
	}

	// Sort phases in execution order
	phases := []ExecutionPhase{
		PhasePreSync,
		PhaseCore,
		PhaseRunOnce,
		PhaseOnChange,
		PhaseIntegration,
		PhasePostSync,
	}

	var orderedPlugins []Plugin

	// Process each phase in order
	for _, phase := range phases {
		phasePlugins := pluginsByPhase[phase]
		if len(phasePlugins) == 0 {
			continue
		}

		// Topologically sort plugins within this phase
		sorted, err := r.topologicalSort(phasePlugins)
		if err != nil {
			return nil, fmt.Errorf("error sorting plugins in phase %s: %w", phase, err)
		}

		orderedPlugins = append(orderedPlugins, sorted...)
	}

	return orderedPlugins, nil
}

// validateDependencies checks that all plugin dependencies exist
func (r *PluginRegistry) validateDependencies() error {
	for _, plugin := range r.plugins {
		for _, dep := range plugin.Dependencies() {
			if _, exists := r.plugins[dep]; !exists {
				return fmt.Errorf("%w: plugin %s depends on %s which is not registered", ErrMissingDependency, plugin.Name(), dep)
			}
		}
	}
	return nil
}

// topologicalSort sorts plugins based on their dependencies using Kahn's algorithm
func (r *PluginRegistry) topologicalSort(plugins []Plugin) ([]Plugin, error) {
	if len(plugins) == 0 {
		return []Plugin{}, nil
	}

	// Build dependency graph
	inDegree := make(map[string]int)
	graph := make(map[string][]string)
	nameToPlugin := make(map[string]Plugin)

	for _, plugin := range plugins {
		name := plugin.Name()
		nameToPlugin[name] = plugin
		inDegree[name] = 0
		graph[name] = []string{}
	}

	// Calculate in-degrees and build graph
	for _, plugin := range plugins {
		name := plugin.Name()
		deps := plugin.Dependencies()

		for _, dep := range deps {
			// Only consider dependencies that are in this phase's plugins
			if _, exists := nameToPlugin[dep]; exists {
				inDegree[name]++
				graph[dep] = append(graph[dep], name)
			}
		}
	}

	// Find all nodes with no incoming edges
	queue := []string{}
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	if len(queue) == 0 {
		return nil, ErrCircularDependency
	}

	// Process queue
	var sorted []Plugin
	processed := 0

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		sorted = append(sorted, nameToPlugin[current])
		processed++

		// Reduce in-degree of neighbors
		for _, neighbor := range graph[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Check for circular dependencies
	if processed != len(plugins) {
		return nil, ErrCircularDependency
	}

	return sorted, nil
}

