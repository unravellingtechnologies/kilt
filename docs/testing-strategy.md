# Testing Strategy

This document outlines the testing strategy, patterns, and best practices for the Kilt project.

## Overview

Kilt uses a comprehensive testing approach with multiple test levels:

- **Unit Tests** (80%): Test individual components in isolation
- **Integration Tests** (15%): Test component interactions and workflows
- **End-to-End Tests** (5%): Test complete user workflows

## Test Organization

### Directory Structure

```
kilt/
├── internal/
│   ├── core/
│   │   ├── config.go
│   │   ├── config_test.go      # Unit tests alongside source
│   │   └── ...
│   └── plugins/
│       └── files/
│           ├── plugin.go
│           └── plugin_test.go
└── test/
    ├── fixtures/                # Test data
    │   ├── configs/            # Sample configurations
    │   ├── repos/              # Sample repositories
    │   └── states/             # Sample state files
    └── integration/            # Integration tests
        ├── init_sync_test.go
        ├── plugins_test.go
        └── helpers.go
```

### Test File Naming

- Unit tests: `*_test.go` alongside source files
- Integration tests: `*_test.go` in `test/integration/`
- Test fixtures: Descriptive names in `test/fixtures/`

## Unit Testing

### Principles

- **Isolation**: Each test is independent and can run in any order
- **Fast**: Unit tests should run in milliseconds
- **Deterministic**: Same input always produces same output
- **Focused**: Test one thing at a time

### Test Structure

Use table-driven tests for multiple test cases:

```go
func TestLoadConfig(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
        want    *Config
    }{
        {
            name:    "valid config",
            input:    "testdata/valid.yaml",
            wantErr:  false,
            want:     &Config{DotfilesRepo: "https://example.com/repo"},
        },
        {
            name:    "invalid YAML",
            input:    "testdata/invalid.yaml",
            wantErr:  true,
            want:     nil,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := LoadConfig(tt.input)
            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
                assert.Equal(t, tt.want, got)
            }
        })
    }
}
```

### Assertions

Use `testify` for assertions:

- **`require`**: Stops test on failure (use for critical checks)
- **`assert`**: Continues test on failure (use for non-critical checks)

```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestExample(t *testing.T) {
    // Critical: test fails if this errors
    result, err := DoSomething()
    require.NoError(t, err)
    require.NotNil(t, result)

    // Non-critical: test continues if this fails
    assert.Equal(t, "expected", result.Value)
    assert.Contains(t, result.Message, "key")
}
```

### Test Helpers

Mark test helpers with `t.Helper()`:

```go
func setupTestConfig(t *testing.T) *Config {
    t.Helper()  // Marks this as a test helper
    
    cfg := &Config{
        DotfilesRepo: "https://test.com/repo",
    }
    return cfg
}
```

### Mocking

#### Filesystem Mocking

Use `afero` for filesystem abstraction:

```go
import "github.com/spf13/afero"

func TestFileOperation(t *testing.T) {
    // Create in-memory filesystem
    fs := afero.NewMemMapFs()
    
    // Create test file
    afero.WriteFile(fs, "/test/file.txt", []byte("content"), 0644)
    
    // Test with mock filesystem
    manager := NewFileManager(fs)
    err := manager.ProcessFile("/test/file.txt")
    require.NoError(t, err)
}
```

#### External Commands

Mock external commands using variables:

```go
// In source file
var execCommand = exec.Command

func RunGitCommand(args ...string) error {
    cmd := execCommand("git", args...)
    return cmd.Run()
}

// In test file
func TestGitCommand(t *testing.T) {
    defer func() { execCommand = exec.Command }()  // Restore
    
    execCommand = func(name string, args ...string) *exec.Cmd {
        return exec.Command("echo", "mock output")
    }
    
    err := RunGitCommand("status")
    require.NoError(t, err)
}
```

### Test Fixtures

Store test data in `test/fixtures/`:

```go
func TestWithFixture(t *testing.T) {
    fixturePath := filepath.Join("test", "fixtures", "configs", "minimal.yaml")
    data, err := os.ReadFile(fixturePath)
    require.NoError(t, err)
    
    cfg, err := LoadConfigFromBytes(data)
    require.NoError(t, err)
    assert.NotNil(t, cfg)
}
```

## Integration Testing

### Purpose

Integration tests verify:
- Component interactions
- Complete workflows
- Real-world scenarios
- Error handling and recovery

### Test Environment

Use `SetupTestEnvironment()` for isolated test environments:

```go
func TestInitSyncFlow(t *testing.T) {
    env := SetupTestEnvironment(t)
    defer env.Cleanup(t)
    
    // Test code here
    // Environment is automatically cleaned up
}
```

See [test/integration/README.md](../test/integration/README.md) for details.

### Integration Test Patterns

#### Full Workflow Tests

```go
func TestFullSyncWorkflow(t *testing.T) {
    env := SetupTestEnvironment(t)
    defer env.Cleanup(t)
    
    // 1. Initialise
    err := env.InitialiseRepo()
    require.NoError(t, err)
    
    // 2. Create engine
    engine := env.CreateEngine(t)
    
    // 3. Execute
    result, err := engine.Execute()
    require.NoError(t, err)
    
    // 4. Verify
    assert.True(t, result.Success)
    assert.FileExists(t, env.HomeDir+"/.zshrc")
}
```

#### Plugin Interaction Tests

```go
func TestPluginDependencies(t *testing.T) {
    env := SetupTestEnvironment(t)
    defer env.Cleanup(t)
    
    // Test that plugins execute in correct order
    engine := env.CreateEngine(t)
    result, err := engine.Execute()
    require.NoError(t, err)
    
    // Verify execution order
    assert.True(t, executedBefore("git", "files", result.PluginsRun))
}
```

#### Error Scenario Tests

```go
func TestErrorRecovery(t *testing.T) {
    env := SetupTestEnvironment(t)
    defer env.Cleanup(t)
    
    // Create invalid configuration
    env.CreateInvalidConfig(t)
    
    engine := env.CreateEngine(t)
    result, err := engine.Execute()
    
    // Should handle error gracefully
    require.Error(t, err)
    assert.True(t, result.RolledBack)
}
```

## Running Tests

### All Tests

```bash
# Run all tests
make test

# Run with race detection
go test -race ./...

# Run with coverage
make test-coverage
```

### Specific Tests

```bash
# Run tests in specific package
go test -v ./internal/core/...

# Run specific test
go test -v ./internal/core/... -run TestLoadConfig

# Run tests matching pattern
go test -v ./internal/... -run "TestLoad.*"
```

### Integration Tests

```bash
# Run all integration tests
make test-integration

# Run specific integration test
go test -v ./test/integration/... -run TestInitSyncFlow
```

### Test Flags

Common test flags:

- `-v`: Verbose output
- `-race`: Enable race detector
- `-cover`: Show coverage
- `-coverprofile=file.out`: Generate coverage profile
- `-count=1`: Disable test caching
- `-timeout=30s`: Set test timeout

## Coverage

### Target

- **Overall coverage**: >80%
- **New code**: >90%
- **Critical paths**: 100%

### Generating Coverage

```bash
# Generate coverage report
make test-coverage

# View in browser
open coverage.html

# Generate coverage for specific package
go test -coverprofile=coverage.out ./internal/core/...
go tool cover -html=coverage.out
```

### Coverage Exclusions

Exclude generated code and test utilities:

```go
// +build ignore

package main

// This file is excluded from coverage
```

Or use build tags:

```go
//go:build !test

package main
```

## Test Data Management

### Fixtures

Store reusable test data in `test/fixtures/`:

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
│       └── zshrc
└── states/
    ├── empty.json
    └── with-history.json
```

### Temporary Files

Use `t.TempDir()` for temporary files:

```go
func TestWithTempFile(t *testing.T) {
    tmpDir := t.TempDir()  // Automatically cleaned up
    
    filePath := filepath.Join(tmpDir, "test.txt")
    err := os.WriteFile(filePath, []byte("test"), 0644)
    require.NoError(t, err)
}
```

## Best Practices

### 1. Test Naming

Use descriptive names that explain what is being tested:

```go
// Good
func TestLoadConfig_InvalidYAML_ReturnsError(t *testing.T)

// Bad
func TestConfig(t *testing.T)
```

### 2. Test Organization

- One test per behaviour
- Group related tests with subtests
- Use table-driven tests for multiple cases

### 3. Test Independence

- Tests should not depend on execution order
- Each test should set up its own state
- Clean up after tests (use `defer` or `t.Cleanup()`)

### 4. Error Testing

Test both success and failure cases:

```go
func TestOperation(t *testing.T) {
    t.Run("success", func(t *testing.T) {
        result, err := DoOperation(validInput)
        require.NoError(t, err)
        assert.NotNil(t, result)
    })
    
    t.Run("invalid input", func(t *testing.T) {
        result, err := DoOperation(invalidInput)
        require.Error(t, err)
        assert.Nil(t, result)
    })
}
```

### 5. Benchmarking

Add benchmarks for performance-critical code:

```go
func BenchmarkLoadConfig(b *testing.B) {
    for i := 0; i < b.N; i++ {
        _, err := LoadConfig("testdata/config.yaml")
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

Run benchmarks:

```bash
go test -bench=. -benchmem ./internal/core/...
```

## CI Integration

Tests run automatically in CI (GitHub Actions):

- All unit tests
- All integration tests
- Coverage reporting
- Linting

See `.github/workflows/ci.yml` for configuration.

## Debugging Tests

### Verbose Output

```bash
go test -v ./internal/core/...
```

### Debug Specific Test

```bash
# Using Delve
dlv test ./internal/core/... -- -test.run TestLoadConfig
```

### Test Timeout

If tests hang, set a timeout:

```bash
go test -timeout=30s ./internal/core/...
```

## Common Issues

### Tests Fail Intermittently

- Check for race conditions: run with `-race`
- Ensure tests are independent
- Check for timing issues

### Tests Fail in CI but Pass Locally

- Check environment differences
- Verify test isolation
- Check for missing dependencies

### Coverage Too Low

- Add tests for uncovered code paths
- Test error cases
- Test edge cases

## Future Improvements

- [ ] Add property-based testing (using `gopter` or similar)
- [ ] Add fuzzing for input validation
- [ ] Add performance regression tests
- [ ] Add concurrency stress tests
- [ ] Add Windows compatibility tests

## Resources

- [Go Testing Documentation](https://go.dev/doc/effective_go#testing)
- [testify Documentation](https://github.com/stretchr/testify)
- [afero Documentation](https://github.com/spf13/afero)
- [Effective Go - Testing](https://go.dev/doc/effective_go#testing)

