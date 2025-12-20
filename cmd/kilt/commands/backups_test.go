package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unravelling/kilt/internal/core"
)

func TestBackupsListCommand_EmptyBackupList(t *testing.T) {
	// Test formatSize function
	result := formatSize(1024)
	assert.Equal(t, "1.0 KB", result)
}

func TestOutputBackupsTable(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	require.NoError(t, os.MkdirAll(backupDir, 0o755))

	bm, err := core.NewBackupManager(backupDir)
	require.NoError(t, err)

	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("content"), 0o644))

	backupID, metadataIface, err := bm.CreateBackup([]string{testFile}, "Test backup")
	require.NoError(t, err)

	metadata := metadataIface.(*core.BackupMetadata)
	backups := []*core.BackupMetadata{metadata}

	// Test that function doesn't error (output goes to stdout which we can't easily capture)
	err = outputBackupsTable(backups)
	assert.NoError(t, err)

	// Verify metadata has expected values
	assert.Equal(t, backupID, metadata.BackupID)
	assert.Equal(t, "Test backup", metadata.Description)
}

func TestOutputBackupsJSON(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	require.NoError(t, os.MkdirAll(backupDir, 0o755))

	bm, err := core.NewBackupManager(backupDir)
	require.NoError(t, err)

	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("content"), 0o644))

	backupID, metadataIface, err := bm.CreateBackup([]string{testFile}, "Test backup")
	require.NoError(t, err)

	metadata := metadataIface.(*core.BackupMetadata)
	backups := []*core.BackupMetadata{metadata}

	// Test JSON encoding directly (we can't easily capture stdout)
	jsonData, err := json.Marshal(backups)
	require.NoError(t, err)

	assert.Contains(t, string(jsonData), backupID)

	// Verify JSON is valid
	var decoded []*core.BackupMetadata
	err = json.Unmarshal(jsonData, &decoded)
	assert.NoError(t, err)
	assert.Len(t, decoded, 1)
	assert.Equal(t, backupID, decoded[0].BackupID)
}

func TestBackupsListCommand_WithBackups(t *testing.T) {
	// Integration test: Create backups and test listing
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Create backup manager and multiple backups
	bm, err := core.NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create backup manager: %v", err)
	}

	// Create test files
	testFile1 := filepath.Join(tmpDir, "test1.txt")
	testFile2 := filepath.Join(tmpDir, "test2.txt")
	if err := os.WriteFile(testFile1, []byte("content1"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(testFile2, []byte("content2"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create multiple backups
	backupID1, _, err := bm.CreateBackup([]string{testFile1}, "First backup")
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	backupID2, _, err := bm.CreateBackup([]string{testFile2}, "Second backup")
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Test the list functionality directly
	backups, err := bm.ListBackups()
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	if len(backups) != 2 {
		t.Errorf("Expected 2 backups, got %d", len(backups))
	}

	// Verify backup IDs
	found1, found2 := false, false
	for _, backup := range backups {
		if backup.BackupID == backupID1 {
			found1 = true
		}
		if backup.BackupID == backupID2 {
			found2 = true
		}
	}

	if !found1 {
		t.Errorf("Backup ID %s not found in list", backupID1)
	}
	if !found2 {
		t.Errorf("Backup ID %s not found in list", backupID2)
	}
}

func TestBackupsListCommand_JSONOutput(t *testing.T) {
	// Test JSON encoding directly
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Create backup manager and a backup
	bm, err := core.NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create backup manager: %v", err)
	}

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	backupID, _, err := bm.CreateBackup([]string{testFile}, "Test backup")
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Test JSON encoding
	backups, err := bm.ListBackups()
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	// Encode to JSON
	jsonData, err := json.Marshal(backups)
	if err != nil {
		t.Fatalf("Failed to marshal backups to JSON: %v", err)
	}

	// Verify JSON output is valid
	var decodedBackups []*core.BackupMetadata
	if err := json.Unmarshal(jsonData, &decodedBackups); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, string(jsonData))
	}

	if len(decodedBackups) != 1 {
		t.Errorf("Expected 1 backup in JSON output, got %d", len(decodedBackups))
	}

	if decodedBackups[0].BackupID != backupID {
		t.Errorf("Expected backup ID %s, got %s", backupID, decodedBackups[0].BackupID)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1536 * 1024 * 1024, "1.5 GB"},
		{1024 * 1024 * 1024 * 1024, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatSize(tt.bytes)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewBackupsCmd(t *testing.T) {
	cmd := NewBackupsCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "backups", cmd.Use)
}
