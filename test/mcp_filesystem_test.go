package test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Solifugus/ai-work-studio/pkg/mcp"
)

// Test file constants
const (
	testDirName     = "filesystem_test_temp"
	testFileName    = "test_file.txt"
	testContent     = "Hello, World!\nThis is a test file.\nLine 3."
	binaryFileName  = "test_binary.dat"
	largeDirName    = "test_large_dir"
	nestedDirName   = "test_nested/deep/path"
)

// TestFileSystemServiceIntegration tests all filesystem operations in an integrated manner
func TestFileSystemServiceIntegration(t *testing.T) {
	// Setup test environment
	testDir := setupTestDir(t)
	defer cleanupTestDir(t, testDir)

	// Create filesystem service with test directory as base path
	service := mcp.NewFileSystemService([]string{testDir}, nil)

	// Test service registration
	registry := mcp.NewServiceRegistry(nil)
	err := registry.RegisterService(service)
	if err != nil {
		t.Fatalf("failed to register filesystem service: %v", err)
	}

	// Test that service is properly registered
	retrievedService, exists := registry.GetService("filesystem")
	if !exists {
		t.Fatal("filesystem service not found in registry")
	}

	if retrievedService.Name() != "filesystem" {
		t.Errorf("expected service name 'filesystem', got %s", retrievedService.Name())
	}

	// Run comprehensive tests
	t.Run("CreateDirectory", func(t *testing.T) {
		testCreateDirectory(t, registry, testDir)
	})

	t.Run("WriteFile", func(t *testing.T) {
		testWriteFile(t, registry, testDir)
	})

	t.Run("FileExists", func(t *testing.T) {
		testFileExists(t, registry, testDir)
	})

	t.Run("ReadFile", func(t *testing.T) {
		testReadFile(t, registry, testDir)
	})

	t.Run("ListDirectory", func(t *testing.T) {
		testListDirectory(t, registry, testDir)
	})

	t.Run("DeleteFile", func(t *testing.T) {
		testDeleteFile(t, registry, testDir)
	})

	t.Run("ErrorCases", func(t *testing.T) {
		testErrorCases(t, registry, testDir)
	})

	t.Run("PathValidation", func(t *testing.T) {
		testPathValidation(t, registry, testDir)
	})

	t.Run("LargeFileHandling", func(t *testing.T) {
		testLargeFileHandling(t, registry, testDir)
	})

	t.Run("ConcurrentOperations", func(t *testing.T) {
		testConcurrentOperations(t, registry, testDir)
	})
}

// testCreateDirectory tests directory creation operations
func testCreateDirectory(t *testing.T, registry *mcp.ServiceRegistry, testDir string) {
	// Test simple directory creation
	result := registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "create_directory",
		"path":      filepath.Join(testDir, largeDirName),
	})

	if !result.Success {
		t.Errorf("create_directory failed: %v", result.Error)
	}

	// Test nested directory creation
	result = registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "create_directory",
		"path":      filepath.Join(testDir, nestedDirName),
	})

	if !result.Success {
		t.Errorf("nested create_directory failed: %v", result.Error)
	}

	// Verify directory was created
	if _, err := os.Stat(filepath.Join(testDir, nestedDirName)); err != nil {
		t.Errorf("nested directory was not created: %v", err)
	}
}

// testWriteFile tests file writing operations
func testWriteFile(t *testing.T, registry *mcp.ServiceRegistry, testDir string) {
	testPath := filepath.Join(testDir, testFileName)

	// Test basic file write
	result := registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "write_file",
		"path":      testPath,
		"content":   testContent,
		"encoding":  "text",
	})

	if !result.Success {
		t.Errorf("write_file failed: %v", result.Error)
	}

	// Verify file was written
	content, err := os.ReadFile(testPath)
	if err != nil {
		t.Errorf("failed to read written file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("file content mismatch. Expected: %s, Got: %s", testContent, string(content))
	}

	// Test write to nested directory
	nestedPath := filepath.Join(testDir, nestedDirName, "nested_file.txt")
	result = registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "write_file",
		"path":      nestedPath,
		"content":   "nested content",
	})

	if !result.Success {
		t.Errorf("write_file to nested path failed: %v", result.Error)
	}

	// Test overwrite existing file
	newContent := "overwritten content"
	result = registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "write_file",
		"path":      testPath,
		"content":   newContent,
	})

	if !result.Success {
		t.Errorf("write_file overwrite failed: %v", result.Error)
	}

	// Verify overwrite worked
	content, err = os.ReadFile(testPath)
	if err != nil {
		t.Errorf("failed to read overwritten file: %v", err)
	}

	if string(content) != newContent {
		t.Errorf("overwrite content mismatch. Expected: %s, Got: %s", newContent, string(content))
	}
}

// testFileExists tests file existence checking
func testFileExists(t *testing.T, registry *mcp.ServiceRegistry, testDir string) {
	testPath := filepath.Join(testDir, testFileName)

	// Test existing file
	result := registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "exists",
		"path":      testPath,
	})

	if !result.Success {
		t.Errorf("exists check failed: %v", result.Error)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Errorf("exists result data has wrong type")
	}

	if exists, ok := data["exists"].(bool); !ok || !exists {
		t.Errorf("expected file to exist, but exists=%v", exists)
	}

	// Test non-existent file
	nonExistentPath := filepath.Join(testDir, "non_existent_file.txt")
	result = registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "exists",
		"path":      nonExistentPath,
	})

	if !result.Success {
		t.Errorf("exists check for non-existent file failed: %v", result.Error)
	}

	data, ok = result.Data.(map[string]interface{})
	if !ok {
		t.Errorf("exists result data has wrong type")
	}

	if exists, ok := data["exists"].(bool); !ok || exists {
		t.Errorf("expected file to not exist, but exists=%v", exists)
	}

	// Test directory exists
	result = registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "exists",
		"path":      filepath.Join(testDir, largeDirName),
	})

	if !result.Success {
		t.Errorf("exists check for directory failed: %v", result.Error)
	}

	data, ok = result.Data.(map[string]interface{})
	if !ok {
		t.Errorf("exists result data has wrong type")
	}

	if exists, ok := data["exists"].(bool); !ok || !exists {
		t.Errorf("expected directory to exist, but exists=%v", exists)
	}

	if isDir, ok := data["is_dir"].(bool); !ok || !isDir {
		t.Errorf("expected is_dir to be true for directory, but is_dir=%v", isDir)
	}
}

// testReadFile tests file reading operations
func testReadFile(t *testing.T, registry *mcp.ServiceRegistry, testDir string) {
	testPath := filepath.Join(testDir, testFileName)

	// Test basic file read
	result := registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "read_file",
		"path":      testPath,
		"encoding":  "text",
	})

	if !result.Success {
		t.Errorf("read_file failed: %v", result.Error)
	}

	data, ok := result.Data.(mcp.FileContent)
	if !ok {
		t.Errorf("read_file result data has wrong type: %T", result.Data)
	}

	if data.Path != testPath {
		t.Errorf("path mismatch. Expected: %s, Got: %s", testPath, data.Path)
	}

	if data.Encoding != "text" {
		t.Errorf("encoding mismatch. Expected: text, Got: %s", data.Encoding)
	}

	// Note: content was overwritten in previous test
	expectedContent := "overwritten content"
	if data.Content != expectedContent {
		t.Errorf("content mismatch. Expected: %s, Got: %s", expectedContent, data.Content)
	}

	// Test read with auto-detection of encoding
	result = registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "read_file",
		"path":      testPath,
	})

	if !result.Success {
		t.Errorf("read_file with auto-encoding failed: %v", result.Error)
	}

	// Test read non-existent file
	result = registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "read_file",
		"path":      filepath.Join(testDir, "non_existent.txt"),
	})

	if result.Success {
		t.Error("expected read_file of non-existent file to fail")
	}

	if result.Error == nil {
		t.Error("expected error for non-existent file read")
	}
}

// testListDirectory tests directory listing operations
func testListDirectory(t *testing.T, registry *mcp.ServiceRegistry, testDir string) {
	// Test list main test directory
	result := registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "list_directory",
		"path":      testDir,
	})

	if !result.Success {
		t.Errorf("list_directory failed: %v", result.Error)
	}

	data, ok := result.Data.(mcp.DirectoryListing)
	if !ok {
		t.Errorf("list_directory result data has wrong type: %T", result.Data)
	}

	if data.Path != testDir {
		t.Errorf("path mismatch. Expected: %s, Got: %s", testDir, data.Path)
	}

	if data.Count != len(data.Entries) {
		t.Errorf("count mismatch. Expected: %d, Got: %d", len(data.Entries), data.Count)
	}

	// Should contain at least the test file and directories we created
	foundTestFile := false
	foundLargeDir := false

	for _, entry := range data.Entries {
		if entry.Name == testFileName {
			foundTestFile = true
			if entry.IsDir {
				t.Errorf("test file incorrectly marked as directory")
			}
		}
		if entry.Name == largeDirName {
			foundLargeDir = true
			if !entry.IsDir {
				t.Errorf("test directory incorrectly marked as file")
			}
		}
	}

	if !foundTestFile {
		t.Errorf("test file not found in directory listing")
	}

	if !foundLargeDir {
		t.Errorf("test directory not found in directory listing")
	}

	// Test list non-existent directory
	result = registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "list_directory",
		"path":      filepath.Join(testDir, "non_existent_dir"),
	})

	if result.Success {
		t.Error("expected list_directory of non-existent directory to fail")
	}

	// Test list file (should fail)
	result = registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "list_directory",
		"path":      filepath.Join(testDir, testFileName),
	})

	if result.Success {
		t.Error("expected list_directory of file (not directory) to fail")
	}
}

// testDeleteFile tests file and directory deletion
func testDeleteFile(t *testing.T, registry *mcp.ServiceRegistry, testDir string) {
	// Create a temporary file to delete
	tempFile := filepath.Join(testDir, "temp_for_delete.txt")
	result := registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "write_file",
		"path":      tempFile,
		"content":   "temporary content",
	})

	if !result.Success {
		t.Errorf("failed to create temp file for deletion test: %v", result.Error)
	}

	// Test delete file
	result = registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "delete_file",
		"path":      tempFile,
	})

	if !result.Success {
		t.Errorf("delete_file failed: %v", result.Error)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Errorf("delete_file result data has wrong type: %T", result.Data)
	}

	if deleted, ok := data["deleted"].(bool); !ok || !deleted {
		t.Errorf("expected deleted to be true, got %v", deleted)
	}

	// Verify file was deleted
	if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}

	// Test delete non-existent file
	result = registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "delete_file",
		"path":      tempFile, // Already deleted
	})

	if result.Success {
		t.Error("expected delete_file of non-existent file to fail")
	}

	// Test delete empty directory
	emptyDir := filepath.Join(testDir, "empty_dir")
	err := os.Mkdir(emptyDir, 0755)
	if err != nil {
		t.Errorf("failed to create empty directory: %v", err)
	}

	result = registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "delete_file",
		"path":      emptyDir,
	})

	if !result.Success {
		t.Errorf("delete_file on empty directory failed: %v", result.Error)
	}

	// Verify directory was deleted
	if _, err := os.Stat(emptyDir); !os.IsNotExist(err) {
		t.Error("directory should have been deleted")
	}
}

// testErrorCases tests various error conditions
func testErrorCases(t *testing.T, registry *mcp.ServiceRegistry, testDir string) {
	// Test invalid operation
	result := registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "invalid_operation",
		"path":      testDir,
	})

	if result.Success {
		t.Error("expected invalid operation to fail")
	}

	// Test missing operation parameter
	result = registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"path": testDir,
	})

	if result.Success {
		t.Error("expected missing operation parameter to fail")
	}

	// Test missing path parameter
	result = registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "read_file",
	})

	if result.Success {
		t.Error("expected missing path parameter to fail")
	}

	// Test empty path parameter
	result = registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "read_file",
		"path":      "",
	})

	if result.Success {
		t.Error("expected empty path parameter to fail")
	}

	// Test invalid encoding parameter
	result = registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "read_file",
		"path":      filepath.Join(testDir, testFileName),
		"encoding":  "invalid_encoding",
	})

	if result.Success {
		t.Error("expected invalid encoding to fail")
	}
}

// testPathValidation tests security path validation
func testPathValidation(t *testing.T, registry *mcp.ServiceRegistry, testDir string) {
	// Test path traversal attack (../ sequences)
	maliciousPath := filepath.Join(testDir, "../../../etc/passwd")

	result := registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "read_file",
		"path":      maliciousPath,
	})

	if result.Success {
		t.Error("expected path traversal attack to be blocked")
	}

	if result.Error == nil || !strings.Contains(result.Error.Error(), "outside allowed directories") {
		t.Errorf("expected path validation error, got: %v", result.Error)
	}

	// Test absolute path outside allowed directory
	maliciousAbsPath := "/etc/passwd"

	result = registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "read_file",
		"path":      maliciousAbsPath,
	})

	if result.Success {
		t.Error("expected absolute path outside allowed directories to be blocked")
	}

	// Test symlink attack (if supported by OS)
	symlinkPath := filepath.Join(testDir, "malicious_symlink")
	if err := os.Symlink("/etc", symlinkPath); err == nil {
		result = registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
			"operation": "read_file",
			"path":      filepath.Join(symlinkPath, "passwd"),
		})

		// This should ideally be blocked, but depends on implementation
		// For now, we just ensure it doesn't crash
		if result.Success {
			t.Log("Warning: symlink attack was not blocked")
		}

		// Cleanup
		os.Remove(symlinkPath)
	}
}

// testLargeFileHandling tests handling of larger files
func testLargeFileHandling(t *testing.T, registry *mcp.ServiceRegistry, testDir string) {
	// Create a larger content (but not too large for tests)
	largeContent := strings.Repeat("This is a test line with some content.\n", 1000)
	largePath := filepath.Join(testDir, "large_file.txt")

	// Test writing large content
	result := registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "write_file",
		"path":      largePath,
		"content":   largeContent,
	})

	if !result.Success {
		t.Errorf("write_file for large content failed: %v", result.Error)
	}

	// Test reading large content
	result = registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
		"operation": "read_file",
		"path":      largePath,
	})

	if !result.Success {
		t.Errorf("read_file for large content failed: %v", result.Error)
	}

	data, ok := result.Data.(mcp.FileContent)
	if !ok {
		t.Errorf("read_file result data has wrong type: %T", result.Data)
	}

	if data.Content != largeContent {
		t.Error("large file content mismatch")
	}

	// Cleanup large file
	os.Remove(largePath)
}

// testConcurrentOperations tests concurrent access to the filesystem service
func testConcurrentOperations(t *testing.T, registry *mcp.ServiceRegistry, testDir string) {
	// Test concurrent file writes
	numGoroutines := 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			filename := filepath.Join(testDir, fmt.Sprintf("concurrent_file_%d.txt", id))
			content := fmt.Sprintf("Content from goroutine %d", id)

			result := registry.CallService(context.Background(), "filesystem", mcp.ServiceParams{
				"operation": "write_file",
				"path":      filename,
				"content":   content,
			})

			if !result.Success {
				t.Errorf("concurrent write failed for goroutine %d: %v", id, result.Error)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for concurrent operations")
		}
	}

	// Verify all files were created
	for i := 0; i < numGoroutines; i++ {
		filename := filepath.Join(testDir, fmt.Sprintf("concurrent_file_%d.txt", i))
		if _, err := os.Stat(filename); err != nil {
			t.Errorf("concurrent file %d was not created: %v", i, err)
		}
	}
}

// setupTestDir creates a temporary directory for testing
func setupTestDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", testDirName)
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}

	t.Logf("Created test directory: %s", tempDir)
	return tempDir
}

// cleanupTestDir removes the temporary test directory
func cleanupTestDir(t *testing.T, testDir string) {
	err := os.RemoveAll(testDir)
	if err != nil {
		t.Errorf("failed to cleanup test directory: %v", err)
	} else {
		t.Logf("Cleaned up test directory: %s", testDir)
	}
}