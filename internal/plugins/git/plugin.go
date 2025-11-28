// Package git provides the Git plugin for Kilt.
// It handles bidirectional Git synchronization for the dotfiles repository and manages extra repositories.
package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/unravelling/kilt/internal/core"
	"github.com/unravelling/kilt/internal/plugin"
)

// GitPlugin handles bidirectional Git synchronization for the dotfiles repository
// and manages extra repositories
type GitPlugin struct {
	ctx            *plugin.PluginContext
	dotfilesPath   string
	repoPath       string
	autoPull       bool
	sshKey         string
	gitToken       string
	clonedRepos    []string // Track cloned repos for rollback
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
	return "Manage bare Git repository and clone extra repositories"
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
	p.autoPull = true
	p.sshKey = ""
	p.gitToken = ""
	p.clonedRepos = make([]string, 0)

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

	// Get plugin-specific configuration
	config := plugin.GetPluginConfig(ctx.Config, p.Name())
	if config != nil {
		// Parse auto_pull flag
		if autoPull, ok := config["auto_pull"].(bool); ok {
			p.autoPull = autoPull
		}

		// Parse SSH key path
		if sshKey, ok := config["ssh_key"].(string); ok {
			expanded, err := core.ExpandPath(sshKey)
			if err != nil {
				return fmt.Errorf("failed to expand ssh_key path: %w", err)
			}
			p.sshKey = expanded
		}

		// Parse Git token (for HTTPS authentication)
		if token, ok := config["git_token"].(string); ok {
			p.gitToken = token
		}
	}

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
	// Reset cloned repos for this execution
	p.clonedRepos = make([]string, 0)

	// 1. Handle main dotfiles repository
	if p.autoPull {
		if err := p.syncMainRepository(ctx); err != nil {
			return fmt.Errorf("failed to sync main repository: %w", err)
		}
	}

	// 2. Handle extra repositories
	cfg, ok := p.ctx.Config.(*core.Config)
	if !ok {
		return fmt.Errorf("invalid config type")
	}

	for _, repo := range cfg.ExtraRepos {
		if err := p.handleExtraRepository(ctx, repo); err != nil {
			return fmt.Errorf("failed to handle extra repository %s: %w", repo.URL, err)
		}
	}

	return nil
}

// syncMainRepository syncs the main dotfiles repository
func (p *GitPlugin) syncMainRepository(ctx *plugin.ExecutionContext) error {
	// 1. Fetch remote
	if err := p.gitFetch(p.repoPath); err != nil {
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

	// Record changes
	ctx.AddChange(plugin.Change{
		Type:        "git_sync",
		Files:       []string{p.repoPath},
		Description: fmt.Sprintf("Synchronized dotfiles repository: %s", p.repoPath),
	})

	return nil
}

// handleExtraRepository handles an extra repository from config
func (p *GitPlugin) handleExtraRepository(ctx *plugin.ExecutionContext, repo core.Repository) error {
	// Expand repository path
	repoPath, err := core.ExpandPath(repo.Path)
	if err != nil {
		return fmt.Errorf("failed to expand repository path: %w", err)
	}

	// Check if repository already exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		// Repository doesn't exist - clone it
		if p.ctx.DryRun {
			ctx.AddChange(plugin.Change{
				Type:        "git_clone",
				Files:       []string{repoPath},
				Description: fmt.Sprintf("Would clone repository: %s to %s", repo.URL, repoPath),
			})
			return nil
		}

		if err := p.cloneRepository(repo, repoPath); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}

		p.clonedRepos = append(p.clonedRepos, repoPath)

		ctx.AddChange(plugin.Change{
			Type:        "git_clone",
			Files:       []string{repoPath},
			Description: fmt.Sprintf("Cloned repository: %s to %s", repo.URL, repoPath),
		})

		if p.ctx.Logger != nil {
			p.ctx.Logger.Info("Cloned extra repository", "url", repo.URL, "path", repoPath)
		}
	} else {
		// Repository exists - update it
		if p.ctx.DryRun {
			ctx.AddChange(plugin.Change{
				Type:        "git_pull",
				Files:       []string{repoPath},
				Description: fmt.Sprintf("Would update repository: %s", repoPath),
			})
			return nil
		}

		if err := p.updateRepository(repo, repoPath); err != nil {
			return fmt.Errorf("failed to update repository: %w", err)
		}

		ctx.AddChange(plugin.Change{
			Type:        "git_pull",
			Files:       []string{repoPath},
			Description: fmt.Sprintf("Updated repository: %s", repoPath),
		})
	}

	return nil
}

// gitFetch fetches remote changes
func (p *GitPlugin) gitFetch(repoPath string) error {
	// Check if remote exists
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		// No remote configured - that's okay for local repos
		if p.ctx.Logger != nil {
			p.ctx.Logger.Debug("No remote 'origin' configured, skipping fetch")
		}
		return nil
	}

	cmd = exec.Command("git", "fetch", "origin")
	cmd.Dir = repoPath
	p.setupGitAuth(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git fetch failed: %s: %w", string(output), err)
	}
	return nil
}

// cloneRepository clones a repository
func (p *GitPlugin) cloneRepository(repo core.Repository, targetPath string) error {
	// Create parent directory if it doesn't exist
	parentDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Build clone command
	args := []string{"clone"}

	// Add branch if specified
	if repo.Branch != "" {
		args = append(args, "--branch", repo.Branch)
	}

	// Add sparse checkout if requested
	if repo.Sparse {
		args = append(args, "--filter=blob:none", "--sparse")
	}

	// Add repository URL and target path
	repoURL := p.authenticateURL(repo.URL)
	args = append(args, repoURL, targetPath)

	// Set timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = filepath.Dir(targetPath)
	p.setupGitAuth(cmd)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("git clone timed out after 5 minutes")
		}
		return fmt.Errorf("git clone failed: %s: %w", string(output), err)
	}

	// If sparse checkout was requested, initialize it
	if repo.Sparse {
		cmd = exec.Command("git", "sparse-checkout", "init", "--cone")
		cmd.Dir = targetPath
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git sparse-checkout init failed: %s: %w", string(output), err)
		}
	}

	return nil
}

// updateRepository updates an existing repository
func (p *GitPlugin) updateRepository(repo core.Repository, repoPath string) error {
	// Fetch latest changes
	if err := p.gitFetch(repoPath); err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	// Get current branch or use specified branch
	branch := repo.Branch
	if branch == "" {
		cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
		cmd.Dir = repoPath
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}
		branch = strings.TrimSpace(string(output))
	}

	// Pull latest changes
	cmd := exec.Command("git", "pull", "--ff-only", "origin", branch)
	cmd.Dir = repoPath
	p.setupGitAuth(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If fast-forward fails, try regular pull
		cmd = exec.Command("git", "pull", "origin", branch)
		cmd.Dir = repoPath
		p.setupGitAuth(cmd)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git pull failed: %s: %w", string(output), err)
		}
	}

	return nil
}

// authenticateURL adds authentication to a Git URL if needed
func (p *GitPlugin) authenticateURL(url string) string {
	// If we have a token and URL is HTTPS, inject token
	if p.gitToken != "" && strings.HasPrefix(url, "https://") {
		// Insert token into URL: https://token@github.com/user/repo.git
		url = strings.Replace(url, "https://", fmt.Sprintf("https://%s@", p.gitToken), 1)
	}
	return url
}

// setupGitAuth configures Git command with authentication
func (p *GitPlugin) setupGitAuth(cmd *exec.Cmd) {
	// Set SSH key if provided
	if p.sshKey != "" {
		// Set GIT_SSH_COMMAND to use specific key
		sshCmd := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", p.sshKey)
		cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCmd)
	}

	// Set Git token in environment if provided (for HTTPS)
	if p.gitToken != "" {
		// Extract host from URL if possible, or use generic
		cmd.Env = append(cmd.Env, "GIT_ASKPASS=echo", "GIT_TERMINAL_PROMPT=0")
	}
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
	p.setupGitAuth(cmd)
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
	p.setupGitAuth(cmd)
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
	p.setupGitAuth(cmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed: %s: %w", string(output), err)
	}

	if p.ctx.Logger != nil {
		p.ctx.Logger.Info("Successfully committed and pushed local changes")
	}

	return nil
}

// Rollback rolls back any changes
func (p *GitPlugin) Rollback(ctx *plugin.ExecutionContext) error {
	// Remove cloned repositories
	for _, repoPath := range p.clonedRepos {
		if p.ctx.Logger != nil {
			p.ctx.Logger.Info("Rolling back cloned repository", "path", repoPath)
		}

		if err := os.RemoveAll(repoPath); err != nil {
			if p.ctx.Logger != nil {
				p.ctx.Logger.Warn("Failed to remove repository during rollback", "path", repoPath, "error", err)
			}
		}

		ctx.AddChange(plugin.Change{
			Type:        "git_rollback",
			Files:       []string{repoPath},
			Description: fmt.Sprintf("Removed cloned repository: %s", repoPath),
		})
	}

	// Clear cloned repos
	p.clonedRepos = make([]string, 0)

	return nil
}

