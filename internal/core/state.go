// Package core provides the core engine and configuration management
// for the Kilt dotfiles manager. It includes configuration parsing,
// state management, backup operations, template rendering, and the
// main orchestration engine.
package core

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// StateManager manages execution state for idempotent operations
type StateManager struct {
	stateDir string
	db       *StateDB
	mu       sync.RWMutex
	lockFile *os.File
}

// StateDB represents the state database structure
type StateDB struct {
	RunOnce  map[string]RunOnceRecord `json:"run_once"`
	Files    map[string]FileRecord    `json:"files"`
	Plugins  map[string]PluginRecord  `json:"plugins"`
	LastSync time.Time                `json:"last_sync"`
}

// RunOnceRecord tracks execution of run-once tasks
type RunOnceRecord struct {
	TaskID     string    `json:"task_id"`
	ExecutedAt time.Time `json:"executed_at"`
	ExitCode   int       `json:"exit_code"`
	Output     string    `json:"output"`
}

// FileRecord tracks file state and checksums
type FileRecord struct {
	SourcePath string    `json:"source_path"`
	TargetPath string    `json:"target_path"`
	Checksum   string    `json:"checksum"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// PluginRecord tracks plugin execution history
type PluginRecord struct {
	PluginName string    `json:"plugin_name"`
	Version    string    `json:"version"`
	ExecutedAt time.Time `json:"executed_at"`
	Success    bool      `json:"success"`
	Duration   string    `json:"duration"`
}

// NewStateManager creates a new state manager
func NewStateManager(stateDir string) (*StateManager, error) {
	// Ensure state directory exists
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	sm := &StateManager{
		stateDir: stateDir,
		db: &StateDB{
			RunOnce: make(map[string]RunOnceRecord),
			Files:   make(map[string]FileRecord),
			Plugins: make(map[string]PluginRecord),
		},
	}

	// Load existing state
	if err := sm.load(); err != nil {
		// If state file doesn't exist, that's okay - start fresh
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load state: %w", err)
		}
	}

	return sm, nil
}

// load loads the state from disk
func (sm *StateManager) load() error {
	stateFile := filepath.Join(sm.stateDir, "state.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return err
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if err := json.Unmarshal(data, sm.db); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Initialize maps if they're nil (for backward compatibility)
	if sm.db.RunOnce == nil {
		sm.db.RunOnce = make(map[string]RunOnceRecord)
	}
	if sm.db.Files == nil {
		sm.db.Files = make(map[string]FileRecord)
	}
	if sm.db.Plugins == nil {
		sm.db.Plugins = make(map[string]PluginRecord)
	}

	return nil
}

// save saves the state to disk atomically
// Note: This method assumes the caller already holds the lock
func (sm *StateManager) save() error {
	stateFile := filepath.Join(sm.stateDir, "state.json")
	tempFile := stateFile + ".tmp"

	// Marshal to JSON
	data, err := json.MarshalIndent(sm.db, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temp file
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp state file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, stateFile); err != nil {
		return fmt.Errorf("failed to rename temp state file: %w", err)
	}

	return nil
}

// AcquireLock acquires a file-based lock for concurrent execution safety
func (sm *StateManager) AcquireLock() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.lockFile != nil {
		return fmt.Errorf("lock already acquired")
	}

	lockPath := filepath.Join(sm.stateDir, ".lock")
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("state is locked by another process")
		}
		return fmt.Errorf("failed to create lock file: %w", err)
	}

	// Write PID to lock file for debugging
	pid := fmt.Sprintf("%d\n", os.Getpid())
	if _, err := lockFile.WriteString(pid); err != nil {
		lockFile.Close()
		os.Remove(lockPath)
		return fmt.Errorf("failed to write PID to lock file: %w", err)
	}

	sm.lockFile = lockFile
	return nil
}

// ReleaseLock releases the file-based lock
func (sm *StateManager) ReleaseLock() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.lockFile == nil {
		return nil // Already released
	}

	lockPath := filepath.Join(sm.stateDir, ".lock")
	if err := sm.lockFile.Close(); err != nil {
		return fmt.Errorf("failed to close lock file: %w", err)
	}

	if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock file: %w", err)
	}

	sm.lockFile = nil
	return nil
}

// IsTaskCompleted checks if a run-once task has been completed
func (sm *StateManager) IsTaskCompleted(taskID string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	_, exists := sm.db.RunOnce[taskID]
	return exists
}

// MarkTaskCompleted marks a run-once task as completed
// This is the internal method with specific type
func (sm *StateManager) markTaskCompletedInternal(taskID string, record RunOnceRecord) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	record.TaskID = taskID
	if record.ExecutedAt.IsZero() {
		record.ExecutedAt = time.Now()
	}

	sm.db.RunOnce[taskID] = record
	return sm.save()
}

// MarkTaskCompleted marks a run-once task as completed (plugin interface compatible)
func (sm *StateManager) MarkTaskCompleted(taskID string, record interface{}) error {
	runOnceRecord, ok := record.(RunOnceRecord)
	if !ok {
		// Try pointer type
		if ptr, ok := record.(*RunOnceRecord); ok {
			runOnceRecord = *ptr
		} else {
			return fmt.Errorf("invalid record type for task %s: expected RunOnceRecord", taskID)
		}
	}
	return sm.markTaskCompletedInternal(taskID, runOnceRecord)
}

// GetRunOnceRecord retrieves a run-once task record (plugin interface compatible)
func (sm *StateManager) GetRunOnceRecord(taskID string) (interface{}, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	record, exists := sm.db.RunOnce[taskID]
	if !exists {
		return nil, false
	}
	return &record, true
}

// GetRunOnceRecordTyped retrieves a run-once task record with specific type (internal use)
func (sm *StateManager) GetRunOnceRecordTyped(taskID string) (*RunOnceRecord, bool) {
	record, exists := sm.GetRunOnceRecord(taskID)
	if !exists {
		return nil, false
	}
	return record.(*RunOnceRecord), true
}

// DeleteRunOnceRecord removes a run-once task record (allows re-execution)
func (sm *StateManager) DeleteRunOnceRecord(taskID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.db.RunOnce, taskID)
	return sm.save()
}

// CalculateChecksum calculates SHA256 checksum of a file
func CalculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// GetFileChecksum gets the stored checksum for a file
func (sm *StateManager) GetFileChecksum(targetPath string) (string, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	record, exists := sm.db.Files[targetPath]
	if !exists {
		return "", false
	}
	return record.Checksum, true
}

// HasFileChanged checks if a file has changed since last sync
func (sm *StateManager) HasFileChanged(targetPath string) (bool, error) {
	// Check if file exists
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		// File doesn't exist - check if we have a record
		sm.mu.RLock()
		_, hasRecord := sm.db.Files[targetPath]
		sm.mu.RUnlock()
		return hasRecord, nil // Changed if we had a record but file is gone
	}

	// Calculate current checksum
	currentChecksum, err := CalculateChecksum(targetPath)
	if err != nil {
		return false, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Get stored checksum
	storedChecksum, exists := sm.GetFileChecksum(targetPath)
	if !exists {
		return true, nil // New file
	}

	return currentChecksum != storedChecksum, nil
}

// UpdateFileRecord updates the file record with new checksum
func (sm *StateManager) UpdateFileRecord(sourcePath, targetPath string) error {
	checksum, err := CalculateChecksum(targetPath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.db.Files[targetPath] = FileRecord{
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Checksum:   checksum,
		UpdatedAt:  time.Now(),
	}

	return sm.save()
}

// RemoveFileRecord removes a file record
func (sm *StateManager) RemoveFileRecord(targetPath string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.db.Files, targetPath)
	return sm.save()
}

// RecordPluginExecution records a plugin execution
func (sm *StateManager) RecordPluginExecution(pluginName, version string, success bool, duration time.Duration) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.db.Plugins[pluginName] = PluginRecord{
		PluginName: pluginName,
		Version:    version,
		ExecutedAt: time.Now(),
		Success:    success,
		Duration:   duration.String(),
	}

	return sm.save()
}

// GetPluginRecord retrieves a plugin execution record (plugin interface compatible)
func (sm *StateManager) GetPluginRecord(pluginName string) (interface{}, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	record, exists := sm.db.Plugins[pluginName]
	if !exists {
		return nil, false
	}
	return &record, true
}

// GetPluginRecordTyped retrieves a plugin execution record with specific type (internal use)
func (sm *StateManager) GetPluginRecordTyped(pluginName string) (*PluginRecord, bool) {
	record, exists := sm.GetPluginRecord(pluginName)
	if !exists {
		return nil, false
	}
	return record.(*PluginRecord), true
}

// UpdateLastSync updates the last sync timestamp
func (sm *StateManager) UpdateLastSync() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.db.LastSync = time.Now()
	return sm.save()
}

// GetLastSync returns the last sync timestamp
func (sm *StateManager) GetLastSync() time.Time {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.db.LastSync
}

// ClearState clears all state (used by reset command)
func (sm *StateManager) ClearState() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.db = &StateDB{
		RunOnce: make(map[string]RunOnceRecord),
		Files:   make(map[string]FileRecord),
		Plugins: make(map[string]PluginRecord),
	}

	return sm.save()
}

// Cleanup removes old records based on retention policy
func (sm *StateManager) Cleanup(maxAge time.Duration) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	cleaned := false

	// Clean old run-once records (keep all for now, but could add age-based cleanup)
	// For now, we'll keep all run-once records as they're important for idempotency

	// Clean old file records
	for path, record := range sm.db.Files {
		if record.UpdatedAt.Before(cutoff) {
			delete(sm.db.Files, path)
			cleaned = true
		}
	}

	// Clean old plugin records
	for name, record := range sm.db.Plugins {
		if record.ExecutedAt.Before(cutoff) {
			delete(sm.db.Plugins, name)
			cleaned = true
		}
	}

	if cleaned {
		return sm.save()
	}

	return nil
}

// GetStateDir returns the state directory path
func (sm *StateManager) GetStateDir() string {
	return sm.stateDir
}
