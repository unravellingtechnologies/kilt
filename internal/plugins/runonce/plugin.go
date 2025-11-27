package runonce

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/unravelling/kilt/internal/core"
	"github.com/unravelling/kilt/internal/plugin"
)

// RunOncePlugin executes bootstrap scripts exactly once per machine
type RunOncePlugin struct {
	ctx          *plugin.PluginContext
	defaultTimeout time.Duration
	forceRun       bool
	executedTasks  []string // Track tasks executed in this run for rollback
}

func init() {
	plugin.RegisterPlugin(&RunOncePlugin{})
}

// Name returns the plugin name
func (p *RunOncePlugin) Name() string {
	return "runonce"
}

// Version returns the plugin version
func (p *RunOncePlugin) Version() string {
	return "1.0.0"
}

// Description returns the plugin description
func (p *RunOncePlugin) Description() string {
	return "Execute bootstrap scripts exactly once per machine"
}

// Dependencies returns plugin dependencies
func (p *RunOncePlugin) Dependencies() []string {
	return []string{}
}

// Phase returns the execution phase
func (p *RunOncePlugin) Phase() plugin.ExecutionPhase {
	return plugin.PhaseRunOnce
}

// Initialize initializes the plugin with context
func (p *RunOncePlugin) Initialize(ctx *plugin.PluginContext) error {
	p.ctx = ctx
	p.defaultTimeout = 5 * time.Minute // Default 5 minute timeout
	p.forceRun = false
	p.executedTasks = make([]string, 0)

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

		// Parse force_run flag
		if forceRun, ok := config["force_run"].(bool); ok {
			p.forceRun = forceRun
		}
	}

	return nil
}

// Validate validates the plugin configuration
func (p *RunOncePlugin) Validate() error {
	if p.ctx == nil {
		return fmt.Errorf("plugin context not initialized")
	}

	cfg, ok := p.ctx.Config.(*core.Config)
	if !ok {
		return fmt.Errorf("invalid config type")
	}

	// Validate that all run_once scripts exist
	for i, script := range cfg.RunOnce {
		// Script path should be relative to workDir or absolute
		scriptPath := script
		if !filepath.IsAbs(script) {
			scriptPath = filepath.Join(p.ctx.WorkDir, script)
		}

		// Check if script exists
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			return fmt.Errorf("run_once[%d]: script not found: %s", i, scriptPath)
		}

		// Check if script is executable (or at least readable)
		info, err := os.Stat(scriptPath)
		if err != nil {
			return fmt.Errorf("run_once[%d]: cannot stat script: %s: %w", i, scriptPath, err)
		}

		// Check if it's a regular file
		if !info.Mode().IsRegular() {
			return fmt.Errorf("run_once[%d]: script is not a regular file: %s", i, scriptPath)
		}
	}

	return nil
}

// Execute executes the plugin logic
func (p *RunOncePlugin) Execute(ctx *plugin.ExecutionContext) error {
	cfg, ok := p.ctx.Config.(*core.Config)
	if !ok {
		return fmt.Errorf("invalid config type")
	}

	// Reset executed tasks for this execution
	p.executedTasks = make([]string, 0)

	// Process each script in order
	for i, script := range cfg.RunOnce {
		if err := p.executeScript(ctx, script, i); err != nil {
			return fmt.Errorf("failed to execute script %d (%s): %w", i, script, err)
		}
	}

	return nil
}

// executeScript executes a single script
func (p *RunOncePlugin) executeScript(ctx *plugin.ExecutionContext, scriptPath string, index int) error {
	// Resolve script path
	fullScriptPath := scriptPath
	if !filepath.IsAbs(scriptPath) {
		fullScriptPath = filepath.Join(p.ctx.WorkDir, scriptPath)
	}

	// Generate task ID from script path (use hash for consistency)
	taskID := p.generateTaskID(fullScriptPath)

	// Check if task has already been completed (unless force_run is enabled)
	if !p.forceRun {
		if p.ctx.State.IsTaskCompleted(taskID) {
			if p.ctx.Logger != nil {
				p.ctx.Logger.Info("Skipping already executed script", "script", scriptPath, "task_id", taskID)
			}

			// Record that we skipped this task
			ctx.AddChange(plugin.Change{
				Type:        "runonce_skipped",
				Files:       []string{scriptPath},
				Description: fmt.Sprintf("Skipped already executed script: %s", scriptPath),
			})
			return nil
		}
	}

	// In dry-run mode, just report what would be executed
	if p.ctx.DryRun {
		ctx.AddChange(plugin.Change{
			Type:        "runonce_execute",
			Files:       []string{scriptPath},
			Description: fmt.Sprintf("Would execute script: %s", scriptPath),
		})
		return nil
	}

	// Execute the script
	output, exitCode, err := p.runScript(fullScriptPath, p.defaultTimeout)
	if err != nil {
		// Log the error
		if p.ctx.Logger != nil {
			p.ctx.Logger.Error("Script execution failed", "script", scriptPath, "error", err, "exit_code", exitCode)
		}

		// Record failure in state (even failures are tracked to prevent retries)
		record := core.RunOnceRecord{
			TaskID:     taskID,
			ExecutedAt: time.Now(),
			ExitCode:   exitCode,
			Output:     output,
		}
		if err := p.ctx.State.MarkTaskCompleted(taskID, record); err != nil {
			if p.ctx.Logger != nil {
				p.ctx.Logger.Warn("Failed to record script execution", "task_id", taskID, "error", err)
			}
		}

		// Return error (this will trigger rollback)
		return fmt.Errorf("script execution failed with exit code %d: %w\nOutput:\n%s", exitCode, err, output)
	}

	// Record successful execution
	record := core.RunOnceRecord{
		TaskID:     taskID,
		ExecutedAt: time.Now(),
		ExitCode:   exitCode,
		Output:     output,
	}
	if err := p.ctx.State.MarkTaskCompleted(taskID, record); err != nil {
		if p.ctx.Logger != nil {
			p.ctx.Logger.Warn("Failed to record script execution", "task_id", taskID, "error", err)
		}
		// Don't fail the execution if state recording fails
	}

	// Track executed task for rollback
	p.executedTasks = append(p.executedTasks, taskID)

	// Record change
	ctx.AddChange(plugin.Change{
		Type:        "runonce_execute",
		Files:       []string{scriptPath},
		Description: fmt.Sprintf("Executed script: %s (exit code: %d)", scriptPath, exitCode),
	})

	if p.ctx.Logger != nil {
		p.ctx.Logger.Info("Script executed successfully", "script", scriptPath, "exit_code", exitCode)
		if output != "" {
			p.ctx.Logger.Debug("Script output", "output", output)
		}
	}

	return nil
}

// runScript executes a script with timeout protection
func (p *RunOncePlugin) runScript(scriptPath string, timeout time.Duration) (string, int, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Determine shell based on script extension or shebang
	shell, args := p.determineShell(scriptPath)

	// Create command
	cmd := exec.CommandContext(ctx, shell, args...)
	cmd.Dir = filepath.Dir(scriptPath)

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
				return "", -1, fmt.Errorf("script execution timed out after %v", timeout)
			}
			return "", -1, fmt.Errorf("failed to execute script: %w", err)
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
		return output, exitCode, fmt.Errorf("script exited with code %d", exitCode)
	}

	return output, exitCode, nil
}

// determineShell determines the shell and arguments to use for executing the script
func (p *RunOncePlugin) determineShell(scriptPath string) (string, []string) {
	// Try to read shebang
	file, err := os.Open(scriptPath)
	if err == nil {
		defer file.Close()
		shebang := make([]byte, 128) // Read first 128 bytes for shebang
		n, _ := file.Read(shebang)
		shebangStr := string(shebang[:n])

		// Check for shebang
		if strings.HasPrefix(shebangStr, "#!") {
			// Parse shebang line
			lines := strings.SplitN(shebangStr, "\n", 2)
			shebangLine := strings.TrimSpace(lines[0][2:]) // Remove "#!"
			parts := strings.Fields(shebangLine)
			if len(parts) > 0 {
				shell := parts[0]
				args := parts[1:]
				args = append(args, scriptPath)
				return shell, args
			}
		}
	}

	// Default: use sh for Unix-like systems
	// Check file extension
	ext := filepath.Ext(scriptPath)
	switch ext {
	case ".sh":
		return "sh", []string{scriptPath}
	case ".bash":
		return "bash", []string{scriptPath}
	case ".zsh":
		return "zsh", []string{scriptPath}
	case ".fish":
		return "fish", []string{scriptPath}
	default:
		// Default to sh
		return "sh", []string{scriptPath}
	}
}

// generateTaskID generates a unique task ID from script path
func (p *RunOncePlugin) generateTaskID(scriptPath string) string {
	// Use absolute path for consistency
	absPath, err := filepath.Abs(scriptPath)
	if err != nil {
		// Fallback to original path if abs fails
		absPath = scriptPath
	}

	// Create hash of the path for a stable ID
	hash := sha256.Sum256([]byte(absPath))
	return fmt.Sprintf("runonce_%x", hash[:8]) // Use first 8 bytes for shorter ID
}

// Rollback rolls back script execution (removes state records)
func (p *RunOncePlugin) Rollback(ctx *plugin.ExecutionContext) error {
	// For run-once scripts, rollback means removing the execution records
	// This allows the scripts to be re-executed on next run
	// Note: We can't undo what the scripts did, but we can allow re-execution

	for _, taskID := range p.executedTasks {
		// Get the state manager
		stateManager, ok := p.ctx.State.(*core.StateManager)
		if !ok {
			// If we can't cast, we can't remove records
			if p.ctx.Logger != nil {
				p.ctx.Logger.Warn("Cannot rollback run-once task: invalid state manager type", "task_id", taskID)
			}
			continue
		}

		// Remove the record to allow re-execution
		if err := stateManager.DeleteRunOnceRecord(taskID); err != nil {
			if p.ctx.Logger != nil {
				p.ctx.Logger.Warn("Failed to delete run-once record during rollback", "task_id", taskID, "error", err)
			}
		} else {
			if p.ctx.Logger != nil {
				p.ctx.Logger.Info("Rolled back run-once task", "task_id", taskID)
			}
		}

		ctx.AddChange(plugin.Change{
			Type:        "runonce_rollback",
			Files:       []string{taskID},
			Description: fmt.Sprintf("Rolled back run-once task: %s (can be re-executed)", taskID),
		})
	}

	// Clear executed tasks
	p.executedTasks = make([]string, 0)

	return nil
}

