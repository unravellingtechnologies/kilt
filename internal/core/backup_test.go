package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewBackupManager(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kilt-backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	backupDir := filepath.Join(tmpDir, "backups")
	bm, err := NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create BackupManager: %v", err)
	}

	if bm == nil {
		t.Fatal("BackupManager is nil")
	}

	if bm.GetBackupDir() != backupDir {
		t.Errorf("Expected backup dir %s, got %s", backupDir, bm.GetBackupDir())
	}

	// Check that backup directory was created
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		t.Error("Backup directory should exist")
	}
}

func TestCreateBackup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kilt-backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	backupDir := filepath.Join(tmpDir, "backups")
	bm, err := NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create BackupManager: %v", err)
	}

	// Create test files
	testFile1 := filepath.Join(tmpDir, "file1.txt")
	testFile2 := filepath.Join(tmpDir, "file2.txt")
	testContent1 := "Content of file 1"
	testContent2 := "Content of file 2"

	if err := os.WriteFile(testFile1, []byte(testContent1), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(testFile2, []byte(testContent2), 0o755); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create backup
	files := []string{testFile1, testFile2}
	backupID, metadata, err := bm.CreateBackup(files, "Test backup")
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	if backupID == "" {
		t.Error("Backup ID should not be empty")
	}

	if metadata == nil {
		t.Fatal("Metadata should not be nil")
	}

	// Type assert to *BackupMetadata
	meta, ok := metadata.(*BackupMetadata)
	if !ok {
		t.Fatal("Metadata should be *BackupMetadata")
	}

	if len(meta.Files) != 2 {
		t.Errorf("Expected 2 files in backup, got %d", len(meta.Files))
	}

	if meta.TotalSize == 0 {
		t.Error("Total size should be greater than 0")
	}

	// Verify backup files exist
	backupPath := filepath.Join(backupDir, backupID)
	for _, file := range meta.Files {
		if _, err := os.Stat(file.BackupPath); os.IsNotExist(err) {
			t.Errorf("Backup file should exist: %s", file.BackupPath)
		}
	}

	// Verify metadata file exists
	metadataPath := filepath.Join(backupPath, "metadata.json")
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		t.Error("Metadata file should exist")
	}
}

func TestCreateBackupWithNonExistentFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kilt-backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	backupDir := filepath.Join(tmpDir, "backups")
	bm, err := NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create BackupManager: %v", err)
	}

	// Create one existing file and one non-existent file
	testFile1 := filepath.Join(tmpDir, "file1.txt")
	testContent1 := "Content of file 1"
	if err := os.WriteFile(testFile1, []byte(testContent1), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	nonExistentFile := filepath.Join(tmpDir, "nonexistent.txt")
	files := []string{testFile1, nonExistentFile}

	// Should succeed but only backup the existing file
	_, metadataIface, err := bm.CreateBackup(files, "Test backup")
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	metadata := metadataIface.(*BackupMetadata)
	if len(metadata.Files) != 1 {
		t.Errorf("Expected 1 file in backup, got %d", len(metadata.Files))
	}
}

func TestCreateIncrementalBackup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kilt-backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	backupDir := filepath.Join(tmpDir, "backups")
	bm, err := NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create BackupManager: %v", err)
	}

	stateDir := filepath.Join(tmpDir, "state")
	sm, err := NewStateManager(stateDir)
	if err != nil {
		t.Fatalf("Failed to create StateManager: %v", err)
	}

	// Create test files
	testFile1 := filepath.Join(tmpDir, "file1.txt")
	testFile2 := filepath.Join(tmpDir, "file2.txt")
	testContent1 := "Content of file 1"
	testContent2 := "Content of file 2"

	if err := os.WriteFile(testFile1, []byte(testContent1), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(testFile2, []byte(testContent2), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Record file state for file1 (so it won't be considered changed)
	if err := sm.UpdateFileRecord("source1.txt", testFile1); err != nil {
		t.Fatalf("Failed to update file record: %v", err)
	}

	// Don't record file2 (so it will be considered changed)

	// Create incremental backup
	files := []string{testFile1, testFile2}
	_, metadataIface, err := bm.CreateIncrementalBackup(files, sm, "Incremental backup")
	if err != nil {
		t.Fatalf("Failed to create incremental backup: %v", err)
	}

	// Should only backup file2 (file1 hasn't changed)
	if metadataIface == nil {
		t.Fatal("Metadata should not be nil")
	}

	metadata := metadataIface.(*BackupMetadata)
	if len(metadata.Files) != 1 {
		t.Errorf("Expected 1 file in backup (only changed file), got %d", len(metadata.Files))
	}

	if metadata.Files[0].OriginalPath != testFile2 {
		t.Errorf("Expected to backup file2, got %s", metadata.Files[0].OriginalPath)
	}
}

func TestCreateIncrementalBackupNoChanges(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kilt-backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	backupDir := filepath.Join(tmpDir, "backups")
	bm, err := NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create BackupManager: %v", err)
	}

	stateDir := filepath.Join(tmpDir, "state")
	sm, err := NewStateManager(stateDir)
	if err != nil {
		t.Fatalf("Failed to create StateManager: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tmpDir, "file.txt")
	testContent := "Content of file"
	if err := os.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Record file state (so it won't be considered changed)
	if err := sm.UpdateFileRecord("source.txt", testFile); err != nil {
		t.Fatalf("Failed to update file record: %v", err)
	}

	// Create incremental backup
	files := []string{testFile}
	backupID, metadata, err := bm.CreateIncrementalBackup(files, sm, "Incremental backup")
	if err != nil {
		t.Fatalf("Failed to create incremental backup: %v", err)
	}

	// Should return nil metadata (no changes)
	if metadata != nil {
		t.Error("Expected nil metadata when no files changed")
	}

	if backupID != "" {
		t.Error("Expected empty backup ID when no files changed")
	}
}

func TestLoadMetadata(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kilt-backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	backupDir := filepath.Join(tmpDir, "backups")
	bm, err := NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create BackupManager: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tmpDir, "file.txt")
	testContent := "Content of file"
	if err := os.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create backup
	files := []string{testFile}
	backupID, originalMetadataIface, err := bm.CreateBackup(files, "Test backup")
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}
	originalMetadata := originalMetadataIface.(*BackupMetadata)

	// Load metadata
	loadedMetadata, err := bm.LoadMetadata(backupID)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	if loadedMetadata.BackupID != originalMetadata.BackupID {
		t.Errorf("Expected backup ID %s, got %s", originalMetadata.BackupID, loadedMetadata.BackupID)
	}

	if len(loadedMetadata.Files) != len(originalMetadata.Files) {
		t.Errorf("Expected %d files, got %d", len(originalMetadata.Files), len(loadedMetadata.Files))
	}
}

func TestRestore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kilt-backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	backupDir := filepath.Join(tmpDir, "backups")
	bm, err := NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create BackupManager: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tmpDir, "file.txt")
	originalContent := "Original content"
	if err := os.WriteFile(testFile, []byte(originalContent), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create backup
	files := []string{testFile}
	backupID, _, err := bm.CreateBackup(files, "Test backup")
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Modify file
	modifiedContent := "Modified content"
	if err := os.WriteFile(testFile, []byte(modifiedContent), 0o644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Restore from backup
	if err := bm.Restore(backupID); err != nil {
		t.Fatalf("Failed to restore backup: %v", err)
	}

	// Verify file was restored
	restoredContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read restored file: %v", err)
	}

	if string(restoredContent) != originalContent {
		t.Errorf("Expected content %s, got %s", originalContent, string(restoredContent))
	}
}

func TestListBackups(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kilt-backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	backupDir := filepath.Join(tmpDir, "backups")
	bm, err := NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create BackupManager: %v", err)
	}

	// Create multiple backups with small delay
	testFile := filepath.Join(tmpDir, "file.txt")
	testContent := "Content"
	if err := os.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	for i := 0; i < 3; i++ {
		_, _, err := bm.CreateBackup([]string{testFile}, "Backup")
		if err != nil {
			t.Fatalf("Failed to create backup: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps
	}

	// List backups
	backups, err := bm.ListBackups()
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	if len(backups) != 3 {
		t.Errorf("Expected 3 backups, got %d", len(backups))
	}

	// Verify backups are sorted newest first
	for i := 0; i < len(backups)-1; i++ {
		if backups[i].Timestamp.Before(backups[i+1].Timestamp) {
			t.Error("Backups should be sorted newest first")
		}
	}
}

func TestGetBackupSize(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kilt-backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	backupDir := filepath.Join(tmpDir, "backups")
	bm, err := NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create BackupManager: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tmpDir, "file.txt")
	testContent := "This is test content for size calculation"
	if err := os.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create backup
	files := []string{testFile}
	backupID, metadataIface, err := bm.CreateBackup(files, "Test backup")
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}
	metadata := metadataIface.(*BackupMetadata)

	// Get backup size
	size, err := bm.GetBackupSize(backupID)
	if err != nil {
		t.Fatalf("Failed to get backup size: %v", err)
	}

	if size == 0 {
		t.Error("Backup size should be greater than 0")
	}

	// Size should include file + metadata
	if size < metadata.TotalSize {
		t.Error("Backup size should be at least the total file size")
	}
}

func TestGetTotalBackupSize(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kilt-backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	backupDir := filepath.Join(tmpDir, "backups")
	bm, err := NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create BackupManager: %v", err)
	}

	// Create multiple backups
	testFile := filepath.Join(tmpDir, "file.txt")
	testContent := "Test content"
	if err := os.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	for i := 0; i < 3; i++ {
		if _, _, err := bm.CreateBackup([]string{testFile}, "Backup"); err != nil {
			t.Fatalf("Failed to create backup: %v", err)
		}
	}

	// Get total backup size
	totalSize, err := bm.GetTotalBackupSize()
	if err != nil {
		t.Fatalf("Failed to get total backup size: %v", err)
	}

	if totalSize == 0 {
		t.Error("Total backup size should be greater than 0")
	}
}

func TestDeleteBackup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kilt-backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	backupDir := filepath.Join(tmpDir, "backups")
	bm, err := NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create BackupManager: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tmpDir, "file.txt")
	testContent := "Content"
	if err := os.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create backup
	files := []string{testFile}
	backupID, _, err := bm.CreateBackup(files, "Test backup")
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Verify backup exists
	backupPath := filepath.Join(backupDir, backupID)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatal("Backup should exist before deletion")
	}

	// Delete backup
	if err := bm.DeleteBackup(backupID); err != nil {
		t.Fatalf("Failed to delete backup: %v", err)
	}

	// Verify backup is deleted
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Error("Backup should be deleted")
	}
}

func TestCleanupOldBackups(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kilt-backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	backupDir := filepath.Join(tmpDir, "backups")
	bm, err := NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create BackupManager: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tmpDir, "file.txt")
	testContent := "Content"
	if err := os.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create 5 backups
	backupIDs := make([]string, 0)
	for i := 0; i < 5; i++ {
		backupID, _, err := bm.CreateBackup([]string{testFile}, "Backup")
		if err != nil {
			t.Fatalf("Failed to create backup: %v", err)
		}
		backupIDs = append(backupIDs, backupID)
		time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps
	}

	// Cleanup, keeping only 2 backups
	if err := bm.CleanupOldBackups(2); err != nil {
		t.Fatalf("Failed to cleanup old backups: %v", err)
	}

	// Verify only 2 backups remain
	backups, err := bm.ListBackups()
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	if len(backups) != 2 {
		t.Errorf("Expected 2 backups after cleanup, got %d", len(backups))
	}

	// Verify the newest backups are kept
	// backups are sorted newest first
	expectedKept := backupIDs[len(backupIDs)-2:] // Last 2 (newest)
	for i, backup := range backups {
		if backup.BackupID != expectedKept[len(expectedKept)-1-i] {
			t.Errorf("Expected to keep backup %s, but got %s", expectedKept[len(expectedKept)-1-i], backup.BackupID)
		}
	}
}

func TestCleanupOldBackupsWithFewerBackups(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kilt-backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	backupDir := filepath.Join(tmpDir, "backups")
	bm, err := NewBackupManager(backupDir)
	if err != nil {
		t.Fatalf("Failed to create BackupManager: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tmpDir, "file.txt")
	testContent := "Content"
	if err := os.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create only 2 backups
	for i := 0; i < 2; i++ {
		if _, _, err := bm.CreateBackup([]string{testFile}, "Backup"); err != nil {
			t.Fatalf("Failed to create backup: %v", err)
		}
	}

	// Try to cleanup keeping 5 backups (more than we have)
	if err := bm.CleanupOldBackups(5); err != nil {
		t.Fatalf("Failed to cleanup old backups: %v", err)
	}

	// Verify all backups still exist
	backups, err := bm.ListBackups()
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	if len(backups) != 2 {
		t.Errorf("Expected 2 backups after cleanup (none should be deleted), got %d", len(backups))
	}
}
