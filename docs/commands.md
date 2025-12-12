# Command Reference

Complete reference for all Kilt commands and flags.

## Table of Contents

- [Global Flags](#global-flags)
- [kilt init](#kilt-init)
- [kilt sync](#kilt-sync)
- [kilt doctor](#kilt-doctor)
- [kilt version](#kilt-version)
- [kilt restore](#kilt-restore)
- [kilt backups](#kilt-backups)
- [kilt reset](#kilt-reset)
- [kilt brew](#kilt-brew)
- [kilt completion](#kilt-completion)

## Global Flags

All commands support the following global flags:

| Flag | Short | Description |
|------|------|-------------|
| `--dry-run` | | Show what would happen without executing |
| `--verbose` | `-v` | Detailed logging output |
| `--config <path>` | | Custom config file location (default: `.kilt/config.yaml` or `~/.kilt/config.yaml`) |
| `--no-colour` | | Disable coloured output |
| `--force` | | Skip confirmation prompts |

### Examples

```bash
# Preview changes without executing
kilt sync --dry-run

# Verbose output
kilt sync --verbose

# Use custom config file
kilt sync --config /path/to/config.yaml

# Disable coloured output
kilt sync --no-colour

# Skip confirmation prompts
kilt restore 20250125-143022 --force
```

## kilt init

Initialise Kilt from a Git repository.

### Synopsis

```bash
kilt init <repo-url>
```

### Description

The `init` command clones a Git repository containing your dotfiles and sets up the Kilt directory structure. It reads the configuration from `.kilt/config.yaml` in the cloned repository.

### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `<repo-url>` | Yes | URL of the Git repository containing your dotfiles |

### Examples

```bash
# Initialise from GitHub repository
kilt init https://github.com/yourusername/dotfiles

# Initialise from SSH URL
kilt init git@github.com:yourusername/dotfiles.git

# Initialise from local repository
kilt init /path/to/local/repo
```

### What It Does

1. Clones the repository to `~/.dotfiles` (or path specified in config)
2. Reads configuration from `~/.dotfiles/.kilt/config.yaml`
3. Creates `.kilt` directory structure:
   - `~/.kilt/state/` - State tracking
   - `~/.kilt/backup/` - Backup storage
4. Sets up initial state

### Notes

- The repository must contain a `.kilt/config.yaml` file
- If the repository already exists, it will be updated (pulled)
- Use `kilt sync` after initialization to apply dotfiles

## kilt sync

Synchronise dotfiles bidirectionally.

### Synopsis

```bash
kilt sync [flags]
```

### Description

The `sync` command is the main command for synchronising your dotfiles. It:
- Pulls remote changes from Git
- Applies local changes to Git (if any)
- Executes plugins in the correct order
- Syncs files to your home directory
- Runs run-once scripts (if not already executed)
- Executes on-change commands (if files changed)

### Examples

```bash
# Basic sync
kilt sync

# Preview changes
kilt sync --dry-run

# Verbose output
kilt sync --verbose

# Force sync (skip confirmations)
kilt sync --force
```

### Execution Flow

1. **Pre-flight checks**: Validates configuration and dependencies
2. **Git operations**: Pulls remote changes, pushes local changes
3. **Backup creation**: Backs up files that will be modified
4. **Plugin execution**:
   - PhasePreSync: Git operations
   - PhaseCore: Files, directories
   - PhaseRunOnce: Bootstrap scripts
   - PhaseOnChange: Change-triggered tasks
   - PhaseIntegration: Package managers
   - PhasePostSync: Validation, cleanup
5. **State update**: Records changes and updates checksums

### Notes

- Safe to run multiple times (idempotent)
- Automatically handles conflicts (alerts user if manual resolution needed)
- Creates backups before modifying files
- Skips unchanged files based on checksums

## kilt doctor

Validate setup and dependencies.

### Synopsis

```bash
kilt doctor [flags]
```

### Description

The `doctor` command validates your Kilt setup and checks for common issues. It verifies:
- Repository initialization
- Configuration file validity
- Required dependencies (Git, etc.)
- File permissions
- Path accessibility

### Examples

```bash
# Run diagnostics
kilt doctor

# Verbose output
kilt doctor --verbose
```

### Checks Performed

- ✅ Repository exists and is accessible
- ✅ Configuration file exists and is valid
- ✅ Git is installed and in PATH
- ✅ Required directories are accessible
- ✅ File permissions are correct
- ✅ State directory is writable

### Notes

- Run this command if you encounter issues
- Provides helpful error messages and suggestions
- Non-destructive (read-only checks)

## kilt version

Show version information.

### Synopsis

```bash
kilt version
```

### Description

Displays the Kilt version, build time, and other version information.

### Examples

```bash
kilt version
```

### Output

```
kilt version 1.0.0
Build time: 2025-01-25T14:30:22Z
```

## kilt restore

Restore files from a backup.

### Synopsis

```bash
kilt restore <backup-id> [flags]
```

### Description

The `restore` command restores files from a previous backup. It shows a preview of files to be restored and asks for confirmation before proceeding.

### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `<backup-id>` | Yes | Backup ID in format `YYYYMMDD-HHMMSS` |

### Flags

| Flag | Description |
|------|-------------|
| `--force` | Skip confirmation prompt |

### Examples

```bash
# List available backups first
kilt backups list

# Restore from backup
kilt restore 20250125-143022

# Restore without confirmation
kilt restore 20250125-143022 --force
```

### What It Does

1. Lists files in the backup
2. Shows preview of files to be restored
3. Asks for confirmation (unless `--force` is used)
4. Restores files to their original locations
5. Preserves file permissions

### Notes

- Backups are stored in `~/.kilt/backup/`
- Use `kilt backups list` to see available backups
- Restoration overwrites existing files (creates new backup first)

## kilt backups

Manage backups.

### Synopsis

```bash
kilt backups <subcommand>
```

### Subcommands

- `list` - List all available backups

### kilt backups list

List all available backups.

#### Synopsis

```bash
kilt backups list [flags]
```

#### Flags

| Flag | Description |
|------|-------------|
| `--json` | Output in JSON format |

#### Examples

```bash
# List backups in human-readable format
kilt backups list

# List backups in JSON format
kilt backups list --json
```

#### Output Format

Human-readable:
```
Backup ID        Date                Time      Files    Size      Description
20250125-143022  2025-01-25          14:30:22  5        1.2 MB    Sync operation
20250124-092011  2025-01-24          09:20:11  3        456 KB    Manual backup
```

JSON:
```json
[
  {
    "backup_id": "20250125-143022",
    "timestamp": "2025-01-25T14:30:22Z",
    "file_count": 5,
    "total_size": 1258291,
    "description": "Sync operation"
  }
]
```

## kilt reset

Clear state and start fresh.

### Synopsis

```bash
kilt reset [flags]
```

### Description

The `reset` command clears the Kilt state, allowing you to start fresh. This is useful if:
- State file is corrupted
- You want to re-run all run-once scripts
- You're troubleshooting issues

### Examples

```bash
# Reset state (with confirmation)
kilt reset

# Reset without confirmation
kilt reset --force
```

### What It Does

1. Removes state file (`~/.kilt/state/state.json`)
2. Clears run-once execution records
3. Clears file checksums
4. Clears plugin execution history

### Notes

- **Warning**: This will cause run-once scripts to execute again
- Backups are not affected
- Configuration is not affected
- Use with caution

## kilt brew

Manage Homebrew packages (Brew plugin).

### Synopsis

```bash
kilt brew <subcommand>
```

### Subcommands

- `install` - Install packages from Brewfile
- `update` - Update Homebrew and packages
- `cleanup` - Clean up old Homebrew files

### kilt brew install

Install packages from Brewfile.

#### Synopsis

```bash
kilt brew install [flags]
```

#### Flags

| Flag | Description |
|------|-------------|
| `--force` | Install Homebrew if not found (requires user interaction) |

#### Examples

```bash
# Install packages from Brewfile
kilt brew install

# Install Homebrew if not found
kilt brew install --force
```

#### What It Does

1. Checks if Homebrew is installed
2. If not found and `--force` is used, installs Homebrew
3. Runs `brew bundle --file=Brewfile` for configured bundle files
4. Optionally runs `brew update` and `brew cleanup` (based on config)

### kilt brew update

Update Homebrew and packages.

#### Synopsis

```bash
kilt brew update
```

#### Description

Updates Homebrew and upgrades all installed packages.

#### Examples

```bash
kilt brew update
```

#### What It Does

1. Runs `brew update`
2. Runs `brew upgrade`

### kilt brew cleanup

Clean up old Homebrew files.

#### Synopsis

```bash
kilt brew cleanup
```

#### Description

Removes old versions of Homebrew packages to free up disk space.

#### Examples

```bash
kilt brew cleanup
```

#### What It Does

1. Runs `brew cleanup`

## kilt completion

Generate shell completion scripts.

### Synopsis

```bash
kilt completion <shell>
```

### Description

Generates shell completion scripts for bash, zsh, fish, or powershell.

### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `<shell>` | Yes | Shell type: `bash`, `zsh`, `fish`, or `powershell` |

### Examples

```bash
# Generate bash completion
kilt completion bash > ~/.kilt/completion.bash
echo "source ~/.kilt/completion.bash" >> ~/.bashrc

# Generate zsh completion
kilt completion zsh > ~/.kilt/completion.zsh
echo "source ~/.kilt/completion.zsh" >> ~/.zshrc

# Generate fish completion
kilt completion fish > ~/.config/fish/completions/kilt.fish
```

### Installation

**Bash**:
```bash
kilt completion bash > /etc/bash_completion.d/kilt
# Or for user-specific:
kilt completion bash > ~/.kilt/completion.bash
echo "source ~/.kilt/completion.bash" >> ~/.bashrc
```

**Zsh**:
```bash
kilt completion zsh > ~/.kilt/completion.zsh
echo "source ~/.kilt/completion.zsh" >> ~/.zshrc
```

**Fish**:
```bash
kilt completion fish > ~/.config/fish/completions/kilt.fish
```

**PowerShell**:
```powershell
kilt completion powershell | Out-File -FilePath $PROFILE -Append
```

## Command Aliases

You can create aliases for common commands:

```bash
# Add to ~/.zshrc or ~/.bashrc
alias ks='kilt sync'
alias kd='kilt doctor'
alias kb='kilt backups list'
```

## Getting Help

For any command, use the `--help` flag:

```bash
kilt sync --help
kilt restore --help
kilt brew install --help
```

## See Also

- [Configuration Guide](../README.md#configuration)
- [Plugin Documentation](plugins.md)
- [Troubleshooting Guide](troubleshooting.md)
- [Example Configurations](examples/)

