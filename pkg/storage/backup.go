package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// BackupManager handles automated backup and recovery operations for the storage system.
type BackupManager struct {
	dataDir    string // Source data directory
	backupDir  string // Backup destination directory
	maxBackups int    // Maximum number of backups to keep
}

// BackupInfo contains metadata about a backup.
type BackupInfo struct {
	Timestamp time.Time // When the backup was created
	Path      string    // Full path to the backup directory
	Size      int64     // Total size in bytes
	FileCount int       // Number of files in the backup
}

// NewBackupManager creates a new backup manager instance.
func NewBackupManager(dataDir string, maxBackups int) *BackupManager {
	if maxBackups <= 0 {
		maxBackups = 10 // Default to keeping 10 backups
	}

	return &BackupManager{
		dataDir:    dataDir,
		backupDir:  filepath.Join(dataDir, "backups"),
		maxBackups: maxBackups,
	}
}

// CreateBackup creates a timestamped backup of the entire data directory.
// Returns the backup path and any error encountered.
func (bm *BackupManager) CreateBackup(ctx context.Context) (string, error) {
	// Ensure backup directory exists
	if err := os.MkdirAll(bm.backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate timestamped backup directory name
	timestamp := time.Now()
	backupName := timestamp.Format("20060102_150405.000000") // YYYYMMDD_HHMMSS.microseconds
	backupPath := filepath.Join(bm.backupDir, backupName)

	// Create the backup directory
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup path %s: %w", backupPath, err)
	}

	// Copy data directories to backup
	if err := bm.copyDirectory(ctx, filepath.Join(bm.dataDir, "nodes"), filepath.Join(backupPath, "nodes")); err != nil {
		// Clean up partial backup on failure
		os.RemoveAll(backupPath)
		return "", fmt.Errorf("failed to backup nodes: %w", err)
	}

	if err := bm.copyDirectory(ctx, filepath.Join(bm.dataDir, "edges"), filepath.Join(backupPath, "edges")); err != nil {
		// Clean up partial backup on failure
		os.RemoveAll(backupPath)
		return "", fmt.Errorf("failed to backup edges: %w", err)
	}

	// Write backup metadata
	if err := bm.writeBackupMetadata(backupPath, timestamp); err != nil {
		// Clean up partial backup on failure
		os.RemoveAll(backupPath)
		return "", fmt.Errorf("failed to write backup metadata: %w", err)
	}

	// Clean up old backups if we exceed the maximum
	if err := bm.cleanupOldBackups(); err != nil {
		// Don't fail the backup for cleanup errors, just log a warning
		// In a real implementation, this would use proper logging
		fmt.Fprintf(os.Stderr, "Warning: failed to cleanup old backups: %v\n", err)
	}

	return backupPath, nil
}

// copyDirectory recursively copies a directory from src to dst.
func (bm *BackupManager) copyDirectory(ctx context.Context, src, dst string) error {
	// Check if source exists
	srcInfo, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			// Source directory doesn't exist, skip it
			return nil
		}
		return fmt.Errorf("failed to stat source directory %s: %w", src, err)
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", dst, err)
	}

	// Walk through source directory
	return filepath.Walk(src, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Calculate relative path from source
		relPath, err := filepath.Rel(src, srcPath)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			// Create directory
			return os.MkdirAll(dstPath, info.Mode())
		} else {
			// Copy file
			return bm.copyFile(srcPath, dstPath, info.Mode())
		}
	})
}

// copyFile copies a single file from src to dst with the specified permissions.
func (bm *BackupManager) copyFile(src, dst string, mode os.FileMode) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer dstFile.Close()

	// Copy file content
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content from %s to %s: %w", src, dst, err)
	}

	return nil
}

// writeBackupMetadata writes metadata about the backup to a file in the backup directory.
func (bm *BackupManager) writeBackupMetadata(backupPath string, timestamp time.Time) error {
	info, err := bm.getBackupInfo(backupPath)
	if err != nil {
		return fmt.Errorf("failed to gather backup info: %w", err)
	}

	metadata := fmt.Sprintf(`{
  "timestamp": "%s",
  "path": "%s",
  "size_bytes": %d,
  "file_count": %d,
  "created_by": "AI Work Studio Backup Manager"
}`, timestamp.Format(time.RFC3339), backupPath, info.Size, info.FileCount)

	metadataPath := filepath.Join(backupPath, "backup_metadata.json")
	return os.WriteFile(metadataPath, []byte(metadata), 0644)
}

// getBackupInfo calculates size and file count statistics for a backup directory.
func (bm *BackupManager) getBackupInfo(backupPath string) (BackupInfo, error) {
	info := BackupInfo{Path: backupPath}

	err := filepath.Walk(backupPath, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !fileInfo.IsDir() {
			info.Size += fileInfo.Size()
			info.FileCount++
		}

		return nil
	})

	return info, err
}

// ListBackups returns a list of all available backups, sorted by creation time (newest first).
func (bm *BackupManager) ListBackups() ([]BackupInfo, error) {
	var backups []BackupInfo

	// Check if backup directory exists
	if _, err := os.Stat(bm.backupDir); os.IsNotExist(err) {
		return backups, nil // No backups exist yet
	}

	// Read backup directory
	entries, err := os.ReadDir(bm.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Parse timestamp from directory name (YYYYMMDD_HHMMSS.microseconds format)
		timestamp, err := time.Parse("20060102_150405.000000", entry.Name())
		if err != nil {
			// Skip directories that don't match expected format
			continue
		}

		backupPath := filepath.Join(bm.backupDir, entry.Name())
		info, err := bm.getBackupInfo(backupPath)
		if err != nil {
			// Skip backups that can't be analyzed
			continue
		}

		info.Timestamp = timestamp
		backups = append(backups, info)
	}

	// Sort by timestamp (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// cleanupOldBackups removes backups that exceed the maximum count.
func (bm *BackupManager) cleanupOldBackups() error {
	backups, err := bm.ListBackups()
	if err != nil {
		return fmt.Errorf("failed to list existing backups: %w", err)
	}

	// Remove excess backups (keep only maxBackups newest)
	if len(backups) > bm.maxBackups {
		for i := bm.maxBackups; i < len(backups); i++ {
			if err := os.RemoveAll(backups[i].Path); err != nil {
				return fmt.Errorf("failed to remove old backup %s: %w", backups[i].Path, err)
			}
		}
	}

	return nil
}

// RestoreFromBackup restores the data directory from the specified backup.
// This completely replaces the current data with the backup data.
func (bm *BackupManager) RestoreFromBackup(ctx context.Context, backupPath string) error {
	// Validate that the backup exists and has expected structure
	if err := bm.validateBackupStructure(backupPath); err != nil {
		return fmt.Errorf("backup validation failed: %w", err)
	}

	// Create a temporary directory for the restoration process
	tempDir := bm.dataDir + "_restore_temp"
	if err := os.RemoveAll(tempDir); err != nil {
		return fmt.Errorf("failed to clean temporary directory: %w", err)
	}

	// Copy backup data to temporary directory
	if err := bm.copyDirectory(ctx, filepath.Join(backupPath, "nodes"), filepath.Join(tempDir, "nodes")); err != nil {
		os.RemoveAll(tempDir)
		return fmt.Errorf("failed to restore nodes: %w", err)
	}

	if err := bm.copyDirectory(ctx, filepath.Join(backupPath, "edges"), filepath.Join(tempDir, "edges")); err != nil {
		os.RemoveAll(tempDir)
		return fmt.Errorf("failed to restore edges: %w", err)
	}

	// Backup current data (in case restoration fails)
	currentBackupPath := bm.dataDir + "_pre_restore_backup"
	if err := os.RemoveAll(currentBackupPath); err != nil {
		return fmt.Errorf("failed to clean pre-restore backup location: %w", err)
	}

	// Create the current backup directory
	if err := os.MkdirAll(currentBackupPath, 0755); err != nil {
		return fmt.Errorf("failed to create pre-restore backup directory: %w", err)
	}

	// Move current data to backup location
	if _, err := os.Stat(filepath.Join(bm.dataDir, "nodes")); err == nil {
		if err := os.Rename(filepath.Join(bm.dataDir, "nodes"), filepath.Join(currentBackupPath, "nodes")); err != nil {
			os.RemoveAll(tempDir)
			return fmt.Errorf("failed to backup current nodes: %w", err)
		}
	}

	if _, err := os.Stat(filepath.Join(bm.dataDir, "edges")); err == nil {
		if err := os.Rename(filepath.Join(bm.dataDir, "edges"), filepath.Join(currentBackupPath, "edges")); err != nil {
			// Try to restore nodes if edges backup failed
			if _, err := os.Stat(filepath.Join(currentBackupPath, "nodes")); err == nil {
				os.Rename(filepath.Join(currentBackupPath, "nodes"), filepath.Join(bm.dataDir, "nodes"))
			}
			os.RemoveAll(tempDir)
			return fmt.Errorf("failed to backup current edges: %w", err)
		}
	}

	// Move restored data into place
	if err := os.Rename(filepath.Join(tempDir, "nodes"), filepath.Join(bm.dataDir, "nodes")); err != nil {
		// Try to restore from current backup
		bm.restoreFromCurrentBackup(currentBackupPath)
		os.RemoveAll(tempDir)
		return fmt.Errorf("failed to move restored nodes: %w", err)
	}

	if err := os.Rename(filepath.Join(tempDir, "edges"), filepath.Join(bm.dataDir, "edges")); err != nil {
		// Try to restore from current backup
		bm.restoreFromCurrentBackup(currentBackupPath)
		os.RemoveAll(tempDir)
		return fmt.Errorf("failed to move restored edges: %w", err)
	}

	// Clean up temporary directories
	os.RemoveAll(tempDir)
	os.RemoveAll(currentBackupPath)

	return nil
}

// restoreFromCurrentBackup attempts to restore from the pre-restore backup.
func (bm *BackupManager) restoreFromCurrentBackup(currentBackupPath string) {
	if _, err := os.Stat(filepath.Join(currentBackupPath, "nodes")); err == nil {
		os.Rename(filepath.Join(currentBackupPath, "nodes"), filepath.Join(bm.dataDir, "nodes"))
	}
	if _, err := os.Stat(filepath.Join(currentBackupPath, "edges")); err == nil {
		os.Rename(filepath.Join(currentBackupPath, "edges"), filepath.Join(bm.dataDir, "edges"))
	}
}

// RestoreFromLatestBackup restores from the most recent backup.
func (bm *BackupManager) RestoreFromLatestBackup(ctx context.Context) error {
	backups, err := bm.ListBackups()
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(backups) == 0 {
		return fmt.Errorf("no backups available for restoration")
	}

	return bm.RestoreFromBackup(ctx, backups[0].Path)
}

// validateBackupStructure checks that a backup directory has the expected structure.
func (bm *BackupManager) validateBackupStructure(backupPath string) error {
	// Check that backup directory exists
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup directory does not exist: %w", err)
	}

	// Check for expected subdirectories
	expectedDirs := []string{"nodes", "edges"}
	for _, dir := range expectedDirs {
		dirPath := filepath.Join(backupPath, dir)
		if info, err := os.Stat(dirPath); err != nil {
			if os.IsNotExist(err) {
				// It's ok if the directory doesn't exist (empty dataset)
				continue
			}
			return fmt.Errorf("cannot access %s directory: %w", dir, err)
		} else if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", dirPath)
		}
	}

	return nil
}

// RecoverFromCorruption attempts to recover from data corruption using validation and backups.
// It validates all data files and restores from backup if corruption is detected.
func (bm *BackupManager) RecoverFromCorruption(ctx context.Context, dataDir string) error {
	// First, validate current data
	results, err := ValidateDataDirectory(dataDir)
	if err != nil {
		return fmt.Errorf("failed to validate data directory: %w", err)
	}

	// Count corrupted files
	var corruptedFiles []string
	for _, result := range results {
		if !result.Valid {
			corruptedFiles = append(corruptedFiles, result.FilePath)
		}
	}

	// If no corruption detected, nothing to do
	if len(corruptedFiles) == 0 {
		return nil
	}

	// Log corruption detected
	fmt.Fprintf(os.Stderr, "Corruption detected in %d files: %v\n", len(corruptedFiles), corruptedFiles)

	// Attempt restoration from latest backup
	backups, err := bm.ListBackups()
	if err != nil {
		return fmt.Errorf("failed to list backups for recovery: %w", err)
	}

	if len(backups) == 0 {
		return fmt.Errorf("no backups available for corruption recovery")
	}

	// Try restoring from backups until we find a good one
	for _, backup := range backups {
		fmt.Fprintf(os.Stderr, "Attempting recovery from backup: %s\n", backup.Path)

		if err := bm.RestoreFromBackup(ctx, backup.Path); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to restore from backup %s: %v\n", backup.Path, err)
			continue
		}

		// Validate restored data
		validationResults, err := ValidateDataDirectory(dataDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to validate restored data: %v\n", err)
			continue
		}

		// Check if restoration fixed the corruption
		var stillCorrupted []string
		for _, result := range validationResults {
			if !result.Valid {
				stillCorrupted = append(stillCorrupted, result.FilePath)
			}
		}

		if len(stillCorrupted) == 0 {
			fmt.Fprintf(os.Stderr, "Recovery successful from backup: %s\n", backup.Path)
			return nil
		}

		fmt.Fprintf(os.Stderr, "Backup %s still has %d corrupted files\n", backup.Path, len(stillCorrupted))
	}

	return fmt.Errorf("failed to recover from corruption - all backups are also corrupted")
}