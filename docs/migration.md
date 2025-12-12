# Migration Guide

This guide helps you migrate from other dotfiles management tools to Kilt.

## Table of Contents

- [Migrating from yadm](#migrating-from-yadm)
- [Migrating from chezmoi](#migrating-from-chezmoi)
- [Migrating from homesick](#migrating-from-homesick)
- [General Migration Steps](#general-migration-steps)

## Migrating from yadm

[yadm](https://yadm.io/) (Yet Another Dotfiles Manager) is a Git-based dotfiles manager similar to Kilt.

### Key Differences

| Feature | yadm | Kilt |
|---------|------|------|
| Repository | Bare Git repo | Regular Git repo |
| Alternates | Built-in | Plugin-based |
| Templates | yadm-specific | Go templates |
| Plugins | None | Extensible plugin system |
| Secrets | Manual handling | 1Password integration |

### Migration Steps

1. **Export your yadm configuration**:
   ```bash
   yadm list -a > yadm-files.txt
   ```

2. **Create Kilt configuration**:
   ```yaml
   # .kilt/config.yaml
   dotfiles_repo: "https://github.com/yourusername/dotfiles"
   dotfiles_path: "~/.dotfiles"
   
   dotfiles:
     # Add your dotfiles (yadm uses .yadm/files/)
     - zsh
     - git
     - vim
   ```

3. **Convert alternates**:
   
   **yadm alternates**:
   ```
   .zshrc##os.Darwin
   .zshrc##os.Linux
   ```
   
   **Kilt alternates** (via plugin):
   ```yaml
   plugins:
     alternates:
       patterns:
         - "##os.{darwin,linux}"
   ```
   
   Files: `.zshrc.mac`, `.zshrc.linux`

4. **Convert templates**:
   
   **yadm templates**:
   ```bash
   # .yadm/files/.gitconfig
   [user]
       name = {{ yadm:class }}
   ```
   
   **Kilt templates**:
   ```yaml
   dotfiles:
     - source: git/config
       target: ~/.gitconfig
       template: true
   ```
   
   ```bash
   # git/config
   [user]
       name = {{ .Custom.class }}
   ```

5. **Migrate bootstrap scripts**:
   
   **yadm bootstrap**:
   ```bash
   # .yadm/bootstrap
   #!/bin/bash
   brew bundle
   ```
   
   **Kilt run-once**:
   ```yaml
   run_once:
     - scripts/bootstrap.sh
   ```

6. **Initialise Kilt**:
   ```bash
   kilt init https://github.com/yourusername/dotfiles
   kilt sync
   ```

### yadm-Specific Features

- **Encrypted files**: Use 1Password plugin instead
- **Alternates**: Use Alternates plugin
- **Bootstrap**: Use Run Once plugin
- **Class**: Use custom data file

## Migrating from chezmoi

[chezmoi](https://www.chezmoi.io/) is a popular dotfiles manager with templating support.

### Key Differences

| Feature | chezmoi | Kilt |
|---------|---------|------|
| Templates | chezmoi-specific | Go templates |
| Source of truth | `~/.local/share/chezmoi` | Git repository |
| Secrets | Age encryption | 1Password integration |
| State | Internal | JSON state file |

### Migration Steps

1. **Export chezmoi source files**:
   ```bash
   chezmoi source-path > chezmoi-files.txt
   ```

2. **Convert templates**:
   
   **chezmoi templates**:
   ```bash
   # .chezmoi/dot_gitconfig.tmpl
   [user]
       name = {{ .name }}
       email = {{ .email }}
   ```
   
   **Kilt templates**:
   ```yaml
   dotfiles:
     - source: git/config
       target: ~/.gitconfig
       template: true
   ```
   
   ```bash
   # git/config
   [user]
       name = {{ .Custom.personal.name }}
       email = {{ .Custom.personal.email }}
   ```

3. **Convert data**:
   
   **chezmoi data**:
   ```toml
   # .chezmoi/data.toml
   [personal]
       name = "John Doe"
       email = "john@example.com"
   ```
   
   **Kilt data**:
   ```yaml
   # ~/.kilt/data.yaml or in repo
   personal:
     name: "John Doe"
     email: "john@example.com"
   ```

4. **Convert encrypted files**:
   
   **chezmoi**:
   ```bash
   chezmoi encrypt ~/.ssh/id_rsa
   ```
   
   **Kilt**:
   - Use 1Password plugin
   - Store secrets in 1Password
   - Reference in templates: `{{ op "Private/ssh-key" }}`

5. **Convert scripts**:
   
   **chezmoi scripts**:
   ```bash
   # .chezmoi/run_install.sh
   #!/bin/bash
   brew bundle
   ```
   
   **Kilt run-once**:
   ```yaml
   run_once:
     - scripts/install.sh
   ```

6. **Create Kilt configuration**:
   ```yaml
   dotfiles_repo: "https://github.com/yourusername/dotfiles"
   dotfiles_path: "~/.dotfiles"
   
   data_file: "data.yaml"
   template_engine: "go"
   
   dotfiles:
     - git
     - zsh
     - vim
   
   run_once:
     - scripts/install.sh
   ```

7. **Initialise and sync**:
   ```bash
   kilt init https://github.com/yourusername/dotfiles
   kilt sync
   ```

### chezmoi-Specific Features

- **Age encryption**: Use 1Password plugin
- **Templates**: Convert to Go template syntax
- **Scripts**: Use Run Once plugin
- **Data**: Use custom data file

## Migrating from homesick

[homesick](https://github.com/technicalpickles/homesick) is a Ruby-based dotfiles manager.

### Key Differences

| Feature | homesick | Kilt |
|---------|----------|------|
| Language | Ruby | Go (single binary) |
| Structure | `~/.homesick/repos/dotfiles` | `~/.dotfiles` |
| Symlinks | Automatic | Automatic |
| Scripts | `install` file | Run Once plugin |

### Migration Steps

1. **Export homesick files**:
   ```bash
   ls -la ~/.homesick/repos/dotfiles/
   ```

2. **Create Kilt configuration**:
   ```yaml
   dotfiles_repo: "https://github.com/yourusername/dotfiles"
   dotfiles_path: "~/.dotfiles"
   
   dotfiles:
     - zsh
     - git
     - vim
   ```

3. **Convert install script**:
   
   **homesick install**:
   ```bash
   # dotfiles/install
   #!/bin/bash
   brew bundle
   ```
   
   **Kilt run-once**:
   ```yaml
   run_once:
     - scripts/install.sh
   ```

4. **Initialise Kilt**:
   ```bash
   kilt init https://github.com/yourusername/dotfiles
   kilt sync
   ```

## General Migration Steps

### 1. Inventory Your Current Setup

List all files and configurations:
```bash
# Find all dotfiles
find ~ -maxdepth 1 -name ".*" -type f > dotfiles.txt

# List current tool's managed files
# (tool-specific command)
```

### 2. Organize Your Repository

Create a clean structure:
```
dotfiles/
├── .kilt/
│   └── config.yaml
├── zsh/
│   ├── .zshrc
│   └── .zshenv
├── git/
│   └── .gitconfig
├── vim/
│   └── .vimrc
└── scripts/
    └── install.sh
```

### 3. Create Kilt Configuration

Start with a minimal config and expand:
```yaml
# .kilt/config.yaml
dotfiles_repo: "https://github.com/yourusername/dotfiles"
dotfiles_path: "~/.dotfiles"

dotfiles:
  - zsh
  - git
```

### 4. Convert Templates

If you use templates:
- Identify template syntax
- Convert to Go template syntax
- Set `template: true` in config

### 5. Convert Scripts

If you have bootstrap/install scripts:
- Move to `scripts/` directory
- Add to `run_once` in config
- Make executable

### 6. Handle Secrets

If you have encrypted files or secrets:
- Set up 1Password CLI
- Store secrets in 1Password
- Use `{{ op "path/to/secret" }}` in templates

### 7. Test Migration

1. **Backup current setup**:
   ```bash
   # Your existing tool may have backups
   # Or create manual backup
   tar -czf dotfiles-backup.tar.gz ~/.zshrc ~/.gitconfig ...
   ```

2. **Initialise Kilt**:
   ```bash
   kilt init https://github.com/yourusername/dotfiles
   ```

3. **Dry run**:
   ```bash
   kilt sync --dry-run
   ```

4. **Sync**:
   ```bash
   kilt sync
   ```

5. **Verify**:
   ```bash
   ls -la ~/.zshrc
   ls -la ~/.gitconfig
   # Check symlinks point to correct locations
   ```

### 8. Clean Up

After successful migration:
1. Remove old tool (if desired)
2. Update documentation
3. Test on another machine

## Common Migration Challenges

### Challenge: Different Template Syntax

**Solution**: Convert templates manually or write a conversion script. Go templates are similar to many other template systems.

### Challenge: Encrypted Files

**Solution**: Use 1Password plugin. Store secrets in 1Password and reference in templates.

### Challenge: Complex Alternates

**Solution**: Use Alternates plugin with custom patterns, or use explicit mappings with templates.

### Challenge: Custom Scripts

**Solution**: Use Run Once plugin for bootstrap scripts, On Change plugin for reactive scripts.

## Getting Help

If you encounter issues during migration:

1. Check [Troubleshooting Guide](troubleshooting.md)
2. Review [Example Configurations](examples/)
3. See [Plugin Documentation](plugins.md)
4. Open an issue on GitHub

## Next Steps

After migration:

1. Read [Configuration Guide](../README.md#configuration)
2. Explore [Plugin Features](plugins.md)
3. Set up [1Password Integration](plugins.md#1password-plugin)
4. Configure [Backups](../README.md#safety-features)

