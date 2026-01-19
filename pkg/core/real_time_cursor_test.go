package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Solifugus/ai-work-studio/pkg/storage"
)

// MockTaskExecutor implements TaskExecutor interface for testing.
type MockTaskExecutor struct {
	// Mock behaviors
	shouldFailExecution bool
	shouldFailEstimation bool
	shouldFailToolsList bool
	simulateTimeout     bool

	// Mock responses
	mockTaskResult     *TaskResult
	mockTokenEstimate  int
	mockAvailableTools []string

	// Recorded calls for verification
	executeTaskCalls     []ExecuteTaskCall
	getToolsCalls        []GetToolsCall
	estimateTokensCalls  []EstimateTokensCall
}

// Call recording structures
type ExecuteTaskCall struct {
	Task        *ExecutionTask
	FullContext map[string]interface{}
}

type GetToolsCall struct {
	CallTime time.Time
}

type EstimateTokensCall struct {
	Task *ExecutionTask
}

func NewMockTaskExecutor() *MockTaskExecutor {
	return &MockTaskExecutor{
		mockTaskResult: &TaskResult{
			Status:     TaskStatusCompleted,
			TokensUsed: 100,
			Duration:   5 * time.Second,
			Confidence: 0.9,
			ToolsUsed:  []string{"mock_tool"},
		},
		mockTokenEstimate:  100,
		mockAvailableTools: []string{"mock_tool", "another_tool"},
		executeTaskCalls:   make([]ExecuteTaskCall, 0),
		getToolsCalls:      make([]GetToolsCall, 0),
		estimateTokensCalls: make([]EstimateTokensCall, 0),
	}
}

func (m *MockTaskExecutor) ExecuteTask(ctx context.Context, task *ExecutionTask, fullContext map[string]interface{}) (*TaskResult, error) {
	m.executeTaskCalls = append(m.executeTaskCalls, ExecuteTaskCall{
		Task:        task,
		FullContext: fullContext,
	})

	if m.shouldFailExecution {
		return nil, fmt.Errorf("mock execution failure")
	}

	if m.simulateTimeout {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	// Return a copy of the mock result with task-specific data
	result := &TaskResult{
		TaskID:     task.ID,
		Status:     m.mockTaskResult.Status,
		TokensUsed: m.mockTaskResult.TokensUsed,
		Duration:   m.mockTaskResult.Duration,
		Confidence: m.mockTaskResult.Confidence,
		ToolsUsed:  make([]string, len(m.mockTaskResult.ToolsUsed)),
		CompletedAt: time.Now(),
	}
	copy(result.ToolsUsed, m.mockTaskResult.ToolsUsed)

	return result, nil
}

func (m *MockTaskExecutor) GetAvailableTools(ctx context.Context) ([]string, error) {
	m.getToolsCalls = append(m.getToolsCalls, GetToolsCall{
		CallTime: time.Now(),
	})

	if m.shouldFailToolsList {
		return nil, fmt.Errorf("mock tools list failure")
	}

	return m.mockAvailableTools, nil
}

func (m *MockTaskExecutor) EstimateTokenUsage(ctx context.Context, task *ExecutionTask) (int, error) {
	m.estimateTokensCalls = append(m.estimateTokensCalls, EstimateTokensCall{
		Task: task,
	})

	if m.shouldFailEstimation {
		return 0, fmt.Errorf("mock estimation failure")
	}

	return m.mockTokenEstimate, nil
}

// MockContextLoader implements ContextLoader interface for testing.
type MockContextLoader struct {
	// Mock behaviors
	shouldFailTaskContext      bool
	shouldFailObjectiveContext bool
	shouldFailResolveReference bool

	// Mock responses
	mockTaskContext      map[string]interface{}
	mockObjectiveContext map[string]interface{}
	mockResolvedReference interface{}

	// Recorded calls for verification
	loadTaskContextCalls      []LoadTaskContextCall
	loadObjectiveContextCalls []LoadObjectiveContextCall
	resolveReferenceCalls     []ResolveReferenceCall
}

// Call recording structures
type LoadTaskContextCall struct {
	Task *ExecutionTask
}

type LoadObjectiveContextCall struct {
	ObjectiveID string
}

type ResolveReferenceCall struct {
	Reference string
}

func NewMockContextLoader() *MockContextLoader {
	return &MockContextLoader{
		mockTaskContext: map[string]interface{}{
			"mock_data": "test_value",
			"context_loaded": true,
		},
		mockObjectiveContext: map[string]interface{}{
			"objective_data": "test_objective_value",
		},
		mockResolvedReference:     "resolved_data",
		loadTaskContextCalls:      make([]LoadTaskContextCall, 0),
		loadObjectiveContextCalls: make([]LoadObjectiveContextCall, 0),
		resolveReferenceCalls:     make([]ResolveReferenceCall, 0),
	}
}

func (m *MockContextLoader) LoadTaskContext(ctx context.Context, task *ExecutionTask) (map[string]interface{}, error) {
	m.loadTaskContextCalls = append(m.loadTaskContextCalls, LoadTaskContextCall{
		Task: task,
	})

	if m.shouldFailTaskContext {
		return nil, fmt.Errorf("mock task context loading failure")
	}

	return m.mockTaskContext, nil
}

func (m *MockContextLoader) LoadObjectiveContext(ctx context.Context, objectiveID string) (map[string]interface{}, error) {
	m.loadObjectiveContextCalls = append(m.loadObjectiveContextCalls, LoadObjectiveContextCall{
		ObjectiveID: objectiveID,
	})

	if m.shouldFailObjectiveContext {
		return nil, fmt.Errorf("mock objective context loading failure")
	}

	return m.mockObjectiveContext, nil
}

func (m *MockContextLoader) ResolveReference(ctx context.Context, ref string) (interface{}, error) {
	m.resolveReferenceCalls = append(m.resolveReferenceCalls, ResolveReferenceCall{
		Reference: ref,
	})

	if m.shouldFailResolveReference {
		return nil, fmt.Errorf("mock reference resolution failure")
	}

	return m.mockResolvedReference, nil
}

// Test helpers

func setupTestRTC(t *testing.T) (*RealTimeCursor, *storage.Store, *MockTaskExecutor, *MockContextLoader) {
	// Create temporary directory for test storage
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("rtc_test_%d", time.Now().UnixNano()))
	store, err := storage.NewStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}

	// Clean up after test
	t.Cleanup(func() {
		store.Close()
		os.RemoveAll(tempDir)
	})

	executor := NewMockTaskExecutor()
	contextLoader := NewMockContextLoader()
	rtc := NewRealTimeCursor(store, executor, contextLoader)

	return rtc, store, executor, contextLoader
}

func createTestPlan() *ExecutionPlan {
	return &ExecutionPlan{
		ID:          "test_plan_123",
		ObjectiveID: "test_objective_456",
		MethodID:    "test_method_789",
		Title:       "Test execution plan",
		Tasks: []ExecutionTask{
			{
				ID:          "task_1",
				Type:        "analyze",
				Description: "Analyze test data",
				Context: TaskContext{
					Parameters: map[string]interface{}{
						"test_param": "test_value",
					},
					TokenBudget: 100,
					Priority:    5,
				},
				EstimatedTokens: 100,
				CreatedAt:      time.Now(),
			},
			{
				ID:          "task_2",
				Type:        "generate",
				Description: "Generate test output",
				Context: TaskContext{
					Parameters: map[string]interface{}{
						"output_format": "json",
					},
					TokenBudget: 150,
					Priority:    6,
				},
				EstimatedTokens: 150,
				CreatedAt:      time.Now(),
			},
		},
		Dependencies: []TaskDependency{
			{
				TaskID:          "task_2",
				DependsOnTaskID: "task_1",
				Reason:          "Task 2 needs results from task 1",
			},
		},
		TotalEstimatedTokens: 250,
		CreatedBy:           "contemplative_cursor",
		CreatedAt:           time.Now(),
	}
}

// Tests

func TestNewRealTimeCursor(t *testing.T) {
	rtc, _, executor, contextLoader := setupTestRTC(t)

	if rtc.store == nil {
		t.Error("RTC store should not be nil")
	}
	if rtc.executor != executor {
		t.Error("RTC executor should match provided executor")
	}
	if rtc.contextLoader != contextLoader {
		t.Error("RTC contextLoader should match provided contextLoader")
	}
	if rtc.methodManager == nil {
		t.Error("RTC methodManager should not be nil")
	}
	if rtc.retryConfig == nil {
		t.Error("RTC retryConfig should not be nil")
	}
	if rtc.maxConcurrentTasks != 1 {
		t.Error("RTC maxConcurrentTasks should default to 1")
	}
}

func TestExecutePlan_Success(t *testing.T) {
	rtc, _, executor, contextLoader := setupTestRTC(t)
	plan := createTestPlan()

	result, err := rtc.ExecutePlan(context.Background(), plan)

	if err != nil {
		t.Fatalf("ExecutePlan should not fail: %v", err)
	}

	// Verify result structure
	if result.PlanID != plan.ID {
		t.Errorf("Expected PlanID %s, got %s", plan.ID, result.PlanID)
	}
	if result.ObjectiveID != plan.ObjectiveID {
		t.Errorf("Expected ObjectiveID %s, got %s", plan.ObjectiveID, result.ObjectiveID)
	}
	if result.Status != ExecutionStatusCompleted {
		t.Errorf("Expected status %s, got %s", ExecutionStatusCompleted, result.Status)
	}
	if result.SuccessfulTasks != 2 {
		t.Errorf("Expected 2 successful tasks, got %d", result.SuccessfulTasks)
	}
	if result.FailedTasks != 0 {
		t.Errorf("Expected 0 failed tasks, got %d", result.FailedTasks)
	}

	// Verify mocks were called correctly
	if len(executor.executeTaskCalls) != 2 {
		t.Errorf("Expected 2 executor calls, got %d", len(executor.executeTaskCalls))
	}
	if len(contextLoader.loadTaskContextCalls) != 2 {
		t.Errorf("Expected 2 context loader calls, got %d", len(contextLoader.loadTaskContextCalls))
	}

	// Verify task execution order (task_1 should execute before task_2 due to dependency)
	if len(executor.executeTaskCalls) >= 2 {
		firstCall := executor.executeTaskCalls[0]
		secondCall := executor.executeTaskCalls[1]
		if firstCall.Task.ID != "task_1" {
			t.Errorf("Expected first task to be task_1, got %s", firstCall.Task.ID)
		}
		if secondCall.Task.ID != "task_2" {
			t.Errorf("Expected second task to be task_2, got %s", secondCall.Task.ID)
		}
	}
}

func TestExecutePlan_TaskFailure(t *testing.T) {
	rtc, _, executor, _ := setupTestRTC(t)
	plan := createTestPlan()

	// Configure executor to fail
	executor.shouldFailExecution = true

	result, err := rtc.ExecutePlan(context.Background(), plan)

	if err == nil {
		t.Error("ExecutePlan should fail when tasks fail")
	}

	if result.Status == ExecutionStatusCompleted {
		t.Error("Execution status should not be completed when tasks fail")
	}
	if result.FailedTasks == 0 {
		t.Error("Failed tasks count should be greater than 0")
	}
}

func TestExecutePlan_InvalidPlan(t *testing.T) {
	rtc, _, _, _ := setupTestRTC(t)

	// Test with nil plan
	result, err := rtc.ExecutePlan(context.Background(), nil)
	if err == nil {
		t.Error("ExecutePlan should fail with nil plan")
	}
	if result == nil {
		t.Fatal("Result should not be nil even for invalid plan")
	}
	if result.Status != ExecutionStatusFailed {
		t.Error("Result status should be failed for invalid plan")
	}

	// Test with invalid plan (missing ID)
	invalidPlan := &ExecutionPlan{}
	result, err = rtc.ExecutePlan(context.Background(), invalidPlan)
	if err == nil {
		t.Error("ExecutePlan should fail with invalid plan")
	}
	if result == nil {
		t.Fatal("Result should not be nil even for invalid plan")
	}
	if result.Status != ExecutionStatusFailed {
		t.Error("Result status should be failed for invalid plan")
	}
}

func TestExecutePlan_CircularDependency(t *testing.T) {
	rtc, _, _, _ := setupTestRTC(t)

	// Create plan with circular dependency
	plan := &ExecutionPlan{
		ID:          "test_plan_circular",
		ObjectiveID: "test_objective",
		Tasks: []ExecutionTask{
			{ID: "task_a", Type: "test", Description: "Task A"},
			{ID: "task_b", Type: "test", Description: "Task B"},
		},
		Dependencies: []TaskDependency{
			{TaskID: "task_a", DependsOnTaskID: "task_b", Reason: "A depends on B"},
			{TaskID: "task_b", DependsOnTaskID: "task_a", Reason: "B depends on A"},
		},
	}

	result, err := rtc.ExecutePlan(context.Background(), plan)

	if err == nil {
		t.Error("ExecutePlan should fail with circular dependency")
	}
	if result.Status != ExecutionStatusFailed {
		t.Error("Result status should be failed for circular dependency")
	}
}

func TestExecutePlan_CancellationHandling(t *testing.T) {
	rtc, _, executor, _ := setupTestRTC(t)
	plan := createTestPlan()

	// Configure executor to simulate timeout
	executor.simulateTimeout = true

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	result, err := rtc.ExecutePlan(ctx, plan)

	if err == nil || err != context.DeadlineExceeded {
		t.Errorf("ExecutePlan should fail with context deadline exceeded, got: %v", err)
	}
	if result.Status != ExecutionStatusCancelled {
		t.Errorf("Expected status %s, got %s", ExecutionStatusCancelled, result.Status)
	}
}

func TestValidatePlan(t *testing.T) {
	rtc, _, _, _ := setupTestRTC(t)

	// Valid plan should pass
	validPlan := createTestPlan()
	err := rtc.validatePlan(validPlan)
	if err != nil {
		t.Errorf("Valid plan should not fail validation: %v", err)
	}

	// Invalid plans should fail
	testCases := []struct {
		name string
		plan *ExecutionPlan
	}{
		{"nil plan", nil},
		{"empty ID", &ExecutionPlan{ObjectiveID: "test", Tasks: []ExecutionTask{{ID: "task1", Type: "test"}}}},
		{"empty objective ID", &ExecutionPlan{ID: "test", Tasks: []ExecutionTask{{ID: "task1", Type: "test"}}}},
		{"no tasks", &ExecutionPlan{ID: "test", ObjectiveID: "test", Tasks: []ExecutionTask{}}},
		{"task without ID", &ExecutionPlan{ID: "test", ObjectiveID: "test", Tasks: []ExecutionTask{{Type: "test"}}}},
		{"task without type", &ExecutionPlan{ID: "test", ObjectiveID: "test", Tasks: []ExecutionTask{{ID: "task1"}}}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := rtc.validatePlan(tc.plan)
			if err == nil {
				t.Errorf("Plan validation should fail for %s", tc.name)
			}
		})
	}
}

func TestResolveDependencies(t *testing.T) {
	rtc, _, _, _ := setupTestRTC(t)

	// Test with simple dependency chain
	plan := createTestPlan()
	taskOrder, err := rtc.resolveDependencies(plan)

	if err != nil {
		t.Fatalf("Dependency resolution should not fail: %v", err)
	}

	if len(taskOrder) != 2 {
		t.Fatalf("Expected 2 tasks in order, got %d", len(taskOrder))
	}

	// task_1 should come before task_2
	if taskOrder[0].ID != "task_1" {
		t.Errorf("Expected first task to be task_1, got %s", taskOrder[0].ID)
	}
	if taskOrder[1].ID != "task_2" {
		t.Errorf("Expected second task to be task_2, got %s", taskOrder[1].ID)
	}
}

func TestRetryLogic(t *testing.T) {
	rtc, _, executor, _ := setupTestRTC(t)

	// Configure short retry delays for testing
	rtc.retryConfig.BaseDelay = 1 * time.Millisecond
	rtc.retryConfig.MaxDelay = 10 * time.Millisecond
	rtc.retryConfig.MaxRetries = 2
	// Add mock error to retriable errors for testing
	rtc.retryConfig.RetriableErrors = append(rtc.retryConfig.RetriableErrors, "mock execution failure")

	task := &ExecutionTask{
		ID:          "retry_test_task",
		Type:        "test",
		Description: "Test retry logic",
	}

	// First attempt should succeed after retries
	executor.shouldFailExecution = true

	startTime := time.Now()
	result, err := rtc.executeTaskWithRetries(context.Background(), task)
	duration := time.Since(startTime)

	// Should have failed after all retries
	if err == nil {
		t.Error("Task should fail after all retries are exhausted")
	}
	if result.Status != TaskStatusFailed {
		t.Errorf("Expected status %s, got %s", TaskStatusFailed, result.Status)
	}

	// Should have taken some time due to retries
	if duration < 1*time.Millisecond {
		t.Error("Should have waited for retries")
	}

	// Verify all retry attempts were made
	expectedCalls := rtc.retryConfig.MaxRetries + 1 // Initial attempt + retries
	if len(executor.executeTaskCalls) != expectedCalls {
		t.Errorf("Expected %d execution attempts, got %d", expectedCalls, len(executor.executeTaskCalls))
	}
}

func TestShouldRetry(t *testing.T) {
	rtc, _, _, _ := setupTestRTC(t)

	testCases := []struct {
		error    string
		attempt  int
		expected bool
	}{
		{"timeout error", 0, true},
		{"rate_limit exceeded", 1, true},
		{"network_error occurred", 2, true},
		{"permanent failure", 0, false},
		{"timeout error", 5, false}, // Max retries exceeded
	}

	for _, tc := range testCases {
		t.Run(tc.error, func(t *testing.T) {
			err := fmt.Errorf("%s", tc.error)
			result := rtc.shouldRetry(err, tc.attempt)
			if result != tc.expected {
				t.Errorf("shouldRetry(%v, %d) = %v, expected %v", err, tc.attempt, result, tc.expected)
			}
		})
	}
}

func TestCalculateExecutionRating(t *testing.T) {
	rtc, _, _, _ := setupTestRTC(t)

	// Test perfect execution
	perfectResult := &ExecutionResult{
		SuccessfulTasks: 5,
		FailedTasks:     0,
		TaskResults: map[string]*TaskResult{
			"task1": {Status: TaskStatusCompleted, Confidence: 0.9},
			"task2": {Status: TaskStatusCompleted, Confidence: 0.95},
		},
	}
	rating := rtc.calculateExecutionRating(perfectResult)
	if rating < 8.0 {
		t.Errorf("Perfect execution should get high rating, got %.2f", rating)
	}

	// Test failed execution
	failedResult := &ExecutionResult{
		SuccessfulTasks: 0,
		FailedTasks:     3,
		TaskResults:     map[string]*TaskResult{},
	}
	rating = rtc.calculateExecutionRating(failedResult)
	if rating > 3.0 {
		t.Errorf("Failed execution should get low rating, got %.2f", rating)
	}

	// Test mixed execution
	mixedResult := &ExecutionResult{
		SuccessfulTasks: 2,
		FailedTasks:     1,
		TaskResults: map[string]*TaskResult{
			"task1": {Status: TaskStatusCompleted, Confidence: 0.7},
		},
	}
	rating = rtc.calculateExecutionRating(mixedResult)
	if rating < 4.0 || rating > 7.0 {
		t.Errorf("Mixed execution should get moderate rating, got %.2f", rating)
	}
}

func TestRetryConfig(t *testing.T) {
	rtc, _, _, _ := setupTestRTC(t)

	// Test default config
	defaultConfig := rtc.GetRetryConfig()
	if defaultConfig.MaxRetries <= 0 {
		t.Error("Default max retries should be positive")
	}

	// Test custom config
	customConfig := &RetryConfig{
		MaxRetries:        5,
		BaseDelay:         2 * time.Second,
		MaxDelay:          1 * time.Minute,
		BackoffMultiplier: 1.5,
		RetriableErrors:   []string{"custom_error"},
	}

	rtc.SetRetryConfig(customConfig)
	retrievedConfig := rtc.GetRetryConfig()

	if retrievedConfig.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries 5, got %d", retrievedConfig.MaxRetries)
	}
	if retrievedConfig.BaseDelay != 2*time.Second {
		t.Errorf("Expected BaseDelay 2s, got %v", retrievedConfig.BaseDelay)
	}
	if len(retrievedConfig.RetriableErrors) != 1 {
		t.Errorf("Expected 1 retriable error, got %d", len(retrievedConfig.RetriableErrors))
	}
}

func TestTaskStatusHelpers(t *testing.T) {
	// Test IsTerminal
	terminalStatuses := []TaskStatus{TaskStatusCompleted, TaskStatusFailed}
	for _, status := range terminalStatuses {
		if !status.IsTerminal() {
			t.Errorf("Status %s should be terminal", status)
		}
	}

	nonTerminalStatuses := []TaskStatus{TaskStatusPending, TaskStatusRunning, TaskStatusRetrying, TaskStatusBlocked}
	for _, status := range nonTerminalStatuses {
		if status.IsTerminal() {
			t.Errorf("Status %s should not be terminal", status)
		}
	}

	// Test String
	status := TaskStatusCompleted
	if status.String() != "completed" {
		t.Errorf("String() should return status value, got %s", status.String())
	}
}

func TestExecutionStatusHelpers(t *testing.T) {
	// Test IsTerminal
	terminalStatuses := []ExecutionStatus{
		ExecutionStatusCompleted, ExecutionStatusFailed,
		ExecutionStatusPartial, ExecutionStatusCancelled,
	}
	for _, status := range terminalStatuses {
		if !status.IsTerminal() {
			t.Errorf("Status %s should be terminal", status)
		}
	}

	nonTerminalStatuses := []ExecutionStatus{ExecutionStatusPending, ExecutionStatusRunning}
	for _, status := range nonTerminalStatuses {
		if status.IsTerminal() {
			t.Errorf("Status %s should not be terminal", status)
		}
	}

	// Test String
	status := ExecutionStatusCompleted
	if status.String() != "completed" {
		t.Errorf("String() should return status value, got %s", status.String())
	}
}

// Benchmark tests for performance validation

func BenchmarkExecutePlan(b *testing.B) {
	rtc, _, _, _ := setupTestRTC(&testing.T{})
	plan := createTestPlan()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := rtc.ExecutePlan(context.Background(), plan)
		if err != nil {
			b.Fatalf("ExecutePlan failed: %v", err)
		}
	}
}

func BenchmarkResolveDependencies(b *testing.B) {
	rtc, _, _, _ := setupTestRTC(&testing.T{})
	plan := createTestPlan()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := rtc.resolveDependencies(plan)
		if err != nil {
			b.Fatalf("resolveDependencies failed: %v", err)
		}
	}
}