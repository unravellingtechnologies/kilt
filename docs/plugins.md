# Plugin Documentation

Kilt uses a plugin architecture where all features are implemented as plugins. This document describes each built-in plugin and how to configure them.

## Table of Contents

- [Dotfiles Plugin](#dotfiles-plugin)
- [Directories Plugin](#directories-plugin)
- [Run Once Plugin](#run-once-plugin)
- [On Change Plugin](#on-change-plugin)
- [Git Plugin](#git-plugin)
- [Brew Plugin](#brew-plugin)
- [1Password Plugin](#1password-plugin)
- [Alternates Plugin](#alternates-plugin)

## Dotfiles Plugin

The Dotfiles Plugin is the core plugin for managing dotfile synchronization. It creates symlinks from your dotfiles repository to your home directory.

### Features

- **Directory-based syncing**: Link all files in a directory to your home directory
- **Symlink creation**: Creates symlinks (not copies) by default
- **Smart dot-prefixing**: Automatically adds `.` prefix to common dotfile names
- **Template rendering**: Render Go templates when `template: true`
- **Permission control**: Set explicit file permissions via `mode`
- **Automatic backups**: Existing files are backed up before modification
- **Idempotency**: Skips unchanged files based on checksum comparison
- **Dry-run support**: Preview changes without modifying filesystem

### Configuration

```yaml
dotfiles:
  # Simple form: directory name
  - zsh    # Links all files in zsh/ to ~/
  - git
  
  # Explicit form: source/target mapping
  - source: ssh/config
    target: ~/.ssh/config
    mode: "0600"
    template: true
```

### Entry Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `directory` | string | No | Directory name to sync (simple form) |
| `source` | string | No | Explicit source file path relative to repo |
| `target` | string | No | Explicit target path (required if source is set) |
| `template` | boolean | No | Whether to render as template (default: `false`) |
| `mode` | string | No | File permissions in octal format (e.g., `"0644"`, `"0600"`) |

### Notes

- For template files, the rendered content is written (not symlinked) to ensure templates are always up-to-date
- Files are automatically backed up before modification
- Unchanged files are skipped based on checksum comparison

## Directories Plugin

The Directories Plugin ensures that specified directories exist with proper permissions.

### Features

- Creates directories with parent directories if needed
- Supports permission specification
- Handles existing directories gracefully
- Dry-run support

### Configuration

```yaml
directories:
  - ~/dev/personal
  - ~/dev/work
  - ~/Documents/notes
  - ~/.local/bin
```

### Behavior

- Directories are created if they don't exist
- Parent directories are created automatically
- Existing directories are left unchanged
- Supports `~` expansion in paths

## Run Once Plugin

The Run Once Plugin executes bootstrap scripts exactly once per machine. Perfect for initial setup tasks.

### Features

- **Idempotent execution**: Scripts run once and are tracked in state
- **Order preservation**: Scripts execute in the order specified
- **Timeout protection**: Configurable timeout prevents hanging scripts
- **Output capture**: Captures stdout/stderr for debugging
- **Force re-execution**: Can force scripts to run again
- **Graceful failure handling**: Failed scripts are recorded
- **Shebang support**: Automatically detects and uses script shebangs

### Configuration

```yaml
run_once:
  - scripts/install_homebrew.sh
  - scripts/setup_macos.sh
  - scripts/setup_ssh_keys.sh

plugins:
  runonce:
    timeout: "15m"      # Custom timeout (default: 5m)
    force_run: false    # Set to true to re-execute all scripts
```

### Plugin Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `timeout` | string | `"5m"` | Maximum execution time per script (supports: `s`, `m`, `h`) |
| `force_run` | boolean | `false` | Force re-execution of already-run scripts |

### Script Requirements

- Scripts must exist and be readable
- Scripts should be executable (`chmod +x`)
- Scripts can use any shebang (`#!/bin/sh`, `#!/bin/bash`, etc.)
- Scripts without shebangs default to `sh`
- Scripts run in their own directory context

### State Tracking

- Each script execution is tracked in `.kilt/state/state.json`
- Task IDs are generated from script paths (SHA256 hash)
- Execution records include: timestamp, exit code, and output
- Failed scripts are recorded to prevent retry loops

## On Change Plugin

The On Change Plugin executes commands when configuration or files change.

### Features

- Detects changes via checksums
- Executes commands in specified order
- Supports conditional execution
- Captures command output
- Dry-run support

### Configuration

```yaml
on_change:
  - brew bundle --file=Brewfile
  - mise install --yes
  - echo "Configuration updated"
```

### Behavior

- Commands execute when files change (detected via checksums)
- Commands run in the order specified
- Output is captured and logged
- Failed commands stop execution and trigger rollback

## Git Plugin

The Git Plugin manages Git repository operations, including pulling remote changes and pushing local changes.

### Features

- **Bidirectional sync**: Pull remote changes and push local changes
- **Conflict handling**: Alerts user if manual resolution needed
- **Auto-pull**: Automatically pull remote changes during sync
- **Extra repositories**: Clone additional repositories to specific paths

### Configuration

```yaml
# Extra repositories to clone
extra_repos:
  - url: "https://github.com/zsh-users/zsh-autosuggestions"
    path: "~/.zsh/plugins/zsh-autosuggestions"
    branch: "main"
    sparse: false

plugins:
  git:
    auto_pull: true  # Automatically pull remote changes
```

### Plugin Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `auto_pull` | boolean | `true` | Automatically pull remote changes during sync |

### Extra Repositories

The `extra_repos` section allows you to clone additional Git repositories:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | Yes | Repository URL |
| `path` | string | Yes | Local path where repository should be cloned |
| `branch` | string | No | Branch to checkout (default: `main`) |
| `sparse` | boolean | No | Enable sparse checkout (default: `false`) |

## Brew Plugin

The Brew Plugin integrates with Homebrew for package management.

### Features

- **Auto-installation**: Automatically install Homebrew if not found
- **Brew bundle execution**: Run `brew bundle` to install packages
- **Multiple bundles**: Support for multiple bundle files
- **Change detection**: Only runs bundle if Brewfile changed
- **Auto-update**: Optionally run `brew update` after bundle
- **Auto-cleanup**: Optionally run `brew cleanup` after bundle
- **Linux compatibility**: Works with Linuxbrew on Linux systems

### Configuration

```yaml
plugins:
  brew:
    auto_install: true   # Install Homebrew if not found
    brewfile: "Brewfile" # Path to Brewfile (default: "Brewfile")
    auto_update: false   # Run brew update after bundle
    cleanup_after: true  # Run brew cleanup after bundle
    bundles:            # Multiple bundle files
      - bootstrap       # Runs brew bundle --file=bootstrap
      - dev             # Runs brew bundle --file=dev
```

### Plugin Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `auto_install` | boolean | `false` | Install Homebrew if not found (requires user interaction) |
| `brewfile` | string | `"Brewfile"` | Path to Brewfile |
| `auto_update` | boolean | `false` | Run `brew update` after bundle |
| `cleanup_after` | boolean | `false` | Run `brew cleanup` after bundle |
| `bundles` | array | `[]` | List of bundle file names (without extension) |

### CLI Commands

```bash
# Install packages from Brewfile
kilt brew install

# Install Homebrew if not found
kilt brew install --force

# Update Homebrew and packages
kilt brew update

# Clean up old Homebrew files
kilt brew cleanup
```

### Auto-Installation

When `auto_install: true` is set and Homebrew is not found:
1. Downloads the official Homebrew installer script
2. Executes it (requires user to enter sudo password)
3. Automatically detects the installed Homebrew after installation
4. Continues with bundle execution

**Requirements**:
- User interaction (sudo password prompt)
- Internet connection
- `curl` command available
- Works on macOS and Linux

### Change Detection

- The plugin only runs `brew bundle` if the Brewfile has changed
- Uses state checksums to detect changes
- Multiple bundles always run (change detection skipped for bundles)

## 1Password Plugin

The 1Password Plugin provides secret injection from 1Password CLI without storing secrets in Git.

### Features

- **Secret injection**: Use `{{ op "path/to/secret" }}` in templates
- **Authentication handling**: Handles 1Password authentication
- **Caching**: Optional caching for secret lookups (with TTL)
- **Multiple accounts**: Support for multiple 1Password accounts
- **Clear error messages**: Provides helpful errors for missing secrets

### Configuration

```yaml
plugins:
  onepassword:
    account: "my.1password.com"  # 1Password account
    cache_ttl: 3600               # Cache secrets for 1 hour (in seconds)
```

### Plugin Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `account` | string | `""` | 1Password account URL |
| `cache_ttl` | integer | `0` | Cache TTL in seconds (0 = no caching) |

### Usage in Templates

```yaml
# In your template file
github_token = {{ op "Private/github-token" }}
api_key = {{ op "Private/api-key" }}
```

### Requirements

- 1Password CLI (`op`) must be installed
- 1Password CLI must be authenticated (`op signin`)
- Secrets must be accessible via the configured account

## Alternates Plugin

The Alternates Plugin provides automatic file selection based on OS, hostname, or architecture.

### Features

- **OS-based selection**: Choose files based on operating system
- **Hostname-based selection**: Choose files based on machine hostname
- **Architecture-based selection**: Choose files based on CPU architecture
- **Priority rules**: Define priority for multiple matches
- **Custom patterns**: Support for custom alternate patterns

### Configuration

```yaml
plugins:
  alternates:
    patterns:
      - "##os.{darwin,linux}"
      - "##hostname.{work,personal}"
      - "##class.{laptop,desktop}"
```

### File Naming

Files can be named with alternates:
- `file.mac` - macOS only
- `file.linux` - Linux only
- `file.hostname@work` - Work machine only
- `file.hostname@personal` - Personal machine only

The plugin automatically selects the appropriate file based on the current system.

## Plugin Execution Order

Plugins execute in predefined phases:

1. **PhasePreSync**: Git operations, fetching external repos
2. **PhaseCore**: Core operations (files, directories)
3. **PhaseRunOnce**: Bootstrap scripts
4. **PhaseOnChange**: Change-triggered tasks
5. **PhaseIntegration**: Package managers (Brew)
6. **PhasePostSync**: Validation, cleanup

Within each phase, plugins are ordered by dependencies.

## See Also

- [Plugin Development Guide](plugin-guide.md) - For developing custom plugins
- [Architecture Documentation](architecture.md) - For system architecture details
- [Example Configurations](examples/) - For configuration examples

