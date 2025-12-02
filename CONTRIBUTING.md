# Contributing to Kilt

Thank you for your interest in contributing to Kilt! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Development Workflow](#development-workflow)
- [Code Standards](#code-standards)
- [Testing Requirements](#testing-requirements)
- [Submitting Changes](#submitting-changes)
- [Documentation](#documentation)
- [Questions and Support](#questions-and-support)

## Code of Conduct

This project adheres to a code of conduct that all contributors are expected to follow. Please be respectful, inclusive, and constructive in all interactions.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/your-username/kilt.git
   cd kilt
   ```
3. **Add the upstream remote**:
   ```bash
   git remote add upstream https://github.com/unravelling/kilt.git
   ```
4. **Create a branch** for your changes:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Setup

See [docs/development-setup.md](docs/development-setup.md) for detailed setup instructions.

### Quick Start

1. **Install Go 1.21+**: Ensure you have Go 1.21 or later installed
   ```bash
   go version
   ```

2. **Install dependencies**:
   ```bash
   make deps
   ```

3. **Install development tools**:
   ```bash
   make lint-install  # Installs golangci-lint
   ```

4. **Build the project**:
   ```bash
   make build
   ```

5. **Run tests**:
   ```bash
   make test
   ```

## Development Workflow

### 1. Before You Start

- Check existing issues and pull requests to avoid duplicate work
- For large changes, consider opening an issue first to discuss the approach
- Ensure you're working on the latest `main` branch

### 2. Making Changes

1. **Create a feature branch** from `main`:
   ```bash
   git checkout main
   git pull upstream main
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following the [Code Standards](#code-standards)

3. **Write tests** for your changes (see [Testing Requirements](#testing-requirements))

4. **Run checks** before committing:
   ```bash
   make check  # Runs fmt-check, vet, lint, and test
   ```

5. **Commit your changes** with clear, descriptive messages:
   ```bash
   git add .
   git commit -m "Add feature: description of what you added"
   ```

### 3. Commit Message Guidelines

Follow these guidelines for commit messages:

- **Use imperative mood**: "Add feature" not "Added feature"
- **Keep first line under 72 characters**
- **Include context in body** for non-obvious changes:
  ```
  Add timeout configuration to runonce plugin

  Why: Scripts were hanging indefinitely on slow systems.
  What: Added configurable timeout with 5m default.
  ```

- **Reference issues**: Use `Fixes #123` or `Closes #123` when applicable

### 4. Keeping Your Branch Updated

Regularly sync your branch with upstream:

```bash
git fetch upstream
git rebase upstream/main
```

## Code Standards

### Formatting

All code must be formatted using `gofmt` and `goimports`:

```bash
make fmt
```

Or check formatting without modifying files:

```bash
make fmt-check
```

### Linting

We use `golangci-lint` with strict configuration. Run:

```bash
make lint
```

Common issues to avoid:
- Unused imports or variables
- Missing error handling
- Inefficient code patterns
- Security vulnerabilities

### Code Style

See [docs/code-style.md](docs/code-style.md) for detailed code style guidelines.

Key principles:
- **Explicit over implicit**: Use explicit types, return values, and error handling
- **Small functions**: Keep functions under 50 lines when reasonable
- **Single responsibility**: Each function should do one thing
- **Meaningful names**: Use descriptive, intention-revealing names
- **No duplication**: Extract helpers for repeated logic

### Documentation

- **Package comments**: Every package must have a package-level comment
- **Exported functions**: All exported functions, types, and variables need doc comments
- **Comments explain why**: Focus on the "why" not the "what"
- **Update docs**: When changing public APIs, CLI commands, or configuration, update relevant documentation

Example:

```go
// Package core provides the core engine and configuration management
// for the Kilt dotfiles manager.
package core

// LoadConfig loads and parses a YAML configuration file.
// It expands ~ in paths and validates the configuration structure.
// Returns an error if the file cannot be read or parsed.
func LoadConfig(path string) (*Config, error) {
    // ...
}
```

## Testing Requirements

See [docs/testing-strategy.md](docs/testing-strategy.md) for detailed testing guidelines.

### General Requirements

- **All new code must have tests**
- **Bug fixes must include regression tests**
- **Aim for >80% code coverage** on new code
- **Tests must pass** before submitting a PR

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests
make test-integration

# Run specific test
go test -v ./internal/core/... -run TestLoadConfig
```

### Test Guidelines

- Use table-driven tests for >3 test cases
- Use `testify/require` for critical assertions
- Use `testify/assert` for non-critical checks
- Mark test helpers with `t.Helper()`
- Use descriptive test names: `TestLoadConfig_InvalidYAML`

### Writing Tests

Example test structure:

```go
func TestLoadConfig(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {
            name:    "valid config",
            input:   "testdata/config.yaml",
            wantErr: false,
        },
        {
            name:    "invalid YAML",
            input:   "testdata/invalid.yaml",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cfg, err := LoadConfig(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                assert.Nil(t, cfg)
            } else {
                require.NoError(t, err)
                assert.NotNil(t, cfg)
            }
        })
    }
}
```

## Submitting Changes

### Pull Request Process

1. **Push your branch** to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Create a Pull Request** on GitHub:
   - Use a clear, descriptive title
   - Fill out the PR template completely
   - Reference related issues
   - Include a description of what changed and why

3. **Ensure CI passes**: All checks must pass before review

4. **Respond to feedback**: Address review comments promptly

### Pull Request Checklist

Before submitting, ensure:

- [ ] Code follows style guidelines (`make fmt-check` passes)
- [ ] All tests pass (`make test`)
- [ ] Linter passes (`make lint`)
- [ ] Code is documented (godoc comments for exported symbols)
- [ ] Tests are added/updated for new/changed functionality
- [ ] Commit messages follow guidelines
- [ ] Branch is up to date with `main`
- [ ] No merge conflicts

### Review Process

- PRs require at least one approval before merging
- All CI checks must pass
- Code review focuses on:
  - Correctness and completeness
  - Code quality and maintainability
  - Test coverage
  - Documentation
  - Performance implications

### After Approval

Once approved, maintainers will:
- Squash and merge your commits
- Update the changelog if needed
- Tag releases when appropriate

## Documentation

### When to Update Documentation

Update documentation when you:
- Add new features or commands
- Change existing behaviour
- Modify configuration options
- Add or remove dependencies
- Change the project structure

### Documentation Files

- **README.md**: User-facing documentation, installation, usage
- **CONTRIBUTING.md**: This file - contribution guidelines
- **docs/**: Developer documentation
  - `architecture.md`: System architecture and design decisions
  - `plugin-guide.md`: Plugin development guide
  - `code-style.md`: Code style guidelines
  - `development-setup.md`: Development environment setup
  - `testing-strategy.md`: Testing guidelines and patterns
  - `releasing.md`: Release process

### Writing Documentation

- Use clear, concise language
- Include code examples where helpful
- Keep documentation up to date with code
- Use proper markdown formatting
- Add diagrams for complex concepts

## Plugin Development

If you're developing a new plugin, see [docs/plugin-guide.md](docs/plugin-guide.md) for detailed instructions.

Key points:
- Plugins must implement the `Plugin` interface
- Plugins must be idempotent
- Plugins must respect dry-run mode
- Plugins must have comprehensive tests

## Questions and Support

- **GitHub Issues**: For bug reports and feature requests
- **GitHub Discussions**: For questions and general discussion
- **Pull Requests**: For code contributions

## Recognition

Contributors will be:
- Listed in the project's contributors
- Credited in release notes for significant contributions
- Acknowledged in the project documentation

Thank you for contributing to Kilt! 🎉

