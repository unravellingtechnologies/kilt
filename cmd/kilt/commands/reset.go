// Package commands provides CLI command implementations for Kilt.
package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/unravelling/kilt/internal/core"
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Clear state and start fresh",
	Long: `Reset clears all Kilt state, allowing plugins to run again as if for the first time.

This is useful when:
- You want to re-run run-once scripts
- State has become corrupted
- You're troubleshooting plugin issues

This does NOT delete your configuration or backups.`,
	RunE: runReset,
}

// runReset clears all kilt state
func runReset(cmd *cobra.Command, args []string) error {
	opts := GetGlobalOptions()
	log := GetLogger()

	// Get state directory
	stateDir, err := core.GetStateDir()
	if err != nil {
		return fmt.Errorf("failed to get state directory: %w", err)
	}

	// Initialise state manager
	sm, err := core.NewStateManager(stateDir)
	if err != nil {
		return fmt.Errorf("failed to initialise state manager: %w", err)
	}

	// Confirm unless --force is set
	forceReset := opts.Force
	if !forceReset {
		log.Warn("This will clear all Kilt state, causing all plugins to run as if for the first time.")
		log.Println("State directory:", stateDir)
		log.Println()

		fmt.Print("Are you sure you want to reset? [y/N]: ")
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			// On read error, treat as "no" for safety
			log.Println("Reset cancelled")
			return nil
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			log.Println("Reset cancelled")
			return nil
		}
	}

	if opts.DryRun {
		log.Info("[DRY-RUN] Would clear state in", "path", stateDir)
		return nil
	}

	// Clear state
	if err := sm.ClearState(); err != nil {
		return fmt.Errorf("failed to clear state: %w", err)
	}

	log.Success("State cleared successfully")
	log.Info("Run 'kilt sync' to re-apply your configuration")

	return nil
}

// NewResetCmd returns the reset command
func NewResetCmd() *cobra.Command {
	return resetCmd
}
