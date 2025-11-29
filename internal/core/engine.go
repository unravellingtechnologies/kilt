// Package core provides the core engine and configuration management
// for the Kilt dotfiles manager. It includes configuration parsing,
// state management, backup operations, template rendering, and the
// main orchestration engine.
package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/unravelling/kilt/internal/plugin"
)

// Engine orchestrates plugin execution and coordinates all core components
type Engine struct {
	config        *Config
	state         *StateManager
	backup        *BackupManager
	template      *TemplateEngine
	registry      *plugin.PluginRegistry
	dryRun        bool
	verbose       bool
	workDir       string
	homeDir       string
	executionCtx  context.Context
	cancelFunc    context.CancelFunc
	rollbackStack []plugin.Plugin
}

// ExecutionResult contains the results of an engine execution
type ExecutionResult struct {
	Success    bool
	PluginsRun int
	Changes    []plugin.Change
	Errors     []error
	Duration   time.Duration
	RolledBack bool
}

// NewEngine creates a new engine instance
func NewEngine(cfg *Config, registry *plugin.PluginRegistry) (*Engine, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if registry == nil {
		return nil, fmt.Errorf("plugin registry cannot be nil")
	}

	// Get directories
	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Initialize state manager
	stateDir, err := GetStateDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get state directory: %w", err)
	}

	state, err := NewStateManager(stateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create state manager: %w", err)
	}

	// Initialize backup manager
	backupDir, err := GetBackupDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get backup directory: %w", err)
	}

	backup, err := NewBackupManager(backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup manager: %w", err)
	}

	// Initialize template engine
	template := NewTemplateEngine()

	// Load custom data if specified
	if cfg.DataFile != "" {
		if err := template.LoadCustomData(cfg.DataFile); err != nil {
			return nil, fmt.Errorf("failed to load custom data: %w", err)
		}
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())

	return &Engine{
		config:        cfg,
		state:         state,
		backup:        backup,
		template:      template,
		registry:      registry,
		workDir:       workDir,
		homeDir:       homeDir,
		executionCtx:  ctx,
		cancelFunc:    cancel,
		rollbackStack: make([]plugin.Plugin, 0),
	}, nil
}

// SetDryRun sets the dry-run mode
func (e *Engine) SetDryRun(dryRun bool) {
	e.dryRun = dryRun
}

// SetVerbose sets the verbose mode
func (e *Engine) SetVerbose(verbose bool) {
	e.verbose = verbose
}

// Cancel cancels the current execution
func (e *Engine) Cancel() {
	if e.cancelFunc != nil {
		e.cancelFunc()
	}
}

// PreflightChecks performs pre-flight validation before execution
func (e *Engine) PreflightChecks() error {
	if e.verbose {
		fmt.Println("Running pre-flight checks...")
	}

	// Check Git is available
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not available in PATH: %w\nHint: Install Git to use Kilt", err)
	}

	// Validate configuration
	if err := ValidateConfig(e.config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Check file system permissions
	if err := e.checkFileSystemPermissions(); err != nil {
		return fmt.Errorf("file system permission check failed: %w", err)
	}

	// Validate plugin dependencies
	orderedPlugins, err := e.registry.GetOrderedPlugins()
	if err != nil {
		return fmt.Errorf("plugin dependency validation failed: %w", err)
	}

	if len(orderedPlugins) == 0 {
		if e.verbose {
			fmt.Println("Warning: No plugins registered")
		}
	}

	if e.verbose {
		fmt.Printf("Pre-flight checks passed (%d plugins ready)\n", len(orderedPlugins))
	}

	return nil
}

// checkFileSystemPermissions checks that we can write to necessary directories
func (e *Engine) checkFileSystemPermissions() error {
	// Check state directory
	stateDir, err := GetStateDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("cannot create state directory: %w", err)
	}

	// Check backup directory
	backupDir, err := GetBackupDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("cannot create backup directory: %w", err)
	}

	return nil
}

// Execute runs the full execution pipeline
func (e *Engine) Execute() (*ExecutionResult, error) {
	startTime := time.Now()
	result := &ExecutionResult{
		Changes: make([]plugin.Change, 0),
		Errors:  make([]error, 0),
	}

	// Acquire state lock
	if err := e.state.AcquireLock(); err != nil {
		return nil, fmt.Errorf("failed to acquire state lock: %w\nHint: Another Kilt process may be running", err)
	}
	defer func() {
		if err := e.state.ReleaseLock(); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to release lock: %w", err))
		}
	}()

	// Pre-flight checks
	if err := e.PreflightChecks(); err != nil {
		return nil, err
	}

	// Get ordered plugins
	orderedPlugins, err := e.registry.GetOrderedPlugins()
	if err != nil {
		return nil, fmt.Errorf("failed to get ordered plugins: %w", err)
	}

	// Create plugin context
	pluginCtx := &plugin.PluginContext{
		Config:   e.config,
		State:    e.state,
		Backup:   e.backup,
		Template: e.template,
		DryRun:   e.dryRun,
		WorkDir:  e.workDir,
		HomeDir:  e.homeDir,
	}

	// Initialize plugins
	if err := e.initializePlugins(orderedPlugins, pluginCtx); err != nil {
		return nil, fmt.Errorf("plugin initialization failed: %w", err)
	}

	// Execute plugins
	executionCtx := &plugin.ExecutionContext{
		PluginContext: pluginCtx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
		StartTime:     startTime,
	}

	pluginsRun := 0
	for _, p := range orderedPlugins {
		// Check for cancellation
		select {
		case <-e.executionCtx.Done():
			return nil, fmt.Errorf("execution cancelled")
		default:
		}

		if e.verbose {
			fmt.Printf("Executing plugin: %s (%s)\n", p.Name(), p.Description())
		}

		if e.dryRun {
			if e.verbose {
				fmt.Printf("  [DRY-RUN] Would execute %s\n", p.Name())
			}
			pluginsRun++
			continue
		}

		// Validate plugin
		if err := p.Validate(); err != nil {
			err := fmt.Errorf("plugin %s validation failed: %w", p.Name(), err)
			result.Errors = append(result.Errors, err)
			if e.verbose {
				fmt.Printf("  Error: %v\n", err)
			}
			continue
		}

		// Execute plugin
		if err := p.Execute(executionCtx); err != nil {
			err := fmt.Errorf("plugin %s execution failed: %w", p.Name(), err)
			result.Errors = append(result.Errors, err)

			// Rollback on failure
			if err := e.rollback(); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("rollback failed: %w", err))
			}
			result.RolledBack = true

			return result, fmt.Errorf("execution failed: %w", err)
		}

		// Add successfully executed plugin to rollback stack
		e.rollbackStack = append(e.rollbackStack, p)

		pluginsRun++
		result.Changes = append(result.Changes, executionCtx.Changes...)
	}

	// Update state
	if !e.dryRun {
		if err := e.state.UpdateLastSync(); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to update last sync: %w", err))
		}
	}

	result.Success = len(result.Errors) == 0
	result.PluginsRun = pluginsRun
	result.Duration = time.Since(startTime)

	if e.verbose {
		fmt.Printf("Execution completed: %d plugins run in %v\n", pluginsRun, result.Duration)
	}

	return result, nil
}

// initializePlugins initializes all plugins with the plugin context
func (e *Engine) initializePlugins(plugins []plugin.Plugin, ctx *plugin.PluginContext) error {
	for _, p := range plugins {
		if err := p.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize plugin %s: %w", p.Name(), err)
		}
	}
	return nil
}

// rollback rolls back all executed plugins in reverse order
func (e *Engine) rollback() error {
	if e.verbose {
		fmt.Println("Rolling back changes...")
	}

	// Create execution context for rollback
	pluginCtx := &plugin.PluginContext{
		Config:   e.config,
		State:    e.state,
		Backup:   e.backup,
		Template: e.template,
		DryRun:   false, // Rollback is always real
		WorkDir:  e.workDir,
		HomeDir:  e.homeDir,
	}

	executionCtx := &plugin.ExecutionContext{
		PluginContext: pluginCtx,
		Changes:       make([]plugin.Change, 0),
		Errors:        make([]error, 0),
		StartTime:     time.Now(),
	}

	// Rollback in reverse order
	for i := len(e.rollbackStack) - 1; i >= 0; i-- {
		p := e.rollbackStack[i]
		if e.verbose {
			fmt.Printf("Rolling back plugin: %s\n", p.Name())
		}

		if err := p.Rollback(executionCtx); err != nil {
			return fmt.Errorf("failed to rollback plugin %s: %w", p.Name(), err)
		}
	}

	return nil
}

// GetExecutionPlan builds and returns an execution plan without executing
func (e *Engine) GetExecutionPlan() ([]plugin.Plugin, error) {
	return e.registry.GetOrderedPlugins()
}

// Cleanup performs cleanup operations
func (e *Engine) Cleanup() error {
	// Release lock (already done in defer, but this is for explicit cleanup)
	if err := e.state.ReleaseLock(); err != nil {
		return fmt.Errorf("failed to release lock during cleanup: %w", err)
	}

	// Cancel context
	if e.cancelFunc != nil {
		e.cancelFunc()
	}

	return nil
}
