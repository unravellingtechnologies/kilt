# Troubleshooting Guide

This guide helps you diagnose and fix common issues with Kilt.

## Table of Contents

- [Installation Issues](#installation-issues)
- [Configuration Issues](#configuration-issues)
- [Sync Issues](#sync-issues)
- [Plugin Issues](#plugin-issues)
- [Backup and Restore Issues](#backup-and-restore-issues)
- [Performance Issues](#performance-issues)
- [Getting Help](#getting-help)

## Installation Issues

### "command not found: kilt"

**Problem**: The `kilt` command is not found in your PATH.

**Solutions**:
1. Check if Kilt is installed:
   ```bash
   which kilt
   ```

2. If not installed, install it:
   ```bash
   # Using the installer script
   curl -sL https://get.kilt.pro | bash
   
   # Or build from source
   make install
   ```

3. Ensure the installation directory is in your PATH:
   ```bash
   # Check common locations
   ls -la /usr/local/bin/kilt
   ls -la ~/.local/bin/kilt
   
   # Add to PATH if needed (add to ~/.zshrc or ~/.bashrc)
   export PATH="$HOME/.local/bin:$PATH"
   ```

### Installation Script Fails

**Problem**: The curl installer script fails to download or install.

**Solutions**:
1. Check your internet connection
2. Verify the script URL is accessible
3. Try downloading manually:
   ```bash
   curl -L https://get.kilt.pro -o install.sh
   bash install.sh
   ```
4. Check for permission issues:
   ```bash
   chmod +x install.sh
   ```

### Build from Source Fails

**Problem**: Building Kilt from source fails.

**Solutions**:
1. Ensure Go 1.21+ is installed:
   ```bash
   go version
   ```

2. Check dependencies:
   ```bash
   go mod download
   go mod tidy
   ```

3. Clean and rebuild:
   ```bash
   make clean
   make build
   ```

## Configuration Issues

### "config file not found"

**Problem**: Kilt cannot find the configuration file.

**Solutions**:
1. Check if the config file exists:
   ```bash
   ls -la ~/.dotfiles/.kilt/config.yaml
   ```

2. Initialise Kilt if not done:
   ```bash
   kilt init https://github.com/yourusername/dotfiles
   ```

3. Use `--config` flag to specify custom location:
   ```bash
   kilt sync --config /path/to/config.yaml
   ```

4. Check the default search locations:
   - `~/.dotfiles/.kilt/config.yaml`
   - `~/.kilt/config.yaml`

### "invalid configuration"

**Problem**: Configuration file has syntax errors or invalid values.

**Solutions**:
1. Validate YAML syntax:
   ```bash
   # Use a YAML validator
   yamllint ~/.dotfiles/.kilt/config.yaml
   ```

2. Check for common issues:
   - Missing quotes around strings with special characters
   - Incorrect indentation (YAML is sensitive to indentation)
   - Invalid field names

3. Use `kilt doctor` to validate:
   ```bash
   kilt doctor
   ```

4. Review the [configuration examples](examples/) for correct format

### Configuration Not Applied

**Problem**: Changes to configuration are not being applied.

**Solutions**:
1. Ensure you're editing the correct config file:
   ```bash
   kilt doctor  # Shows config file location
   ```

2. Run sync to apply changes:
   ```bash
   kilt sync
   ```

3. Check for YAML syntax errors (see above)

4. Use `--verbose` flag to see what's happening:
   ```bash
   kilt sync --verbose
   ```

## Sync Issues

### "failed to acquire lock"

**Problem**: Another Kilt process is already running.

**Solutions**:
1. Check for running processes:
   ```bash
   ps aux | grep kilt
   ```

2. If no process is running, remove stale lock:
   ```bash
   rm ~/.kilt/state/.lock
   ```

3. Wait for the other process to complete

4. Check lock file age (stale locks > 5 minutes are auto-removed)

### "git pull failed"

**Problem**: Git operations fail during sync.

**Solutions**:
1. Check Git is installed:
   ```bash
   git --version
   ```

2. Verify repository access:
   ```bash
   cd ~/.dotfiles
   git status
   ```

3. Check authentication:
   ```bash
   # For SSH
   ssh -T git@github.com
   
   # For HTTPS, ensure credentials are configured
   ```

4. Resolve conflicts manually:
   ```bash
   cd ~/.dotfiles
   git pull
   # Resolve conflicts
   git add .
   git commit
   ```

5. Use `--dry-run` to preview changes:
   ```bash
   kilt sync --dry-run
   ```

### Files Not Syncing

**Problem**: Files are not being synced to home directory.

**Solutions**:
1. Check configuration:
   ```bash
   kilt doctor
   ```

2. Verify dotfiles are in the repository:
   ```bash
   ls -la ~/.dotfiles/zsh/
   ```

3. Use `--verbose` to see what's happening:
   ```bash
   kilt sync --verbose
   ```

4. Check for permission issues:
   ```bash
   ls -la ~/.zshrc
   ```

5. Use `--dry-run` to preview:
   ```bash
   kilt sync --dry-run
   ```

### Symlinks Not Created

**Problem**: Symlinks are not being created.

**Solutions**:
1. Check if target already exists (not a symlink):
   ```bash
   ls -la ~/.zshrc
   ```

2. Remove existing file (backup first):
   ```bash
   kilt restore <backup-id>  # If needed
   rm ~/.zshrc
   kilt sync
   ```

3. Check permissions:
   ```bash
   ls -ld ~
   ```

4. Use `--force` to overwrite:
   ```bash
   kilt sync --force
   ```

## Plugin Issues

### Run Once Scripts Not Executing

**Problem**: Run-once scripts are not executing.

**Solutions**:
1. Check if script already executed:
   ```bash
   cat ~/.kilt/state/state.json | grep -A 5 "run_once"
   ```

2. Force re-execution:
   ```yaml
   plugins:
     runonce:
       force_run: true
   ```

3. Check script exists and is executable:
   ```bash
   ls -la ~/.dotfiles/scripts/install.sh
   chmod +x ~/.dotfiles/scripts/install.sh
   ```

4. Check script path in config is correct

### Brew Plugin Not Working

**Problem**: Brew plugin fails to install packages.

**Solutions**:
1. Check Homebrew is installed:
   ```bash
   which brew
   ```

2. Enable auto-installation:
   ```yaml
   plugins:
     brew:
       auto_install: true
   ```

3. Check Brewfile exists:
   ```bash
   ls -la ~/.dotfiles/Brewfile
   ```

4. Test manually:
   ```bash
   brew bundle --file=~/.dotfiles/Brewfile
   ```

5. Check permissions:
   ```bash
   brew doctor
   ```

### 1Password Plugin Not Working

**Problem**: 1Password secrets are not being injected.

**Solutions**:
1. Check 1Password CLI is installed:
   ```bash
   which op
   ```

2. Verify authentication:
   ```bash
   op signin
   ```

3. Test secret access:
   ```bash
   op read "Private/github-token"
   ```

4. Check plugin configuration:
   ```yaml
   plugins:
     onepassword:
       account: "my.1password.com"
   ```

5. Verify secret path in template:
   ```yaml
   # Correct format
   token = {{ op "Private/github-token" }}
   ```

### Template Rendering Fails

**Problem**: Templates are not rendering correctly.

**Solutions**:
1. Check template syntax:
   ```bash
   # Valid Go template syntax
   {{ .Hostname }}
   {{ env "VAR" }}
   {{ op "path/to/secret" }}
   ```

2. Verify template flag is set:
   ```yaml
   dotfiles:
     - source: ssh/config
       target: ~/.ssh/config
       template: true  # Required for template rendering
   ```

3. Check custom data file:
   ```bash
   cat ~/.kilt/data.yaml
   ```

4. Use `--verbose` to see template errors:
   ```bash
   kilt sync --verbose
   ```

## Backup and Restore Issues

### Backup Not Created

**Problem**: Backups are not being created before file modifications.

**Solutions**:
1. Check backup directory exists:
   ```bash
   ls -la ~/.kilt/backup/
   ```

2. Verify permissions:
   ```bash
   ls -ld ~/.kilt/backup
   ```

3. Check disk space:
   ```bash
   df -h ~/.kilt
   ```

4. Use `--verbose` to see backup operations:
   ```bash
   kilt sync --verbose
   ```

### Restore Fails

**Problem**: Cannot restore from backup.

**Solutions**:
1. List available backups:
   ```bash
   kilt backups list
   ```

2. Verify backup exists:
   ```bash
   ls -la ~/.kilt/backup/20250125-143022/
   ```

3. Check backup metadata:
   ```bash
   cat ~/.kilt/backup/20250125-143022/metadata.json
   ```

4. Use `--force` to skip confirmation:
   ```bash
   kilt restore 20250125-143022 --force
   ```

5. Check permissions:
   ```bash
   ls -la ~/.kilt/backup/
   ```

## Performance Issues

### Sync is Slow

**Problem**: `kilt sync` takes too long.

**Solutions**:
1. Check for large files:
   ```bash
   du -sh ~/.dotfiles
   ```

2. Use `--dry-run` to see what's being processed:
   ```bash
   kilt sync --dry-run
   ```

3. Check network speed (for Git operations):
   ```bash
   cd ~/.dotfiles
   time git pull
   ```

4. Review plugin configuration (some plugins may be slow)

5. Check state file size:
   ```bash
   ls -lh ~/.kilt/state/state.json
   ```

### High Memory Usage

**Problem**: Kilt uses too much memory.

**Solutions**:
1. Check for large state file:
   ```bash
   ls -lh ~/.kilt/state/state.json
   ```

2. Clean old backups:
   ```bash
   # Manual cleanup
   rm -rf ~/.kilt/backup/old-backups/
   ```

3. Reset state if needed:
   ```bash
   kilt reset
   ```

4. Report issue if persistent (may be a bug)

## Getting Help

### Diagnostic Information

Before asking for help, gather diagnostic information:

```bash
# Run doctor command
kilt doctor

# Check version
kilt version

# Verbose sync output
kilt sync --verbose > sync.log 2>&1

# System information
uname -a
go version
git --version
```

### Resources

- **GitHub Issues**: [Report bugs or request features](https://github.com/unravelling/kilt/issues)
- **Documentation**: See [docs/](.) for detailed documentation
- **Examples**: Check [examples/](examples/) for configuration examples
- **Architecture**: See [architecture.md](architecture.md) for system details

### Reporting Issues

When reporting an issue, include:
1. Kilt version (`kilt version`)
2. Operating system and version
3. Configuration file (sanitized, no secrets)
4. Error messages or logs
5. Steps to reproduce
6. Expected vs actual behaviour

### Common Commands for Debugging

```bash
# Validate configuration
kilt doctor

# Preview changes
kilt sync --dry-run

# Verbose output
kilt sync --verbose

# Check state
cat ~/.kilt/state/state.json

# List backups
kilt backups list

# Reset state
kilt reset
```

