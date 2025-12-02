# Security Audit

This document outlines the security considerations and audit results for Kilt.

## Security Model

Kilt follows a security model that prioritizes:
- **Input validation** - All user inputs are validated
- **Path safety** - Path traversal attacks are prevented
- **Secret protection** - Secrets never touch Git
- **Safe execution** - Command injection is prevented
- **Backup safety** - Data is never lost without backup

## Threat Model

### What We Protect Against

1. **Path Traversal Attacks**
   - All paths are validated and sanitized
   - `~` expansion is safe
   - Relative paths are resolved to absolute paths
   - Paths outside allowed directories are rejected

2. **Command Injection**
   - Commands are executed with explicit arguments
   - No shell interpolation of user input
   - Script execution is controlled

3. **Secret Leakage**
   - Secrets are never stored in Git
   - 1Password integration for secret management
   - No logging of secret values

4. **Data Loss**
   - Automatic backups before modifications
   - Rollback capability on failure
   - State locking prevents concurrent modifications

5. **Malicious Configuration**
   - Configuration validation
   - Schema checking
   - Type validation

### What We Don't Protect Against

1. **Malicious Dotfiles Repository**
   - Users trust their own repositories
   - We assume the repository is trusted

2. **Compromised System**
   - If the system is compromised, Kilt cannot protect against it
   - We are not a security tool

3. **Supply Chain Attacks**
   - Go module checksums help, but we rely on Go's security
   - Binary distribution should be verified

## Security Audit Results

### ✅ Input Validation

**Status**: Implemented

**Checks**:
- Configuration file paths are validated
- File paths are sanitized and validated
- YAML parsing validates structure
- Environment variable expansion is safe

**Implementation**:
```go
// Path validation in internal/core/paths.go
func ValidatePath(path string) error {
    // Expand to absolute path
    absPath, err := filepath.Abs(path)
    if err != nil {
        return err
    }
    
    // Check for directory traversal
    if strings.Contains(absPath, "..") {
        return ErrInvalidPath
    }
    
    // Ensure within allowed directories
    if !isAllowedPath(absPath) {
        return ErrForbiddenPath
    }
    
    return nil
}
```

### ✅ Path Traversal Protection

**Status**: Implemented

**Checks**:
- All paths are resolved to absolute paths
- Directory traversal (`..`) is detected and rejected
- Paths are validated against allowed directories
- `~` expansion is safe (uses `os.UserHomeDir()`)

**Implementation**:
- Path expansion uses `filepath.Abs()` and `filepath.Clean()`
- Relative paths are resolved before use
- Path validation checks for `..` sequences

### ✅ Command Injection Prevention

**Status**: Implemented

**Checks**:
- Commands are executed with explicit arguments
- No shell interpolation (`sh -c` is never used with user input)
- Script execution uses explicit paths
- Command arguments are validated

**Implementation**:
```go
// Safe command execution
func RunCommand(name string, args ...string) error {
    cmd := exec.Command(name, args...)
    cmd.Env = sanitizeEnv(os.Environ())
    return cmd.Run()
}

// NEVER do this:
// exec.Command("sh", "-c", userInput) // ❌ UNSAFE
```

### ✅ Secret Management

**Status**: Implemented

**Checks**:
- Secrets are never stored in Git
- 1Password CLI integration for secret injection
- Secrets only in memory during template rendering
- No logging of secret values
- Cache TTL for secrets (optional, in memory only)

**Implementation**:
- Template function `{{ op "path/to/secret" }}` retrieves secrets at render time
- Secrets are never written to disk (except in rendered templates)
- No secret values in logs or error messages

### ✅ File Permissions

**Status**: Implemented

**Checks**:
- File permissions are preserved or set explicitly
- Sensitive files (SSH keys) get `0600`
- Config files get `0644`
- Scripts get `0755`
- Directories get `0755`

**Implementation**:
- File modes are set explicitly when creating files
- Existing file permissions are preserved when backing up
- Permission validation in configuration

### ✅ Backup Safety

**Status**: Implemented

**Checks**:
- Files are backed up before modification
- Backup metadata is stored
- Restore functionality is available
- Backup integrity is maintained

**Implementation**:
- Automatic backups before file modifications
- Timestamped backup directories
- Backup metadata in JSON format
- Restore command with preview

### ✅ State Management

**Status**: Implemented

**Checks**:
- State file is protected
- Lock file prevents concurrent execution
- State corruption is handled gracefully
- State cleanup is available

**Implementation**:
- File-based locking with timeout
- Atomic writes (temp file + rename)
- State validation on load
- Lock cleanup on process exit

### ✅ Configuration Validation

**Status**: Implemented

**Checks**:
- YAML syntax validation
- Schema validation
- Type checking
- Required field validation
- Path validation

**Implementation**:
- YAML parsing with error handling
- Configuration struct validation
- Path expansion and validation
- Error messages with suggestions

## Security Best Practices

### 1. Path Validation

All paths are validated:
- Expanded to absolute paths
- Checked for directory traversal
- Validated against allowed directories
- Sanitized before use

### 2. Command Execution

Commands are executed safely:
- Explicit command and arguments
- No shell interpolation
- Environment sanitization
- Timeout protection

### 3. Secret Handling

Secrets are handled securely:
- Never stored in Git
- Retrieved at render time
- Not logged
- Cached in memory only (optional)

### 4. Error Handling

Errors are handled securely:
- No secret leakage in error messages
- Helpful error messages without exposing internals
- Graceful degradation
- Clear user guidance

## Known Limitations

1. **Repository Trust**: Kilt assumes the dotfiles repository is trusted
2. **System Compromise**: Cannot protect against compromised systems
3. **Binary Verification**: Users should verify binary checksums
4. **1Password CLI**: Relies on 1Password CLI security

## Recommendations

1. **Binary Signing**: Sign releases with GPG
2. **Checksum Verification**: Provide checksums for all releases
3. **Security Advisories**: Maintain a security advisory process
4. **Regular Audits**: Conduct regular security audits
5. **Dependency Updates**: Keep dependencies up to date

## Reporting Security Issues

If you discover a security vulnerability, please report it responsibly:

1. **Do not** open a public issue
2. Email security concerns to the maintainers
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

## Security Checklist

When adding new features, ensure:

- [ ] Input validation is implemented
- [ ] Path traversal is prevented
- [ ] Command injection is prevented
- [ ] Secrets are not logged
- [ ] Error messages don't leak sensitive information
- [ ] File permissions are set correctly
- [ ] Backups are created before modifications
- [ ] State is protected with locking
- [ ] Configuration is validated
- [ ] Tests cover security scenarios

## References

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Go Security Best Practices](https://go.dev/doc/security/best-practices)
- [Path Traversal Prevention](https://owasp.org/www-community/attacks/Path_Traversal)





