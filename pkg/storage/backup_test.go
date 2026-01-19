package storage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewBackupManager(t *testing.T) {
	dataDir := t.TempDir()

	// Test default maxBackups
	bm := NewBackupManager(dataDir, 0)
	if bm.maxBackups != 10 {
		t.Errorf("expected default maxBackups=10, got %d", bm.maxBackups)
	}

	// Test custom maxBackups
	bm2 := NewBackupManager(dataDir, 5)
	if bm2.maxBackups != 5 {
		t.Errorf("expected maxBackups=5, got %d", bm2.maxBackups)
	}

	// Test that backup directory path is correct
	expectedBackupDir := filepath.Join(dataDir, "backups")
	if bm.backupDir != expectedBackupDir {
		t.Errorf("expected backupDir=%s, got %s", expectedBackupDir, bm.backupDir)
	}
}

func TestCreateBackup(t *testing.T) {
	dataDir := setupTestData(t)
	bm := NewBackupManager(dataDir, 10)

	ctx := context.Background()
	backupPath, err := bm.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Verify backup directory exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("backup directory was not created: %s", backupPath)
	}

	// Verify backup contains nodes directory
	nodesBackupPath := filepath.Join(backupPath, "nodes")
	if _, err := os.Stat(nodesBackupPath); os.IsNotExist(err) {
		t.Errorf("nodes directory was not backed up")
	}

	// Verify backup contains edges directory
	edgesBackupPath := filepath.Join(backupPath, "edges")
	if _, err := os.Stat(edgesBackupPath); os.IsNotExist(err) {
		t.Errorf("edges directory was not backed up")
	}

	// Verify backup metadata file exists
	metadataPath := filepath.Join(backupPath, "backup_metadata.json")
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		t.Errorf("backup metadata was not created")
	}

	// Verify that node files were copied
	originalNodeFiles, _ := filepath.Glob(filepath.Join(dataDir, "nodes", "*", "*.json"))
	for _, originalFile := range originalNodeFiles {
		relPath, _ := filepath.Rel(dataDir, originalFile)
		backupFile := filepath.Join(backupPath, relPath)
		if _, err := os.Stat(backupFile); os.IsNotExist(err) {
			t.Errorf("node file was not backed up: %s", backupFile)
		}
	}

	// Verify that edge files were copied
	originalEdgeFiles, _ := filepath.Glob(filepath.Join(dataDir, "edges", "*.json"))
	for _, originalFile := range originalEdgeFiles {
		relPath, _ := filepath.Rel(dataDir, originalFile)
		backupFile := filepath.Join(backupPath, relPath)
		if _, err := os.Stat(backupFile); os.IsNotExist(err) {
			t.Errorf("edge file was not backed up: %s", backupFile)
		}
	}
}

func TestCreateBackupWithEmptyData(t *testing.T) {
	dataDir := t.TempDir()
	bm := NewBackupManager(dataDir, 10)

	ctx := context.Background()
	backupPath, err := bm.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("CreateBackup failed with empty data: %v", err)
	}

	// Backup should still succeed even with no data
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("backup directory was not created for empty data")
	}

	// Metadata should still be created
	metadataPath := filepath.Join(backupPath, "backup_metadata.json")
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		t.Errorf("backup metadata was not created for empty backup")
	}
}

func TestListBackups(t *testing.T) {
	dataDir := setupTestData(t)
	bm := NewBackupManager(dataDir, 10)

	ctx := context.Background()

	// Initially, no backups should exist
	backups, err := bm.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	if len(backups) != 0 {
		t.Errorf("expected 0 backups initially, got %d", len(backups))
	}

	// Create some backups with different timestamps
	backup1, err := bm.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("failed to create first backup: %v", err)
	}

	time.Sleep(100 * time.Millisecond) // Ensure different timestamp

	backup2, err := bm.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("failed to create second backup: %v", err)
	}

	// List backups
	backups, err = bm.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}

	if len(backups) != 2 {
		t.Errorf("expected 2 backups, got %d", len(backups))
	}

	// Verify that backups are sorted by timestamp (newest first)
	if !backups[0].Timestamp.After(backups[1].Timestamp) {
		t.Errorf("backups are not sorted correctly: %v vs %v", backups[0].Timestamp, backups[1].Timestamp)
	}

	// Verify backup paths are correct
	if backups[0].Path != backup2 {
		t.Errorf("expected newest backup path %s, got %s", backup2, backups[0].Path)
	}
	if backups[1].Path != backup1 {
		t.Errorf("expected oldest backup path %s, got %s", backup1, backups[1].Path)
	}

	// Verify backup info contains useful data
	for i, backup := range backups {
		if backup.FileCount <= 0 {
			t.Errorf("backup %d has invalid file count: %d", i, backup.FileCount)
		}
		if backup.Size <= 0 {
			t.Errorf("backup %d has invalid size: %d", i, backup.Size)
		}
	}
}

func TestCleanupOldBackups(t *testing.T) {
	dataDir := setupTestData(t)
	bm := NewBackupManager(dataDir, 2) // Keep only 2 backups

	ctx := context.Background()

	// Create 4 backups
	var backupPaths []string
	for i := 0; i < 4; i++ {
		backup, err := bm.CreateBackup(ctx)
		if err != nil {
			t.Fatalf("failed to create backup %d: %v", i, err)
		}
		backupPaths = append(backupPaths, backup)
		time.Sleep(100 * time.Millisecond) // Ensure different timestamps
	}

	// List backups - should only have 2 newest ones
	backups, err := bm.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}

	if len(backups) != 2 {
		t.Errorf("expected 2 backups after cleanup, got %d", len(backups))
	}

	// Verify that oldest backups were removed
	if _, err := os.Stat(backupPaths[0]); !os.IsNotExist(err) {
		t.Errorf("oldest backup was not removed: %s", backupPaths[0])
	}
	if _, err := os.Stat(backupPaths[1]); !os.IsNotExist(err) {
		t.Errorf("second oldest backup was not removed: %s", backupPaths[1])
	}

	// Verify that newest backups still exist
	if _, err := os.Stat(backupPaths[2]); os.IsNotExist(err) {
		t.Errorf("second newest backup was incorrectly removed: %s", backupPaths[2])
	}
	if _, err := os.Stat(backupPaths[3]); os.IsNotExist(err) {
		t.Errorf("newest backup was incorrectly removed: %s", backupPaths[3])
	}
}

func TestRestoreFromBackup(t *testing.T) {
	dataDir := setupTestData(t)
	bm := NewBackupManager(dataDir, 10)

	ctx := context.Background()

	// Create a backup
	backupPath, err := bm.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	// Modify original data (add a new node)
	newNode := NewNode("goal", map[string]interface{}{"title": "New Goal After Backup"})
	store, err := NewStore(dataDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	err = store.AddNode(ctx, newNode)
	if err != nil {
		t.Fatalf("failed to add new node: %v", err)
	}
	store.Close()

	// Verify new node exists before restore
	newNodePath := filepath.Join(dataDir, "nodes", newNode.Type, newNode.ID+".json")
	if _, err := os.Stat(newNodePath); os.IsNotExist(err) {
		t.Fatalf("new node file was not created")
	}

	// Restore from backup
	err = bm.RestoreFromBackup(ctx, backupPath)
	if err != nil {
		t.Fatalf("RestoreFromBackup failed: %v", err)
	}

	// Verify new node no longer exists (rolled back to backup state)
	if _, err := os.Stat(newNodePath); !os.IsNotExist(err) {
		t.Errorf("new node still exists after restore, should have been removed")
	}

	// Verify original data is restored
	originalFiles, _ := filepath.Glob(filepath.Join(backupPath, "nodes", "*", "*.json"))
	for _, backupFile := range originalFiles {
		relPath, _ := filepath.Rel(backupPath, backupFile)
		restoredFile := filepath.Join(dataDir, relPath)
		if _, err := os.Stat(restoredFile); os.IsNotExist(err) {
			t.Errorf("backup file was not restored: %s", restoredFile)
		}
	}
}

func TestRestoreFromLatestBackup(t *testing.T) {
	dataDir := setupTestData(t)
	bm := NewBackupManager(dataDir, 10)

	ctx := context.Background()

	// Create multiple backups
	_, err := bm.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("failed to create first backup: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	latestBackupPath, err := bm.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("failed to create latest backup: %v", err)
	}

	// Modify data
	store, err := NewStore(dataDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	newNode := NewNode("goal", map[string]interface{}{"title": "Modification"})
	err = store.AddNode(ctx, newNode)
	if err != nil {
		t.Fatalf("failed to modify data: %v", err)
	}
	store.Close()

	// Restore from latest backup
	err = bm.RestoreFromLatestBackup(ctx)
	if err != nil {
		t.Fatalf("RestoreFromLatestBackup failed: %v", err)
	}

	// Verify data matches latest backup
	latestBackupFiles, _ := filepath.Glob(filepath.Join(latestBackupPath, "nodes", "*", "*.json"))
	currentFiles, _ := filepath.Glob(filepath.Join(dataDir, "nodes", "*", "*.json"))

	if len(currentFiles) != len(latestBackupFiles) {
		t.Errorf("restored data has different file count than latest backup")
	}
}

func TestRestoreFromLatestBackupNoBackups(t *testing.T) {
	dataDir := t.TempDir()
	bm := NewBackupManager(dataDir, 10)

	ctx := context.Background()

	// Try to restore when no backups exist
	err := bm.RestoreFromLatestBackup(ctx)
	if err == nil {
		t.Errorf("expected error when restoring with no backups, got nil")
	}
	if !strings.Contains(err.Error(), "no backups available") {
		t.Errorf("expected 'no backups available' error, got: %v", err)
	}
}

func TestValidateBackupStructure(t *testing.T) {
	tempDir := t.TempDir()
	bm := NewBackupManager(tempDir, 10)

	// Test valid backup structure
	validBackupDir := filepath.Join(tempDir, "valid_backup")
	os.MkdirAll(filepath.Join(validBackupDir, "nodes"), 0755)
	os.MkdirAll(filepath.Join(validBackupDir, "edges"), 0755)

	err := bm.validateBackupStructure(validBackupDir)
	if err != nil {
		t.Errorf("expected valid backup structure, got error: %v", err)
	}

	// Test backup directory doesn't exist
	nonexistentDir := filepath.Join(tempDir, "nonexistent")
	err = bm.validateBackupStructure(nonexistentDir)
	if err == nil {
		t.Errorf("expected error for nonexistent backup directory")
	}

	// Test missing nodes directory (should still be valid - empty dataset)
	partialBackupDir := filepath.Join(tempDir, "partial_backup")
	os.MkdirAll(filepath.Join(partialBackupDir, "edges"), 0755)

	err = bm.validateBackupStructure(partialBackupDir)
	if err != nil {
		t.Errorf("expected backup without nodes directory to be valid, got: %v", err)
	}

	// Test nodes is a file instead of directory
	invalidBackupDir := filepath.Join(tempDir, "invalid_backup")
	os.MkdirAll(invalidBackupDir, 0755)
	os.WriteFile(filepath.Join(invalidBackupDir, "nodes"), []byte("not a directory"), 0644)

	err = bm.validateBackupStructure(invalidBackupDir)
	if err == nil {
		t.Errorf("expected error when nodes is a file instead of directory")
	}
}

func TestRecoverFromCorruption(t *testing.T) {
	dataDir := setupTestData(t)
	bm := NewBackupManager(dataDir, 10)

	ctx := context.Background()

	// Create a good backup
	goodBackupPath, err := bm.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("failed to create good backup: %v", err)
	}

	// Corrupt some data files
	nodeFiles, _ := filepath.Glob(filepath.Join(dataDir, "nodes", "*", "*.json"))
	if len(nodeFiles) > 0 {
		// Write invalid JSON to first node file
		os.WriteFile(nodeFiles[0], []byte("{corrupted json"), 0644)
	}

	edgeFiles, _ := filepath.Glob(filepath.Join(dataDir, "edges", "*.json"))
	if len(edgeFiles) > 0 {
		// Write invalid JSON to first edge file
		os.WriteFile(edgeFiles[0], []byte("not json at all"), 0644)
	}

	// Attempt recovery
	err = bm.RecoverFromCorruption(ctx, dataDir)
	if err != nil {
		t.Fatalf("RecoverFromCorruption failed: %v", err)
	}

	// Verify corruption was fixed by validating all files
	results, err := ValidateDataDirectory(dataDir)
	if err != nil {
		t.Fatalf("validation failed after recovery: %v", err)
	}

	invalidCount := 0
	for _, result := range results {
		if !result.Valid {
			invalidCount++
		}
	}

	if invalidCount > 0 {
		t.Errorf("found %d invalid files after corruption recovery", invalidCount)
	}

	// Verify that recovered files match the good backup
	for _, nodeFile := range nodeFiles {
		relPath, _ := filepath.Rel(dataDir, nodeFile)
		backupFile := filepath.Join(goodBackupPath, relPath)

		if _, err := os.Stat(backupFile); err == nil {
			// Compare file contents
			originalData, _ := os.ReadFile(backupFile)
			restoredData, _ := os.ReadFile(nodeFile)

			if string(originalData) != string(restoredData) {
				t.Errorf("restored file %s doesn't match backup", nodeFile)
			}
		}
	}
}

func TestRecoverFromCorruptionNoBackups(t *testing.T) {
	dataDir := t.TempDir()

	// Create corrupted data without any backups
	nodesDir := filepath.Join(dataDir, "nodes", "goal")
	os.MkdirAll(nodesDir, 0755)
	os.WriteFile(filepath.Join(nodesDir, "corrupt.json"), []byte("{invalid"), 0644)

	bm := NewBackupManager(dataDir, 10)
	ctx := context.Background()

	// Attempt recovery - should fail because no backups exist
	err := bm.RecoverFromCorruption(ctx, dataDir)
	if err == nil {
		t.Errorf("expected error when recovering with no backups")
	}
	if !strings.Contains(err.Error(), "no backups available") {
		t.Errorf("expected 'no backups available' error, got: %v", err)
	}
}

func TestRecoverFromCorruptionNoCorruption(t *testing.T) {
	dataDir := setupTestData(t)
	bm := NewBackupManager(dataDir, 10)

	ctx := context.Background()

	// Attempt recovery on clean data - should succeed without doing anything
	err := bm.RecoverFromCorruption(ctx, dataDir)
	if err != nil {
		t.Errorf("RecoverFromCorruption should succeed on clean data, got: %v", err)
	}
}

func TestGetBackupInfo(t *testing.T) {
	dataDir := setupTestData(t)
	bm := NewBackupManager(dataDir, 10)

	ctx := context.Background()
	backupPath, err := bm.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	info, err := bm.getBackupInfo(backupPath)
	if err != nil {
		t.Fatalf("getBackupInfo failed: %v", err)
	}

	if info.Path != backupPath {
		t.Errorf("expected path %s, got %s", backupPath, info.Path)
	}

	if info.FileCount <= 0 {
		t.Errorf("expected positive file count, got %d", info.FileCount)
	}

	if info.Size <= 0 {
		t.Errorf("expected positive size, got %d", info.Size)
	}
}

func TestCopyFile(t *testing.T) {
	tempDir := t.TempDir()
	bm := NewBackupManager(tempDir, 10)

	// Create source file
	srcContent := "test file content"
	srcPath := filepath.Join(tempDir, "source.txt")
	err := os.WriteFile(srcPath, []byte(srcContent), 0644)
	if err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Copy file
	dstPath := filepath.Join(tempDir, "destination.txt")
	err = bm.copyFile(srcPath, dstPath, 0644)
	if err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	// Verify destination file was created with correct content
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("failed to read destination file: %v", err)
	}

	if string(dstContent) != srcContent {
		t.Errorf("expected content '%s', got '%s'", srcContent, string(dstContent))
	}
}

// setupTestData creates a test data directory with some sample nodes and edges
func setupTestData(t *testing.T) string {
	tempDir := t.TempDir()

	// Create directory structure
	nodesDir := filepath.Join(tempDir, "nodes", "goal")
	edgesDir := filepath.Join(tempDir, "edges")
	os.MkdirAll(nodesDir, 0755)
	os.MkdirAll(edgesDir, 0755)

	// Create test nodes
	node1 := NewNode("goal", map[string]interface{}{"title": "Test Goal 1", "description": "First test goal"})
	node2 := NewNode("goal", map[string]interface{}{"title": "Test Goal 2", "description": "Second test goal"})

	// Create test edge
	edge := NewEdge(node1.ID, node2.ID, "depends_on", map[string]interface{}{"strength": 0.8})

	// Write node files
	node1History := NodeHistory{node1}
	node1Data, _ := json.MarshalIndent(node1History, "", "  ")
	os.WriteFile(filepath.Join(nodesDir, node1.ID+".json"), node1Data, 0644)

	node2History := NodeHistory{node2}
	node2Data, _ := json.MarshalIndent(node2History, "", "  ")
	os.WriteFile(filepath.Join(nodesDir, node2.ID+".json"), node2Data, 0644)

	// Write edge file
	edgeHistory := EdgeHistory{edge}
	edgeData, _ := json.MarshalIndent(edgeHistory, "", "  ")
	os.WriteFile(filepath.Join(edgesDir, edge.ID+".json"), edgeData, 0644)

	return tempDir
}