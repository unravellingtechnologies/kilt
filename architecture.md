# Kilt — Architecture Documentation

**Version**: 1.0.0  
**Date**: November 25, 2025  
**Status**: Design Phase

---

## Table of Contents

1. [System Overview](#system-overview)
2. [Architecture Principles](#architecture-principles)
3. [Component Architecture](#component-architecture)
4. [Plugin System Design](#plugin-system-design)
5. [Data Flow](#data-flow)
6. [Directory Structure](#directory-structure)
7. [Configuration Schema](#configuration-schema)
8. [State Management](#state-management)
9. [Security Model](#security-model)
10. [Error Handling Strategy](#error-handling-strategy)
11. [Testing Architecture](#testing-architecture)
12. [Performance Considerations](#performance-considerations)

---

## System Overview

### Purpose
Kilt is a Git-first dotfiles manager that transforms a fresh macOS (or Linux) system into a personalized development environment using a single command. It emphasizes idempotency, safety, and extensibility through a plugin architecture.

### Core Design Goals
- **Simplicity**: One command to bootstrap everything
- **Safety**: Never destroy data without backup
- **Idempotency**: Run 100 times with same result
- **Extensibility**: Plugin architecture for new features
- **Offline-capable**: Works without network after initial setup
- **Fast**: Minimal runtime dependencies, compiled binary

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        CLI Layer                             │
│  (cobra framework - commands, flags, help)                   │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                   Core Engine                                │
│  - Config Parser                                             │
│  - State Manager                                             │
│  - Backup System                                             │
│  - Template Engine                                           │
│  - Orchestrator                                              │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                  Plugin System                               │
│  - Plugin Registry                                           │
│  - Plugin Loader                                             │
│  - Execution Pipeline                                        │
└────────────────────────┬────────────────────────────────────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
┌───────▼──────┐  ┌──────▼─────┐  ┌──────▼─────────┐
│Core Plugins  │  │Integration │  │Optional Plugins│
│- Files       │  │Plugins     │  │- Custom        │
│- Directories │  │- Git       │  │  User-defined  │
│- Run Once    │  │- Brew      │  │                │
│- On Change   │  │- 1Password │  │                │
│- Alternates  │  │            │  │                │
└──────────────┘  └────────────┘  └────────────────┘
```

---

## Architecture Principles

### 1. Separation of Concerns
- **CLI Layer**: User interaction, command parsing, output formatting
- **Core Engine**: Configuration, state, orchestration, non-plugin logic
- **Plugin Layer**: Feature implementations, external integrations

### 2. Plugin-First Design
- Core features implemented as plugins where possible
- Plugins are self-contained and independently testable
- Plugin dependencies explicitly declared
- No circular dependencies between plugins

### 3. Immutable Configuration
- Configuration loaded once at startup
- No runtime config modification
- Changes require re-execution

### 4. Fail-Safe Operations
- All destructive operations backed up first
- Atomic operations where possible
- Rollback capability on failure
- Clear error messages with recovery suggestions

### 5. Dependency Minimization
- Core functionality requires only: Go stdlib, Git, Bash
- Optional features (Brew, 1Password) gracefully degrade if unavailable
- No runtime interpreters (Ruby, Python, Node.js)

---

## Component Architecture

### CLI Layer (`cmd/kilt/`)

#### Main Commands
```go
kilt
├── init <repo-url>      // Initialize from Git repository
├── sync                 // Pull latest and apply changes
├── doctor               // Validate setup and dependencies
├── version              // Show version information
├── restore <backup-id>  // Restore files from a backup
├── backups              // Manage backups
│   └── list             // List all available backups
└── reset                // Clear state and start fresh
```

#### Global Flags
- `--dry-run`: Show what would happen without executing
- `--verbose`: Detailed logging output
- `--debug`: Extra debug information
- `--config <path>`: Custom config file location
- `--no-color`: Disable colored output
- `--force`: Skip confirmation prompts

#### Command Flow
```
User Command
    ↓
Cobra Command Handler
    ↓
Parse Flags & Arguments
    ↓
Initialize Core Engine
    ↓
Load Configuration
    ↓
Execute Command Logic
    ↓
Format & Display Output
```

---

### Core Engine (`internal/core/`)

#### Config Parser (`config.go`)
```go
type Config struct {
    DotfilesRepo   string          `yaml:"dotfiles_repo"`
    DotfilesPath   string          `yaml:"dotfiles_path"`
    DataFile       string          `yaml:"data_file"`
    TemplateEngine string          `yaml:"template_engine"`
    Dotfiles       []DotfileEntry  `yaml:"dotfiles"`
    ExtraRepos     []Repository    `yaml:"extra_repos"`
    Directories    []string        `yaml:"directories"`
    RunOnce        []string        `yaml:"run_once"`
    OnChange       []string        `yaml:"on_change"`
    Plugins        map[string]any  `yaml:"plugins"`
}

type DotfileEntry struct {
    Directory string `yaml:"directory,omitempty"` // Simple form: directory name
    Source    string `yaml:"source,omitempty"`    // Explicit source path
    Target    string `yaml:"target,omitempty"`    // Explicit target path
    Template  bool   `yaml:"template,omitempty"`
    Mode      string `yaml:"mode,omitempty"`      // file permissions
}

type Repository struct {
    URL    string `yaml:"url"`
    Path   string `yaml:"path"`
    Branch string `yaml:"branch,omitempty"`
    Sparse bool   `yaml:"sparse,omitempty"`
}
```

**Responsibilities**:
- Parse YAML configuration file
- Validate configuration schema
- Expand `~` and environment variables
- Merge multiple config sources (global, local)
- Provide config access to plugins

**Key Functions**:
- `LoadConfig(path string) (*Config, error)`
- `ValidateConfig(cfg *Config) error`
- `ExpandPaths(cfg *Config) error`
- `MergeConfigs(base, override *Config) *Config`

---

#### State Manager (`state.go`)
```go
type StateManager struct {
    stateDir string
    db       *StateDB
}

type StateDB struct {
    RunOnce   map[string]RunOnceRecord   `json:"run_once"`
    Files     map[string]FileRecord      `json:"files"`
    Plugins   map[string]PluginRecord    `json:"plugins"`
    LastSync  time.Time                  `json:"last_sync"`
}

type RunOnceRecord struct {
    TaskID      string    `json:"task_id"`
    ExecutedAt  time.Time `json:"executed_at"`
    ExitCode    int       `json:"exit_code"`
    Output      string    `json:"output"`
}

type FileRecord struct {
    SourcePath string    `json:"source_path"`
    TargetPath string    `json:"target_path"`
    Checksum   string    `json:"checksum"`
    UpdatedAt  time.Time `json:"updated_at"`
}
```

**Responsibilities**:
- Track execution state for idempotency
- Store checksums for change detection
- Record run-once task completion
- Maintain plugin execution history
- File-based locking for concurrent safety

**Key Functions**:
- `IsTaskCompleted(taskID string) bool`
- `MarkTaskCompleted(taskID string, record RunOnceRecord) error`
- `GetFileChecksum(path string) (string, error)`
- `HasFileChanged(path string) (bool, error)`
- `AcquireLock() error`
- `ReleaseLock() error`

**Storage**:
- Location: `~/.kilt/state/state.json`
- Format: JSON for human readability
- Atomic writes with temp file + rename
- Lock file: `~/.kilt/state/.lock`

---

#### Backup System (`backup.go`)
```go
type BackupManager struct {
    backupDir string
}

type Backup struct {
    Timestamp time.Time
    Files     []BackedUpFile
    Size      int64
}

type BackedUpFile struct {
    OriginalPath string
    BackupPath   string
    Size         int64
    Mode         os.FileMode
}
```

**Responsibilities**:
- Create timestamped backups before file modifications
- Maintain directory structure in backups
- Implement retention policy
- Provide restore functionality

**Key Functions**:
- `CreateBackup(files []string) (*Backup, error)`
- `RestoreBackup(timestamp time.Time) error`
- `ListBackups() ([]Backup, error)`
- `CleanOldBackups(keepLast int) error`

**Backup Strategy**:
- Directory: `~/.kilt/backup/YYYYMMDD-HHMMSS/`
- Only backup files that will be modified
- Preserve file permissions and timestamps
- Incremental: skip if file unchanged since last backup
- Default retention: keep last 5 backups

---

#### Template Engine (`template.go`)
```go
type TemplateEngine struct {
    funcMap template.FuncMap
    data    map[string]interface{}
}

type TemplateData struct {
    Hostname string
    OS       string
    Arch     string
    User     string
    Home     string
    Custom   map[string]interface{}
}
```

**Responsibilities**:
- Render Go templates with built-in variables
- Load custom data from `data.yaml`
- Integrate with 1Password plugin for secrets
- Handle template errors gracefully

**Built-in Functions**:
- `{{ .hostname }}` - System hostname
- `{{ .os }}` - Operating system (darwin, linux)
- `{{ .arch }}` - Architecture (amd64, arm64)
- `{{ .user }}` - Current username
- `{{ .home }}` - Home directory path
- `{{ env "VAR" }}` - Environment variable
- `{{ op "path/to/secret" }}` - 1Password secret (via plugin)

**Key Functions**:
- `Render(templatePath string, data TemplateData) (string, error)`
- `RenderString(template string, data TemplateData) (string, error)`
- `LoadCustomData(dataPath string) error`
- `RegisterFunction(name string, fn interface{})`

---

#### Orchestrator (`engine.go`)
```go
type Engine struct {
    config        *Config
    state         *StateManager
    backup        *BackupManager
    template      *TemplateEngine
    pluginManager *PluginManager
    dryRun        bool
    logger        *Logger
}

type ExecutionPlan struct {
    Phases []ExecutionPhase
}

type ExecutionPhase struct {
    Name    string
    Plugins []Plugin
}
```

**Responsibilities**:
- Coordinate all components
- Build execution plan based on config
- Execute plugins in correct order
- Handle errors and rollback
- Report progress and results

**Execution Flow**:
```
1. Pre-flight Checks
   - Verify dependencies (git, config file)
   - Check file system permissions
   - Validate configuration

2. State Loading
   - Load existing state
   - Acquire lock

3. Backup Creation
   - Identify files to modify
   - Create backup if needed

4. Plugin Execution (ordered phases)
   Phase 1: Pre-sync (Git pull, etc.)
   Phase 2: Core operations (Files, Directories)
   Phase 3: Post-sync (Run Once, On Change)
   Phase 4: Integrations (Brew, etc.)

5. State Update
   - Record changes
   - Update checksums
   - Mark tasks complete

6. Cleanup
   - Release lock
   - Report summary

7. Error Handling
   - Rollback on failure
   - Release resources
   - Log errors
```

**Key Functions**:
- `Initialize(cfg *Config) error`
- `BuildExecutionPlan() (*ExecutionPlan, error)`
- `Execute(plan *ExecutionPlan) error`
- `Rollback() error`

---

## Plugin System Design

### Plugin Interface

```go
// Plugin is the interface all plugins must implement
type Plugin interface {
    // Metadata
    Name() string
    Version() string
    Description() string
    
    // Lifecycle hooks
    Initialize(ctx *PluginContext) error
    Validate() error
    Execute(ctx *ExecutionContext) error
    Rollback(ctx *ExecutionContext) error
    
    // Dependencies
    Dependencies() []string
    Phase() ExecutionPhase
}

// PluginContext provides shared state to plugins
type PluginContext struct {
    Config       *Config
    State        *StateManager
    Backup       *BackupManager
    Template     *TemplateEngine
    Logger       *Logger
    DryRun       bool
    WorkDir      string
    HomeDir      string
}

// ExecutionContext provides runtime context
type ExecutionContext struct {
    *PluginContext
    Changes      []Change
    Errors       []error
    StartTime    time.Time
}
```

### Plugin Lifecycle

```
Registration → Initialize → Validate → Execute → Cleanup
                   ↓            ↓          ↓
                 Error ←────────┴──────────┴──→ Rollback
```

### Plugin Registry

```go
type PluginRegistry struct {
    plugins map[string]Plugin
    mu      sync.RWMutex
}

func (r *PluginRegistry) Register(p Plugin) error {
    // Check for duplicate names
    // Validate dependencies
    // Add to registry
}

func (r *PluginRegistry) Get(name string) (Plugin, error) {
    // Retrieve plugin by name
}

func (r *PluginRegistry) GetOrderedPlugins() ([]Plugin, error) {
    // Topological sort based on dependencies
    // Group by execution phase
}
```

### Plugin Discovery

Plugins are statically compiled into the binary:

```go
// internal/plugins/registry.go
func init() {
    // Auto-register all built-in plugins
    RegisterPlugin(&files.Plugin{})
    RegisterPlugin(&directories.Plugin{})
    RegisterPlugin(&git.Plugin{})
    RegisterPlugin(&brew.Plugin{})
    RegisterPlugin(&onepassword.Plugin{})
    // ... more plugins
}
```

### Execution Phases

Plugins execute in predefined phases:

```go
const (
    PhasePreSync  ExecutionPhase = iota // Git operations
    PhaseCore                            // Files, directories
    PhaseRunOnce                         // Bootstrap scripts
    PhaseOnChange                        // Change-triggered tasks
    PhaseIntegration                     // Brew, package managers
    PhasePostSync                        // Validation, cleanup
)
```

**Phase Ordering**:
1. **PreSync**: Git pull, fetch external repos
2. **Core**: Place files, create directories
3. **RunOnce**: Execute bootstrap scripts
4. **OnChange**: Run change-triggered commands
5. **Integration**: Homebrew, package managers
6. **PostSync**: Validation, reporting

### Plugin Communication

Plugins don't directly communicate. They interact through:
- **Shared State**: Via `StateManager`
- **Config**: Read-only configuration access
- **Context**: Pass data through `ExecutionContext.Changes`

Example:
```go
// Git plugin records files changed
ctx.Changes = append(ctx.Changes, Change{
    Type: "git_pull",
    Files: []string{"Brewfile", ".zshrc"},
})

// OnChange plugin reacts to changes
func (p *OnChangePlugin) Execute(ctx *ExecutionContext) error {
    if hasChange(ctx.Changes, "Brewfile") {
        return p.runBrewBundle()
    }
    return nil
}
```

---

## Data Flow

### Initialization Flow (`kilt init`)

```
User: kilt init https://github.com/user/dotfiles
    ↓
1. Parse arguments (repo URL)
    ↓
2. Create directory structure (~/.kilt/)
    ↓
3. Clone repository to ~/.dotfiles (or configured dotfiles_path)
    ↓
4. Read config from ~/.dotfiles/.kilt/config.yaml
    ↓
5. Parse YAML configuration
    ↓
6. Initialize plugins
    ↓
7. Execute sync flow
```

### Sync Flow (`kilt sync`)

```
User: kilt sync
    ↓
1. Load configuration (~/.dotfiles/.kilt/config.yaml)
    ↓
2. Load state (~/.kilt/state/state.json)
    ↓
3. Acquire lock
    ↓
4. Git Plugin (PhasePreSync)
    │   ├─ Fetch remote changes
    │   ├─ Check for local changes
    │   ├─ Check for remote changes
    │   ├─ Pull remote changes (if any)
    │   └─ Commit and push local changes (if any)
    ↓
5. Build execution plan
    ↓
6. Create backup (if needed)
    ↓
7. Execute plugins (ordered by phase)
    │   ├─ Alternates Plugin (PhasePreSync)
    │   │   └─ Resolve OS/hostname-specific files
    │   ├─ Dotfiles Plugin (PhaseCore)
    │   │   ├─ For each dotfile entry
    │   │   ├─ Process directory-based entries (symlink all files)
    │   │   ├─ Process explicit mappings (symlink or template)
    │   │   └─ Smart dot-prefixing for common names
    │   ├─ Directories Plugin (PhaseCore)
    │   │   └─ Create missing directories
    │   ├─ Run Once Plugin (PhaseRunOnce)
    │   │   ├─ Check state
    │   │   └─ Execute if not done
    │   └─ Brew Plugin (PhaseIntegration)
    │       ├─ Detect Brewfile changes
    │       └─ Run brew bundle
    ↓
8. Update state
    ↓
9. Release lock
    ↓
10. Report summary
```

### Template Rendering Flow

```
File with template: true
    ↓
1. Read file content
    ↓
2. Load template data
    │   ├─ Built-in vars (hostname, os, arch)
    │   └─ Custom data (data.yaml)
    ↓
3. Parse template
    ↓
4. Execute template functions
    │   ├─ Standard Go functions
    │   └─ Custom functions (op, env)
    ↓
5. Render output
    ↓
6. Write to target location
```

---

## Directory Structure

### Project Structure

```
kilt/
├── cmd/
│   └── kilt/
│       └── main.go                 # CLI entry point
├── internal/
│   ├── core/
│   │   ├── config.go               # Config parser
│   │   ├── state.go                # State management
│   │   ├── backup.go               # Backup system
│   │   ├── template.go             # Template engine
│   │   └── engine.go               # Orchestrator
│   ├── plugin/
│   │   ├── interface.go            # Plugin interface
│   │   ├── registry.go             # Plugin registry
│   │   ├── loader.go               # Plugin loader
│   │   └── context.go              # Plugin contexts
│   └── plugins/
│       ├── files/
│       │   ├── plugin.go           # Files plugin
│       │   └── alternates.go       # Alternates logic
│       ├── directories/
│       │   └── plugin.go
│       ├── runonce/
│       │   └── plugin.go
│       ├── onchange/
│       │   └── plugin.go
│       ├── git/
│       │   └── plugin.go
│       ├── brew/
│       │   └── plugin.go
│       └── onepassword/
│           └── plugin.go
├── pkg/
│   ├── logger/
│   │   └── logger.go               # Logging utilities
│   └── utils/
│       ├── fs.go                   # Filesystem helpers
│       ├── cmd.go                  # Command execution
│       └── checksum.go             # Checksum utilities
├── test/
│   ├── fixtures/                   # Test data
│   ├── integration/                # Integration tests
│   └── e2e/                        # End-to-end tests
├── scripts/
│   ├── install.sh                  # Curl installer
│   └── build.sh                    # Build helper
├── docs/
│   ├── architecture.md             # This document
│   ├── plugin-guide.md             # Plugin development
│   └── examples/                   # Example configs
├── go.mod
├── go.sum
├── Makefile
├── .goreleaser.yml
├── README.md
└── tasks.md
```

### User Directory Structure

```
~/.dotfiles/                        # Dotfiles repository (cloned by kilt init)
├── .kilt/
│   └── config.yaml                 # Main configuration
├── zsh/                            # Example dotfile directory
├── git/                            # Example dotfile directory
└── ...

~/.kilt/
├── data.yaml                       # Custom template data (optional)
├── state/
│   ├── HEAD
│   ├── config
│   ├── objects/
│   └── refs/
├── state/
│   ├── state.json                  # Execution state
│   └── .lock                       # Lock file
├── backup/
│   ├── 20251125-143022/            # Timestamped backups
│   │   ├── .zshrc
│   │   └── .gitconfig
│   └── 20251124-092011/
├── cache/
│   └── templates/                  # Compiled templates (optional)
└── logs/
    ├── kilt.log                    # Main log file
    └── kilt-20251125.log           # Dated logs
```

---

## Configuration Schema

### Complete YAML Schema

```yaml
# ~/.dotfiles/.kilt/config.yaml

# Dotfiles repository (required for kilt init)
dotfiles_repo: "https://github.com/user/dotfiles"
dotfiles_path: "~/.dotfiles"  # default

# Optional: custom data file for templates
data_file: "data.yaml"

# Template engine (currently only "go")
template_engine: "go"

# Dotfiles to sync (symlinks from repo to ~)
dotfiles:
  # Simple form: directory name (links all files in directory)
  - zsh              # Links all files in zsh/ to ~/
  - git              # Links all files in git/ to ~/
  - vim
  
  # Explicit form: source/target mapping
  - source: ssh/config
    target: ~/.ssh/config
    template: true  # Will render as Go template
    mode: "0600"

# Extra Git repositories
extra_repos:
  - url: "https://github.com/zdharma-continuum/fast-syntax-highlighting"
    path: "~/.zsh/plugins/fast-syntax-highlighting"
    branch: "main"
    sparse: false
  - url: "https://github.com/zsh-users/zsh-autosuggestions"
    path: "~/.zsh/plugins/zsh-autosuggestions"

# Directories to ensure exist
directories:
  - "~/dev/personal"
  - "~/dev/work"
  - "~/screenshots"
  - "~/Documents/notes"

# Run-once scripts (executed in order, YAML arrays preserve order)
run_once:
  - "scripts/install_homebrew.sh"
  - "scripts/install_1password_cli.sh"
  - "scripts/setup_macos_defaults.sh"
  - "scripts/setup_ssh_keys.sh"

# On-change commands (executed when files change)
on_change:
  - "brew bundle --file=Brewfile"
  - "mise install --yes"

# Plugin-specific configuration
plugins:
  git:
    auto_pull: true
    ssh_key: "~/.ssh/id_ed25519"
  
  brew:
    brewfile: "Brewfile"
    auto_update: false
    cleanup_after: true
    bundles:
      - bootstrap
      - dev
  
  onepassword:
    account: "my.1password.com"
    cache_ttl: 3600  # Cache secrets for 1 hour
  
  alternates:
    patterns:
      - "##os.{darwin,linux}"
      - "##hostname.{work,personal}"
      - "##class.{laptop,desktop}"
```

### Custom Data File (`data.yaml`)

```yaml
# ~/.kilt/data.yaml
# Custom variables for templates

personal:
  email: "user@example.com"
  github_username: "johndoe"

work:
  email: "john.doe@company.com"
  github_username: "jdoe-company"

paths:
  projects_dir: "~/dev"
  notes_dir: "~/Documents/notes"

tools:
  editor: "nvim"
  terminal: "kitty"
```

Usage in templates:
```
# ~/.gitconfig template
[user]
    name = John Doe
    email = {{ .Custom.personal.email }}

[github]
    user = {{ .Custom.personal.github_username }}
```

---

## State Management

### State File Structure

```json
{
  "version": "1.0.0",
  "last_sync": "2025-11-25T14:30:22Z",
  "last_git_commit": "abc123def456",
  "run_once": {
    "install_homebrew.sh": {
      "task_id": "install_homebrew.sh",
      "executed_at": "2025-11-25T10:15:30Z",
      "exit_code": 0,
      "output": "Homebrew installed successfully",
      "checksum": "sha256:abc123..."
    }
  },
  "files": {
    "/Users/john/.zshrc": {
      "source_path": "zsh/zshrc",
      "target_path": "/Users/john/.zshrc",
      "checksum": "sha256:def456...",
      "updated_at": "2025-11-25T14:30:20Z",
      "backed_up": true
    }
  },
  "plugins": {
    "brew": {
      "last_run": "2025-11-25T14:30:21Z",
      "brewfile_checksum": "sha256:789abc...",
      "packages_installed": 42
    }
  }
}
```

### Concurrency Safety

- File-based locking using `~/.kilt/state/.lock`
- Lock acquisition with timeout (30 seconds default)
- Automatic lock cleanup on process exit
- Stale lock detection (> 5 minutes old)

```go
func (sm *StateManager) AcquireLock() error {
    lockFile := filepath.Join(sm.stateDir, ".lock")
    
    // Check for stale lock
    if isLockStale(lockFile) {
        os.Remove(lockFile)
    }
    
    // Try to acquire lock with timeout
    timeout := time.After(30 * time.Second)
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    
    for {
        select {
        case <-timeout:
            return ErrLockTimeout
        case <-ticker.C:
            if tryLock(lockFile) {
                return nil
            }
        }
    }
}
```

---

## Security Model

### Threat Model

**What we protect against**:
- Accidental data loss (backups)
- Path traversal attacks (validate all paths)
- Command injection (no shell interpolation)
- Secret leakage to Git (1Password integration)
- Malicious configuration files (validation)

**What we don't protect against**:
- Malicious dotfiles repository (user trusts their repo)
- Compromised system (not a security tool)
- Supply chain attacks on dependencies (Go module checksums help)

### Security Best Practices

#### 1. Path Validation
```go
func ValidatePath(path string) error {
    // Expand to absolute path
    absPath, err := filepath.Abs(path)
    if err != nil {
        return err
    }
    
    // Check for directory traversal
    if strings.Contains(absPath, "..") {
        return ErrInvalidPath
    }
    
    // Ensure within allowed directories
    if !isAllowedPath(absPath) {
        return ErrForbiddenPath
    }
    
    return nil
}
```

#### 2. Command Execution
```go
// NEVER use shell interpolation
// BAD: exec.Command("sh", "-c", userInput)
// GOOD:
func RunCommand(name string, args ...string) error {
    cmd := exec.Command(name, args...)
    cmd.Env = sanitizeEnv(os.Environ())
    return cmd.Run()
}
```

#### 3. Secret Handling
- Secrets never stored in Git or config files
- 1Password CLI used for secret injection
- Secrets only in memory during template rendering
- No logging of secret values
- Cache secrets with TTL (optional, in memory only)

#### 4. File Permissions
- Preserve or set explicit file modes
- Sensitive files (SSH keys) get 0600
- Config files get 0644
- Scripts get 0755
- Directories get 0755

### Audit Log

Optional audit log for security-sensitive operations:

```json
{
  "timestamp": "2025-11-25T14:30:22Z",
  "operation": "file_write",
  "user": "john",
  "source": "ssh/config",
  "target": "/Users/john/.ssh/config",
  "mode": "0600",
  "backed_up": true,
  "success": true
}
```

---

## Error Handling Strategy

### Error Types

```go
// Domain-specific errors
var (
    ErrConfigNotFound    = errors.New("config file not found")
    ErrInvalidConfig     = errors.New("invalid configuration")
    ErrLockTimeout       = errors.New("failed to acquire lock")
    ErrPluginFailed      = errors.New("plugin execution failed")
    ErrBackupFailed      = errors.New("backup creation failed")
    ErrRollbackFailed    = errors.New("rollback failed")
)

// Wrapping for context
type KiltError struct {
    Op      string  // Operation being performed
    Path    string  // File path (if applicable)
    Plugin  string  // Plugin name (if applicable)
    Err     error   // Underlying error
}

func (e *KiltError) Error() string {
    if e.Path != "" {
        return fmt.Sprintf("%s %s: %v", e.Op, e.Path, e.Err)
    }
    return fmt.Sprintf("%s: %v", e.Op, e.Err)
}
```

### Error Recovery

```go
func (e *Engine) Execute() error {
    // Create restore point
    checkpoint := e.CreateCheckpoint()
    
    // Execute with recovery
    err := e.executeInternal()
    if err != nil {
        // Attempt rollback
        if rbErr := e.Rollback(checkpoint); rbErr != nil {
            return fmt.Errorf("execution failed and rollback failed: %w, %v", err, rbErr)
        }
        return fmt.Errorf("execution failed (rolled back): %w", err)
    }
    
    return nil
}
```

### User-Friendly Error Messages

```go
func FormatError(err error) string {
    switch {
    case errors.Is(err, ErrConfigNotFound):
        return `Configuration file not found.
Run 'kilt init <repo-url>' to initialize, or create ~/.dotfiles/.kilt/config.yaml manually.`
    
    case errors.Is(err, ErrLockTimeout):
        return `Another kilt process is running.
Wait for it to complete or remove ~/.kilt/state/.lock if it's stale.`
    
    default:
        return fmt.Sprintf("Error: %v\nRun 'kilt doctor' for diagnostics.", err)
    }
}
```

---

## Testing Architecture

### Test Pyramid

```
        ┌──────────┐
        │   E2E    │  5% - Full workflow tests
        └──────────┘
      ┌──────────────┐
      │ Integration  │  15% - Plugin + core interaction
      └──────────────┘
    ┌──────────────────┐
    │   Unit Tests     │  80% - Individual components
    └──────────────────┘
```

### Testing Strategies

#### Unit Tests
```go
// Use interfaces and dependency injection
type FileSystem interface {
    ReadFile(path string) ([]byte, error)
    WriteFile(path string, data []byte, perm os.FileMode) error
    // ...
}

// Test with mock filesystem
func TestFilesPlugin(t *testing.T) {
    mockFS := &MockFileSystem{
        files: map[string][]byte{
            "source.txt": []byte("content"),
        },
    }
    
    plugin := NewFilesPlugin(mockFS)
    err := plugin.Execute(ctx)
    assert.NoError(t, err)
}
```

#### Integration Tests
```go
// Test plugin interactions
func TestGitAndFilesPlugins(t *testing.T) {
    // Create temp repo
    repo := setupTestRepo(t)
    defer repo.Cleanup()
    
    // Run full sync
    engine := NewEngine(testConfig)
    err := engine.Execute()
    require.NoError(t, err)
    
    // Verify files were created
    assertFileExists(t, "~/.zshrc")
}
```

#### E2E Tests
```bash
#!/bin/bash
# test/e2e/test_full_workflow.sh

# Test full installation workflow
export TEST_HOME=$(mktemp -d)
export HOME=$TEST_HOME

# Install kilt
curl -sL http://localhost:8080/install.sh | bash -s -- https://github.com/test/dotfiles

# Verify installation
test -f "$HOME/.zshrc" || exit 1
test -d "$HOME/.kilt" || exit 1

# Run sync again (idempotency)
kilt sync

# Cleanup
rm -rf $TEST_HOME
```

### Test Fixtures

```
test/fixtures/
├── configs/
│   ├── minimal.yaml
│   ├── full.yaml
│   └── invalid.yaml
├── repos/
│   └── test-dotfiles/
│       ├── .kilt/
│       │   └── config.yaml
│       ├── zshrc
│       └── Brewfile
└── states/
    ├── empty.json
    └── with-history.json
```

### Mocking External Commands

```go
// pkg/utils/cmd.go
var execCommand = exec.Command  // var for testing

func RunGitCommand(args ...string) error {
    cmd := execCommand("git", args...)
    return cmd.Run()
}

// In tests
func TestGitPull(t *testing.T) {
    defer func() { execCommand = exec.Command }()
    
    execCommand = func(name string, args ...string) *exec.Cmd {
        return exec.Command("echo", "mock output")
    }
    
    err := RunGitCommand("pull")
    assert.NoError(t, err)
}
```

---

## Performance Considerations

### Optimization Strategies

#### 1. Lazy Loading
- Load configuration only when needed
- Parse templates on-demand
- Initialize plugins only when used

#### 2. Parallel Execution
- Independent plugins can run concurrently
- File operations parallelized where safe
- Use worker pools for bulk operations

```go
func (p *FilesPlugin) Execute(ctx *ExecutionContext) error {
    files := p.getFilesToProcess()
    
    // Process files in parallel
    workers := runtime.NumCPU()
    errChan := make(chan error, len(files))
    
    var wg sync.WaitGroup
    sem := make(chan struct{}, workers)
    
    for _, file := range files {
        wg.Add(1)
        sem <- struct{}{}  // Acquire
        
        go func(f FileMapping) {
            defer wg.Done()
            defer func() { <-sem }()  // Release
            
            if err := p.processFile(ctx, f); err != nil {
                errChan <- err
            }
        }(file)
    }
    
    wg.Wait()
    close(errChan)
    
    // Collect errors
    var errs []error
    for err := range errChan {
        errs = append(errs, err)
    }
    
    return errors.Join(errs...)
}
```

#### 3. Caching
- Cache template compilations
- Cache checksum calculations
- Cache 1Password secrets (with TTL)
- Cache Git status checks

#### 4. Incremental Operations
- Only process changed files
- Skip unmodified directories
- Reuse existing backups if unchanged

### Performance Targets

- **Initialization**: < 5 seconds (including Git clone)
- **Sync (no changes)**: < 1 second
- **Sync (with changes)**: < 10 seconds for typical setup
- **Memory**: < 50MB peak usage
- **Binary size**: < 20MB (compressed)

### Profiling

Enable profiling with debug flags:

```bash
kilt sync --profile=cpu --profile-out=cpu.prof
kilt sync --profile=mem --profile-out=mem.prof

# Analyze with pprof
go tool pprof cpu.prof
```

---

## Future Considerations

### Phase 2 Features

1. **Web UI** (optional)
   - View configuration
   - Manage backups
   - Visualize file mappings

2. **Plugin Marketplace**
   - Community plugins
   - Plugin discovery
   - Version management

3. **Multi-machine Sync**
   - Detect configuration drift
   - Sync state across machines
   - Conflict resolution

4. **Enhanced Templating**
   - Jinja2-style templates (optional)
   - Template includes/inheritance
   - Conditional rendering

5. **Encrypted Secrets**
   - Alternative to 1Password
   - Age encryption
   - Git-crypt integration

### Extensibility Points

1. **Custom Plugins**
   - Go plugins (plugin.Open)
   - WASM plugins
   - External executables

2. **Custom Template Functions**
   - User-defined functions
   - Plugin-provided functions

3. **Custom Hooks**
   - Pre/post sync hooks
   - File change hooks
   - Error hooks

---

## Appendix

### Technology Choices

| Component | Technology | Rationale |
|-----------|-----------|-----------|
| Language | Go 1.21+ | Fast, single binary, great stdlib |
| CLI Framework | cobra + pflag | Industry standard, great UX |
| Config Format | YAML | Order-preserving arrays, human-readable |
| Template Engine | Go text/template | Stdlib, powerful, familiar |
| Testing | testify + mockery | Best Go testing ecosystem |
| Filesystem Mocking | afero | Standard for FS abstraction |
| Logging | zap or zerolog | High-performance structured logging |
| Build Tool | goreleaser | Multi-platform builds, GitHub integration |
| Linting | golangci-lint | Comprehensive linter aggregator |

### Key Dependencies

```go
require (
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.0  // Optional for advanced config
    gopkg.in/yaml.v3 v3.0.1
    github.com/spf13/afero v1.11.0
    github.com/stretchr/testify v1.8.4
    go.uber.org/zap v1.26.0
    golang.org/x/sync v0.5.0  // For errgroup
)
```

### Glossary

- **Bare Repository**: Git repo without a working tree, stores only .git contents
- **Idempotent**: Operation that produces the same result regardless of how many times it's executed
- **Plugin**: Self-contained module that implements specific functionality
- **Template**: File with placeholders that get replaced with actual values
- **Alternates**: File variants for different OS/hostname/context
- **Run Once**: Task that executes exactly once per machine
- **On Change**: Task that executes when specific files change
- **Dry Run**: Simulation mode that shows what would happen without executing

---

**Document Version**: 1.0.0  
**Last Updated**: November 25, 2025  
**Next Review**: After Phase 1 implementation

