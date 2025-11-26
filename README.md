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
- **Files Plugin**: File placement with source → target mapping
- **Alternates Plugin**: OS/hostname-based file selection
- **Directories Plugin**: Ensure directories exist with proper permissions
- **Run Once Plugin**: Execute bootstrap scripts exactly once
- **On Change Plugin**: Run commands when files change
- **Git Plugin**: Manage bare repository and clone extra repos
- **Brew Plugin**: Homebrew integration for package management
- **1Password Plugin**: Secret injection from 1Password CLI

### 🎨 Template Support
Full Go template support with built-in variables:
- System info: `{{ .hostname }}`, `{{ .os }}`, `{{ .arch }}`, `{{ .user }}`, `{{ .home }}`
- Environment variables: `{{ env "VAR" }}`
- 1Password secrets: `{{ op "path/to/secret" }}`
- Custom data from `data.toml`

### 🛡️ Safety Features
- Automatic backups before file modifications
- Dry-run mode to preview changes
- Rollback capability on failure
- State locking for concurrent execution safety

## Project Status

**Status**: 🚧 In Development (Phase 1: Project Foundation)

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

Kilt uses TOML configuration files. Create a `config.toml` in your dotfiles repository:

```toml
# Example configuration
[[files]]
source = "zsh/zshrc"
target = "~/.zshrc"
template = false

[[files]]
source = "gitconfig.tmpl"
target = "~/.gitconfig"
template = true

directories = [
    "~/dev/personal",
    "~/dev/work"
]

run_once = [
    "scripts/install_homebrew.sh"
]
```

See [architecture.md](architecture.md) for the complete configuration schema.

## Commands

- `kilt init <repo-url>` — Initialize from Git repository
- `kilt sync` — Pull latest and apply changes
- `kilt doctor` — Validate setup and dependencies
- `kilt version` — Show version information
- `kilt restore <timestamp>` — Restore from backup
- `kilt reset` — Clear state and start fresh

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

- ✅ Phase 1: Project Foundation & Core Architecture (in progress)
- ⏳ Phase 2: Core Engine Features
- ⏳ Phase 3: Core Plugins
- ⏳ Phase 4: Installation & Distribution
- ⏳ Phase 5: Testing & Quality Assurance
- ⏳ Phase 6: Documentation & Polish

---

**Note**: This project is under active development. Features and APIs may change before the first stable release.
