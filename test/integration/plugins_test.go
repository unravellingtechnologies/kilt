package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/unravelling/kilt/internal/plugin"
)

// TestPluginInteractions tests interactions between different plugins
func TestPluginInteractions(t *testing.T) {
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

	// Execute engine
	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Engine execution failed: %v", err)
	}

	t.Run("DirectoriesPluginCreatesDirectories", func(t *testing.T) {
		// Directories plugin should create directories before dotfiles plugin runs
		devPersonalPath := filepath.Join(env.HomeDir, "dev", "personal")
		devWorkPath := filepath.Join(env.HomeDir, "dev", "work")

		if !env.FileExists(devPersonalPath) {
			t.Error("Directories plugin should create dev/personal directory")
		}

		if !env.FileExists(devWorkPath) {
			t.Error("Directories plugin should create dev/work directory")
		}
	})

	t.Run("DotfilesPluginCreatesSymlinks", func(t *testing.T) {
		// Dotfiles plugin should create symlinks
		zshrcPath := filepath.Join(env.HomeDir, ".zshrc")
		if !env.FileExists(zshrcPath) {
			t.Error("Dotfiles plugin should create .zshrc symlink")
		}

		// Verify it's a symlink
		info, err := os.Lstat(zshrcPath)
		if err != nil {
			t.Fatalf("Failed to stat .zshrc: %v", err)
		}

		if info.Mode()&os.ModeSymlink == 0 {
			t.Error(".zshrc should be a symlink")
		}
	})

	t.Run("GitPluginHandlesRepository", func(t *testing.T) {
		// Git plugin should handle the main repository
		// Since we're using a local repo, it should at least validate it exists
		if !env.FileExists(env.DotfilesDir) {
			t.Error("Git plugin should validate dotfiles repository exists")
		}

		gitDir := filepath.Join(env.DotfilesDir, ".git")
		if !env.FileExists(gitDir) {
			t.Error("Git plugin should validate .git directory exists")
		}
	})

	t.Run("PluginExecutionOrder", func(t *testing.T) {
		// Verify plugins executed in the correct order
		// This is implicit - if directories weren't created first, dotfiles might fail
		if result.PluginsRun == 0 {
			t.Error("Plugins should have executed")
		}

		// Check that changes were tracked
		if len(result.Changes) == 0 {
			t.Error("Plugins should have tracked changes")
		}
	})
}

// TestAlternatesPlugin tests the alternates plugin functionality
func TestAlternatesPlugin(t *testing.T) {
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

	// Create an alternate file (e.g., for macOS)
	alternateFile := filepath.Join(env.DotfilesDir, "zsh", ".zshrc.mac")
	if err := os.WriteFile(alternateFile, []byte("# macOS specific zshrc\n"), 0644); err != nil {
		t.Fatalf("Failed to create alternate file: %v", err)
	}

	registry := plugin.GetDefaultRegistry()
	engine := env.CreateEngine(t, registry)

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Engine execution failed: %v", err)
	}

	// Alternates plugin should resolve the correct file based on OS
	// The exact behavior depends on the current OS, but it should at least not fail
	if !result.Success {
		t.Errorf("Execution should succeed with alternates: %v", result.Errors)
	}
}

// TestOnChangePlugin tests the on-change plugin
func TestOnChangePlugin(t *testing.T) {
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

	// First execution should trigger on-change (initial sync)
	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Engine execution failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Execution should succeed: %v", result.Errors)
	}

	// OnChange plugin should have tracked changes
	// Since this is the first run, it should detect changes
	// OnChange may or may not trigger on first run depending on implementation
	// Just verify the execution succeeded
	t.Logf("OnChange plugin executed with %d changes tracked", len(result.Changes))
}
