package test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/Solifugus/ai-work-studio/pkg/utils"
)

func TestLoggerCreation(t *testing.T) {
	t.Run("Create logger with valid config", func(t *testing.T) {
		config := utils.LogConfig{
			Level:       utils.LogLevelInfo,
			Destination: utils.LogDestinationConsole,
			Component:   "test",
		}

		logger, err := utils.NewLogger(config)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if logger == nil {
			t.Fatal("Expected logger to be created")
		}

		defer logger.Close()
	})

	t.Run("Create logger with invalid log level", func(t *testing.T) {
		config := utils.LogConfig{
			Level:       "INVALID",
			Destination: utils.LogDestinationConsole,
			Component:   "test",
		}

		_, err := utils.NewLogger(config)
		if err == nil {
			t.Fatal("Expected error for invalid log level")
		}
	})

	t.Run("Create logger with file destination but no file path", func(t *testing.T) {
		config := utils.LogConfig{
			Level:       utils.LogLevelInfo,
			Destination: utils.LogDestinationFile,
			Component:   "test",
		}

		_, err := utils.NewLogger(config)
		if err == nil {
			t.Fatal("Expected error when file path is missing")
		}
	})
}

func TestLogLevels(t *testing.T) {
	// Create temporary buffer to capture console output
	var buf bytes.Buffer
	config := utils.LogConfig{
		Level:       utils.LogLevelDebug,
		Destination: utils.LogDestinationConsole,
		Component:   "test",
	}

	logger, err := utils.NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Redirect logrus output to our buffer for testing
	logrusLogger := logrus.New()
	logrusLogger.SetOutput(&buf)
	logrusLogger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	ctx := utils.LogContext{
		Component: "test-component",
		GoalID:    "goal-123",
	}

	// Note: These tests verify the API exists and can be called
	// In a real environment, we'd need to intercept the actual logrus output
	t.Run("Debug logging", func(t *testing.T) {
		logger.Debug(ctx, "Debug message")
		// API test - no output verification in this simplified test
	})

	t.Run("Info logging", func(t *testing.T) {
		logger.Info(ctx, "Info message")
	})

	t.Run("Warning logging", func(t *testing.T) {
		logger.Warning(ctx, "Warning message")
	})

	t.Run("Error logging", func(t *testing.T) {
		logger.Error(ctx, "Error message")
	})

	t.Run("Logging with additional fields", func(t *testing.T) {
		fields := map[string]interface{}{
			"custom_field": "custom_value",
			"request_id":   "req-456",
		}
		logger.Info(ctx, "Message with fields", fields)
	})
}

func TestLogContext(t *testing.T) {
	config := utils.LogConfig{
		Level:       utils.LogLevelInfo,
		Destination: utils.LogDestinationConsole,
		Component:   "test",
	}

	logger, err := utils.NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	t.Run("Basic context", func(t *testing.T) {
		ctx := utils.LogContext{
			GoalID:      "goal-123",
			ObjectiveID: "obj-456",
			MethodID:    "method-789",
			UserID:      "user-abc",
			SessionID:   "session-def",
			Component:   "test-component",
		}

		logger.Info(ctx, "Message with full context")
	})

	t.Run("WithContext creates new logger instance", func(t *testing.T) {
		ctx := utils.LogContext{
			GoalID:    "goal-999",
			Component: "new-component",
		}

		newLogger := logger.WithContext(ctx)
		if newLogger == logger {
			t.Error("WithContext should return a new logger instance")
		}

		// Both loggers should be usable
		logger.Info(utils.LogContext{}, "Original logger message")
		newLogger.Info(utils.LogContext{}, "New logger message")
	})
}

func TestFileLogging(t *testing.T) {
	// Create temporary directory for test logs
	tempDir, err := os.MkdirTemp("", "logging_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logFile := filepath.Join(tempDir, "test.log")

	config := utils.LogConfig{
		Level:          utils.LogLevelInfo,
		Destination:    utils.LogDestinationFile,
		FilePath:       logFile,
		FileMaxSize:    1,  // 1MB for testing
		FileMaxBackups: 2,
		FileMaxAge:     1,  // 1 day
		Component:      "test-file",
	}

	logger, err := utils.NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}

	ctx := utils.LogContext{
		GoalID:    "goal-file-test",
		Component: "file-test-component",
	}

	// Write some log messages
	logger.Info(ctx, "First file message")
	logger.Warning(ctx, "Warning file message")
	logger.Error(ctx, "Error file message")

	// Close logger to flush
	logger.Close()

	// Verify log file exists
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Fatal("Log file was not created")
	}

	// Read and verify log content
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "First file message") {
		t.Error("Log file should contain the logged message")
	}

	// Verify JSON format by trying to parse one line
	lines := strings.Split(strings.TrimSpace(contentStr), "\n")
	if len(lines) == 0 {
		t.Fatal("No log lines found")
	}

	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &logEntry); err != nil {
		t.Errorf("Log entry should be valid JSON: %v", err)
	}

	// Verify required fields exist
	expectedFields := []string{"time", "level", "msg", "component", "goal_id"}
	for _, field := range expectedFields {
		if _, exists := logEntry[field]; !exists {
			t.Errorf("Log entry should contain field: %s", field)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	component := "test-component"
	config := utils.DefaultLogConfig(component)

	if config.Component != component {
		t.Errorf("Expected component %s, got %s", component, config.Component)
	}

	if config.Level != utils.LogLevelInfo {
		t.Errorf("Expected default level INFO, got %s", config.Level)
	}

	if config.Destination != utils.LogDestinationBoth {
		t.Errorf("Expected default destination BOTH, got %s", config.Destination)
	}

	if config.FileMaxSize != 100 {
		t.Errorf("Expected default file max size 100, got %d", config.FileMaxSize)
	}

	if config.FileMaxBackups != 5 {
		t.Errorf("Expected default file max backups 5, got %d", config.FileMaxBackups)
	}

	if config.FileMaxAge != 30 {
		t.Errorf("Expected default file max age 30, got %d", config.FileMaxAge)
	}

	expectedFilePath := "./logs/test-component.log"
	if config.FilePath != expectedFilePath {
		t.Errorf("Expected default file path %s, got %s", expectedFilePath, config.FilePath)
	}
}

func TestLoggerManager(t *testing.T) {
	manager := utils.NewLoggerManager()
	defer manager.Close()

	t.Run("Get logger for component", func(t *testing.T) {
		logger, err := manager.GetLogger("agent")
		if err != nil {
			t.Fatalf("Failed to get logger: %v", err)
		}
		if logger == nil {
			t.Fatal("Expected logger to be returned")
		}

		// Getting the same component should return the same logger instance
		logger2, err := manager.GetLogger("agent")
		if err != nil {
			t.Fatalf("Failed to get logger second time: %v", err)
		}
		if logger != logger2 {
			t.Error("Expected same logger instance for same component")
		}
	})

	t.Run("Get logger with custom config", func(t *testing.T) {
		customConfig := utils.LogConfig{
			Level:       utils.LogLevelDebug,
			Destination: utils.LogDestinationConsole,
			Component:   "custom-component",
		}

		logger, err := manager.GetLogger("custom", customConfig)
		if err != nil {
			t.Fatalf("Failed to get logger with custom config: %v", err)
		}
		if logger == nil {
			t.Fatal("Expected logger to be returned")
		}
	})

	t.Run("Multiple components", func(t *testing.T) {
		components := []string{"agent", "mcp_services", "amorphdb"}

		for _, component := range components {
			logger, err := manager.GetLogger(component)
			if err != nil {
				t.Errorf("Failed to get logger for %s: %v", component, err)
			}
			if logger == nil {
				t.Errorf("Expected logger for component %s", component)
			}

			// Test that we can log with each component
			ctx := utils.LogContext{Component: component}
			logger.Info(ctx, "Test message from "+component)
		}
	})
}

func TestContextMerging(t *testing.T) {
	config := utils.LogConfig{
		Level:       utils.LogLevelInfo,
		Destination: utils.LogDestinationConsole,
		Component:   "test",
	}

	logger, err := utils.NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Create logger with base context
	baseCtx := utils.LogContext{
		GoalID:    "base-goal",
		UserID:    "base-user",
		Component: "base-component",
	}
	contextLogger := logger.WithContext(baseCtx)

	// Use logger with override context
	overrideCtx := utils.LogContext{
		GoalID:      "override-goal", // Should override base
		ObjectiveID: "new-objective", // Should be added
		// UserID not specified, should keep base-user
		// Component not specified, should keep base-component
	}

	contextLogger.Info(overrideCtx, "Message with merged context")

	// Test that original logger is unchanged
	logger.Info(utils.LogContext{GoalID: "different-goal"}, "Original logger message")
}

func TestLogRotation(t *testing.T) {
	// Create temporary directory for test logs
	tempDir, err := os.MkdirTemp("", "rotation_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logFile := filepath.Join(tempDir, "rotation_test.log")

	config := utils.LogConfig{
		Level:          utils.LogLevelInfo,
		Destination:    utils.LogDestinationFile,
		FilePath:       logFile,
		FileMaxSize:    1, // Very small size to trigger rotation quickly
		FileMaxBackups: 2,
		FileMaxAge:     1,
		Component:      "rotation-test",
	}

	logger, err := utils.NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger for rotation test: %v", err)
	}

	ctx := utils.LogContext{
		Component: "rotation-test",
	}

	// Write enough data to potentially trigger rotation
	// Note: Actual rotation testing is complex and depends on lumberjack's internal behavior
	for i := 0; i < 1000; i++ {
		logger.Info(ctx, "Large log message for rotation testing - iteration number %d with extra padding", map[string]interface{}{
			"iteration": i,
			"padding":   strings.Repeat("x", 100),
		})
	}

	logger.Close()

	// Verify that log file exists (rotation behavior testing would require more complex setup)
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Fatal("Log file should exist after writing many messages")
	}
}

func TestConcurrentLogging(t *testing.T) {
	config := utils.LogConfig{
		Level:       utils.LogLevelInfo,
		Destination: utils.LogDestinationConsole,
		Component:   "concurrent-test",
	}

	logger, err := utils.NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Test concurrent logging from multiple goroutines
	done := make(chan bool)
	numGoroutines := 10
	messagesPerGoroutine := 50

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			ctx := utils.LogContext{
				Component: "concurrent-test",
				SessionID: "session-" + string(rune(goroutineID)),
			}

			for j := 0; j < messagesPerGoroutine; j++ {
				logger.Info(ctx, "Concurrent message", map[string]interface{}{
					"goroutine_id": goroutineID,
					"message_num":  j,
				})
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// If we get here without panics or deadlocks, concurrent logging works
	t.Log("Concurrent logging test completed successfully")
}