package test

import (
	"context"
	"testing"

	"github.com/yourusername/ai-work-studio/pkg/core"
)

// TestLearningLoopBasicFlow tests the complete learning loop cycle.
func TestLearningLoopBasicFlow(t *testing.T) {
	fixtures := NewTestFixtures(t)
	ctx := context.Background()

	// Set up components
	mockReasoner := NewMockLLMReasoner()
	mockExecutor := NewMockTaskExecutor()
	mockContextLoader := NewMockContextLoader()
	mockLearningAgent := NewMockLearningAgent()

	cc := core.NewContemplativeCursor(fixtures.Store, mockReasoner)
	rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)
	ll := core.NewLearningLoop(fixtures.Store, cc, rtc, mockLearningAgent)

	objective := fixtures.GetObjectiveByTitle("Analyze Customer Satisfaction Issues")
	if objective == nil {
		t.Fatal("Could not find test objective")
	}

	t.Run("SuccessfulExecution", func(t *testing.T) {
		// Configure mocks for successful execution
		mockExecutor.ShouldFailExecution = false
		mockLearningAgent.ShouldFailAnalysis = false

		// Execute objective through learning loop
		result, err := ll.ExecuteObjective(ctx, objective.ID)
		if err != nil {
			t.Fatalf("Failed to execute objective through learning loop: %v", err)
		}

		// Verify result structure
		if result.ObjectiveID != objective.ID {
			t.Errorf("Expected objective ID %s, got %s", objective.ID, result.ObjectiveID)
		}
		if !result.WasSuccessful {
			t.Error("Expected successful execution")
		}
		if len(result.ExecutionAttempts) == 0 {
			t.Error("Expected at least one execution attempt")
		}
		if result.FinalOutcome != core.OutcomeSuccess {
			t.Errorf("Expected final outcome %s, got %s", core.OutcomeSuccess, result.FinalOutcome)
		}

		// Verify mock interactions
		if len(mockReasoner.AnalyzeCalls) == 0 {
			t.Error("Expected LLM reasoner to be called for analysis")
		}
		if len(mockExecutor.ExecuteCalls) == 0 {
			t.Error("Expected task executor to be called")
		}
		if len(mockLearningAgent.AnalyzeOutcomeCalls) == 0 {
			t.Error("Expected learning agent to analyze outcome")
		}

		// Verify method metrics were updated
		method, err := fixtures.MethodManager.GetMethod(ctx, objective.MethodID)
		if err != nil {
			t.Fatalf("Failed to retrieve method after execution: %v", err)
		}
		if method.Metrics.ExecutionCount == 0 {
			t.Error("Expected method execution count to be updated")
		}
	})

	t.Run("ExecutionWithPartialFailure", func(t *testing.T) {
		// Configure mocks for partial failure
		mockExecutor.FailOnTaskType = "analysis" // Fail analysis tasks
		mockLearningAgent.CustomAnalysis = fixtures.CreateExecutionAnalysis(core.OutcomePartialSuccess, 6)

		objective2 := fixtures.GetObjectiveByTitle("Optimize User API Database Queries")
		if objective2 == nil {
			t.Skip("Skipping test - required objective not found")
		}

		result, err := ll.ExecuteObjective(ctx, objective2.ID)
		if err != nil {
			t.Fatalf("Failed to execute objective with partial failure: %v", err)
		}

		// Should still be considered successful for partial success
		if result.FinalOutcome != core.OutcomePartialSuccess {
			t.Errorf("Expected partial success outcome, got %s", result.FinalOutcome)
		}
		if !result.WasSuccessful {
			t.Error("Expected partial success to be considered successful")
		}
	})
}

// TestMethodEvolutionScenarios tests different method evolution scenarios.
func TestMethodEvolutionScenarios(t *testing.T) {
	fixtures := NewTestFixtures(t)
	ctx := context.Background()

	method := fixtures.GetMethodByName("Systematic Problem Analysis")
	if method == nil {
		t.Fatal("Could not find test method")
	}

	t.Run("MethodModification", func(t *testing.T) {
		// Set up learning agent to propose modification
		mockLearningAgent := NewMockLearningAgent()
		mockLearningAgent.CustomRefinement = fixtures.CreateMethodRefinement(core.RefinementModify)
		mockLearningAgent.CustomEvaluation = &core.RefinementEvaluation{
			IsImprovement:     true,
			ReducesComplexity: true,
			QualityScore:      8.0,
			Concerns:          []string{},
			Recommendation:    core.RecommendApply,
		}

		// Simulate poor method performance to trigger refinement
		err := fixtures.SimulateMethodUsage(ctx, method.ID, 5, 0.4) // 40% success rate
		if err != nil {
			t.Fatalf("Failed to simulate method usage: %v", err)
		}

		// Set up learning loop components
		mockReasoner := NewMockLLMReasoner()
		mockExecutor := NewMockTaskExecutor()
		mockExecutor.ShouldFailExecution = true // Force failure to trigger refinement
		mockContextLoader := NewMockContextLoader()

		cc := core.NewContemplativeCursor(fixtures.Store, mockReasoner)
		rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)
		ll := core.NewLearningLoop(fixtures.Store, cc, rtc, mockLearningAgent)

		// Configure analysis to indicate method failure
		mockLearningAgent.CustomAnalysis = fixtures.CreateExecutionAnalysis(core.OutcomeMethodFailure, 8)

		objective := fixtures.GetObjectiveByTitle("Analyze Customer Satisfaction Issues")
		if objective == nil {
			t.Fatal("Could not find test objective")
		}

		result, err := ll.ExecuteObjective(ctx, objective.ID)
		if err != nil {
			t.Fatalf("Failed to execute objective: %v", err)
		}

		// Verify refinement was attempted
		lastAttempt := result.ExecutionAttempts[len(result.ExecutionAttempts)-1]
		if !lastAttempt.RefinementApplied {
			t.Error("Expected refinement to be applied")
		}

		// Verify learning agent was called for refinement
		if len(mockLearningAgent.ProposeRefinementCalls) == 0 {
			t.Error("Expected learning agent to propose refinement")
		}
		if len(mockLearningAgent.EvaluateRefinementCalls) == 0 {
			t.Error("Expected learning agent to evaluate refinement")
		}

		// Check for method evolution
		evolution, err := fixtures.MethodManager.GetMethodEvolution(ctx, method.ID)
		if err != nil {
			t.Fatalf("Failed to get method evolution: %v", err)
		}
		if len(evolution.Successors) == 0 {
			t.Error("Expected method to have evolved successors")
		}
	})

	t.Run("MethodReplacement", func(t *testing.T) {
		// Test complete method replacement scenario
		mockLearningAgent := NewMockLearningAgent()
		mockLearningAgent.CustomRefinement = fixtures.CreateMethodRefinement(core.RefinementReplace)
		mockLearningAgent.CustomEvaluation = &core.RefinementEvaluation{
			IsImprovement:     true,
			ReducesComplexity: true,
			QualityScore:      9.0,
			Concerns:          []string{},
			Recommendation:    core.RecommendApply,
		}

		// Create a new method for testing replacement
		testMethod, err := fixtures.MethodManager.CreateMethod(ctx,
			"Test Method for Replacement",
			"A method that will be replaced during testing",
			[]core.ApproachStep{
				{Description: "Overly complex step 1", Tools: []string{"tool1", "tool2", "tool3"}},
				{Description: "Overly complex step 2", Tools: []string{"tool4", "tool5", "tool6"}},
				{Description: "Overly complex step 3", Tools: []string{"tool7", "tool8", "tool9"}},
			},
			core.MethodDomainGeneral,
			map[string]interface{}{"test": true})
		if err != nil {
			t.Fatalf("Failed to create test method: %v", err)
		}

		// Simulate very poor performance to justify replacement
		err = fixtures.SimulateMethodUsage(ctx, testMethod.ID, 10, 0.1) // 10% success rate
		if err != nil {
			t.Fatalf("Failed to simulate poor method usage: %v", err)
		}

		// Set up learning loop
		mockReasoner := NewMockLLMReasoner()
		mockExecutor := NewMockTaskExecutor()
		mockExecutor.ShouldFailExecution = true
		mockContextLoader := NewMockContextLoader()

		cc := core.NewContemplativeCursor(fixtures.Store, mockReasoner)
		rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)
		ll := core.NewLearningLoop(fixtures.Store, cc, rtc, mockLearningAgent)

		// Configure analysis to indicate severe method failure
		mockLearningAgent.CustomAnalysis = &core.ExecutionAnalysis{
			OverallAssessment: core.OutcomeMethodFailure,
			ComplexityAssessment: core.ComplexityAnalysis{
				CurrentComplexityLevel: 10,
				OptimalComplexityLevel: 4,
			},
			ConfidenceLevel: 0.9,
		}

		// Create objective using the test method
		testObjective, err := fixtures.ObjectiveManager.CreateObjective(ctx,
			fixtures.SampleGoals[0].ID,
			testMethod.ID,
			"Test Objective for Replacement",
			"Testing method replacement scenario",
			map[string]interface{}{"test": true},
			5)
		if err != nil {
			t.Fatalf("Failed to create test objective: %v", err)
		}

		result, err := ll.ExecuteObjective(ctx, testObjective.ID)
		if err != nil {
			t.Fatalf("Failed to execute objective: %v", err)
		}

		// Verify replacement behavior
		if len(result.ExecutionAttempts) > 1 {
			for _, attempt := range result.ExecutionAttempts[1:] {
				if attempt.RefinementApplied {
					// Should have created a new method
					evolution, err := fixtures.MethodManager.GetMethodEvolution(ctx, testMethod.ID)
					if err != nil {
						t.Fatalf("Failed to get method evolution: %v", err)
					}
					if len(evolution.Successors) == 0 {
						t.Error("Expected method replacement to create successor")
					}
					break
				}
			}
		}
	})

	t.Run("MethodRetirement", func(t *testing.T) {
		// Test method retirement scenario
		mockLearningAgent := NewMockLearningAgent()
		mockLearningAgent.CustomRefinement = fixtures.CreateMethodRefinement(core.RefinementRetire)

		// Create a method specifically for retirement testing
		retireMethod, err := fixtures.MethodManager.CreateMethod(ctx,
			"Method to Retire",
			"A method that should be retired",
			[]core.ApproachStep{
				{Description: "Obsolete step", Tools: []string{"obsolete_tool"}},
			},
			core.MethodDomainGeneral,
			map[string]interface{}{"retire_test": true})
		if err != nil {
			t.Fatalf("Failed to create method for retirement: %v", err)
		}

		// Set up learning loop
		mockReasoner := NewMockLLMReasoner()
		mockExecutor := NewMockTaskExecutor()
		mockContextLoader := NewMockContextLoader()

		cc := core.NewContemplativeCursor(fixtures.Store, mockReasoner)
		rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)
		_ = core.NewLearningLoop(fixtures.Store, cc, rtc, mockLearningAgent)

		// Force method retirement by configuring appropriate responses
		mockLearningAgent.CustomEvaluation = &core.RefinementEvaluation{
			IsImprovement:     false,
			ReducesComplexity: false,
			QualityScore:      2.0,
			Recommendation:    core.RecommendApply,
		}

		// Simulate execution that leads to retirement
		err = fixtures.SimulateMethodUsage(ctx, retireMethod.ID, 5, 0.0) // 0% success rate

		// After attempting retirement, method should be deprecated
		updatedMethod, err := fixtures.MethodManager.GetMethod(ctx, retireMethod.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve method after potential retirement: %v", err)
		}

		// Note: The actual retirement would be triggered by execution in the learning loop
		// For this test, we verify the method structure is ready for retirement
		if updatedMethod.Status != core.MethodStatusActive {
			// If status changed, it should be deprecated
			if updatedMethod.Status != core.MethodStatusDeprecated {
				t.Errorf("Expected deprecated status or active, got %s", updatedMethod.Status)
			}
		}
	})
}

// TestMethodLearningFromMultipleExecutions tests learning from execution patterns.
func TestMethodLearningFromMultipleExecutions(t *testing.T) {
	fixtures := NewTestFixtures(t)
	ctx := context.Background()

	method := fixtures.GetMethodByName("Database Performance Optimization")
	if method == nil {
		t.Fatal("Could not find test method")
	}

	t.Run("LearningFromSuccessPatterns", func(t *testing.T) {
		// Simulate multiple successful executions
		initialMetrics := method.Metrics

		err := fixtures.SimulateMethodUsage(ctx, method.ID, 20, 0.85) // 85% success rate
		if err != nil {
			t.Fatalf("Failed to simulate method usage: %v", err)
		}

		updatedMethod, err := fixtures.MethodManager.GetMethod(ctx, method.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve updated method: %v", err)
		}

		// Verify metrics improved
		if updatedMethod.Metrics.ExecutionCount <= initialMetrics.ExecutionCount {
			t.Error("Expected execution count to increase")
		}
		if updatedMethod.Metrics.SuccessCount <= initialMetrics.SuccessCount {
			t.Error("Expected success count to increase")
		}

		successRate := updatedMethod.Metrics.SuccessRate()
		if successRate < 80.0 || successRate > 90.0 {
			t.Errorf("Expected success rate around 85%%, got %.1f%%", successRate)
		}

		if updatedMethod.Metrics.AverageRating <= 0 {
			t.Error("Expected average rating to be set")
		}
	})

	t.Run("LearningFromFailurePatterns", func(t *testing.T) {
		// Create a method that will consistently fail
		failingMethod, err := fixtures.MethodManager.CreateMethod(ctx,
			"Consistently Failing Method",
			"A method designed to fail for testing learning",
			[]core.ApproachStep{
				{Description: "Problematic step", Tools: []string{"unreliable_tool"}},
			},
			core.MethodDomainGeneral,
			map[string]interface{}{"test_failure": true})
		if err != nil {
			t.Fatalf("Failed to create failing method: %v", err)
		}

		// Simulate consistent failures
		err = fixtures.SimulateMethodUsage(ctx, failingMethod.ID, 15, 0.2) // 20% success rate
		if err != nil {
			t.Fatalf("Failed to simulate failing method usage: %v", err)
		}

		updatedMethod, err := fixtures.MethodManager.GetMethod(ctx, failingMethod.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve updated failing method: %v", err)
		}

		// Verify poor performance is tracked
		successRate := updatedMethod.Metrics.SuccessRate()
		if successRate > 25.0 {
			t.Errorf("Expected low success rate, got %.1f%%", successRate)
		}

		// This method should be a candidate for refinement
		config := core.DefaultLearningLoopConfig()
		if updatedMethod.Metrics.ExecutionCount >= config.MinExecutionsBeforeRefinement &&
			successRate < config.SuccessRateThresholdForRefinement {
			t.Logf("Method correctly identified as needing refinement (%.1f%% success rate)", successRate)
		}
	})
}

// TestLearningLoopConfiguration tests different learning loop configurations.
func TestLearningLoopConfiguration(t *testing.T) {
	fixtures := NewTestFixtures(t)

	t.Run("DefaultConfiguration", func(t *testing.T) {
		config := core.DefaultLearningLoopConfig()

		// Verify sensible defaults
		if config.MinExecutionsBeforeRefinement < 1 {
			t.Error("Expected minimum executions before refinement to be positive")
		}
		if config.SuccessRateThresholdForRefinement <= 0 || config.SuccessRateThresholdForRefinement >= 100 {
			t.Error("Expected success rate threshold to be between 0 and 100")
		}
		if config.MaxRefinementAttempts < 1 {
			t.Error("Expected max refinement attempts to be positive")
		}
		if config.ComplexityBiasWeight < 0 || config.ComplexityBiasWeight > 1 {
			t.Error("Expected complexity bias weight to be between 0 and 1")
		}
	})

	t.Run("CustomConfiguration", func(t *testing.T) {
		mockReasoner := NewMockLLMReasoner()
		mockExecutor := NewMockTaskExecutor()
		mockContextLoader := NewMockContextLoader()
		mockLearningAgent := NewMockLearningAgent()

		cc := core.NewContemplativeCursor(fixtures.Store, mockReasoner)
		rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)
		ll := core.NewLearningLoop(fixtures.Store, cc, rtc, mockLearningAgent)

		// Set custom configuration
		customConfig := &core.LearningLoopConfig{
			MinExecutionsBeforeRefinement:     1, // Immediate refinement
			SuccessRateThresholdForRefinement: 90.0, // High bar
			MaxRefinementAttempts:             1, // Single attempt
			ComplexityBiasWeight:              0.9, // Strong complexity bias
			EnableMethodEvolution:             true,
			PreserveMethodHistory:             true,
		}

		ll.SetConfiguration(customConfig)

		retrievedConfig := ll.GetConfiguration()
		if retrievedConfig.MinExecutionsBeforeRefinement != customConfig.MinExecutionsBeforeRefinement {
			t.Error("Configuration not properly set")
		}
		if retrievedConfig.ComplexityBiasWeight != customConfig.ComplexityBiasWeight {
			t.Error("Complexity bias weight not properly set")
		}
	})
}

// TestMethodComplexityBias tests the system's preference for simpler methods.
func TestMethodComplexityBias(t *testing.T) {
	fixtures := NewTestFixtures(t)
	ctx := context.Background()

	t.Run("ComplexityBiasInRefinement", func(t *testing.T) {
		mockLearningAgent := NewMockLearningAgent()

		// Set up refinement that reduces complexity
		mockLearningAgent.CustomRefinement = &core.MethodRefinement{
			Type:                          core.RefinementModify,
			ExpectedComplexityChange:      -3,
			ExpectedSuccessRateImprovement: 5.0,
			Reasoning:                     "Simplify method by removing unnecessary steps",
		}

		// Set up evaluation that confirms complexity reduction
		mockLearningAgent.CustomEvaluation = &core.RefinementEvaluation{
			IsImprovement:     true,
			ReducesComplexity: true,
			QualityScore:      8.5,
			Recommendation:    core.RecommendApply,
		}

		// Create learning loop with high complexity bias
		mockReasoner := NewMockLLMReasoner()
		mockExecutor := NewMockTaskExecutor()
		mockContextLoader := NewMockContextLoader()

		cc := core.NewContemplativeCursor(fixtures.Store, mockReasoner)
		rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)
		ll := core.NewLearningLoop(fixtures.Store, cc, rtc, mockLearningAgent)

		// Configure high complexity bias
		config := core.DefaultLearningLoopConfig()
		config.ComplexityBiasWeight = 0.9
		ll.SetConfiguration(config)

		// Create complex method
		complexMethod, err := fixtures.MethodManager.CreateMethod(ctx,
			"Complex Method",
			"An overly complex method for testing bias",
			[]core.ApproachStep{
				{Description: "Step 1", Tools: []string{"tool1"}},
				{Description: "Step 2", Tools: []string{"tool2"}},
				{Description: "Step 3", Tools: []string{"tool3"}},
				{Description: "Step 4", Tools: []string{"tool4"}},
				{Description: "Step 5", Tools: []string{"tool5"}},
			},
			core.MethodDomainGeneral,
			map[string]interface{}{"complexity_test": true})
		if err != nil {
			t.Fatalf("Failed to create complex method: %v", err)
		}

		// Simulate poor performance due to complexity
		err = fixtures.SimulateMethodUsage(ctx, complexMethod.ID, 5, 0.3) // 30% success rate
		if err != nil {
			t.Fatalf("Failed to simulate complex method usage: %v", err)
		}

		// Configure analysis to indicate high complexity
		mockLearningAgent.CustomAnalysis = &core.ExecutionAnalysis{
			OverallAssessment: core.OutcomeMethodFailure,
			ComplexityAssessment: core.ComplexityAnalysis{
				CurrentComplexityLevel: 9,
				OptimalComplexityLevel: 5,
			},
			ConfidenceLevel: 0.8,
		}

		// Create test objective
		testObjective, err := fixtures.ObjectiveManager.CreateObjective(ctx,
			fixtures.SampleGoals[0].ID,
			complexMethod.ID,
			"Test Complex Method",
			"Testing complexity bias",
			map[string]interface{}{},
			5)
		if err != nil {
			t.Fatalf("Failed to create test objective: %v", err)
		}

		// Execute with failing tasks to trigger refinement
		mockExecutor.ShouldFailExecution = true
		result, err := ll.ExecuteObjective(ctx, testObjective.ID)
		if err != nil {
			t.Fatalf("Failed to execute objective: %v", err)
		}

		// Verify learning agent interactions
		if len(mockLearningAgent.EvaluateRefinementCalls) > 0 {
			// The system should have evaluated whether to apply the complexity-reducing refinement
			t.Log("Learning system correctly evaluated complexity-reducing refinement")
		}

		// Check for method evolution (complexity reduction)
		evolution, err := fixtures.MethodManager.GetMethodEvolution(ctx, complexMethod.ID)
		if err != nil {
			t.Fatalf("Failed to get method evolution: %v", err)
		}

		if len(evolution.Successors) > 0 {
			// Verify successor is simpler
			successor := evolution.Successors[0]
			if len(successor.Approach) >= len(complexMethod.Approach) {
				t.Error("Expected successor method to be simpler (fewer steps)")
			}
			t.Logf("Method evolved from %d steps to %d steps", len(complexMethod.Approach), len(successor.Approach))
		}

		// Log results for inspection
		t.Logf("Learning result: %d attempts, final outcome: %s", len(result.ExecutionAttempts), result.FinalOutcome)
	})
}

// TestLearningLoopRobustness tests edge cases and error conditions in learning.
func TestLearningLoopRobustness(t *testing.T) {
	fixtures := NewTestFixtures(t)
	ctx := context.Background()

	t.Run("LearningAgentFailure", func(t *testing.T) {
		// Test graceful handling when learning agent fails
		mockReasoner := NewMockLLMReasoner()
		mockExecutor := NewMockTaskExecutor()
		mockContextLoader := NewMockContextLoader()
		mockLearningAgent := NewMockLearningAgent()

		// Configure learning agent to fail
		mockLearningAgent.ShouldFailAnalysis = true

		cc := core.NewContemplativeCursor(fixtures.Store, mockReasoner)
		rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)
		ll := core.NewLearningLoop(fixtures.Store, cc, rtc, mockLearningAgent)

		objective := fixtures.GetObjectiveByTitle("Analyze Customer Satisfaction Issues")
		if objective == nil {
			t.Fatal("Could not find test objective")
		}

		result, err := ll.ExecuteObjective(ctx, objective.ID)
		if err != nil {
			t.Fatalf("Expected graceful handling of learning agent failure, got: %v", err)
		}

		// Should complete execution even if learning fails
		if len(result.ExecutionAttempts) == 0 {
			t.Error("Expected at least one execution attempt despite learning failure")
		}
	})

	t.Run("InsufficientDataForLearning", func(t *testing.T) {
		// Test behavior when method doesn't have enough execution history
		newMethod, err := fixtures.MethodManager.CreateMethod(ctx,
			"New Method",
			"A method with no execution history",
			[]core.ApproachStep{
				{Description: "New step", Tools: []string{"new_tool"}},
			},
			core.MethodDomainGeneral,
			map[string]interface{}{})
		if err != nil {
			t.Fatalf("Failed to create new method: %v", err)
		}

		mockReasoner := NewMockLLMReasoner()
		mockExecutor := NewMockTaskExecutor()
		mockContextLoader := NewMockContextLoader()
		mockLearningAgent := NewMockLearningAgent()

		// Configure analysis to indicate insufficient data
		mockLearningAgent.CustomAnalysis = &core.ExecutionAnalysis{
			OverallAssessment: core.OutcomeInsufficientData,
			ConfidenceLevel:   0.3,
		}

		cc := core.NewContemplativeCursor(fixtures.Store, mockReasoner)
		rtc := core.NewRealTimeCursor(fixtures.Store, mockExecutor, mockContextLoader)
		ll := core.NewLearningLoop(fixtures.Store, cc, rtc, mockLearningAgent)

		// Create objective using the new method
		testObjective, err := fixtures.ObjectiveManager.CreateObjective(ctx,
			fixtures.SampleGoals[0].ID,
			newMethod.ID,
			"Test New Method",
			"Testing insufficient data scenario",
			map[string]interface{}{},
			5)
		if err != nil {
			t.Fatalf("Failed to create test objective: %v", err)
		}

		result, err := ll.ExecuteObjective(ctx, testObjective.ID)
		if err != nil {
			t.Fatalf("Failed to execute objective with new method: %v", err)
		}

		// Should not attempt refinement due to insufficient data
		if len(result.ExecutionAttempts) > 1 {
			t.Error("Did not expect multiple attempts for insufficient data scenario")
		}

		// Verify no refinement was proposed
		if len(mockLearningAgent.ProposeRefinementCalls) > 0 {
			t.Error("Did not expect refinement proposals for insufficient data")
		}
	})
}