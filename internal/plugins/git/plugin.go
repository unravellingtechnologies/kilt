package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/unravelling/kilt/internal/core"
	"github.com/unravelling/kilt/internal/plugin"
)

// GitPlugin handles bidirectional Git synchronization for the dotfiles repository
type GitPlugin struct {
	ctx          *plugin.PluginContext
	dotfilesPath string
	repoPath     string
}

func init() {
	plugin.RegisterPlugin(&GitPlugin{})
}

// Name returns the plugin name
func (p *GitPlugin) Name() string {
	return "git"
}

// Version returns the plugin version
func (p *GitPlugin) Version() string {
	return "1.0.0"
}

// Description returns the plugin description
func (p *GitPlugin) Description() string {
	return "Bidirectional Git synchronization for dotfiles repository"
}

// Dependencies returns plugin dependencies
func (p *GitPlugin) Dependencies() []string {
	return []string{}
}

// Phase returns the execution phase
func (p *GitPlugin) Phase() plugin.ExecutionPhase {
	return plugin.PhasePreSync
}

// Initialize initializes the plugin with context
func (p *GitPlugin) Initialize(ctx *plugin.PluginContext) error {
	p.ctx = ctx

	cfg, ok := ctx.Config.(*core.Config)
	if !ok {
		return fmt.Errorf("invalid config type")
	}

	// Get dotfiles path from config
	p.dotfilesPath = cfg.DotfilesPath
	if p.dotfilesPath == "" {
		p.dotfilesPath = core.DefaultDotfilesPath()
	}

	// Expand path
	expandedPath, err := core.ExpandPath(p.dotfilesPath)
	if err != nil {
		return fmt.Errorf("failed to expand dotfiles path: %w", err)
	}
	p.repoPath = expandedPath

	return nil
}

// Validate validates the plugin configuration
func (p *GitPlugin) Validate() error {
	if p.ctx == nil {
		return fmt.Errorf("plugin context not initialized")
	}

	// Check if dotfiles repository exists
	if _, err := os.Stat(p.repoPath); os.IsNotExist(err) {
		return fmt.Errorf("dotfiles repository not found at %s (run 'kilt init' first)", p.repoPath)
	}

	// Check if it's a git repository
	gitDir := filepath.Join(p.repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository: %s", p.repoPath)
	}

	return nil
}

// Execute executes the plugin logic
func (p *GitPlugin) Execute(ctx *plugin.ExecutionContext) error {
	// 1. Fetch remote
	if err := p.gitFetch(); err != nil {
		return fmt.Errorf("git fetch failed: %w", err)
	}

	// 2. Check for local changes
	hasLocalChanges, err := p.hasUncommittedChanges()
	if err != nil {
		return fmt.Errorf("failed to check local changes: %w", err)
	}

	// 3. Check for remote changes
	hasRemoteChanges, err := p.hasRemoteChanges()
	if err != nil {
		return fmt.Errorf("failed to check remote changes: %w", err)
	}

	// 4. Handle based on state
	if hasLocalChanges && hasRemoteChanges {
		// Both local and remote changes - attempt to pull with rebase
		if err := p.pullWithRebase(); err != nil {
			return fmt.Errorf("merge conflict detected in %s. Please resolve manually: %w", p.repoPath, err)
		}
		// After successful pull, commit and push local changes
		if err := p.commitAndPush(); err != nil {
			return fmt.Errorf("failed to commit and push local changes: %w", err)
		}
	} else if hasRemoteChanges {
		// Only remote changes - pull
		if err := p.gitPull(); err != nil {
			return fmt.Errorf("git pull failed: %w", err)
		}
	} else if hasLocalChanges {
		// Only local changes - commit and push
		if err := p.commitAndPush(); err != nil {
			return fmt.Errorf("failed to commit and push local changes: %w", err)
		}
	} else {
		// No changes - nothing to do
		if p.ctx.Logger != nil {
			p.ctx.Logger.Info("No changes detected in dotfiles repository")
		}
	}

	return nil
}

// gitFetch fetches remote changes
func (p *GitPlugin) gitFetch() error {
	cmd := exec.Command("git", "fetch", "origin")
	cmd.Dir = p.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git fetch failed: %s: %w", string(output), err)
	}
	return nil
}

// hasUncommittedChanges checks if there are uncommitted local changes
func (p *GitPlugin) hasUncommittedChanges() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = p.repoPath
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}

	return len(strings.TrimSpace(string(output))) > 0, nil
}

// hasRemoteChanges checks if there are remote changes
func (p *GitPlugin) hasRemoteChanges() (bool, error) {
	// Get current branch
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = p.repoPath
	branchOutput, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to get current branch: %w", err)
	}
	branch := strings.TrimSpace(string(branchOutput))

	// Compare HEAD with origin/branch
	cmd = exec.Command("git", "rev-list", "--count", fmt.Sprintf("HEAD..origin/%s", branch))
	cmd.Dir = p.repoPath
	output, err := cmd.Output()
	if err != nil {
		// If origin/branch doesn't exist, assume no remote changes
		return false, nil
	}

	count := strings.TrimSpace(string(output))
	return count != "0", nil
}

// gitPull pulls remote changes with fast-forward only
func (p *GitPlugin) gitPull() error {
	cmd := exec.Command("git", "pull", "--ff-only", "origin")
	cmd.Dir = p.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %s: %w", string(output), err)
	}
	return nil
}

// pullWithRebase attempts to pull with rebase
func (p *GitPlugin) pullWithRebase() error {
	// First, try to stash any uncommitted changes
	cmd := exec.Command("git", "stash", "push", "-m", "kilt: auto-stash before pull")
	cmd.Dir = p.repoPath
	_ = cmd.Run() // Ignore errors (might be nothing to stash)

	// Pull with rebase
	cmd = exec.Command("git", "pull", "--rebase", "origin")
	cmd.Dir = p.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Restore stash if pull failed
		cmd = exec.Command("git", "stash", "pop")
		cmd.Dir = p.repoPath
		_ = cmd.Run()
		return fmt.Errorf("git pull --rebase failed: %s: %w", string(output), err)
	}

	// Restore stash if it exists
	cmd = exec.Command("git", "stash", "pop")
	cmd.Dir = p.repoPath
	_ = cmd.Run() // Ignore errors (might be no stash)

	return nil
}

// commitAndPush commits local changes and pushes them
func (p *GitPlugin) commitAndPush() error {
	// Get list of changed files
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = p.repoPath
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get changed files: %w", err)
	}

	changedFiles := strings.TrimSpace(string(output))
	if changedFiles == "" {
		return nil // Nothing to commit
	}

	// Get hostname for commit message
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// Add all changes
	cmd = exec.Command("git", "add", "-A")
	cmd.Dir = p.repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %s: %w", string(output), err)
	}

	// Build commit message
	lines := strings.Split(changedFiles, "\n")
	fileList := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			// Extract filename (skip status prefix)
			parts := strings.Fields(line)
			if len(parts) > 1 {
				fileList = append(fileList, "- "+parts[len(parts)-1])
			}
		}
	}

	commitMsg := fmt.Sprintf("kilt: auto-sync from %s\n\nFiles changed:\n%s", hostname, strings.Join(fileList, "\n"))

	// Commit
	cmd = exec.Command("git", "commit", "-m", commitMsg)
	cmd.Dir = p.repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		// Check if there's actually something to commit
		if strings.Contains(string(output), "nothing to commit") {
			return nil // Nothing to commit, that's fine
		}
		return fmt.Errorf("git commit failed: %s: %w", string(output), err)
	}

	// Push
	cmd = exec.Command("git", "push", "origin")
	cmd.Dir = p.repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed: %s: %w", string(output), err)
	}

	if p.ctx.Logger != nil {
		p.ctx.Logger.Info("Successfully committed and pushed local changes")
	}

	return nil
}

// Rollback rolls back any changes (no-op for Git plugin)
func (p *GitPlugin) Rollback(ctx *plugin.ExecutionContext) error {
	// Git plugin doesn't modify files directly, so rollback is a no-op
	return nil
}

