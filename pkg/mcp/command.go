package mcp

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CommandService provides command execution capabilities as an MCP service.
// It includes security restrictions, timeout controls, and output capture.
type CommandService struct {
	*BaseService
	allowedCommands   []string      // Allowlist of permitted commands for security
	allowedDirs       []string      // Allowed working directories for execution
	defaultTimeout    time.Duration // Default timeout for command execution
	maxOutputSize     int           // Maximum size of captured output in bytes
	dangerousCommands []string      // Commands that require explicit approval
}

// CommandResult represents the result of a command execution.
type CommandResult struct {
	Command    string `json:"command"`
	Args       []string `json:"args"`
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	WorkingDir string `json:"working_dir"`
	Duration   string `json:"duration"`
}

// CommandRequest represents a command execution request.
type CommandRequest struct {
	Command    string   `json:"command"`
	Args       []string `json:"args"`
	WorkingDir string   `json:"working_dir"`
	Timeout    int      `json:"timeout"` // Timeout in seconds
	Env        []string `json:"env"`     // Environment variables
}

// NewCommandService creates a new command execution MCP service.
// allowedCommands defines the commands that can be executed (security restriction).
// allowedDirs defines the directories where commands can be executed.
func NewCommandService(allowedCommands, allowedDirs []string, logger *log.Logger) *CommandService {
	base := NewBaseService(
		"command",
		"Command execution with security restrictions and output capture",
		logger,
	)

	// Default allowed commands if none provided (safe subset)
	if len(allowedCommands) == 0 {
		allowedCommands = []string{
			"ls", "pwd", "echo", "cat", "head", "tail", "grep", "find", "wc",
			"git", "go", "npm", "node", "python", "python3",
			"mkdir", "touch", "cp", "mv", // Basic file operations
		}
	}

	// Default to current working directory if no allowed dirs provided
	if len(allowedDirs) == 0 {
		if cwd, err := os.Getwd(); err == nil {
			allowedDirs = []string{cwd}
		} else {
			allowedDirs = []string{"."}
		}
	}

	// Clean and normalize all allowed directories
	cleanDirs := make([]string, len(allowedDirs))
	for i, dir := range allowedDirs {
		if absDir, err := filepath.Abs(dir); err == nil {
			cleanDirs[i] = filepath.Clean(absDir)
		} else {
			cleanDirs[i] = filepath.Clean(dir)
		}
	}

	// Commands that are potentially dangerous and require explicit approval
	dangerousCommands := []string{
		"rm", "rmdir", "del", "delete",
		"sudo", "su", "chmod", "chown",
		"format", "fdisk", "mkfs",
		"shutdown", "reboot", "halt",
		"kill", "killall", "pkill",
		"dd", "shred", "wipe",
	}

	return &CommandService{
		BaseService:       base,
		allowedCommands:   allowedCommands,
		allowedDirs:       cleanDirs,
		defaultTimeout:    30 * time.Second,
		maxOutputSize:     1024 * 1024, // 1MB
		dangerousCommands: dangerousCommands,
	}
}

// ValidateParams validates parameters for command execution operations.
func (cs *CommandService) ValidateParams(params ServiceParams) error {
	if err := cs.BaseService.ValidateParams(params); err != nil {
		return err
	}

	// Validate command parameter
	if err := ValidateStringParam(params, "command", true); err != nil {
		return err
	}

	// Validate optional parameters
	if err := cs.validateOptionalParams(params); err != nil {
		return err
	}

	return cs.validateSecurity(params)
}

// validateOptionalParams validates optional command parameters.
func (cs *CommandService) validateOptionalParams(params ServiceParams) error {
	// Validate working directory if provided
	if err := ValidateStringParam(params, "working_dir", false); err != nil {
		return err
	}

	// Validate timeout if provided
	minTimeout := 1
	maxTimeout := 300 // 5 minutes max
	if err := ValidateIntParam(params, "timeout", false, &minTimeout, &maxTimeout); err != nil {
		return err
	}

	// Validate args if provided
	if args, exists := params["args"]; exists {
		if args != nil {
			switch v := args.(type) {
			case []interface{}:
				// Convert []interface{} to []string and validate
				for i, arg := range v {
					if _, ok := arg.(string); !ok {
						return NewValidationError("args", fmt.Sprintf("argument at index %d must be a string", i))
					}
				}
			case []string:
				// Already valid
			default:
				return NewValidationError("args", "args must be an array of strings")
			}
		}
	}

	// Validate environment variables if provided
	if env, exists := params["env"]; exists {
		if env != nil {
			switch v := env.(type) {
			case []interface{}:
				for i, envVar := range v {
					if envStr, ok := envVar.(string); ok {
						if !strings.Contains(envStr, "=") {
							return NewValidationError("env", fmt.Sprintf("environment variable at index %d must be in KEY=VALUE format", i))
						}
					} else {
						return NewValidationError("env", fmt.Sprintf("environment variable at index %d must be a string", i))
					}
				}
			case []string:
				for i, envStr := range v {
					if !strings.Contains(envStr, "=") {
						return NewValidationError("env", fmt.Sprintf("environment variable at index %d must be in KEY=VALUE format", i))
					}
				}
			default:
				return NewValidationError("env", "env must be an array of strings in KEY=VALUE format")
			}
		}
	}

	return nil
}

// validateSecurity performs security validation for command execution.
func (cs *CommandService) validateSecurity(params ServiceParams) error {
	command := params["command"].(string)

	// Check if command is dangerous first (higher priority)
	for _, dangerous := range cs.dangerousCommands {
		if command == dangerous {
			return NewValidationError("command", fmt.Sprintf("command '%s' is potentially dangerous and requires explicit approval", command))
		}
	}

	// Check if command is in allowlist
	commandAllowed := false
	for _, allowed := range cs.allowedCommands {
		if command == allowed {
			commandAllowed = true
			break
		}
	}

	if !commandAllowed {
		return NewValidationError("command", fmt.Sprintf("command '%s' is not in the allowed commands list", command))
	}

	// Validate working directory if provided
	if workingDir, exists := params["working_dir"]; exists && workingDir != nil {
		workDirStr := workingDir.(string)
		if workDirStr != "" {
			if err := cs.validateWorkingDirectory(workDirStr); err != nil {
				return NewValidationError("working_dir", err.Error())
			}
		}
	}

	return nil
}

// validateWorkingDirectory checks if the working directory is allowed.
func (cs *CommandService) validateWorkingDirectory(workingDir string) error {
	// Get absolute path
	absDir, err := filepath.Abs(workingDir)
	if err != nil {
		return fmt.Errorf("invalid working directory: %w", err)
	}

	cleanDir := filepath.Clean(absDir)

	// Check if directory is within any allowed directory
	for _, allowedDir := range cs.allowedDirs {
		relPath, err := filepath.Rel(allowedDir, cleanDir)
		if err != nil {
			continue
		}

		// Directory is valid if it doesn't contain ".." (trying to escape)
		if !strings.Contains(relPath, "..") {
			return nil
		}
	}

	return fmt.Errorf("working directory '%s' is outside allowed directories", workingDir)
}

// Execute performs command execution with the given parameters.
func (cs *CommandService) Execute(ctx context.Context, params ServiceParams) ServiceResult {
	// Extract command parameters
	commandStr := params["command"].(string)

	// Extract and convert args
	var args []string
	if argsParam, exists := params["args"]; exists && argsParam != nil {
		switch v := argsParam.(type) {
		case []interface{}:
			args = make([]string, len(v))
			for i, arg := range v {
				args[i] = arg.(string)
			}
		case []string:
			args = v
		}
	}

	// Extract working directory
	workingDir := ""
	if wd, exists := params["working_dir"]; exists && wd != nil {
		workingDir = wd.(string)
	}
	if workingDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			workingDir = cwd
		} else {
			workingDir = "."
		}
	}

	// Extract timeout
	timeout := cs.defaultTimeout
	if timeoutParam, exists := params["timeout"]; exists && timeoutParam != nil {
		if timeoutInt, ok := timeoutParam.(int); ok {
			timeout = time.Duration(timeoutInt) * time.Second
		} else if timeoutFloat, ok := timeoutParam.(float64); ok {
			timeout = time.Duration(timeoutFloat) * time.Second
		}
	}

	// Extract environment variables
	var env []string
	if envParam, exists := params["env"]; exists && envParam != nil {
		switch v := envParam.(type) {
		case []interface{}:
			env = make([]string, len(v))
			for i, envVar := range v {
				env[i] = envVar.(string)
			}
		case []string:
			env = v
		}
	}

	return cs.executeCommand(ctx, commandStr, args, workingDir, timeout, env)
}

// executeCommand performs the actual command execution.
func (cs *CommandService) executeCommand(ctx context.Context, command string, args []string, workingDir string, timeout time.Duration, env []string) ServiceResult {
	start := time.Now()

	// Create command context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create the command
	cmd := exec.CommandContext(cmdCtx, command, args...)
	cmd.Dir = workingDir

	// Set environment
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	// Create buffers for output capture
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute the command
	err := cmd.Run()

	duration := time.Since(start)

	// Get exit code
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			// Command failed to start or other error
			return ErrorResult(fmt.Errorf("command execution failed: %w", err))
		}
	}

	// Capture outputs with size limits
	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	if len(stdoutStr) > cs.maxOutputSize {
		stdoutStr = stdoutStr[:cs.maxOutputSize] + "\n... (output truncated)"
	}
	if len(stderrStr) > cs.maxOutputSize {
		stderrStr = stderrStr[:cs.maxOutputSize] + "\n... (output truncated)"
	}

	result := CommandResult{
		Command:    command,
		Args:       args,
		ExitCode:   exitCode,
		Stdout:     stdoutStr,
		Stderr:     stderrStr,
		WorkingDir: workingDir,
		Duration:   duration.String(),
	}

	return SuccessResult(result)
}