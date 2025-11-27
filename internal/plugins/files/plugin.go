package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/unravelling/kilt/internal/core"
	"github.com/unravelling/kilt/internal/plugin"
)

// FilesPlugin handles file placement with source → target mapping
type FilesPlugin struct {
	ctx            *plugin.PluginContext
	createSymlinks bool
	preservePerms  bool
	processedFiles []ProcessedFile
}

// ProcessedFile tracks a file operation for rollback
type ProcessedFile struct {
	Source      string
	Target      string
	WasBackedUp bool
	BackupID    string
	OriginalMode os.FileMode
	WasSymlink  bool
}

func init() {
	plugin.RegisterPlugin(&FilesPlugin{})
}

// Name returns the plugin name
func (p *FilesPlugin) Name() string {
	return "files"
}

// Version returns the plugin version
func (p *FilesPlugin) Version() string {
	return "1.0.0"
}

// Description returns the plugin description
func (p *FilesPlugin) Description() string {
	return "Core plugin for file placement with source → target mapping"
}

// Dependencies returns plugin dependencies
func (p *FilesPlugin) Dependencies() []string {
	return []string{}
}

// Phase returns the execution phase
func (p *FilesPlugin) Phase() plugin.ExecutionPhase {
	return plugin.PhaseCore
}

// Initialize initializes the plugin with context
func (p *FilesPlugin) Initialize(ctx *plugin.PluginContext) error {
	p.ctx = ctx
	p.processedFiles = make([]ProcessedFile, 0)

	// Get plugin-specific configuration
	config := plugin.GetPluginConfig(ctx.Config, p.Name())
	if config != nil {
		if val, ok := config["create_symlinks"].(bool); ok {
			p.createSymlinks = val
		}
		if val, ok := config["preserve_permissions"].(bool); ok {
			p.preservePerms = val
		}
	}

	// Default values
	if config == nil || config["create_symlinks"] == nil {
		p.createSymlinks = false
	}
	if config == nil || config["preserve_permissions"] == nil {
		p.preservePerms = true
	}

	return nil
}

// Validate validates the plugin configuration
func (p *FilesPlugin) Validate() error {
	if p.ctx == nil {
		return fmt.Errorf("plugin context not initialized")
	}

	// Get config to validate file mappings
	cfg, ok := p.ctx.Config.(*core.Config)
	if !ok {
		return fmt.Errorf("invalid config type")
	}

	// Validate that all source files exist (relative to workDir)
	for i, file := range cfg.Files {
		sourcePath := filepath.Join(p.ctx.WorkDir, file.Source)
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			return fmt.Errorf("files[%d]: source file does not exist: %s", i, sourcePath)
		}
	}

	return nil
}

// Execute executes the plugin logic
func (p *FilesPlugin) Execute(ctx *plugin.ExecutionContext) error {
	cfg, ok := p.ctx.Config.(*core.Config)
	if !ok {
		return fmt.Errorf("invalid config type")
	}

	// Reset processed files for this execution
	p.processedFiles = make([]ProcessedFile, 0)

	// Process each file mapping
	for _, fileMapping := range cfg.Files {
		if err := p.processFileMapping(ctx, fileMapping); err != nil {
			return fmt.Errorf("failed to process file mapping %s -> %s: %w", fileMapping.Source, fileMapping.Target, err)
		}
	}

	return nil
}

// processFileMapping processes a single file mapping
func (p *FilesPlugin) processFileMapping(ctx *plugin.ExecutionContext, fileMapping core.FileMapping) error {
	// Build source path (relative to workDir)
	sourcePath := filepath.Join(p.ctx.WorkDir, fileMapping.Source)
	
	// Target path is already expanded by config system
	targetPath := fileMapping.Target

	// Check if file has changed (for optimization)
	// Skip this check for new files (when target doesn't exist)
	targetExists := false
	if _, err := os.Stat(targetPath); err == nil {
		targetExists = true
	}

	if !p.ctx.DryRun && targetExists {
		changed, err := p.ctx.State.HasFileChanged(targetPath)
		if err != nil {
			// Log error but continue (treat as changed to be safe)
			if p.ctx.Logger != nil {
				p.ctx.Logger.Warn("Failed to check if file changed", "target", targetPath, "error", err)
			}
		} else if !changed {
			// File hasn't changed, skip it
			if p.ctx.Logger != nil {
				p.ctx.Logger.Debug("File unchanged, skipping", "target", targetPath)
			}
			return nil
		}
	}

	// Get original mode if target exists
	var originalMode os.FileMode
	if targetExists {
		if info, err := os.Stat(targetPath); err == nil {
			originalMode = info.Mode()
		}
	}

	// Backup existing file if needed
	var backupID string
	if targetExists && !p.ctx.DryRun {
		// Use type assertion to access BackupManager methods
		if backupMgr, ok := p.ctx.Backup.(*core.BackupManager); ok {
			id, _, err := backupMgr.CreateBackup([]string{targetPath}, fmt.Sprintf("Files plugin: %s", targetPath))
			if err != nil {
				return fmt.Errorf("failed to backup file: %w", err)
			}
			backupID = id
		}
	}

	// Track processed file for rollback
	processedFile := ProcessedFile{
		Source:      sourcePath,
		Target:      targetPath,
		WasBackedUp:  backupID != "",
		BackupID:    backupID,
		OriginalMode: originalMode,
		WasSymlink:  false,
	}

	// Handle dry-run mode
	if p.ctx.DryRun {
		return p.handleDryRun(ctx, sourcePath, targetPath, fileMapping)
	}

	// Create target directory if needed
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Process file based on configuration
	var err error
	if p.createSymlinks {
		err = p.createSymlink(sourcePath, targetPath)
		processedFile.WasSymlink = true
	} else {
		err = p.copyFile(ctx, sourcePath, targetPath, fileMapping)
	}

	if err != nil {
		return err
	}

	// Set file permissions
	if err := p.setFilePermissions(targetPath, fileMapping, originalMode); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	// Update state
	if err := p.ctx.State.UpdateFileRecord(sourcePath, targetPath); err != nil {
		if p.ctx.Logger != nil {
			p.ctx.Logger.Warn("Failed to update file record", "target", targetPath, "error", err)
		}
	}

	// Record change
	ctx.AddChange(plugin.Change{
		Type:        "file_write",
		Files:       []string{targetPath},
		Description: fmt.Sprintf("Placed file %s -> %s", fileMapping.Source, targetPath),
	})

	p.processedFiles = append(p.processedFiles, processedFile)

	return nil
}

// handleDryRun handles dry-run mode by generating diffs
func (p *FilesPlugin) handleDryRun(ctx *plugin.ExecutionContext, sourcePath, targetPath string, fileMapping core.FileMapping) error {
	// Read source file
	sourceContent, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Apply template if needed
	if fileMapping.Template {
		templateEngine, ok := p.ctx.Template.(*core.TemplateEngine)
		if !ok {
			return fmt.Errorf("invalid template engine type")
		}
		rendered, err := templateEngine.RenderString(string(sourceContent))
		if err != nil {
			return fmt.Errorf("failed to render template: %w", err)
		}
		sourceContent = []byte(rendered)
	}

	// Check if target exists
	targetExists := false
	var targetContent []byte
	if info, err := os.Stat(targetPath); err == nil && !info.IsDir() {
		targetExists = true
		targetContent, err = os.ReadFile(targetPath)
		if err != nil {
			// If we can't read it, that's okay for dry-run
			targetContent = nil
		}
	}

	// Generate diff description
	var description string
	if !targetExists {
		description = fmt.Sprintf("Would create new file: %s", targetPath)
	} else if string(sourceContent) != string(targetContent) {
		description = fmt.Sprintf("Would update file: %s (content differs)", targetPath)
	} else {
		description = fmt.Sprintf("Would update file: %s (permissions may differ)", targetPath)
	}

	ctx.AddChange(plugin.Change{
		Type:        "file_write",
		Files:       []string{targetPath},
		Description: description,
	})

	return nil
}

// createSymlink creates a symlink from target to source
func (p *FilesPlugin) createSymlink(sourcePath, targetPath string) error {
	// Remove existing file/symlink if it exists
	if _, err := os.Lstat(targetPath); err == nil {
		if err := os.Remove(targetPath); err != nil {
			return fmt.Errorf("failed to remove existing file: %w", err)
		}
	}

	// Create symlink
	// Use absolute path for source to avoid broken symlinks
	absSource, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute source path: %w", err)
	}

	if err := os.Symlink(absSource, targetPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// copyFile copies a file from source to target, optionally rendering templates
func (p *FilesPlugin) copyFile(ctx *plugin.ExecutionContext, sourcePath, targetPath string, fileMapping core.FileMapping) error {
	// Read source file
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Render template if needed
	if fileMapping.Template {
		templateEngine, ok := p.ctx.Template.(*core.TemplateEngine)
		if !ok {
			return fmt.Errorf("invalid template engine type")
		}
		rendered, err := templateEngine.RenderString(string(content))
		if err != nil {
			return fmt.Errorf("failed to render template: %w", err)
		}
		content = []byte(rendered)
	}

	// Write target file
	if err := os.WriteFile(targetPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write target file: %w", err)
	}

	return nil
}

// setFilePermissions sets file permissions based on configuration
func (p *FilesPlugin) setFilePermissions(targetPath string, fileMapping core.FileMapping, originalMode os.FileMode) error {
	var mode os.FileMode

	if fileMapping.Mode != "" {
		// Parse mode from config
		modeVal, err := strconv.ParseUint(fileMapping.Mode, 8, 32)
		if err != nil {
			return fmt.Errorf("invalid file mode: %s", fileMapping.Mode)
		}
		mode = os.FileMode(modeVal)
	} else if p.preservePerms {
		// Get source file mode
		sourcePath := filepath.Join(p.ctx.WorkDir, fileMapping.Source)
		if info, err := os.Stat(sourcePath); err == nil {
			mode = info.Mode()
		} else {
			// Default mode if we can't read source
			mode = 0644
		}
	} else {
		// Default mode
		mode = 0644
	}

	// Set permissions
	if err := os.Chmod(targetPath, mode); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}

// Rollback rolls back file operations
func (p *FilesPlugin) Rollback(ctx *plugin.ExecutionContext) error {
	// Rollback in reverse order
	for i := len(p.processedFiles) - 1; i >= 0; i-- {
		processed := p.processedFiles[i]

		if processed.WasSymlink {
			// Remove symlink
			if err := os.Remove(processed.Target); err != nil && !os.IsNotExist(err) {
				if p.ctx.Logger != nil {
					p.ctx.Logger.Error("Failed to remove symlink during rollback", "target", processed.Target, "error", err)
				}
			}
		} else {
			// Remove copied file
			if err := os.Remove(processed.Target); err != nil && !os.IsNotExist(err) {
				if p.ctx.Logger != nil {
					p.ctx.Logger.Error("Failed to remove file during rollback", "target", processed.Target, "error", err)
				}
			}
		}

		// Restore from backup if available
		if processed.WasBackedUp && processed.BackupID != "" {
			if backupMgr, ok := p.ctx.Backup.(*core.BackupManager); ok {
				if err := backupMgr.Restore(processed.BackupID); err != nil {
					if p.ctx.Logger != nil {
						p.ctx.Logger.Error("Failed to restore from backup during rollback", "backup_id", processed.BackupID, "error", err)
					}
				}
			}
		}

		// Remove file record from state
		if err := p.ctx.State.RemoveFileRecord(processed.Target); err != nil {
			if p.ctx.Logger != nil {
				p.ctx.Logger.Warn("Failed to remove file record during rollback", "target", processed.Target, "error", err)
			}
		}
	}

	return nil
}

