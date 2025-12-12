package integration

import (
	"path/filepath"
	"testing"

	"github.com/unravelling/kilt/internal/plugin"
)

// TestInitSyncFlow tests the complete init -> sync workflow
func TestInitSyncFlow(t *testing.T) {
	// Setup test environment
	env := SetupTestEnvironment(t)
	defer env.Cleanup(t)

	// Get fixture path
	fixturePath, err := GetFixturePath()
	if err != nil {
		t.Fatalf("Failed to get fixture path: %v", err)
	}

	// Copy fixture repository
	env.CopyFixtureRepo(t, fixturePath)

	// Test: Clone repository to dotfiles directory (simulating kilt init)
	t.Run("CloneRepository", func(t *testing.T) {
		if err := env.CloneRepoToDotfiles(t); err != nil {
			t.Fatalf("Failed to clone repository: %v", err)
		}

		// Verify repository was cloned
		if !env.FileExists(env.DotfilesDir) {
			t.Error("Dotfiles directory should exist")
		}

		// Verify config file exists
		if !env.FileExists(env.ConfigPath) {
			t.Error("Config file should exist")
		}
	})

	// Test: Load configuration
	t.Run("LoadConfiguration", func(t *testing.T) {
		config := env.LoadConfig(t)

		if config.DotfilesPath == "" {
			t.Error("Dotfiles path should be set")
		}

		if len(config.Dotfiles) == 0 {
			t.Error("Dotfiles should be configured")
		}

		if len(config.Directories) == 0 {
			t.Error("Directories should be configured")
		}
	})

	// Test: Execute sync (simulating kilt sync)
	t.Run("ExecuteSync", func(t *testing.T) {
		registry := plugin.GetDefaultRegistry()
		engine := env.CreateEngine(t, registry)

		// Pre-flight checks
		if err := engine.PreflightChecks(); err != nil {
			t.Fatalf("Pre-flight checks failed: %v", err)
		}

		// Execute engine
		result, err := engine.Execute()
		if err != nil {
			t.Fatalf("Engine execution failed: %v", err)
		}

		if !result.Success {
			t.Errorf("Execution should succeed. Errors: %v", result.Errors)
		}

		if result.PluginsRun == 0 {
			t.Error("At least one plugin should have run")
		}
	})

	// Test: Verify files were created
	t.Run("VerifyFilesCreated", func(t *testing.T) {
		// Check that dotfiles were linked
		zshrcPath := filepath.Join(env.HomeDir, ".zshrc")
		if !env.FileExists(zshrcPath) {
			t.Error(".zshrc should exist in home directory")
		}

		gitconfigPath := filepath.Join(env.HomeDir, ".gitconfig")
		if !env.FileExists(gitconfigPath) {
			t.Error(".gitconfig should exist in home directory")
		}

		// Check that directories were created
		devPersonalPath := filepath.Join(env.HomeDir, "dev", "personal")
		if !env.FileExists(devPersonalPath) {
			t.Error("dev/personal directory should exist")
		}

		devWorkPath := filepath.Join(env.HomeDir, "dev", "work")
		if !env.FileExists(devWorkPath) {
			t.Error("dev/work directory should exist")
		}
	})

	// Test: Verify idempotency (run sync again)
	t.Run("Idempotency", func(t *testing.T) {
		registry := plugin.GetDefaultRegistry()
		engine := env.CreateEngine(t, registry)

		result, err := engine.Execute()
		if err != nil {
			t.Fatalf("Second execution failed: %v", err)
		}

		if !result.Success {
			t.Errorf("Second execution should succeed. Errors: %v", result.Errors)
		}

		// Files should still exist
		zshrcPath := filepath.Join(env.HomeDir, ".zshrc")
		if !env.FileExists(zshrcPath) {
			t.Error(".zshrc should still exist after second run")
		}
	})
}
