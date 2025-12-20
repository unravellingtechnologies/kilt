package commands

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestoreCommand_InvalidBackupID(t *testing.T) {
	cmd := NewRestoreCmd()
	cmd.SetArgs([]string{"invalid-id"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "invalid backup id format")
}

func TestRestoreCommand_NonExistentBackup(t *testing.T) {
	// Set up test environment with KILT_BACKUP_DIR
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Use environment variable to override backup directory
	originalEnv := os.Getenv("KILT_BACKUP_DIR")
	if err := os.Setenv("KILT_BACKUP_DIR", backupDir); err != nil {
		t.Fatalf("Failed to set KILT_BACKUP_DIR: %v", err)
	}
	defer func() {
		if originalEnv != "" {
			os.Setenv("KILT_BACKUP_DIR", originalEnv)
		} else {
			os.Unsetenv("KILT_BACKUP_DIR")
		}
	}()

	// Note: This test will fail if GetBackupDir doesn't check KILT_BACKUP_DIR
	// For now, we'll test the validation logic which doesn't require file system access
	// Use --force to skip the confirmation prompt that would block on fmt.Scanln()
	cmd := NewRestoreCmd()
	cmd.SetArgs([]string{"20250101-120000", "--force"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	// This will fail because backup doesn't exist, but we're testing the command structure
	err := cmd.Execute()
	// We expect an error (backup not found), which is fine for this test
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "backup not found")
}

func TestNewRestoreCmd(t *testing.T) {
	cmd := NewRestoreCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "restore <backup-id>", cmd.Use)
}

func TestBackupIDPattern(t *testing.T) {
	tests := []struct {
		id     string
		valid  bool
		reason string
	}{
		{"20250101-120000", true, "valid format"},
		{"20250101-120000-1", true, "valid format with counter"},
		{"20250101-120000-123", true, "valid format with large counter"},
		{"invalid", false, "invalid format"},
		{"20250101", false, "missing time"},
		{"2025-01-01-12:00:00", false, "wrong separators"},
		{"", false, "empty string"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			valid := backupIDPattern.MatchString(tt.id)
			if valid != tt.valid {
				t.Errorf("backupIDPattern.MatchString(%q) = %v, want %v (%s)", tt.id, valid, tt.valid, tt.reason)
			}
		})
	}
}
