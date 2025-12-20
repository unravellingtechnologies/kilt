# Git Hooks

This directory contains git hooks for code quality checks.

## Installation

### Manual Installation

To install the pre-commit hook manually:

```bash
ln -s ../../.githooks/pre-commit .git/hooks/pre-commit
```

### Using git config (Recommended)

Alternatively, you can configure git to use the hooks directory:

```bash
git config core.hooksPath .githooks
```

This makes git automatically use hooks from `.githooks/` directory.

## Available Hooks

### pre-commit

Runs before each commit to ensure:
- Code is properly formatted (`gofmt`, `goimports`)
- Code passes `go vet` checks
- Code passes `golangci-lint` checks (if installed)

**Requirements:**
- `make` command available
- `golangci-lint` installed (optional, will warn if missing)

**To skip hooks (not recommended):**
```bash
git commit --no-verify
```

## Uninstalling

To remove the pre-commit hook:

```bash
rm .git/hooks/pre-commit
```

Or reset the hooks path:

```bash
git config --unset core.hooksPath
```


