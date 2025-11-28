// Package alternates provides the alternates plugin for Kilt.
// It handles automatic file selection based on OS, hostname, and architecture using alternate file patterns.
package alternates

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/unravelling/kilt/internal/core"
	"github.com/unravelling/kilt/internal/plugin"
)

// AlternatesPlugin handles automatic file selection based on OS, hostname, and architecture
type AlternatesPlugin struct {
	ctx            *plugin.PluginContext
	customPatterns []string
	resolutions    map[string]string // original source -> resolved source
}

func init() {
	plugin.RegisterPlugin(&AlternatesPlugin{})
}

// Name returns the plugin name
func (p *AlternatesPlugin) Name() string {
	return "alternates"
}

// Version returns the plugin version
func (p *AlternatesPlugin) Version() string {
	return "1.0.0"
}

// Description returns the plugin description
func (p *AlternatesPlugin) Description() string {
	return "Automatic file selection based on OS, hostname, architecture"
}

// Dependencies returns plugin dependencies
func (p *AlternatesPlugin) Dependencies() []string {
	return []string{}
}

// Phase returns the execution phase
func (p *AlternatesPlugin) Phase() plugin.ExecutionPhase {
	return plugin.PhasePreSync
}

// Initialize initializes the plugin with context
func (p *AlternatesPlugin) Initialize(ctx *plugin.PluginContext) error {
	p.ctx = ctx
	p.resolutions = make(map[string]string)

	// Get plugin-specific configuration
	config := plugin.GetPluginConfig(ctx.Config, p.Name())
	if config != nil {
		if patterns, ok := config["patterns"].([]interface{}); ok {
			p.customPatterns = make([]string, 0, len(patterns))
			for _, pattern := range patterns {
				if str, ok := pattern.(string); ok {
					p.customPatterns = append(p.customPatterns, str)
				}
			}
		}
	}

	return nil
}

// Validate validates the plugin configuration
func (p *AlternatesPlugin) Validate() error {
	if p.ctx == nil {
		return fmt.Errorf("plugin context not initialized")
	}
	return nil
}

// Execute executes the plugin logic
func (p *AlternatesPlugin) Execute(ctx *plugin.ExecutionContext) error {
	cfg, ok := p.ctx.Config.(*core.Config)
	if !ok {
		return fmt.Errorf("invalid config type")
	}

	// Get system information
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %w", err)
	}

	osName := runtime.GOOS
	arch := runtime.GOARCH

	// Resolve alternates for each dotfile entry with explicit Source
	for i := range cfg.Dotfiles {
		// Skip directory-mode entries (only process explicit source mappings)
		if cfg.Dotfiles[i].Source == "" {
			continue
		}

		originalSource := cfg.Dotfiles[i].Source
		resolvedSource, err := p.resolveAlternate(originalSource, hostname, osName, arch)
		if err != nil {
			if p.ctx.Logger != nil {
				p.ctx.Logger.Warn("Failed to resolve alternate", "source", originalSource, "error", err)
			}
			// Continue with original source if resolution fails
			continue
		}

		if resolvedSource != originalSource {
			// Update the source path
			cfg.Dotfiles[i].Source = resolvedSource
			p.resolutions[originalSource] = resolvedSource

			// Log the resolution
			if p.ctx.Logger != nil {
				p.ctx.Logger.Info("Resolved alternate", "original", originalSource, "resolved", resolvedSource)
			}

			// Record change
			ctx.AddChange(plugin.Change{
				Type:        "alternate_resolved",
				Files:       []string{originalSource, resolvedSource},
				Description: fmt.Sprintf("Resolved alternate: %s -> %s", originalSource, resolvedSource),
			})
		}
	}

	return nil
}

// resolveAlternate resolves the best alternate file for a given source
func (p *AlternatesPlugin) resolveAlternate(source, hostname, osName, arch string) (string, error) {
	// Get directory and base filename
	dir := filepath.Dir(source)
	base := filepath.Base(source)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)

	// Build search directory (relative to workDir)
	searchDir := filepath.Join(p.ctx.WorkDir, dir)

	// Get all candidate files
	candidates, err := p.findCandidates(searchDir, nameWithoutExt, ext)
	if err != nil {
		// If directory doesn't exist or we can't read it, return original
		return source, nil
	}

	if len(candidates) == 0 {
		// No alternates found, return original
		return source, nil
	}

	// Score and select best candidate
	bestCandidate := p.selectBestCandidate(candidates, source, hostname, osName, arch)
	if bestCandidate == "" {
		// No match found, return original
		return source, nil
	}

	// Return relative path from workDir
	relPath, err := filepath.Rel(p.ctx.WorkDir, bestCandidate)
	if err != nil {
		return source, fmt.Errorf("failed to get relative path: %w", err)
	}

	return relPath, nil
}

// findCandidates finds all candidate alternate files
func (p *AlternatesPlugin) findCandidates(searchDir, nameWithoutExt, ext string) ([]string, error) {
	candidates := make([]string, 0)

	// Check if search directory exists
	if _, err := os.Stat(searchDir); os.IsNotExist(err) {
		return candidates, nil
	}

	// Read directory
	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Look for files matching alternate patterns
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if !strings.HasPrefix(filename, nameWithoutExt) {
			continue
		}

		// Check if it matches alternate patterns
		if p.matchesAlternatePattern(filename, nameWithoutExt, ext) {
			candidatePath := filepath.Join(searchDir, filename)
			candidates = append(candidates, candidatePath)
		}
	}

	return candidates, nil
}

// matchesAlternatePattern checks if a filename matches alternate patterns
func (p *AlternatesPlugin) matchesAlternatePattern(filename, nameWithoutExt, ext string) bool {
	// Remove extension for matching
	nameWithoutExtAndExt := strings.TrimSuffix(filename, ext)

	// Must start with base name
	if !strings.HasPrefix(nameWithoutExtAndExt, nameWithoutExt) {
		return false
	}

	// Get the suffix (everything after base name)
	suffix := strings.TrimPrefix(nameWithoutExtAndExt, nameWithoutExt)
	if suffix == "" {
		// This is the base file itself, not an alternate
		return false
	}

	// Remove leading dot or underscore
	suffix = strings.TrimPrefix(suffix, ".")
	suffix = strings.TrimPrefix(suffix, "_")

	// Check built-in patterns
	if p.matchesBuiltInPattern(suffix) {
		return true
	}

	// Check custom patterns
	for _, pattern := range p.customPatterns {
		if p.matchesCustomPattern(suffix, pattern) {
			return true
		}
	}

	return false
}

// matchesBuiltInPattern checks if suffix matches built-in patterns
func (p *AlternatesPlugin) matchesBuiltInPattern(suffix string) bool {
	// OS patterns: mac, darwin, linux, windows
	osPatterns := []string{"mac", "darwin", "linux", "windows"}
	for _, pattern := range osPatterns {
		if suffix == pattern {
			return true
		}
	}

	// Architecture patterns: arm64, amd64, x86_64, i386
	archPatterns := []string{"arm64", "amd64", "x86_64", "i386"}
	for _, pattern := range archPatterns {
		if suffix == pattern {
			return true
		}
	}

	// Hostname pattern: hostname@tag (e.g., "work@laptop", "home@desktop")
	if strings.Contains(suffix, "@") {
		return true
	}

	return false
}

// matchesCustomPattern checks if suffix matches a custom pattern
func (p *AlternatesPlugin) matchesCustomPattern(suffix, pattern string) bool {
	// Simple pattern matching - can be extended
	// For now, support exact match and wildcard-like patterns
	if pattern == suffix {
		return true
	}

	// Support patterns like "##os.{darwin,linux}"
	if strings.HasPrefix(pattern, "##os.") {
		osList := strings.TrimPrefix(pattern, "##os.")
		osList = strings.Trim(osList, "{}")
		oses := strings.Split(osList, ",")
		for _, os := range oses {
			if strings.TrimSpace(os) == suffix {
				return true
			}
		}
	}

	// Support patterns like "##hostname.{work,personal}"
	if strings.HasPrefix(pattern, "##hostname.") {
		hostnameList := strings.TrimPrefix(pattern, "##hostname.")
		hostnameList = strings.Trim(hostnameList, "{}")
		hostnames := strings.Split(hostnameList, ",")
		for _, hn := range hostnames {
			if strings.TrimSpace(hn) == suffix {
				return true
			}
		}
	}

	return false
}

// selectBestCandidate selects the best candidate based on priority rules
func (p *AlternatesPlugin) selectBestCandidate(candidates []string, originalSource, hostname, osName, arch string) string {
	type candidateScore struct {
		path  string
		score int
	}

	// Extract base name from original source
	originalBase := filepath.Base(originalSource)
	originalExt := filepath.Ext(originalBase)
	originalNameWithoutExt := strings.TrimSuffix(originalBase, originalExt)

	scores := make([]candidateScore, 0, len(candidates))

	for _, candidate := range candidates {
		filename := filepath.Base(candidate)
		score := p.scoreCandidate(filename, originalNameWithoutExt, hostname, osName, arch)
		scores = append(scores, candidateScore{
			path:  candidate,
			score: score,
		})
	}

	// Find highest score
	bestScore := -1
	bestPath := ""
	for _, cs := range scores {
		if cs.score > bestScore {
			bestScore = cs.score
			bestPath = cs.path
		}
	}

	// Only return if we have a positive match (score > 0)
	if bestScore > 0 {
		return bestPath
	}

	return ""
}

// scoreCandidate scores a candidate file based on how well it matches
// Higher score = better match
func (p *AlternatesPlugin) scoreCandidate(filename, originalBaseName, hostname, osName, arch string) int {
	score := 0
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)

	// Extract suffix (everything after original base name)
	if !strings.HasPrefix(nameWithoutExt, originalBaseName) {
		return 0
	}

	suffix := strings.TrimPrefix(nameWithoutExt, originalBaseName)
	// Remove leading separators
	suffix = strings.TrimPrefix(suffix, ".")
	suffix = strings.TrimPrefix(suffix, "_")

	if suffix == "" {
		// This is the base file itself, not an alternate
		return 0
	}

	// First check for hostname@tag pattern (highest priority, handles dots in hostname)
	if strings.Contains(suffix, "@") {
		// Split by @ to get hostname and tag
		parts := strings.SplitN(suffix, "@", 2)
		if len(parts) == 2 {
			hostnamePart := parts[0]
			// Remove leading/trailing dots and underscores from hostname part
			hostnamePart = strings.Trim(hostnamePart, "._")
			if hostnamePart == hostname {
				score += 100
				// Hostname match found, return early (highest priority)
				return score
			}
		}
	}

	// Split suffix by dots and underscores to get parts (for other patterns)
	parts := strings.FieldsFunc(suffix, func(r rune) bool {
		return r == '.' || r == '_'
	})

	// Score based on parts
	for _, part := range parts {
		partLower := strings.ToLower(part)

		// Skip if this part contains @ (already handled above)
		if strings.Contains(part, "@") {
			continue
		}

		// OS match (50 points)
		if partLower == osName || (partLower == "mac" && osName == "darwin") {
			score += 50
		}

		// Architecture match (30 points)
		if partLower == arch || (partLower == "x86_64" && arch == "amd64") {
			score += 30
		}

		// Custom pattern matches (20 points each)
		for _, pattern := range p.customPatterns {
			if p.matchesCustomPattern(part, pattern) {
				score += 20
			}
		}
	}

	return score
}

// Rollback rolls back alternate resolutions
func (p *AlternatesPlugin) Rollback(ctx *plugin.ExecutionContext) error {
	// Alternates plugin doesn't modify files, only config
	// So rollback is a no-op, but we clear resolutions
	p.resolutions = make(map[string]string)
	return nil
}

