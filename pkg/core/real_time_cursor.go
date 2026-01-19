package core

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/ai-work-studio/pkg/storage"
)

// TaskExecutor defines the interface for executing individual tasks.
// This abstracts the LLM and MCP tool usage to allow for testing and flexibility.
type TaskExecutor interface {
	// ExecuteTask runs a single task using available tools via MCP
	ExecuteTask(ctx context.Context, task *ExecutionTask, fullContext map[string]interface{}) (*TaskResult, error)

	// GetAvailableTools returns the list of MCP tools available for execution
	GetAvailableTools(ctx context.Context) ([]string, error)

	// EstimateTokenUsage provides an estimate of tokens that will be used for a task
	EstimateTokenUsage(ctx context.Context, task *ExecutionTask) (int, error)
}

// ContextLoader defines the interface for loading full context when needed.
// Follows the minimal context design principle.
type ContextLoader interface {
	// LoadTaskContext loads the full context for a task based on its references
	LoadTaskContext(ctx context.Context, task *ExecutionTask) (map[string]interface{}, error)

	// LoadObjectiveContext loads context for the entire objective
	LoadObjectiveContext(ctx context.Context, objectiveID string) (map[string]interface{}, error)

	// ResolveReference resolves a data reference to its actual content
	ResolveReference(ctx context.Context, ref string) (interface{}, error)
}

// TaskResult represents the outcome of executing a single task.
type TaskResult struct {
	// TaskID identifies which task this result belongs to
	TaskID string

	// Status indicates whether the task completed successfully
	Status TaskStatus

	// Output contains the task's output data (may be large, stored by reference)
	Output interface{}

	// OutputRef is a reference to where the output is stored
	OutputRef string

	// TokensUsed tracks actual LLM token consumption
	TokensUsed int

	// Duration is how long the task took to execute
	Duration time.Duration

	// ErrorMessage contains error details if the task failed
	ErrorMessage string

	// ToolsUsed lists the MCP tools that were invoked
	ToolsUsed []string

	// Confidence indicates how confident the system is in this result (0.0-1.0)
	Confidence float64

	// CompletedAt is when the task finished execution
	CompletedAt time.Time
}

// TaskStatus represents the execution status of a task.
type TaskStatus string

const (
	// TaskStatusPending indicates the task has not started
	TaskStatusPending TaskStatus = "pending"

	// TaskStatusRunning indicates the task is currently executing
	TaskStatusRunning TaskStatus = "running"

	// TaskStatusCompleted indicates the task finished successfully
	TaskStatusCompleted TaskStatus = "completed"

	// TaskStatusFailed indicates the task failed and cannot be retried
	TaskStatusFailed TaskStatus = "failed"

	// TaskStatusRetrying indicates the task failed but will be retried
	TaskStatusRetrying TaskStatus = "retrying"

	// TaskStatusBlocked indicates the task cannot proceed due to dependency issues
	TaskStatusBlocked TaskStatus = "blocked"
)

// ExecutionResult represents the overall result of executing an entire plan.
type ExecutionResult struct {
	// PlanID identifies the execution plan that was run
	PlanID string

	// ObjectiveID identifies the objective this execution served
	ObjectiveID string

	// Status indicates the overall outcome of the execution
	Status ExecutionStatus

	// TaskResults contains results for each task in the plan
	TaskResults map[string]*TaskResult

	// TotalTokensUsed is the sum of tokens used across all tasks
	TotalTokensUsed int

	// TotalDuration is the total time spent executing the plan
	TotalDuration time.Duration

	// StartTime is when execution began
	StartTime time.Time

	// EndTime is when execution completed (or failed)
	EndTime time.Time

	// ErrorMessage contains error details if execution failed
	ErrorMessage string

	// SuccessfulTasks counts how many tasks completed successfully
	SuccessfulTasks int

	// FailedTasks counts how many tasks failed permanently
	FailedTasks int

	// MethodRefinementData contains feedback for improving the method
	MethodRefinementData map[string]interface{}
}

// ExecutionStatus represents the overall execution status of a plan.
type ExecutionStatus string

const (
	// ExecutionStatusPending indicates execution has not started
	ExecutionStatusPending ExecutionStatus = "pending"

	// ExecutionStatusRunning indicates execution is in progress
	ExecutionStatusRunning ExecutionStatus = "running"

	// ExecutionStatusCompleted indicates execution finished successfully
	ExecutionStatusCompleted ExecutionStatus = "completed"

	// ExecutionStatusFailed indicates execution failed and cannot continue
	ExecutionStatusFailed ExecutionStatus = "failed"

	// ExecutionStatusPartial indicates execution completed with some task failures
	ExecutionStatusPartial ExecutionStatus = "partial"

	// ExecutionStatusCancelled indicates execution was cancelled by user
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)

// RetryConfig defines configuration for task retry behavior.
type RetryConfig struct {
	// MaxRetries is the maximum number of times to retry a failed task
	MaxRetries int

	// BaseDelay is the initial delay between retries
	BaseDelay time.Duration

	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration

	// BackoffMultiplier controls exponential backoff growth
	BackoffMultiplier float64

	// RetriableErrors lists error types that should trigger retries
	RetriableErrors []string
}

// DefaultRetryConfig provides sensible defaults for retry configuration.
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:        3,
		BaseDelay:         1 * time.Second,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
		RetriableErrors: []string{
			"timeout",
			"rate_limit",
			"temporary_unavailable",
			"network_error",
		},
	}
}

// RealTimeCursor represents the tactical execution component of the agent system.
// It takes execution plans from CC and executes tasks sequentially using available tools.
type RealTimeCursor struct {
	// store provides access to the temporal storage system
	store *storage.Store

	// executor handles task execution via MCP tools and LLM
	executor TaskExecutor

	// contextLoader handles loading full context when needed
	contextLoader ContextLoader

	// methodManager provides access to method operations for feedback
	methodManager *MethodManager

	// retryConfig defines retry behavior for failed tasks
	retryConfig *RetryConfig

	// maxConcurrentTasks limits parallel task execution (future enhancement)
	maxConcurrentTasks int
}

// NewRealTimeCursor creates a new RTC instance with the given dependencies.
func NewRealTimeCursor(store *storage.Store, executor TaskExecutor, contextLoader ContextLoader) *RealTimeCursor {
	return &RealTimeCursor{
		store:              store,
		executor:           executor,
		contextLoader:      contextLoader,
		methodManager:      NewMethodManager(store),
		retryConfig:        DefaultRetryConfig(),
		maxConcurrentTasks: 1, // Sequential execution for now
	}
}

// ExecutePlan runs the given execution plan and returns the overall result.
// This is the main entry point for RTC execution capabilities.
func (rtc *RealTimeCursor) ExecutePlan(ctx context.Context, plan *ExecutionPlan) (*ExecutionResult, error) {
	startTime := time.Now()

	// Validate the plan before creating result to avoid nil pointer access
	if err := rtc.validatePlan(plan); err != nil {
		result := &ExecutionResult{
			Status:               ExecutionStatusFailed,
			ErrorMessage:         fmt.Sprintf("Plan validation failed: %v", err),
			StartTime:            startTime,
			EndTime:              time.Now(),
			TotalDuration:        time.Since(startTime),
			TaskResults:          make(map[string]*TaskResult),
			MethodRefinementData: make(map[string]interface{}),
		}
		// Set plan-specific fields only if plan is not nil
		if plan != nil {
			result.PlanID = plan.ID
			result.ObjectiveID = plan.ObjectiveID
		}
		return result, fmt.Errorf("plan validation failed: %w", err)
	}

	result := &ExecutionResult{
		PlanID:               plan.ID,
		ObjectiveID:          plan.ObjectiveID,
		Status:               ExecutionStatusRunning,
		TaskResults:          make(map[string]*TaskResult),
		StartTime:            startTime,
		MethodRefinementData: make(map[string]interface{}),
	}

	// Store the execution result for tracking
	if err := rtc.storeExecutionResult(ctx, result); err != nil {
		// Log warning but continue - execution tracking shouldn't fail the execution
		fmt.Printf("Warning: failed to store initial execution result: %v\n", err)
	}

	// Execute tasks in dependency order
	taskOrder, err := rtc.resolveDependencies(plan)
	if err != nil {
		result.Status = ExecutionStatusFailed
		result.ErrorMessage = fmt.Sprintf("Dependency resolution failed: %v", err)
		result.EndTime = time.Now()
		result.TotalDuration = time.Since(startTime)
		return result, fmt.Errorf("dependency resolution failed: %w", err)
	}

	// Execute each task in order
	for _, task := range taskOrder {
		select {
		case <-ctx.Done():
			result.Status = ExecutionStatusCancelled
			result.ErrorMessage = "Execution cancelled"
			result.EndTime = time.Now()
			result.TotalDuration = time.Since(startTime)
			return result, ctx.Err()
		default:
			// Execute the task
			taskResult, err := rtc.executeTaskWithRetries(ctx, task)
			result.TaskResults[task.ID] = taskResult

			// Update counters
			if taskResult.Status == TaskStatusCompleted {
				result.SuccessfulTasks++
			} else if taskResult.Status == TaskStatusFailed {
				result.FailedTasks++
			}

			// Update token usage
			result.TotalTokensUsed += taskResult.TokensUsed

			// Handle task failure
			if err != nil {
				// Check if this is a cancellation error
				if err == context.Canceled || err == context.DeadlineExceeded {
					result.Status = ExecutionStatusCancelled
					result.ErrorMessage = "Execution cancelled"
					result.EndTime = time.Now()
					result.TotalDuration = time.Since(startTime)
					rtc.storeExecutionResult(ctx, result)
					return result, err
				}

				// Check if this is a critical task that should fail the entire plan
				if rtc.isCriticalTask(task, plan) {
					result.Status = ExecutionStatusFailed
					result.ErrorMessage = fmt.Sprintf("Critical task %s failed: %v", task.ID, err)
					result.EndTime = time.Now()
					result.TotalDuration = time.Since(startTime)

					// Still record refinement data even on failure
					rtc.collectRefinementData(result, plan)

					// Update stored result
					rtc.storeExecutionResult(ctx, result)
					return result, fmt.Errorf("execution failed on critical task: %w", err)
				}

				// Non-critical task failure - log and continue
				fmt.Printf("Warning: non-critical task %s failed: %v\n", task.ID, err)
			}
		}
	}

	// Determine final status
	if result.FailedTasks == 0 {
		result.Status = ExecutionStatusCompleted
	} else if result.SuccessfulTasks > 0 {
		result.Status = ExecutionStatusPartial
	} else {
		result.Status = ExecutionStatusFailed
		result.ErrorMessage = "All tasks failed"
	}

	// Finalize result
	result.EndTime = time.Now()
	result.TotalDuration = time.Since(startTime)

	// Collect method refinement data
	rtc.collectRefinementData(result, plan)

	// Update method metrics based on execution outcome
	if err := rtc.updateMethodMetrics(ctx, plan, result); err != nil {
		fmt.Printf("Warning: failed to update method metrics: %v\n", err)
	}

	// Store final result
	if err := rtc.storeExecutionResult(ctx, result); err != nil {
		fmt.Printf("Warning: failed to store final execution result: %v\n", err)
	}

	return result, nil
}

// executeTaskWithRetries executes a single task with retry logic.
func (rtc *RealTimeCursor) executeTaskWithRetries(ctx context.Context, task *ExecutionTask) (*TaskResult, error) {
	result := &TaskResult{
		TaskID:      task.ID,
		Status:      TaskStatusPending,
		CompletedAt: time.Time{},
	}

	var lastError error
	for attempt := 0; attempt <= rtc.retryConfig.MaxRetries; attempt++ {
		// Check for context cancellation before each attempt
		select {
		case <-ctx.Done():
			result.Status = TaskStatusFailed
			result.ErrorMessage = "Task cancelled"
			result.CompletedAt = time.Now()
			return result, ctx.Err()
		default:
		}

		// Update status for tracking
		if attempt == 0 {
			result.Status = TaskStatusRunning
		} else {
			result.Status = TaskStatusRetrying
		}

		// Load context for this task
		fullContext, err := rtc.contextLoader.LoadTaskContext(ctx, task)
		if err != nil {
			lastError = fmt.Errorf("failed to load task context: %w", err)
			if err == context.Canceled || err == context.DeadlineExceeded || !rtc.shouldRetry(lastError, attempt) {
				break
			}
			rtc.waitForRetryWithContext(ctx, attempt)
			continue
		}

		// Execute the task
		startTime := time.Now()
		taskResult, err := rtc.executor.ExecuteTask(ctx, task, fullContext)
		duration := time.Since(startTime)

		if err != nil {
			lastError = err
			if err == context.Canceled || err == context.DeadlineExceeded || !rtc.shouldRetry(err, attempt) {
				break
			}
			rtc.waitForRetryWithContext(ctx, attempt)
			continue
		}

		// Success - update result with execution data
		result.Status = TaskStatusCompleted
		result.Output = taskResult.Output
		result.OutputRef = taskResult.OutputRef
		result.TokensUsed = taskResult.TokensUsed
		result.Duration = duration
		result.ToolsUsed = taskResult.ToolsUsed
		result.Confidence = taskResult.Confidence
		result.CompletedAt = time.Now()

		return result, nil
	}

	// All retries exhausted - mark as failed
	result.Status = TaskStatusFailed
	result.ErrorMessage = lastError.Error()
	result.CompletedAt = time.Now()

	return result, lastError
}

// validatePlan performs basic validation on the execution plan.
func (rtc *RealTimeCursor) validatePlan(plan *ExecutionPlan) error {
	if plan == nil {
		return fmt.Errorf("plan cannot be nil")
	}

	if plan.ID == "" {
		return fmt.Errorf("plan must have an ID")
	}

	if plan.ObjectiveID == "" {
		return fmt.Errorf("plan must reference an objective")
	}

	if len(plan.Tasks) == 0 {
		return fmt.Errorf("plan must contain at least one task")
	}

	// Validate each task
	for _, task := range plan.Tasks {
		if task.ID == "" {
			return fmt.Errorf("task must have an ID")
		}
		if task.Type == "" {
			return fmt.Errorf("task %s must have a type", task.ID)
		}
	}

	// Validate dependencies reference existing tasks
	taskIDs := make(map[string]bool)
	for _, task := range plan.Tasks {
		taskIDs[task.ID] = true
	}

	for _, dep := range plan.Dependencies {
		if !taskIDs[dep.TaskID] {
			return fmt.Errorf("dependency references unknown task: %s", dep.TaskID)
		}
		if !taskIDs[dep.DependsOnTaskID] {
			return fmt.Errorf("dependency references unknown prerequisite task: %s", dep.DependsOnTaskID)
		}
	}

	return nil
}

// resolveDependencies determines the correct execution order for tasks based on dependencies.
func (rtc *RealTimeCursor) resolveDependencies(plan *ExecutionPlan) ([]*ExecutionTask, error) {
	// Build dependency graph
	dependencies := make(map[string][]string) // map[taskID]prerequisiteTaskIDs
	for _, dep := range plan.Dependencies {
		dependencies[dep.TaskID] = append(dependencies[dep.TaskID], dep.DependsOnTaskID)
	}

	// Topological sort to determine execution order
	var result []*ExecutionTask
	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	taskMap := make(map[string]*ExecutionTask)

	// Create task lookup map
	for i := range plan.Tasks {
		taskMap[plan.Tasks[i].ID] = &plan.Tasks[i]
	}

	var visit func(taskID string) error
	visit = func(taskID string) error {
		if visiting[taskID] {
			return fmt.Errorf("circular dependency detected involving task: %s", taskID)
		}
		if visited[taskID] {
			return nil
		}

		visiting[taskID] = true

		// Visit all prerequisites first
		for _, prereqID := range dependencies[taskID] {
			if err := visit(prereqID); err != nil {
				return err
			}
		}

		visiting[taskID] = false
		visited[taskID] = true

		// Add task to result
		if task := taskMap[taskID]; task != nil {
			result = append(result, task)
		}

		return nil
	}

	// Visit all tasks
	for _, task := range plan.Tasks {
		if err := visit(task.ID); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// shouldRetry determines if a task execution error should trigger a retry.
func (rtc *RealTimeCursor) shouldRetry(err error, attempt int) bool {
	if attempt >= rtc.retryConfig.MaxRetries {
		return false
	}

	errorMsg := err.Error()
	for _, retriableError := range rtc.retryConfig.RetriableErrors {
		if len(errorMsg) >= len(retriableError) {
			for i := 0; i <= len(errorMsg)-len(retriableError); i++ {
				if errorMsg[i:i+len(retriableError)] == retriableError {
					return true
				}
			}
		}
	}

	return false
}

// waitForRetry implements exponential backoff delay between retries.
func (rtc *RealTimeCursor) waitForRetry(attempt int) {
	if attempt <= 0 {
		return
	}

	delay := float64(rtc.retryConfig.BaseDelay)
	for i := 1; i < attempt; i++ {
		delay *= rtc.retryConfig.BackoffMultiplier
	}

	finalDelay := time.Duration(delay)
	if finalDelay > rtc.retryConfig.MaxDelay {
		finalDelay = rtc.retryConfig.MaxDelay
	}

	time.Sleep(finalDelay)
}

// waitForRetryWithContext implements exponential backoff delay between retries that can be cancelled.
func (rtc *RealTimeCursor) waitForRetryWithContext(ctx context.Context, attempt int) {
	if attempt <= 0 {
		return
	}

	delay := float64(rtc.retryConfig.BaseDelay)
	for i := 1; i < attempt; i++ {
		delay *= rtc.retryConfig.BackoffMultiplier
	}

	finalDelay := time.Duration(delay)
	if finalDelay > rtc.retryConfig.MaxDelay {
		finalDelay = rtc.retryConfig.MaxDelay
	}

	timer := time.NewTimer(finalDelay)
	defer timer.Stop()

	select {
	case <-timer.C:
		// Normal delay completed
	case <-ctx.Done():
		// Context cancelled, return immediately
	}
}

// isCriticalTask determines if a task failure should stop the entire plan execution.
func (rtc *RealTimeCursor) isCriticalTask(task *ExecutionTask, plan *ExecutionPlan) bool {
	// For now, consider high-priority tasks or tasks that many other tasks depend on as critical
	if task.Context.Priority >= 8 {
		return true
	}

	// Count how many other tasks depend on this one
	dependentCount := 0
	for _, dep := range plan.Dependencies {
		if dep.DependsOnTaskID == task.ID {
			dependentCount++
		}
	}

	// If more than half the remaining tasks depend on this one, it's critical
	return dependentCount > (len(plan.Tasks)-1)/2
}

// collectRefinementData gathers feedback for improving the method used in this execution.
func (rtc *RealTimeCursor) collectRefinementData(result *ExecutionResult, plan *ExecutionPlan) {
	refinement := make(map[string]interface{})

	// Calculate success metrics
	totalTasks := len(plan.Tasks)
	successRate := float64(result.SuccessfulTasks) / float64(totalTasks) * 100

	refinement["success_rate"] = successRate
	refinement["total_tasks"] = totalTasks
	refinement["successful_tasks"] = result.SuccessfulTasks
	refinement["failed_tasks"] = result.FailedTasks
	refinement["total_duration"] = result.TotalDuration.Seconds()
	refinement["total_tokens"] = result.TotalTokensUsed

	// Analyze token efficiency
	if plan.TotalEstimatedTokens > 0 {
		tokenAccuracy := float64(result.TotalTokensUsed) / float64(plan.TotalEstimatedTokens)
		refinement["token_estimation_accuracy"] = tokenAccuracy
	}

	// Identify problematic task types
	failedTaskTypes := make(map[string]int)
	averageTaskDuration := make(map[string]time.Duration)
	taskTypeCounts := make(map[string]int)

	for taskID, taskResult := range result.TaskResults {
		// Find the task definition to get its type
		for _, task := range plan.Tasks {
			if task.ID == taskID {
				taskTypeCounts[task.Type]++
				averageTaskDuration[task.Type] += taskResult.Duration

				if taskResult.Status == TaskStatusFailed {
					failedTaskTypes[task.Type]++
				}
				break
			}
		}
	}

	// Calculate averages
	for taskType, totalDuration := range averageTaskDuration {
		count := taskTypeCounts[taskType]
		if count > 0 {
			averageTaskDuration[taskType] = totalDuration / time.Duration(count)
		}
	}

	refinement["failed_task_types"] = failedTaskTypes
	refinement["average_task_duration"] = averageTaskDuration
	refinement["execution_timestamp"] = time.Now().Format(time.RFC3339)

	result.MethodRefinementData = refinement
}

// updateMethodMetrics updates the method's success metrics based on execution results.
func (rtc *RealTimeCursor) updateMethodMetrics(ctx context.Context, plan *ExecutionPlan, result *ExecutionResult) error {
	if plan.MethodID == "" {
		return nil // No method to update (custom plan)
	}

	// Determine if the execution was successful overall
	wasSuccessful := result.Status == ExecutionStatusCompleted

	// Calculate a quality rating based on various factors
	rating := rtc.calculateExecutionRating(result)

	// Update the method metrics
	return rtc.methodManager.UpdateMethodMetrics(ctx, plan.MethodID, wasSuccessful, rating)
}

// calculateExecutionRating computes a quality rating (1-10) for the execution.
func (rtc *RealTimeCursor) calculateExecutionRating(result *ExecutionResult) float64 {
	baseRating := 5.0 // Start with neutral rating

	// Factor in success rate
	if result.SuccessfulTasks+result.FailedTasks > 0 {
		successRate := float64(result.SuccessfulTasks) / float64(result.SuccessfulTasks+result.FailedTasks)
		baseRating += (successRate - 0.5) * 6.0 // Scale success rate impact
	}

	// Penalize for excessive duration (if this becomes measurable in the future)
	// For now, we don't have baseline duration expectations

	// Bonus for high confidence results
	avgConfidence := 0.0
	confidenceCount := 0
	for _, taskResult := range result.TaskResults {
		if taskResult.Status == TaskStatusCompleted && taskResult.Confidence > 0 {
			avgConfidence += taskResult.Confidence
			confidenceCount++
		}
	}
	if confidenceCount > 0 {
		avgConfidence /= float64(confidenceCount)
		baseRating += (avgConfidence - 0.5) * 2.0 // Moderate bonus for high confidence
	}

	// Clamp to valid range
	if baseRating < 1.0 {
		baseRating = 1.0
	}
	if baseRating > 10.0 {
		baseRating = 10.0
	}

	return baseRating
}

// storeExecutionResult persists the execution result for tracking and analysis.
func (rtc *RealTimeCursor) storeExecutionResult(ctx context.Context, result *ExecutionResult) error {
	// Convert result to storage node data
	data := map[string]interface{}{
		"plan_id":                result.PlanID,
		"objective_id":           result.ObjectiveID,
		"status":                 string(result.Status),
		"total_tokens_used":      result.TotalTokensUsed,
		"total_duration":         result.TotalDuration.Seconds(),
		"start_time":             result.StartTime.Format(time.RFC3339),
		"end_time":               result.EndTime.Format(time.RFC3339),
		"error_message":          result.ErrorMessage,
		"successful_tasks":       result.SuccessfulTasks,
		"failed_tasks":           result.FailedTasks,
		"method_refinement_data": result.MethodRefinementData,
	}

	// Add task results summary (avoiding too much detail in main node)
	taskSummary := make(map[string]interface{})
	for taskID, taskResult := range result.TaskResults {
		taskSummary[taskID] = map[string]interface{}{
			"status":       string(taskResult.Status),
			"tokens_used":  taskResult.TokensUsed,
			"duration":     taskResult.Duration.Seconds(),
			"confidence":   taskResult.Confidence,
			"tools_used":   taskResult.ToolsUsed,
		}
	}
	data["task_summary"] = taskSummary

	// Create storage node
	node := storage.NewNode("execution_result", data)

	// Store the node
	return rtc.store.AddNode(ctx, node)
}

// GetExecutionHistory returns recent execution results for analysis.
func (rtc *RealTimeCursor) GetExecutionHistory(ctx context.Context, limit int) ([]*ExecutionResult, error) {
	nodes, err := rtc.store.GetNodesByType(ctx, "execution_result")
	if err != nil {
		return nil, fmt.Errorf("failed to query execution results: %w", err)
	}

	var results []*ExecutionResult
	for _, node := range nodes {
		result, err := rtc.nodeToExecutionResult(node)
		if err != nil {
			continue // Skip invalid nodes
		}
		results = append(results, result)
	}

	// Sort by start time (most recent first) and apply limit
	// For simplicity, we'll just return the first 'limit' results
	// A full implementation would sort by timestamp
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// nodeToExecutionResult converts a storage node to an ExecutionResult object.
func (rtc *RealTimeCursor) nodeToExecutionResult(node *storage.Node) (*ExecutionResult, error) {
	if node == nil || node.Type != "execution_result" {
		return nil, fmt.Errorf("invalid execution result node")
	}

	result := &ExecutionResult{
		TaskResults:          make(map[string]*TaskResult),
		MethodRefinementData: make(map[string]interface{}),
	}

	// Extract basic fields
	if planID, ok := node.Data["plan_id"].(string); ok {
		result.PlanID = planID
	}
	if objectiveID, ok := node.Data["objective_id"].(string); ok {
		result.ObjectiveID = objectiveID
	}
	if statusStr, ok := node.Data["status"].(string); ok {
		result.Status = ExecutionStatus(statusStr)
	}

	// Extract numeric fields (handle both int and float64 from JSON)
	if tokensUsed, ok := node.Data["total_tokens_used"].(float64); ok {
		result.TotalTokensUsed = int(tokensUsed)
	}
	if successfulTasks, ok := node.Data["successful_tasks"].(float64); ok {
		result.SuccessfulTasks = int(successfulTasks)
	}
	if failedTasks, ok := node.Data["failed_tasks"].(float64); ok {
		result.FailedTasks = int(failedTasks)
	}

	// Extract duration
	if duration, ok := node.Data["total_duration"].(float64); ok {
		result.TotalDuration = time.Duration(duration) * time.Second
	}

	// Extract timestamps
	if startTime, ok := node.Data["start_time"].(string); ok {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			result.StartTime = t
		}
	}
	if endTime, ok := node.Data["end_time"].(string); ok {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			result.EndTime = t
		}
	}

	// Extract error message
	if errorMsg, ok := node.Data["error_message"].(string); ok {
		result.ErrorMessage = errorMsg
	}

	// Extract method refinement data
	if refinementData, ok := node.Data["method_refinement_data"].(map[string]interface{}); ok {
		result.MethodRefinementData = refinementData
	}

	// For task results, we store a summary to avoid excessive data in the main node
	// Full task results would be stored separately if needed
	if taskSummary, ok := node.Data["task_summary"].(map[string]interface{}); ok {
		for taskID, summaryData := range taskSummary {
			if summary, ok := summaryData.(map[string]interface{}); ok {
				taskResult := &TaskResult{
					TaskID: taskID,
				}
				if status, ok := summary["status"].(string); ok {
					taskResult.Status = TaskStatus(status)
				}
				if tokensUsed, ok := summary["tokens_used"].(float64); ok {
					taskResult.TokensUsed = int(tokensUsed)
				}
				if duration, ok := summary["duration"].(float64); ok {
					taskResult.Duration = time.Duration(duration) * time.Second
				}
				if confidence, ok := summary["confidence"].(float64); ok {
					taskResult.Confidence = confidence
				}
				if toolsUsed, ok := summary["tools_used"].([]interface{}); ok {
					var tools []string
					for _, tool := range toolsUsed {
						if toolStr, ok := tool.(string); ok {
							tools = append(tools, toolStr)
						}
					}
					taskResult.ToolsUsed = tools
				}
				result.TaskResults[taskID] = taskResult
			}
		}
	}

	return result, nil
}

// SetRetryConfig allows customizing retry behavior.
func (rtc *RealTimeCursor) SetRetryConfig(config *RetryConfig) {
	if config != nil {
		rtc.retryConfig = config
	}
}

// GetRetryConfig returns the current retry configuration.
func (rtc *RealTimeCursor) GetRetryConfig() *RetryConfig {
	return rtc.retryConfig
}

// String returns a string representation of task status.
func (ts TaskStatus) String() string {
	return string(ts)
}

// String returns a string representation of execution status.
func (es ExecutionStatus) String() string {
	return string(es)
}

// IsTerminal returns true if the task status indicates the task is finished (success or failure).
func (ts TaskStatus) IsTerminal() bool {
	return ts == TaskStatusCompleted || ts == TaskStatusFailed
}

// IsTerminal returns true if the execution status indicates execution is finished.
func (es ExecutionStatus) IsTerminal() bool {
	return es == ExecutionStatusCompleted || es == ExecutionStatusFailed ||
	       es == ExecutionStatusPartial || es == ExecutionStatusCancelled
}