// Package commands provides CLI command implementations for Kilt.
// It includes commands for restoring backups, listing backups, and other operations.
package commands

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/unravelling/kilt/internal/core"
	"github.com/unravelling/kilt/pkg/logger"
)

var restoreCmd = &cobra.Command{
	Use:   "restore <backup-id>",
	Short: "Restore files from a backup",
	Long: `Restore files from a backup using the backup ID (timestamp format: YYYYMMDD-HHMMSS).

The backup ID can be found by running 'kilt backups list'.`,
	Args: cobra.ExactArgs(1),
	RunE: runRestore,
}

// backupIDPattern matches backup ID format: YYYYMMDD-HHMMSS or YYYYMMDD-HHMMSS-N
var backupIDPattern = regexp.MustCompile(`^\d{8}-\d{6}(-\d+)?$`)

// init sets up command line flags for the restore command
func init() {
	restoreCmd.Flags().BoolVarP(&restoreForce, "force", "f", false, "Skip confirmation prompt")
}

var restoreForce bool

// runRestore executes the restore command, restoring files from a backup.
// It validates the backup ID format, checks if the backup exists, shows a preview
// of files to be restored (unless --force is used), and restores the files.
func runRestore(cmd *cobra.Command, args []string) error {
	opts := GetGlobalOptions()
	log := GetLogger()
	backupID := args[0]

	// Validate backup ID format
	if !backupIDPattern.MatchString(backupID) {
		return logger.WithSuggestion(
			fmt.Errorf("invalid backup ID format: %s (expected format: YYYYMMDD-HHMMSS or YYYYMMDD-HHMMSS-N)", backupID),
			"Run 'kilt backups list' to see available backups",
		)
	}

	// Get backup directory
	backupDir, err := core.GetBackupDir()
	if err != nil {
		return fmt.Errorf("failed to get backup directory: %w", err)
	}

	// Initialise backup manager
	bm, err := core.NewBackupManager(backupDir)
	if err != nil {
		return fmt.Errorf("failed to initialise backup manager: %w", err)
	}

	// Check if backup exists
	metadata, err := bm.LoadMetadata(backupID)
	if err != nil {
		return logger.WithSuggestion(
			fmt.Errorf("backup not found: %s", backupID),
			"Run 'kilt backups list' to see available backups",
		)
	}

	// Validate backup metadata integrity
	if len(metadata.Files) == 0 {
		return fmt.Errorf("backup %s contains no files", backupID)
	}

	// Use global force flag or local force flag
	forceRestore := restoreForce || opts.Force

	// Show preview of files to be restored
	if !forceRestore {
		log.Println("Backup:", backupID)
		log.Printf("Created: %s\n", metadata.Timestamp.Format("2006-01-02 15:04:05"))
		if metadata.Description != "" {
			log.Printf("Description: %s\n", metadata.Description)
		}
		log.Printf("Files to restore: %d\n\n", len(metadata.Files))
		log.Println("Files:")
		for i, file := range metadata.Files {
			if i < 10 {
				log.Printf("  - %s\n", file.OriginalPath)
			} else if i == 10 {
				log.Printf("  ... and %d more files\n", len(metadata.Files)-10)
				break
			}
		}
		log.Println()

		// Prompt for confirmation
		fmt.Print("Restore these files? [y/N]: ")
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			// On read error, treat as "no" for safety
			log.Println("Restore cancelled")
			return nil
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			log.Println("Restore cancelled")
			return nil
		}
	}

	// Restore files
	log.Info("Restoring backup", "backup_id", backupID, "files", len(metadata.Files))
	if err := bm.Restore(backupID); err != nil {
		return logger.WithSuggestion(
			fmt.Errorf("failed to restore backup: %w", err),
			"If this error persists, check backup integrity with 'kilt backups list'",
		)
	}

	log.Success("Backup restored successfully", "backup_id", backupID, "files", len(metadata.Files))
	return nil
}

// NewRestoreCmd returns the restore command
func NewRestoreCmd() *cobra.Command {
	return restoreCmd
}
