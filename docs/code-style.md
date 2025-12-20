# Code Style Guidelines

This document outlines the code style guidelines and conventions for the Kilt project. These guidelines ensure consistency, readability, and maintainability across the codebase.

## Formatting

### gofmt and goimports

All code must be formatted using `gofmt` and `goimports`. Run:

```bash
make fmt
```

Or check formatting without modifying files:

```bash
make fmt-check
```

**Rules:**
- Use tabs for indentation (go fmt default)
- One blank line between top-level declarations
- Group imports with blank lines: standard library, third-party, local
- Local imports use prefix: `github.com/unravelling/kilt`

**Example:**
```go
package main

import (
	"fmt"
	"os"
	
	"github.com/spf13/cobra"
	
	"github.com/unravelling/kilt/internal/core"
	"github.com/unravelling/kilt/internal/plugin"
)
```

### gofumpt

We use `gofumpt` (via golangci-lint) for stricter formatting:
- Remove extra blank lines
- Simplify complex expressions
- Enforce consistent spacing

## Naming Conventions

### Exported Identifiers

- Use PascalCase: `Config`, `Registry`, `Execute()`
- Be descriptive: `LoadConfiguration()` not `LoadConfig()` when context is needed
- Acronyms are all uppercase: `HTTPClient`, `JSONMarshal` (not `HttpClient`, `JsonMarshal`)

### Unexported Identifiers

- Use camelCase: `config`, `pluginRegistry`, `execute()`
- Prefix with package name if needed to avoid ambiguity: `pluginLoader`, `stateManager`

### Variables

- Use descriptive names that reveal intent
- Prefer full words over abbreviations unless widely understood (e.g., `id`, `url`)
- Boolean variables should be clear: `isReady`, `hasError`, `shouldRetry`

### Functions

- Use verbs for functions that do something: `Load()`, `Execute()`, `Validate()`
- Use nouns for getters: `Name()`, `Version()`
- Getters that return booleans: `IsValid()`, `HasError()`

### Error Variables

- Use `Err` prefix: `ErrNotFound`, `ErrInvalidConfig`
- Use descriptive names: `ErrPluginNotFound` not `ErrNotFound` when context is needed

```go
var (
	ErrInvalidConfig = errors.New("invalid configuration")
	ErrPluginNotFound = errors.New("plugin not found")
)
```

### Test Functions

- Use `Test` prefix: `TestLoadConfig`
- Use descriptive names: `TestLoadConfig_InvalidYAML`
- For subtests, use table-driven format when >3 cases

```go
func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid config", "config.yaml", false},
		{"invalid YAML", "invalid.yaml", true},
	}
	// ...
}
```

## Code Organization

### Package Structure

- Keep packages focused and cohesive
- Use `internal/` for private packages
- One package per directory
- Package name should match directory name (lowercase, no underscores)

### File Organization

Within a package, organize in this order:
1. Package declaration
2. Imports
3. Constants
4. Variables
5. Types
6. Functions (constructors first, then methods)

### Function Size

- Functions should be small and focused (<50 lines when reasonable)
- Extract helpers for repeated logic
- Use early returns to reduce nesting

### Complexity

- Keep cyclomatic complexity low (<15)
- Extract complex logic into helper functions
- Use table-driven tests for complex validation

## Error Handling

### Error Wrapping

Always wrap errors with context using `fmt.Errorf()`:

```go
if err != nil {
	return fmt.Errorf("failed to load config: %w", err)
}
```

### Error Checking

- Always check errors explicitly
- Don't ignore errors with `_`
- Use `errors.Is()` and `errors.As()` for error inspection

```go
if errors.Is(err, os.ErrNotExist) {
	// handle not found
}
```

### Error Messages

- Use clear, user-friendly messages
- Include context: what operation failed and why
- Provide hints when appropriate

```go
return fmt.Errorf("configuration validation failed: %w\nHint: Check your .kilt/config.yaml file", err)
```

## Logging

### Structured Logging

- Use `slog` (stdlib) or `zap` for structured logging
- Never use `fmt.Println()` for application output
- Use appropriate log levels: debug, info, warn, error

```go
logger.Info("Plugin executed", "plugin", p.Name(), "duration", duration)
logger.Error("Failed to execute plugin", "error", err, "plugin", p.Name())
```

### Console Output

- For user-facing output, use `fmt.Fprintf(os.Stderr, ...)` or `fmt.Fprintf(os.Stdout, ...)`
- Use coloured output only when appropriate (respect `--no-colour` flag)

## Testing

### Test Organization

- Tests in `_test.go` files alongside source
- Use table-driven tests for >3 test cases
- Use subtests (`t.Run()`) for complex test scenarios

### Test Naming

- Test functions: `TestFunctionName`
- Subtests: descriptive names like `TestFunctionName_Condition`

### Assertions

- Use `testify/require` for assertions that should stop the test
- Use `testify/assert` for assertions that can continue
- Prefer `require` for critical checks

```go
require.NoError(t, err)
assert.Equal(t, expected, actual)
```

### Test Helpers

- Mark test helpers with `t.Helper()`
- Keep test helpers simple and focused

```go
func setupTestEnv(t *testing.T) *TestEnvironment {
	t.Helper()
	// setup code
}
```

## Documentation

### Package Documentation

Every package should have a package-level comment:

```go
// Package core provides the core engine and configuration management
// for the Kilt dotfiles manager.
package core
```

### Exported Functions

All exported functions, types, and variables should have doc comments:

```go
// LoadConfig loads and parses a YAML configuration file.
// It expands ~ in paths and validates the configuration structure.
func LoadConfig(path string) (*Config, error) {
	// ...
}
```

### Comments

- Comments should explain **why**, not **what**
- Use complete sentences
- Start with the name of what you're documenting

```go
// CalculateChecksum computes the SHA256 checksum of the file.
// We use SHA256 for compatibility with older systems that don't support SHA512.
func CalculateChecksum(path string) (string, error) {
	// ...
}
```

## Type Safety

### Explicit Types

- Use explicit return types on public functions
- Avoid `any` type; use specific types or interfaces
- Prefer interfaces over concrete types when appropriate

### Null Safety

- Handle nil checks explicitly
- Return errors rather than panic on nil input
- Use pointers only when needed

## Performance

### Allocations

- Prefer slice pre-allocation when size is known
- Reuse buffers when possible
- Profile before optimizing

```go
// Good: pre-allocate
result := make([]string, 0, len(items))
for _, item := range items {
	result = append(result, item)
}
```

### Context Usage

- Use `context.Context` as first parameter in cancellable functions
- Propagate context through call chains
- Respect cancellation and timeouts

```go
func Execute(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// continue
	}
}
```

## Linting

We use `golangci-lint` with strict configuration. Run:

```bash
make lint
```

Common linters enabled:
- `gofmt` - Code formatting
- `goimports` - Import formatting
- `govet` - Go vet checks
- `staticcheck` - Static analysis
- `errcheck` - Error checking
- `gosimple` - Code simplification
- `gocyclo` - Complexity checking
- `revive` - Style checking
- `gosec` - Security checks

## Pre-commit Checks

Before committing, run:

```bash
make check
```

This runs:
- `fmt-check` - Verify formatting
- `vet` - Go vet checks
- `lint` - golangci-lint
- `test` - All tests

## CI Integration

All code is automatically linted in CI. The build will fail if:
- Code is not formatted
- Linter errors are found
- Tests fail

## Additional Resources

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [golangci-lint Documentation](https://golangci-lint.run/)





