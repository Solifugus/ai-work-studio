package core

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/ai-work-studio/pkg/storage"
)

// TaskDependency represents a dependency between tasks in an execution plan.
type TaskDependency struct {
	// TaskID is the ID of the task that depends on another
	TaskID string

	// DependsOnTaskID is the ID of the task that must complete first
	DependsOnTaskID string

	// Reason explains why this dependency exists
	Reason string
}

// TaskContext contains minimal context needed for a task execution.
// Follows the principle of passing references, not full data.
type TaskContext struct {
	// InputRefs contains references to input data/artifacts needed for this task
	InputRefs []string `json:"input_refs,omitempty"`

	// OutputRef specifies where the task should store its output
	OutputRef string `json:"output_ref,omitempty"`

	// Parameters contains task-specific parameters as key-value pairs
	Parameters map[string]interface{} `json:"parameters,omitempty"`

	// TokenBudget is the maximum number of LLM tokens this task should use
	TokenBudget int `json:"token_budget,omitempty"`

	// Priority indicates task priority within the plan (1-10)
	Priority int `json:"priority,omitempty"`
}

// ExecutionTask represents a single task in an execution plan.
type ExecutionTask struct {
	// ID uniquely identifies this task within the plan
	ID string

	// Type indicates the kind of operation (e.g., "analyze", "generate", "validate")
	Type string

	// Description provides human-readable explanation of what this task does
	Description string

	// Context contains minimal execution context
	Context TaskContext

	// MethodStepIndex indicates which step of the method this task implements (-1 if not method-based)
	MethodStepIndex int

	// EstimatedTokens is the expected LLM token consumption
	EstimatedTokens int

	// CreatedAt is when this task was created
	CreatedAt time.Time
}

// ExecutionPlan represents a sequence of tasks designed to achieve an objective.
type ExecutionPlan struct {
	// ID uniquely identifies this plan
	ID string

	// ObjectiveID links this plan to the objective it serves
	ObjectiveID string

	// MethodID links this plan to the method it implements (empty if custom)
	MethodID string

	// Title provides a short description of what the plan accomplishes
	Title string

	// Tasks is the ordered sequence of execution tasks
	Tasks []ExecutionTask

	// Dependencies specify prerequisite relationships between tasks
	Dependencies []TaskDependency

	// TotalEstimatedTokens is the sum of all task token estimates
	TotalEstimatedTokens int

	// CreatedBy indicates what created this plan (e.g., "contemplative_cursor")
	CreatedBy string

	// CreatedAt is when this plan was generated
	CreatedAt time.Time
}

// LLMReasoner defines the interface for LLM-based reasoning operations.
// This allows the CC to be decoupled from specific LLM implementations.
type LLMReasoner interface {
	// AnalyzeObjective examines an objective and provides strategic insights
	AnalyzeObjective(ctx context.Context, objective *Objective, goalContext *Goal) (*ObjectiveAnalysis, error)

	// SelectMethod chooses the best method from available options
	SelectMethod(ctx context.Context, analysis *ObjectiveAnalysis, methods []*Method) (*MethodSelection, error)

	// DesignMethod creates a new method for objectives without cached approaches
	DesignMethod(ctx context.Context, analysis *ObjectiveAnalysis) (*Method, error)

	// DecomposePlan breaks down complex objectives into subtasks
	DecomposePlan(ctx context.Context, objective *Objective, method *Method) (*ExecutionPlan, error)
}

// ObjectiveAnalysis captures the CC's understanding of what an objective requires.
type ObjectiveAnalysis struct {
	// ComplexityLevel indicates how complex the objective is (1-10)
	ComplexityLevel int

	// RequiredCapabilities lists the tools/skills needed
	RequiredCapabilities []string

	// KeyChallenges identifies potential difficulties
	KeyChallenges []string

	// SuccessCriteria defines how to measure completion
	SuccessCriteria []string

	// EstimatedTokenBudget suggests LLM token usage
	EstimatedTokenBudget int

	// RecommendedApproach suggests high-level strategy
	RecommendedApproach string

	// DomainContext identifies the domain this objective belongs to
	DomainContext string
}

// MethodSelection captures the CC's choice of method and reasoning.
type MethodSelection struct {
	// SelectedMethodID is the ID of the chosen method
	SelectedMethodID string

	// ConfidenceLevel indicates how confident the selection is (0.0-1.0)
	ConfidenceLevel float64

	// SelectionReason explains why this method was chosen
	SelectionReason string

	// RequiredAdaptations lists changes needed to apply the method
	RequiredAdaptations []string

	// AlternativeMethodIDs lists other viable options
	AlternativeMethodIDs []string
}

// ContemplativeCursor represents the strategic planning component of the agent system.
// It analyzes objectives, selects or designs methods, and creates execution plans.
type ContemplativeCursor struct {
	// store provides access to the temporal storage system
	store *storage.Store

	// goalManager provides access to goal operations
	goalManager *GoalManager

	// methodManager provides access to method operations
	methodManager *MethodManager

	// objectiveManager provides access to objective operations
	objectiveManager *ObjectiveManager

	// reasoner provides LLM-based analysis and planning capabilities
	reasoner LLMReasoner
}

// NewContemplativeCursor creates a new CC instance with the given dependencies.
func NewContemplativeCursor(store *storage.Store, reasoner LLMReasoner) *ContemplativeCursor {
	return &ContemplativeCursor{
		store:            store,
		goalManager:      NewGoalManager(store),
		methodManager:    NewMethodManager(store),
		objectiveManager: NewObjectiveManager(store),
		reasoner:         reasoner,
	}
}

// CreateExecutionPlan generates an execution plan for the given objective.
// This is the main entry point for CC planning capabilities.
func (cc *ContemplativeCursor) CreateExecutionPlan(ctx context.Context, objectiveID string) (*ExecutionPlan, error) {
	// Get the objective and its associated goal
	objective, err := cc.objectiveManager.GetObjective(ctx, objectiveID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve objective: %w", err)
	}

	goal, err := cc.goalManager.GetGoal(ctx, objective.GoalID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve goal context: %w", err)
	}

	// Analyze the objective to understand its requirements
	analysis, err := cc.reasoner.AnalyzeObjective(ctx, objective, goal)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze objective: %w", err)
	}

	// Query method cache for applicable methods
	methodID, err := cc.findBestMethod(ctx, analysis)
	if err != nil {
		return nil, fmt.Errorf("failed to find method: %w", err)
	}

	var selectedMethod *Method
	if methodID != "" {
		// Use cached method
		selectedMethod, err = cc.methodManager.GetMethod(ctx, methodID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve cached method: %w", err)
		}
	} else {
		// Design new method
		selectedMethod, err = cc.reasoner.DesignMethod(ctx, analysis)
		if err != nil {
			return nil, fmt.Errorf("failed to design new method: %w", err)
		}

		// Cache the new method for future use
		if err := cc.cacheNewMethod(ctx, selectedMethod); err != nil {
			// Log warning but don't fail - we can still proceed with the plan
			fmt.Printf("Warning: failed to cache new method: %v\n", err)
		}
	}

	// Create the execution plan
	plan, err := cc.reasoner.DecomposePlan(ctx, objective, selectedMethod)
	if err != nil {
		return nil, fmt.Errorf("failed to decompose execution plan: %w", err)
	}

	// Set plan metadata
	plan.ID = generatePlanID()
	plan.ObjectiveID = objectiveID
	plan.MethodID = selectedMethod.ID
	plan.CreatedBy = "contemplative_cursor"
	plan.CreatedAt = time.Now()

	// Calculate total estimated tokens
	totalTokens := 0
	for _, task := range plan.Tasks {
		totalTokens += task.EstimatedTokens
	}
	plan.TotalEstimatedTokens = totalTokens

	return plan, nil
}

// findBestMethod queries the method cache for the most suitable method.
// Returns empty string if no suitable cached method is found.
func (cc *ContemplativeCursor) findBestMethod(ctx context.Context, analysis *ObjectiveAnalysis) (string, error) {
	// Query active methods that might be applicable
	methodFilter := MethodFilter{
		Status: &[]MethodStatus{MethodStatusActive}[0],
	}

	// First try domain-specific methods
	if analysis.DomainContext != "" {
		domainFilter := methodFilter
		domain := MethodDomainSpecific
		domainFilter.Domain = &domain

		methods, err := cc.methodManager.ListMethods(ctx, domainFilter)
		if err != nil {
			return "", fmt.Errorf("failed to query domain-specific methods: %w", err)
		}

		if bestMethod := cc.selectBestFromCandidates(methods, analysis); bestMethod != nil {
			return bestMethod.ID, nil
		}
	}

	// Try user-specific methods
	userFilter := methodFilter
	userDomain := MethodDomainUser
	userFilter.Domain = &userDomain

	methods, err := cc.methodManager.ListMethods(ctx, userFilter)
	if err != nil {
		return "", fmt.Errorf("failed to query user-specific methods: %w", err)
	}

	if bestMethod := cc.selectBestFromCandidates(methods, analysis); bestMethod != nil {
		return bestMethod.ID, nil
	}

	// Finally try general methods
	generalFilter := methodFilter
	generalDomain := MethodDomainGeneral
	generalFilter.Domain = &generalDomain

	methods, err = cc.methodManager.ListMethods(ctx, generalFilter)
	if err != nil {
		return "", fmt.Errorf("failed to query general methods: %w", err)
	}

	if bestMethod := cc.selectBestFromCandidates(methods, analysis); bestMethod != nil {
		return bestMethod.ID, nil
	}

	// No suitable method found
	return "", nil
}

// selectBestFromCandidates chooses the best method from a list of candidates.
// Returns nil if no method is sufficiently suitable.
func (cc *ContemplativeCursor) selectBestFromCandidates(candidates []*Method, analysis *ObjectiveAnalysis) *Method {
	if len(candidates) == 0 {
		return nil
	}

	// For now, use a simple heuristic based on success rate and recent usage
	// In a full implementation, this would use the LLM reasoner for more sophisticated selection
	var bestMethod *Method
	bestScore := 0.0

	for _, method := range candidates {
		// Calculate a simple score based on success rate and recency
		successRate := method.Metrics.SuccessRate()

		// Boost score for recently used methods
		recencyBonus := 0.0
		if !method.Metrics.LastUsed.IsZero() {
			daysSinceLastUse := time.Since(method.Metrics.LastUsed).Hours() / 24
			if daysSinceLastUse < 30 { // Boost methods used in last 30 days
				recencyBonus = (30 - daysSinceLastUse) / 30 * 20 // Up to 20 point bonus
			}
		}

		score := successRate + recencyBonus

		// Prefer methods that have been used before (avoid completely untested methods)
		if method.Metrics.ExecutionCount > 0 && score > bestScore {
			bestScore = score
			bestMethod = method
		}
	}

	// Only return a method if it has a reasonable success rate
	if bestMethod != nil && bestMethod.Metrics.SuccessRate() >= 60.0 {
		return bestMethod
	}

	return nil
}

// cacheNewMethod stores a newly designed method in the system for future reuse.
func (cc *ContemplativeCursor) cacheNewMethod(ctx context.Context, method *Method) error {
	// Create the method in storage using the method manager
	createdMethod, err := cc.methodManager.CreateMethod(
		ctx,
		method.Name,
		method.Description,
		method.Approach,
		method.Domain,
		method.UserContext,
	)
	if err != nil {
		return fmt.Errorf("failed to create method: %w", err)
	}

	// Update the method ID to the one assigned by storage
	method.ID = createdMethod.ID
	return nil
}

// generatePlanID creates a unique identifier for an execution plan.
func generatePlanID() string {
	// Use timestamp-based ID for simplicity
	// In production, would use UUID or more sophisticated ID generation
	return fmt.Sprintf("plan_%d", time.Now().UnixNano())
}

// AnalyzeObjectiveComplexity provides a high-level complexity assessment without full LLM analysis.
// Useful for quick triage and resource allocation decisions.
func (cc *ContemplativeCursor) AnalyzeObjectiveComplexity(ctx context.Context, objectiveID string) (int, error) {
	objective, err := cc.objectiveManager.GetObjective(ctx, objectiveID)
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve objective: %w", err)
	}

	// Simple heuristic-based complexity analysis
	complexity := 1 // Base complexity

	// Longer descriptions suggest more complexity
	if len(objective.Description) > 200 {
		complexity += 2
	} else if len(objective.Description) > 100 {
		complexity += 1
	}

	// High priority often indicates complexity
	if objective.Priority >= 8 {
		complexity += 2
	} else if objective.Priority >= 6 {
		complexity += 1
	}

	// Context size might indicate complexity
	if len(objective.Context) > 5 {
		complexity += 1
	}

	// Cap at 10
	if complexity > 10 {
		complexity = 10
	}

	return complexity, nil
}

// EstimateTokenBudget provides a rough estimate of token requirements for an objective.
// This helps with resource planning before full analysis.
func (cc *ContemplativeCursor) EstimateTokenBudget(ctx context.Context, objectiveID string) (int, error) {
	complexity, err := cc.AnalyzeObjectiveComplexity(ctx, objectiveID)
	if err != nil {
		return 0, err
	}

	// Simple token estimation based on complexity
	baseTokens := 500
	complexityMultiplier := complexity * 200

	return baseTokens + complexityMultiplier, nil
}

// GetMethodCacheStatistics returns information about the current method cache.
func (cc *ContemplativeCursor) GetMethodCacheStatistics(ctx context.Context) (map[string]interface{}, error) {
	allMethods, err := cc.methodManager.ListMethods(ctx, MethodFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to query methods: %w", err)
	}

	stats := map[string]interface{}{
		"total_methods":    len(allMethods),
		"active_methods":   0,
		"deprecated_methods": 0,
		"superseded_methods": 0,
		"general_domain":   0,
		"specific_domain":  0,
		"user_domain":      0,
		"avg_success_rate": 0.0,
		"most_used_method": "",
		"best_rated_method": "",
	}

	totalSuccessRate := 0.0
	validMethods := 0
	maxExecutions := 0
	bestRating := 0.0
	var mostUsedMethod, bestRatedMethod string

	for _, method := range allMethods {
		// Count by status
		switch method.Status {
		case MethodStatusActive:
			stats["active_methods"] = stats["active_methods"].(int) + 1
		case MethodStatusDeprecated:
			stats["deprecated_methods"] = stats["deprecated_methods"].(int) + 1
		case MethodStatusSuperseded:
			stats["superseded_methods"] = stats["superseded_methods"].(int) + 1
		}

		// Count by domain
		switch method.Domain {
		case MethodDomainGeneral:
			stats["general_domain"] = stats["general_domain"].(int) + 1
		case MethodDomainSpecific:
			stats["specific_domain"] = stats["specific_domain"].(int) + 1
		case MethodDomainUser:
			stats["user_domain"] = stats["user_domain"].(int) + 1
		}

		// Track metrics
		if method.Metrics.ExecutionCount > 0 {
			totalSuccessRate += method.Metrics.SuccessRate()
			validMethods++

			// Find most used
			if method.Metrics.ExecutionCount > maxExecutions {
				maxExecutions = method.Metrics.ExecutionCount
				mostUsedMethod = method.Name
			}

			// Find best rated
			if method.Metrics.AverageRating > bestRating {
				bestRating = method.Metrics.AverageRating
				bestRatedMethod = method.Name
			}
		}
	}

	// Calculate averages
	if validMethods > 0 {
		stats["avg_success_rate"] = totalSuccessRate / float64(validMethods)
	}
	stats["most_used_method"] = mostUsedMethod
	stats["best_rated_method"] = bestRatedMethod

	return stats, nil
}