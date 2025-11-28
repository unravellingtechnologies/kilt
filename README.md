# Kilt

**Bootstrap manager for Mac. Think dot files management, but with a beard**

A Git-first, curl-installable bootstrapper that transforms a fresh macOS (or Linux) system into a personalized development environment using a single command. Kilt emphasizes idempotency, safety, and extensibility through a plugin architecture.

## Vision

One command to turn any fresh Mac into *your* Mac in under 5 minutes — completely from a single Git repository, with zero secrets ever touching Git, full idempotency, and the flexibility to manage arbitrary files in arbitrary locations.

## Quick Start

### Installation

**One-line install** (recommended):
```bash
curl -sL https://get.kilt.pro | bash -s -- https://github.com/you/dotfiles
```

This command will:
1. Download and install Kilt
2. Clone your dotfiles repository
3. Set up the Kilt directory structure
4. Apply your dotfiles configuration

**Manual installation**:
```bash
# Download the binary (replace with latest version)
curl -L https://github.com/unravelling/kilt/releases/latest/download/kilt-darwin-arm64 -o /usr/local/bin/kilt
chmod +x /usr/local/bin/kilt

# Initialize with your repository
kilt init https://github.com/you/dotfiles

# Sync your dotfiles
kilt sync
```

**Build from source**:
```bash
git clone https://github.com/unravelling/kilt.git
cd kilt
make build
make install
```

### First-Time Setup

1. **Create a dotfiles repository** (if you don't have one):
   ```bash
   mkdir -p ~/dotfiles
   cd ~/dotfiles
   git init
   mkdir -p .kilt
   ```

2. **Create a configuration file**:
   ```yaml
   # .kilt/config.yaml
   dotfiles_repo: "https://github.com/yourusername/dotfiles"
   dotfiles_path: "~/.dotfiles"
   
   dotfiles:
     - zsh
     - git
   ```

3. **Commit and push**:
   ```bash
   git add .
   git commit -m "Initial Kilt configuration"
   git remote add origin https://github.com/yourusername/dotfiles.git
   git push -u origin main
   ```

4. **Install and initialize**:
   ```bash
   curl -sL https://get.kilt.pro | bash -s -- https://github.com/yourusername/dotfiles
   ```

5. **Sync your dotfiles**:
   ```bash
   kilt sync
   ```

See [Example Configurations](docs/examples/) for more configuration examples.

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
- **On Change Plugin** ✅: Execute commands when configuration or files change
- **Git Plugin** ✅: Manage bare repository and clone extra repos
- **Brew Plugin** ✅: Homebrew integration for package management with auto-installation
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

**In Progress**: Phase 6 - Documentation & Polish

This project is in active development. See [tasks.md](tasks.md) for the full development roadmap and [architecture.md](architecture.md) for detailed architecture documentation.

## Requirements

### Runtime Requirements

- **Git** - For repository operations
- **Bash** - For installer script (only needed for installation)
- **macOS or Linux** - Supported operating systems

### Build Requirements (for building from source)

- **Go 1.21+** - For building from source
- **Make** - For build automation (optional, can use `go build` directly)

### Optional Dependencies

- **Homebrew** - For Brew plugin (can be auto-installed)
- **1Password CLI** - For 1Password plugin (for secret management)

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
    auto_install: true   # Optional: automatically install Homebrew if not found
    brewfile: "Brewfile" # Optional: path to Brewfile (default: "Brewfile")
    auto_update: false   # Optional: run brew update after bundle
    cleanup_after: true  # Optional: run brew cleanup after bundle
    bundles:            # Optional: multiple bundle files
      - bootstrap       # expects brew/bootstrap file
      - dev             # expects brew/dev file
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

### Brew Plugin

The Brew Plugin integrates with Homebrew for package management. It can automatically install Homebrew if it's not found, run `brew bundle` to install packages from Brewfiles, and manage Homebrew updates and cleanup.

**Key Features**:
- **Auto-installation**: Automatically install Homebrew if not found (requires user interaction for sudo password)
- **Brew bundle execution**: Run `brew bundle --file=Brewfile` to install packages
- **Multiple bundles**: Support for multiple bundle files
- **Change detection**: Only runs bundle if Brewfile changed
- **Auto-update**: Optionally run `brew update` after bundle
- **Auto-cleanup**: Optionally run `brew cleanup` after bundle
- **Linux compatibility**: Works with Linuxbrew on Linux systems
- **Dry-run support**: Preview what would be executed without running commands

**Basic Configuration**:

```yaml
plugins:
  brew:
    brewfile: "Brewfile"
```

**With Auto-Installation**:

```yaml
plugins:
  brew:
    auto_install: true   # Install Homebrew if not found (requires user interaction)
    brewfile: "Brewfile"
    auto_update: false
    cleanup_after: true
```

**Multiple Bundles**:

```yaml
plugins:
  brew:
    auto_install: true
    bundles:
      - bootstrap  # Runs brew bundle --file=bootstrap
      - dev        # Runs brew bundle --file=dev
```

**Auto-Installation**:
When `auto_install: true` is set and Homebrew is not found, the plugin will:
1. Download the official Homebrew installer script
2. Execute it (requires user to enter sudo password)
3. Automatically detect the installed Homebrew after installation
4. Continue with bundle execution

**Note**: Auto-installation requires:
- User interaction (sudo password prompt)
- Internet connection
- `curl` command available
- Works on macOS and Linux

**CLI Commands**:

```bash
# Install packages from Brewfile
kilt brew install

# Install Homebrew if not found (requires user interaction)
kilt brew install --force

# Update Homebrew and packages
kilt brew update

# Clean up old Homebrew files
kilt brew cleanup
```

**Brewfile Format**:

```ruby
# Brewfile example
brew "git"
brew "vim"
brew "tmux"
cask "firefox"
cask "visual-studio-code"
```

**Change Detection**:
- The plugin only runs `brew bundle` if the Brewfile has changed
- Uses state checksums to detect changes
- Multiple bundles always run (change detection skipped for bundles)

**Error Handling**:
- Gracefully skips if Homebrew is not installed (unless `auto_install: true`)
- Installation failures return clear error messages
- Bundle failures stop execution and trigger rollback

See [architecture.md](architecture.md) for the complete configuration schema.

For more configuration examples, see [docs/examples/](docs/examples/).

## Commands

Kilt provides a comprehensive set of commands for managing your dotfiles. Here's a quick overview:

### Core Commands

- **`kilt init <repo-url>`** - Initialize from Git repository
- **`kilt sync`** - Synchronize dotfiles bidirectionally
- **`kilt doctor`** - Validate setup and dependencies
- **`kilt version`** - Show version information

### Backup Commands

- **`kilt restore <backup-id>`** - Restore files from a backup
- **`kilt backups list`** - List all available backups
- **`kilt reset`** - Clear state and start fresh

### Plugin Commands

- **`kilt brew install`** - Install packages from Brewfile
- **`kilt brew update`** - Update Homebrew and packages
- **`kilt brew cleanup`** - Clean up old Homebrew files

### Utility Commands

- **`kilt completion <shell>`** - Generate shell completion scripts

### Global Flags

All commands support these global flags:

- `--dry-run` - Show what would happen without executing
- `--verbose, -v` - Detailed logging output
- `--debug` - Enable debug logging (includes verbose, shows timestamps)
- `--config <path>` - Custom config file location
- `--no-color` - Disable colored output
- `--force` - Skip confirmation prompts

### Logging and Output

Kilt provides structured logging with color support:

- **Colorized output**: Success messages (green), errors (red), warnings (yellow), info (blue)
- **Log levels**: Debug, Info, Warn, Error
- **Progress indicators**: Progress bars for long operations, spinners for indeterminate tasks
- **Error messages**: Helpful error messages with suggestions and context
- **Respects `--no-color`**: Automatically disables colors when output is not a terminal

Example output:
```
INFO Syncing dotfiles...
✓ Successfully synced 15 files
WARN Some files were skipped (unchanged)
ERROR Failed to sync file: permission denied
Suggestion: Check file permissions with 'ls -la ~/.zshrc'
```

### Examples

```bash
# Initialize from repository
kilt init https://github.com/yourusername/dotfiles

# Sync dotfiles
kilt sync

# Preview changes
kilt sync --dry-run

# Validate setup
kilt doctor

# List backups
kilt backups list

# Restore from backup
kilt restore 20250125-143022

# Install Homebrew packages
kilt brew install
```

For complete command documentation, see [Command Reference](docs/commands.md).

## Development

### Code Style

This project follows strict Go code style guidelines. See [docs/code-style.md](docs/code-style.md) for details.

**Quick Start:**
```bash
make fmt        # Format code
make fmt-check  # Check formatting without modifying
make lint       # Run linters
make check      # Run all checks (fmt-check, vet, lint, test)
```

### Git Hooks

Pre-commit hooks are available to automatically check code quality. See [.githooks/README.md](.githooks/README.md) for installation instructions.

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

## Documentation

Comprehensive documentation is available in the `docs/` directory:

- **[Command Reference](docs/commands.md)** - Complete command documentation
- **[Plugin Documentation](docs/plugins.md)** - Detailed plugin guides
- **[Example Configurations](docs/examples/)** - Configuration examples
- **[Troubleshooting Guide](docs/troubleshooting.md)** - Common issues and solutions
- **[Migration Guide](docs/migration.md)** - Migrating from other tools
- **[Security Audit](docs/security-audit.md)** - Security considerations and audit results
- **[Architecture Documentation](architecture.md)** - System architecture
- **[Developer Documentation](CONTRIBUTING.md)** - Contributing guidelines

## Contributing

Contributions are welcome! Please read the [Contributing Guide](CONTRIBUTING.md) before submitting PRs.

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run `make check` to ensure quality
5. Submit a pull request

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

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
