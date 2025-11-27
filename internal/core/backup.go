package core

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// BackupManager manages file backups before modification
type BackupManager struct {
	backupDir string
	mu        sync.RWMutex
}

// BackupMetadata contains information about a backup
type BackupMetadata struct {
	Timestamp   time.Time         `json:"timestamp"`
	BackupID    string            `json:"backup_id"`
	Files       []BackedUpFile    `json:"files"`
	TotalSize   int64             `json:"total_size"`
	Description string            `json:"description,omitempty"`
}

// BackedUpFile represents a single file in a backup
type BackedUpFile struct {
	OriginalPath string `json:"original_path"`
	BackupPath   string `json:"backup_path"`
	Size         int64  `json:"size"`
	Mode         os.FileMode `json:"mode"`
}

// NewBackupManager creates a new backup manager
func NewBackupManager(backupDir string) (*BackupManager, error) {
	// Ensure backup directory exists
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	return &BackupManager{
		backupDir: backupDir,
	}, nil
}

// CreateBackup creates a new backup with the given files
// Returns the backup ID (timestamp format: YYYYMMDD-HHMMSS)
func (bm *BackupManager) CreateBackup(files []string, description string) (string, *BackupMetadata, error) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	return bm.createBackupLocked(files, description)
}

// createBackupLocked creates a backup without acquiring the lock (internal use)
// Caller must hold bm.mu.Lock()
func (bm *BackupManager) createBackupLocked(files []string, description string) (string, *BackupMetadata, error) {
	now := time.Now()
	baseBackupID := now.Format("20060102-150405")
	backupID := baseBackupID
	backupPath := filepath.Join(bm.backupDir, backupID)

	// Handle collisions: if directory exists, append a counter
	counter := 0
	for {
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			break
		}
		counter++
		backupID = fmt.Sprintf("%s-%d", baseBackupID, counter)
		backupPath = filepath.Join(bm.backupDir, backupID)
	}

	// Create backup directory
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return "", nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	metadata := &BackupMetadata{
		Timestamp:   now,
		BackupID:    backupID,
		Files:       make([]BackedUpFile, 0),
		TotalSize:   0,
		Description: description,
	}

	// Backup each file
	for _, filePath := range files {
		backedUpFile, err := bm.backupFile(filePath, backupPath)
		if err != nil {
			// If file doesn't exist, skip it (might be a new file)
			if os.IsNotExist(err) {
				continue
			}
			return "", nil, fmt.Errorf("failed to backup file %s: %w", filePath, err)
		}
		if backedUpFile != nil {
			metadata.Files = append(metadata.Files, *backedUpFile)
			metadata.TotalSize += backedUpFile.Size
		}
	}

	// Save metadata
	if err := bm.saveMetadata(backupPath, metadata); err != nil {
		return "", nil, fmt.Errorf("failed to save backup metadata: %w", err)
	}

	return backupID, metadata, nil
}

// CreateIncrementalBackup creates a backup only for files that have changed
// Uses StateManager to check if files have changed
func (bm *BackupManager) CreateIncrementalBackup(files []string, stateManager *StateManager, description string) (string, *BackupMetadata, error) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// Filter files that have changed
	changedFiles := make([]string, 0)
	for _, filePath := range files {
		changed, err := stateManager.HasFileChanged(filePath)
		if err != nil {
			// If we can't determine, include it in backup to be safe
			changedFiles = append(changedFiles, filePath)
			continue
		}
		if changed {
			changedFiles = append(changedFiles, filePath)
		}
	}

	// If no files changed, return empty backup
	if len(changedFiles) == 0 {
		return "", nil, nil
	}

	// Create backup with only changed files (using internal method since we already hold the lock)
	return bm.createBackupLocked(changedFiles, description)
}

// backupFile backs up a single file to the backup directory
func (bm *BackupManager) backupFile(filePath, backupDir string) (*BackedUpFile, error) {
	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	// Skip directories
	if info.IsDir() {
		return nil, fmt.Errorf("cannot backup directory: %s", filePath)
	}

	// Create relative path structure in backup
	// Use absolute path to avoid conflicts
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create a safe backup path (replace / with _ to avoid directory structure issues)
	// Or maintain directory structure by using the full path
	backupFileName := filepath.Base(filePath)
	backupFilePath := filepath.Join(backupDir, backupFileName)

	// If file already exists in backup (same name), add a suffix
	counter := 1
	for {
		if _, err := os.Stat(backupFilePath); os.IsNotExist(err) {
			break
		}
		ext := filepath.Ext(backupFileName)
		name := backupFileName[:len(backupFileName)-len(ext)]
		backupFilePath = filepath.Join(backupDir, fmt.Sprintf("%s_%d%s", name, counter, ext))
		counter++
	}

	// Copy file
	if err := copyFile(filePath, backupFilePath); err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	// Preserve file permissions
	if err := os.Chmod(backupFilePath, info.Mode()); err != nil {
		return nil, fmt.Errorf("failed to set file permissions: %w", err)
	}

	return &BackedUpFile{
		OriginalPath: absPath,
		BackupPath:   backupFilePath,
		Size:         info.Size(),
		Mode:         info.Mode(),
	}, nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create destination directory if needed
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// saveMetadata saves backup metadata to a JSON file
func (bm *BackupManager) saveMetadata(backupPath string, metadata *BackupMetadata) error {
	metadataPath := filepath.Join(backupPath, "metadata.json")
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return os.WriteFile(metadataPath, data, 0644)
}

// LoadMetadata loads backup metadata from a backup directory
func (bm *BackupManager) LoadMetadata(backupID string) (*BackupMetadata, error) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.loadMetadataLocked(backupID)
}

// loadMetadataLocked loads metadata without acquiring the lock (internal use)
// Caller must hold bm.mu.RLock() or bm.mu.Lock()
func (bm *BackupManager) loadMetadataLocked(backupID string) (*BackupMetadata, error) {
	backupPath := filepath.Join(bm.backupDir, backupID)
	metadataPath := filepath.Join(backupPath, "metadata.json")

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata BackupMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// Restore restores files from a backup
func (bm *BackupManager) Restore(backupID string) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// Load metadata (using locked version since we already hold the lock)
	metadata, err := bm.loadMetadataLocked(backupID)
	if err != nil {
		return fmt.Errorf("failed to load backup metadata for %s: %w\nHint: Verify the backup ID is correct and the backup directory is accessible", backupID, err)
	}

	// Validate backup integrity
	if len(metadata.Files) == 0 {
		return fmt.Errorf("backup %s contains no files - backup may be corrupted or incomplete", backupID)
	}

	// Restore each file
	restoredCount := 0
	for i, file := range metadata.Files {
		// Check if backup file exists
		if _, err := os.Stat(file.BackupPath); os.IsNotExist(err) {
			return fmt.Errorf("backup file not found: %s\nThis indicates backup corruption. File %d of %d failed to restore.\nHint: Check backup integrity or try a different backup", file.BackupPath, i+1, len(metadata.Files))
		}

		// Create destination directory if needed
		destDir := filepath.Dir(file.OriginalPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create destination directory for %s: %w\nHint: Check filesystem permissions and available disk space", file.OriginalPath, err)
		}

		// Copy file back
		if err := copyFile(file.BackupPath, file.OriginalPath); err != nil {
			return fmt.Errorf("failed to restore file %s (file %d of %d): %w\nHint: Check filesystem permissions and available disk space", file.OriginalPath, i+1, len(metadata.Files), err)
		}

		// Restore file permissions
		if err := os.Chmod(file.OriginalPath, file.Mode); err != nil {
			return fmt.Errorf("failed to restore file permissions for %s: %w\nHint: Check filesystem permissions", file.OriginalPath, err)
		}

		restoredCount++
	}

	// Verify all files were restored
	if restoredCount != len(metadata.Files) {
		return fmt.Errorf("restore incomplete: restored %d of %d files", restoredCount, len(metadata.Files))
	}

	return nil
}

// ListBackups returns a list of all backups sorted by timestamp (newest first)
func (bm *BackupManager) ListBackups() ([]*BackupMetadata, error) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.listBackupsLocked()
}

// listBackupsLocked lists backups without acquiring the lock (internal use)
// Caller must hold bm.mu.RLock() or bm.mu.Lock()
func (bm *BackupManager) listBackupsLocked() ([]*BackupMetadata, error) {
	entries, err := os.ReadDir(bm.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	backups := make([]*BackupMetadata, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		backupID := entry.Name()
		metadata, err := bm.loadMetadataLocked(backupID)
		if err != nil {
			// Skip invalid backups
			continue
		}

		backups = append(backups, metadata)
	}

	// Sort by timestamp (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// GetBackupSize calculates the total size of a backup directory
func (bm *BackupManager) GetBackupSize(backupID string) (int64, error) {
	backupPath := filepath.Join(bm.backupDir, backupID)
	return bm.calculateDirSize(backupPath)
}

// calculateDirSize calculates the total size of a directory
func (bm *BackupManager) calculateDirSize(dirPath string) (int64, error) {
	var size int64

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}

// GetTotalBackupSize calculates the total size of all backups
func (bm *BackupManager) GetTotalBackupSize() (int64, error) {
	return bm.calculateDirSize(bm.backupDir)
}

// DeleteBackup deletes a backup
func (bm *BackupManager) DeleteBackup(backupID string) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	return bm.deleteBackupLocked(backupID)
}

// deleteBackupLocked deletes a backup without acquiring the lock (internal use)
// Caller must hold bm.mu.Lock()
func (bm *BackupManager) deleteBackupLocked(backupID string) error {
	backupPath := filepath.Join(bm.backupDir, backupID)
	return os.RemoveAll(backupPath)
}

// CleanupOldBackups removes old backups, keeping only the last N backups
func (bm *BackupManager) CleanupOldBackups(keepCount int) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	backups, err := bm.listBackupsLocked()
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	// If we have fewer backups than keepCount, nothing to do
	if len(backups) <= keepCount {
		return nil
	}

	// Delete oldest backups (they're sorted newest first)
	for i := keepCount; i < len(backups); i++ {
		if err := bm.deleteBackupLocked(backups[i].BackupID); err != nil {
			return fmt.Errorf("failed to delete backup %s: %w", backups[i].BackupID, err)
		}
	}

	return nil
}

// GetBackupDir returns the backup directory path
func (bm *BackupManager) GetBackupDir() string {
	return bm.backupDir
}

