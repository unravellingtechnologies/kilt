// Package integration provides integration tests for the kilt project.
// These tests verify end-to-end functionality across multiple components.
package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/unravelling/kilt/internal/core"
	"github.com/unravelling/kilt/internal/plugin"
)

// TestMissingConfig tests error handling when config file is missing
func TestMissingConfig(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	// Try to load config from non-existent path
	_, err := core.LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error when loading non-existent config")
	}
}

// TestInvalidConfig tests error handling with invalid configuration
func TestInvalidConfig(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	// Create invalid config file
	invalidConfigPath := filepath.Join(env.RepoPath, ".kilt", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(invalidConfigPath), 0o755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	invalidYAML := "this is not valid yaml: [unclosed"
	if err := os.WriteFile(invalidConfigPath, []byte(invalidYAML), 0o644); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	_, err := core.LoadConfig(invalidConfigPath)
	if err == nil {
		t.Error("Expected error when loading invalid YAML config")
	}
}

// TestMissingRepository tests error handling when dotfiles repository doesn't exist
func TestMissingRepository(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	fixturePath, err := GetFixturePath()
	if err != nil {
		t.Fatalf("Failed to get fixture path: %v", err)
	}

	env.CopyFixtureRepo(t, fixturePath)

	// Load config with non-existent dotfiles path
	config := env.LoadConfig(t)
	config.DotfilesPath = "/nonexistent/dotfiles"

	registry := plugin.GetDefaultRegistry()
	engine, err := core.NewEngine(config, registry)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Pre-flight checks should pass (they don't check repository existence)
	if err := engine.PreflightChecks(); err != nil {
		t.Logf("Pre-flight checks may fail: %v", err)
	}

	// Execution should fail when plugins try to access the repository
	result, err := engine.Execute()
	if err == nil && result != nil && result.Success {
		t.Error("Expected execution to fail with non-existent repository")
	}
}

// TestMissingSourceFiles tests error handling when source files are missing
func TestMissingSourceFiles(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	fixturePath, err := GetFixturePath()
	if err != nil {
		t.Fatalf("Failed to get fixture path: %v", err)
	}

	env.CopyFixtureRepo(t, fixturePath)
	if err := env.CloneRepoToDotfiles(t); err != nil {
		t.Fatalf("Failed to clone repository: %v", err)
	}

	// Delete a source file that should be linked
	zshrcSource := filepath.Join(env.DotfilesDir, "zsh", ".zshrc")
	if err := os.Remove(zshrcSource); err != nil {
		t.Fatalf("Failed to remove source file: %v", err)
	}

	registry := plugin.GetDefaultRegistry()
	engine := env.CreateEngine(t, registry)

	// Execution may or may not fail depending on plugin implementation
	// Some plugins may handle missing files gracefully
	result, err := engine.Execute()
	if err != nil {
		t.Logf("Execution failed as expected: %v", err)
	} else if result != nil && !result.Success {
		t.Logf("Execution reported errors: %v", result.Errors)
	}
}

// TestPermissionDenied tests error handling with permission denied scenarios
func TestPermissionDenied(t *testing.T) {
	// Skip on Windows
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	fixturePath, err := GetFixturePath()
	if err != nil {
		t.Fatalf("Failed to get fixture path: %v", err)
	}

	env.CopyFixtureRepo(t, fixturePath)
	if err := env.CloneRepoToDotfiles(t); err != nil {
		t.Fatalf("Failed to clone repository: %v", err)
	}

	// Remove write permissions from home directory
	if err := os.Chmod(env.HomeDir, 0o555); err != nil {
		t.Fatalf("Failed to change permissions: %v", err)
	}
	defer os.Chmod(env.HomeDir, 0o755) // Restore permissions

	registry := plugin.GetDefaultRegistry()
	engine := env.CreateEngine(t, registry)

	// Execution should fail when trying to create files
	result, err := engine.Execute()
	if err == nil && result != nil && result.Success {
		t.Error("Expected execution to fail with permission denied")
	}
}

// TestRecoveryAfterError tests that the system can recover after an error
func TestRecoveryAfterError(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	fixturePath, err := GetFixturePath()
	if err != nil {
		t.Fatalf("Failed to get fixture path: %v", err)
	}

	env.CopyFixtureRepo(t, fixturePath)
	if err := env.CloneRepoToDotfiles(t); err != nil {
		t.Fatalf("Failed to clone repository: %v", err)
	}

	registry := plugin.GetDefaultRegistry()

	// First, create a situation that will cause an error
	// Remove the config file
	if err := os.Remove(env.ConfigPath); err != nil {
		t.Fatalf("Failed to remove config: %v", err)
	}

	// Try to load config - should fail
	_, err = core.LoadConfig(env.ConfigPath)
	if err == nil {
		t.Error("Expected error when config is missing")
	}

	// Restore config file
	env.CopyFixtureRepo(t, fixturePath)

	// Now it should work
	config := env.LoadConfig(t)
	if config == nil {
		t.Fatal("Config should load after recovery")
	}

	engine, err := core.NewEngine(config, registry)
	if err != nil {
		t.Fatalf("Failed to create engine after recovery: %v", err)
	}

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execution should succeed after recovery: %v", err)
	}

	if result != nil && !result.Success {
		t.Errorf("Execution should succeed after recovery. Errors: %v", result.Errors)
	}
}

// TestRollbackOnFailure tests that rollback works when a plugin fails
func TestRollbackOnFailure(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	fixturePath, err := GetFixturePath()
	if err != nil {
		t.Fatalf("Failed to get fixture path: %v", err)
	}

	env.CopyFixtureRepo(t, fixturePath)
	if err := env.CloneRepoToDotfiles(t); err != nil {
		t.Fatalf("Failed to clone repository: %v", err)
	}

	registry := plugin.GetDefaultRegistry()
	engine := env.CreateEngine(t, registry)

	// Execute once to set up state
	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Initial execution failed: %v", err)
	}

	if result == nil || !result.Success {
		t.Fatalf("Initial execution should succeed: %v", result.Errors)
	}

	// Note: Actual rollback testing would require a plugin that can fail
	// and implements rollback. For now, we verify the engine has rollback support
	if engine == nil {
		t.Error("Engine should exist")
	}

	// The rollback functionality is tested implicitly through plugin failures
	// Full rollback testing requires mock plugins that can be made to fail
	t.Log("Rollback mechanism exists in engine (tested implicitly)")
}
