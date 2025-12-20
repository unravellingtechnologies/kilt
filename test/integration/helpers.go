package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/unravelling/kilt/internal/core"
	"github.com/unravelling/kilt/internal/plugin"
	_ "github.com/unravelling/kilt/internal/plugins/alternates"
	_ "github.com/unravelling/kilt/internal/plugins/brew"
	_ "github.com/unravelling/kilt/internal/plugins/directories"
	_ "github.com/unravelling/kilt/internal/plugins/dotfiles"
	_ "github.com/unravelling/kilt/internal/plugins/git"
	_ "github.com/unravelling/kilt/internal/plugins/onchange"
	_ "github.com/unravelling/kilt/internal/plugins/onepassword"
	_ "github.com/unravelling/kilt/internal/plugins/runonce"
)

// TestEnvironment represents an isolated test environment
type TestEnvironment struct {
	RootDir      string
	HomeDir      string
	DotfilesDir  string
	KiltDir      string
	StateDir     string
	BackupDir    string
	ConfigPath   string
	RepoPath     string
	OriginalHome string
}

// SetupTestEnvironment creates a complete test environment with all necessary directories
func SetupTestEnvironment(t *testing.T) *TestEnvironment {
	t.Helper()

	rootDir := t.TempDir()
	homeDir := filepath.Join(rootDir, "home")
	kiltDir := filepath.Join(homeDir, ".kilt")
	stateDir := filepath.Join(kiltDir, "state")
	backupDir := filepath.Join(kiltDir, "backup")
	dotfilesDir := filepath.Join(homeDir, ".dotfiles")
	repoPath := filepath.Join(rootDir, "test-repo")
	configPath := filepath.Join(repoPath, ".kilt", "config.yaml")

	// Create directory structure
	for _, dir := range []string{
		homeDir,
		kiltDir,
		stateDir,
		backupDir,
		dotfilesDir,
		repoPath,
		filepath.Join(repoPath, ".kilt"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Store original HOME
	originalHome := os.Getenv("HOME")

	// Set HOME to test home directory
	if err := os.Setenv("HOME", homeDir); err != nil {
		t.Fatalf("Failed to set HOME: %v", err)
	}

	return &TestEnvironment{
		RootDir:      rootDir,
		HomeDir:      homeDir,
		DotfilesDir:  dotfilesDir,
		KiltDir:      kiltDir,
		StateDir:     stateDir,
		BackupDir:    backupDir,
		ConfigPath:   configPath,
		RepoPath:     repoPath,
		OriginalHome: originalHome,
	}
}

// Cleanup restores the original environment
func (te *TestEnvironment) Cleanup(t *testing.T) {
	t.Helper()

	// Restore original HOME
	if err := os.Setenv("HOME", te.OriginalHome); err != nil {
		t.Logf("Warning: Failed to restore HOME: %v", err)
	}
}

// CopyFixtureRepo copies the fixture repository to the test environment
func (te *TestEnvironment) CopyFixtureRepo(t *testing.T, fixturePath string) {
	t.Helper()

	sourceRepo := filepath.Join(fixturePath, "repo")
	if _, err := os.Stat(sourceRepo); os.IsNotExist(err) {
		t.Fatalf("Fixture repository not found at %s", sourceRepo)
	}

	// Copy fixture repo to test repo path
	if err := copyDir(sourceRepo, te.RepoPath); err != nil {
		t.Fatalf("Failed to copy fixture repository: %v", err)
	}

	// Initialise as git repository
	if err := te.InitGitRepo(t); err != nil {
		t.Fatalf("Failed to initialise git repository: %v", err)
	}
}

// InitGitRepo initialises the repository as a git repository (idempotent)
func (te *TestEnvironment) InitGitRepo(t *testing.T) error {
	t.Helper()

	// Check if already a git repository
	gitDir := filepath.Join(te.RepoPath, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		// Already initialised - just ensure config is set and commit any new files
		cmd := exec.Command("git", "config", "user.name", "Test User")
		cmd.Dir = te.RepoPath
		_ = cmd.Run() // Ignore errors - config may already be set

		cmd = exec.Command("git", "config", "user.email", "test@example.com")
		cmd.Dir = te.RepoPath
		_ = cmd.Run()

		cmd = exec.Command("git", "config", "receive.denyCurrentBranch", "ignore")
		cmd.Dir = te.RepoPath
		_ = cmd.Run()

		// Add and commit any new files (ignore errors if nothing to commit)
		cmd = exec.Command("git", "add", ".")
		cmd.Dir = te.RepoPath
		_ = cmd.Run()

		cmd = exec.Command("git", "commit", "-m", "Update files")
		cmd.Dir = te.RepoPath
		_ = cmd.Run() // May fail if nothing to commit - that's OK

		return nil
	}

	// Initialise git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = te.RepoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to init git repo: %w", err)
	}

	// Configure git user (required for commits)
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = te.RepoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to configure git user: %w", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = te.RepoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to configure git email: %w", err)
	}

	// Allow pushing to the checked-out branch (required for test scenarios where
	// we clone from a non-bare repo and push back to it)
	cmd = exec.Command("git", "config", "receive.denyCurrentBranch", "ignore")
	cmd.Dir = te.RepoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to configure receive.denyCurrentBranch: %w", err)
	}

	// Add all files and create initial commit
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = te.RepoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to git add: %w", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = te.RepoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to git commit: %w", err)
	}

	return nil
}

// CloneRepoToDotfiles clones the test repository to the dotfiles directory
func (te *TestEnvironment) CloneRepoToDotfiles(t *testing.T) error {
	t.Helper()

	// Clone repo to dotfiles directory
	//nolint:gosec // G204: te.RepoPath and te.DotfilesDir are test-controlled paths, not user input
	cmd := exec.Command("git", "clone", te.RepoPath, te.DotfilesDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Update config path to point to cloned repo
	te.ConfigPath = filepath.Join(te.DotfilesDir, ".kilt", "config.yaml")

	return nil
}

// LoadConfig loads the configuration from the test environment
func (te *TestEnvironment) LoadConfig(t *testing.T) *core.Config {
	t.Helper()

	config, err := core.LoadConfig(te.ConfigPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Update dotfiles path to point to our test dotfiles directory
	config.DotfilesPath = te.DotfilesDir

	return config
}

// CreateEngine creates an engine instance with the test configuration
func (te *TestEnvironment) CreateEngine(t *testing.T, registry *plugin.Registry) *core.Engine {
	t.Helper()

	config := te.LoadConfig(t)

	// Change to dotfiles directory so relative paths in config (e.g., scripts/install.sh)
	// are resolved correctly. This matches real-world usage where kilt operates
	// with the dotfiles directory as the working directory.
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	if err := os.Chdir(te.DotfilesDir); err != nil {
		t.Fatalf("Failed to change to dotfiles directory: %v", err)
	}
	t.Cleanup(func() {
		// Restore original working directory after test
		if err := os.Chdir(originalWd); err != nil {
			t.Logf("Warning: Failed to restore working directory: %v", err)
		}
	})

	engine, err := core.NewEngine(config, registry)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	return engine
}

// FileExists checks if a file exists in the test environment
func (te *TestEnvironment) FileExists(path string) bool {
	expandedPath, err := core.ExpandPath(path)
	if err != nil {
		return false
	}
	if _, err := os.Stat(expandedPath); err != nil {
		if os.IsNotExist(err) {
			return false
		}
		// Treat other errors (e.g., permission) as non-existent for test helpers
		return false
	}
	return true
}

// ReadFile reads a file from the test environment
func (te *TestEnvironment) ReadFile(t *testing.T, path string) string {
	t.Helper()

	expandedPath, err := core.ExpandPath(path)
	if err != nil {
		t.Fatalf("Failed to expand path %s: %v", path, err)
	}

	data, err := os.ReadFile(expandedPath)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", expandedPath, err)
	}

	return string(data)
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		srcData, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if err := os.WriteFile(dstPath, srcData, info.Mode()); err != nil {
			return err
		}

		return nil
	})
}

// GetFixturePath returns the path to the test fixtures directory
func GetFixturePath() (string, error) {
	// Get the project root by looking for go.mod
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up to find go.mod
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "test", "fixtures"), nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root")
		}
		dir = parent
	}
}
