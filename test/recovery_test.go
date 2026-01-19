package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/yourusername/ai-work-studio/pkg/core"
)

// TestExecutionRecoveryScenarios tests various failure and recovery scenarios.
func TestExecutionRecoveryScenarios(t *testing.T) {
	fixtures := NewTestFixtures(t)
	ctx := context.Background()

	t.Run("TaskFailureRecovery", func(t *testing.T) {
		// Test recovery from individual task failures
		mockReasoner := NewMockLLMReasoner()
		mockExecutor := NewMockTaskExecutor()
		mockContextLoader := NewMockContextLoader()

		// Configure executor to fail specific task types
		mockExecutor.FailOnTaskType = "data_collection"

		cc := core.NewContemplativeCursor(fixtures.Store, mockReasoner)
		rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)

		objective := fixtures.GetObjectiveByTitle("Analyze Customer Satisfaction Issues")
		if objective == nil {
			t.Fatal("Could not find test objective")
		}

		plan, err := cc.CreateExecutionPlan(ctx, objective.ID)
		if err != nil {
			t.Fatalf("Failed to create execution plan: %v", err)
		}

		result, err := rtc.ExecutePlan(ctx, plan)
		if err != nil {
			t.Fatalf("Failed to execute plan with task failures: %v", err)
		}

		// Should handle failures gracefully
		if result.FailedTasks == 0 {
			t.Error("Expected some tasks to fail based on mock configuration")
		}

		// Non-critical failures should allow execution to continue
		if result.SuccessfulTasks == 0 {
			t.Error("Expected some tasks to succeed despite failures")
		}

		// Verify appropriate status
		expectedStatus := core.ExecutionStatusPartial
		if result.FailedTasks == len(plan.Tasks) {
			expectedStatus = core.ExecutionStatusFailed
		}
		if result.Status != expectedStatus {
			t.Errorf("Expected status %s, got %s", expectedStatus, result.Status)
		}
	})

	t.Run("CriticalTaskFailureHandling", func(t *testing.T) {
		// Test handling of critical task failures that should stop execution
		mockExecutor := NewMockTaskExecutor()
		mockContextLoader := NewMockContextLoader()

		rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)

		// Create plan with a high-priority task that will fail
		plan := fixtures.CreateTestExecutionPlan("critical-test-obj", "critical-test-method")

		// Make the first task high priority (critical)
		plan.Tasks[0].Context.Priority = 10

		// Configure executor to fail the critical task
		mockExecutor.CustomResults[plan.Tasks[0].ID] = &core.TaskResult{
			TaskID:       plan.Tasks[0].ID,
			Status:       core.TaskStatusFailed,
			ErrorMessage: "Critical task failure",
			TokensUsed:   100,
			Duration:     50 * time.Millisecond,
			CompletedAt:  time.Now(),
		}

		result, err := rtc.ExecutePlan(ctx, plan)

		// Critical task failure should stop execution
		if err == nil {
			t.Error("Expected error due to critical task failure")
		}

		if result.Status != core.ExecutionStatusFailed {
			t.Errorf("Expected failed status due to critical task failure, got %s", result.Status)
		}
	})

	t.Run("NetworkTimeoutRecovery", func(t *testing.T) {
		// Test recovery from network timeouts and temporary failures
		mockExecutor := NewMockTaskExecutor()
		mockContextLoader := NewMockContextLoader()

		// Simulate timeout behavior
		mockExecutor.ShouldTimeOut = true

		rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)

		// Configure retry behavior
		retryConfig := core.DefaultRetryConfig()
		retryConfig.MaxRetries = 2
		retryConfig.BaseDelay = 10 * time.Millisecond
		retryConfig.RetriableErrors = []string{"timeout", "context deadline exceeded"}
		rtc.SetRetryConfig(retryConfig)

		plan := fixtures.CreateTestExecutionPlan("timeout-test-obj", "timeout-test-method")

		// Execute with short context timeout to trigger timeout handling
		timeoutCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()

		result, err := rtc.ExecutePlan(timeoutCtx, plan)

		// Should handle timeout gracefully
		if result == nil {
			t.Fatal("Expected result even with timeout")
		}

		// May have error due to timeout
		if err != nil {
			t.Logf("Expected timeout error occurred: %v", err)
		}

		// Should have attempted execution
		if len(result.TaskResults) == 0 {
			t.Error("Expected some task results despite timeout")
		}

		// Status should reflect cancellation
		if result.Status != core.ExecutionStatusCancelled && result.Status != core.ExecutionStatusFailed {
			t.Errorf("Expected cancelled or failed status, got %s", result.Status)
		}
	})

	t.Run("RetryMechanismTesting", func(t *testing.T) {
		// Test retry logic for transient failures
		mockExecutor := NewMockTaskExecutor()
		mockContextLoader := NewMockContextLoader()

		rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)

		// Configure custom retry behavior
		retryConfig := &core.RetryConfig{
			MaxRetries:        2,
			BaseDelay:         5 * time.Millisecond,
			MaxDelay:          20 * time.Millisecond,
			BackoffMultiplier: 2.0,
			RetriableErrors:   []string{"temporary_failure", "rate_limit"},
		}
		rtc.SetRetryConfig(retryConfig)

		plan := fixtures.CreateTestExecutionPlan("retry-test-obj", "retry-test-method")

		// Use the mock's built-in failure mechanism
		mockExecutor.ShouldFailExecution = true

		result, err := rtc.ExecutePlan(ctx, plan)
		if err != nil {
			t.Fatalf("Failed to execute plan with retry: %v", err)
		}

		// Should handle retries (though may not succeed with constant failure)
		if result.Status == core.ExecutionStatusPending {
			t.Error("Expected execution to complete (success or failure), not remain pending")
		}

		// Verify execution was attempted
		if len(mockExecutor.ExecuteCalls) == 0 {
			t.Error("Expected at least one execution attempt")
		}
	})

	t.Run("ContextCancellationHandling", func(t *testing.T) {
		// Test proper handling of context cancellation
		mockExecutor := NewMockTaskExecutor()
		mockContextLoader := NewMockContextLoader()

		rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)

		plan := fixtures.CreateTestExecutionPlan("cancel-test-obj", "cancel-test-method")

		// Create context that will be cancelled mid-execution
		cancelCtx, cancel := context.WithCancel(ctx)

		// Start execution
		resultChan := make(chan *core.ExecutionResult, 1)
		errChan := make(chan error, 1)

		go func() {
			result, err := rtc.ExecutePlan(cancelCtx, plan)
			resultChan <- result
			errChan <- err
		}()

		// Cancel after short delay
		time.Sleep(10 * time.Millisecond)
		cancel()

		// Wait for execution to complete
		select {
		case result := <-resultChan:
			err := <-errChan

			// Should handle cancellation gracefully
			if result == nil {
				t.Fatal("Expected result even with cancellation")
			}

			if result.Status != core.ExecutionStatusCancelled {
				t.Errorf("Expected cancelled status, got %s", result.Status)
			}

			if err != context.Canceled {
				t.Errorf("Expected context.Canceled error, got %v", err)
			}

		case <-time.After(1 * time.Second):
			t.Fatal("Execution did not complete within timeout after cancellation")
		}
	})
}

// TestStorageFailureRecovery tests recovery from storage-related failures.
func TestStorageFailureRecovery(t *testing.T) {
	fixtures := NewTestFixtures(t)
	ctx := context.Background()

	t.Run("NodeRetrievalFailure", func(t *testing.T) {
		// Test handling of storage failures during node retrieval
		mockReasoner := NewMockLLMReasoner()
		cc := core.NewContemplativeCursor(fixtures.Store, mockReasoner)

		// Try to create plan for non-existent objective
		invalidObjectiveID := "non-existent-objective-id"

		_, err := cc.CreateExecutionPlan(ctx, invalidObjectiveID)

		// Should fail gracefully with appropriate error
		if err == nil {
			t.Error("Expected error for non-existent objective")
		}

		// Error should be descriptive
		if !containsString(err.Error(), "objective") {
			t.Errorf("Expected error message to mention objective, got: %v", err)
		}
	})

	t.Run("MethodLookupFailure", func(t *testing.T) {
		// Test handling when referenced method doesn't exist
		mockReasoner := NewMockLLMReasoner()
		cc := core.NewContemplativeCursor(fixtures.Store, mockReasoner)

		// Create objective with invalid method ID
		goal := fixtures.SampleGoals[0]
		invalidMethodID := "non-existent-method-id"

		invalidObjective, err := fixtures.ObjectiveManager.CreateObjective(ctx,
			goal.ID,
			invalidMethodID,
			"Test Invalid Method Reference",
			"Testing method lookup failure",
			map[string]interface{}{},
			5)
		if err != nil {
			t.Fatalf("Failed to create objective with invalid method: %v", err)
		}

		// Should handle method lookup failure gracefully
		_, err = cc.CreateExecutionPlan(ctx, invalidObjective.ID)

		// Should either fail gracefully or create new method
		// (Depending on implementation - either behavior is acceptable)
		if err != nil {
			// If it fails, error should be descriptive
			if !containsString(err.Error(), "method") {
				t.Errorf("Expected error message to mention method, got: %v", err)
			}
		}
	})

	t.Run("StorageCorruptionRecovery", func(t *testing.T) {
		// Test behavior when storage contains corrupted data
		// This is more of a robustness test since we can't easily corrupt the mock storage

		// Create method with minimal valid data
		minimalMethod, err := fixtures.MethodManager.CreateMethod(ctx,
			"Minimal Method",
			"Method with minimal data for corruption testing",
			[]core.ApproachStep{
				{Description: "Basic step", Tools: []string{}},
			},
			core.MethodDomainGeneral,
			map[string]interface{}{})
		if err != nil {
			t.Fatalf("Failed to create minimal method: %v", err)
		}

		// Verify we can retrieve it
		retrievedMethod, err := fixtures.MethodManager.GetMethod(ctx, minimalMethod.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve minimal method: %v", err)
		}

		if retrievedMethod.Name != minimalMethod.Name {
			t.Errorf("Retrieved method name mismatch: expected %s, got %s",
				minimalMethod.Name, retrievedMethod.Name)
		}
	})
}

// TestSystemLimitsAndBoundaryConditions tests system behavior at limits.
func TestSystemLimitsAndBoundaryConditions(t *testing.T) {
	fixtures := NewTestFixtures(t)
	ctx := context.Background()

	t.Run("EmptyPlanExecution", func(t *testing.T) {
		// Test execution of empty plan
		mockExecutor := NewMockTaskExecutor()
		mockContextLoader := NewMockContextLoader()
		rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)

		emptyPlan := &core.ExecutionPlan{
			ID:                   "empty-plan",
			ObjectiveID:          "test-obj",
			MethodID:             "test-method",
			Title:                "Empty Plan",
			Tasks:                []core.ExecutionTask{},
			Dependencies:         []core.TaskDependency{},
			TotalEstimatedTokens: 0,
			CreatedBy:            "test",
			CreatedAt:            time.Now(),
		}

		result, err := rtc.ExecutePlan(ctx, emptyPlan)

		// Should handle empty plan gracefully
		if err == nil {
			t.Error("Expected error for empty plan")
		}

		if result == nil {
			t.Fatal("Expected result structure even for empty plan")
		}

		if result.Status != core.ExecutionStatusFailed {
			t.Errorf("Expected failed status for empty plan, got %s", result.Status)
		}
	})

	t.Run("NilParameterHandling", func(t *testing.T) {
		// Test handling of nil parameters
		mockExecutor := NewMockTaskExecutor()
		mockContextLoader := NewMockContextLoader()
		rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)

		// Try to execute nil plan
		result, err := rtc.ExecutePlan(ctx, nil)

		// Should fail gracefully
		if err == nil {
			t.Error("Expected error for nil plan")
		}

		if result == nil {
			t.Fatal("Expected result structure even for nil plan")
		}

		if result.Status != core.ExecutionStatusFailed {
			t.Errorf("Expected failed status for nil plan, got %s", result.Status)
		}
	})

	t.Run("ExtremelyLargePlan", func(t *testing.T) {
		// Test handling of plan with many tasks
		if testing.Short() {
			t.Skip("Skipping large plan test in short mode")
		}

		mockExecutor := NewMockTaskExecutor()
		mockContextLoader := NewMockContextLoader()

		// Create plan with many tasks
		largePlan := &core.ExecutionPlan{
			ID:          "large-plan",
			ObjectiveID: "large-test-obj",
			MethodID:    "large-test-method",
			Title:       "Large Plan",
			Tasks:       []core.ExecutionTask{},
			CreatedBy:   "test",
			CreatedAt:   time.Now(),
		}

		taskCount := 100
		for i := 0; i < taskCount; i++ {
			task := core.ExecutionTask{
				ID:          fmt.Sprintf("task-%d", i),
				Type:        "test_task",
				Description: fmt.Sprintf("Test task %d", i),
				Context: core.TaskContext{
					TokenBudget: 100,
					Priority:    5,
				},
				EstimatedTokens: 100,
				CreatedAt:      time.Now(),
			}
			largePlan.Tasks = append(largePlan.Tasks, task)
		}

		largePlan.TotalEstimatedTokens = taskCount * 100

		rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)

		startTime := time.Now()
		result, err := rtc.ExecutePlan(ctx, largePlan)
		executionTime := time.Since(startTime)

		if err != nil {
			t.Fatalf("Failed to execute large plan: %v", err)
		}

		t.Logf("Executed %d tasks in %v", taskCount, executionTime)

		// Verify all tasks were processed
		if len(result.TaskResults) != taskCount {
			t.Errorf("Expected %d task results, got %d", taskCount, len(result.TaskResults))
		}

		if result.Status != core.ExecutionStatusCompleted {
			t.Errorf("Expected completed status for large plan, got %s", result.Status)
		}
	})

	t.Run("HighConcurrencyStress", func(t *testing.T) {
		// Test concurrent access to the same components
		if testing.Short() {
			t.Skip("Skipping concurrency test in short mode")
		}

		mockReasoner := NewMockLLMReasoner()
		cc := core.NewContemplativeCursor(fixtures.Store, mockReasoner)

		objective := fixtures.SampleObjectives[0]
		concurrency := 10

		// Create multiple plans concurrently
		plans := make(chan *core.ExecutionPlan, concurrency)
		errors := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(index int) {
				plan, err := cc.CreateExecutionPlan(ctx, objective.ID)
				plans <- plan
				errors <- err
			}(i)
		}

		// Collect results
		successCount := 0
		errorCount := 0

		for i := 0; i < concurrency; i++ {
			plan := <-plans
			err := <-errors

			if err != nil {
				errorCount++
				t.Logf("Concurrent plan creation %d failed: %v", i, err)
			} else if plan != nil {
				successCount++
			}
		}

		t.Logf("Concurrent plan creation: %d successes, %d errors", successCount, errorCount)

		// Should handle concurrent access gracefully
		if successCount == 0 {
			t.Error("Expected at least some successful concurrent plan creations")
		}
	})
}

// TestRecoveryMetricsAndLogging tests that recovery actions are properly tracked.
func TestRecoveryMetricsAndLogging(t *testing.T) {
	fixtures := NewTestFixtures(t)
	ctx := context.Background()

	t.Run("FailureMetricsTracking", func(t *testing.T) {
		// Test that failure metrics are properly recorded
		mockExecutor := NewMockTaskExecutor()
		mockContextLoader := NewMockContextLoader()

		// Configure to have some failures
		mockExecutor.FailOnTaskType = "analysis"

		rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)

		plan := fixtures.CreateTestExecutionPlan("metrics-test-obj", "metrics-test-method")

		result, err := rtc.ExecutePlan(ctx, plan)
		if err != nil {
			t.Fatalf("Failed to execute plan for metrics test: %v", err)
		}

		// Verify failure metrics are captured
		if result.FailedTasks == 0 && mockExecutor.FailOnTaskType != "" {
			t.Error("Expected some failed tasks based on mock configuration")
		}

		// Check refinement data
		if result.MethodRefinementData == nil {
			t.Error("Expected method refinement data to be captured")
		}

		// Verify data contains useful metrics
		if successRate, exists := result.MethodRefinementData["success_rate"]; exists {
			if rate, ok := successRate.(float64); ok {
				t.Logf("Captured success rate: %.1f%%", rate)
			}
		}
	})

	t.Run("ExecutionHistoryRetrieval", func(t *testing.T) {
		// Test that execution history can be retrieved for analysis
		mockExecutor := NewMockTaskExecutor()
		mockContextLoader := NewMockContextLoader()

		rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)

		// Execute multiple plans to build history
		for i := 0; i < 3; i++ {
			plan := fixtures.CreateTestExecutionPlan(
				fmt.Sprintf("history-test-obj-%d", i),
				fmt.Sprintf("history-test-method-%d", i))

			_, err := rtc.ExecutePlan(ctx, plan)
			if err != nil {
				t.Fatalf("Failed to execute plan %d: %v", i, err)
			}
		}

		// Retrieve execution history
		history, err := rtc.GetExecutionHistory(ctx, 5)
		if err != nil {
			t.Fatalf("Failed to retrieve execution history: %v", err)
		}

		if len(history) == 0 {
			t.Error("Expected some execution history")
		}

		// Verify history contains expected data
		for _, result := range history {
			if result.PlanID == "" {
				t.Error("Expected plan ID in execution history")
			}
			if result.TotalDuration == 0 {
				t.Error("Expected execution duration to be recorded")
			}
		}

		t.Logf("Retrieved %d execution history records", len(history))
	})
}

// Helper functions for recovery tests

func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}()
}