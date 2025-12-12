// Package commands provides CLI command implementations for Kilt.
package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/unravelling/kilt/internal/core"
	"github.com/unravelling/kilt/internal/plugin"
	_ "github.com/unravelling/kilt/internal/plugins/alternates"
	_ "github.com/unravelling/kilt/internal/plugins/brew"
	_ "github.com/unravelling/kilt/internal/plugins/directories"
	_ "github.com/unravelling/kilt/internal/plugins/dotfiles"
	_ "github.com/unravelling/kilt/internal/plugins/git"
	_ "github.com/unravelling/kilt/internal/plugins/onchange"
	_ "github.com/unravelling/kilt/internal/plugins/onepassword"
	_ "github.com/unravelling/kilt/internal/plugins/runonce"
	"github.com/unravelling/kilt/pkg/logger"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronise dotfiles",
	Long: `Sync pulls the latest changes from your dotfiles repository and applies them.

This command will:
1. Pull the latest changes from Git (if configured)
2. Run all enabled plugins in dependency order
3. Apply dotfile changes (symlinks, templates, etc.)
4. Execute configured scripts and commands

Use --dry-run to preview changes without applying them.`,
	RunE: runSync,
}

// runSync executes the synchronisation process
func runSync(cmd *cobra.Command, args []string) error {
	opts := GetGlobalOptions()
	log := GetLogger()

	log.Info("Starting Kilt sync...")

	// Find and load configuration
	var configPath string
	if opts.Config != "" {
		configPath = opts.Config
	} else {
		var err error
		configPath, err = core.FindConfigFile()
		if err != nil {
			return logger.WithSuggestion(
				fmt.Errorf("failed to find configuration: %w", err),
				"Run 'kilt init <repo-url>' to initialise Kilt first",
			)
		}
	}

	if opts.Verbose {
		log.Debug("Using configuration file", "path", configPath)
	}

	cfg, err := core.LoadConfig(configPath)
	if err != nil {
		return logger.WithSuggestion(
			fmt.Errorf("failed to load configuration: %w", err),
			"Check your configuration file syntax with 'kilt doctor'",
		)
	}

	// Expand paths in configuration
	if err := core.ExpandPaths(cfg); err != nil {
		return logger.WithSuggestion(
			fmt.Errorf("failed to expand paths: %w", err),
			"Check for invalid path references in your configuration",
		)
	}

	// Create engine with the default plugin registry
	registry := plugin.GetDefaultRegistry()
	engine, err := core.NewEngine(cfg, registry)
	if err != nil {
		return logger.WithSuggestion(
			fmt.Errorf("failed to create engine: %w", err),
			"Run 'kilt doctor' to diagnose the issue",
		)
	}

	// Configure engine
	engine.SetDryRun(opts.DryRun)
	engine.SetVerbose(opts.Verbose)
	engine.SetLogger(log)

	// Execute
	result, err := engine.Execute()
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Report results
	log.Println()
	if result.Success {
		if opts.DryRun {
			log.Success("[DRY-RUN] Sync completed successfully", "plugins", result.PluginsRun, "duration", result.Duration)
		} else {
			log.Success("Sync completed successfully", "plugins", result.PluginsRun, "duration", result.Duration)
		}

		if len(result.Changes) > 0 && opts.Verbose {
			log.Println()
			log.Info("Changes made:")
			for _, change := range result.Changes {
				log.Printf("  - [%s] %s\n", change.Type, change.Description)
			}
		}
	} else {
		log.Error("Sync completed with errors")
		for _, err := range result.Errors {
			log.Error("Error", "message", err.Error())
		}
		if result.RolledBack {
			log.Warn("Changes were rolled back due to errors")
		}
	}

	if !result.Success {
		return fmt.Errorf("sync completed with %d error(s)", len(result.Errors))
	}

	return nil
}

// NewSyncCmd returns the sync command
func NewSyncCmd() *cobra.Command {
	return syncCmd
}
