// Package plugin provides the plugin system architecture for Kilt.
// It defines the Plugin interface, execution phases, plugin contexts,
// and provides utilities for plugin registration, discovery, and execution.
package plugin

import (
	"time"
)

// Logger is an interface for logging operations
// This will be implemented by pkg/logger package
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// Config is an interface for configuration access
// This will be implemented by internal/core/config package
type Config interface {
	// GetPluginConfig returns plugin-specific configuration
	GetPluginConfig(pluginName string) map[string]interface{}
}

// StateManager is an interface for state management
// This will be implemented by internal/core/state package
type StateManager interface {
	// Run-once task management
	IsTaskCompleted(taskID string) bool
	MarkTaskCompleted(taskID string, record interface{}) error // Use interface{} to avoid circular dependency
	GetRunOnceRecord(taskID string) (interface{}, bool)       // Returns RunOnceRecord-like struct

	// File state management
	GetFileChecksum(targetPath string) (string, bool)
	HasFileChanged(targetPath string) (bool, error)
	UpdateFileRecord(sourcePath, targetPath string) error
	RemoveFileRecord(targetPath string) error

	// Plugin execution tracking
	RecordPluginExecution(pluginName, version string, success bool, duration time.Duration) error
	GetPluginRecord(pluginName string) (interface{}, bool) // Returns PluginRecord-like struct

	// Sync tracking
	UpdateLastSync() error
	GetLastSync() time.Time

	// Locking
	AcquireLock() error
	ReleaseLock() error

	// State management
	ClearState() error
	GetStateDir() string
}

// BackupManager is an interface for backup operations
// This will be implemented by internal/core/backup package
type BackupManager interface {
	// Backup methods will be defined when backup package is implemented
}

// TemplateEngine is an interface for template rendering
// This will be implemented by internal/core/template package
type TemplateEngine interface {
	// Template methods will be defined when template package is implemented
}

// PluginContext provides shared state to plugins during initialization
type PluginContext struct {
	Config       Config
	State        StateManager
	Backup       BackupManager
	Template     TemplateEngine
	Logger       Logger
	DryRun       bool
	WorkDir      string
	HomeDir      string
}

// ExecutionContext provides runtime context during plugin execution
type ExecutionContext struct {
	*PluginContext
	Changes   []Change
	Errors    []error
	StartTime time.Time
}

// AddChange adds a change to the execution context
func (ctx *ExecutionContext) AddChange(change Change) {
	change.Timestamp = time.Now()
	ctx.Changes = append(ctx.Changes, change)
}

// AddError adds an error to the execution context
func (ctx *ExecutionContext) AddError(err error) {
	if err != nil {
		ctx.Errors = append(ctx.Errors, err)
	}
}

// HasErrors returns true if there are any errors in the context
func (ctx *ExecutionContext) HasErrors() bool {
	return len(ctx.Errors) > 0
}

