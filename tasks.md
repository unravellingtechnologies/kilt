# Kilt — Development Task List

**Project**: Kilt - Git-first Mac bootstrapper  
**Start Date**: November 25, 2025  
**Tech Stack**: Go 1.21+, TOML config, Plugin architecture

---

## Phase 1: Project Foundation & Core Architecture

### 1.1 Project Setup
**Status**: DONE  
**Description**: Initialize Go project with proper structure, dependencies, and tooling setup.

**Tasks**:
- [x] Initialize Go module (`go mod init`)
- [x] Create directory structure (`cmd/`, `internal/`, `pkg/`, `scripts/`, `test/`)
- [x] Set up `.gitignore` for Go projects
- [x] Create `Makefile` with build, test, lint, install targets
- [x] Add `go.sum` and initial dependencies (cobra, viper, TOML parser) - dependencies ready, will be added when imported
- [x] Set up GitHub Actions for CI (test, lint, build)
- [x] Create initial README.md with project overview

**Dependencies**: None  
**Estimated Complexity**: Low

---

### 1.2 Plugin System Architecture
**Status**: DONE  
**Description**: Design and implement the core plugin system that allows modular feature additions.

**Tasks**:
- [x] Define `Plugin` interface in `internal/plugin/interface.go`
  - Lifecycle methods: `Initialize()`, `Execute()`, `Validate()`
  - Metadata: `Name()`, `Version()`, `Dependencies()`
- [x] Create `PluginRegistry` in `internal/plugin/registry.go`
  - Register/unregister plugins
  - Dependency resolution
  - Execution order management
- [x] Implement `PluginLoader` in `internal/plugin/loader.go`
  - Auto-discover plugins from `internal/plugins/`
  - Load plugin configuration from TOML
- [x] Create plugin context struct for sharing state
- [x] Write unit tests for plugin system
- [x] Document plugin development guide

**Dependencies**: 1.1  
**Estimated Complexity**: High

---

### 1.3 Configuration System
**Status**: DONE  
**Description**: TOML-based configuration parser with validation and schema support.

**Tasks**:
- [x] Define configuration structs in `internal/core/config.go`
  - Global config, plugin configs, file mappings
  - Use TOML tags for parsing
- [x] Implement TOML parser using `github.com/BurntSushi/toml`
- [x] Add configuration validation
  - Required fields, path validation, circular dependencies
- [x] Support `~` expansion and environment variables
- [x] Implement config file discovery (`.kilt/config.toml`, `~/.kilt/config.toml`)
- [x] Add schema documentation generator
- [x] Write comprehensive config parsing tests

**Dependencies**: 1.1  
**Estimated Complexity**: Medium

---

### 1.4 CLI Framework
**Status**: DONE  
**Description**: Set up cobra-based CLI with all main commands and flags.

**Tasks**:
- [x] Set up cobra CLI in `cmd/kilt/main.go`
- [x] Implement root command with global flags
  - `--dry-run`, `--verbose`, `--config`, `--no-color`, `--force`
- [x] Create `kilt init <repo-url>` command
  - Clone repository, set up bare repo structure
- [x] Create `kilt sync` command
  - Pull latest, apply changes, run plugins (plugin execution pending engine implementation)
- [x] Create `kilt doctor` command
  - Validate configuration, check dependencies
- [x] Create `kilt version` command
- [x] Add command completion (bash, zsh, fish, powershell)
- [x] Write CLI integration tests

**Dependencies**: 1.1, 1.2, 1.3  
**Estimated Complexity**: Medium

---

## Phase 2: Core Engine Features

### 2.1 State Management & Idempotency
**Status**: DONE  
**Description**: Track execution state to ensure idempotent operations.

**Tasks**:
- [x] Create state directory structure (`.kilt/state/`)
- [x] Implement state tracking in `internal/core/state.go`
  - Track run-once tasks, file checksums, plugin executions
- [x] Use JSON or SQLite for state persistence (JSON chosen for human readability)
- [x] Add state locking for concurrent execution safety
- [x] Implement state cleanup and garbage collection
- [x] Add `kilt reset` command to clear state
- [x] Write state management tests

**Dependencies**: 1.1, 1.3  
**Estimated Complexity**: Medium

---

### 2.2 Backup System
**Status**: TODO  
**Description**: Automatic backup of existing files before modification.

**Tasks**:
- [ ] Create backup directory structure (`.kilt/backup/YYYYMMDD-HHMMSS/`)
- [ ] Implement backup logic in `internal/core/backup.go`
  - Copy files before overwrite, maintain directory structure
- [ ] Add incremental backup (only changed files)
- [ ] Implement `kilt restore <timestamp>` command
- [ ] Add backup retention policy (keep last N backups)
- [ ] Add backup size reporting
- [ ] Write backup tests with fixtures

**Dependencies**: 1.1  
**Estimated Complexity**: Low

---

### 2.3 Template Engine
**Status**: TODO  
**Description**: Go template support with custom functions and variable injection.

**Tasks**:
- [ ] Implement template engine in `internal/core/template.go`
- [ ] Use Go's `text/template` package
- [ ] Add built-in template functions
  - `{{ .hostname }}`, `{{ .os }}`, `{{ .arch }}`
- [ ] Support custom data file (`data.toml`, `data.yaml`)
- [ ] Add template validation and error reporting
- [ ] Support template delimiters customization
- [ ] Create template testing utilities
- [ ] Write comprehensive template tests

**Dependencies**: 1.1, 1.3  
**Estimated Complexity**: Medium

---

### 2.4 Core Engine Orchestrator
**Status**: TODO  
**Description**: Main orchestrator that coordinates plugins and core features.

**Tasks**:
- [ ] Create `Engine` struct in `internal/core/engine.go`
- [ ] Implement plugin execution pipeline
  - Pre-flight checks, plugin ordering, error handling
- [ ] Add dry-run mode simulation
- [ ] Implement rollback on failure
- [ ] Add progress reporting and logging
- [ ] Create execution context with cancellation
- [ ] Write orchestrator integration tests

**Dependencies**: 1.2, 1.3, 2.1  
**Estimated Complexity**: High

---

## Phase 3: Core Plugins

### 3.1 Files Plugin
**Status**: TODO  
**Description**: Core plugin for file placement with source → target mapping.

**Tasks**:
- [ ] Create plugin structure in `internal/plugins/files/`
- [ ] Implement `Plugin` interface
- [ ] Parse `[[files]]` sections from TOML config
- [ ] Support arbitrary target paths with `~` expansion
- [ ] Add file diff generation for dry-run
- [ ] Handle symlinks vs copies (make configurable)
- [ ] Implement file permission preservation
- [ ] Write plugin tests with mock filesystem

**Dependencies**: 1.2, 2.2, 2.3  
**Estimated Complexity**: Medium

---

### 3.2 Alternates Plugin
**Status**: TODO  
**Description**: Automatic file selection based on OS, hostname, architecture.

**Tasks**:
- [ ] Create plugin in `internal/plugins/alternates/`
- [ ] Implement file matching logic
  - `file.mac`, `file.linux`, `file.hostname@work`
- [ ] Define priority rules for multiple matches
- [ ] Integrate with Files plugin
- [ ] Support custom alternate patterns
- [ ] Add alternate resolution logging
- [ ] Write tests for all matching scenarios

**Dependencies**: 3.1  
**Estimated Complexity**: Medium

---

### 3.3 Directories Plugin
**Status**: TODO  
**Description**: Ensure specified directories exist with proper permissions.

**Tasks**:
- [ ] Create plugin in `internal/plugins/directories/`
- [ ] Parse `directories` array from config
- [ ] Create directories with parents if needed
- [ ] Support permission specification
- [ ] Add dry-run reporting
- [ ] Handle existing directories gracefully
- [ ] Write plugin tests

**Dependencies**: 1.2  
**Estimated Complexity**: Low

---

### 3.4 Run Once Plugin
**Status**: TODO  
**Description**: Execute bootstrap scripts exactly once per machine.

**Tasks**:
- [ ] Create plugin in `internal/plugins/runonce/`
- [ ] Track execution in state system (`.kilt/state/runonce/`)
- [ ] Execute shell scripts in order
- [ ] Capture stdout/stderr for logging
- [ ] Add `--force-run` flag to re-execute
- [ ] Implement timeout protection
- [ ] Handle script failures gracefully
- [ ] Write plugin tests with mock scripts

**Dependencies**: 1.2, 2.1  
**Estimated Complexity**: Medium

---

### 3.5 On Change Plugin
**Status**: TODO  
**Description**: Execute commands when configuration or files change.

**Tasks**:
- [ ] Create plugin in `internal/plugins/onchange/`
- [ ] Detect changes via checksums or git diff
- [ ] Execute commands in specified order
- [ ] Add change detection granularity (file-level, global)
- [ ] Support conditional execution (only if X changed)
- [ ] Capture command output
- [ ] Write plugin tests

**Dependencies**: 1.2, 2.1  
**Estimated Complexity**: Medium

---

### 3.6 Git Plugin
**Status**: TODO  
**Description**: Manage bare Git repository and clone extra repositories.

**Tasks**:
- [ ] Create plugin in `internal/plugins/git/`
- [ ] Implement bare repo setup at `~/.kilt/repo`
- [ ] Add Git operations: clone, pull, status
- [ ] Handle extra repositories from config (`[[extra_repos]]`)
- [ ] Support sparse checkouts
- [ ] Add authentication handling (SSH keys, tokens)
- [ ] Implement `kilt sync` git pull integration
- [ ] Write plugin tests (mock git commands or use test repos)

**Dependencies**: 1.2  
**Estimated Complexity**: High

---

### 3.7 Brew Plugin
**Status**: TODO  
**Description**: Homebrew integration for package management.

**Tasks**:
- [ ] Create plugin in `internal/plugins/brew/`
- [ ] Check if Homebrew is installed
- [ ] Run `brew bundle --file=Brewfile`
- [ ] Add `kilt brew` subcommand
  - `kilt brew install`, `kilt brew update`, `kilt brew cleanup`
- [ ] Detect Brewfile changes for on-change execution
- [ ] Handle Linux (Linuxbrew) compatibility
- [ ] Write plugin tests (mock brew commands)

**Dependencies**: 1.2  
**Estimated Complexity**: Medium

---

### 3.8 1Password Plugin
**Status**: TODO  
**Description**: Secret injection from 1Password CLI without storing in Git.

**Tasks**:
- [ ] Create plugin in `internal/plugins/onepassword/`
- [ ] Check if `op` CLI is installed
- [ ] Implement template function `{{ op "path/to/secret" }}`
- [ ] Handle 1Password authentication
- [ ] Add caching for secret lookups (optional, with TTL)
- [ ] Provide clear error messages for missing secrets
- [ ] Support multiple 1Password accounts
- [ ] Write plugin tests (mock `op` commands)

**Dependencies**: 1.2, 2.3  
**Estimated Complexity**: High

---

## Phase 4: Installation & Distribution

### 4.1 Curl Installer Script
**Status**: TODO  
**Description**: Bash script for one-line installation and bootstrapping.

**Tasks**:
- [ ] Create `scripts/install.sh`
- [ ] Detect OS and architecture
- [ ] Download appropriate binary from GitHub releases
- [ ] Verify checksum/signature
- [ ] Install to `/usr/local/bin/kilt` or `~/.local/bin/kilt`
- [ ] Support `curl ... | bash -s -- <repo-url>` for full bootstrap
- [ ] Add error handling and rollback
- [ ] Test on macOS (Intel, Apple Silicon) and Linux

**Dependencies**: 1.4  
**Estimated Complexity**: Medium

---

### 4.2 Binary Building & Releases
**Status**: TODO  
**Description**: Cross-platform builds and GitHub release automation.

**Tasks**:
- [ ] Add `goreleaser` configuration
- [ ] Configure multi-platform builds
  - macOS (amd64, arm64)
  - Linux (amd64, arm64)
- [ ] Set up GitHub Actions for releases
- [ ] Add version injection at build time
- [ ] Create checksums and signatures
- [ ] Add Homebrew tap/formula (optional)
- [ ] Document release process

**Dependencies**: 1.4, 4.1  
**Estimated Complexity**: Medium

---

## Phase 5: Testing & Quality Assurance

### 5.1 Unit Test Coverage
**Status**: TODO  
**Description**: Comprehensive unit tests for all packages.

**Tasks**:
- [ ] Aim for >80% code coverage
- [ ] Use `testify` for assertions
- [ ] Create test fixtures in `test/fixtures/`
- [ ] Mock filesystem operations (`afero`)
- [ ] Mock external commands (git, brew, op)
- [ ] Set up coverage reporting
- [ ] Add coverage to CI pipeline

**Dependencies**: All implementation tasks  
**Estimated Complexity**: High

---

### 5.2 Integration Tests
**Status**: TODO  
**Description**: End-to-end tests for real-world scenarios.

**Tasks**:
- [ ] Create test repository with sample configs
- [ ] Test full `kilt init` → `kilt sync` flow
- [ ] Test plugin interactions
- [ ] Test error scenarios and recovery
- [ ] Use Docker for isolated test environments
- [ ] Add integration tests to CI
- [ ] Document test setup

**Dependencies**: 1.4, All plugins  
**Estimated Complexity**: High

---

### 5.3 Linting & Code Quality
**Status**: TODO  
**Description**: Enforce code quality standards.

**Tasks**:
- [ ] Set up `golangci-lint` with strict config
- [ ] Add pre-commit hooks (optional)
- [ ] Run linter in CI
- [ ] Add `make lint` target
- [ ] Document code style guidelines
- [ ] Use `gofmt` and `goimports`

**Dependencies**: 1.1  
**Estimated Complexity**: Low

---

## Phase 6: Documentation & Polish

### 6.1 User Documentation
**Status**: TODO  
**Description**: Comprehensive user guides and examples.

**Tasks**:
- [ ] Write detailed README.md
  - Quick start, installation, usage
- [ ] Create example configurations
- [ ] Write plugin documentation
- [ ] Add troubleshooting guide
- [ ] Create migration guide from other tools (yadm, chezmoi)
- [ ] Add command reference (auto-generated from cobra)
- [ ] Create video tutorial (optional)

**Dependencies**: All implementation tasks  
**Estimated Complexity**: Medium

---

### 6.2 Developer Documentation
**Status**: TODO  
**Description**: Documentation for contributors and plugin developers.

**Tasks**:
- [ ] Write CONTRIBUTING.md
- [ ] Create plugin development guide
- [ ] Document architecture and design decisions
- [ ] Add code comments and godoc
- [ ] Create development setup guide
- [ ] Document testing strategy

**Dependencies**: All implementation tasks  
**Estimated Complexity**: Low

---

### 6.3 Final Polish
**Status**: TODO  
**Description**: UX improvements and edge case handling.

**Tasks**:
- [ ] Add colorized output (respect `--no-color`)
- [ ] Improve error messages with suggestions
- [ ] Add progress bars for long operations
- [ ] Implement `--verbose` and `--debug` logging
- [ ] Add telemetry/analytics (opt-in, privacy-focused)
- [ ] Performance profiling and optimization
- [ ] Security audit (input validation, path traversal, etc.)

**Dependencies**: All implementation tasks  
**Estimated Complexity**: Medium

---

## Summary

**Total Tasks**: 23 major milestones  
**Estimated Timeline**: 8-12 weeks (solo developer)  
**Critical Path**: Phase 1 → Phase 2 → Phase 3 → Phase 4

**Key Risks**:
1. Plugin system complexity
2. Cross-platform compatibility (macOS/Linux)
3. 1Password CLI integration reliability
4. Git bare repository edge cases

**Next Steps**:
1. Review and refine this plan
2. Set up project repository
3. Begin Phase 1.1 (Project Setup)

