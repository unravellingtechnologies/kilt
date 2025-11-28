// Package directories provides the directories plugin for Kilt.
// It ensures specified directories exist with proper permissions.
package directories

import (
	"fmt"
	"os"
	"strconv"

	"github.com/unravelling/kilt/internal/core"
	"github.com/unravelling/kilt/internal/plugin"
)

// DirectoriesPlugin ensures specified directories exist with proper permissions
type DirectoriesPlugin struct {
	ctx           *plugin.PluginContext
	defaultMode   os.FileMode
	createdDirs   []string // For rollback tracking
}

func init() {
	plugin.RegisterPlugin(&DirectoriesPlugin{})
}

// Name returns the plugin name
func (p *DirectoriesPlugin) Name() string {
	return "directories"
}

// Version returns the plugin version
func (p *DirectoriesPlugin) Version() string {
	return "1.0.0"
}

// Description returns the plugin description
func (p *DirectoriesPlugin) Description() string {
	return "Ensure specified directories exist with proper permissions"
}

// Dependencies returns plugin dependencies
func (p *DirectoriesPlugin) Dependencies() []string {
	return []string{}
}

// Phase returns the execution phase
func (p *DirectoriesPlugin) Phase() plugin.ExecutionPhase {
	return plugin.PhaseCore
}

// Initialize initializes the plugin with context
func (p *DirectoriesPlugin) Initialize(ctx *plugin.PluginContext) error {
	p.ctx = ctx
	p.createdDirs = make([]string, 0)
	p.defaultMode = 0755 // Default directory permissions

	// Get plugin-specific configuration
	config := plugin.GetPluginConfig(ctx.Config, p.Name())
	if config != nil {
		if modeStr, ok := config["default_mode"].(string); ok {
			modeVal, err := strconv.ParseUint(modeStr, 8, 32)
			if err != nil {
				return fmt.Errorf("invalid default_mode: %s (must be octal like 0755)", modeStr)
			}
			p.defaultMode = os.FileMode(modeVal)
		}
	}

	return nil
}

// Validate validates the plugin configuration
func (p *DirectoriesPlugin) Validate() error {
	if p.ctx == nil {
		return fmt.Errorf("plugin context not initialized")
	}
	return nil
}

// Execute executes the plugin logic
func (p *DirectoriesPlugin) Execute(ctx *plugin.ExecutionContext) error {
	cfg, ok := p.ctx.Config.(*core.Config)
	if !ok {
		return fmt.Errorf("invalid config type")
	}

	// Reset created directories for this execution
	p.createdDirs = make([]string, 0)

	// Process each directory
	for _, dirPath := range cfg.Directories {
		if err := p.processDirectory(ctx, dirPath); err != nil {
			return fmt.Errorf("failed to process directory %s: %w", dirPath, err)
		}
	}

	return nil
}

// processDirectory processes a single directory
func (p *DirectoriesPlugin) processDirectory(ctx *plugin.ExecutionContext, dirPath string) error {
	// Check if directory already exists
	info, err := os.Stat(dirPath)
	if err == nil {
		// Directory exists
		if !info.IsDir() {
			return fmt.Errorf("path exists but is not a directory: %s", dirPath)
		}

		// Check if permissions need to be updated
		if !p.ctx.DryRun {
			// Get target mode from config or use default
			targetMode := p.getTargetMode(dirPath)
			currentMode := info.Mode().Perm()

			if currentMode != targetMode {
				if err := os.Chmod(dirPath, targetMode); err != nil {
					if p.ctx.Logger != nil {
						p.ctx.Logger.Warn("Failed to update directory permissions", "path", dirPath, "error", err)
					}
					// Don't fail on permission update errors
				} else {
					ctx.AddChange(plugin.Change{
						Type:        "directory_permissions",
						Files:       []string{dirPath},
						Description: fmt.Sprintf("Updated permissions for existing directory: %s", dirPath),
					})
				}
			}
		} else {
			// Dry-run: just report
			ctx.AddChange(plugin.Change{
				Type:        "directory_exists",
				Files:       []string{dirPath},
				Description: fmt.Sprintf("Directory already exists: %s", dirPath),
			})
		}

		return nil
	}

	// Directory doesn't exist
	if os.IsNotExist(err) {
		if p.ctx.DryRun {
			// Dry-run: report what would be created
			ctx.AddChange(plugin.Change{
				Type:        "directory_create",
				Files:       []string{dirPath},
				Description: fmt.Sprintf("Would create directory: %s", dirPath),
			})
			return nil
		}

		// Create directory with parents
		targetMode := p.getTargetMode(dirPath)
		if err := os.MkdirAll(dirPath, targetMode); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Track created directory for rollback
		p.createdDirs = append(p.createdDirs, dirPath)

		// Record change
		ctx.AddChange(plugin.Change{
			Type:        "directory_create",
			Files:       []string{dirPath},
			Description: fmt.Sprintf("Created directory: %s", dirPath),
		})

		if p.ctx.Logger != nil {
			p.ctx.Logger.Info("Created directory", "path", dirPath, "mode", targetMode)
		}

		return nil
	}

	// Other error (permission denied, etc.)
	return fmt.Errorf("failed to stat directory: %w", err)
}

// getTargetMode returns the target mode for a directory
func (p *DirectoriesPlugin) getTargetMode(dirPath string) os.FileMode {
	// Check if there's a per-directory mode in plugin config
	config := plugin.GetPluginConfig(p.ctx.Config, p.Name())
	if config != nil {
		// Support per-directory modes via a map
		if modes, ok := config["modes"].(map[string]interface{}); ok {
			if modeStr, ok := modes[dirPath].(string); ok {
				modeVal, err := strconv.ParseUint(modeStr, 8, 32)
				if err == nil {
					return os.FileMode(modeVal)
				}
			}
		}
	}

	// Use default mode
	return p.defaultMode
}

// Rollback rolls back directory creation
func (p *DirectoriesPlugin) Rollback(ctx *plugin.ExecutionContext) error {
	// Remove created directories in reverse order
	for i := len(p.createdDirs) - 1; i >= 0; i-- {
		dir := p.createdDirs[i]
		if err := os.Remove(dir); err != nil {
			if p.ctx.Logger != nil {
				p.ctx.Logger.Error("Failed to remove directory during rollback", "path", dir, "error", err)
			}
			// Continue with other directories
		}
	}

	// Clear created directories list
	p.createdDirs = make([]string, 0)

	return nil
}

