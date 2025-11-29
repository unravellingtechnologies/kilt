// Package core provides the core engine and configuration management
// for the Kilt dotfiles manager. It includes configuration parsing,
// state management, backup operations, template rendering, and the
// main orchestration engine.
package core

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetKiltDir returns the path to the .kilt directory in the user's home directory
func GetKiltDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".kilt"), nil
}

// GetBackupDir returns the path to the backup directory (~/.kilt/backup)
func GetBackupDir() (string, error) {
	kiltDir, err := GetKiltDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(kiltDir, "backup"), nil
}

// GetStateDir returns the path to the state directory (~/.kilt/state)
func GetStateDir() (string, error) {
	kiltDir, err := GetKiltDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(kiltDir, "state"), nil
}

// PathsMatch checks if two paths refer to the same file by comparing their absolute paths.
// Returns true if both paths resolve to the same absolute path, false otherwise.
// If either path cannot be converted to absolute, falls back to string comparison.
func PathsMatch(path1, path2 string) bool {
	abs1, err1 := filepath.Abs(path1)
	abs2, err2 := filepath.Abs(path2)

	if err1 != nil || err2 != nil {
		return path1 == path2
	}

	return abs1 == abs2
}
