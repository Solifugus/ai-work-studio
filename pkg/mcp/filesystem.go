package mcp

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileSystemService provides file system operations as an MCP service.
// It includes path validation, security restrictions, and support for both
// text and binary file operations.
type FileSystemService struct {
	*BaseService
	basePaths   []string // Allowed base directories for security
	maxFileSize int64    // Maximum file size in bytes for read/write operations
	chunkSize   int      // Chunk size for large file operations
}

// FileInfo represents metadata about a file or directory.
type FileInfo struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	IsDir   bool      `json:"is_dir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
	Mode    string    `json:"mode"`
}

// FileContent represents the content of a file with metadata.
type FileContent struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Size     int64  `json:"size"`
	Encoding string `json:"encoding"` // "text" or "binary"
}

// DirectoryListing represents the contents of a directory.
type DirectoryListing struct {
	Path    string     `json:"path"`
	Entries []FileInfo `json:"entries"`
	Count   int        `json:"count"`
}

// NewFileSystemService creates a new file system MCP service.
// basePaths defines the allowed directories for operations (security restriction).
// If basePaths is empty, operations are restricted to the current working directory.
func NewFileSystemService(basePaths []string, logger *log.Logger) *FileSystemService {
	base := NewBaseService(
		"filesystem",
		"File system operations with path validation and security restrictions",
		logger,
	)

	// Default to current working directory if no base paths provided
	if len(basePaths) == 0 {
		if cwd, err := os.Getwd(); err == nil {
			basePaths = []string{cwd}
		} else {
			basePaths = []string{"."}
		}
	}

	// Clean and normalize all base paths
	cleanPaths := make([]string, len(basePaths))
	for i, path := range basePaths {
		if absPath, err := filepath.Abs(path); err == nil {
			cleanPaths[i] = filepath.Clean(absPath)
		} else {
			cleanPaths[i] = filepath.Clean(path)
		}
	}

	return &FileSystemService{
		BaseService: base,
		basePaths:   cleanPaths,
		maxFileSize: 100 * 1024 * 1024, // 100MB default
		chunkSize:   64 * 1024,          // 64KB chunks
	}
}

// ValidateParams validates parameters for file system operations.
func (fs *FileSystemService) ValidateParams(params ServiceParams) error {
	if err := fs.BaseService.ValidateParams(params); err != nil {
		return err
	}

	operation, exists := params["operation"]
	if !exists {
		return NewValidationError("operation", "operation parameter is required")
	}

	operationStr, ok := operation.(string)
	if !ok {
		return NewValidationError("operation", "operation must be a string")
	}

	// Validate operation-specific parameters
	switch operationStr {
	case "read_file":
		return fs.validateReadFileParams(params)
	case "write_file":
		return fs.validateWriteFileParams(params)
	case "list_directory":
		return fs.validateListDirectoryParams(params)
	case "exists":
		return fs.validateExistsParams(params)
	case "create_directory":
		return fs.validateCreateDirectoryParams(params)
	case "delete_file":
		return fs.validateDeleteFileParams(params)
	default:
		return NewValidationError("operation", fmt.Sprintf("unsupported operation: %s", operationStr))
	}
}

// Execute performs the requested file system operation.
func (fs *FileSystemService) Execute(ctx context.Context, params ServiceParams) ServiceResult {
	operation := params["operation"].(string)

	switch operation {
	case "read_file":
		return fs.readFile(ctx, params)
	case "write_file":
		return fs.writeFile(ctx, params)
	case "list_directory":
		return fs.listDirectory(ctx, params)
	case "exists":
		return fs.checkExists(ctx, params)
	case "create_directory":
		return fs.createDirectory(ctx, params)
	case "delete_file":
		return fs.deleteFile(ctx, params)
	default:
		return ErrorResult(fmt.Errorf("unsupported operation: %s", operation))
	}
}

// validatePath checks if the given path is within allowed base directories.
func (fs *FileSystemService) validatePath(path string) error {
	// Clean and get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	cleanPath := filepath.Clean(absPath)

	// Check if path is within any allowed base directory
	for _, basePath := range fs.basePaths {
		relPath, err := filepath.Rel(basePath, cleanPath)
		if err != nil {
			continue
		}

		// Path is valid if it doesn't contain ".." (trying to escape)
		if !strings.Contains(relPath, "..") {
			return nil
		}
	}

	return fmt.Errorf("path '%s' is outside allowed directories", path)
}

// validateReadFileParams validates parameters for read_file operation.
func (fs *FileSystemService) validateReadFileParams(params ServiceParams) error {
	if err := ValidateStringParam(params, "path", true); err != nil {
		return err
	}

	// Optional parameters
	if err := ValidateStringParam(params, "encoding", false); err != nil {
		return err
	}

	// Validate encoding if provided
	if encoding, exists := params["encoding"]; exists {
		encodingStr := encoding.(string)
		if encodingStr != "" && encodingStr != "text" && encodingStr != "binary" {
			return NewValidationError("encoding", "encoding must be 'text' or 'binary'")
		}
	}

	return nil
}

// validateWriteFileParams validates parameters for write_file operation.
func (fs *FileSystemService) validateWriteFileParams(params ServiceParams) error {
	if err := ValidateStringParam(params, "path", true); err != nil {
		return err
	}

	if err := ValidateRequiredParam(params, "content"); err != nil {
		return err
	}

	// Optional parameters
	if err := ValidateStringParam(params, "encoding", false); err != nil {
		return err
	}

	// Validate encoding if provided
	if encoding, exists := params["encoding"]; exists {
		encodingStr := encoding.(string)
		if encodingStr != "" && encodingStr != "text" && encodingStr != "binary" {
			return NewValidationError("encoding", "encoding must be 'text' or 'binary'")
		}
	}

	return nil
}

// validateListDirectoryParams validates parameters for list_directory operation.
func (fs *FileSystemService) validateListDirectoryParams(params ServiceParams) error {
	return ValidateStringParam(params, "path", true)
}

// validateExistsParams validates parameters for exists operation.
func (fs *FileSystemService) validateExistsParams(params ServiceParams) error {
	return ValidateStringParam(params, "path", true)
}

// validateCreateDirectoryParams validates parameters for create_directory operation.
func (fs *FileSystemService) validateCreateDirectoryParams(params ServiceParams) error {
	return ValidateStringParam(params, "path", true)
}

// validateDeleteFileParams validates parameters for delete_file operation.
func (fs *FileSystemService) validateDeleteFileParams(params ServiceParams) error {
	return ValidateStringParam(params, "path", true)
}

// readFile reads a file and returns its content.
func (fs *FileSystemService) readFile(ctx context.Context, params ServiceParams) ServiceResult {
	path := params["path"].(string)

	// Validate path
	if err := fs.validatePath(path); err != nil {
		return ErrorResult(fmt.Errorf("path validation failed: %w", err))
	}

	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrorResult(fmt.Errorf("file not found: %s", path))
		}
		return ErrorResult(fmt.Errorf("error accessing file: %w", err))
	}

	if info.IsDir() {
		return ErrorResult(fmt.Errorf("path is a directory, not a file: %s", path))
	}

	// Check file size limit
	if info.Size() > fs.maxFileSize {
		return ErrorResult(fmt.Errorf("file too large: %d bytes (max %d bytes)", info.Size(), fs.maxFileSize))
	}

	// Determine encoding
	encoding := "text"
	if enc, exists := params["encoding"]; exists {
		if encStr := enc.(string); encStr != "" {
			encoding = encStr
		}
	} else {
		// Auto-detect based on file extension if not specified
		if fs.isBinaryFile(path) {
			encoding = "binary"
		}
	}

	// Read file content
	file, err := os.Open(path)
	if err != nil {
		return ErrorResult(fmt.Errorf("error opening file: %w", err))
	}
	defer file.Close()

	var content strings.Builder
	if encoding == "binary" {
		// For binary files, read in chunks and encode as base64 or hex
		// For now, we'll return an error suggesting to use a different approach
		return ErrorResult(fmt.Errorf("binary file reading not yet implemented - use encoding='text' to read as text"))
	}

	// Read text file - use io.ReadAll to preserve exact content
	buffer := make([]byte, fs.chunkSize)
	for {
		n, err := file.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return ErrorResult(fmt.Errorf("error reading file: %w", err))
		}

		content.Write(buffer[:n])

		// Check context cancellation during large file reads
		select {
		case <-ctx.Done():
			return ErrorResult(fmt.Errorf("operation cancelled: %w", ctx.Err()))
		default:
			// Continue reading
		}
	}

	result := FileContent{
		Path:     path,
		Content:  content.String(),
		Size:     info.Size(),
		Encoding: encoding,
	}

	return SuccessResult(result)
}

// writeFile writes content to a file.
func (fs *FileSystemService) writeFile(ctx context.Context, params ServiceParams) ServiceResult {
	path := params["path"].(string)
	content := params["content"]

	// Validate path
	if err := fs.validatePath(path); err != nil {
		return ErrorResult(fmt.Errorf("path validation failed: %w", err))
	}

	// Determine encoding
	encoding := "text"
	if enc, exists := params["encoding"]; exists {
		if encStr := enc.(string); encStr != "" {
			encoding = encStr
		}
	}

	// Convert content to string
	var contentStr string
	switch v := content.(type) {
	case string:
		contentStr = v
	default:
		contentStr = fmt.Sprintf("%v", v)
	}

	// Check content size limit
	if int64(len(contentStr)) > fs.maxFileSize {
		return ErrorResult(fmt.Errorf("content too large: %d bytes (max %d bytes)", len(contentStr), fs.maxFileSize))
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return ErrorResult(fmt.Errorf("error creating parent directory: %w", err))
	}

	// Write file
	var file *os.File
	var err error

	if encoding == "binary" {
		file, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	} else {
		file, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	}

	if err != nil {
		return ErrorResult(fmt.Errorf("error opening file for writing: %w", err))
	}
	defer file.Close()

	// Write content in chunks for large files
	reader := strings.NewReader(contentStr)
	buffer := make([]byte, fs.chunkSize)

	totalWritten := int64(0)
	for {
		n, err := reader.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return ErrorResult(fmt.Errorf("error reading content: %w", err))
		}

		written, err := file.Write(buffer[:n])
		if err != nil {
			return ErrorResult(fmt.Errorf("error writing to file: %w", err))
		}

		totalWritten += int64(written)

		// Check context cancellation during large file writes
		select {
		case <-ctx.Done():
			return ErrorResult(fmt.Errorf("operation cancelled: %w", ctx.Err()))
		default:
			// Continue writing
		}
	}

	// Sync to ensure data is written
	if err := file.Sync(); err != nil {
		return ErrorResult(fmt.Errorf("error syncing file: %w", err))
	}

	result := map[string]interface{}{
		"path":          path,
		"bytes_written": totalWritten,
		"encoding":      encoding,
	}

	return SuccessResult(result)
}

// listDirectory lists the contents of a directory.
func (fs *FileSystemService) listDirectory(ctx context.Context, params ServiceParams) ServiceResult {
	path := params["path"].(string)

	// Validate path
	if err := fs.validatePath(path); err != nil {
		return ErrorResult(fmt.Errorf("path validation failed: %w", err))
	}

	// Check if directory exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrorResult(fmt.Errorf("directory not found: %s", path))
		}
		return ErrorResult(fmt.Errorf("error accessing directory: %w", err))
	}

	if !info.IsDir() {
		return ErrorResult(fmt.Errorf("path is not a directory: %s", path))
	}

	// Read directory contents
	entries, err := os.ReadDir(path)
	if err != nil {
		return ErrorResult(fmt.Errorf("error reading directory: %w", err))
	}

	// Build file info list
	var fileInfos []FileInfo
	for _, entry := range entries {
		// Check context cancellation during directory listing
		select {
		case <-ctx.Done():
			return ErrorResult(fmt.Errorf("operation cancelled: %w", ctx.Err()))
		default:
			// Continue processing
		}

		info, err := entry.Info()
		if err != nil {
			continue // Skip entries we can't access
		}

		fileInfo := FileInfo{
			Name:    entry.Name(),
			Path:    filepath.Join(path, entry.Name()),
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Mode:    info.Mode().String(),
		}

		fileInfos = append(fileInfos, fileInfo)
	}

	result := DirectoryListing{
		Path:    path,
		Entries: fileInfos,
		Count:   len(fileInfos),
	}

	return SuccessResult(result)
}

// checkExists checks if a file or directory exists.
func (fs *FileSystemService) checkExists(ctx context.Context, params ServiceParams) ServiceResult {
	path := params["path"].(string)

	// Validate path
	if err := fs.validatePath(path); err != nil {
		return ErrorResult(fmt.Errorf("path validation failed: %w", err))
	}

	info, err := os.Stat(path)
	exists := !os.IsNotExist(err)

	result := map[string]interface{}{
		"path":   path,
		"exists": exists,
	}

	if exists && err == nil {
		result["is_dir"] = info.IsDir()
		result["size"] = info.Size()
		result["mod_time"] = info.ModTime()
	}

	return SuccessResult(result)
}

// createDirectory creates a directory and any necessary parent directories.
func (fs *FileSystemService) createDirectory(ctx context.Context, params ServiceParams) ServiceResult {
	path := params["path"].(string)

	// Validate path
	if err := fs.validatePath(path); err != nil {
		return ErrorResult(fmt.Errorf("path validation failed: %w", err))
	}

	// Create directory with parent directories
	if err := os.MkdirAll(path, 0755); err != nil {
		return ErrorResult(fmt.Errorf("error creating directory: %w", err))
	}

	// Verify creation
	info, err := os.Stat(path)
	if err != nil {
		return ErrorResult(fmt.Errorf("error verifying directory creation: %w", err))
	}

	result := map[string]interface{}{
		"path":     path,
		"created":  true,
		"is_dir":   info.IsDir(),
		"mod_time": info.ModTime(),
	}

	return SuccessResult(result)
}

// deleteFile deletes a file or empty directory.
func (fs *FileSystemService) deleteFile(ctx context.Context, params ServiceParams) ServiceResult {
	path := params["path"].(string)

	// Validate path
	if err := fs.validatePath(path); err != nil {
		return ErrorResult(fmt.Errorf("path validation failed: %w", err))
	}

	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrorResult(fmt.Errorf("file not found: %s", path))
		}
		return ErrorResult(fmt.Errorf("error accessing file: %w", err))
	}

	isDir := info.IsDir()

	// Delete the file or directory
	if err := os.Remove(path); err != nil {
		return ErrorResult(fmt.Errorf("error deleting file: %w", err))
	}

	result := map[string]interface{}{
		"path":    path,
		"deleted": true,
		"was_dir": isDir,
	}

	return SuccessResult(result)
}

// isBinaryFile determines if a file is likely to be binary based on its extension.
func (fs *FileSystemService) isBinaryFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))

	binaryExts := []string{
		".exe", ".dll", ".so", ".dylib",
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff",
		".mp3", ".mp4", ".wav", ".avi", ".mkv",
		".zip", ".tar", ".gz", ".rar", ".7z",
		".pdf", ".doc", ".docx", ".xls", ".xlsx",
		".bin", ".dat", ".img", ".iso",
	}

	for _, binaryExt := range binaryExts {
		if ext == binaryExt {
			return true
		}
	}

	return false
}