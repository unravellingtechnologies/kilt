# Plugin Development Guide

This guide explains how to develop plugins for Kilt. Plugins are the building blocks of Kilt's functionality and allow you to extend the system with custom features.

## Overview

Kilt uses a plugin architecture where all features are implemented as plugins. Plugins are statically compiled into the binary and execute in a predefined order based on their execution phase and dependencies.

## Plugin Interface

All plugins must implement the `Plugin` interface defined in `internal/plugin/interface.go`:

```go
type Plugin interface {
    // Metadata
    Name() string
    Version() string
    Description() string
    
    // Lifecycle hooks
    Initialise(ctx *Context) error
    Validate() error
    Execute(ctx *ExecutionContext) error
    Rollback(ctx *ExecutionContext) error
    
    // Dependencies and execution
    Dependencies() []string
    Phase() ExecutionPhase
}
```

## Execution Phases

Plugins execute in predefined phases, in this order:

1. **PhasePreSync**: Git operations, fetching external repos
2. **PhaseCore**: Core operations like placing files, creating directories
3. **PhaseRunOnce**: Bootstrap scripts that run exactly once
4. **PhaseOnChange**: Change-triggered tasks
5. **PhaseIntegration**: Package managers and external integrations
6. **PhasePostSync**: Validation, cleanup, reporting

Within each phase, plugins are topologically sorted based on their dependencies.

## Plugin Lifecycle

```
Registration → Initialise → Validate → Execute → Cleanup
                   ↓            ↓          ↓
                 Error ←────────┴──────────┴──→ Rollback
```

### 1. Registration

Plugins are registered when the binary starts. Register your plugin in an `init()` function:

```go
package myplugin

import "github.com/unravelling/kilt/internal/plugin"

func init() {
    plugin.RegisterPlugin(&MyPlugin{})
}
```

### 2. Initialise

The `Initialise()` method is called once during system startup. Use this to:
- Store references to shared context
- Validate plugin configuration
- Prepare any required resources

```go
func (p *MyPlugin) Initialise(ctx *plugin.Context) error {
    p.ctx = ctx
    config := plugin.GetPluginConfig(ctx.Config, p.Name())
    // Process configuration...
    return nil
}
```

### 3. Validate

The `Validate()` method is called after initialization to verify the plugin is correctly configured:

```go
func (p *MyPlugin) Validate() error {
    if p.requiredConfig == "" {
        return errors.New("required config missing")
    }
    return nil
}
```

### 4. Execute

The `Execute()` method contains your plugin's main logic:

```go
func (p *MyPlugin) Execute(ctx *plugin.ExecutionContext) error {
    // Perform plugin operations
    ctx.AddChange(plugin.Change{
        Type:        "my_operation",
        Files:       []string{"file1", "file2"},
        Description: "Did something important",
    })
    return nil
}
```

### 5. Rollback

If an error occurs during execution, `Rollback()` is called to undo any changes:

```go
func (p *MyPlugin) Rollback(ctx *plugin.ExecutionContext) error {
    // Undo any changes made during Execute()
    return nil
}
```

## Plugin Contexts

### Context

Provided during initialization and contains shared resources:

```go
type Context struct {
    Config       Config        // Configuration access
    State        StateManager  // State management
    Backup       BackupManager // Backup operations
    Template     TemplateEngine // Template rendering
    Logger       Logger        // Logging
    DryRun       bool          // Whether we're in dry-run mode
    WorkDir      string        // Working directory
    HomeDir      string        // Home directory
}
```

### ExecutionContext

Provided during execution and extends Context:

```go
type ExecutionContext struct {
    *Context
    Changes   []Change  // Changes made by plugins
    Errors    []error   // Errors collected
    StartTime time.Time // Execution start time
}
```

Useful methods:
- `AddChange(change Change)`: Record a change made by your plugin
- `AddError(err error)`: Record an error
- `HasErrors() bool`: Check if there are any errors

## Dependencies

Plugins can declare dependencies on other plugins:

```go
func (p *MyPlugin) Dependencies() []string {
    return []string{"files", "directories"}
}
```

Dependencies ensure:
- Plugins execute in the correct order
- Dependencies are initialised before dependents
- Circular dependencies are detected and rejected

## Example Plugin

Here's a complete example plugin:

```go
package myplugin

import (
    "errors"
    "github.com/unravelling/kilt/internal/plugin"
)

type MyPlugin struct {
    ctx            *plugin.Context
    configValue    string
}

func init() {
    plugin.RegisterPlugin(&MyPlugin{})
}

func (p *MyPlugin) Name() string {
    return "my-plugin"
}

func (p *MyPlugin) Version() string {
    return "1.0.0"
}

func (p *MyPlugin) Description() string {
    return "My custom plugin"
}

func (p *MyPlugin) Dependencies() []string {
    return []string{"files"}
}

func (p *MyPlugin) Phase() plugin.ExecutionPhase {
    return plugin.PhaseCore
}

func (p *MyPlugin) Initialise(ctx *plugin.Context) error {
    p.ctx = ctx
    config := plugin.GetPluginConfig(ctx.Config, p.Name())
    if val, ok := config["value"].(string); ok {
        p.configValue = val
    }
    return nil
}

func (p *MyPlugin) Validate() error {
    if p.configValue == "" {
        return errors.New("config value is required")
    }
    return nil
}

func (p *MyPlugin) Execute(ctx *plugin.ExecutionContext) error {
    p.ctx.Logger.Info("MyPlugin executing", "value", p.configValue)
    
    // Do something...
    
    ctx.AddChange(plugin.Change{
        Type:        "my_operation",
        Description: "Did something with " + p.configValue,
    })
    
    return nil
}

func (p *MyPlugin) Rollback(ctx *plugin.ExecutionContext) error {
    // Undo changes if needed
    p.ctx.Logger.Info("MyPlugin rolling back")
    return nil
}
```

## Configuration

Plugins can access their configuration from the main config file:

```yaml
plugins:
  my-plugin:
    value: "example"
    enabled: true
```

Access it in your plugin:

```go
config := plugin.GetPluginConfig(ctx.Config, p.Name())
value := config["value"].(string)
```

## Best Practices

1. **Idempotency**: Make your plugin idempotent - running it multiple times should produce the same result.

2. **Dry Run**: Respect the `DryRun` flag in `Context`. Don't make changes when `DryRun` is true.

3. **Error Handling**: Always return errors, never panic. Use `ExecutionContext.AddError()` to collect non-fatal errors.

4. **Change Tracking**: Record changes using `ExecutionContext.AddChange()` so other plugins can react to them.

5. **Logging**: Use the provided logger for all log messages. Use appropriate log levels (Debug, Info, Warn, Error).

6. **Rollback**: Implement proper rollback logic to undo changes on failure.

7. **Validation**: Always validate configuration and state in the `Validate()` method.

8. **Documentation**: Document your plugin's configuration options and behaviour.

## Testing

Create unit tests for your plugin:

```go
func TestMyPlugin(t *testing.T) {
    plugin := &MyPlugin{}
    
    ctx := &plugin.Context{
        Config: &mockConfig{},
        Logger: &mockLogger{},
        // ...
    }
    
    err := plugin.Initialise(ctx)
    assert.NoError(t, err)
    
    err = plugin.Validate()
    assert.NoError(t, err)
    
    execCtx := &plugin.ExecutionContext{
        Context: ctx,
        Changes: []plugin.Change{},
        Errors:  []error{},
        StartTime: time.Now(),
    }
    
    err = plugin.Execute(execCtx)
    assert.NoError(t, err)
}
```

## Plugin Discovery

Plugins are automatically discovered through Go's `init()` functions. Simply register your plugin:

```go
func init() {
    plugin.RegisterPlugin(&MyPlugin{})
}
```

The plugin system will automatically:
- Register your plugin
- Validate dependencies
- Initialise and validate it
- Execute it in the correct order

## Common Patterns

### Conditional Execution

```go
func (p *MyPlugin) Execute(ctx *plugin.ExecutionContext) error {
    if ctx.DryRun {
        p.ctx.Logger.Info("Would execute my plugin")
        return nil
    }
    // Actual execution...
}
```

### Checking Previous Changes

```go
func (p *MyPlugin) Execute(ctx *plugin.ExecutionContext) error {
    for _, change := range ctx.Changes {
        if change.Type == "file_write" {
            // React to file changes...
        }
    }
    return nil
}
```

### Using Templates

```go
template, err := p.ctx.Template.Render("template.tmpl", data)
if err != nil {
    return err
}
```

## See Also

- [Architecture Documentation](../architecture.md)
- [Task List](../tasks.md)
- Plugin examples in `internal/plugins/`

