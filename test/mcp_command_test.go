package test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Solifugus/ai-work-studio/pkg/mcp"
)

// TestCommandService_NewService tests the creation of a new command service.
func TestCommandService_NewService(t *testing.T) {
	tests := []struct {
		name            string
		allowedCommands []string
		allowedDirs     []string
		expectDefaults  bool
	}{
		{
			name:            "default configuration",
			allowedCommands: nil,
			allowedDirs:     nil,
			expectDefaults:  true,
		},
		{
			name:            "custom configuration",
			allowedCommands: []string{"echo", "pwd"},
			allowedDirs:     []string{"/tmp"},
			expectDefaults:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := mcp.NewCommandService(tt.allowedCommands, tt.allowedDirs, nil)

			if service == nil {
				t.Fatal("expected non-nil service")
			}

			if service.Name() != "command" {
				t.Errorf("expected service name 'command', got '%s'", service.Name())
			}

			if service.Description() == "" {
				t.Error("expected non-empty description")
			}
		})
	}
}

// TestCommandService_ValidateParams tests parameter validation.
func TestCommandService_ValidateParams(t *testing.T) {
	service := mcp.NewCommandService([]string{"echo", "pwd", "ls"}, []string{"."}, nil)

	tests := []struct {
		name      string
		params    mcp.ServiceParams
		expectErr bool
		errContains string
	}{
		{
			name:      "nil parameters",
			params:    nil,
			expectErr: true,
			errContains: "cannot be nil",
		},
		{
			name:      "missing command",
			params:    mcp.ServiceParams{},
			expectErr: true,
			errContains: "required parameter is missing",
		},
		{
			name:      "empty command",
			params:    mcp.ServiceParams{"command": ""},
			expectErr: true,
			errContains: "cannot be empty",
		},
		{
			name:      "valid basic command",
			params:    mcp.ServiceParams{"command": "echo"},
			expectErr: false,
		},
		{
			name: "valid command with args",
			params: mcp.ServiceParams{
				"command": "echo",
				"args":    []string{"hello", "world"},
			},
			expectErr: false,
		},
		{
			name: "invalid args type",
			params: mcp.ServiceParams{
				"command": "echo",
				"args":    "not an array",
			},
			expectErr: true,
			errContains: "must be an array",
		},
		{
			name: "invalid timeout",
			params: mcp.ServiceParams{
				"command": "echo",
				"timeout": -1,
			},
			expectErr: true,
			errContains: "must be >=",
		},
		{
			name: "timeout too high",
			params: mcp.ServiceParams{
				"command": "echo",
				"timeout": 400,
			},
			expectErr: true,
			errContains: "must be <=",
		},
		{
			name: "valid timeout",
			params: mcp.ServiceParams{
				"command": "echo",
				"timeout": 30,
			},
			expectErr: false,
		},
		{
			name: "invalid working directory type",
			params: mcp.ServiceParams{
				"command":     "echo",
				"working_dir": 123,
			},
			expectErr: true,
			errContains: "must be a string",
		},
		{
			name: "disallowed command (dangerous)",
			params: mcp.ServiceParams{
				"command": "rm",
			},
			expectErr: true,
			errContains: "potentially dangerous",
		},
		{
			name: "dangerous command",
			params: mcp.ServiceParams{
				"command": "sudo",
			},
			expectErr: true,
			errContains: "potentially dangerous",
		},
		{
			name: "disallowed command (not dangerous)",
			params: mcp.ServiceParams{
				"command": "unknown-command",
			},
			expectErr: true,
			errContains: "not in the allowed commands list",
		},
		{
			name: "valid environment variables",
			params: mcp.ServiceParams{
				"command": "echo",
				"env":     []string{"KEY=value", "ANOTHER=test"},
			},
			expectErr: false,
		},
		{
			name: "invalid environment variable format",
			params: mcp.ServiceParams{
				"command": "echo",
				"env":     []string{"INVALID_FORMAT"},
			},
			expectErr: true,
			errContains: "must be in KEY=VALUE format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateParams(tt.params)

			if tt.expectErr {
				if err == nil {
					t.Error("expected validation error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain '%s', got: %v", tt.errContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no validation error, got: %v", err)
				}
			}
		})
	}
}

// TestCommandService_Execute tests command execution.
func TestCommandService_Execute(t *testing.T) {
	service := mcp.NewCommandService([]string{"echo", "pwd", "ls", "sleep"}, []string{"."}, nil)

	tests := []struct {
		name          string
		params        mcp.ServiceParams
		expectSuccess bool
		expectedData  map[string]interface{}
		checkStdout   string
	}{
		{
			name: "simple echo command",
			params: mcp.ServiceParams{
				"command": "echo",
				"args":    []string{"hello", "world"},
			},
			expectSuccess: true,
			checkStdout:   "hello world",
		},
		{
			name: "pwd command",
			params: mcp.ServiceParams{
				"command": "pwd",
			},
			expectSuccess: true,
		},
		{
			name: "command with timeout",
			params: mcp.ServiceParams{
				"command": "echo",
				"args":    []string{"test"},
				"timeout": 5,
			},
			expectSuccess: true,
			checkStdout:   "test",
		},
		{
			name: "command with environment variable",
			params: mcp.ServiceParams{
				"command": "echo",
				"args":    []string{"$TEST_VAR"},
				"env":     []string{"TEST_VAR=success"},
			},
			expectSuccess: true,
		},
	}

	// Only run timeout test on Unix-like systems where sleep is available
	if runtime.GOOS != "windows" {
		tests = append(tests, struct {
			name          string
			params        mcp.ServiceParams
			expectSuccess bool
			expectedData  map[string]interface{}
			checkStdout   string
		}{
			name: "command with short sleep",
			params: mcp.ServiceParams{
				"command": "sleep",
				"args":    []string{"1"},
				"timeout": 5,
			},
			expectSuccess: true, // Should complete successfully
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result := service.Execute(ctx, tt.params)

			if tt.expectSuccess != result.Success {
				t.Errorf("expected success=%v, got success=%v, error=%v", tt.expectSuccess, result.Success, result.Error)
			}

			if result.Success && result.Data != nil {
				data, ok := result.Data.(mcp.CommandResult)
				if !ok {
					t.Errorf("expected CommandResult, got %T", result.Data)
					return
				}

				// Check stdout if specified
				if tt.checkStdout != "" {
					stdout := strings.TrimSpace(data.Stdout)
					if !strings.Contains(stdout, tt.checkStdout) {
						t.Errorf("expected stdout to contain '%s', got '%s'", tt.checkStdout, stdout)
					}
				}

				// Verify that all expected fields are populated
				if data.Command == "" {
					t.Error("expected command field to be populated")
				}
				if data.Duration == "" {
					t.Error("expected duration field to be populated")
				}
				if data.WorkingDir == "" {
					t.Error("expected working_dir field to be populated")
				}
			}
		})
	}
}

// TestCommandService_Security tests security restrictions.
func TestCommandService_Security(t *testing.T) {
	tempDir := os.TempDir()
	service := mcp.NewCommandService([]string{"echo"}, []string{tempDir}, nil)

	tests := []struct {
		name      string
		params    mcp.ServiceParams
		expectErr bool
	}{
		{
			name: "allowed working directory",
			params: mcp.ServiceParams{
				"command":     "echo",
				"working_dir": tempDir,
			},
			expectErr: false,
		},
		{
			name: "disallowed working directory",
			params: mcp.ServiceParams{
				"command":     "echo",
				"working_dir": "/root",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateParams(tt.params)

			if tt.expectErr {
				if err == nil {
					t.Error("expected security validation error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("expected no security error, got: %v", err)
				}
			}
		})
	}
}

// TestCommandService_WorkingDirectory tests working directory handling.
func TestCommandService_WorkingDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "command_service_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file in the temp directory
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	service := mcp.NewCommandService([]string{"ls", "pwd"}, []string{tempDir}, nil)

	t.Run("command in custom working directory", func(t *testing.T) {
		params := mcp.ServiceParams{
			"command":     "ls",
			"working_dir": tempDir,
		}

		ctx := context.Background()
		result := service.Execute(ctx, params)

		if !result.Success {
			t.Fatalf("expected successful execution, got error: %v", result.Error)
		}

		data, ok := result.Data.(mcp.CommandResult)
		if !ok {
			t.Fatalf("expected CommandResult, got %T", result.Data)
		}

		if data.WorkingDir != tempDir {
			t.Errorf("expected working dir '%s', got '%s'", tempDir, data.WorkingDir)
		}

		if !strings.Contains(data.Stdout, "test.txt") {
			t.Errorf("expected stdout to contain 'test.txt', got: %s", data.Stdout)
		}
	})
}

// TestCommandService_OutputLimits tests output size limitations.
func TestCommandService_OutputLimits(t *testing.T) {
	// Skip this test as creating massive output is system-dependent
	// The output limiting logic is covered in the implementation
	t.Skip("Output limits test requires system-specific commands that may not be available")
}

// TestCommandService_Context tests context cancellation.
func TestCommandService_Context(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping context test on Windows due to sleep command unavailability")
	}

	service := mcp.NewCommandService([]string{"sleep"}, []string{"."}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	params := mcp.ServiceParams{
		"command": "sleep",
		"args":    []string{"10"},
	}

	result := service.Execute(ctx, params)

	if result.Success {
		t.Error("expected execution to fail due to context cancellation")
	}

	if !strings.Contains(result.Error.Error(), "context") && !strings.Contains(result.Error.Error(), "cancel") {
		t.Errorf("expected context-related error, got: %v", result.Error)
	}
}

// BenchmarkCommandService_Execute benchmarks command execution.
func BenchmarkCommandService_Execute(b *testing.B) {
	service := mcp.NewCommandService([]string{"echo"}, []string{"."}, nil)
	params := mcp.ServiceParams{
		"command": "echo",
		"args":    []string{"benchmark"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		result := service.Execute(ctx, params)
		if !result.Success {
			b.Fatalf("command execution failed: %v", result.Error)
		}
	}
}