// Package integration contains end-to-end integration tests for Kilt.
// This file contains build verification tests that ensure the project compiles correctly.
package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestBuildCompiles verifies that the project can be compiled successfully.
// This test catches issues like missing source files (e.g., due to .gitignore problems)
// early in the test phase rather than waiting for the build step.
func TestBuildCompiles(t *testing.T) {
	projectRoot := findProjectRoot(t)

	// Verify critical source files exist
	mainGo := filepath.Join(projectRoot, "cmd", "kilt", "main.go")
	if _, err := os.Stat(mainGo); os.IsNotExist(err) {
		t.Fatalf("cmd/kilt/main.go not found - this may indicate a .gitignore issue excluding source files")
	}

	// Verify commands package exists
	commandsDir := filepath.Join(projectRoot, "cmd", "kilt", "commands")
	if _, err := os.Stat(commandsDir); os.IsNotExist(err) {
		t.Fatalf("cmd/kilt/commands/ directory not found - this may indicate a .gitignore issue excluding source files")
	}

	// Attempt to build - use /dev/null on Unix, NUL on Windows
	outputPath := os.DevNull
	if runtime.GOOS == "windows" {
		outputPath = "NUL"
	}

	//nolint:gosec // G204: projectRoot is derived from finding go.mod, not user input
	cmd := exec.Command("go", "build", "-o", outputPath, "./cmd/kilt")
	cmd.Dir = projectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Build failed: %v\nOutput:\n%s", err, output)
	}
}

// findProjectRoot locates the project root by searching for go.mod.
func findProjectRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("Could not find project root (no go.mod found)")
		}
		dir = parent
	}
}
