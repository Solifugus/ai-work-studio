package test

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Solifugus/ai-work-studio/pkg/core"
)

// MockLLMReasoner implements the LLMReasoner interface for testing.
type MockLLMReasoner struct {
	// Configuration for mock behavior
	ShouldFailAnalysis     bool
	ShouldFailMethodSelect bool
	ShouldFailMethodDesign bool
	ShouldFailPlanDecomp   bool

	// Captured calls for verification
	AnalyzeCalls        []MockAnalyzeCall
	SelectMethodCalls   []MockSelectMethodCall
	DesignMethodCalls   []MockDesignMethodCall
	DecomposePlanCalls  []MockDecomposePlanCall

	// Customizable responses
	CustomAnalysis      *core.ObjectiveAnalysis
	CustomMethodID      string
	CustomMethod        *core.Method
	CustomPlan          *core.ExecutionPlan
}

type MockAnalyzeCall struct {
	Objective   *core.Objective
	GoalContext *core.Goal
}

type MockSelectMethodCall struct {
	Analysis *core.ObjectiveAnalysis
	Methods  []*core.Method
}

type MockDesignMethodCall struct {
	Analysis *core.ObjectiveAnalysis
}

type MockDecomposePlanCall struct {
	Objective *core.Objective
	Method    *core.Method
}

// NewMockLLMReasoner creates a mock LLM reasoner with default behavior.
func NewMockLLMReasoner() *MockLLMReasoner {
	return &MockLLMReasoner{
		AnalyzeCalls:        make([]MockAnalyzeCall, 0),
		SelectMethodCalls:   make([]MockSelectMethodCall, 0),
		DesignMethodCalls:   make([]MockDesignMethodCall, 0),
		DecomposePlanCalls:  make([]MockDecomposePlanCall, 0),
	}
}

// AnalyzeObjective implements LLMReasoner interface.
func (m *MockLLMReasoner) AnalyzeObjective(ctx context.Context, objective *core.Objective, goalContext *core.Goal) (*core.ObjectiveAnalysis, error) {
	m.AnalyzeCalls = append(m.AnalyzeCalls, MockAnalyzeCall{
		Objective:   objective,
		GoalContext: goalContext,
	})

	if m.ShouldFailAnalysis {
		return nil, fmt.Errorf("mock analysis failure")
	}

	if m.CustomAnalysis != nil {
		return m.CustomAnalysis, nil
	}

	// Default analysis based on objective complexity
	complexityLevel := 5
	if len(objective.Description) > 200 {
		complexityLevel = 8
	} else if len(objective.Description) > 100 {
		complexityLevel = 6
	}

	return &core.ObjectiveAnalysis{
		ComplexityLevel: complexityLevel,
		RequiredCapabilities: []string{"analysis", "problem_solving"},
		KeyChallenges: []string{"data_quality", "stakeholder_alignment"},
		SuccessCriteria: []string{"clear_deliverables", "measurable_outcomes"},
		EstimatedTokenBudget: complexityLevel * 400,
		RecommendedApproach: "systematic_analysis",
		DomainContext: determineDomain(objective.Description),
	}, nil
}

// SelectMethod implements LLMReasoner interface.
func (m *MockLLMReasoner) SelectMethod(ctx context.Context, analysis *core.ObjectiveAnalysis, methods []*core.Method) (*core.MethodSelection, error) {
	m.SelectMethodCalls = append(m.SelectMethodCalls, MockSelectMethodCall{
		Analysis: analysis,
		Methods:  methods,
	})

	if m.ShouldFailMethodSelect {
		return nil, fmt.Errorf("mock method selection failure")
	}

	if m.CustomMethodID != "" {
		return &core.MethodSelection{
			SelectedMethodID:     m.CustomMethodID,
			ConfidenceLevel:      0.85,
			SelectionReason:      "custom mock selection",
			RequiredAdaptations:  []string{},
			AlternativeMethodIDs: []string{},
		}, nil
	}

	// Default: select first available method
	if len(methods) > 0 {
		return &core.MethodSelection{
			SelectedMethodID:     methods[0].ID,
			ConfidenceLevel:      0.75,
			SelectionReason:      "mock default selection - first available method",
			RequiredAdaptations:  []string{},
			AlternativeMethodIDs: getAlternativeMethodIDs(methods[1:]),
		}, nil
	}

	return nil, fmt.Errorf("no methods available for selection")
}

// DesignMethod implements LLMReasoner interface.
func (m *MockLLMReasoner) DesignMethod(ctx context.Context, analysis *core.ObjectiveAnalysis) (*core.Method, error) {
	m.DesignMethodCalls = append(m.DesignMethodCalls, MockDesignMethodCall{
		Analysis: analysis,
	})

	if m.ShouldFailMethodDesign {
		return nil, fmt.Errorf("mock method design failure")
	}

	if m.CustomMethod != nil {
		return m.CustomMethod, nil
	}

	// Default: create a simple method based on analysis
	approach := createDefaultApproach(analysis.ComplexityLevel)
	domain := core.MethodDomainGeneral
	if analysis.DomainContext != "" {
		domain = core.MethodDomainSpecific
	}

	return &core.Method{
		Name:        fmt.Sprintf("Generated Method for %s", analysis.DomainContext),
		Description: "Automatically generated method based on objective analysis",
		Approach:    approach,
		Domain:      domain,
		Version:     "1.0.0",
		Status:      core.MethodStatusActive,
		Metrics: core.SuccessMetrics{
			ExecutionCount: 0,
			SuccessCount:   0,
			LastUsed:       time.Time{},
			AverageRating:  0.0,
		},
		UserContext: map[string]interface{}{
			"generated": true,
			"complexity": analysis.ComplexityLevel,
		},
		CreatedAt: time.Now(),
	}, nil
}

// DecomposePlan implements LLMReasoner interface.
func (m *MockLLMReasoner) DecomposePlan(ctx context.Context, objective *core.Objective, method *core.Method) (*core.ExecutionPlan, error) {
	m.DecomposePlanCalls = append(m.DecomposePlanCalls, MockDecomposePlanCall{
		Objective: objective,
		Method:    method,
	})

	if m.ShouldFailPlanDecomp {
		return nil, fmt.Errorf("mock plan decomposition failure")
	}

	if m.CustomPlan != nil {
		return m.CustomPlan, nil
	}

	// Default: create plan based on method steps
	tasks := make([]core.ExecutionTask, len(method.Approach))
	dependencies := make([]core.TaskDependency, 0)

	for i, step := range method.Approach {
		taskID := fmt.Sprintf("task-%d", i+1)
		tasks[i] = core.ExecutionTask{
			ID:          taskID,
			Type:        determineTaskType(step.Description),
			Description: step.Description,
			Context: core.TaskContext{
				InputRefs:   generateInputRefs(i),
				OutputRef:   fmt.Sprintf("data://task_%d_output", i+1),
				Parameters:  map[string]interface{}{"step_index": i},
				TokenBudget: 500,
				Priority:    objective.Priority,
			},
			MethodStepIndex: i,
			EstimatedTokens: 500,
			CreatedAt:      time.Now(),
		}

		// Add dependency on previous task (except for first task)
		if i > 0 {
			dependencies = append(dependencies, core.TaskDependency{
				TaskID:          taskID,
				DependsOnTaskID: fmt.Sprintf("task-%d", i),
				Reason:          "Sequential execution required",
			})
		}
	}

	return &core.ExecutionPlan{
		ObjectiveID:          objective.ID,
		MethodID:             method.ID,
		Title:                fmt.Sprintf("Mock Plan for %s", objective.Title),
		Tasks:                tasks,
		Dependencies:         dependencies,
		TotalEstimatedTokens: len(tasks) * 500,
		CreatedBy:            "mock-contemplative-cursor",
		CreatedAt:            time.Now(),
	}, nil
}

// MockTaskExecutor implements the TaskExecutor interface for testing.
type MockTaskExecutor struct {
	// Configuration for mock behavior
	ShouldFailExecution bool
	ShouldTimeOut      bool
	FailOnTaskType     string // Fail when encountering specific task types

	// Captured calls for verification
	ExecuteCalls []MockExecuteCall
	ToolCalls    []MockToolCall

	// Customizable responses
	CustomResults map[string]*core.TaskResult // Map task ID to custom result
	TokenUsage    int                         // Tokens to report per task
}

type MockExecuteCall struct {
	Task        *core.ExecutionTask
	FullContext map[string]interface{}
}

type MockToolCall struct {
	Tool string
}

// NewMockTaskExecutor creates a mock task executor with default behavior.
func NewMockTaskExecutor() *MockTaskExecutor {
	return &MockTaskExecutor{
		ExecuteCalls:  make([]MockExecuteCall, 0),
		ToolCalls:     make([]MockToolCall, 0),
		CustomResults: make(map[string]*core.TaskResult),
		TokenUsage:    250, // Default token usage per task
	}
}

// ExecuteTask implements TaskExecutor interface.
func (m *MockTaskExecutor) ExecuteTask(ctx context.Context, task *core.ExecutionTask, fullContext map[string]interface{}) (*core.TaskResult, error) {
	m.ExecuteCalls = append(m.ExecuteCalls, MockExecuteCall{
		Task:        task,
		FullContext: fullContext,
	})

	if m.ShouldTimeOut {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond): // Short timeout for testing
		}
	}

	if m.ShouldFailExecution || (m.FailOnTaskType != "" && task.Type == m.FailOnTaskType) {
		return &core.TaskResult{
			TaskID:       task.ID,
			Status:       core.TaskStatusFailed,
			TokensUsed:   m.TokenUsage / 2, // Use some tokens even on failure
			Duration:     50 * time.Millisecond,
			ErrorMessage: fmt.Sprintf("mock execution failure for task type: %s", task.Type),
			ToolsUsed:    []string{"mock_tool"},
			Confidence:   0.0,
			CompletedAt:  time.Now(),
		}, fmt.Errorf("mock execution failure")
	}

	// Check for custom result
	if customResult, exists := m.CustomResults[task.ID]; exists {
		return customResult, nil
	}

	// Default successful execution
	return &core.TaskResult{
		TaskID:     task.ID,
		Status:     core.TaskStatusCompleted,
		Output:     generateMockOutput(task),
		OutputRef:  fmt.Sprintf("data://mock_output_%s", task.ID),
		TokensUsed: m.TokenUsage,
		Duration:   100 * time.Millisecond,
		ToolsUsed:  []string{"mock_tool", determineToolForTask(task.Type)},
		Confidence: 0.85,
		CompletedAt: time.Now(),
	}, nil
}

// GetAvailableTools implements TaskExecutor interface.
func (m *MockTaskExecutor) GetAvailableTools(ctx context.Context) ([]string, error) {
	return []string{
		"mock_tool",
		"research",
		"analysis",
		"planning",
		"implementation",
		"testing",
		"validation",
	}, nil
}

// EstimateTokenUsage implements TaskExecutor interface.
func (m *MockTaskExecutor) EstimateTokenUsage(ctx context.Context, task *core.ExecutionTask) (int, error) {
	// Simple estimation based on task type and description length
	baseTokens := 200
	descriptionFactor := len(task.Description) / 10
	return baseTokens + descriptionFactor, nil
}

// MockContextLoader implements the ContextLoader interface for testing.
type MockContextLoader struct {
	// Configuration for mock behavior
	ShouldFailContextLoad bool
	ShouldFailRefResolve  bool

	// Captured calls for verification
	LoadTaskContextCalls      []MockLoadTaskContextCall
	LoadObjectiveContextCalls []MockLoadObjectiveContextCall
	ResolveReferenceCalls     []MockResolveReferenceCall

	// Customizable data
	ContextData map[string]interface{}
}

type MockLoadTaskContextCall struct {
	Task *core.ExecutionTask
}

type MockLoadObjectiveContextCall struct {
	ObjectiveID string
}

type MockResolveReferenceCall struct {
	Reference string
}

// NewMockContextLoader creates a mock context loader with default behavior.
func NewMockContextLoader() *MockContextLoader {
	return &MockContextLoader{
		LoadTaskContextCalls:      make([]MockLoadTaskContextCall, 0),
		LoadObjectiveContextCalls: make([]MockLoadObjectiveContextCall, 0),
		ResolveReferenceCalls:     make([]MockResolveReferenceCall, 0),
		ContextData:              make(map[string]interface{}),
	}
}

// LoadTaskContext implements ContextLoader interface.
func (m *MockContextLoader) LoadTaskContext(ctx context.Context, task *core.ExecutionTask) (map[string]interface{}, error) {
	m.LoadTaskContextCalls = append(m.LoadTaskContextCalls, MockLoadTaskContextCall{
		Task: task,
	})

	if m.ShouldFailContextLoad {
		return nil, fmt.Errorf("mock context load failure")
	}

	// Default context with task-specific data
	context := map[string]interface{}{
		"task_id":     task.ID,
		"task_type":   task.Type,
		"priority":    task.Context.Priority,
		"mock_data":   fmt.Sprintf("Mock context for task %s", task.ID),
		"input_data":  generateMockInputData(task.Context.InputRefs),
		"parameters":  task.Context.Parameters,
		"tools_available": []string{"mock_tool", "analysis", "research"},
	}

	// Add any custom context data
	for key, value := range m.ContextData {
		context[key] = value
	}

	return context, nil
}

// LoadObjectiveContext implements ContextLoader interface.
func (m *MockContextLoader) LoadObjectiveContext(ctx context.Context, objectiveID string) (map[string]interface{}, error) {
	m.LoadObjectiveContextCalls = append(m.LoadObjectiveContextCalls, MockLoadObjectiveContextCall{
		ObjectiveID: objectiveID,
	})

	if m.ShouldFailContextLoad {
		return nil, fmt.Errorf("mock objective context load failure")
	}

	return map[string]interface{}{
		"objective_id": objectiveID,
		"mock_data":   fmt.Sprintf("Mock context for objective %s", objectiveID),
		"resources":   []string{"database", "api", "documentation"},
	}, nil
}

// ResolveReference implements ContextLoader interface.
func (m *MockContextLoader) ResolveReference(ctx context.Context, ref string) (interface{}, error) {
	m.ResolveReferenceCalls = append(m.ResolveReferenceCalls, MockResolveReferenceCall{
		Reference: ref,
	})

	if m.ShouldFailRefResolve {
		return nil, fmt.Errorf("mock reference resolution failure")
	}

	// Parse reference and return appropriate mock data
	if strings.HasPrefix(ref, "data://") {
		return map[string]interface{}{
			"reference": ref,
			"data":     fmt.Sprintf("Mock data for reference: %s", ref),
			"quality":  0.9,
			"size":     1024,
		}, nil
	}

	return fmt.Sprintf("Mock resolved data for: %s", ref), nil
}

// MockLearningAgent implements the LearningAgent interface for testing.
type MockLearningAgent struct {
	// Configuration for mock behavior
	ShouldFailAnalysis    bool
	ShouldFailRefinement  bool
	ShouldFailEvaluation  bool

	// Captured calls for verification
	AnalyzeOutcomeCalls []MockAnalyzeOutcomeCall
	ProposeRefinementCalls []MockProposeRefinementCall
	EvaluateRefinementCalls []MockEvaluateRefinementCall

	// Customizable responses
	CustomAnalysis   *core.ExecutionAnalysis
	CustomRefinement *core.MethodRefinement
	CustomEvaluation *core.RefinementEvaluation
}

type MockAnalyzeOutcomeCall struct {
	Result *core.ExecutionResult
	Plan   *core.ExecutionPlan
	Method *core.Method
}

type MockProposeRefinementCall struct {
	Analysis *core.ExecutionAnalysis
	Method   *core.Method
}

type MockEvaluateRefinementCall struct {
	Original   *core.Method
	Refinement *core.MethodRefinement
}

// NewMockLearningAgent creates a mock learning agent with default behavior.
func NewMockLearningAgent() *MockLearningAgent {
	return &MockLearningAgent{
		AnalyzeOutcomeCalls:     make([]MockAnalyzeOutcomeCall, 0),
		ProposeRefinementCalls:  make([]MockProposeRefinementCall, 0),
		EvaluateRefinementCalls: make([]MockEvaluateRefinementCall, 0),
	}
}

// AnalyzeExecutionOutcome implements LearningAgent interface.
func (m *MockLearningAgent) AnalyzeExecutionOutcome(ctx context.Context, result *core.ExecutionResult, plan *core.ExecutionPlan, method *core.Method) (*core.ExecutionAnalysis, error) {
	m.AnalyzeOutcomeCalls = append(m.AnalyzeOutcomeCalls, MockAnalyzeOutcomeCall{
		Result: result,
		Plan:   plan,
		Method: method,
	})

	if m.ShouldFailAnalysis {
		return nil, fmt.Errorf("mock analysis failure")
	}

	if m.CustomAnalysis != nil {
		return m.CustomAnalysis, nil
	}

	// Default analysis based on execution result
	outcome := core.OutcomeSuccess
	if result.Status == core.ExecutionStatusFailed {
		outcome = core.OutcomeMethodFailure
	} else if result.Status == core.ExecutionStatusPartial {
		outcome = core.OutcomePartialSuccess
	}

	return &core.ExecutionAnalysis{
		OverallAssessment:   outcome,
		PrimaryFailureCause: "mock_failure_cause",
		MethodPerformanceIssues: []core.PerformanceIssue{
			{
				Category:      core.IssueComplexity,
				Description:   "Mock performance issue",
				AffectedSteps: []int{1},
				Severity:      5,
				SuggestedFix:  "Mock suggested fix",
			},
		},
		SuccessFactors:          []string{"mock_success_factor"},
		ImprovementOpportunities: []string{"mock_improvement"},
		ComplexityAssessment: core.ComplexityAnalysis{
			CurrentComplexityLevel:      6,
			ComplexityFactors:          []string{"mock_complexity_factor"},
			SimplificationOpportunities: []string{"mock_simplification"},
			OptimalComplexityLevel:     4,
		},
		ConfidenceLevel: 0.8,
	}, nil
}

// ProposeMethodRefinement implements LearningAgent interface.
func (m *MockLearningAgent) ProposeMethodRefinement(ctx context.Context, analysis *core.ExecutionAnalysis, method *core.Method) (*core.MethodRefinement, error) {
	m.ProposeRefinementCalls = append(m.ProposeRefinementCalls, MockProposeRefinementCall{
		Analysis: analysis,
		Method:   method,
	})

	if m.ShouldFailRefinement {
		return nil, fmt.Errorf("mock refinement failure")
	}

	if m.CustomRefinement != nil {
		return m.CustomRefinement, nil
	}

	// Default: suggest modification if complexity can be reduced
	if analysis.ComplexityAssessment.CurrentComplexityLevel > analysis.ComplexityAssessment.OptimalComplexityLevel {
		return &core.MethodRefinement{
			Type:        core.RefinementModify,
			NewApproach: simplifyApproach(method.Approach),
			Reasoning:   "Reduce complexity to improve success rate",
			ExpectedComplexityChange:      -2,
			ExpectedSuccessRateImprovement: 10.0,
			RequiredVersion:               bumpVersion(method.Version),
		}, nil
	}

	return &core.MethodRefinement{
		Type:      core.RefinementNone,
		Reasoning: "Method performance is acceptable",
	}, nil
}

// EvaluateRefinement implements LearningAgent interface.
func (m *MockLearningAgent) EvaluateRefinement(ctx context.Context, original *core.Method, refinement *core.MethodRefinement) (*core.RefinementEvaluation, error) {
	m.EvaluateRefinementCalls = append(m.EvaluateRefinementCalls, MockEvaluateRefinementCall{
		Original:   original,
		Refinement: refinement,
	})

	if m.ShouldFailEvaluation {
		return nil, fmt.Errorf("mock evaluation failure")
	}

	if m.CustomEvaluation != nil {
		return m.CustomEvaluation, nil
	}

	// Default evaluation: approve refinements that reduce complexity
	isImprovement := refinement.ExpectedComplexityChange < 0
	reducesComplexity := refinement.ExpectedComplexityChange < 0
	recommendation := core.RecommendApply
	if !isImprovement {
		recommendation = core.RecommendReject
	}

	return &core.RefinementEvaluation{
		IsImprovement:    isImprovement,
		ReducesComplexity: reducesComplexity,
		QualityScore:     7.5,
		Concerns:         []string{},
		Recommendation:   recommendation,
	}, nil
}

// Helper functions for mock implementations

func determineDomain(description string) string {
	description = strings.ToLower(description)
	if strings.Contains(description, "database") || strings.Contains(description, "query") {
		return "database"
	}
	if strings.Contains(description, "training") || strings.Contains(description, "education") {
		return "training"
	}
	if strings.Contains(description, "customer") || strings.Contains(description, "satisfaction") {
		return "customer_service"
	}
	return "general"
}

func getAlternativeMethodIDs(methods []*core.Method) []string {
	var ids []string
	for _, method := range methods {
		ids = append(ids, method.ID)
	}
	return ids
}

func createDefaultApproach(complexityLevel int) []core.ApproachStep {
	steps := []core.ApproachStep{
		{
			Description: "Analyze the problem and gather relevant data",
			Tools:       []string{"research", "analysis"},
			Heuristics:  []string{"be_systematic", "gather_complete_data"},
		},
	}

	if complexityLevel > 5 {
		steps = append(steps, core.ApproachStep{
			Description: "Design solution based on analysis findings",
			Tools:       []string{"planning", "design"},
			Heuristics:  []string{"consider_alternatives", "evaluate_trade_offs"},
		})
	}

	steps = append(steps, core.ApproachStep{
		Description: "Implement the solution and validate results",
		Tools:       []string{"implementation", "validation"},
		Heuristics:  []string{"test_incrementally", "measure_outcomes"},
	})

	return steps
}

func determineTaskType(description string) string {
	description = strings.ToLower(description)
	if strings.Contains(description, "analyze") || strings.Contains(description, "analysis") {
		return "analysis"
	}
	if strings.Contains(description, "gather") || strings.Contains(description, "collect") {
		return "data_collection"
	}
	if strings.Contains(description, "design") || strings.Contains(description, "create") {
		return "design"
	}
	if strings.Contains(description, "implement") || strings.Contains(description, "build") {
		return "implementation"
	}
	return "general"
}

func generateInputRefs(taskIndex int) []string {
	if taskIndex == 0 {
		return []string{"data://initial_input"}
	}
	return []string{fmt.Sprintf("data://task_%d_output", taskIndex)}
}

func generateMockOutput(task *core.ExecutionTask) map[string]interface{} {
	return map[string]interface{}{
		"task_id":    task.ID,
		"task_type":  task.Type,
		"result":     fmt.Sprintf("Mock output for %s", task.Description),
		"quality":    0.85,
		"confidence": 0.8,
		"metadata": map[string]interface{}{
			"execution_time": "100ms",
			"tokens_used":    250,
		},
	}
}

func determineToolForTask(taskType string) string {
	switch taskType {
	case "analysis":
		return "analytical_tool"
	case "data_collection":
		return "data_collector"
	case "design":
		return "design_tool"
	case "implementation":
		return "implementation_tool"
	default:
		return "generic_tool"
	}
}

func generateMockInputData(inputRefs []string) map[string]interface{} {
	inputData := make(map[string]interface{})
	for _, ref := range inputRefs {
		inputData[ref] = fmt.Sprintf("Mock data for reference: %s", ref)
	}
	return inputData
}

func simplifyApproach(original []core.ApproachStep) []core.ApproachStep {
	if len(original) <= 2 {
		return original
	}
	// Simplify by combining steps and removing redundancy
	return original[:len(original)-1]
}

func bumpVersion(version string) string {
	// Simple version bumping for mock
	if version == "1.0.0" {
		return "1.1.0"
	}
	return "2.0.0"
}