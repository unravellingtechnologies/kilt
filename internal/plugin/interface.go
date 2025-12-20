// Package plugin provides the plugin system architecture for Kilt.
// It defines the Plugin interface, execution phases, plugin contexts,
// and provides utilities for plugin registration, discovery, and execution.
package plugin

import "time"

// ExecutionPhase represents the execution phase of a plugin
type ExecutionPhase int

const (
	// PhasePreSync executes before syncing files (e.g., Git operations)
	PhasePreSync ExecutionPhase = iota
	// PhaseCore executes core operations (files, directories)
	PhaseCore
	// PhaseRunOnce executes bootstrap scripts that run once
	PhaseRunOnce
	// PhaseOnChange executes change-triggered tasks
	PhaseOnChange
	// PhaseIntegration executes integration tasks (e.g., package managers)
	PhaseIntegration
	// PhasePostSync executes after syncing (validation, cleanup)
	PhasePostSync
)

// String returns the string representation of the execution phase
func (p ExecutionPhase) String() string {
	switch p {
	case PhasePreSync:
		return "PreSync"
	case PhaseCore:
		return "Core"
	case PhaseRunOnce:
		return "RunOnce"
	case PhaseOnChange:
		return "OnChange"
	case PhaseIntegration:
		return "Integration"
	case PhasePostSync:
		return "PostSync"
	default:
		return "Unknown"
	}
}

// Change represents a change made during plugin execution
type Change struct {
	Type        string    // Type of change (e.g., "git_pull", "file_write")
	Files       []string  // Files affected by this change
	Description string    // Human-readable description
	Timestamp   time.Time // When the change occurred
}

// Plugin is the interface all plugins must implement
type Plugin interface {
	// Metadata
	Name() string        // Unique plugin name
	Version() string     // Plugin version
	Description() string // Human-readable description

	// Lifecycle hooks
	Initialise(ctx *Context) error        // Initialise plugin with context
	Validate() error                      // Validate plugin configuration
	Execute(ctx *ExecutionContext) error  // Execute plugin logic
	Rollback(ctx *ExecutionContext) error // Rollback plugin changes on error

	// Dependencies and execution
	Dependencies() []string // Names of plugins this plugin depends on
	Phase() ExecutionPhase  // Execution phase for this plugin
}
