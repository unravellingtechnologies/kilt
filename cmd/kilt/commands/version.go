// Package commands provides CLI command implementations for Kilt.
package commands

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version information - set at build time via ldflags
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display Kilt version, build time, and Git commit information.`,
	Run:   runVersion,
}

// runVersion displays version information
func runVersion(cmd *cobra.Command, args []string) {
	log := GetLogger()
	opts := GetGlobalOptions()

	if opts.Verbose {
		log.Println("Version:   ", Version)
		log.Println("Build Time:", BuildTime)
		log.Println("Git Commit:", GitCommit)
		log.Println("Go Version:", runtime.Version())
		log.Println("OS/Arch:   ", runtime.GOOS+"/"+runtime.GOARCH)
	} else {
		fmt.Printf("kilt version %s\n", Version)
	}
}

// NewVersionCmd returns the version command
func NewVersionCmd() *cobra.Command {
	return versionCmd
}

// SetVersionInfo sets the version information (called from main)
func SetVersionInfo(version, buildTime, gitCommit string) {
	Version = version
	BuildTime = buildTime
	GitCommit = gitCommit
}
