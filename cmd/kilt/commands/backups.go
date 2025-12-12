// Package commands provides CLI command implementations for Kilt.
// It includes commands for restoring backups, listing backups, and other operations.
package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/unravelling/kilt/internal/core"
)

var backupsCmd = &cobra.Command{
	Use:   "backups",
	Short: "Manage backups",
	Long:  "List and manage backup snapshots",
}

var backupsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all backups",
	Long:  "List all available backups with details",
	RunE:  runBackupsList,
}

var backupsListJSON bool

// init sets up command line flags and subcommands for the backups command
func init() {
	backupsListCmd.Flags().BoolVar(&backupsListJSON, "json", false, "Output in JSON format")
	backupsCmd.AddCommand(backupsListCmd)
}

// runBackupsList executes the backups list command, displaying all available backups
// in either table or JSON format based on the --json flag.
func runBackupsList(cmd *cobra.Command, args []string) error {
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

	// List backups
	backups, err := bm.ListBackups()
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(backups) == 0 {
		if backupsListJSON {
			fmt.Println("[]")
		} else {
			fmt.Println("No backups found")
		}
		return nil
	}

	// Output in JSON format if requested
	if backupsListJSON {
		return outputBackupsJSON(backups)
	}

	// Output in table format
	return outputBackupsTable(backups)
}

// outputBackupsJSON outputs backup metadata in JSON format to stdout.
func outputBackupsJSON(backups []*core.BackupMetadata) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(backups)
}

// outputBackupsTable outputs backup metadata in a formatted table to stdout.
func outputBackupsTable(backups []*core.BackupMetadata) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, "BACKUP ID\tDATE\tTIME\tFILES\tSIZE\tDESCRIPTION") //nolint:errcheck // Writing to stdout rarely fails and is non-recoverable
	_, _ = fmt.Fprintln(w, "---------\t----\t----\t-----\t----\t-----------") //nolint:errcheck // Writing to stdout rarely fails and is non-recoverable

	for _, backup := range backups {
		date := backup.Timestamp.Format("2006-01-02")
		timeStr := backup.Timestamp.Format("15:04:05")
		fileCount := len(backup.Files)
		sizeStr := formatSize(backup.TotalSize)
		desc := backup.Description
		if desc == "" {
			desc = "-"
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n", //nolint:errcheck // Writing to stdout rarely fails and is non-recoverable
			backup.BackupID, date, timeStr, fileCount, sizeStr, desc)
	}

	return w.Flush()
}

// formatSize formats a byte count into a human-readable string with appropriate units (B, KB, MB, GB, TB, PB, EB).
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// NewBackupsCmd returns the backups command
func NewBackupsCmd() *cobra.Command {
	return backupsCmd
}
