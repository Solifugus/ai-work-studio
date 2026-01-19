package core

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/ai-work-studio/pkg/storage"
)

// LearningAgent defines the interface for LLM-based learning and method refinement.
// This abstracts the reasoning required for analyzing failures and improving methods.
type LearningAgent interface {
	// AnalyzeExecutionOutcome examines execution results to determine what happened and why
	AnalyzeExecutionOutcome(ctx context.Context, result *ExecutionResult, plan *ExecutionPlan, method *Method) (*ExecutionAnalysis, error)

	// ProposeMethodRefinement suggests improvements based on execution analysis
	ProposeMethodRefinement(ctx context.Context, analysis *ExecutionAnalysis, method *Method) (*MethodRefinement, error)

	// EvaluateRefinement assesses whether a proposed refinement reduces complexity
	EvaluateRefinement(ctx context.Context, original *Method, refinement *MethodRefinement) (*RefinementEvaluation, error)
}

// ExecutionAnalysis captures the learning agent's understanding of execution outcomes.
type ExecutionAnalysis struct {
	// OverallAssessment categorizes the execution outcome
	OverallAssessment ExecutionOutcome

	// PrimaryFailureCause identifies the main reason for failure (if any)
	PrimaryFailureCause string

	// MethodPerformanceIssues lists specific problems with the method
	MethodPerformanceIssues []PerformanceIssue

	// SuccessFactors identifies what contributed to success
	SuccessFactors []string

	// ImprovementOpportunities suggests areas for enhancement
	ImprovementOpportunities []string

	// ComplexityAssessment evaluates method complexity
	ComplexityAssessment ComplexityAnalysis

	// ConfidenceLevel indicates confidence in the analysis (0.0-1.0)
	ConfidenceLevel float64
}

// ExecutionOutcome represents different categories of execution results.
type ExecutionOutcome string

const (
	// OutcomeSuccess indicates the execution was successful and method worked well
	OutcomeSuccess ExecutionOutcome = "success"

	// OutcomePartialSuccess indicates some tasks succeeded but method needs refinement
	OutcomePartialSuccess ExecutionOutcome = "partial_success"

	// OutcomeMethodFailure indicates the method itself has flaws that need addressing
	OutcomeMethodFailure ExecutionOutcome = "method_failure"

	// OutcomeEnvironmentFailure indicates external factors caused failure, method is likely fine
	OutcomeEnvironmentFailure ExecutionOutcome = "environment_failure"

	// OutcomeInsufficientData indicates more execution data is needed before making changes
	OutcomeInsufficientData ExecutionOutcome = "insufficient_data"
)

// PerformanceIssue identifies a specific problem with method execution.
type PerformanceIssue struct {
	// Category classifies the type of issue
	Category IssueCategory

	// Description explains the specific problem
	Description string

	// AffectedSteps lists which method steps were problematic
	AffectedSteps []int

	// Severity indicates how critical this issue is (1-10)
	Severity int

	// SuggestedFix provides a high-level suggestion for addressing the issue
	SuggestedFix string
}

// IssueCategory represents different types of performance problems.
type IssueCategory string

const (
	// IssueComplexity indicates the method is too complex or convoluted
	IssueComplexity IssueCategory = "complexity"

	// IssueEfficiency indicates the method is inefficient or wasteful
	IssueEfficiency IssueCategory = "efficiency"

	// IssueReliability indicates the method is unreliable or unpredictable
	IssueReliability IssueCategory = "reliability"

	// IssueRelevance indicates the method doesn't match the objective well
	IssueRelevance IssueCategory = "relevance"

	// IssueTools indicates the method requires tools that aren't available
	IssueTools IssueCategory = "tools"
)

// ComplexityAnalysis assesses the complexity of a method.
type ComplexityAnalysis struct {
	// CurrentComplexityLevel rates method complexity (1-10)
	CurrentComplexityLevel int

	// ComplexityFactors lists what contributes to complexity
	ComplexityFactors []string

	// SimplificationOpportunities suggests ways to reduce complexity
	SimplificationOpportunities []string

	// OptimalComplexityLevel suggests ideal complexity for this type of objective
	OptimalComplexityLevel int
}

// MethodRefinement proposes changes to improve a method.
type MethodRefinement struct {
	// Type indicates the kind of refinement being proposed
	Type RefinementType

	// NewApproach contains the refined approach steps (for modify/replace types)
	NewApproach []ApproachStep

	// Reasoning explains why this refinement is beneficial
	Reasoning string

	// ExpectedComplexityChange estimates complexity impact (-10 to +10)
	ExpectedComplexityChange int

	// ExpectedSuccessRateImprovement estimates success rate improvement (0-100%)
	ExpectedSuccessRateImprovement float64

	// RequiredVersion indicates what version number the refined method should have
	RequiredVersion string
}

// RefinementType represents different ways to refine a method.
type RefinementType string

const (
	// RefinementNone indicates no changes are needed
	RefinementNone RefinementType = "none"

	// RefinementModify indicates the method should be updated
	RefinementModify RefinementType = "modify"

	// RefinementReplace indicates the method should be replaced entirely
	RefinementReplace RefinementType = "replace"

	// RefinementRetire indicates the method should be deprecated
	RefinementRetire RefinementType = "retire"
)

// RefinementEvaluation assesses whether a proposed refinement meets quality standards.
type RefinementEvaluation struct {
	// IsImprovement indicates whether the refinement is beneficial
	IsImprovement bool

	// ReducesComplexity indicates whether complexity is reduced
	ReducesComplexity bool

	// QualityScore rates the refinement quality (1-10)
	QualityScore float64

	// Concerns lists any issues with the proposed refinement
	Concerns []string

	// Recommendation suggests whether to apply the refinement
	Recommendation RefinementRecommendation
}

// RefinementRecommendation suggests what action to take with a refinement.
type RefinementRecommendation string

const (
	// RecommendApply suggests the refinement should be applied
	RecommendApply RefinementRecommendation = "apply"

	// RecommendRevise suggests the refinement needs further work
	RecommendRevise RefinementRecommendation = "revise"

	// RecommendReject suggests the refinement should be discarded
	RecommendReject RefinementRecommendation = "reject"
)

// LearningLoop orchestrates the integration between CC and RTC with learning feedback.
// It implements the core learning cycle: plan → execute → analyze → refine.
type LearningLoop struct {
	// store provides access to the temporal storage system
	store *storage.Store

	// contemplativeCursor handles strategic planning and method selection
	contemplativeCursor *ContemplativeCursor

	// realTimeCursor handles tactical execution
	realTimeCursor *RealTimeCursor

	// learningAgent provides LLM-based analysis and refinement capabilities
	learningAgent LearningAgent

	// methodManager provides access to method operations
	methodManager *MethodManager

	// objectiveManager provides access to objective operations
	objectiveManager *ObjectiveManager

	// config contains learning loop configuration
	config *LearningLoopConfig
}

// LearningLoopConfig defines configuration for the learning loop behavior.
type LearningLoopConfig struct {
	// MinExecutionsBeforeRefinement sets minimum executions before considering refinement
	MinExecutionsBeforeRefinement int

	// SuccessRateThresholdForRefinement sets the success rate below which refinement is considered
	SuccessRateThresholdForRefinement float64

	// MaxRefinementAttempts limits how many times we try to refine a method
	MaxRefinementAttempts int

	// ComplexityBiasWeight controls how strongly we favor simplicity (0.0-1.0)
	ComplexityBiasWeight float64

	// EnableMethodEvolution controls whether new method versions are created
	EnableMethodEvolution bool

	// PreserveMethodHistory controls whether old method versions are kept
	PreserveMethodHistory bool
}

// DefaultLearningLoopConfig provides sensible defaults for learning loop configuration.
func DefaultLearningLoopConfig() *LearningLoopConfig {
	return &LearningLoopConfig{
		MinExecutionsBeforeRefinement:     3,
		SuccessRateThresholdForRefinement: 75.0,
		MaxRefinementAttempts:             3,
		ComplexityBiasWeight:              0.7,
		EnableMethodEvolution:             true,
		PreserveMethodHistory:             true,
	}
}

// NewLearningLoop creates a new learning loop instance.
func NewLearningLoop(
	store *storage.Store,
	contemplativeCursor *ContemplativeCursor,
	realTimeCursor *RealTimeCursor,
	learningAgent LearningAgent,
) *LearningLoop {
	return &LearningLoop{
		store:               store,
		contemplativeCursor: contemplativeCursor,
		realTimeCursor:      realTimeCursor,
		learningAgent:       learningAgent,
		methodManager:       NewMethodManager(store),
		objectiveManager:    NewObjectiveManager(store),
		config:              DefaultLearningLoopConfig(),
	}
}

// ExecuteObjective is the main entry point for the learning loop.
// It orchestrates the complete cycle: objective analysis → planning → execution → learning.
func (ll *LearningLoop) ExecuteObjective(ctx context.Context, objectiveID string) (*LearningResult, error) {
	startTime := time.Now()

	result := &LearningResult{
		ObjectiveID:   objectiveID,
		StartTime:     startTime,
		ExecutionAttempts: make([]AttemptResult, 0),
	}

	// Main execution loop with retry on method refinement
	for attempt := 0; attempt < ll.config.MaxRefinementAttempts; attempt++ {
		// CC: Create execution plan
		plan, err := ll.contemplativeCursor.CreateExecutionPlan(ctx, objectiveID)
		if err != nil {
			return ll.finalizeResult(result, fmt.Errorf("failed to create execution plan: %w", err))
		}

		// RTC: Execute the plan
		executionResult, err := ll.realTimeCursor.ExecutePlan(ctx, plan)
		if err != nil {
			return ll.finalizeResult(result, fmt.Errorf("failed to execute plan: %w", err))
		}

		// Record this attempt
		attemptResult := AttemptResult{
			AttemptNumber:   attempt + 1,
			PlanID:          plan.ID,
			MethodID:        plan.MethodID,
			ExecutionResult: executionResult,
			CompletedAt:     time.Now(),
		}
		result.ExecutionAttempts = append(result.ExecutionAttempts, attemptResult)

		// Analyze execution outcome
		shouldContinue, err := ll.analyzeAndLearnFromExecution(ctx, plan, executionResult, &attemptResult)
		if err != nil {
			return ll.finalizeResult(result, fmt.Errorf("failed to analyze execution: %w", err))
		}

		// If successful or no improvement possible, we're done
		if !shouldContinue {
			result.FinalOutcome = OutcomeSuccess
			if executionResult.Status == ExecutionStatusCompleted {
				result.WasSuccessful = true
			} else if executionResult.Status == ExecutionStatusPartial {
				result.FinalOutcome = OutcomePartialSuccess
				result.WasSuccessful = true // Partial success is still success
			}
			break
		}

		// If this was the last attempt, mark as failed
		if attempt == ll.config.MaxRefinementAttempts-1 {
			result.FinalOutcome = OutcomeMethodFailure
			result.WasSuccessful = false
		}
	}

	return ll.finalizeResult(result, nil)
}

// analyzeAndLearnFromExecution analyzes execution results and performs learning.
// Returns true if another attempt should be made (method was refined), false if we're done.
func (ll *LearningLoop) analyzeAndLearnFromExecution(
	ctx context.Context,
	plan *ExecutionPlan,
	executionResult *ExecutionResult,
	attemptResult *AttemptResult,
) (bool, error) {
	// Skip learning if method ID is empty (custom plan)
	if plan.MethodID == "" {
		return false, nil
	}

	// Get the method that was executed
	method, err := ll.methodManager.GetMethod(ctx, plan.MethodID)
	if err != nil {
		return false, fmt.Errorf("failed to retrieve method for learning: %w", err)
	}

	// Learning agent analyzes the execution
	analysis, err := ll.learningAgent.AnalyzeExecutionOutcome(ctx, executionResult, plan, method)
	if err != nil {
		// Log warning but don't fail - learning is optional
		fmt.Printf("Warning: failed to analyze execution outcome: %v\n", err)
		return false, nil
	}

	attemptResult.ExecutionAnalysis = analysis

	// Update method metrics based on execution outcome
	wasSuccessful := executionResult.Status == ExecutionStatusCompleted || executionResult.Status == ExecutionStatusPartial
	rating := ll.calculateMethodRating(executionResult, analysis)

	if err := ll.methodManager.UpdateMethodMetrics(ctx, plan.MethodID, wasSuccessful, rating); err != nil {
		fmt.Printf("Warning: failed to update method metrics: %v\n", err)
	}

	// Decide whether to attempt method refinement
	if !ll.shouldAttemptRefinement(analysis, method) {
		return false, nil
	}

	// Attempt to refine the method
	refined, err := ll.attemptMethodRefinement(ctx, analysis, method)
	if err != nil {
		fmt.Printf("Warning: failed to refine method: %v\n", err)
		return false, nil
	}

	attemptResult.RefinementApplied = refined
	return refined, nil
}

// shouldAttemptRefinement determines if method refinement should be attempted.
func (ll *LearningLoop) shouldAttemptRefinement(analysis *ExecutionAnalysis, method *Method) bool {
	// Don't refine if execution was successful
	if analysis.OverallAssessment == OutcomeSuccess {
		return false
	}

	// Don't refine if it's an environment failure (method is fine)
	if analysis.OverallAssessment == OutcomeEnvironmentFailure {
		return false
	}

	// Don't refine if we need more data
	if analysis.OverallAssessment == OutcomeInsufficientData {
		return false
	}

	// Check if method has enough execution history
	if method.Metrics.ExecutionCount < ll.config.MinExecutionsBeforeRefinement {
		return false
	}

	// Check if success rate is below threshold
	if method.Metrics.SuccessRate() >= ll.config.SuccessRateThresholdForRefinement {
		return false
	}

	return true
}

// attemptMethodRefinement tries to refine a method based on execution analysis.
// Returns true if refinement was successfully applied, false otherwise.
func (ll *LearningLoop) attemptMethodRefinement(
	ctx context.Context,
	analysis *ExecutionAnalysis,
	method *Method,
) (bool, error) {
	// Learning agent proposes refinement
	refinement, err := ll.learningAgent.ProposeMethodRefinement(ctx, analysis, method)
	if err != nil {
		return false, fmt.Errorf("failed to propose method refinement: %w", err)
	}

	// Skip if no refinement is proposed
	if refinement.Type == RefinementNone {
		return false, nil
	}

	// Evaluate the proposed refinement
	evaluation, err := ll.learningAgent.EvaluateRefinement(ctx, method, refinement)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate method refinement: %w", err)
	}

	// Apply complexity bias - strongly prefer refinements that reduce complexity
	if evaluation.Recommendation != RecommendApply {
		return false, nil
	}

	if !evaluation.ReducesComplexity && ll.config.ComplexityBiasWeight > 0.5 {
		fmt.Printf("Rejecting refinement that doesn't reduce complexity (bias weight: %.2f)\n", ll.config.ComplexityBiasWeight)
		return false, nil
	}

	// Apply the refinement based on type
	switch refinement.Type {
	case RefinementModify:
		return ll.applyMethodModification(ctx, method, refinement)
	case RefinementReplace:
		return ll.applyMethodReplacement(ctx, method, refinement)
	case RefinementRetire:
		return ll.applyMethodRetirement(ctx, method)
	default:
		return false, fmt.Errorf("unknown refinement type: %s", refinement.Type)
	}
}

// applyMethodModification updates an existing method with refinements.
func (ll *LearningLoop) applyMethodModification(
	ctx context.Context,
	method *Method,
	refinement *MethodRefinement,
) (bool, error) {
	// Create method evolution with the refined approach
	newMethod := &Method{
		Name:        method.Name,
		Description: method.Description + " (refined)",
		Approach:    refinement.NewApproach,
		Domain:      method.Domain,
		Version:     refinement.RequiredVersion,
		Status:      MethodStatusActive,
		Metrics: SuccessMetrics{
			ExecutionCount: 0,
			SuccessCount:   0,
			LastUsed:       time.Time{},
			AverageRating:  0.0,
		},
		UserContext: method.UserContext,
		CreatedAt:   time.Now(),
	}

	// Create evolution relationship
	evolutionReason := fmt.Sprintf("Refined due to: %s", refinement.Reasoning)
	if err := ll.methodManager.CreateMethodEvolution(ctx, method.ID, newMethod, evolutionReason); err != nil {
		return false, fmt.Errorf("failed to create method evolution: %w", err)
	}

	return true, nil
}

// applyMethodReplacement creates a completely new method to replace an old one.
func (ll *LearningLoop) applyMethodReplacement(
	ctx context.Context,
	method *Method,
	refinement *MethodRefinement,
) (bool, error) {
	// Create entirely new method
	newMethod := &Method{
		Name:        method.Name + " v2",
		Description: "Replacement method: " + refinement.Reasoning,
		Approach:    refinement.NewApproach,
		Domain:      method.Domain,
		Version:     "1.0.0", // Reset version for replacement
		Status:      MethodStatusActive,
		Metrics: SuccessMetrics{
			ExecutionCount: 0,
			SuccessCount:   0,
			LastUsed:       time.Time{},
			AverageRating:  0.0,
		},
		UserContext: method.UserContext,
		CreatedAt:   time.Now(),
	}

	// Create evolution relationship
	evolutionReason := fmt.Sprintf("Replaced due to: %s", refinement.Reasoning)
	if err := ll.methodManager.CreateMethodEvolution(ctx, method.ID, newMethod, evolutionReason); err != nil {
		return false, fmt.Errorf("failed to create method replacement: %w", err)
	}

	return true, nil
}

// applyMethodRetirement marks a method as deprecated.
func (ll *LearningLoop) applyMethodRetirement(ctx context.Context, method *Method) (bool, error) {
	// Mark method as deprecated
	deprecated := MethodStatusDeprecated
	updates := MethodUpdates{
		Status: &deprecated,
	}

	_, err := ll.methodManager.UpdateMethod(ctx, method.ID, updates)
	if err != nil {
		return false, fmt.Errorf("failed to retire method: %w", err)
	}

	return false, nil // Don't retry when retiring a method
}

// calculateMethodRating computes a quality rating for method performance.
func (ll *LearningLoop) calculateMethodRating(result *ExecutionResult, analysis *ExecutionAnalysis) float64 {
	baseRating := 5.0

	// Factor in execution success rate
	if result.SuccessfulTasks+result.FailedTasks > 0 {
		successRate := float64(result.SuccessfulTasks) / float64(result.SuccessfulTasks+result.FailedTasks)
		baseRating += (successRate - 0.5) * 6.0 // Scale to contribute up to ±3 points
	}

	// Factor in analysis confidence
	if analysis != nil {
		baseRating += (analysis.ConfidenceLevel - 0.5) * 2.0 // Scale to contribute up to ±1 point
	}

	// Apply complexity bias - reward simpler approaches
	if analysis != nil && analysis.ComplexityAssessment.CurrentComplexityLevel > 0 {
		complexityPenalty := float64(analysis.ComplexityAssessment.CurrentComplexityLevel) / 10.0 * ll.config.ComplexityBiasWeight
		baseRating -= complexityPenalty
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

// finalizeResult completes the learning result with final status and error handling.
func (ll *LearningLoop) finalizeResult(result *LearningResult, err error) (*LearningResult, error) {
	result.EndTime = time.Now()
	result.TotalDuration = result.EndTime.Sub(result.StartTime)

	if err != nil {
		result.ErrorMessage = err.Error()
		result.WasSuccessful = false
		if result.FinalOutcome == "" {
			result.FinalOutcome = OutcomeMethodFailure
		}
	}

	return result, err
}

// GetConfiguration returns the current learning loop configuration.
func (ll *LearningLoop) GetConfiguration() *LearningLoopConfig {
	return ll.config
}

// SetConfiguration updates the learning loop configuration.
func (ll *LearningLoop) SetConfiguration(config *LearningLoopConfig) {
	if config != nil {
		ll.config = config
	}
}

// LearningResult represents the outcome of executing an objective through the learning loop.
type LearningResult struct {
	// ObjectiveID identifies which objective was executed
	ObjectiveID string

	// WasSuccessful indicates if the objective was ultimately achieved
	WasSuccessful bool

	// FinalOutcome categorizes the final result
	FinalOutcome ExecutionOutcome

	// ExecutionAttempts contains results from each execution attempt
	ExecutionAttempts []AttemptResult

	// StartTime is when the learning loop execution began
	StartTime time.Time

	// EndTime is when the learning loop execution completed
	EndTime time.Time

	// TotalDuration is the total time spent in the learning loop
	TotalDuration time.Duration

	// ErrorMessage contains error details if execution failed
	ErrorMessage string
}

// AttemptResult represents the outcome of a single execution attempt.
type AttemptResult struct {
	// AttemptNumber indicates which attempt this was (1-based)
	AttemptNumber int

	// PlanID identifies the execution plan used
	PlanID string

	// MethodID identifies the method that was executed
	MethodID string

	// ExecutionResult contains the raw execution results from RTC
	ExecutionResult *ExecutionResult

	// ExecutionAnalysis contains the learning agent's analysis (may be nil)
	ExecutionAnalysis *ExecutionAnalysis

	// RefinementApplied indicates whether method refinement was applied after this attempt
	RefinementApplied bool

	// CompletedAt is when this attempt finished
	CompletedAt time.Time
}

// GetSuccessfulAttempts returns only the attempts that were successful.
func (lr *LearningResult) GetSuccessfulAttempts() []AttemptResult {
	var successful []AttemptResult
	for _, attempt := range lr.ExecutionAttempts {
		if attempt.ExecutionResult.Status == ExecutionStatusCompleted ||
			attempt.ExecutionResult.Status == ExecutionStatusPartial {
			successful = append(successful, attempt)
		}
	}
	return successful
}

// GetTotalTokensUsed returns the sum of tokens used across all attempts.
func (lr *LearningResult) GetTotalTokensUsed() int {
	total := 0
	for _, attempt := range lr.ExecutionAttempts {
		total += attempt.ExecutionResult.TotalTokensUsed
	}
	return total
}

// GetMethodsUsed returns the unique set of method IDs that were executed.
func (lr *LearningResult) GetMethodsUsed() []string {
	seen := make(map[string]bool)
	var methods []string

	for _, attempt := range lr.ExecutionAttempts {
		if attempt.MethodID != "" && !seen[attempt.MethodID] {
			methods = append(methods, attempt.MethodID)
			seen[attempt.MethodID] = true
		}
	}

	return methods
}

// String returns string representations for enums.
func (eo ExecutionOutcome) String() string {
	return string(eo)
}

func (rt RefinementType) String() string {
	return string(rt)
}

func (rr RefinementRecommendation) String() string {
	return string(rr)
}

func (ic IssueCategory) String() string {
	return string(ic)
}