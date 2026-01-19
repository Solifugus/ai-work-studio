package utils

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"deadline exceeded", context.DeadlineExceeded, true},
		{"canceled context", context.Canceled, false},
		{"connection refused", errors.New("connection refused"), true},
		{"timeout error", errors.New("timeout"), true},
		{"temporary failure", errors.New("temporary failure"), true},
		{"service unavailable", errors.New("service unavailable"), true},
		{"too many requests", errors.New("too many requests"), true},
		{"rate limit", errors.New("rate limit"), true},
		{"network error", errors.New("network error"), true},
		{"generic error", errors.New("some other error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryableError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestCircuitBreaker(t *testing.T) {
	// Create a test logger with console output only
	logger, err := NewLogger(LogConfig{
		Level:       LogLevelError, // Reduce noise in tests
		Destination: LogDestinationConsole,
		Component:   "test",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	context := LogContext{Component: "test"}
	config := DefaultCircuitBreakerConfig()
	config.FailureThreshold = 3 // Lower threshold for testing
	config.Timeout = 100 * time.Millisecond

	cb := NewCircuitBreaker("test-service", config, logger, context)

	// Test initial state (should be closed)
	if !cb.AllowRequest() {
		t.Error("Circuit breaker should initially allow requests")
	}

	// Record failures to open the circuit
	for i := 0; i < config.FailureThreshold; i++ {
		cb.RecordFailure()
	}

	// Circuit should now be open
	if cb.AllowRequest() {
		t.Error("Circuit breaker should be open after threshold failures")
	}

	// Wait for timeout to elapse
	time.Sleep(config.Timeout + 10*time.Millisecond)

	// Should now allow requests (half-open state)
	if !cb.AllowRequest() {
		t.Error("Circuit breaker should allow requests after timeout")
	}

	// Record success to close circuit
	cb.RecordSuccess()

	// Should now be fully closed
	if !cb.AllowRequest() {
		t.Error("Circuit breaker should be closed after success")
	}
}

func TestResilienceManagerBasic(t *testing.T) {
	// Create a test logger
	logger, err := NewLogger(LogConfig{
		Level:       LogLevelError,
		Destination: LogDestinationConsole,
		Component:   "test",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	manager := NewResilienceManager(logger)

	t.Run("SuccessfulOperation", func(t *testing.T) {
		operation := DefaultRetryConfig()
		operation.Name = "test-success"
		operation.ServiceID = "test-service"
		operation.Context = LogContext{Component: "test"}

		fn := func(ctx context.Context) error {
			return nil // Always succeed
		}

		result := manager.ExecuteWithResilience(operation, fn)

		if !result.Success {
			t.Errorf("Expected success, got error: %v", result.Error)
		}
		if result.Attempts != 1 {
			t.Errorf("Expected 1 attempt, got %d", result.Attempts)
		}
	})

	t.Run("RetryAndSuccess", func(t *testing.T) {
		operation := DefaultRetryConfig()
		operation.Name = "test-retry"
		operation.ServiceID = "test-service-2"
		operation.Context = LogContext{Component: "test"}
		operation.MaxRetries = 2
		operation.BaseDelay = 1 * time.Millisecond

		attempts := 0
		fn := func(ctx context.Context) error {
			attempts++
			if attempts < 2 {
				return errors.New("temporary failure")
			}
			return nil
		}

		result := manager.ExecuteWithResilience(operation, fn)

		if !result.Success {
			t.Errorf("Expected success after retry, got error: %v", result.Error)
		}
		if result.Attempts != 2 {
			t.Errorf("Expected 2 attempts, got %d", result.Attempts)
		}
	})

	t.Run("CheckpointCreationAndRecovery", func(t *testing.T) {
		checkpointID := "test-checkpoint"
		operation := "test-operation"
		state := map[string]interface{}{
			"step":   1,
			"status": "processing",
		}
		context := LogContext{Component: "test"}

		// Create checkpoint
		manager.CreateCheckpoint(checkpointID, operation, state, context)

		// Restore checkpoint
		restored, err := manager.RestoreFromCheckpoint(checkpointID)
		if err != nil {
			t.Fatalf("Failed to restore checkpoint: %v", err)
		}

		if restored.ID != checkpointID {
			t.Errorf("Expected ID %s, got %s", checkpointID, restored.ID)
		}
		if restored.Operation != operation {
			t.Errorf("Expected operation %s, got %s", operation, restored.Operation)
		}
		if restored.State["step"] != 1 {
			t.Errorf("Expected step 1, got %v", restored.State["step"])
		}

		// Clear checkpoint
		manager.ClearCheckpoint(checkpointID)

		// Verify it's gone
		_, err = manager.RestoreFromCheckpoint(checkpointID)
		if err == nil {
			t.Error("Checkpoint should not exist after clearing")
		}
	})
}

func TestCalculateDelay(t *testing.T) {
	baseDelay := 100 * time.Millisecond
	maxDelay := 1 * time.Second
	jitterMax := 50 * time.Millisecond

	tests := []struct {
		attempt     int
		expectedMin time.Duration
		expectedMax time.Duration
	}{
		{1, baseDelay, baseDelay + jitterMax},
		{2, 2 * baseDelay, 2*baseDelay + jitterMax},
		{3, 4 * baseDelay, 4*baseDelay + jitterMax},
		{10, maxDelay, maxDelay + jitterMax}, // Should be capped at maxDelay
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			delay := calculateDelay(tt.attempt, baseDelay, maxDelay, jitterMax)

			if delay < tt.expectedMin {
				t.Errorf("Delay %v is less than expected minimum %v", delay, tt.expectedMin)
			}
			if delay > tt.expectedMax {
				t.Errorf("Delay %v is greater than expected maximum %v", delay, tt.expectedMax)
			}
		})
	}
}

func BenchmarkRetryExecution(b *testing.B) {
	logger, err := NewLogger(LogConfig{
		Level:       LogLevelError,
		Destination: LogDestinationConsole,
		Component:   "benchmark",
	})
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	manager := NewResilienceManager(logger)

	operation := DefaultRetryConfig()
	operation.Name = "benchmark-operation"
	operation.ServiceID = "benchmark-service"
	operation.Context = LogContext{Component: "benchmark"}
	operation.MaxRetries = 0 // No retries for pure performance

	successfulOperation := func(ctx context.Context) error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := manager.ExecuteWithResilience(operation, successfulOperation)
		if !result.Success {
			b.Fatalf("Operation should succeed: %v", result.Error)
		}
	}
}