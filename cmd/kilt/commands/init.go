// Package commands provides CLI command implementations for Kilt.
package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/unravelling/kilt/internal/core"
	"github.com/unravelling/kilt/pkg/logger"
)

var initCmd = &cobra.Command{
	Use:   "init <repo-url>",
	Short: "Initialise from a Git repository",
	Long: `Initialise Kilt with a dotfiles repository.

This command will:
1. Clone the repository to ~/.dotfiles
2. Create the Kilt directory structure (~/.kilt)
3. Set up initial state

Examples:
  kilt init https://github.com/you/dotfiles
  kilt init git@github.com:you/dotfiles.git`,
	Args: cobra.ExactArgs(1),
	RunE: runInit,
}

var initBranch string

// init sets up command line flags for the init command
func init() {
	initCmd.Flags().StringVarP(&initBranch, "branch", "b", "", "Branch to clone (default: repository default)")
}

// runInit initialises Kilt with a dotfiles repository
func runInit(cmd *cobra.Command, args []string) error {
	opts := GetGlobalOptions()
	log := GetLogger()

	repoURL := args[0]

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	dotfilesDir := filepath.Join(homeDir, ".dotfiles")
	kiltDir, err := core.GetKiltDir()
	if err != nil {
		return fmt.Errorf("failed to get kilt directory: %w", err)
	}

	log.Info("Initialising Kilt", "repo", repoURL)

	// Check if dotfiles directory already exists
	if _, err := os.Stat(dotfilesDir); err == nil {
		if !opts.Force {
			return logger.WithSuggestion(
				fmt.Errorf("dotfiles directory already exists: %s", dotfilesDir),
				"Use --force to overwrite or remove it manually",
			)
		}
		log.Warn("Removing existing dotfiles directory", "path", dotfilesDir)
		if !opts.DryRun {
			if err := os.RemoveAll(dotfilesDir); err != nil {
				return fmt.Errorf("failed to remove existing dotfiles directory: %w", err)
			}
		}
	}

	// Clone repository
	if opts.DryRun {
		log.Info("[DRY-RUN] Would clone repository", "url", repoURL, "dest", dotfilesDir)
	} else {
		log.Info("Cloning repository", "url", repoURL, "dest", dotfilesDir)

		cloneArgs := []string{"clone"}
		if initBranch != "" {
			cloneArgs = append(cloneArgs, "-b", initBranch)
		}
		cloneArgs = append(cloneArgs, repoURL, dotfilesDir)

		cloneCmd := exec.Command("git", cloneArgs...)
		cloneCmd.Stdout = os.Stdout
		cloneCmd.Stderr = os.Stderr

		if err := cloneCmd.Run(); err != nil {
			return logger.WithSuggestion(
				fmt.Errorf("failed to clone repository: %w", err),
				"Make sure the URL is correct and you have access to the repository",
			)
		}
	}

	// Create Kilt directory structure
	directories := []string{
		kiltDir,
		filepath.Join(kiltDir, "state"),
		filepath.Join(kiltDir, "backup"),
	}

	for _, dir := range directories {
		if opts.DryRun {
			log.Info("[DRY-RUN] Would create directory", "path", dir)
		} else {
			if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // G301: user directory needs 0755 for user access
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
			if opts.Verbose {
				log.Debug("Created directory", "path", dir)
			}
		}
	}

	// Initialise state
	if !opts.DryRun {
		stateDir, err := core.GetStateDir()
		if err != nil {
			return fmt.Errorf("failed to get state directory: %w", err)
		}

		_, err = core.NewStateManager(stateDir)
		if err != nil {
			return fmt.Errorf("failed to initialise state: %w", err)
		}
	}

	log.Success("Kilt initialised successfully!")
	log.Println()
	log.Info("Next steps:")
	log.Println("  1. Review your configuration in ~/.dotfiles/kilt.yaml (or .kilt/config.yaml)")
	log.Println("  2. Run 'kilt doctor' to verify your setup")
	log.Println("  3. Run 'kilt sync' to apply your dotfiles")

	return nil
}

// NewInitCmd returns the init command
func NewInitCmd() *cobra.Command {
	return initCmd
}
