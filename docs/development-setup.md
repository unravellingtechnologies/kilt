# Development Setup Guide

This guide will help you set up a development environment for contributing to Kilt.

## Prerequisites

### Required

- **Go 1.21 or later**: [Install Go](https://go.dev/doc/install)
  ```bash
  go version  # Should show 1.21 or later
  ```

- **Git**: For version control and repository operations
  ```bash
  git --version
  ```

- **Make**: For running build tasks (usually pre-installed on macOS/Linux)
  ```bash
  make --version
  ```

### Optional but Recommended

- **golangci-lint**: For code linting (can be installed via `make lint-install`)
- **goimports**: For import formatting (install with `go install golang.org/x/tools/cmd/goimports@latest`)

## Initial Setup

### 1. Clone the Repository

```bash
# Clone your fork (or the main repository)
git clone https://github.com/your-username/kilt.git
cd kilt
```

### 2. Install Dependencies

```bash
# Download and verify dependencies
make deps

# This runs:
# - go mod download
# - go mod tidy
```

### 3. Install Development Tools

```bash
# Install golangci-lint (if not already installed)
make lint-install

# Install goimports (optional but recommended)
go install golang.org/x/tools/cmd/goimports@latest
```

### 4. Verify Setup

```bash
# Build the project
make build

# Run tests
make test

# Check code formatting
make fmt-check

# Run linter
make lint
```

If all commands succeed, your development environment is ready!

## Project Structure

```
kilt/
├── cmd/kilt/              # CLI entry point and commands
│   ├── main.go           # Main entry point
│   └── commands/         # Command implementations
├── internal/              # Internal packages (not for external use)
│   ├── core/             # Core engine (config, state, backup, templates)
│   ├── plugin/           # Plugin system (registry, loader, interfaces)
│   └── plugins/           # Built-in plugins
├── test/                  # Test fixtures and integration tests
│   ├── fixtures/         # Test data and sample configs
│   └── integration/      # End-to-end tests
├── docs/                  # Documentation
├── scripts/               # Build and install scripts
├── Makefile              # Build automation
├── go.mod                # Go module definition
└── README.md             # Project documentation
```

## Development Workflow

### Building

```bash
# Build for current platform
make build

# Build for all supported platforms
make build-all

# Build and install locally
make install
```

The binary will be created in `bin/kilt` (or `bin/kilt-<platform>` for cross-compilation).

### Running Locally

```bash
# Build and run
make run

# Or run directly
./bin/kilt --help
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests
make test-integration

# Run specific test package
go test -v ./internal/core/...

# Run specific test
go test -v ./internal/core/... -run TestLoadConfig
```

### Code Quality Checks

```bash
# Format code
make fmt

# Check formatting (without modifying)
make fmt-check

# Run go vet
make vet

# Run linter
make lint

# Run all checks (fmt-check, vet, lint, test)
make check
```

## IDE Setup

### VS Code

Recommended extensions:
- **Go** (official Go extension)
- **golangci-lint** (for linting integration)

Settings (`.vscode/settings.json`):
```json
{
  "go.formatTool": "goimports",
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "go.testFlags": ["-v", "-race"],
  "editor.formatOnSave": true
}
```

### GoLand / IntelliJ IDEA

1. Install the Go plugin
2. Configure `golangci-lint` as the linter
3. Enable format on save
4. Configure test runner to use `-v -race` flags

### Vim / Neovim

Recommended plugins:
- `vim-go` for Go support
- `nvim-lspconfig` with `gopls` for LSP
- `null-ls` for formatting and linting

## Environment Variables

Kilt doesn't require any environment variables for development, but you can set:

- `GOOS`: Target operating system (e.g., `darwin`, `linux`)
- `GOARCH`: Target architecture (e.g., `amd64`, `arm64`)
- `CGO_ENABLED`: Enable/disable CGO (default: auto)

Example:
```bash
GOOS=linux GOARCH=amd64 make build
```

## Debugging

### Using Delve (dlv)

```bash
# Install Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug the binary
dlv exec ./bin/kilt -- sync

# Set breakpoints and debug
```

### Using VS Code

1. Create `.vscode/launch.json`:
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Kilt",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/kilt",
      "args": ["sync"],
      "env": {}
    }
  ]
}
```

2. Set breakpoints and press F5 to start debugging

### Verbose Logging

Use the `--verbose` flag for detailed output:

```bash
./bin/kilt sync --verbose
```

## Testing with Real Data

### Using Test Fixtures

Test fixtures are located in `test/fixtures/`:

```bash
# Copy fixture repository for testing
cp -r test/fixtures/repo ~/test-dotfiles

# Point kilt to test repository
./bin/kilt init ~/test-dotfiles
./bin/kilt sync
```

### Creating Test Configurations

Create test configurations in `test/fixtures/configs/`:

```yaml
# test/fixtures/configs/my-test.yaml
dotfiles_repo: "https://github.com/test/dotfiles"
dotfiles:
  - zsh
  - git
```

## Common Issues

### "command not found: golangci-lint"

Install it:
```bash
make lint-install
```

Or manually:
```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2
```

### "goimports: command not found"

Install it:
```bash
go install golang.org/x/tools/cmd/goimports@latest
```

### Tests Fail with "git: command not found"

Ensure Git is installed and in your PATH:
```bash
which git
```

### Permission Errors

Some tests modify file permissions. Ensure you have write permissions to temporary directories. On macOS/Linux, you may need to adjust umask:

```bash
umask 0022
```

### Import Errors

If you see import errors, ensure dependencies are downloaded:

```bash
go mod download
go mod tidy
```

### Build Errors

If build fails, try:
```bash
# Clean and rebuild
make clean
make deps
make build
```

## Performance Profiling

### CPU Profiling

```bash
# Build with profiling enabled
go build -o bin/kilt ./cmd/kilt

# Run with CPU profiling
./bin/kilt sync -cpuprofile=cpu.prof

# Analyze
go tool pprof bin/kilt cpu.prof
```

### Memory Profiling

```bash
# Run with memory profiling
./bin/kilt sync -memprofile=mem.prof

# Analyze
go tool pprof bin/kilt mem.prof
```

## Continuous Integration

The project uses GitHub Actions for CI. See `.github/workflows/ci.yml` for configuration.

CI runs:
- Code formatting checks
- Linting
- Unit tests
- Integration tests
- Coverage reporting

You can run the same checks locally:
```bash
make check
```

## Next Steps

- Read [CONTRIBUTING.md](../CONTRIBUTING.md) for contribution guidelines
- Review [docs/code-style.md](code-style.md) for code standards
- Check [docs/plugin-guide.md](plugin-guide.md) if developing plugins
- See [docs/testing-strategy.md](testing-strategy.md) for testing guidelines

## Getting Help

- Check existing GitHub issues
- Review the documentation
- Ask questions in GitHub Discussions
- Open an issue for bugs or feature requests

Happy coding! 🚀

