// Package dotfiles provides the dotfiles plugin for Kilt.
// It handles symlink-based synchronization of dotfiles from the dotfiles repository to the home directory.
package dotfiles

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/unravelling/kilt/internal/core"
	"github.com/unravelling/kilt/internal/plugin"
)

// DotfilesPlugin handles symlink-based dotfile synchronization
type DotfilesPlugin struct {
	ctx          *plugin.PluginContext
	dotfilesPath string
	linkedFiles  []LinkedFile // for rollback
}

// LinkedFile tracks a symlink that was created
type LinkedFile struct {
	Source string
	Target string
}

// Common dotfile names that should get auto-prefixed with dot
var commonDotfiles = map[string]bool{
	"zshrc": true, "bashrc": true, "profile": true,
	"gitconfig": true, "gitignore": true, "gitattributes": true,
	"vimrc": true, "gvimrc": true, "nvimrc": true,
	"tmux.conf": true, "screenrc": true,
	"inputrc": true, "curlrc": true, "wgetrc": true,
	"hgrc": true, "hgignore": true,
	"dockerignore": true,
}

func init() {
	plugin.RegisterPlugin(&DotfilesPlugin{})
}

// Name returns the plugin name
func (p *DotfilesPlugin) Name() string {
	return "dotfiles"
}

// Version returns the plugin version
func (p *DotfilesPlugin) Version() string {
	return "1.0.0"
}

// Description returns the plugin description
func (p *DotfilesPlugin) Description() string {
	return "Symlink-based dotfile synchronization"
}

// Dependencies returns plugin dependencies
func (p *DotfilesPlugin) Dependencies() []string {
	return []string{"alternates"} // Run after alternates to use resolved paths
}

// Phase returns the execution phase
func (p *DotfilesPlugin) Phase() plugin.ExecutionPhase {
	return plugin.PhaseCore
}

// Initialize initializes the plugin with context
func (p *DotfilesPlugin) Initialize(ctx *plugin.PluginContext) error {
	p.ctx = ctx
	p.linkedFiles = make([]LinkedFile, 0)

	cfg, ok := ctx.Config.(*core.Config)
	if !ok {
		return fmt.Errorf("invalid config type")
	}

	// Get dotfiles path from config
	p.dotfilesPath = cfg.DotfilesPath
	if p.dotfilesPath == "" {
		p.dotfilesPath = core.DefaultDotfilesPath()
	}

	// Expand path
	expandedPath, err := core.ExpandPath(p.dotfilesPath)
	if err != nil {
		return fmt.Errorf("failed to expand dotfiles path: %w", err)
	}
	p.dotfilesPath = expandedPath

	return nil
}

// Validate validates the plugin configuration
func (p *DotfilesPlugin) Validate() error {
	if p.ctx == nil {
		return fmt.Errorf("plugin context not initialized")
	}

	// Check if dotfiles repository exists
	if _, err := os.Stat(p.dotfilesPath); os.IsNotExist(err) {
		return fmt.Errorf("dotfiles repository not found at %s (run 'kilt init' first)", p.dotfilesPath)
	}

	return nil
}

// Execute executes the plugin logic
func (p *DotfilesPlugin) Execute(ctx *plugin.ExecutionContext) error {
	cfg, ok := p.ctx.Config.(*core.Config)
	if !ok {
		return fmt.Errorf("invalid config type")
	}

	// Process each dotfile entry
	for _, entry := range cfg.Dotfiles {
		if err := p.processEntry(entry, ctx); err != nil {
			return fmt.Errorf("failed to process dotfile entry: %w", err)
		}
	}

	return nil
}

// processEntry processes a single dotfile entry
func (p *DotfilesPlugin) processEntry(entry core.DotfileEntry, ctx *plugin.ExecutionContext) error {
	// Handle directory mode (simple form)
	if entry.Directory != "" {
		return p.processDirectory(entry.Directory, ctx)
	}

	// Handle explicit source/target mapping
	if entry.Source != "" && entry.Target != "" {
		return p.processExplicitMapping(entry, ctx)
	}

	return fmt.Errorf("invalid dotfile entry: must have either directory or source/target")
}

// processDirectory processes a directory entry (links all files in directory to home)
func (p *DotfilesPlugin) processDirectory(dirName string, ctx *plugin.ExecutionContext) error {
	sourceDir := filepath.Join(p.dotfilesPath, dirName)

	// Check if directory exists
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		if p.ctx.Logger != nil {
			p.ctx.Logger.Warn("Directory not found, skipping", "directory", sourceDir)
		}
		return nil
	}

	// Walk directory and link all files
	return filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Get relative path from source directory
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Determine target path
		targetPath := p.determineTargetPath(dirName, relPath)

		// Create symlink
		return p.createSymlink(path, targetPath, ctx)
	})
}

// processExplicitMapping processes an explicit source/target mapping
func (p *DotfilesPlugin) processExplicitMapping(entry core.DotfileEntry, ctx *plugin.ExecutionContext) error {
	// Expand source path relative to dotfiles directory
	sourcePath := filepath.Join(p.dotfilesPath, entry.Source)

	// Check if source exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source file not found: %s", sourcePath)
	}

	// Expand target path
	targetPath, err := core.ExpandPath(entry.Target)
	if err != nil {
		return fmt.Errorf("failed to expand target path: %w", err)
	}

	// Handle template rendering if needed
	if entry.Template {
		return p.processTemplateFile(sourcePath, targetPath, entry, ctx)
	}

	// Create symlink
	return p.createSymlink(sourcePath, targetPath, ctx)
}

// processTemplateFile processes a template file (copy rendered content, not symlink)
func (p *DotfilesPlugin) processTemplateFile(sourcePath, targetPath string, entry core.DotfileEntry, ctx *plugin.ExecutionContext) error {
	// Read template
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	// Render template (type assert to concrete type)
	templateEngine, ok := p.ctx.Template.(*core.TemplateEngine)
	if !ok {
		return fmt.Errorf("template engine type assertion failed")
	}
	rendered, err := templateEngine.RenderString(string(content))
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Check if file has changed (after writing, we'll check the actual file)
	// For templates, we always write since content is dynamic

	// Create target directory if needed
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Backup existing file if it exists
	if _, err := os.Stat(targetPath); err == nil {
		if p.ctx.Backup != nil {
			// Type assert to concrete type
			backupManager, ok := p.ctx.Backup.(*core.BackupManager)
			if ok {
				// CreateBackup takes a slice of files
				_, _, err := backupManager.CreateBackup([]string{targetPath}, "kilt: backup before template update")
				if err != nil {
					if p.ctx.Logger != nil {
						p.ctx.Logger.Warn("Failed to create backup", "file", targetPath, "error", err)
					}
				}
			}
		}
	}

	if p.ctx.DryRun {
		if p.ctx.Logger != nil {
			p.ctx.Logger.Info("Would write template file", "target", targetPath)
		}
		ctx.AddChange(plugin.Change{
			Type:        "template_file",
			Files:       []string{sourcePath, targetPath},
			Description: fmt.Sprintf("Would write template file: %s", targetPath),
		})
		return nil
	}

	// Write rendered content
	if err := os.WriteFile(targetPath, []byte(rendered), p.getFileMode(entry.Mode)); err != nil {
		return fmt.Errorf("failed to write template file: %w", err)
	}

	// Update state
	if p.ctx.State != nil {
		// Type assert to concrete type
		stateManager, ok := p.ctx.State.(*core.StateManager)
		if ok {
			if err := stateManager.UpdateFileRecord(sourcePath, targetPath); err != nil {
				if p.ctx.Logger != nil {
					p.ctx.Logger.Warn("Failed to record file checksum", "file", targetPath, "error", err)
				}
			}
		}
	}

	ctx.AddChange(plugin.Change{
		Type:        "template_file",
		Files:       []string{sourcePath, targetPath},
		Description: fmt.Sprintf("Wrote template file: %s", targetPath),
	})

	return nil
}

// createSymlink creates a symlink from target to source
func (p *DotfilesPlugin) createSymlink(sourcePath, targetPath string, ctx *plugin.ExecutionContext) error {
	// Check if target already exists and is correct symlink
	if linkTarget, err := os.Readlink(targetPath); err == nil {
		// Resolve absolute paths for comparison
		absSource, err := filepath.Abs(sourcePath)
		if err != nil {
			return fmt.Errorf("resolving absolute path for source: %w", err)
		}
		absTarget, err := filepath.Abs(linkTarget)
		if err != nil {
			return fmt.Errorf("resolving absolute path for link target: %w", err)
		}
		if absSource == absTarget {
			// Symlink already points to correct location
			return nil
		}
	}

	// Check if target exists (not a symlink)
	if _, err := os.Stat(targetPath); err == nil {
		// File exists, check if it's a symlink
		if _, err := os.Readlink(targetPath); err != nil {
			// Not a symlink, need to backup
			if p.ctx.Backup != nil {
				// Type assert to concrete type
				backupManager, ok := p.ctx.Backup.(*core.BackupManager)
				if ok {
					_, _, err := backupManager.CreateBackup([]string{targetPath}, "kilt: backup before symlink creation")
					if err != nil {
						if p.ctx.Logger != nil {
							p.ctx.Logger.Warn("Failed to create backup", "file", targetPath, "error", err)
						}
					}
				}
			}
			// Remove existing file (always attempt, regardless of backup success/failure)
			if err := os.Remove(targetPath); err != nil {
				return fmt.Errorf("failed to remove existing file: %w", err)
			}
		}
	}

	// Create target directory if needed
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	if p.ctx.DryRun {
		if p.ctx.Logger != nil {
			p.ctx.Logger.Info("Would create symlink", "source", sourcePath, "target", targetPath)
		}
		ctx.AddChange(plugin.Change{
			Type:        "symlink",
			Files:       []string{sourcePath, targetPath},
			Description: fmt.Sprintf("Would create symlink: %s -> %s", targetPath, sourcePath),
		})
		return nil
	}

	// Create symlink (use absolute path for source)
	absSource, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute source path: %w", err)
	}

	if err := os.Symlink(absSource, targetPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	// Track for rollback
	p.linkedFiles = append(p.linkedFiles, LinkedFile{
		Source: absSource,
		Target: targetPath,
	})

	ctx.AddChange(plugin.Change{
		Type:        "symlink",
		Files:       []string{sourcePath, targetPath},
		Description: fmt.Sprintf("Created symlink: %s -> %s", targetPath, sourcePath),
	})

	return nil
}

// determineTargetPath determines the target path for a file in a directory
func (p *DotfilesPlugin) determineTargetPath(dirName, relPath string) string {
	homeDir := p.ctx.HomeDir

	// Split path components
	parts := strings.Split(relPath, string(filepath.Separator))

	// Handle special case: if file is directly in directory root
	if len(parts) == 1 {
		filename := parts[0]
		// Check if it's a common dotfile name
		if commonDotfiles[filename] {
			// Auto-prefix with dot
			return filepath.Join(homeDir, "."+filename)
		}
		// If it already starts with dot, keep it
		if strings.HasPrefix(filename, ".") {
			return filepath.Join(homeDir, filename)
		}
		// Otherwise, add dot prefix
		return filepath.Join(homeDir, "."+filename)
	}

	// Handle nested paths (e.g., zsh/.config/foo -> ~/.config/foo)
	// First part might be a dot-prefixed directory
	result := homeDir
	for i, part := range parts {
		if i == 0 {
			// First part: check if it's a common dotfile name
			if commonDotfiles[part] {
				result = filepath.Join(result, "."+part)
			} else if strings.HasPrefix(part, ".") {
				result = filepath.Join(result, part)
			} else {
				result = filepath.Join(result, "."+part)
			}
		} else {
			result = filepath.Join(result, part)
		}
	}

	return result
}

// getFileMode parses file mode string and returns os.FileMode
func (p *DotfilesPlugin) getFileMode(modeStr string) os.FileMode {
	if modeStr == "" {
		return 0644 // Default
	}

	mode, err := strconv.ParseUint(modeStr, 8, 32)
	if err != nil {
		return 0644 // Default on error
	}

	return os.FileMode(mode)
}

// Rollback rolls back created symlinks
func (p *DotfilesPlugin) Rollback(ctx *plugin.ExecutionContext) error {
	for _, linked := range p.linkedFiles {
		if err := os.Remove(linked.Target); err != nil {
			if p.ctx.Logger != nil {
				p.ctx.Logger.Warn("Failed to remove symlink during rollback", "target", linked.Target, "error", err)
			}
		}
	}
	p.linkedFiles = make([]LinkedFile, 0)
	return nil
}

