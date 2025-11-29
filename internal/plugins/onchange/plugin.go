// Package onchange provides the onchange plugin for Kilt.
// It executes commands when configuration or files change, supporting different detection modes.
package onchange

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/unravelling/kilt/internal/core"
	"github.com/unravelling/kilt/internal/plugin"
)

// OnChangePlugin executes commands when configuration or files change
type OnChangePlugin struct {
	ctx            *plugin.PluginContext
	defaultTimeout time.Duration
	detectionMode  string // "global", "file", "conditional"
	watchFiles     []string // Files to watch for conditional mode
	executedCmds   []string // Track commands executed in this run for rollback
}

func init() {
	if err := plugin.RegisterPlugin(&OnChangePlugin{}); err != nil {
		panic(fmt.Errorf("failed to register onchange plugin: %w", err))
	}
}

// Name returns the plugin name
func (p *OnChangePlugin) Name() string {
	return "onchange"
}

// Version returns the plugin version
func (p *OnChangePlugin) Version() string {
	return "1.0.0"
}

// Description returns the plugin description
func (p *OnChangePlugin) Description() string {
	return "Execute commands when configuration or files change"
}

// Dependencies returns plugin dependencies
func (p *OnChangePlugin) Dependencies() []string {
	return []string{}
}

// Phase returns the execution phase
func (p *OnChangePlugin) Phase() plugin.ExecutionPhase {
	return plugin.PhaseOnChange
}

// Initialize initializes the plugin with context
func (p *OnChangePlugin) Initialize(ctx *plugin.PluginContext) error {
	p.ctx = ctx
	p.defaultTimeout = 5 * time.Minute // Default 5 minute timeout
	p.detectionMode = "global"         // Default: run if any file changed
	p.watchFiles = make([]string, 0)
	p.executedCmds = make([]string, 0)

	// Get plugin-specific configuration
	config := plugin.GetPluginConfig(ctx.Config, p.Name())
	if config != nil {
		// Parse timeout
		if timeoutStr, ok := config["timeout"].(string); ok {
			duration, err := time.ParseDuration(timeoutStr)
			if err != nil {
				return fmt.Errorf("invalid timeout: %s (must be a valid duration like '5m', '30s'): %w", timeoutStr, err)
			}
			p.defaultTimeout = duration
		}

		// Parse detection mode
		if mode, ok := config["detection_mode"].(string); ok {
			switch mode {
			case "global", "file", "conditional":
				p.detectionMode = mode
			default:
				return fmt.Errorf("invalid detection_mode: %s (must be 'global', 'file', or 'conditional')", mode)
			}
		}

		// Parse watch files for conditional mode
		if watchFiles, ok := config["watch_files"].([]interface{}); ok {
			p.watchFiles = make([]string, 0, len(watchFiles))
			for _, file := range watchFiles {
				if fileStr, ok := file.(string); ok {
					// Expand path
					expanded, err := core.ExpandPath(fileStr)
					if err != nil {
						return fmt.Errorf("failed to expand watch file path %s: %w", fileStr, err)
					}
					p.watchFiles = append(p.watchFiles, expanded)
				}
			}
		}
	}

	return nil
}

// Validate validates the plugin configuration
func (p *OnChangePlugin) Validate() error {
	if p.ctx == nil {
		return fmt.Errorf("plugin context not initialized")
	}

	cfg, ok := p.ctx.Config.(*core.Config)
	if !ok {
		return fmt.Errorf("invalid config type")
	}

	// Validate on_change commands are not empty
	for i, cmd := range cfg.OnChange {
		if cmd == "" {
			return fmt.Errorf("on_change[%d]: command cannot be empty", i)
		}
	}

	// Validate watch files exist if in conditional mode
	if p.detectionMode == "conditional" {
		if len(p.watchFiles) == 0 {
			return fmt.Errorf("watch_files must be specified when detection_mode is 'conditional'")
		}

		for i, watchFile := range p.watchFiles {
			// Check if file exists
			if _, err := os.Stat(watchFile); os.IsNotExist(err) {
				// File doesn't exist yet - that's okay, it might be created
				// But we should at least check the directory exists
				dir := filepath.Dir(watchFile)
				if _, err := os.Stat(dir); os.IsNotExist(err) {
					return fmt.Errorf("watch_files[%d]: directory does not exist: %s", i, dir)
				}
			}
		}
	}

	return nil
}

// Execute executes the plugin logic
func (p *OnChangePlugin) Execute(ctx *plugin.ExecutionContext) error {
	cfg, ok := p.ctx.Config.(*core.Config)
	if !ok {
		return fmt.Errorf("invalid config type")
	}

	// Reset executed commands for this execution
	p.executedCmds = make([]string, 0)

	// Check if changes were detected
	shouldExecute, changedFiles, err := p.shouldExecute(ctx)
	if err != nil {
		return fmt.Errorf("failed to determine if commands should execute: %w", err)
	}

	if !shouldExecute {
		if p.ctx.Logger != nil {
			p.ctx.Logger.Info("No changes detected, skipping on_change commands")
		}
		return nil
	}

	// Log detected changes
	if p.ctx.Logger != nil {
		if len(changedFiles) > 0 {
			p.ctx.Logger.Info("Changes detected, executing on_change commands", "changed_files", changedFiles)
		} else {
			p.ctx.Logger.Info("Changes detected, executing on_change commands")
		}
	}

	// Execute each command in order
	for i, cmd := range cfg.OnChange {
		if err := p.executeCommand(ctx, cmd, i, changedFiles); err != nil {
			return fmt.Errorf("failed to execute command %d (%s): %w", i, cmd, err)
		}
	}

	return nil
}

// shouldExecute determines if commands should be executed based on detection mode
func (p *OnChangePlugin) shouldExecute(ctx *plugin.ExecutionContext) (bool, []string, error) {
	switch p.detectionMode {
	case "global":
		// Run if any file changed (check execution context changes or state)
		return p.checkGlobalChanges(ctx)
	case "file":
		// Run if any tracked file changed
		return p.checkFileChanges(ctx)
	case "conditional":
		// Run only if watched files changed
		return p.checkConditionalChanges(ctx)
	default:
		return false, nil, fmt.Errorf("unknown detection mode: %s", p.detectionMode)
	}
}

// checkGlobalChanges checks if any file changed (global mode)
func (p *OnChangePlugin) checkGlobalChanges(ctx *plugin.ExecutionContext) (bool, []string, error) {
	// Check execution context changes first (most recent)
	changedFiles := make([]string, 0)
	for _, change := range ctx.Changes {
		if len(change.Files) > 0 {
			changedFiles = append(changedFiles, change.Files...)
		}
	}

	if len(changedFiles) > 0 {
		return true, changedFiles, nil
	}

	// Also check if last sync was recent (within this execution)
	// If this is the first sync, we might not have changes in context yet
	lastSync := p.ctx.State.GetLastSync()
	if !lastSync.IsZero() {
		// Check if sync happened recently (within last minute)
		if time.Since(lastSync) < time.Minute {
			// Likely a change occurred, but we'll be conservative
			// In practice, changes should be in the context
			return true, []string{}, nil
		}
	}

	return false, nil, nil
}

// checkFileChanges checks if any tracked files changed (file mode)
func (p *OnChangePlugin) checkFileChanges(ctx *plugin.ExecutionContext) (bool, []string, error) {
	// Get all files that were changed in this execution
	changedFiles := make([]string, 0)
	seenFiles := make(map[string]bool)

	for _, change := range ctx.Changes {
		for _, file := range change.Files {
			if !seenFiles[file] {
				changedFiles = append(changedFiles, file)
				seenFiles[file] = true
			}
		}
	}

	// Also check state for files that changed
	cfg, ok := p.ctx.Config.(*core.Config)
	if ok {
		for _, dotfile := range cfg.Dotfiles {
			if dotfile.Target != "" {
				hasChanged, err := p.ctx.State.HasFileChanged(dotfile.Target)
				if err != nil {
					// Log but don't fail
					if p.ctx.Logger != nil {
						p.ctx.Logger.Warn("Failed to check file change", "file", dotfile.Target, "error", err)
					}
					continue
				}
				if hasChanged && !seenFiles[dotfile.Target] {
					changedFiles = append(changedFiles, dotfile.Target)
					seenFiles[dotfile.Target] = true
				}
			}
		}
	}

	return len(changedFiles) > 0, changedFiles, nil
}

// checkConditionalChanges checks if watched files changed (conditional mode)
func (p *OnChangePlugin) checkConditionalChanges(ctx *plugin.ExecutionContext) (bool, []string, error) {
	changedFiles := make([]string, 0)

	// Check each watched file
	for _, watchFile := range p.watchFiles {
		hasChanged, err := p.ctx.State.HasFileChanged(watchFile)
		if err != nil {
			// If file doesn't exist, check if it was previously tracked (deletion)
			if _, statErr := os.Stat(watchFile); os.IsNotExist(statErr) {
				// Only deletions of previously-tracked files are considered changes
				// New files with no record are not treated as changed
				_, hasRecord := p.ctx.State.GetFileChecksum(watchFile)
				if hasRecord {
					// File was deleted (had a record but no longer exists)
					changedFiles = append(changedFiles, watchFile)
				}
				continue
			}
			// Other errors - log but continue
			if p.ctx.Logger != nil {
				p.ctx.Logger.Warn("Failed to check watched file change", "file", watchFile, "error", err)
			}
			continue
		}

		if hasChanged {
			changedFiles = append(changedFiles, watchFile)
		}
	}

	// Also check execution context changes for watched files
	for _, change := range ctx.Changes {
		for _, file := range change.Files {
			for _, watchFile := range p.watchFiles {
				// Check if file matches watched file (handle relative/absolute paths)
				if core.PathsMatch(file, watchFile) {
					// Check if already in changedFiles
					found := false
					for _, cf := range changedFiles {
						if core.PathsMatch(cf, watchFile) {
							found = true
							break
						}
					}
					if !found {
						changedFiles = append(changedFiles, watchFile)
					}
				}
			}
		}
	}

	return len(changedFiles) > 0, changedFiles, nil
}


// executeCommand executes a single command
func (p *OnChangePlugin) executeCommand(ctx *plugin.ExecutionContext, cmdStr string, index int, changedFiles []string) error {
	// In dry-run mode, just report what would be executed
	if p.ctx.DryRun {
		ctx.AddChange(plugin.Change{
			Type:        "onchange_execute",
			Files:       changedFiles,
			Description: fmt.Sprintf("Would execute command: %s", cmdStr),
		})
		return nil
	}

	// Execute the command
	output, exitCode, err := p.runCommand(cmdStr, p.defaultTimeout)
	if err != nil {
		// Log the error
		if p.ctx.Logger != nil {
			p.ctx.Logger.Error("Command execution failed", "command", cmdStr, "error", err, "exit_code", exitCode)
		}

		// Return error (this will trigger rollback)
		return fmt.Errorf("command execution failed with exit code %d: %w\nOutput:\n%s", exitCode, err, output)
	}

	// Track executed command for rollback
	p.executedCmds = append(p.executedCmds, cmdStr)

	// Record change
	ctx.AddChange(plugin.Change{
		Type:        "onchange_execute",
		Files:       changedFiles,
		Description: fmt.Sprintf("Executed command: %s (exit code: %d)", cmdStr, exitCode),
	})

	if p.ctx.Logger != nil {
		p.ctx.Logger.Info("Command executed successfully", "command", cmdStr, "exit_code", exitCode)
		if output != "" {
			p.ctx.Logger.Debug("Command output", "output", output)
		}
	}

	return nil
}

// runCommand executes a command with timeout protection
func (p *OnChangePlugin) runCommand(cmdStr string, timeout time.Duration) (string, int, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Parse command (simple shell parsing)
	// For now, we'll use sh -c to execute the command
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Dir = p.ctx.WorkDir

	// Capture stdout and stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set environment variables (inherit from parent)
	cmd.Env = os.Environ()

	// Run the command
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			// Context timeout or other error
			if ctx.Err() == context.DeadlineExceeded {
				return "", -1, fmt.Errorf("command execution timed out after %v", timeout)
			}
			return "", -1, fmt.Errorf("failed to execute command: %w", err)
		}
	}

	// Combine stdout and stderr
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += "STDERR:\n" + stderr.String()
	}

	// If exit code is non-zero, return error
	if exitCode != 0 {
		return output, exitCode, fmt.Errorf("command exited with code %d", exitCode)
	}

	return output, exitCode, nil
}

// Rollback rolls back command execution
func (p *OnChangePlugin) Rollback(ctx *plugin.ExecutionContext) error {
	// For on-change commands, rollback means logging what was executed
	// We can't undo what the commands did, but we can log for debugging
	for _, cmd := range p.executedCmds {
		if p.ctx.Logger != nil {
			p.ctx.Logger.Info("Rolling back on-change command", "command", cmd)
		}

		ctx.AddChange(plugin.Change{
			Type:        "onchange_rollback",
			Files:       []string{},
			Description: fmt.Sprintf("Rolled back on-change command: %s", cmd),
		})
	}

	// Clear executed commands
	p.executedCmds = make([]string, 0)

	return nil
}

