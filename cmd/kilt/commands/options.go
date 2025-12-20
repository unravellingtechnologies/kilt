// Package commands provides CLI command implementations for Kilt.
package commands

import "github.com/unravelling/kilt/pkg/logger"

// GlobalOptions holds the global command-line options shared across all commands.
// These are set by main.go after parsing persistent flags.
type GlobalOptions struct {
	DryRun  bool
	Verbose bool
	Debug   bool
	Config  string
	NoColor bool
	Force   bool
	Logger  *logger.Logger
}

// globalOpts holds the current global options
var globalOpts GlobalOptions

// SetGlobalOptions sets the global options from main.go
func SetGlobalOptions(opts GlobalOptions) {
	globalOpts = opts
}

// GetGlobalOptions returns the current global options
func GetGlobalOptions() GlobalOptions {
	return globalOpts
}

// GetLogger returns a configured logger based on global options.
// If no logger is set, creates a default one.
func GetLogger() *logger.Logger {
	if globalOpts.Logger != nil {
		return globalOpts.Logger
	}

	level := logger.LevelInfo
	if globalOpts.Debug {
		level = logger.LevelDebug
	}

	return logger.NewLogger(level, !globalOpts.NoColor)
}
