// Package test contains integration tests for the AI Work Studio resilience system.
package test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Solifugus/ai-work-studio/pkg/utils"
)

// TestRetryWithExponentialBackoff tests the retry mechanism with exponential backoff.
func TestRetryWithExponentialBackoff(t *testing.T) {
	// Create logger for testing
	logger, err := utils.NewLogger(utils.LogConfig{
		Level:       utils.LogLevelDebug,
		Destination: utils.LogDestinationConsole,
		Component:   "test-resilience",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Create resilience manager
	manager := utils.NewResilienceManager(logger)

	// Test successful operation after retries
	t.Run("SuccessAfterRetries", func(t *testing.T) {
		attempts := 0
		operation := utils.Operation{
			Name:       "test-operation",
			ServiceID:  "test-service",
			Context:    utils.LogContext{Component: "test"},
			MaxRetries: 3,
			BaseDelay:  10 * time.Millisecond,
			MaxDelay:   100 * time.Millisecond,
			JitterMax:  5 * time.Millisecond,
			Timeout:    1 * time.Second,
		}

		fn := func(ctx context.Context) error {
			attempts++
			if attempts < 3 {
				return errors.New("temporary failure")
			}
			return nil
		}

		result := manager.ExecuteWithResilience(operation, fn)

		if !result.Success {
			t.Errorf("Expected success, got error: %v", result.Error)
		}
		if result.Attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", result.Attempts)
		}
		if result.Duration < 20*time.Millisecond { // Should include delays
			t.Errorf("Expected duration > 20ms due to retries, got %v", result.Duration)
		}
	})

	// Test non-retryable error
	t.Run("NonRetryableError", func(t *testing.T) {
		attempts := 0
		operation := utils.DefaultRetryConfig()
		operation.Name = "non-retryable-test"
		operation.ServiceID = "test-service-2"
		operation.Context = utils.LogContext{Component: "test"}

		fn := func(ctx context.Context) error {
			attempts++
			return context.Canceled // Non-retryable error
		}

		result := manager.ExecuteWithResilience(operation, fn)

		if result.Success {
			t.Error("Expected failure for non-retryable error")
		}
		if result.Attempts != 1 {
			t.Errorf("Expected 1 attempt for non-retryable error, got %d", result.Attempts)
		}
		if !errors.Is(result.Error, context.Canceled) {
			t.Errorf("Expected context.Canceled error, got %v", result.Error)
		}
	})

	// Test timeout
	t.Run("TimeoutError", func(t *testing.T) {
		operation := utils.DefaultRetryConfig()
		operation.Name = "timeout-test"
		operation.ServiceID = "test-service-3"
		operation.Context = utils.LogContext{Component: "test"}
		operation.Timeout = 50 * time.Millisecond

		fn := func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		}

		result := manager.ExecuteWithResilience(operation, fn)

		if result.Success {
			t.Error("Expected failure due to timeout")
		}
		if !errors.Is(result.Error, context.DeadlineExceeded) {
			t.Errorf("Expected deadline exceeded error, got %v", result.Error)
		}
	})
}

// TestCircuitBreaker tests the circuit breaker functionality.
func TestCircuitBreaker(t *testing.T) {
	// Create logger for testing
	logger, err := utils.NewLogger(utils.LogConfig{
		Level:       utils.LogLevelDebug,
		Destination: utils.LogDestinationConsole,
		Component:   "test-circuit-breaker",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Create resilience manager
	manager := utils.NewResilienceManager(logger)

	t.Run("CircuitBreakerOpensAfterFailures", func(t *testing.T) {
		serviceID := "failing-service"
		operation := utils.DefaultRetryConfig()
		operation.Name = "circuit-breaker-test"
		operation.ServiceID = serviceID
		operation.Context = utils.LogContext{Component: "test"}
		operation.MaxRetries = 0 // No retries to trigger circuit breaker faster

		// Create a function that always fails
		alwaysFail := func(ctx context.Context) error {
			return errors.New("service unavailable")
		}

		// Execute requests until circuit breaker opens
		failureCount := 0
		for i := 0; i < 10; i++ {
			result := manager.ExecuteWithResilience(operation, alwaysFail)
			if !result.Success {
				failureCount++
			}

			// Check if circuit breaker is open
			state, exists := manager.GetCircuitBreakerState(serviceID)
			if exists && state == utils.CircuitOpen {
				t.Logf("Circuit breaker opened after %d failures", failureCount)
				break
			}
		}

		// Verify circuit breaker is open
		state, exists := manager.GetCircuitBreakerState(serviceID)
		if !exists {
			t.Error("Circuit breaker should exist")
		}
		if state != utils.CircuitOpen {
			t.Errorf("Expected circuit breaker to be open, got state: %v", state)
		}

		// Test that requests are rejected when circuit is open
		result := manager.ExecuteWithResilience(operation, alwaysFail)
		if result.Success {
			t.Error("Expected request to be rejected when circuit is open")
		}
		if result.Attempts != 0 {
			t.Errorf("Expected 0 attempts when circuit is open, got %d", result.Attempts)
		}
	})

	t.Run("CircuitBreakerRecovery", func(t *testing.T) {
		serviceID := "recovery-service"
		context := utils.LogContext{Component: "test"}

		// Create circuit breaker with short timeout for testing
		config := utils.CircuitBreakerConfig{
			FailureThreshold: 2,
			SuccessThreshold: 1,
			Timeout:          100 * time.Millisecond,
			MaxRequests:      5,
		}

		circuitBreaker := utils.NewCircuitBreaker(serviceID, config, logger, context)

		// Record failures to open the circuit
		circuitBreaker.RecordFailure()
		circuitBreaker.RecordFailure()

		// Verify circuit is open
		if circuitBreaker.AllowRequest() {
			t.Error("Circuit should be open after failures")
		}

		// Wait for timeout to elapse
		time.Sleep(150 * time.Millisecond)

		// Circuit should now allow requests (half-open)
		if !circuitBreaker.AllowRequest() {
			t.Error("Circuit should allow requests after timeout")
		}

		// Record success to close circuit
		circuitBreaker.RecordSuccess()

		// Circuit should now be closed
		if !circuitBreaker.AllowRequest() {
			t.Error("Circuit should be closed after success")
		}
	})
}

// TestIsRetryableError tests error classification for retry logic.
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
			result := utils.IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryableError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

// TestCheckpointRecovery tests the checkpoint and recovery functionality.
func TestCheckpointRecovery(t *testing.T) {
	// Create logger for testing
	logger, err := utils.NewLogger(utils.LogConfig{
		Level:       utils.LogLevelDebug,
		Destination: utils.LogDestinationConsole,
		Component:   "test-checkpoint",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Create resilience manager
	manager := utils.NewResilienceManager(logger)

	t.Run("CreateAndRestoreCheckpoint", func(t *testing.T) {
		checkpointID := "test-checkpoint-1"
		operation := "test-operation"
		state := map[string]interface{}{
			"step":      3,
			"processed": 100,
			"remaining": 200,
		}
		context := utils.LogContext{
			Component:   "test",
			ObjectiveID: "obj-123",
		}

		// Create checkpoint
		manager.CreateCheckpoint(checkpointID, operation, state, context)

		// Restore checkpoint
		restored, err := manager.RestoreFromCheckpoint(checkpointID)
		if err != nil {
			t.Fatalf("Failed to restore checkpoint: %v", err)
		}

		// Verify restored data
		if restored.ID != checkpointID {
			t.Errorf("Expected ID %s, got %s", checkpointID, restored.ID)
		}
		if restored.Operation != operation {
			t.Errorf("Expected operation %s, got %s", operation, restored.Operation)
		}
		if restored.State["step"] != 3 {
			t.Errorf("Expected step 3, got %v", restored.State["step"])
		}
		if restored.Context.ObjectiveID != "obj-123" {
			t.Errorf("Expected objective ID obj-123, got %s", restored.Context.ObjectiveID)
		}
	})

	t.Run("RestoreNonexistentCheckpoint", func(t *testing.T) {
		_, err := manager.RestoreFromCheckpoint("nonexistent")
		if err == nil {
			t.Error("Expected error when restoring nonexistent checkpoint")
		}
		expectedMsg := "checkpoint nonexistent not found"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("ClearCheckpoint", func(t *testing.T) {
		checkpointID := "test-checkpoint-2"
		state := map[string]interface{}{"test": "data"}
		context := utils.LogContext{Component: "test"}

		// Create checkpoint
		manager.CreateCheckpoint(checkpointID, "test-op", state, context)

		// Verify it exists
		_, err := manager.RestoreFromCheckpoint(checkpointID)
		if err != nil {
			t.Fatalf("Checkpoint should exist before clearing: %v", err)
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

// TestResilienceIntegration tests the complete resilience system integration.
func TestResilienceIntegration(t *testing.T) {
	// Create logger for testing
	logger, err := utils.NewLogger(utils.LogConfig{
		Level:       utils.LogLevelDebug,
		Destination: utils.LogDestinationConsole,
		Component:   "test-integration",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Create resilience manager
	manager := utils.NewResilienceManager(logger)

	t.Run("CompleteWorkflowWithRecovery", func(t *testing.T) {
		serviceID := "integration-service"
		checkpointID := "workflow-checkpoint"

		// Simulate a multi-step operation that fails partway through
		operation := utils.DefaultRetryConfig()
		operation.Name = "multi-step-workflow"
		operation.ServiceID = serviceID
		operation.Context = utils.LogContext{
			Component:   "integration-test",
			ObjectiveID: "obj-integration",
		}
		operation.MaxRetries = 2
		operation.BaseDelay = 10 * time.Millisecond

		// Step 1: Initial work (succeeds)
		step1 := func(ctx context.Context) error {
			// Create checkpoint after step 1
			state := map[string]interface{}{
				"completed_steps": []string{"step1"},
				"current_step":    "step2",
			}
			manager.CreateCheckpoint(checkpointID, "workflow", state, operation.Context)
			return nil
		}

		result1 := manager.ExecuteWithResilience(operation, step1)
		if !result1.Success {
			t.Fatalf("Step 1 should succeed: %v", result1.Error)
		}

		// Step 2: Simulated failure and recovery
		attempts := 0
		step2 := func(ctx context.Context) error {
			attempts++
			if attempts <= 2 {
				return errors.New("temporary failure") // Will be retried
			}

			// Restore from checkpoint to verify recovery works
			checkpoint, err := manager.RestoreFromCheckpoint(checkpointID)
			if err != nil {
				return fmt.Errorf("failed to restore checkpoint: %w", err)
			}

			completedSteps, ok := checkpoint.State["completed_steps"].([]interface{})
			if !ok || len(completedSteps) != 1 {
				return fmt.Errorf("invalid checkpoint state: %v", checkpoint.State)
			}

			// Success after recovery
			return nil
		}

		result2 := manager.ExecuteWithResilience(operation, step2)
		if !result2.Success {
			t.Fatalf("Step 2 should succeed after retries: %v", result2.Error)
		}
		if result2.Attempts != 3 {
			t.Errorf("Expected 3 attempts for step 2, got %d", result2.Attempts)
		}

		// Clean up checkpoint
		manager.ClearCheckpoint(checkpointID)
	})
}

// BenchmarkRetryPerformance benchmarks the retry mechanism performance.
func BenchmarkRetryPerformance(b *testing.B) {
	logger, err := utils.NewLogger(utils.LogConfig{
		Level:       utils.LogLevelError, // Reduce logging noise in benchmark
		Destination: utils.LogDestinationConsole,
		Component:   "benchmark",
	})
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	manager := utils.NewResilienceManager(logger)

	operation := utils.DefaultRetryConfig()
	operation.Name = "benchmark-operation"
	operation.ServiceID = "benchmark-service"
	operation.Context = utils.LogContext{Component: "benchmark"}
	operation.MaxRetries = 0 // No retries for pure performance test
	operation.Timeout = 10 * time.Second

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