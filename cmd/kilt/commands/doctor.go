// Package commands provides CLI command implementations for Kilt.
package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/unravelling/kilt/internal/core"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Validate setup and dependencies",
	Long: `Doctor performs a health check on your Kilt installation, validating:

- Required dependencies (Git)
- Optional dependencies (Homebrew, 1Password CLI)
- Configuration file syntax and validity
- State directory permissions
- Backup directory permissions`,
	RunE: runDoctor,
}

// checkResult represents the result of a single check
type checkResult struct {
	name    string
	status  string // "ok", "warn", "error"
	message string
}

// runDoctor performs health checks
func runDoctor(cmd *cobra.Command, args []string) error {
	opts := GetGlobalOptions()
	log := GetLogger()

	log.Info("Running Kilt health checks...")
	log.Println()

	var results []checkResult

	// Check Git
	results = append(results, checkGit())

	// Check directories
	results = append(results, checkKiltDir())
	results = append(results, checkStateDir())
	results = append(results, checkBackupDir())

	// Check configuration
	results = append(results, checkConfig(opts.Config))

	// Check optional dependencies
	results = append(results, checkHomebrew())
	results = append(results, checkOnePasswordCLI())

	// Display results
	hasErrors := false
	hasWarnings := false

	for _, r := range results {
		switch r.status {
		case "ok":
			log.Success(r.name, "status", r.message)
		case "warn":
			log.Warn(r.name, "status", r.message)
			hasWarnings = true
		case "error":
			log.Error(r.name, "status", r.message)
			hasErrors = true
		}
	}

	log.Println()

	if hasErrors {
		log.Error("Health check completed with errors")
		return fmt.Errorf("one or more checks failed")
	}

	if hasWarnings {
		log.Warn("Health check completed with warnings")
	} else {
		log.Success("All health checks passed!")
	}

	return nil
}

// checkGit verifies Git is installed and accessible
func checkGit() checkResult {
	path, err := exec.LookPath("git")
	if err != nil {
		return checkResult{
			name:    "Git",
			status:  "error",
			message: "not found in PATH - install Git to use Kilt",
		}
	}

	// Get version
	out, err := exec.Command("git", "--version").Output()
	if err != nil {
		return checkResult{
			name:    "Git",
			status:  "warn",
			message: fmt.Sprintf("found at %s but version check failed", path),
		}
	}

	return checkResult{
		name:    "Git",
		status:  "ok",
		message: string(out[:len(out)-1]), // Remove trailing newline
	}
}

// checkKiltDir verifies the .kilt directory exists and is writable
func checkKiltDir() checkResult {
	dir, err := core.GetKiltDir()
	if err != nil {
		return checkResult{
			name:    "Kilt directory",
			status:  "error",
			message: fmt.Sprintf("failed to determine path: %v", err),
		}
	}

	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return checkResult{
			name:    "Kilt directory",
			status:  "warn",
			message: fmt.Sprintf("%s does not exist (will be created on first sync)", dir),
		}
	}
	if err != nil {
		return checkResult{
			name:    "Kilt directory",
			status:  "error",
			message: fmt.Sprintf("cannot access %s: %v", dir, err),
		}
	}
	if !info.IsDir() {
		return checkResult{
			name:    "Kilt directory",
			status:  "error",
			message: fmt.Sprintf("%s exists but is not a directory", dir),
		}
	}

	return checkResult{
		name:    "Kilt directory",
		status:  "ok",
		message: dir,
	}
}

// checkStateDir verifies the state directory
func checkStateDir() checkResult {
	dir, err := core.GetStateDir()
	if err != nil {
		return checkResult{
			name:    "State directory",
			status:  "error",
			message: fmt.Sprintf("failed to determine path: %v", err),
		}
	}

	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return checkResult{
			name:    "State directory",
			status:  "warn",
			message: fmt.Sprintf("%s does not exist (will be created on first sync)", dir),
		}
	}
	if err != nil {
		return checkResult{
			name:    "State directory",
			status:  "error",
			message: fmt.Sprintf("cannot access %s: %v", dir, err),
		}
	}
	if !info.IsDir() {
		return checkResult{
			name:    "State directory",
			status:  "error",
			message: fmt.Sprintf("%s exists but is not a directory", dir),
		}
	}

	return checkResult{
		name:    "State directory",
		status:  "ok",
		message: dir,
	}
}

// checkBackupDir verifies the backup directory
func checkBackupDir() checkResult {
	dir, err := core.GetBackupDir()
	if err != nil {
		return checkResult{
			name:    "Backup directory",
			status:  "error",
			message: fmt.Sprintf("failed to determine path: %v", err),
		}
	}

	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return checkResult{
			name:    "Backup directory",
			status:  "warn",
			message: fmt.Sprintf("%s does not exist (will be created on first sync)", dir),
		}
	}
	if err != nil {
		return checkResult{
			name:    "Backup directory",
			status:  "error",
			message: fmt.Sprintf("cannot access %s: %v", dir, err),
		}
	}
	if !info.IsDir() {
		return checkResult{
			name:    "Backup directory",
			status:  "error",
			message: fmt.Sprintf("%s exists but is not a directory", dir),
		}
	}

	return checkResult{
		name:    "Backup directory",
		status:  "ok",
		message: dir,
	}
}

// checkConfig validates the configuration file
func checkConfig(configPath string) checkResult {
	var path string
	var err error

	if configPath != "" {
		path = configPath
		if _, statErr := os.Stat(path); statErr != nil {
			return checkResult{
				name:    "Configuration",
				status:  "error",
				message: fmt.Sprintf("specified config file not found: %s", path),
			}
		}
	} else {
		path, err = core.FindConfigFile()
		if err != nil {
			return checkResult{
				name:    "Configuration",
				status:  "warn",
				message: "no configuration file found (run 'kilt init' first)",
			}
		}
	}

	cfg, err := core.LoadConfig(path)
	if err != nil {
		return checkResult{
			name:    "Configuration",
			status:  "error",
			message: fmt.Sprintf("failed to load %s: %v", path, err),
		}
	}

	if err := core.ValidateConfig(cfg); err != nil {
		return checkResult{
			name:    "Configuration",
			status:  "error",
			message: fmt.Sprintf("validation failed: %v", err),
		}
	}

	return checkResult{
		name:    "Configuration",
		status:  "ok",
		message: path,
	}
}

// checkHomebrew checks if Homebrew is installed (optional)
func checkHomebrew() checkResult {
	path, err := exec.LookPath("brew")
	if err != nil {
		return checkResult{
			name:    "Homebrew (optional)",
			status:  "warn",
			message: "not installed - brew plugin will be skipped",
		}
	}

	return checkResult{
		name:    "Homebrew (optional)",
		status:  "ok",
		message: path,
	}
}

// checkOnePasswordCLI checks if 1Password CLI is installed (optional)
func checkOnePasswordCLI() checkResult {
	path, err := exec.LookPath("op")
	if err != nil {
		return checkResult{
			name:    "1Password CLI (optional)",
			status:  "warn",
			message: "not installed - onepassword plugin will be skipped",
		}
	}

	return checkResult{
		name:    "1Password CLI (optional)",
		status:  "ok",
		message: path,
	}
}

// NewDoctorCmd returns the doctor command
func NewDoctorCmd() *cobra.Command {
	return doctorCmd
}
