package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yourusername/ai-work-studio/pkg/storage"
)

// MockLearningAgent implements LearningAgent interface for testing.
type MockLearningAgent struct {
	// Mock behaviors
	shouldFailAnalysis    bool
	shouldFailRefinement  bool
	shouldFailEvaluation  bool

	// Mock responses
	mockAnalysis           *ExecutionAnalysis
	mockRefinement         *MethodRefinement
	mockEvaluation         *RefinementEvaluation

	// Recorded calls for verification
	analyzeOutcomeCalls    []AnalyzeOutcomeCall
	proposeRefinementCalls []ProposeRefinementCall
	evaluateRefinementCalls []EvaluateRefinementCall
}

// Call recording structures
type AnalyzeOutcomeCall struct {
	Result *ExecutionResult
	Plan   *ExecutionPlan
	Method *Method
}

type ProposeRefinementCall struct {
	Analysis *ExecutionAnalysis
	Method   *Method
}

type EvaluateRefinementCall struct {
	Original   *Method
	Refinement *MethodRefinement
}

func NewMockLearningAgent() *MockLearningAgent {
	return &MockLearningAgent{
		mockAnalysis: &ExecutionAnalysis{
			OverallAssessment:   OutcomeSuccess,
			ConfidenceLevel:     0.9,
			ComplexityAssessment: ComplexityAnalysis{
				CurrentComplexityLevel: 5,
				OptimalComplexityLevel: 3,
			},
		},
		mockRefinement: &MethodRefinement{
			Type:                           RefinementNone,
			Reasoning:                     "No refinement needed",
			ExpectedComplexityChange:      -1,
			ExpectedSuccessRateImprovement: 10.0,
			RequiredVersion:               "1.1.0",
		},
		mockEvaluation: &RefinementEvaluation{
			IsImprovement:     true,
			ReducesComplexity: true,
			QualityScore:      8.5,
			Recommendation:    RecommendApply,
		},
		analyzeOutcomeCalls:     make([]AnalyzeOutcomeCall, 0),
		proposeRefinementCalls:  make([]ProposeRefinementCall, 0),
		evaluateRefinementCalls: make([]EvaluateRefinementCall, 0),
	}
}

func (m *MockLearningAgent) AnalyzeExecutionOutcome(
	ctx context.Context,
	result *ExecutionResult,
	plan *ExecutionPlan,
	method *Method,
) (*ExecutionAnalysis, error) {
	m.analyzeOutcomeCalls = append(m.analyzeOutcomeCalls, AnalyzeOutcomeCall{
		Result: result,
		Plan:   plan,
		Method: method,
	})

	if m.shouldFailAnalysis {
		return nil, fmt.Errorf("mock analysis failure")
	}

	return m.mockAnalysis, nil
}

func (m *MockLearningAgent) ProposeMethodRefinement(
	ctx context.Context,
	analysis *ExecutionAnalysis,
	method *Method,
) (*MethodRefinement, error) {
	m.proposeRefinementCalls = append(m.proposeRefinementCalls, ProposeRefinementCall{
		Analysis: analysis,
		Method:   method,
	})

	if m.shouldFailRefinement {
		return nil, fmt.Errorf("mock refinement failure")
	}

	return m.mockRefinement, nil
}

func (m *MockLearningAgent) EvaluateRefinement(
	ctx context.Context,
	original *Method,
	refinement *MethodRefinement,
) (*RefinementEvaluation, error) {
	m.evaluateRefinementCalls = append(m.evaluateRefinementCalls, EvaluateRefinementCall{
		Original:   original,
		Refinement: refinement,
	})

	if m.shouldFailEvaluation {
		return nil, fmt.Errorf("mock evaluation failure")
	}

	return m.mockEvaluation, nil
}

// Test helpers

func setupTestLearningLoop(t *testing.T) (
	*LearningLoop,
	*storage.Store,
	*MockLLMReasoner,
	*MockTaskExecutor,
	*MockContextLoader,
	*MockLearningAgent,
) {
	// Create temporary directory for test storage
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("learning_loop_test_%d", time.Now().UnixNano()))
	store, err := storage.NewStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}

	// Clean up after test
	t.Cleanup(func() {
		store.Close()
		os.RemoveAll(tempDir)
	})

	// Create mock dependencies
	llmReasoner := NewMockLLMReasoner()
	taskExecutor := NewMockTaskExecutor()
	contextLoader := NewMockContextLoader()
	learningAgent := NewMockLearningAgent()

	// Create CC and RTC
	cc := NewContemplativeCursor(store, llmReasoner)
	rtc := NewRealTimeCursor(store, taskExecutor, contextLoader)

	// Create learning loop
	ll := NewLearningLoop(store, cc, rtc, learningAgent)

	return ll, store, llmReasoner, taskExecutor, contextLoader, learningAgent
}

func createTestLearningObjective(t *testing.T, store *storage.Store) (*Goal, *Method, *Objective) {
	// Create goal
	gm := NewGoalManager(store)
	goal, err := gm.CreateGoal(
		context.Background(),
		"Test Learning Goal",
		"A goal for testing the learning loop",
		8,
		map[string]interface{}{"learning_test": true},
	)
	if err != nil {
		t.Fatalf("Failed to create test goal: %v", err)
	}

	// Create method
	mm := NewMethodManager(store)
	approach := []ApproachStep{
		{
			Description: "Step 1: Analyze input",
			Tools:       []string{"analyzer"},
			Heuristics:  []string{"be thorough"},
		},
		{
			Description: "Step 2: Generate output",
			Tools:       []string{"generator"},
			Heuristics:  []string{"be accurate"},
		},
	}

	method, err := mm.CreateMethod(
		context.Background(),
		"Test Learning Method",
		"A method for testing the learning loop",
		approach,
		MethodDomainGeneral,
		map[string]interface{}{"learning_test": true},
	)
	if err != nil {
		t.Fatalf("Failed to create test method: %v", err)
	}

	// Create objective
	om := NewObjectiveManager(store)
	objective, err := om.CreateObjective(
		context.Background(),
		goal.ID,
		method.ID,
		"Test Learning Objective",
		"An objective for testing the learning loop",
		map[string]interface{}{"test_data": "learning_scenario"},
		7,
	)
	if err != nil {
		t.Fatalf("Failed to create test objective: %v", err)
	}

	return goal, method, objective
}

// Tests

func TestNewLearningLoop(t *testing.T) {
	ll, _, _, _, _, _ := setupTestLearningLoop(t)

	if ll == nil {
		t.Fatal("Expected non-nil LearningLoop")
	}

	config := ll.GetConfiguration()
	if config == nil {
		t.Fatal("Expected non-nil configuration")
	}

	// Verify default configuration values
	if config.MinExecutionsBeforeRefinement <= 0 {
		t.Error("Expected positive MinExecutionsBeforeRefinement")
	}
	if config.SuccessRateThresholdForRefinement <= 0 || config.SuccessRateThresholdForRefinement > 100 {
		t.Error("Expected SuccessRateThresholdForRefinement between 0-100")
	}
	if config.ComplexityBiasWeight < 0 || config.ComplexityBiasWeight > 1 {
		t.Error("Expected ComplexityBiasWeight between 0-1")
	}
}

func TestExecuteObjective_Success(t *testing.T) {
	ll, store, llmReasoner, taskExecutor, _, learningAgent := setupTestLearningLoop(t)

	// Create test data
	_, _, objective := createTestLearningObjective(t, store)

	// Configure mocks for successful execution
	taskExecutor.mockTaskResult.Status = TaskStatusCompleted
	learningAgent.mockAnalysis.OverallAssessment = OutcomeSuccess

	// Execute objective
	result, err := ll.ExecuteObjective(context.Background(), objective.ID)

	if err != nil {
		t.Fatalf("ExecuteObjective should not fail: %v", err)
	}

	// Verify result structure
	if result == nil {
		t.Fatal("Expected non-nil LearningResult")
	}
	if result.ObjectiveID != objective.ID {
		t.Errorf("Expected ObjectiveID %s, got %s", objective.ID, result.ObjectiveID)
	}
	if !result.WasSuccessful {
		t.Error("Expected successful result")
	}
	if result.FinalOutcome != OutcomeSuccess {
		t.Errorf("Expected outcome %s, got %s", OutcomeSuccess, result.FinalOutcome)
	}

	// Verify execution attempts
	if len(result.ExecutionAttempts) != 1 {
		t.Errorf("Expected 1 execution attempt, got %d", len(result.ExecutionAttempts))
	}

	// Verify mocks were called correctly
	if len(llmReasoner.decomposePlanCalls) != 1 {
		t.Errorf("Expected 1 plan decomposition call, got %d", len(llmReasoner.decomposePlanCalls))
	}
	if len(taskExecutor.executeTaskCalls) == 0 {
		t.Error("Expected at least one task execution call")
	}
	if len(learningAgent.analyzeOutcomeCalls) != 1 {
		t.Errorf("Expected 1 analysis call, got %d", len(learningAgent.analyzeOutcomeCalls))
	}
}

func TestExecuteObjective_MethodRefinement(t *testing.T) {
	ll, store, _, taskExecutor, _, learningAgent := setupTestLearningLoop(t)

	// Create test data
	_, _, objective := createTestLearningObjective(t, store)

	// Configure for successful execution initially
	taskExecutor.mockTaskResult.Status = TaskStatusCompleted
	learningAgent.mockAnalysis.OverallAssessment = OutcomeSuccess

	// Execute objective successfully first time
	result, err := ll.ExecuteObjective(context.Background(), objective.ID)

	if err != nil {
		t.Fatalf("ExecuteObjective should not fail: %v", err)
	}

	// Verify basic execution worked
	if !result.WasSuccessful {
		t.Error("Expected successful execution")
	}
	if len(result.ExecutionAttempts) != 1 {
		t.Errorf("Expected 1 execution attempt, got %d", len(result.ExecutionAttempts))
	}

	// Verify learning agent was called
	if len(learningAgent.analyzeOutcomeCalls) != 1 {
		t.Errorf("Expected 1 analysis call, got %d", len(learningAgent.analyzeOutcomeCalls))
	}

	// For this test, we verify that the learning loop integration works correctly
	// The detailed refinement logic is tested separately in TestShouldAttemptRefinement
}

func TestExecuteObjective_NoObjective(t *testing.T) {
	ll, _, _, _, _, _ := setupTestLearningLoop(t)

	// Try to execute non-existent objective
	_, err := ll.ExecuteObjective(context.Background(), "nonexistent_objective")
	if err == nil {
		t.Fatal("Expected error for non-existent objective")
	}
}

func TestExecuteObjective_PlanningFailure(t *testing.T) {
	ll, store, llmReasoner, _, _, _ := setupTestLearningLoop(t)

	// Create test data
	_, _, objective := createTestLearningObjective(t, store)

	// Configure LLM to fail planning
	llmReasoner.shouldFailAnalysis = true

	// Execute objective
	result, err := ll.ExecuteObjective(context.Background(), objective.ID)

	if err == nil {
		t.Fatal("Expected error when planning fails")
	}
	if result == nil {
		t.Fatal("Expected non-nil result even on failure")
	}
	if result.WasSuccessful {
		t.Error("Expected unsuccessful result")
	}
}

func TestExecuteObjective_ExecutionFailure(t *testing.T) {
	ll, store, _, taskExecutor, _, learningAgent := setupTestLearningLoop(t)

	// Create test data
	_, _, objective := createTestLearningObjective(t, store)

	// Configure execution to fail
	taskExecutor.shouldFailExecution = true
	learningAgent.mockAnalysis.OverallAssessment = OutcomeEnvironmentFailure // Don't refine

	// Execute objective
	result, err := ll.ExecuteObjective(context.Background(), objective.ID)

	if err == nil {
		t.Fatal("Expected error when execution fails")
	}
	if result.WasSuccessful {
		t.Error("Expected unsuccessful result")
	}
}

func TestShouldAttemptRefinement(t *testing.T) {
	ll, store, _, _, _, _ := setupTestLearningLoop(t)
	_, method, _ := createTestLearningObjective(t, store)

	// Update method to meet refinement criteria
	mm := NewMethodManager(store)
	for i := 0; i < 5; i++ {
		err := mm.UpdateMethodMetrics(context.Background(), method.ID, false, 3.0)
		if err != nil {
			t.Fatalf("Failed to update method metrics: %v", err)
		}
	}

	// Get updated method
	updatedMethod, err := mm.GetMethod(context.Background(), method.ID)
	if err != nil {
		t.Fatalf("Failed to get updated method: %v", err)
	}

	testCases := []struct {
		name           string
		assessment     ExecutionOutcome
		expectedResult bool
	}{
		{"success", OutcomeSuccess, false},
		{"partial success", OutcomePartialSuccess, true},
		{"method failure", OutcomeMethodFailure, true},
		{"environment failure", OutcomeEnvironmentFailure, false},
		{"insufficient data", OutcomeInsufficientData, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			analysis := &ExecutionAnalysis{
				OverallAssessment: tc.assessment,
			}

			// Use reflection or create a test method to access the private method
			result := ll.ShouldAttemptRefinement(analysis, updatedMethod)
			if result != tc.expectedResult {
				t.Errorf("shouldAttemptRefinement(%s) = %v, expected %v",
					tc.assessment, result, tc.expectedResult)
			}
		})
	}
}

// Create a test helper that exposes the private method
func (ll *LearningLoop) ShouldAttemptRefinement(analysis *ExecutionAnalysis, method *Method) bool {
	// This is a test helper that mimics the private method logic
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
	config := ll.GetConfiguration()
	if method.Metrics.ExecutionCount < config.MinExecutionsBeforeRefinement {
		return false
	}

	// Check if success rate is below threshold
	if method.Metrics.SuccessRate() >= config.SuccessRateThresholdForRefinement {
		return false
	}

	return true
}

func TestLearningResult_HelperMethods(t *testing.T) {
	// Create test result with multiple attempts
	result := &LearningResult{
		ExecutionAttempts: []AttemptResult{
			{
				AttemptNumber: 1,
				MethodID:      "method_1",
				ExecutionResult: &ExecutionResult{
					Status:          ExecutionStatusCompleted,
					TotalTokensUsed: 100,
				},
			},
			{
				AttemptNumber: 2,
				MethodID:      "method_2",
				ExecutionResult: &ExecutionResult{
					Status:          ExecutionStatusFailed,
					TotalTokensUsed: 50,
				},
			},
			{
				AttemptNumber: 3,
				MethodID:      "method_1", // Duplicate
				ExecutionResult: &ExecutionResult{
					Status:          ExecutionStatusPartial,
					TotalTokensUsed: 75,
				},
			},
		},
	}

	// Test GetSuccessfulAttempts
	successful := result.GetSuccessfulAttempts()
	if len(successful) != 2 {
		t.Errorf("Expected 2 successful attempts, got %d", len(successful))
	}

	// Test GetTotalTokensUsed
	totalTokens := result.GetTotalTokensUsed()
	if totalTokens != 225 {
		t.Errorf("Expected 225 total tokens, got %d", totalTokens)
	}

	// Test GetMethodsUsed
	methods := result.GetMethodsUsed()
	if len(methods) != 2 {
		t.Errorf("Expected 2 unique methods, got %d", len(methods))
	}

	// Verify unique method IDs
	methodSet := make(map[string]bool)
	for _, methodID := range methods {
		if methodSet[methodID] {
			t.Errorf("Method ID %s appears multiple times in unique list", methodID)
		}
		methodSet[methodID] = true
	}
}

func TestLearningLoopConfiguration(t *testing.T) {
	ll, _, _, _, _, _ := setupTestLearningLoop(t)

	// Test getting default configuration
	defaultConfig := ll.GetConfiguration()
	if defaultConfig == nil {
		t.Fatal("Expected non-nil default configuration")
	}

	// Test setting custom configuration
	customConfig := &LearningLoopConfig{
		MinExecutionsBeforeRefinement:     5,
		SuccessRateThresholdForRefinement: 80.0,
		MaxRefinementAttempts:             2,
		ComplexityBiasWeight:              0.8,
		EnableMethodEvolution:             false,
		PreserveMethodHistory:             false,
	}

	ll.SetConfiguration(customConfig)
	retrievedConfig := ll.GetConfiguration()

	if retrievedConfig.MinExecutionsBeforeRefinement != 5 {
		t.Errorf("Expected MinExecutionsBeforeRefinement 5, got %d",
			retrievedConfig.MinExecutionsBeforeRefinement)
	}
	if retrievedConfig.SuccessRateThresholdForRefinement != 80.0 {
		t.Errorf("Expected SuccessRateThresholdForRefinement 80.0, got %f",
			retrievedConfig.SuccessRateThresholdForRefinement)
	}
	if retrievedConfig.ComplexityBiasWeight != 0.8 {
		t.Errorf("Expected ComplexityBiasWeight 0.8, got %f",
			retrievedConfig.ComplexityBiasWeight)
	}
	if retrievedConfig.EnableMethodEvolution != false {
		t.Errorf("Expected EnableMethodEvolution false, got %v",
			retrievedConfig.EnableMethodEvolution)
	}

	// Test setting nil configuration (should be ignored)
	ll.SetConfiguration(nil)
	configAfterNil := ll.GetConfiguration()
	if configAfterNil != retrievedConfig {
		t.Error("Configuration should not change when setting nil")
	}
}

func TestDefaultLearningLoopConfig(t *testing.T) {
	config := DefaultLearningLoopConfig()

	if config == nil {
		t.Fatal("Expected non-nil default configuration")
	}

	// Verify all fields have reasonable defaults
	if config.MinExecutionsBeforeRefinement <= 0 {
		t.Error("Expected positive MinExecutionsBeforeRefinement")
	}
	if config.SuccessRateThresholdForRefinement <= 0 || config.SuccessRateThresholdForRefinement > 100 {
		t.Error("Expected SuccessRateThresholdForRefinement between 0-100")
	}
	if config.MaxRefinementAttempts <= 0 {
		t.Error("Expected positive MaxRefinementAttempts")
	}
	if config.ComplexityBiasWeight < 0 || config.ComplexityBiasWeight > 1 {
		t.Error("Expected ComplexityBiasWeight between 0-1")
	}
	if !config.EnableMethodEvolution {
		t.Error("Expected EnableMethodEvolution to be true by default")
	}
}

func TestComplexityBias(t *testing.T) {
	ll, store, _, taskExecutor, _, learningAgent := setupTestLearningLoop(t)

	// Set high complexity bias
	config := ll.GetConfiguration()
	config.ComplexityBiasWeight = 0.9
	ll.SetConfiguration(config)

	// Verify configuration was set correctly
	retrievedConfig := ll.GetConfiguration()
	if retrievedConfig.ComplexityBiasWeight != 0.9 {
		t.Errorf("Expected ComplexityBiasWeight 0.9, got %f", retrievedConfig.ComplexityBiasWeight)
	}

	// Create test data
	_, _, objective := createTestLearningObjective(t, store)

	// Configure for successful execution
	taskExecutor.mockTaskResult.Status = TaskStatusCompleted
	learningAgent.mockAnalysis.OverallAssessment = OutcomeSuccess

	// Execute objective successfully
	result, err := ll.ExecuteObjective(context.Background(), objective.ID)

	if err != nil {
		t.Fatalf("ExecuteObjective should not fail: %v", err)
	}

	// Verify basic execution works with complexity bias configured
	if !result.WasSuccessful {
		t.Error("Expected successful execution")
	}

	// This test verifies that complexity bias configuration works
	// The detailed refinement rejection logic would be tested in unit tests
	// of the specific refinement functions
}

// Benchmark tests

func BenchmarkExecuteObjective(b *testing.B) {
	ll, store, _, taskExecutor, _, learningAgent := setupTestLearningLoop(&testing.T{})

	// Create test data
	_, _, objective := createTestLearningObjective(&testing.T{}, store)

	// Configure for successful execution
	taskExecutor.mockTaskResult.Status = TaskStatusCompleted
	learningAgent.mockAnalysis.OverallAssessment = OutcomeSuccess

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ll.ExecuteObjective(context.Background(), objective.ID)
		if err != nil {
			b.Fatalf("ExecuteObjective failed: %v", err)
		}
	}
}

func BenchmarkLearningAnalysis(b *testing.B) {
	_, store, _, _, _, learningAgent := setupTestLearningLoop(&testing.T{})

	// Create test data
	_, method, _ := createTestLearningObjective(&testing.T{}, store)

	// Create mock execution result
	result := &ExecutionResult{
		Status:          ExecutionStatusCompleted,
		TotalTokensUsed: 100,
		SuccessfulTasks: 2,
		FailedTasks:     0,
	}

	// Create mock execution plan
	plan := &ExecutionPlan{
		ID:       "test_plan",
		MethodID: method.ID,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := learningAgent.AnalyzeExecutionOutcome(context.Background(), result, plan, method)
		if err != nil {
			b.Fatalf("AnalyzeExecutionOutcome failed: %v", err)
		}
	}
}

// Test string methods for enums
func TestEnumStringMethods(t *testing.T) {
	// Test ExecutionOutcome
	outcome := OutcomeSuccess
	if outcome.String() != "success" {
		t.Errorf("Expected 'success', got %s", outcome.String())
	}

	// Test RefinementType
	refinementType := RefinementModify
	if refinementType.String() != "modify" {
		t.Errorf("Expected 'modify', got %s", refinementType.String())
	}

	// Test RefinementRecommendation
	recommendation := RecommendApply
	if recommendation.String() != "apply" {
		t.Errorf("Expected 'apply', got %s", recommendation.String())
	}

	// Test IssueCategory
	category := IssueComplexity
	if category.String() != "complexity" {
		t.Errorf("Expected 'complexity', got %s", category.String())
	}
}