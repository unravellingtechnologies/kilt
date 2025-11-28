# Example Configurations

This directory contains example Kilt configuration files to help you get started.

## Files

- **minimal.yaml** - The simplest possible configuration
- **basic.yaml** - A typical configuration for common use cases
- **advanced.yaml** - A comprehensive configuration showcasing all features
- **data.yaml** - Example custom data file for templates

## Usage

1. Copy one of the example files to your dotfiles repository:
   ```bash
   cp docs/examples/basic.yaml ~/.dotfiles/.kilt/config.yaml
   ```

2. Edit the configuration to match your needs:
   - Update `dotfiles_repo` with your repository URL
   - Add your dotfile directories
   - Configure plugins as needed

3. Initialize Kilt:
   ```bash
   kilt init https://github.com/yourusername/dotfiles
   ```

4. Sync your dotfiles:
   ```bash
   kilt sync
   ```

## Starting from Scratch

If you're new to Kilt, start with **minimal.yaml** and gradually add features as you need them.

## Custom Data File

The `data.yaml` file provides custom variables for use in templates. Place it at:
- `~/.kilt/data.yaml` (user-specific, not in Git)
- Or in your dotfiles repository (shared across machines)

Use variables in templates like this:
```yaml
# In your template file
email = {{ .Custom.personal.email }}
github_user = {{ .Custom.personal.github_username }}
```

## Next Steps

- See [Configuration Guide](../README.md#configuration) for detailed documentation
- Check [Plugin Documentation](../plugins.md) for plugin-specific options
- Review [Troubleshooting Guide](../troubleshooting.md) if you encounter issues

