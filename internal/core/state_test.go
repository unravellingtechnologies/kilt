package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStateManager(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kilt-state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm, err := NewStateManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create StateManager: %v", err)
	}

	if sm == nil {
		t.Fatal("StateManager is nil")
	}

	if sm.GetStateDir() != tmpDir {
		t.Errorf("Expected state dir %s, got %s", tmpDir, sm.GetStateDir())
	}
}

func TestStateManagerLocking(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kilt-state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm, err := NewStateManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create StateManager: %v", err)
	}

	// Acquire lock
	if err := sm.AcquireLock(); err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	// Try to acquire lock again (should fail)
	if err := sm.AcquireLock(); err == nil {
		t.Error("Expected error when acquiring lock twice, got nil")
	}

	// Release lock
	if err := sm.ReleaseLock(); err != nil {
		t.Fatalf("Failed to release lock: %v", err)
	}

	// Acquire lock again (should succeed)
	if err := sm.AcquireLock(); err != nil {
		t.Fatalf("Failed to acquire lock after release: %v", err)
	}

	if err := sm.ReleaseLock(); err != nil {
		t.Fatalf("Failed to release lock: %v", err)
	}
}

func TestRunOnceTasks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kilt-state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm, err := NewStateManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create StateManager: %v", err)
	}

	taskID := "test-task-1"

	// Task should not be completed initially
	if sm.IsTaskCompleted(taskID) {
		t.Error("Task should not be completed initially")
	}

	// Mark task as completed
	record := RunOnceRecord{
		ExitCode: 0,
		Output:   "Task completed successfully",
	}
	if err := sm.MarkTaskCompleted(taskID, record); err != nil {
		t.Fatalf("Failed to mark task as completed: %v", err)
	}

	// Task should now be completed
	if !sm.IsTaskCompleted(taskID) {
		t.Error("Task should be completed after marking")
	}

	// Retrieve record (use typed version)
	retrieved, exists := sm.GetRunOnceRecordTyped(taskID)
	if !exists {
		t.Fatal("Record should exist")
	}
	if retrieved.TaskID != taskID {
		t.Errorf("Expected task ID %s, got %s", taskID, retrieved.TaskID)
	}
	if retrieved.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", retrieved.ExitCode)
	}
}

func TestFileChecksums(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kilt-state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm, err := NewStateManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create StateManager: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate checksum
	checksum1, err := CalculateChecksum(testFile)
	if err != nil {
		t.Fatalf("Failed to calculate checksum: %v", err)
	}
	if checksum1 == "" {
		t.Error("Checksum should not be empty")
	}

	// Update file record
	if err := sm.UpdateFileRecord("source.txt", testFile); err != nil {
		t.Fatalf("Failed to update file record: %v", err)
	}

	// Get stored checksum
	storedChecksum, exists := sm.GetFileChecksum(testFile)
	if !exists {
		t.Fatal("File record should exist")
	}
	if storedChecksum != checksum1 {
		t.Errorf("Expected checksum %s, got %s", checksum1, storedChecksum)
	}

	// Check if file has changed (should be false)
	changed, err := sm.HasFileChanged(testFile)
	if err != nil {
		t.Fatalf("Failed to check if file changed: %v", err)
	}
	if changed {
		t.Error("File should not have changed")
	}

	// Modify file
	if err := os.WriteFile(testFile, []byte("Modified content"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Check if file has changed (should be true)
	changed, err = sm.HasFileChanged(testFile)
	if err != nil {
		t.Fatalf("Failed to check if file changed: %v", err)
	}
	if !changed {
		t.Error("File should have changed")
	}
}

func TestPluginRecords(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kilt-state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm, err := NewStateManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create StateManager: %v", err)
	}

	pluginName := "test-plugin"
	version := "1.0.0"

	// Record plugin execution
	duration := 100 * time.Millisecond
	if err := sm.RecordPluginExecution(pluginName, version, true, duration); err != nil {
		t.Fatalf("Failed to record plugin execution: %v", err)
	}

	// Retrieve record (use typed version)
	record, exists := sm.GetPluginRecordTyped(pluginName)
	if !exists {
		t.Fatal("Plugin record should exist")
	}
	if record.PluginName != pluginName {
		t.Errorf("Expected plugin name %s, got %s", pluginName, record.PluginName)
	}
	if record.Version != version {
		t.Errorf("Expected version %s, got %s", version, record.Version)
	}
	if !record.Success {
		t.Error("Plugin execution should be marked as successful")
	}
}

func TestLastSync(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kilt-state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm, err := NewStateManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create StateManager: %v", err)
	}

	// Initial last sync should be zero
	lastSync := sm.GetLastSync()
	if !lastSync.IsZero() {
		t.Error("Initial last sync should be zero")
	}

	// Update last sync
	if err := sm.UpdateLastSync(); err != nil {
		t.Fatalf("Failed to update last sync: %v", err)
	}

	// Check last sync
	newLastSync := sm.GetLastSync()
	if newLastSync.IsZero() {
		t.Error("Last sync should not be zero after update")
	}
	if newLastSync.Before(time.Now().Add(-1 * time.Second)) {
		t.Error("Last sync should be recent")
	}
}

func TestClearState(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kilt-state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm, err := NewStateManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create StateManager: %v", err)
	}

	// Add some state
	taskID := "test-task"
	if err := sm.MarkTaskCompleted(taskID, RunOnceRecord{}); err != nil {
		t.Fatalf("Failed to mark task: %v", err)
	}

	if err := sm.UpdateLastSync(); err != nil {
		t.Fatalf("Failed to update last sync: %v", err)
	}

	// Clear state
	if err := sm.ClearState(); err != nil {
		t.Fatalf("Failed to clear state: %v", err)
	}

	// Verify state is cleared
	if sm.IsTaskCompleted(taskID) {
		t.Error("Task should not be completed after clearing state")
	}

	lastSync := sm.GetLastSync()
	if !lastSync.IsZero() {
		t.Error("Last sync should be zero after clearing state")
	}
}

func TestCleanup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kilt-state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm, err := NewStateManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create StateManager: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tmpDir, "old-file.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Update file record with old timestamp (simulate old record)
	sm.mu.Lock()
	sm.db.Files[testFile] = FileRecord{
		SourcePath: "source.txt",
		TargetPath: testFile,
		Checksum:   "old-checksum",
		UpdatedAt:  time.Now().Add(-2 * time.Hour), // 2 hours ago
	}
	sm.mu.Unlock()

	// Create a new file record
	newFile := filepath.Join(tmpDir, "new-file.txt")
	if err := os.WriteFile(newFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create new test file: %v", err)
	}
	if err := sm.UpdateFileRecord("source2.txt", newFile); err != nil {
		t.Fatalf("Failed to update new file record: %v", err)
	}

	// Cleanup records older than 1 hour
	if err := sm.Cleanup(1 * time.Hour); err != nil {
		t.Fatalf("Failed to cleanup: %v", err)
	}

	// Old file record should be removed
	_, exists := sm.GetFileChecksum(testFile)
	if exists {
		t.Error("Old file record should be removed")
	}

	// New file record should still exist
	_, exists = sm.GetFileChecksum(newFile)
	if !exists {
		t.Error("New file record should still exist")
	}
}

func TestStatePersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kilt-state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create first state manager and add some state
	sm1, err := NewStateManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create StateManager: %v", err)
	}

	taskID := "persistent-task"
	if err := sm1.MarkTaskCompleted(taskID, RunOnceRecord{ExitCode: 0}); err != nil {
		t.Fatalf("Failed to mark task: %v", err)
	}

	// Create a new state manager (simulating restart)
	sm2, err := NewStateManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create new StateManager: %v", err)
	}

	// State should be persisted
	if !sm2.IsTaskCompleted(taskID) {
		t.Error("Task should be completed after state persistence")
	}

	record, exists := sm2.GetRunOnceRecordTyped(taskID)
	if !exists {
		t.Fatal("Record should exist after persistence")
	}
	if record.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", record.ExitCode)
	}
}
