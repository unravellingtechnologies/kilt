# Kilt

**Bootstrap manager for Mac. Think dot files management, but with a beard**

A Git-first, curl-installable bootstrapper that transforms a fresh macOS (or Linux) system into a personalized development environment using a single command. Kilt emphasizes idempotency, safety, and extensibility through a plugin architecture.

## Vision

One command to turn any fresh Mac into *your* Mac in under 5 minutes — completely from a single Git repository, with zero secrets ever touching Git, full idempotency, and the flexibility to manage arbitrary files in arbitrary locations.

## Quick Start

```bash
curl -sL https://get.kilt.pro | bash -s -- https://github.com/you/dotfiles
```

## Core Principles

- **Git is the source of truth** — no external backends
- **Zero secrets in Git** — secrets pulled live from 1Password during apply
- **Idempotent & safe** — run 100 times with the same result
- **Works offline** — after first install, no network required
- **No runtime dependencies** — only Git and Bash after bootstrap
- **macOS-first, Linux-friendly** — optimized for Mac, works on Linux

## Features

### 🚀 One-line Installation
Install and bootstrap everything with a single curl command that clones your dotfiles repository and applies all configurations.

### 📁 Flexible File Management
Place files anywhere on your filesystem with explicit source → target mappings. Supports `~` expansion and arbitrary paths.

### 🎯 Idempotent Operations
All operations are safe to run multiple times. State tracking ensures run-once tasks execute exactly once.

### 🔒 Secure Secret Management
Secrets are never stored in Git. Use 1Password CLI integration to inject secrets dynamically during template rendering.

### 🧩 Plugin Architecture
Extensible plugin system allows modular feature additions:
- **Dotfiles Plugin** ✅: Symlink-based dotfile synchronization with directory support and smart dot-prefixing
- **Git Plugin** ✅: Bidirectional Git synchronization (pull remote changes, push local changes)
- **Alternates Plugin** ✅: OS/hostname-based file selection for explicit dotfile mappings
- **Directories Plugin** ✅: Ensure directories exist with proper permissions
- **Run Once Plugin** ✅: Execute bootstrap scripts exactly once per machine
- **On Change Plugin**: Run commands when files change
- **Git Plugin**: Manage bare repository and clone extra repos
- **Brew Plugin**: Homebrew integration for package management
- **1Password Plugin**: Secret injection from 1Password CLI

### 🎨 Template Support
Full Go template support with built-in variables:
- System info: `{{ .Hostname }}`, `{{ .OS }}`, `{{ .Arch }}`, `{{ .User }}`, `{{ .Home }}`
- Environment variables: `{{ env "VAR" }}`
- 1Password secrets: `{{ op "path/to/secret" }}` (requires 1Password plugin)
- Custom data from `data.toml` or `data.yaml`: `{{ .Custom.key }}`
- Customizable delimiters (default: `{{` and `}}`)

### 🛡️ Safety Features
- **Automatic backups** before file modifications with timestamped directories
- **Incremental backups** that only backup changed files
- **Backup retention policy** to manage disk space
- **Restore functionality** to recover from any backup
- Dry-run mode to preview changes
- Rollback capability on failure
- State locking for concurrent execution safety

## Project Status

**Status**: 🚧 In Development (Phase 3: Core Plugins)

**Completed**:
- ✅ Phase 1: Project Foundation & Core Architecture
  - ✅ Project Setup (1.1)
  - ✅ Plugin System Architecture (1.2)
  - ✅ Configuration System (1.3)
  - ✅ CLI Framework (1.4)
- ✅ Phase 2: Core Engine Features
  - ✅ State Management & Idempotency (2.1)
  - ✅ Backup System (2.2)
  - ✅ Template Engine (2.3)
  - ✅ Core Engine Orchestrator (2.4)
- ✅ Phase 3: Core Plugins
  - ✅ Files Plugin (3.1)
  - ✅ Alternates Plugin (3.2)
  - ✅ Directories Plugin (3.3)
  - ✅ Run Once Plugin (3.4)

**In Progress**: Phase 3 - Core Plugins (Alternates, Directories, Run Once, etc.)

This project is currently in early development. See [tasks.md](tasks.md) for the full development roadmap and [architecture.md](architecture.md) for detailed architecture documentation.

## Requirements

- Go 1.21+ (for building from source)
- Git
- Bash (for installer script)
- macOS or Linux

## Building from Source

```bash
# Clone the repository
git clone https://github.com/unravelling/kilt.git
cd kilt

# Build the binary
make build

# Install locally
make install

# Run tests
make test

# Run linter
make lint
```

See `make help` for all available targets.

## Configuration

Kilt uses YAML configuration files. Create a `config.yaml` in your dotfiles repository:

```yaml
# Dotfiles repository (required for kilt init)
dotfiles_repo: "https://github.com/user/dotfiles"
dotfiles_path: "~/.dotfiles"  # default

# Template data
data_file: data.yaml
template_engine: go

# Dotfiles to sync (symlinks from repo to ~)
dotfiles:
  - zsh              # Links all files in zsh/ to ~/
  - git              # Links all files in git/ to ~/
  - vim
  # Explicit mapping for special cases
  - source: ssh/config
    target: ~/.ssh/config
    mode: "0600"
    template: true

# Additional directories to create
directories:
  - ~/dev/personal
  - ~/dev/work

# Run-once scripts (order preserved)
run_once:
  - scripts/install_homebrew.sh
  - scripts/setup_macos.sh
  - scripts/setup_ssh_keys.sh

# Plugin-specific configuration
plugins:
  runonce:
    timeout: "10m"      # Optional: custom timeout per script (default: 5m)
    force_run: false     # Optional: force re-execution of already-run scripts
  
  brew:
    bundles:
      - bootstrap    # expects brew/bootstrap file
      - dev          # expects brew/dev file
```

### Dotfiles Plugin

The Dotfiles Plugin is the core plugin for managing dotfile synchronization. It supports:

- **Directory-based syncing**: Link all files in a directory to your home directory
- **Symlink creation**: Creates symlinks (not copies) by default for all dotfiles
- **Smart dot-prefixing**: Automatically adds `.` prefix to common dotfile names (e.g., `zshrc` → `~/.zshrc`)
- **Template rendering**: For explicit mappings, render Go templates when `template = true`
- **Permission control**: Set explicit file permissions via `mode` for special cases
- **Automatic backups**: Existing files are automatically backed up before modification
- **Idempotency**: Skips unchanged files based on checksum comparison
- **Dry-run support**: Preview changes without modifying filesystem

**Dotfile Entry Formats**:

1. **Simple form** (directory name):
   ```yaml
   dotfiles:
     - zsh    # Links all files in zsh/ to ~/
     - git
   ```

2. **Explicit mapping** (source/target):
   ```yaml
   dotfiles:
     - source: ssh/config
       target: ~/.ssh/config
       mode: "0600"
       template: true
   ```

**Dotfile Entry Fields**:
- `directory` (optional): Directory name to sync (simple form)
- `source` (optional): Explicit source file path relative to dotfiles repo
- `target` (optional): Explicit target path (required if source is set)
- `template` (optional): Whether to render as template (default: `false`)
- `mode` (optional): File permissions in octal format (e.g., `"0644"`, `"0600"`)

**Note**: For template files, the rendered content is written (not symlinked) to ensure templates are always up-to-date.

### Run Once Plugin

The Run Once Plugin executes bootstrap scripts exactly once per machine. It's perfect for initial setup tasks like installing Homebrew, setting up SSH keys, or configuring system preferences.

**Key Features**:
- **Idempotent execution**: Scripts run once and are tracked in state
- **Order preservation**: Scripts execute in the order specified in config
- **Timeout protection**: Configurable timeout prevents hanging scripts (default: 5 minutes)
- **Output capture**: Captures and logs stdout/stderr for debugging
- **Force re-execution**: Can force scripts to run again via plugin config
- **Graceful failure handling**: Failed scripts are recorded to prevent infinite retries
- **Shebang support**: Automatically detects and uses script shebangs
- **Dry-run support**: Preview which scripts would execute without running them

**Basic Configuration**:

```yaml
run_once:
  - scripts/install_homebrew.sh
  - scripts/setup_macos.sh
  - scripts/setup_ssh_keys.sh
```

**With Plugin Configuration**:

```yaml
run_once:
  - scripts/install_homebrew.sh
  - scripts/setup_macos.sh
  - scripts/long_running_script.sh

plugins:
  runonce:
    timeout: "15m"       # Custom timeout (default: 5m)
    force_run: false     # Set to true to re-execute all scripts
```

**Script Requirements**:
- Scripts must exist and be readable
- Scripts should be executable (chmod +x)
- Scripts can use any shebang (#!/bin/sh, #!/bin/bash, #!/usr/bin/env python3, etc.)
- Scripts without shebangs default to `sh`
- Scripts run in their own directory context

**Example Script**:

```bash
#!/bin/sh
# scripts/install_homebrew.sh

set -e

if ! command -v brew &> /dev/null; then
  echo "Installing Homebrew..."
  /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
else
  echo "Homebrew already installed"
fi
```

**State Tracking**:
- Each script execution is tracked in `.kilt/state/state.json`
- Task IDs are generated from script paths (SHA256 hash)
- Execution records include: timestamp, exit code, and output
- Failed scripts are also recorded to prevent retry loops

**Force Re-execution**:
To re-run scripts that have already executed, set `force_run: true` in plugin config:

```yaml
plugins:
  runonce:
    force_run: true  # Re-execute all scripts, even if already completed
```

**Timeout Configuration**:
Set a custom timeout for all scripts:

```yaml
plugins:
  runonce:
    timeout: "30m"  # 30 minutes (supports: s, m, h)
```

**Error Handling**:
- Scripts that exit with non-zero codes are treated as failures
- Failed scripts are recorded in state (prevents infinite retries)
- Execution stops on first failure (other plugins may rollback)
- Script output (stdout/stderr) is captured and stored for debugging

See [architecture.md](architecture.md) for the complete configuration schema.

## Commands

- `kilt init <repo-url>` — Initialize from Git repository
  - Clones the repository to `~/.dotfiles` (configurable via `dotfiles_path`)
  - Reads configuration from `~/.dotfiles/.kilt/config.yaml`
  - Sets up the `.kilt` directory structure
- `kilt sync` — Synchronize dotfiles bidirectionally
  - **Git synchronization**: Automatically pulls remote changes and pushes local changes
  - Handles conflicts gracefully (alerts user if manual resolution needed)
  - Loads and validates configuration
  - Executes plugins in the correct order to sync dotfiles
- `kilt doctor` — Validate setup and dependencies
  - Checks repository initialization
  - Validates configuration file
  - Verifies required dependencies (git, etc.)
  - Checks file permissions and paths
- `kilt version` — Show version information
- `kilt completion [bash|zsh|fish|powershell]` — Generate shell completion scripts
- `kilt restore <backup-id>` — Restore files from a backup
  - Restores files from a backup using the backup ID (format: `YYYYMMDD-HHMMSS`)
  - Shows preview of files to be restored before confirmation
  - Use `--force` flag to skip confirmation prompt
  - Example: `kilt restore 20250125-143022`
- `kilt backups list` — List all available backups
  - Shows backup ID, date, time, file count, size, and description
  - Use `--json` flag for machine-readable JSON output
  - Example: `kilt backups list` or `kilt backups list --json`
- `kilt reset` — Clear state and start fresh

### Global Flags

All commands support the following global flags:

- `--dry-run` — Show what would happen without executing
- `--verbose, -v` — Detailed logging output
- `--config <path>` — Custom config file location (default: `.kilt/config.toml` or `~/.kilt/config.toml`)
- `--no-color` — Disable colored output
- `--force` — Skip confirmation prompts

## Development

### Project Structure

```
kilt/
├── cmd/kilt/          # CLI entry point
├── internal/          # Internal packages
│   ├── core/          # Core engine (config, state, backup, templates)
│   ├── plugin/        # Plugin system (registry, loader, interfaces)
│   └── plugins/       # Built-in plugins
├── pkg/               # Public packages
│   ├── logger/        # Logging utilities
│   └── utils/         # Utility functions
├── scripts/           # Build and install scripts
└── test/              # Test fixtures and integration tests
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific test package
go test ./internal/core/...
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint

# Run all checks (fmt, vet, lint, test)
make check
```

## Contributing

Contributions are welcome! Please read the architecture documentation and development guide before submitting PRs.

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run `make check` to ensure quality
5. Submit a pull request

## License

See [LICENSE](LICENSE) for details.

## Acknowledgments

Inspired by tools like:
- [yadm](https://yadm.io/) — Yet Another Dotfiles Manager
- [chezmoi](https://www.chezmoi.io/) — Manage your dotfiles across multiple machines
- [homesick](https://github.com/technicalpickles/homesick) — Your home directory is your castle

## Roadmap

See [tasks.md](tasks.md) for the complete development roadmap. Current focus:

- ✅ Phase 1: Project Foundation & Core Architecture
- ✅ Phase 2: Core Engine Features
- 🚧 Phase 3: Core Plugins (Files ✅, Alternates ✅, Directories ✅, Run Once ✅, others in progress)
- ⏳ Phase 4: Installation & Distribution
- ⏳ Phase 5: Testing & Quality Assurance
- ⏳ Phase 6: Documentation & Polish

---

**Note**: This project is under active development. Features and APIs may change before the first stable release.
