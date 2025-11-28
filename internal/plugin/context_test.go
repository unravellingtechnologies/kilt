package plugin

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConfig implements Config interface for testing
type mockConfig struct {
	pluginConfigs map[string]map[string]interface{}
}

func (m *mockConfig) GetPluginConfig(pluginName string) map[string]interface{} {
	if config, exists := m.pluginConfigs[pluginName]; exists {
		return config
	}
	return make(map[string]interface{})
}

// mockLogger implements Logger interface for testing
type mockLogger struct {
	debugLogs []string
	infoLogs  []string
	warnLogs  []string
	errorLogs []string
}

func (m *mockLogger) Debug(msg string, fields ...interface{}) {
	m.debugLogs = append(m.debugLogs, msg)
}
func (m *mockLogger) Info(msg string, fields ...interface{}) {
	m.infoLogs = append(m.infoLogs, msg)
}
func (m *mockLogger) Warn(msg string, fields ...interface{}) {
	m.warnLogs = append(m.warnLogs, msg)
}
func (m *mockLogger) Error(msg string, fields ...interface{}) {
	m.errorLogs = append(m.errorLogs, msg)
}

func TestExecutionContext_AddChange(t *testing.T) {
	ctx := &ExecutionContext{
		PluginContext: &PluginContext{},
		Changes:       []Change{},
		Errors:        []error{},
		StartTime:     time.Now(),
	}

	change := Change{
		Type:        "test_change",
		Files:       []string{"file1", "file2"},
		Description: "Test change",
	}

	ctx.AddChange(change)

	require.Len(t, ctx.Changes, 1)

	addedChange := ctx.Changes[0]
	assert.Equal(t, "test_change", addedChange.Type)
	assert.Len(t, addedChange.Files, 2)
	assert.False(t, addedChange.Timestamp.IsZero(), "AddChange() change.Timestamp should be set")
}

func TestExecutionContext_AddError(t *testing.T) {
	ctx := &ExecutionContext{
		PluginContext: &PluginContext{},
		Changes:       []Change{},
		Errors:        []error{},
		StartTime:     time.Now(),
	}

	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	ctx.AddError(err1)
	ctx.AddError(nil) // Should not add nil errors
	ctx.AddError(err2)

	require.Len(t, ctx.Errors, 2)
	assert.Equal(t, err1, ctx.Errors[0])
	assert.Equal(t, err2, ctx.Errors[1])
}

func TestExecutionContext_HasErrors(t *testing.T) {
	tests := []struct {
		name   string
		errors []error
		want   bool
	}{
		{
			name:   "no errors",
			errors: []error{},
			want:   false,
		},
		{
			name:   "has errors",
			errors: []error{errors.New("test error")},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &ExecutionContext{
				PluginContext: &PluginContext{},
				Changes:       []Change{},
				Errors:        tt.errors,
				StartTime:     time.Now(),
			}

			assert.Equal(t, tt.want, ctx.HasErrors())
		})
	}
}

func TestGetPluginConfig(t *testing.T) {
	config := &mockConfig{
		pluginConfigs: map[string]map[string]interface{}{
			"test-plugin": {
				"key1": "value1",
				"key2": 42,
			},
		},
	}

	tests := []struct {
		name       string
		pluginName string
		wantKeys   []string
	}{
		{
			name:       "existing plugin config",
			pluginName: "test-plugin",
			wantKeys:   []string{"key1", "key2"},
		},
		{
			name:       "non-existing plugin config",
			pluginName: "non-existent",
			wantKeys:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPluginConfig(config, tt.pluginName)
			assert.Len(t, result, len(tt.wantKeys))
		})
	}

	// Test with nil config
	result := GetPluginConfig(nil, "test-plugin")
	assert.NotNil(t, result, "GetPluginConfig() with nil config should return empty map, not nil")
	assert.Empty(t, result, "GetPluginConfig() with nil config should return empty map")
}
