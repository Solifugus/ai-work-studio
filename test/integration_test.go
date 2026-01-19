package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/yourusername/ai-work-studio/pkg/core"
)

// TestFullObjectiveExecution tests the complete flow from goal creation to objective completion.
func TestFullObjectiveExecution(t *testing.T) {
	fixtures := NewTestFixtures(t)
	ctx := context.Background()

	// Test scenario: Execute a customer satisfaction analysis objective
	goal := fixtures.GetGoalByTitle("Improve Customer Satisfaction")
	if goal == nil {
		t.Fatal("Could not find test goal")
	}

	objective := fixtures.GetObjectiveByTitle("Analyze Customer Satisfaction Issues")
	if objective == nil {
		t.Fatal("Could not find test objective")
	}

	method := fixtures.GetMethodByName("Systematic Problem Analysis")
	if method == nil {
		t.Fatal("Could not find test method")
	}

	// Set up mocks
	mockReasoner := NewMockLLMReasoner()
	mockExecutor := NewMockTaskExecutor()
	mockContextLoader := NewMockContextLoader()

	// Create system components
	cc := core.NewContemplativeCursor(fixtures.Store, mockReasoner)
	rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)

	t.Run("CreateExecutionPlan", func(t *testing.T) {
		plan, err := cc.CreateExecutionPlan(ctx, objective.ID)
		if err != nil {
			t.Fatalf("Failed to create execution plan: %v", err)
		}

		// Verify plan structure
		if plan.ObjectiveID != objective.ID {
			t.Errorf("Expected plan objective ID %s, got %s", objective.ID, plan.ObjectiveID)
		}
		if len(plan.Tasks) == 0 {
			t.Error("Expected plan to contain tasks")
		}
		if plan.TotalEstimatedTokens == 0 {
			t.Error("Expected plan to have estimated token usage")
		}

		// Verify mock interactions
		if len(mockReasoner.AnalyzeCalls) != 1 {
			t.Errorf("Expected 1 analyze call, got %d", len(mockReasoner.AnalyzeCalls))
		}
		if len(mockReasoner.DecomposePlanCalls) != 1 {
			t.Errorf("Expected 1 decompose plan call, got %d", len(mockReasoner.DecomposePlanCalls))
		}
	})

	t.Run("ExecutePlan", func(t *testing.T) {
		plan, err := cc.CreateExecutionPlan(ctx, objective.ID)
		if err != nil {
			t.Fatalf("Failed to create execution plan: %v", err)
		}

		result, err := rtc.ExecutePlan(ctx, plan)
		if err != nil {
			t.Fatalf("Failed to execute plan: %v", err)
		}

		// Verify execution result
		if result.Status != core.ExecutionStatusCompleted {
			t.Errorf("Expected execution status %s, got %s", core.ExecutionStatusCompleted, result.Status)
		}
		if result.SuccessfulTasks == 0 {
			t.Error("Expected some successful tasks")
		}
		if result.TotalTokensUsed == 0 {
			t.Error("Expected some token usage")
		}

		// Verify all tasks were executed
		if len(result.TaskResults) != len(plan.Tasks) {
			t.Errorf("Expected %d task results, got %d", len(plan.Tasks), len(result.TaskResults))
		}

		// Verify mock interactions
		if len(mockExecutor.ExecuteCalls) != len(plan.Tasks) {
			t.Errorf("Expected %d execute calls, got %d", len(plan.Tasks), len(mockExecutor.ExecuteCalls))
		}
		if len(mockContextLoader.LoadTaskContextCalls) != len(plan.Tasks) {
			t.Errorf("Expected %d context load calls, got %d", len(plan.Tasks), len(mockContextLoader.LoadTaskContextCalls))
		}
	})
}

// TestGoalHierarchyIntegration tests working with hierarchical goal structures.
func TestGoalHierarchyIntegration(t *testing.T) {
	fixtures := NewTestFixtures(t)
	ctx := context.Background()

	parentGoal := fixtures.GetGoalByTitle("Improve Customer Satisfaction")
	childGoal := fixtures.GetGoalByTitle("Train Support Team")

	if parentGoal == nil || childGoal == nil {
		t.Fatal("Could not find required test goals")
	}

	t.Run("VerifyGoalRelationships", func(t *testing.T) {
		// Test getting sub-goals
		subGoals, err := fixtures.GoalManager.GetSubGoals(ctx, parentGoal.ID)
		if err != nil {
			t.Fatalf("Failed to get sub-goals: %v", err)
		}

		found := false
		for _, subGoal := range subGoals {
			if subGoal.ID == childGoal.ID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Child goal not found in parent's sub-goals")
		}

		// Test getting parent goals
		parentGoals, err := fixtures.GoalManager.GetParentGoals(ctx, childGoal.ID)
		if err != nil {
			t.Fatalf("Failed to get parent goals: %v", err)
		}

		found = false
		for _, parent := range parentGoals {
			if parent.ID == parentGoal.ID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Parent goal not found in child's parent goals")
		}
	})

	t.Run("ObjectivePropagation", func(t *testing.T) {
		// Test that objectives for child goals contribute to parent goal progress
		objectives, err := fixtures.ObjectiveManager.GetObjectivesForGoal(ctx, childGoal.ID)
		if err != nil {
			t.Fatalf("Failed to get objectives for child goal: %v", err)
		}

		if len(objectives) == 0 {
			t.Error("Expected objectives for child goal")
		}

		// Verify objective is linked to correct goal
		for _, obj := range objectives {
			if obj.GoalID != childGoal.ID {
				t.Errorf("Objective %s has wrong goal ID: expected %s, got %s", obj.ID, childGoal.ID, obj.GoalID)
			}
		}
	})
}

// TestMethodCacheIntegration tests method selection and caching behavior.
func TestMethodCacheIntegration(t *testing.T) {
	fixtures := NewTestFixtures(t)
	ctx := context.Background()

	// Simulate method usage to build cache statistics
	method1 := fixtures.GetMethodByName("Systematic Problem Analysis")
	method2 := fixtures.GetMethodByName("Database Performance Optimization")

	if method1 == nil || method2 == nil {
		t.Fatal("Could not find required test methods")
	}

	t.Run("MethodUsageTracking", func(t *testing.T) {
		// Simulate successful executions
		err := fixtures.SimulateMethodUsage(ctx, method1.ID, 10, 0.8) // 80% success rate
		if err != nil {
			t.Fatalf("Failed to simulate method usage: %v", err)
		}

		// Verify metrics were updated
		updatedMethod, err := fixtures.MethodManager.GetMethod(ctx, method1.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve updated method: %v", err)
		}

		if updatedMethod.Metrics.ExecutionCount != 10 {
			t.Errorf("Expected 10 executions, got %d", updatedMethod.Metrics.ExecutionCount)
		}
		if updatedMethod.Metrics.SuccessCount != 8 {
			t.Errorf("Expected 8 successes, got %d", updatedMethod.Metrics.SuccessCount)
		}

		successRate := updatedMethod.Metrics.SuccessRate()
		if successRate < 79.0 || successRate > 81.0 {
			t.Errorf("Expected ~80%% success rate, got %.1f%%", successRate)
		}
	})

	t.Run("MethodSelection", func(t *testing.T) {
		mockReasoner := NewMockLLMReasoner()
		cc := core.NewContemplativeCursor(fixtures.Store, mockReasoner)

		// Create an objective that could use either method
		objective := fixtures.GetObjectiveByTitle("Analyze Customer Satisfaction Issues")
		if objective == nil {
			t.Fatal("Could not find test objective")
		}

		plan, err := cc.CreateExecutionPlan(ctx, objective.ID)
		if err != nil {
			t.Fatalf("Failed to create execution plan: %v", err)
		}

		// Verify that a method was selected (could be cached or newly designed)
		if plan.MethodID == "" {
			t.Error("Expected plan to have a method ID")
		}
	})

	t.Run("CacheStatistics", func(t *testing.T) {
		mockReasoner := NewMockLLMReasoner()
		cc := core.NewContemplativeCursor(fixtures.Store, mockReasoner)

		stats, err := cc.GetMethodCacheStatistics(ctx)
		if err != nil {
			t.Fatalf("Failed to get cache statistics: %v", err)
		}

		// Verify basic statistics structure
		if totalMethods, ok := stats["total_methods"].(int); !ok || totalMethods == 0 {
			t.Error("Expected positive total methods count")
		}
		if activeMethods, ok := stats["active_methods"].(int); !ok || activeMethods == 0 {
			t.Error("Expected positive active methods count")
		}
	})
}

// TestTaskDependencyResolution tests complex task dependency handling.
func TestTaskDependencyResolution(t *testing.T) {
	fixtures := NewTestFixtures(t)
	ctx := context.Background()

	mockExecutor := NewMockTaskExecutor()
	mockContextLoader := NewMockContextLoader()
	rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)

	t.Run("SequentialDependencies", func(t *testing.T) {
		// Create a plan with sequential task dependencies
		plan := fixtures.CreateTestExecutionPlan("test-obj-1", "test-method-1")

		result, err := rtc.ExecutePlan(ctx, plan)
		if err != nil {
			t.Fatalf("Failed to execute plan with dependencies: %v", err)
		}

		if result.Status != core.ExecutionStatusCompleted {
			t.Errorf("Expected execution status %s, got %s", core.ExecutionStatusCompleted, result.Status)
		}

		// Verify tasks were executed in correct order
		expectedOrder := []string{"task-1", "task-2", "task-3", "task-4"}
		executionOrder := make([]string, len(mockExecutor.ExecuteCalls))
		for i, call := range mockExecutor.ExecuteCalls {
			executionOrder[i] = call.Task.ID
		}

		for i, expectedTaskID := range expectedOrder {
			if i >= len(executionOrder) || executionOrder[i] != expectedTaskID {
				t.Errorf("Task execution order mismatch at position %d: expected %s, got %s",
					i, expectedTaskID, executionOrder[i])
				break
			}
		}
	})

	t.Run("CircularDependencyDetection", func(t *testing.T) {
		// Create a plan with circular dependencies
		plan := fixtures.CreateTestExecutionPlan("test-obj-2", "test-method-2")

		// Add circular dependency
		plan.Dependencies = append(plan.Dependencies, core.TaskDependency{
			TaskID:          "task-1",
			DependsOnTaskID: "task-4",
			Reason:          "Creates circular dependency",
		})

		result, err := rtc.ExecutePlan(ctx, plan)

		// Should fail due to circular dependency
		if err == nil {
			t.Error("Expected error due to circular dependency")
		}
		if result.Status != core.ExecutionStatusFailed {
			t.Errorf("Expected status %s, got %s", core.ExecutionStatusFailed, result.Status)
		}
	})
}

// TestErrorHandlingAndRecovery tests system behavior under error conditions.
func TestErrorHandlingAndRecovery(t *testing.T) {
	fixtures := NewTestFixtures(t)
	ctx := context.Background()

	t.Run("TaskExecutionFailure", func(t *testing.T) {
		// Set up mock to fail on specific task type
		mockExecutor := NewMockTaskExecutor()
		mockExecutor.FailOnTaskType = "analysis"
		mockContextLoader := NewMockContextLoader()

		rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)
		plan := fixtures.CreateTestExecutionPlan("test-obj-fail", "test-method-fail")

		result, err := rtc.ExecutePlan(ctx, plan)

		// Execution should complete but with failures
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if result.FailedTasks == 0 {
			t.Error("Expected some failed tasks")
		}

		// Check that non-critical failures allow execution to continue
		if result.Status == core.ExecutionStatusFailed && result.SuccessfulTasks > 0 {
			// This is partial success, which is acceptable for non-critical task failures
			if result.Status != core.ExecutionStatusPartial {
				t.Errorf("Expected partial status for mixed success/failure, got %s", result.Status)
			}
		}
	})

	t.Run("ContextLoadFailure", func(t *testing.T) {
		// Set up mock to fail context loading
		mockExecutor := NewMockTaskExecutor()
		mockContextLoader := NewMockContextLoader()
		mockContextLoader.ShouldFailContextLoad = true

		rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)
		plan := fixtures.CreateTestExecutionPlan("test-obj-context-fail", "test-method-context-fail")

		result, err := rtc.ExecutePlan(ctx, plan)

		// Should handle context load failures gracefully
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if result.Status == core.ExecutionStatusCompleted {
			t.Error("Did not expect successful completion with context load failures")
		}
	})

	t.Run("LLMReasonerFailure", func(t *testing.T) {
		// Set up mock to fail analysis
		mockReasoner := NewMockLLMReasoner()
		mockReasoner.ShouldFailAnalysis = true

		cc := core.NewContemplativeCursor(fixtures.Store, mockReasoner)
		objective := fixtures.GetObjectiveByTitle("Analyze Customer Satisfaction Issues")

		_, err := cc.CreateExecutionPlan(ctx, objective.ID)

		// Should fail plan creation when analysis fails
		if err == nil {
			t.Error("Expected error when LLM reasoner fails")
		}
	})
}

// TestConcurrentObjectiveExecution tests system behavior with multiple concurrent objectives.
func TestConcurrentObjectiveExecution(t *testing.T) {
	fixtures := NewTestFixtures(t)
	ctx := context.Background()

	t.Run("MultipleObjectivesSequentially", func(t *testing.T) {
		// Execute multiple objectives one after another
		objectives := []*core.Objective{
			fixtures.GetObjectiveByTitle("Analyze Customer Satisfaction Issues"),
			fixtures.GetObjectiveByTitle("Optimize User API Database Queries"),
			fixtures.GetObjectiveByTitle("Deliver Product Feature Training"),
		}

		mockReasoner := NewMockLLMReasoner()
		mockExecutor := NewMockTaskExecutor()
		mockContextLoader := NewMockContextLoader()

		cc := core.NewContemplativeCursor(fixtures.Store, mockReasoner)
		rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)

		var results []*core.ExecutionResult

		for _, obj := range objectives {
			if obj == nil {
				continue
			}

			plan, err := cc.CreateExecutionPlan(ctx, obj.ID)
			if err != nil {
				t.Fatalf("Failed to create plan for objective %s: %v", obj.Title, err)
			}

			result, err := rtc.ExecutePlan(ctx, plan)
			if err != nil {
				t.Fatalf("Failed to execute plan for objective %s: %v", obj.Title, err)
			}

			results = append(results, result)
		}

		// Verify all objectives were executed successfully
		if len(results) == 0 {
			t.Fatal("No objectives were executed")
		}

		for i, result := range results {
			if result.Status != core.ExecutionStatusCompleted {
				t.Errorf("Objective %d failed with status: %s", i, result.Status)
			}
		}

		// Verify method metrics were updated for each execution
		for _, obj := range objectives {
			if obj == nil {
				continue
			}
			method, err := fixtures.MethodManager.GetMethod(ctx, obj.MethodID)
			if err != nil {
				t.Fatalf("Failed to retrieve method %s: %v", obj.MethodID, err)
			}
			if method.Metrics.ExecutionCount == 0 {
				t.Errorf("Expected method %s to have execution count > 0", method.Name)
			}
		}
	})
}

// TestStorageConsistency tests that storage operations maintain consistency.
func TestStorageConsistency(t *testing.T) {
	fixtures := NewTestFixtures(t)
	ctx := context.Background()

	t.Run("NodeTemporalConsistency", func(t *testing.T) {
		// Test that temporal storage maintains version history
		goal := fixtures.SampleGoals[0]
		originalTitle := goal.Title

		// Update the goal
		newTitle := "Updated " + originalTitle
		updates := core.GoalUpdates{
			Title: &newTitle,
		}

		updatedGoal, err := fixtures.GoalManager.UpdateGoal(ctx, goal.ID, updates)
		if err != nil {
			t.Fatalf("Failed to update goal: %v", err)
		}

		// Verify current version has new title
		if updatedGoal.Title != newTitle {
			t.Errorf("Expected updated title %s, got %s", newTitle, updatedGoal.Title)
		}

		// Verify we can retrieve historical version
		historicalGoal, err := fixtures.GoalManager.GetGoalAtTime(ctx, goal.ID, goal.CreatedAt.Add(1*time.Minute))
		if err != nil {
			t.Fatalf("Failed to retrieve historical goal: %v", err)
		}

		if historicalGoal.Title != originalTitle {
			t.Errorf("Expected historical title %s, got %s", originalTitle, historicalGoal.Title)
		}
	})

	t.Run("EdgeConsistency", func(t *testing.T) {
		// Test that relationships are maintained correctly
		parentGoal := fixtures.GetGoalByTitle("Improve Customer Satisfaction")
		childGoal := fixtures.GetGoalByTitle("Train Support Team")

		if parentGoal == nil || childGoal == nil {
			t.Fatal("Could not find required goals for edge consistency test")
		}

		// Verify relationship exists
		subGoals, err := fixtures.GoalManager.GetSubGoals(ctx, parentGoal.ID)
		if err != nil {
			t.Fatalf("Failed to get sub-goals: %v", err)
		}

		found := false
		for _, subGoal := range subGoals {
			if subGoal.ID == childGoal.ID {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected child goal relationship to be maintained")
		}
	})

	t.Run("CrossEntityConsistency", func(t *testing.T) {
		// Test that references between entities are maintained
		objective := fixtures.SampleObjectives[0]

		// Verify objective references valid goal and method
		_, err := fixtures.GoalManager.GetGoal(ctx, objective.GoalID)
		if err != nil {
			t.Errorf("Objective references invalid goal: %v", err)
		}

		_, err = fixtures.MethodManager.GetMethod(ctx, objective.MethodID)
		if err != nil {
			t.Errorf("Objective references invalid method: %v", err)
		}

		// Verify reverse relationships
		objectivesForGoal, err := fixtures.ObjectiveManager.GetObjectivesForGoal(ctx, objective.GoalID)
		if err != nil {
			t.Fatalf("Failed to get objectives for goal: %v", err)
		}

		found := false
		for _, obj := range objectivesForGoal {
			if obj.ID == objective.ID {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected objective to be found in goal's objectives")
		}
	})
}

// TestSystemScalability tests system performance with larger data sets.
func TestSystemScalability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scalability test in short mode")
	}

	fixtures := NewTestFixtures(t)
	ctx := context.Background()

	t.Run("LargeNumberOfGoals", func(t *testing.T) {
		// Create 100 goals to test query performance
		startTime := time.Now()
		goalCount := 100

		for i := 0; i < goalCount; i++ {
			_, err := fixtures.GoalManager.CreateGoal(ctx,
				fmt.Sprintf("Test Goal %d", i),
				fmt.Sprintf("Description for test goal %d with some detail to make it realistic", i),
				(i%10)+1, // Priority 1-10
				map[string]interface{}{
					"index": i,
					"batch": "scalability_test",
				})
			if err != nil {
				t.Fatalf("Failed to create goal %d: %v", i, err)
			}
		}

		creationTime := time.Since(startTime)
		t.Logf("Created %d goals in %v", goalCount, creationTime)

		// Test query performance
		startTime = time.Now()
		allGoals, err := fixtures.GoalManager.ListGoals(ctx, core.GoalFilter{})
		if err != nil {
			t.Fatalf("Failed to list goals: %v", err)
		}

		queryTime := time.Since(startTime)
		t.Logf("Queried %d goals in %v", len(allGoals), queryTime)

		// Verify we got the expected number of goals (original + created)
		expectedCount := len(fixtures.SampleGoals) + goalCount
		if len(allGoals) < expectedCount {
			t.Errorf("Expected at least %d goals, got %d", expectedCount, len(allGoals))
		}
	})

	t.Run("MethodCachePerformance", func(t *testing.T) {
		// Test method cache performance with many methods
		startTime := time.Now()
		methodCount := 50

		for i := 0; i < methodCount; i++ {
			_, err := fixtures.MethodManager.CreateMethod(ctx,
				fmt.Sprintf("Scalability Test Method %d", i),
				fmt.Sprintf("Test method %d for scalability testing", i),
				[]core.ApproachStep{
					{
						Description: fmt.Sprintf("Step 1 for method %d", i),
						Tools:       []string{"tool1", "tool2"},
					},
					{
						Description: fmt.Sprintf("Step 2 for method %d", i),
						Tools:       []string{"tool3", "tool4"},
					},
				},
				core.MethodDomainGeneral,
				map[string]interface{}{
					"test_index": i,
				})
			if err != nil {
				t.Fatalf("Failed to create method %d: %v", i, err)
			}
		}

		creationTime := time.Since(startTime)
		t.Logf("Created %d methods in %v", methodCount, creationTime)

		// Test query performance
		startTime = time.Now()
		allMethods, err := fixtures.MethodManager.ListMethods(ctx, core.MethodFilter{})
		if err != nil {
			t.Fatalf("Failed to list methods: %v", err)
		}

		queryTime := time.Since(startTime)
		t.Logf("Queried %d methods in %v", len(allMethods), queryTime)
	})
}