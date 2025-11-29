// Package brew provides the Homebrew plugin for Kilt.
// It handles Homebrew integration for package management, including bundle installation and updates.
package brew

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/unravelling/kilt/internal/core"
	"github.com/unravelling/kilt/internal/plugin"
)

// BrewPlugin handles Homebrew integration for package management
type BrewPlugin struct {
	ctx            *plugin.PluginContext
	brewPath       string
	brewfile       string
	autoUpdate     bool
	cleanupAfter   bool
	autoInstall    bool
	bundles        []string
	executedBundles []string // Track bundles executed for rollback
	installedBrew  bool     // Track if we installed brew in this run
}

func init() {
	if err := plugin.RegisterPlugin(&BrewPlugin{}); err != nil {
		panic(fmt.Errorf("failed to register brew plugin: %w", err))
	}
}

// Name returns the plugin name
func (p *BrewPlugin) Name() string {
	return "brew"
}

// Version returns the plugin version
func (p *BrewPlugin) Version() string {
	return "1.0.0"
}

// Description returns the plugin description
func (p *BrewPlugin) Description() string {
	return "Homebrew integration for package management"
}

// Dependencies returns plugin dependencies
func (p *BrewPlugin) Dependencies() []string {
	return []string{}
}

// Phase returns the execution phase
func (p *BrewPlugin) Phase() plugin.ExecutionPhase {
	return plugin.PhaseIntegration
}

// Initialize initializes the plugin with context
func (p *BrewPlugin) Initialize(ctx *plugin.PluginContext) error {
	p.ctx = ctx
	p.brewfile = "Brewfile"
	p.autoUpdate = false
	p.cleanupAfter = false
	p.autoInstall = false
	p.bundles = make([]string, 0)
	p.executedBundles = make([]string, 0)
	p.installedBrew = false

	// Get plugin-specific configuration first (to check auto_install)
	config := plugin.GetPluginConfig(ctx.Config, p.Name())
	if config != nil {
		// Parse auto_install flag
		if autoInstall, ok := config["auto_install"].(bool); ok {
			p.autoInstall = autoInstall
		}

		// Parse brewfile path
		if brewfile, ok := config["brewfile"].(string); ok {
			p.brewfile = brewfile
		}

		// Parse auto_update flag
		if autoUpdate, ok := config["auto_update"].(bool); ok {
			p.autoUpdate = autoUpdate
		}

		// Parse cleanup_after flag
		if cleanupAfter, ok := config["cleanup_after"].(bool); ok {
			p.cleanupAfter = cleanupAfter
		}

		// Parse bundles list
		if bundles, ok := config["bundles"].([]interface{}); ok {
			p.bundles = make([]string, 0, len(bundles))
			for _, bundle := range bundles {
				if bundleStr, ok := bundle.(string); ok {
					p.bundles = append(p.bundles, bundleStr)
				}
			}
		}
	}

	// Find Homebrew installation
	brewPath, err := p.findBrew()
	if err != nil {
		// Homebrew not found
		if p.autoInstall {
			// Will attempt to install during Execute
			if ctx.Logger != nil {
				ctx.Logger.Info("Homebrew not found, will attempt installation if auto_install is enabled")
			}
		} else {
			// Homebrew not found and auto_install disabled - plugin will skip execution
			if ctx.Logger != nil {
				ctx.Logger.Warn("Homebrew not found, brew plugin will be skipped (set auto_install: true to install automatically)", "error", err)
			}
		}
		p.brewPath = ""
	} else {
		p.brewPath = brewPath
	}

	return nil
}

// Validate validates the plugin configuration
func (p *BrewPlugin) Validate() error {
	if p.ctx == nil {
		return fmt.Errorf("plugin context not initialized")
	}

	// If Homebrew is not installed, that's okay - plugin will skip
	if p.brewPath == "" {
		return nil
	}

	// Validate brewfile exists if specified
	if p.brewfile != "" {
		brewfilePath := p.brewfile
		if !filepath.IsAbs(brewfilePath) {
			brewfilePath = filepath.Join(p.ctx.WorkDir, brewfilePath)
		}

		// Check if brewfile exists
		if _, err := os.Stat(brewfilePath); os.IsNotExist(err) {
			// That's okay - brew bundle will handle missing file
			if p.ctx.Logger != nil {
				p.ctx.Logger.Debug("Brewfile not found, will be created on first bundle", "path", brewfilePath)
			}
		}
	}

	return nil
}

// Execute executes the plugin logic
func (p *BrewPlugin) Execute(ctx *plugin.ExecutionContext) error {
	// Reset executed bundles for this execution
	p.executedBundles = make([]string, 0)
	p.installedBrew = false

	// Try to install Homebrew if not found and auto_install is enabled
	if p.brewPath == "" {
		if p.autoInstall {
			if err := p.installHomebrew(ctx); err != nil {
				return fmt.Errorf("failed to install Homebrew: %w", err)
			}
			// After installation, find brew again
			brewPath, err := p.findBrew()
			if err != nil {
				return fmt.Errorf("Homebrew installation completed but brew command not found: %w", err)
			}
			p.brewPath = brewPath
			p.installedBrew = true
		} else {
			if p.ctx.Logger != nil {
				p.ctx.Logger.Info("Skipping brew plugin: Homebrew not installed (set auto_install: true to install automatically)")
			}
			return nil
		}
	}

	// Check if Brewfile has changed (for on-change execution)
	shouldRun, err := p.shouldRunBundle(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if bundle should run: %w", err)
	}

	if !shouldRun {
		if p.ctx.Logger != nil {
			p.ctx.Logger.Info("Brewfile unchanged, skipping brew bundle")
		}
		return nil
	}

	// Run brew bundle
	if len(p.bundles) > 0 {
		// Run multiple bundles (skip change detection for bundles)
		for _, bundle := range p.bundles {
			if err := p.runBundle(ctx, bundle); err != nil {
				return fmt.Errorf("failed to run bundle %s: %w", bundle, err)
			}
		}
	} else {
		// Run single brewfile (only if changed)
		if shouldRun, err := p.shouldRunBundle(ctx); err == nil && shouldRun {
			if err := p.runBundle(ctx, p.brewfile); err != nil {
				return fmt.Errorf("failed to run brew bundle: %w", err)
			}
		}
	}

	// Auto-update if enabled
	if p.autoUpdate {
		if err := p.runBrewUpdate(ctx); err != nil {
			if p.ctx.Logger != nil {
				p.ctx.Logger.Warn("Brew update failed", "error", err)
			}
			// Don't fail the execution if update fails
		}
	}

	// Cleanup if enabled
	if p.cleanupAfter {
		if err := p.runBrewCleanup(ctx); err != nil {
			if p.ctx.Logger != nil {
				p.ctx.Logger.Warn("Brew cleanup failed", "error", err)
			}
			// Don't fail the execution if cleanup fails
		}
	}

	return nil
}

// shouldRunBundle checks if brew bundle should run based on Brewfile changes
func (p *BrewPlugin) shouldRunBundle(ctx *plugin.ExecutionContext) (bool, error) {
	// If bundles are specified, always run (they're separate files)
	if len(p.bundles) > 0 {
		return true, nil
	}

	// If no brewfile specified, always run
	if p.brewfile == "" {
		return true, nil
	}

	// Resolve brewfile path
	brewfilePath := p.brewfile
	if !filepath.IsAbs(brewfilePath) {
		brewfilePath = filepath.Join(p.ctx.WorkDir, brewfilePath)
	}

	// Check if brewfile has changed
	hasChanged, err := p.ctx.State.HasFileChanged(brewfilePath)
	if err != nil {
		// If file doesn't exist, we should run to create it
		if _, statErr := os.Stat(brewfilePath); os.IsNotExist(statErr) {
			return true, nil
		}
		return false, err
	}

	// Also check execution context changes
	for _, change := range ctx.Changes {
		for _, file := range change.Files {
			if core.PathsMatch(file, brewfilePath) {
				return true, nil
			}
		}
	}

	return hasChanged, nil
}


// runBundle runs brew bundle for a specific file
func (p *BrewPlugin) runBundle(ctx *plugin.ExecutionContext, bundleFile string) error {
	// Resolve bundle file path
	bundlePath := bundleFile
	if !filepath.IsAbs(bundlePath) {
		bundlePath = filepath.Join(p.ctx.WorkDir, bundlePath)
	}

	// In dry-run mode, just report
	if p.ctx.DryRun {
		ctx.AddChange(plugin.Change{
			Type:        "brew_bundle",
			Files:       []string{bundlePath},
			Description: fmt.Sprintf("Would run: brew bundle --file=%s", bundlePath),
		})
		return nil
	}

	// Set timeout
	cmdCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Run brew bundle
	cmd := exec.CommandContext(cmdCtx, p.brewPath, "bundle", "--file", bundlePath)
	cmd.Dir = p.ctx.WorkDir
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("brew bundle timed out after 15 minutes")
		}
		return fmt.Errorf("brew bundle failed: %s: %w", string(output), err)
	}

	// Track executed bundle
	p.executedBundles = append(p.executedBundles, bundlePath)

	// Record change
	ctx.AddChange(plugin.Change{
		Type:        "brew_bundle",
		Files:       []string{bundlePath},
		Description: fmt.Sprintf("Ran brew bundle: %s", bundlePath),
	})

	if p.ctx.Logger != nil {
		p.ctx.Logger.Info("Brew bundle executed successfully", "file", bundlePath)
		if len(output) > 0 {
			p.ctx.Logger.Debug("Brew bundle output", "output", string(output))
		}
	}

	return nil
}

// runBrewUpdate runs brew update
func (p *BrewPlugin) runBrewUpdate(ctx *plugin.ExecutionContext) error {
	if p.ctx.DryRun {
		ctx.AddChange(plugin.Change{
			Type:        "brew_update",
			Files:       []string{},
			Description: "Would run: brew update",
		})
		return nil
	}

	cmd := exec.Command(p.brewPath, "update")
	cmd.Dir = p.ctx.WorkDir
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("brew update failed: %s: %w", string(output), err)
	}

	ctx.AddChange(plugin.Change{
		Type:        "brew_update",
		Files:       []string{},
		Description: "Ran brew update",
	})

	if p.ctx.Logger != nil {
		p.ctx.Logger.Info("Brew update completed")
	}

	return nil
}

// runBrewCleanup runs brew cleanup
func (p *BrewPlugin) runBrewCleanup(ctx *plugin.ExecutionContext) error {
	if p.ctx.DryRun {
		ctx.AddChange(plugin.Change{
			Type:        "brew_cleanup",
			Files:       []string{},
			Description: "Would run: brew cleanup",
		})
		return nil
	}

	cmd := exec.Command(p.brewPath, "cleanup")
	cmd.Dir = p.ctx.WorkDir
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("brew cleanup failed: %s: %w", string(output), err)
	}

	ctx.AddChange(plugin.Change{
		Type:        "brew_cleanup",
		Files:       []string{},
		Description: "Ran brew cleanup",
	})

	if p.ctx.Logger != nil {
		p.ctx.Logger.Info("Brew cleanup completed")
	}

	return nil
}

// findBrew finds the Homebrew installation
func (p *BrewPlugin) findBrew() (string, error) {
	// Common Homebrew locations
	brewPaths := []string{
		"/usr/local/bin/brew",           // macOS Intel
		"/opt/homebrew/bin/brew",        // macOS Apple Silicon
		"/home/linuxbrew/.linuxbrew/bin/brew", // Linuxbrew
		"/home/linuxbrew/.linuxbrew/bin/brew", // Linuxbrew (alternative)
	}

	// First, try to find brew in PATH
	if brewPath, err := exec.LookPath("brew"); err == nil {
		// Verify it's executable
		if info, err := os.Stat(brewPath); err == nil && info.Mode().IsRegular() {
			return brewPath, nil
		}
	}

	// Try common locations
	for _, path := range brewPaths {
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return path, nil
		}
	}

	return "", fmt.Errorf("homebrew not found in PATH or common locations")
}

// installHomebrew installs Homebrew using the official installer
func (p *BrewPlugin) installHomebrew(ctx *plugin.ExecutionContext) error {
	// In dry-run mode, just report
	if p.ctx.DryRun {
		ctx.AddChange(plugin.Change{
			Type:        "brew_install",
			Files:       []string{},
			Description: "Would install Homebrew (requires user interaction for sudo password)",
		})
		return nil
	}

	if p.ctx.Logger != nil {
		p.ctx.Logger.Info("Installing Homebrew...")
	}

	// Determine OS to use correct installer
	osType := runtime.GOOS
	var installScript string

	if osType == "darwin" {
		// macOS - use official Homebrew installer
		installScript = "https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh"
	} else if osType == "linux" {
		// Linux - use Linuxbrew installer
		installScript = "https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh"
	} else {
		return fmt.Errorf("Homebrew installation not supported on %s", osType)
	}

	// Download and execute installer
	// Note: This requires curl to be available
	// Pipe curl output into bash for execution
	cmd := exec.Command("bash", "-c", fmt.Sprintf("curl -fsSL %s | bash", installScript))
	cmd.Stdin = os.Stdin  // Allow user interaction for password prompts
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	// Set NONINTERACTIVE=0 to allow prompts (user needs to enter password)
	cmd.Env = append(cmd.Env, "NONINTERACTIVE=0")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Homebrew installation failed: %w", err)
	}

	// Record installation
	ctx.AddChange(plugin.Change{
		Type:        "brew_install",
		Files:       []string{},
		Description: "Installed Homebrew",
	})

	if p.ctx.Logger != nil {
		p.ctx.Logger.Info("Homebrew installed successfully")
	}

	return nil
}

// Rollback rolls back brew bundle execution
func (p *BrewPlugin) Rollback(ctx *plugin.ExecutionContext) error {
	// For brew plugin, rollback means logging what was executed
	// We can't easily undo package installations, but we can log for debugging
	for _, bundle := range p.executedBundles {
		if p.ctx.Logger != nil {
			p.ctx.Logger.Info("Rolling back brew bundle", "bundle", bundle)
		}

		ctx.AddChange(plugin.Change{
			Type:        "brew_rollback",
			Files:       []string{bundle},
			Description: fmt.Sprintf("Rolled back brew bundle: %s", bundle),
		})
	}

	// If we installed Homebrew, log that (can't easily uninstall)
	if p.installedBrew {
		if p.ctx.Logger != nil {
			p.ctx.Logger.Warn("Homebrew was installed during this run - manual uninstallation may be required")
		}

		ctx.AddChange(plugin.Change{
			Type:        "brew_rollback",
			Files:       []string{},
			Description: "Homebrew was installed (manual uninstallation may be required)",
		})
	}

	// Clear executed bundles
	p.executedBundles = make([]string, 0)

	return nil
}

