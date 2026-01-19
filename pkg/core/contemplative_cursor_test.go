package core

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/yourusername/ai-work-studio/pkg/storage"
)

// MockLLMReasoner implements LLMReasoner interface for testing.
type MockLLMReasoner struct {
	// Mock behaviors
	shouldFailAnalysis bool
	shouldFailSelection bool
	shouldFailDesign   bool
	shouldFailDecompose bool

	// Recorded calls for verification
	analyzeObjectiveCalls []AnalyzeObjectiveCall
	selectMethodCalls     []SelectMethodCall
	designMethodCalls     []DesignMethodCall
	decomposePlanCalls    []DecomposePlanCall
}

// Call recording structures
type AnalyzeObjectiveCall struct {
	Objective   *Objective
	GoalContext *Goal
}

type SelectMethodCall struct {
	Analysis *ObjectiveAnalysis
	Methods  []*Method
}

type DesignMethodCall struct {
	Analysis *ObjectiveAnalysis
}

type DecomposePlanCall struct {
	Objective *Objective
	Method    *Method
}

func NewMockLLMReasoner() *MockLLMReasoner {
	return &MockLLMReasoner{
		analyzeObjectiveCalls: make([]AnalyzeObjectiveCall, 0),
		selectMethodCalls:     make([]SelectMethodCall, 0),
		designMethodCalls:     make([]DesignMethodCall, 0),
		decomposePlanCalls:    make([]DecomposePlanCall, 0),
	}
}

func (m *MockLLMReasoner) AnalyzeObjective(ctx context.Context, objective *Objective, goalContext *Goal) (*ObjectiveAnalysis, error) {
	m.analyzeObjectiveCalls = append(m.analyzeObjectiveCalls, AnalyzeObjectiveCall{
		Objective:   objective,
		GoalContext: goalContext,
	})

	if m.shouldFailAnalysis {
		return nil, fmt.Errorf("mock analysis failure")
	}

	// Return a realistic mock analysis
	return &ObjectiveAnalysis{
		ComplexityLevel:      5,
		RequiredCapabilities: []string{"analysis", "synthesis"},
		KeyChallenges:        []string{"data quality", "time constraints"},
		SuccessCriteria:      []string{"accurate results", "timely delivery"},
		EstimatedTokenBudget: 1000,
		RecommendedApproach:  "systematic analysis with validation",
		DomainContext:        "data_analysis",
	}, nil
}

func (m *MockLLMReasoner) SelectMethod(ctx context.Context, analysis *ObjectiveAnalysis, methods []*Method) (*MethodSelection, error) {
	m.selectMethodCalls = append(m.selectMethodCalls, SelectMethodCall{
		Analysis: analysis,
		Methods:  methods,
	})

	if m.shouldFailSelection {
		return nil, fmt.Errorf("mock selection failure")
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("no methods to select from")
	}

	// Select first method for simplicity
	return &MethodSelection{
		SelectedMethodID:     methods[0].ID,
		ConfidenceLevel:      0.8,
		SelectionReason:      "best match for requirements",
		RequiredAdaptations:  []string{"adjust parameters"},
		AlternativeMethodIDs: []string{},
	}, nil
}

func (m *MockLLMReasoner) DesignMethod(ctx context.Context, analysis *ObjectiveAnalysis) (*Method, error) {
	m.designMethodCalls = append(m.designMethodCalls, DesignMethodCall{
		Analysis: analysis,
	})

	if m.shouldFailDesign {
		return nil, fmt.Errorf("mock design failure")
	}

	// Return a realistic mock method
	approach := []ApproachStep{
		{
			Description: "Analyze the problem",
			Tools:       []string{"analysis_tool"},
			Heuristics:  []string{"break down into components"},
		},
		{
			Description: "Generate solution",
			Tools:       []string{"generator_tool"},
			Heuristics:  []string{"iterative refinement"},
		},
		{
			Description: "Validate results",
			Tools:       []string{"validation_tool"},
			Heuristics:  []string{"cross-check with criteria"},
		},
	}

	return &Method{
		ID:          "mock_method_" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Name:        "Mock Analysis Method",
		Description: "A method designed for mock testing",
		Approach:    approach,
		Domain:      MethodDomainGeneral,
		Version:     "1.0.0",
		Status:      MethodStatusActive,
		Metrics: SuccessMetrics{
			ExecutionCount: 0,
			SuccessCount:   0,
			LastUsed:       time.Time{},
			AverageRating:  0.0,
		},
		UserContext: map[string]interface{}{
			"created_by": "mock_reasoner",
		},
		CreatedAt: time.Now(),
	}, nil
}

func (m *MockLLMReasoner) DecomposePlan(ctx context.Context, objective *Objective, method *Method) (*ExecutionPlan, error) {
	m.decomposePlanCalls = append(m.decomposePlanCalls, DecomposePlanCall{
		Objective: objective,
		Method:    method,
	})

	if m.shouldFailDecompose {
		return nil, fmt.Errorf("mock decompose failure")
	}

	// Create a realistic mock execution plan
	tasks := []ExecutionTask{
		{
			ID:          "task_1",
			Type:        "analysis",
			Description: "Analyze the input data",
			Context: TaskContext{
				InputRefs:   []string{"input_data_ref"},
				OutputRef:   "analysis_output",
				Parameters:  map[string]interface{}{"depth": "detailed"},
				TokenBudget: 300,
				Priority:    8,
			},
			MethodStepIndex: 0,
			EstimatedTokens: 300,
			CreatedAt:       time.Now(),
		},
		{
			ID:          "task_2",
			Type:        "synthesis",
			Description: "Synthesize the findings",
			Context: TaskContext{
				InputRefs:   []string{"analysis_output"},
				OutputRef:   "final_result",
				Parameters:  map[string]interface{}{"format": "summary"},
				TokenBudget: 200,
				Priority:    7,
			},
			MethodStepIndex: 1,
			EstimatedTokens: 200,
			CreatedAt:       time.Now(),
		},
	}

	dependencies := []TaskDependency{
		{
			TaskID:          "task_2",
			DependsOnTaskID: "task_1",
			Reason:          "synthesis requires analysis output",
		},
	}

	return &ExecutionPlan{
		ID:                   "", // Will be set by CC
		ObjectiveID:          objective.ID,
		MethodID:             method.ID,
		Title:                "Mock execution plan for " + objective.Title,
		Tasks:                tasks,
		Dependencies:         dependencies,
		TotalEstimatedTokens: 0, // Will be calculated by CC
		CreatedBy:            "", // Will be set by CC
		CreatedAt:            time.Time{}, // Will be set by CC
	}, nil
}

// Test helper to create a temporary store
func createTestStore(t *testing.T) *storage.Store {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "test_data")

	store, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}

	return store
}

// Test helper to create a test goal
func createTestGoal(t *testing.T, store *storage.Store) *Goal {
	gm := NewGoalManager(store)
	goal, err := gm.CreateGoal(
		context.Background(),
		"Test Goal",
		"A goal created for testing purposes",
		7,
		map[string]interface{}{"test": true},
	)
	if err != nil {
		t.Fatalf("Failed to create test goal: %v", err)
	}
	return goal
}

// Test helper to create a test method
func createTestMethod(t *testing.T, store *storage.Store) *Method {
	mm := NewMethodManager(store)

	approach := []ApproachStep{
		{
			Description: "Step 1: Analyze",
			Tools:       []string{"analyzer"},
			Heuristics:  []string{"be thorough"},
		},
		{
			Description: "Step 2: Act",
			Tools:       []string{"executor"},
			Heuristics:  []string{"be precise"},
		},
	}

	method, err := mm.CreateMethod(
		context.Background(),
		"Test Method",
		"A method created for testing purposes",
		approach,
		MethodDomainGeneral,
		map[string]interface{}{"test": true},
	)
	if err != nil {
		t.Fatalf("Failed to create test method: %v", err)
	}

	// Update metrics to make it appear successful
	err = mm.UpdateMethodMetrics(context.Background(), method.ID, true, 8.0)
	if err != nil {
		t.Fatalf("Failed to update method metrics: %v", err)
	}
	err = mm.UpdateMethodMetrics(context.Background(), method.ID, true, 8.5)
	if err != nil {
		t.Fatalf("Failed to update method metrics: %v", err)
	}

	return method
}

// Test helper to create a test objective
func createTestObjective(t *testing.T, store *storage.Store, goalID, methodID string) *Objective {
	om := NewObjectiveManager(store)
	objective, err := om.CreateObjective(
		context.Background(),
		goalID,
		methodID,
		"Test Objective",
		"An objective created for testing purposes",
		map[string]interface{}{"test_context": "value"},
		6,
	)
	if err != nil {
		t.Fatalf("Failed to create test objective: %v", err)
	}
	return objective
}

func TestNewContemplativeCursor(t *testing.T) {
	store := createTestStore(t)
	reasoner := NewMockLLMReasoner()

	cc := NewContemplativeCursor(store, reasoner)

	if cc == nil {
		t.Fatal("Expected non-nil ContemplativeCursor")
	}
	if cc.store != store {
		t.Fatal("Expected store to be set correctly")
	}
	if cc.reasoner != reasoner {
		t.Fatal("Expected reasoner to be set correctly")
	}
}

func TestCreateExecutionPlan_WithCachedMethod(t *testing.T) {
	store := createTestStore(t)
	reasoner := NewMockLLMReasoner()
	cc := NewContemplativeCursor(store, reasoner)

	// Create test data
	goal := createTestGoal(t, store)
	method := createTestMethod(t, store)
	objective := createTestObjective(t, store, goal.ID, method.ID)

	// Create execution plan
	plan, err := cc.CreateExecutionPlan(context.Background(), objective.ID)
	if err != nil {
		t.Fatalf("CreateExecutionPlan failed: %v", err)
	}

	// Verify plan structure
	if plan == nil {
		t.Fatal("Expected non-nil execution plan")
	}
	if plan.ObjectiveID != objective.ID {
		t.Errorf("Expected ObjectiveID %s, got %s", objective.ID, plan.ObjectiveID)
	}
	if plan.MethodID == "" {
		t.Error("Expected non-empty MethodID")
	}
	if len(plan.Tasks) == 0 {
		t.Error("Expected at least one task in the plan")
	}
	if plan.CreatedBy != "contemplative_cursor" {
		t.Errorf("Expected CreatedBy 'contemplative_cursor', got %s", plan.CreatedBy)
	}
	if plan.TotalEstimatedTokens <= 0 {
		t.Error("Expected positive TotalEstimatedTokens")
	}

	// Verify LLM reasoner was called appropriately
	if len(reasoner.analyzeObjectiveCalls) != 1 {
		t.Errorf("Expected 1 AnalyzeObjective call, got %d", len(reasoner.analyzeObjectiveCalls))
	}
	if len(reasoner.decomposePlanCalls) != 1 {
		t.Errorf("Expected 1 DecomposePlan call, got %d", len(reasoner.decomposePlanCalls))
	}
}

func TestCreateExecutionPlan_WithNewMethod(t *testing.T) {
	store := createTestStore(t)
	reasoner := NewMockLLMReasoner()
	cc := NewContemplativeCursor(store, reasoner)

	// Create test data WITHOUT a cached method
	goal := createTestGoal(t, store)
	// Create a dummy method just to satisfy the objective creation,
	// but we'll ensure it won't be selected
	dummyMethod := createTestMethod(t, store)
	// Mark it as deprecated so it won't be selected
	mm := NewMethodManager(store)
	deprecated := MethodStatusDeprecated
	updates := MethodUpdates{Status: &deprecated}
	_, err := mm.UpdateMethod(context.Background(), dummyMethod.ID, updates)
	if err != nil {
		t.Fatalf("Failed to deprecate dummy method: %v", err)
	}

	objective := createTestObjective(t, store, goal.ID, dummyMethod.ID)

	// Create execution plan - should design new method
	plan, err := cc.CreateExecutionPlan(context.Background(), objective.ID)
	if err != nil {
		t.Fatalf("CreateExecutionPlan failed: %v", err)
	}

	// Verify plan was created
	if plan == nil {
		t.Fatal("Expected non-nil execution plan")
	}
	if len(plan.Tasks) == 0 {
		t.Error("Expected at least one task in the plan")
	}

	// Verify LLM reasoner was called to design new method
	if len(reasoner.designMethodCalls) != 1 {
		t.Errorf("Expected 1 DesignMethod call, got %d", len(reasoner.designMethodCalls))
	}

	// Verify new method was cached
	allMethods, err := mm.ListMethods(context.Background(), MethodFilter{})
	if err != nil {
		t.Fatalf("Failed to list methods: %v", err)
	}

	// Should have the original dummy method plus the new one
	activeMethods := 0
	for _, method := range allMethods {
		if method.Status == MethodStatusActive {
			activeMethods++
		}
	}
	if activeMethods != 1 {
		t.Errorf("Expected 1 active method after caching, got %d", activeMethods)
	}
}

func TestCreateExecutionPlan_ObjectiveNotFound(t *testing.T) {
	store := createTestStore(t)
	reasoner := NewMockLLMReasoner()
	cc := NewContemplativeCursor(store, reasoner)

	// Try to create plan for non-existent objective
	_, err := cc.CreateExecutionPlan(context.Background(), "nonexistent_objective")
	if err == nil {
		t.Fatal("Expected error for non-existent objective")
	}
}

func TestCreateExecutionPlan_LLMFailure(t *testing.T) {
	store := createTestStore(t)
	reasoner := NewMockLLMReasoner()
	reasoner.shouldFailAnalysis = true // Make LLM fail
	cc := NewContemplativeCursor(store, reasoner)

	// Create test data
	goal := createTestGoal(t, store)
	method := createTestMethod(t, store)
	objective := createTestObjective(t, store, goal.ID, method.ID)

	// Try to create execution plan - should fail
	_, err := cc.CreateExecutionPlan(context.Background(), objective.ID)
	if err == nil {
		t.Fatal("Expected error when LLM analysis fails")
	}
}

func TestAnalyzeObjectiveComplexity(t *testing.T) {
	store := createTestStore(t)
	reasoner := NewMockLLMReasoner()
	cc := NewContemplativeCursor(store, reasoner)

	// Create test data
	goal := createTestGoal(t, store)
	method := createTestMethod(t, store)

	// Test with simple objective
	simpleObj := createTestObjective(t, store, goal.ID, method.ID)
	complexity, err := cc.AnalyzeObjectiveComplexity(context.Background(), simpleObj.ID)
	if err != nil {
		t.Fatalf("AnalyzeObjectiveComplexity failed: %v", err)
	}
	if complexity < 1 || complexity > 10 {
		t.Errorf("Expected complexity between 1-10, got %d", complexity)
	}

	// Test with complex objective
	om := NewObjectiveManager(store)
	complexObj, err := om.CreateObjective(
		context.Background(),
		goal.ID,
		method.ID,
		"Complex High Priority Task",
		"This is a very long and detailed description that exceeds 200 characters. It contains multiple requirements, complex dependencies, and sophisticated processing needs that will require careful analysis and substantial computational resources to complete successfully.",
		map[string]interface{}{
			"complexity": "high",
			"dependencies": []string{"dep1", "dep2", "dep3"},
			"requirements": []string{"req1", "req2", "req3", "req4", "req5", "req6"},
		},
		9, // High priority
	)
	if err != nil {
		t.Fatalf("Failed to create complex objective: %v", err)
	}

	complexComplexity, err := cc.AnalyzeObjectiveComplexity(context.Background(), complexObj.ID)
	if err != nil {
		t.Fatalf("AnalyzeObjectiveComplexity failed for complex objective: %v", err)
	}

	// Complex objective should have higher complexity
	if complexComplexity <= complexity {
		t.Errorf("Expected complex objective to have higher complexity. Simple: %d, Complex: %d", complexity, complexComplexity)
	}
}

func TestEstimateTokenBudget(t *testing.T) {
	store := createTestStore(t)
	reasoner := NewMockLLMReasoner()
	cc := NewContemplativeCursor(store, reasoner)

	// Create test data
	goal := createTestGoal(t, store)
	method := createTestMethod(t, store)
	objective := createTestObjective(t, store, goal.ID, method.ID)

	// Estimate token budget
	budget, err := cc.EstimateTokenBudget(context.Background(), objective.ID)
	if err != nil {
		t.Fatalf("EstimateTokenBudget failed: %v", err)
	}

	if budget <= 0 {
		t.Errorf("Expected positive token budget, got %d", budget)
	}

	// Budget should be reasonable (not too small or too large)
	if budget < 500 || budget > 5000 {
		t.Errorf("Expected token budget between 500-5000, got %d", budget)
	}
}

func TestGetMethodCacheStatistics(t *testing.T) {
	store := createTestStore(t)
	reasoner := NewMockLLMReasoner()
	cc := NewContemplativeCursor(store, reasoner)

	// Create some test methods
	_ = createTestMethod(t, store) // Create a general domain method

	mm := NewMethodManager(store)
	method2, err := mm.CreateMethod(
		context.Background(),
		"User Method",
		"User-specific method",
		[]ApproachStep{
			{Description: "User step", Tools: []string{"user_tool"}},
		},
		MethodDomainUser,
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("Failed to create user method: %v", err)
	}

	// Update metrics for method2
	err = mm.UpdateMethodMetrics(context.Background(), method2.ID, true, 9.0)
	if err != nil {
		t.Fatalf("Failed to update method2 metrics: %v", err)
	}

	// Get statistics
	stats, err := cc.GetMethodCacheStatistics(context.Background())
	if err != nil {
		t.Fatalf("GetMethodCacheStatistics failed: %v", err)
	}

	// Verify statistics structure
	expectedKeys := []string{
		"total_methods", "active_methods", "deprecated_methods", "superseded_methods",
		"general_domain", "specific_domain", "user_domain", "avg_success_rate",
		"most_used_method", "best_rated_method",
	}

	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Expected key %s in statistics", key)
		}
	}

	// Verify counts
	if stats["total_methods"].(int) != 2 {
		t.Errorf("Expected total_methods 2, got %v", stats["total_methods"])
	}
	if stats["active_methods"].(int) != 2 {
		t.Errorf("Expected active_methods 2, got %v", stats["active_methods"])
	}
	if stats["general_domain"].(int) != 1 {
		t.Errorf("Expected general_domain 1, got %v", stats["general_domain"])
	}
	if stats["user_domain"].(int) != 1 {
		t.Errorf("Expected user_domain 1, got %v", stats["user_domain"])
	}
}

func TestFindBestMethod_NoMethodsAvailable(t *testing.T) {
	store := createTestStore(t)
	reasoner := NewMockLLMReasoner()
	cc := NewContemplativeCursor(store, reasoner)

	analysis := &ObjectiveAnalysis{
		ComplexityLevel: 5,
		DomainContext:   "nonexistent_domain",
	}

	// Should return empty string when no methods are available
	methodID, err := cc.findBestMethod(context.Background(), analysis)
	if err != nil {
		t.Fatalf("findBestMethod failed: %v", err)
	}
	if methodID != "" {
		t.Errorf("Expected empty method ID when no methods available, got %s", methodID)
	}
}

func TestSelectBestFromCandidates(t *testing.T) {
	store := createTestStore(t)
	reasoner := NewMockLLMReasoner()
	cc := NewContemplativeCursor(store, reasoner)

	// Create test methods with different success rates
	mm := NewMethodManager(store)

	// Good method with high success rate
	goodMethod, err := mm.CreateMethod(
		context.Background(),
		"Good Method",
		"A reliable method",
		[]ApproachStep{{Description: "Do good work"}},
		MethodDomainGeneral,
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("Failed to create good method: %v", err)
	}

	// Simulate successful executions
	for i := 0; i < 8; i++ {
		err = mm.UpdateMethodMetrics(context.Background(), goodMethod.ID, true, 8.0)
		if err != nil {
			t.Fatalf("Failed to update good method metrics: %v", err)
		}
	}
	// Add some failures
	for i := 0; i < 2; i++ {
		err = mm.UpdateMethodMetrics(context.Background(), goodMethod.ID, false, 4.0)
		if err != nil {
			t.Fatalf("Failed to update good method metrics: %v", err)
		}
	}

	// Bad method with low success rate
	badMethod, err := mm.CreateMethod(
		context.Background(),
		"Bad Method",
		"An unreliable method",
		[]ApproachStep{{Description: "Do bad work"}},
		MethodDomainGeneral,
		map[string]interface{}{},
	)
	if err != nil {
		t.Fatalf("Failed to create bad method: %v", err)
	}

	// Simulate mostly failed executions
	for i := 0; i < 2; i++ {
		err = mm.UpdateMethodMetrics(context.Background(), badMethod.ID, true, 6.0)
		if err != nil {
			t.Fatalf("Failed to update bad method metrics: %v", err)
		}
	}
	for i := 0; i < 8; i++ {
		err = mm.UpdateMethodMetrics(context.Background(), badMethod.ID, false, 3.0)
		if err != nil {
			t.Fatalf("Failed to update bad method metrics: %v", err)
		}
	}

	// Get updated methods
	goodMethodUpdated, err := mm.GetMethod(context.Background(), goodMethod.ID)
	if err != nil {
		t.Fatalf("Failed to get updated good method: %v", err)
	}
	badMethodUpdated, err := mm.GetMethod(context.Background(), badMethod.ID)
	if err != nil {
		t.Fatalf("Failed to get updated bad method: %v", err)
	}

	candidates := []*Method{badMethodUpdated, goodMethodUpdated} // Bad method first
	analysis := &ObjectiveAnalysis{}

	// Should select the good method despite being second in list
	selected := cc.selectBestFromCandidates(candidates, analysis)
	if selected == nil {
		t.Fatal("Expected a method to be selected")
	}
	if selected.ID != goodMethod.ID {
		t.Errorf("Expected good method to be selected, got %s", selected.ID)
	}

	// Test with no candidates
	selected = cc.selectBestFromCandidates([]*Method{}, analysis)
	if selected != nil {
		t.Error("Expected nil when no candidates provided")
	}
}

// Benchmark tests
func BenchmarkCreateExecutionPlan(b *testing.B) {
	store := createTestStore(&testing.T{}) // Use a dummy testing.T for benchmark
	reasoner := NewMockLLMReasoner()
	cc := NewContemplativeCursor(store, reasoner)

	// Create test data
	goal := createTestGoal(&testing.T{}, store)
	method := createTestMethod(&testing.T{}, store)
	objective := createTestObjective(&testing.T{}, store, goal.ID, method.ID)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := cc.CreateExecutionPlan(context.Background(), objective.ID)
		if err != nil {
			b.Fatalf("CreateExecutionPlan failed: %v", err)
		}
	}
}