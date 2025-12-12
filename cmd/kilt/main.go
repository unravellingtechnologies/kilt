// Package main provides the CLI entry point for Kilt, a Git-first dotfiles manager.
// It sets up the cobra command framework, registers commands, and handles
// global flags and error reporting.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/unravelling/kilt/cmd/kilt/commands"
	"github.com/unravelling/kilt/pkg/logger"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

var (
	globalDryRun  bool
	globalVerbose bool
	globalDebug   bool
	globalConfig  string
	globalNoColor bool
	globalForce   bool
)

var rootCmd = &cobra.Command{
	Use:   "kilt",
	Short: "Git-first Mac bootstrapper",
	Long: `Kilt is a Git-first dotfiles manager that transforms a fresh macOS (or Linux)
system into a personalised development environment using a single command.`,
	Version: version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Configure global options for all commands
		level := logger.LevelInfo
		if globalDebug {
			level = logger.LevelDebug
		} else if globalVerbose {
			level = logger.LevelInfo
		}

		log := logger.NewLogger(level, !globalNoColor)

		commands.SetGlobalOptions(commands.GlobalOptions{
			DryRun:  globalDryRun,
			Verbose: globalVerbose,
			Debug:   globalDebug,
			Config:  globalConfig,
			NoColor: globalNoColor,
			Force:   globalForce,
			Logger:  log,
		})
	},
}

// init sets up global command line flags
func init() {
	rootCmd.PersistentFlags().BoolVar(&globalDryRun, "dry-run", false, "Show what would happen without executing")
	rootCmd.PersistentFlags().BoolVarP(&globalVerbose, "verbose", "v", false, "Detailed logging output")
	rootCmd.PersistentFlags().BoolVar(&globalDebug, "debug", false, "Enable debug logging (includes verbose)")
	rootCmd.PersistentFlags().StringVar(&globalConfig, "config", "", "Custom config file location (default: .kilt/config.yaml or ~/.kilt/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&globalNoColor, "no-colour", false, "Disable coloured output")
	rootCmd.PersistentFlags().BoolVar(&globalForce, "force", false, "Skip confirmation prompts")

	// Set version information in commands package
	commands.SetVersionInfo(version, buildTime, "")

	// Register commands
	rootCmd.AddCommand(commands.NewInitCmd())
	rootCmd.AddCommand(commands.NewSyncCmd())
	rootCmd.AddCommand(commands.NewDoctorCmd())
	rootCmd.AddCommand(commands.NewVersionCmd())
	rootCmd.AddCommand(commands.NewResetCmd())
	rootCmd.AddCommand(commands.NewRestoreCmd())
	rootCmd.AddCommand(commands.NewBackupsCmd())
}

// main is the entry point for the Kilt CLI application
func main() {
	// Set up error formatting with colour support
	if err := rootCmd.Execute(); err != nil {
		errorMsg := logger.FormatError(err, !globalNoColor)
		fmt.Fprintln(os.Stderr, errorMsg)
		os.Exit(1)
	}
}
