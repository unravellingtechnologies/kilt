// Package onepassword provides the 1Password plugin for Kilt.
// It handles 1Password CLI integration for secret injection into templates, with caching support.
package onepassword

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/unravelling/kilt/internal/plugin"
)

// Plugin handles 1Password CLI integration for secret injection
type Plugin struct {
	ctx           *plugin.Context
	opPath        string
	account       string
	vault         string
	cacheEnabled  bool
	cacheTTL      time.Duration
	cache         map[string]cacheEntry
	cacheMu       sync.RWMutex
	authenticated atomic.Bool
}

// cacheEntry represents a cached secret with expiration
type cacheEntry struct {
	value     string
	expiresAt time.Time
}

// init registers the onepassword plugin
func init() {
	if err := plugin.RegisterPlugin(&Plugin{}); err != nil {
		panic(fmt.Errorf("failed to register onepassword plugin: %w", err))
	}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "onepassword"
}

// Version returns the plugin version
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Description returns the plugin description
func (p *Plugin) Description() string {
	return "1Password CLI integration for secret injection in templates"
}

// Dependencies returns plugin dependencies
func (p *Plugin) Dependencies() []string {
	return []string{}
}

// Phase returns the execution phase
func (p *Plugin) Phase() plugin.ExecutionPhase {
	// This plugin runs early to register template function before templates are rendered
	return plugin.PhasePreSync
}

// Initialise initialises the plugin with context
func (p *Plugin) Initialise(ctx *plugin.Context) error {
	p.ctx = ctx
	p.account = ""
	p.vault = ""
	p.cacheEnabled = true
	p.cacheTTL = 5 * time.Minute // Default 5 minute cache TTL
	p.cache = make(map[string]cacheEntry)
	p.authenticated.Store(false)

	// Get plugin-specific configuration
	config := plugin.GetPluginConfig(ctx.Config, p.Name())
	if config != nil {
		// Parse account
		if account, ok := config["account"].(string); ok {
			p.account = account
		}

		// Parse vault
		if vault, ok := config["vault"].(string); ok {
			p.vault = vault
		}

		// Parse cache_enabled flag
		if cacheEnabled, ok := config["cache_enabled"].(bool); ok {
			p.cacheEnabled = cacheEnabled
		}

		// Parse cache_ttl
		if cacheTTLStr, ok := config["cache_ttl"].(string); ok {
			duration, err := time.ParseDuration(cacheTTLStr)
			if err != nil {
				return fmt.Errorf("invalid cache_ttl: %s (must be a valid duration like '5m', '30s'): %w", cacheTTLStr, err)
			}
			p.cacheTTL = duration
		}
	}

	// Find 1Password CLI
	opPath, err := p.findOP()
	if err != nil {
		// 1Password CLI not found - plugin will still register but will return errors
		if p.ctx.Logger != nil {
			p.ctx.Logger.Warn("1Password CLI (op) not found, op template function will not work", "error", err)
		}
		p.opPath = ""
	} else {
		p.opPath = opPath
		// Check authentication status
		if err := p.checkAuthentication(); err != nil {
			if p.ctx.Logger != nil {
				p.ctx.Logger.Warn("1Password CLI not authenticated, op template function will not work", "error", err)
			}
		} else {
			p.authenticated.Store(true)
		}
	}

	// Register the op function with the template engine if available
	if ctx.Template != nil {
		ctx.Template.RegisterOPFunction(p.getSecret)
		if p.ctx.Logger != nil {
			p.ctx.Logger.Info("1Password template function registered")
		}
	} else if p.ctx.Logger != nil {
		p.ctx.Logger.Warn("Template engine not available, 1Password template function not registered")
	}

	return nil
}

// Validate validates the plugin configuration
func (p *Plugin) Validate() error {
	if p.ctx == nil {
		return fmt.Errorf("plugin context not initialised")
	}

	// If op CLI is not installed, that's okay - plugin will return errors when used
	if p.opPath == "" {
		return nil
	}

	// If not authenticated, that's okay - user will need to authenticate
	if !p.authenticated.Load() {
		return nil
	}

	return nil
}

// Execute executes the plugin logic
// For 1Password plugin, most work is done during Initialise (registering template function)
// Execute can be used to verify authentication or perform any runtime checks
func (p *Plugin) Execute(ctx *plugin.ExecutionContext) error {
	// Verify authentication status if op is available
	if p.opPath != "" && !p.authenticated.Load() {
		if err := p.checkAuthentication(); err != nil {
			if p.ctx.Logger != nil {
				p.ctx.Logger.Warn("1Password CLI authentication check failed", "error", err)
			}
			// Don't fail execution - template function will handle errors
		} else {
			p.authenticated.Store(true)
			if p.ctx.Logger != nil {
				p.ctx.Logger.Info("1Password CLI authenticated")
			}
		}
	}

	return nil
}

// Rollback rolls back plugin changes
// For 1Password plugin, there's nothing to rollback (no state changes)
func (p *Plugin) Rollback(ctx *plugin.ExecutionContext) error {
	// Clear cache on rollback
	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()
	p.cache = make(map[string]cacheEntry)

	if p.ctx.Logger != nil {
		p.ctx.Logger.Info("1Password plugin rolled back (cache cleared)")
	}

	return nil
}

// findOP finds the 1Password CLI installation
func (p *Plugin) findOP() (string, error) {
	// First, try to find op in PATH
	// LookPath already ensures the file is present and executable
	if opPath, err := exec.LookPath("op"); err == nil {
		return opPath, nil
	}

	// Common installation locations
	opPaths := []string{
		"/usr/local/bin/op",
		"/opt/homebrew/bin/op",
		filepath.Join(os.Getenv("HOME"), ".local/bin/op"),
	}

	// Try common locations
	for _, path := range opPaths {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		// Ensure file exists, is regular, and has execute permissions
		mode := info.Mode()
		if mode.IsRegular() && mode&0o111 != 0 {
			return path, nil
		}
	}

	return "", fmt.Errorf("1Password CLI (op) not found in PATH or common locations")
}

// checkAuthentication checks if 1Password CLI is authenticated
func (p *Plugin) checkAuthentication() error {
	if p.opPath == "" {
		return fmt.Errorf("1Password CLI not found")
	}

	// Run `op account list` to check authentication
	//nolint:gosec // G204: p.opPath is validated config value, command args are hardcoded
	cmd := exec.Command(p.opPath, "account", "list")
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("1Password CLI authentication check failed: %s: %w", string(output), err)
	}

	// If we get output, we're likely authenticated (or at least op is working)
	if len(output) == 0 {
		return fmt.Errorf("1Password CLI not authenticated (run 'op signin' to authenticate)")
	}

	return nil
}

// getSecret retrieves a secret from 1Password
// This is the function registered with the template engine
func (p *Plugin) getSecret(path string) (string, error) {
	if p.opPath == "" {
		return "", fmt.Errorf("1Password CLI (op) is not installed. Install it from https://1password.com/downloads/command-line/")
	}

	if err := p.ensureAuthenticated(); err != nil {
		return "", err
	}

	// Check cache first
	if cached, found := p.getCachedSecret(path); found {
		return cached, nil
	}

	// Build and run op command
	output, err := p.runOpCommand(path)
	if err != nil {
		return "", err
	}

	// Parse and cache the result
	secret := p.parseOpOutput(output)
	if p.cacheEnabled {
		p.cacheSecret(path, secret)
	}

	return secret, nil
}

// ensureAuthenticated checks and authenticates if needed
func (p *Plugin) ensureAuthenticated() error {
	if p.authenticated.Load() {
		return nil
	}

	// Try to check authentication again (might have been authenticated since init)
	if err := p.checkAuthentication(); err != nil {
		return fmt.Errorf("1Password CLI is not authenticated. Run 'op signin' to authenticate: %w", err)
	}

	p.authenticated.Store(true)
	return nil
}

// getCachedSecret retrieves a secret from cache if available and not expired
func (p *Plugin) getCachedSecret(path string) (string, bool) {
	if !p.cacheEnabled {
		return "", false
	}

	p.cacheMu.RLock()
	entry, found := p.cache[path]
	p.cacheMu.RUnlock()

	if !found {
		return "", false
	}

	if time.Now().Before(entry.expiresAt) {
		return entry.value, true
	}

	// Cache expired, remove it
	p.cacheMu.Lock()
	// Re-check: another goroutine may have refreshed or deleted the entry
	if entry, found := p.cache[path]; found && time.Now().After(entry.expiresAt) {
		delete(p.cache, path)
	}
	p.cacheMu.Unlock()

	return "", false
}

// buildOpArgs builds the command arguments for op read
func (p *Plugin) buildOpArgs(path string) []string {
	args := []string{"read", path}

	if p.account != "" {
		args = append(args, "--account", p.account)
	}

	if p.vault != "" {
		args = append(args, "--vault", p.vault)
	}

	return args
}

// runOpCommand executes the op command and returns the output
func (p *Plugin) runOpCommand(path string) ([]byte, error) {
	args := p.buildOpArgs(path)

	cmdCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	//nolint:gosec // G204: p.opPath is validated config value, args come from validated config
	cmd := exec.CommandContext(cmdCtx, p.opPath, args...)
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, p.handleOpError(path, err, cmdCtx, output)
	}

	return output, nil
}

// handleOpError formats error messages from op command failures
func (p *Plugin) handleOpError(path string, err error, cmdCtx context.Context, output []byte) error {
	if cmdCtx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("1Password secret lookup timed out after 30 seconds for path: %s", path)
	}

	errorMsg := string(output)
	if len(errorMsg) > 0 {
		return fmt.Errorf("failed to retrieve secret from 1Password for path '%s': %s\nHint: Ensure the secret path is correct and you have access to it. Run 'op signin' if authentication is required", path, errorMsg)
	}

	return fmt.Errorf("failed to retrieve secret from 1Password for path '%s': %w\nHint: Ensure the secret path is correct and you have access to it", path, err)
}

// parseOpOutput parses the output from op read command
func (p *Plugin) parseOpOutput(output []byte) string {
	// Try to parse as JSON first
	var jsonData map[string]interface{}
	if err := json.Unmarshal(output, &jsonData); err == nil {
		// It's JSON - extract the value field if it exists
		if value, ok := jsonData["value"].(string); ok {
			return value
		}
		// If no value field, return the whole JSON as string (for structured items)
		return string(output)
	}

	// Not JSON - return as plain text (trimmed)
	secret := string(output)
	if len(secret) > 0 && secret[len(secret)-1] == '\n' {
		secret = secret[:len(secret)-1]
	}

	return secret
}

// cacheSecret caches a secret with expiration
func (p *Plugin) cacheSecret(path, value string) {
	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()

	p.cache[path] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(p.cacheTTL),
	}
}
