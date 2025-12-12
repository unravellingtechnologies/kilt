// Package git provides the Git plugin for Kilt.
// It handles bidirectional Git synchronisation for the dotfiles repository and manages extra repositories.
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

// Plugin handles bidirectional Git synchronisation for the dotfiles repository
// and manages extra repositories
type Plugin struct {
	ctx           *plugin.Context
	dotfilesPath  string
	repoPath      string
	autoPull      bool
	sshKey        string
	gitToken      string
	clonedRepos   []string // Track cloned repos for rollback
	askpassScript string   // Path to temporary GIT_ASKPASS script (for token auth)
}

// init registers the git plugin
func init() {
	if err := plugin.RegisterPlugin(&Plugin{}); err != nil {
		panic(fmt.Errorf("failed to register git plugin: %w", err))
	}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "git"
}

// Version returns the plugin version
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Description returns the plugin description
func (p *Plugin) Description() string {
	return "Manage bare Git repository and clone extra repositories"
}

// Dependencies returns plugin dependencies
func (p *Plugin) Dependencies() []string {
	return []string{}
}

// Phase returns the execution phase
func (p *Plugin) Phase() plugin.ExecutionPhase {
	return plugin.PhasePreSync
}

// Initialise initialises the plugin with context
func (p *Plugin) Initialise(ctx *plugin.Context) error {
	p.ctx = ctx
	p.autoPull = true
	p.sshKey = ""
	p.gitToken = ""
	p.clonedRepos = make([]string, 0)
	p.askpassScript = ""

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
		// WARNING: Tokens are stored in memory and used via transient GIT_ASKPASS scripts.
		// Tokens are never logged, persisted in URLs, or written to repository configuration.
		// The temporary askpass script is created per-command and cleaned up immediately after use.
		if token, ok := config["git_token"].(string); ok {
			p.gitToken = token
		}
	}

	return nil
}

// Validate validates the plugin configuration
func (p *Plugin) Validate() error {
	if p.ctx == nil {
		return fmt.Errorf("plugin context not initialised")
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
func (p *Plugin) Execute(ctx *plugin.ExecutionContext) error {
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
			// For extra repos, clone failures are non-fatal - log and continue
			// This allows the tool to work even if optional extra repos are unavailable
			if p.ctx.Logger != nil {
				p.ctx.Logger.Warn("Failed to handle extra repository, skipping",
					"url", repo.URL,
					"path", repo.Path,
					"error", err)
			}
			// Continue with other repos instead of failing the entire sync
			continue
		}
	}

	return nil
}

// syncMainRepository syncs the main dotfiles repository
func (p *Plugin) syncMainRepository(ctx *plugin.ExecutionContext) error {
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
		Description: fmt.Sprintf("Synchronised dotfiles repository: %s", p.repoPath),
	})

	return nil
}

// handleExtraRepository handles an extra repository from config
func (p *Plugin) handleExtraRepository(ctx *plugin.ExecutionContext, repo core.Repository) error {
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
func (p *Plugin) gitFetch(repoPath string) error {
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
	if err := p.setupGitAuth(cmd); err != nil {
		return fmt.Errorf("failed to setup git authentication: %w", err)
	}
	defer p.cleanupAskpassScript()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git fetch failed: %s: %w", string(output), err)
	}
	return nil
}

// cloneRepository clones a repository
func (p *Plugin) cloneRepository(repo core.Repository, targetPath string) error {
	// Create parent directory if it doesn't exist
	parentDir := filepath.Dir(targetPath)
	//nolint:gosec // G301: git repo parent directory needs 0755 for user access
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
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

	// Add repository URL and target path (do not modify URL with credentials)
	args = append(args, repo.URL, targetPath)

	// Set timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = filepath.Dir(targetPath)
	if err := p.setupGitAuth(cmd); err != nil {
		return fmt.Errorf("failed to setup git authentication: %w", err)
	}
	defer p.cleanupAskpassScript()

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("git clone timed out after 5 minutes")
		}
		return fmt.Errorf("git clone failed: %s: %w", string(output), err)
	}

	// If sparse checkout was requested, initialise it
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
func (p *Plugin) updateRepository(repo core.Repository, repoPath string) error {
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
	//nolint:gosec // G204: branch name comes from git output (validated), not user input
	cmd := exec.Command("git", "pull", "--ff-only", "origin", branch)
	cmd.Dir = repoPath
	if err := p.setupGitAuth(cmd); err != nil {
		return fmt.Errorf("failed to setup git authentication: %w", err)
	}
	defer p.cleanupAskpassScript()
	if _, err := cmd.CombinedOutput(); err != nil {
		// If fast-forward fails, try regular pull
		//nolint:gosec // G204: branch name comes from git output (validated), not user input
		cmd = exec.Command("git", "pull", "origin", branch)
		cmd.Dir = repoPath
		if err := p.setupGitAuth(cmd); err != nil {
			return fmt.Errorf("failed to setup git authentication: %w", err)
		}
		defer p.cleanupAskpassScript()
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git pull failed: %s: %w", string(output), err)
		}
	}

	return nil
}

// createAskpassScript creates a temporary GIT_ASKPASS script that outputs the token
// This provides transient credential authentication without persisting tokens in URLs or config
func (p *Plugin) createAskpassScript() (string, error) {
	if p.gitToken == "" {
		return "", nil
	}

	// Create temporary script file
	tmpFile, err := os.CreateTemp("", "kilt-git-askpass-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary askpass script: %w", err)
	}
	scriptPath := tmpFile.Name()

	// Write script that outputs the token
	// The script will be called by Git when it needs credentials
	scriptContent := fmt.Sprintf(`#!/bin/sh
echo "%s"
`, p.gitToken)

	if _, err := tmpFile.WriteString(scriptContent); err != nil {
		_ = tmpFile.Close()       //nolint:errcheck // Cleanup in error path
		_ = os.Remove(scriptPath) //nolint:errcheck // Cleanup in error path
		return "", fmt.Errorf("failed to write askpass script: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(scriptPath) //nolint:errcheck // Cleanup in error path
		return "", fmt.Errorf("failed to close askpass script: %w", err)
	}

	// Make script executable
	//nolint:gosec // G302: executable script requires 0700 permissions
	if err := os.Chmod(scriptPath, 0o700); err != nil {
		_ = os.Remove(scriptPath) //nolint:errcheck // Cleanup in error path
		return "", fmt.Errorf("failed to make askpass script executable: %w", err)
	}

	return scriptPath, nil
}

// cleanupAskpassScript removes the temporary GIT_ASKPASS script
func (p *Plugin) cleanupAskpassScript() {
	if p.askpassScript != "" {
		_ = os.Remove(p.askpassScript) //nolint:errcheck // Cleanup operation - errors are non-critical
		p.askpassScript = ""
	}
}

// setupGitAuth configures Git command with authentication using transient credentials
// For HTTPS URLs with tokens, creates a temporary GIT_ASKPASS script that provides
// the token only for the duration of the Git operation, preventing token persistence
// in repository configuration or process arguments.
func (p *Plugin) setupGitAuth(cmd *exec.Cmd) error {
	// Initialise base environment: use existing cmd.Env if non-nil, else os.Environ()
	baseEnv := cmd.Env
	if baseEnv == nil {
		baseEnv = os.Environ()
	}

	// Build list of env vars we'll be setting (to filter out existing values)
	overrideVars := make(map[string]bool)
	if p.sshKey != "" {
		overrideVars["GIT_SSH_COMMAND"] = true
	}
	if p.gitToken != "" {
		overrideVars["GIT_ASKPASS"] = true
		overrideVars["GIT_TERMINAL_PROMPT"] = true
	}

	// Copy base environment, filtering out vars we'll override
	env := make([]string, 0, len(baseEnv))
	for _, e := range baseEnv {
		varName := strings.SplitN(e, "=", 2)[0]
		if !overrideVars[varName] {
			env = append(env, e)
		}
	}

	// Set SSH key if provided
	if p.sshKey != "" {
		// Set GIT_SSH_COMMAND to use specific key
		sshCmd := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", p.sshKey)
		env = append(env, "GIT_SSH_COMMAND="+sshCmd)
	}

	// Set up transient credential mechanism for HTTPS token authentication
	// This avoids persisting tokens in URLs or repository configuration
	if p.gitToken != "" {
		// Create temporary GIT_ASKPASS script if not already created
		if p.askpassScript == "" {
			scriptPath, err := p.createAskpassScript()
			if err != nil {
				return fmt.Errorf("failed to create askpass script: %w", err)
			}
			p.askpassScript = scriptPath
		}

		// Configure Git to use the askpass script for credentials
		env = append(env, "GIT_ASKPASS="+p.askpassScript)
		env = append(env, "GIT_TERMINAL_PROMPT=0")
	}

	// Assign the final environment back to cmd
	cmd.Env = env
	return nil
}

// hasUncommittedChanges checks if there are uncommitted local changes
func (p *Plugin) hasUncommittedChanges() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = p.repoPath
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}

	return len(strings.TrimSpace(string(output))) > 0, nil
}

// hasRemoteChanges checks if there are remote changes
func (p *Plugin) hasRemoteChanges() (bool, error) {
	// Get current branch
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = p.repoPath
	branchOutput, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to get current branch: %w", err)
	}
	branch := strings.TrimSpace(string(branchOutput))

	// Compare HEAD with origin/branch
	//nolint:gosec // G204: branch name comes from git output (validated), not user input
	cmd = exec.Command("git", "rev-list", "--count", fmt.Sprintf("HEAD..origin/%s", branch))
	cmd.Dir = p.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if error indicates no remote branch (not a real error)
		errorText := strings.ToLower(string(output))
		isNoRemoteBranch := strings.Contains(errorText, "unknown revision or path not in the working tree") ||
			strings.Contains(errorText, "couldn't find remote ref") ||
			strings.Contains(errorText, "ambiguous argument") ||
			strings.Contains(errorText, "fatal: ambiguous argument")

		if isNoRemoteBranch {
			// No remote branch exists - this is expected, not an error
			return false, nil
		}

		// Other errors (network, auth, etc.) should be surfaced
		return false, fmt.Errorf("failed to check remote changes: %s: %w", string(output), err)
	}

	count := strings.TrimSpace(string(output))
	return count != "0", nil
}

// gitPull pulls remote changes with fast-forward only
func (p *Plugin) gitPull() error {
	cmd := exec.Command("git", "pull", "--ff-only", "origin")
	cmd.Dir = p.repoPath
	if err := p.setupGitAuth(cmd); err != nil {
		return fmt.Errorf("failed to setup git authentication: %w", err)
	}
	defer p.cleanupAskpassScript()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %s: %w", string(output), err)
	}
	return nil
}

// pullWithRebase attempts to pull with rebase
func (p *Plugin) pullWithRebase() error {
	// First, try to stash any uncommitted changes
	cmd := exec.Command("git", "stash", "push", "-m", "kilt: auto-stash before pull")
	cmd.Dir = p.repoPath
	_ = cmd.Run() // Ignore errors (might be nothing to stash)

	// Pull with rebase
	cmd = exec.Command("git", "pull", "--rebase", "origin")
	cmd.Dir = p.repoPath
	if err := p.setupGitAuth(cmd); err != nil {
		return fmt.Errorf("failed to setup git authentication: %w", err)
	}
	defer p.cleanupAskpassScript()
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Restore stash if pull failed
		cmd = exec.Command("git", "stash", "pop")
		cmd.Dir = p.repoPath
		stashOutput, stashErr := cmd.CombinedOutput()
		if stashErr != nil {
			// Log the stash pop error but return the original pull error
			if p.ctx.Logger != nil {
				p.ctx.Logger.Warn("Failed to restore stashed changes after pull failure",
					"error", stashErr,
					"output", string(stashOutput),
					"hint", "run 'git stash list' to check your stashes")
			}
		}
		return fmt.Errorf("git pull --rebase failed: %s: %w", string(output), err)
	}

	// Restore stash if it exists
	cmd = exec.Command("git", "stash", "pop")
	cmd.Dir = p.repoPath
	stashOutput, stashErr := cmd.CombinedOutput()
	if stashErr != nil {
		// Log the stash pop failure and return an error so users are notified
		if p.ctx.Logger != nil {
			p.ctx.Logger.Error("Failed to restore stashed changes after successful pull",
				"error", stashErr,
				"output", string(stashOutput),
				"hint", "run 'git stash list' to check your stashes")
		}
		return fmt.Errorf("failed to restore stashed changes: %s: %w", string(stashOutput), stashErr)
	}

	return nil
}

// commitAndPush commits local changes and pushes them
func (p *Plugin) commitAndPush() error {
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
	//nolint:gosec // G204: commitMsg is constructed from safe inputs (hostname, fileList from validated paths)
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
	if err := p.setupGitAuth(cmd); err != nil {
		return fmt.Errorf("failed to setup git authentication: %w", err)
	}
	defer p.cleanupAskpassScript()
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed: %s: %w", string(output), err)
	}

	if p.ctx.Logger != nil {
		p.ctx.Logger.Info("Successfully committed and pushed local changes")
	}

	return nil
}

// Rollback rolls back any changes
func (p *Plugin) Rollback(ctx *plugin.ExecutionContext) error {
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

	// Clean up askpass script if it exists
	p.cleanupAskpassScript()

	return nil
}
