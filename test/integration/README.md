# Integration Tests

This directory contains end-to-end integration tests for Kilt that test the complete workflow and plugin interactions.

## Overview

Integration tests simulate real-world usage scenarios by:
- Setting up isolated test environments with temporary directories
- Creating sample dotfiles repositories with realistic configurations
- Testing the full `kilt init` → `kilt sync` workflow
- Verifying plugin interactions and execution order
- Testing error scenarios and recovery

## Test Structure

### Test Files

- `init_sync_test.go` - Tests the complete init → sync workflow
- `plugins_test.go` - Tests plugin interactions and execution
- `errors_test.go` - Tests error handling and recovery scenarios
- `helpers.go` - Test utilities and environment setup

### Test Fixtures

Test fixtures are located in `test/fixtures/repo/` and include:
- Sample `.kilt/config.yaml` configuration file
- Sample dotfiles (zsh, git, ssh)
- Sample scripts for run-once execution
- Directory structures that should be created

## Running Tests

### Run All Integration Tests

```bash
go test -v ./test/integration/...
```

### Run Specific Test

```bash
go test -v ./test/integration/... -run TestInitSyncFlow
```

### Run with Race Detection

```bash
go test -v -race ./test/integration/...
```

### Run with Coverage

```bash
go test -v -coverprofile=coverage.out ./test/integration/...
go tool cover -html=coverage.out
```

## Test Environment

Each test creates an isolated environment using `SetupTestEnvironment()` which:
- Creates a temporary root directory
- Sets up a temporary home directory (`$HOME`)
- Creates `.kilt` directory structure (state, backup)
- Creates dotfiles repository directory
- Sets up git repository for testing

The test environment is automatically cleaned up after each test using `defer env.Cleanup()`.

## Requirements

Integration tests require:
- Git (for repository operations)
- A writable temporary directory
- Sufficient permissions to create files and directories

## Writing New Integration Tests

1. **Use TestEnvironment**: Always use `SetupTestEnvironment(t)` to create an isolated test environment.

```go
func TestMyFeature(t *testing.T) {
    env := SetupTestEnvironment(t)
    defer env.Cleanup(t)
    
    // Your test code here
}
```

2. **Use Fixtures**: Copy the fixture repository when needed:

```go
fixturePath, err := GetFixturePath()
if err != nil {
    t.Fatalf("Failed to get fixture path: %v", err)
}
env.CopyFixtureRepo(t, fixturePath)
```

3. **Clone Repository**: To simulate `kilt init`:

```go
if err := env.CloneRepoToDotfiles(t); err != nil {
    t.Fatalf("Failed to clone repository: %v", err)
}
```

4. **Create Engine**: Use the test environment to create an engine:

```go
registry := plugin.GetDefaultRegistry()
engine := env.CreateEngine(t, registry)
```

5. **Execute and Verify**: Execute the engine and verify results:

```go
result, err := engine.Execute()
if err != nil {
    t.Fatalf("Execution failed: %v", err)
}

// Verify files were created
if !env.FileExists("~/.zshrc") {
    t.Error(".zshrc should exist")
}
```

## Known Limitations

1. **Docker Not Required**: Tests use temporary directories instead of Docker for simplicity and speed. This means tests run on the host system, but each test is isolated.

2. **Permission Tests**: Some permission-related tests may be skipped in CI environments where they can't be reliably tested.

3. **Git Requirements**: Tests require Git to be installed and available in PATH.

4. **External Dependencies**: Tests that require external services (like 1Password CLI) may be skipped if those tools are not available.

## CI Integration

Integration tests are automatically run in the CI pipeline. See `.github/workflows/ci.yml` for configuration.

## Troubleshooting

### Tests Fail with "git: command not found"

Ensure Git is installed and available in your PATH:

```bash
which git
```

### Tests Fail with Permission Errors

Some tests modify file permissions. On macOS/Linux, ensure you have write permissions to the temporary directory. These tests may be skipped in CI.

### Tests Fail with "no such file or directory"

Ensure you're running tests from the project root:

```bash
cd /path/to/kilt
go test ./test/integration/...
```

### Fixture Repository Not Found

Ensure the fixture repository exists at `test/fixtures/repo/`. The test will automatically find it relative to the project root.

## Future Improvements

- [ ] Add Docker-based test isolation for complete environment control
- [ ] Add tests for Windows compatibility
- [ ] Add performance benchmarks
- [ ] Add tests for concurrent execution scenarios
- [ ] Add tests for large repository scenarios
