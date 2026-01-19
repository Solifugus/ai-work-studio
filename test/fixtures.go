package test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/yourusername/ai-work-studio/pkg/core"
	"github.com/yourusername/ai-work-studio/pkg/storage"
)

// TestFixtures provides common test data and utilities for integration tests.
type TestFixtures struct {
	TestDataDir       string
	Store             *storage.Store
	GoalManager       *core.GoalManager
	MethodManager     *core.MethodManager
	ObjectiveManager  *core.ObjectiveManager
	SampleGoals       []*core.Goal
	SampleMethods     []*core.Method
	SampleObjectives  []*core.Objective
}

// NewTestFixtures creates a fresh set of test fixtures with isolated data directory.
func NewTestFixtures(t *testing.T) *TestFixtures {
	// Create temporary directory for test data
	testDir, err := os.MkdirTemp("", "ai-work-studio-test-")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Clean up on test completion
	t.Cleanup(func() {
		os.RemoveAll(testDir)
	})

	// Initialize storage
	store, err := storage.NewStore(testDir)
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}

	fixtures := &TestFixtures{
		TestDataDir:      testDir,
		Store:            store,
		GoalManager:      core.NewGoalManager(store),
		MethodManager:    core.NewMethodManager(store),
		ObjectiveManager: core.NewObjectiveManager(store),
	}

	// Create sample data
	fixtures.createSampleData(t)

	return fixtures
}

// createSampleData populates the fixtures with realistic test data.
func (tf *TestFixtures) createSampleData(t *testing.T) {
	ctx := context.Background()

	// Sample Goals
	tf.SampleGoals = []*core.Goal{}

	// Goal 1: High-level business goal
	goal1, err := tf.GoalManager.CreateGoal(ctx,
		"Improve Customer Satisfaction",
		"Increase customer satisfaction scores by implementing better support processes and product improvements",
		8, // High priority
		map[string]interface{}{
			"target_score":      4.5,
			"current_score":     3.8,
			"deadline":          "2024-06-30",
			"budget":            "$50000",
			"stakeholders":      []string{"customer_success", "product", "support"},
		})
	if err != nil {
		t.Fatalf("Failed to create test goal 1: %v", err)
	}
	tf.SampleGoals = append(tf.SampleGoals, goal1)

	// Goal 2: Technical goal
	goal2, err := tf.GoalManager.CreateGoal(ctx,
		"Optimize Database Performance",
		"Reduce database query times and improve system responsiveness for better user experience",
		7,
		map[string]interface{}{
			"target_latency":    "< 100ms",
			"current_latency":   "250ms",
			"affected_systems":  []string{"user_api", "analytics_api", "reporting"},
			"approach":          "indexing_and_query_optimization",
		})
	if err != nil {
		t.Fatalf("Failed to create test goal 2: %v", err)
	}
	tf.SampleGoals = append(tf.SampleGoals, goal2)

	// Goal 3: Learning goal (sub-goal of goal 1)
	goal3, err := tf.GoalManager.CreateGoal(ctx,
		"Train Support Team",
		"Provide comprehensive training to support team on new product features and escalation procedures",
		6,
		map[string]interface{}{
			"training_duration": "2 weeks",
			"team_size":         12,
			"completion_rate":   "95%",
		})
	if err != nil {
		t.Fatalf("Failed to create test goal 3: %v", err)
	}
	tf.SampleGoals = append(tf.SampleGoals, goal3)

	// Create goal hierarchy
	err = tf.GoalManager.AddSubGoal(ctx, goal1.ID, goal3.ID)
	if err != nil {
		t.Fatalf("Failed to create goal hierarchy: %v", err)
	}

	// Sample Methods
	tf.SampleMethods = []*core.Method{}

	// Method 1: General analysis method
	method1, err := tf.MethodManager.CreateMethod(ctx,
		"Systematic Problem Analysis",
		"A structured approach to analyzing complex problems by breaking them down into components and identifying root causes",
		[]core.ApproachStep{
			{
				Description: "Define the problem clearly and gather all relevant context",
				Tools:       []string{"research", "data_collection"},
				Heuristics:  []string{"ask_why_five_times", "gather_stakeholder_input"},
				Conditions: map[string]interface{}{
					"min_data_quality": 0.7,
					"stakeholder_availability": true,
				},
			},
			{
				Description: "Analyze the problem to identify root causes and contributing factors",
				Tools:       []string{"analysis", "pattern_recognition"},
				Heuristics:  []string{"look_for_patterns", "consider_systemic_issues"},
			},
			{
				Description: "Develop solution options and evaluate their feasibility",
				Tools:       []string{"brainstorming", "feasibility_analysis"},
				Heuristics:  []string{"consider_multiple_options", "assess_trade_offs"},
			},
			{
				Description: "Create implementation plan with clear steps and success metrics",
				Tools:       []string{"planning", "metric_definition"},
				Heuristics:  []string{"define_clear_milestones", "plan_for_contingencies"},
			},
		},
		core.MethodDomainGeneral,
		map[string]interface{}{
			"complexity_level": 6,
			"estimated_time":   "4-8 hours",
			"required_skills":  []string{"analytical_thinking", "problem_solving"},
		})
	if err != nil {
		t.Fatalf("Failed to create test method 1: %v", err)
	}
	tf.SampleMethods = append(tf.SampleMethods, method1)

	// Method 2: Database optimization method
	method2, err := tf.MethodManager.CreateMethod(ctx,
		"Database Performance Optimization",
		"Specialized approach for identifying and resolving database performance bottlenecks",
		[]core.ApproachStep{
			{
				Description: "Profile current database performance and identify slow queries",
				Tools:       []string{"database_profiling", "query_analysis"},
				Heuristics:  []string{"focus_on_high_frequency_queries", "check_resource_utilization"},
			},
			{
				Description: "Analyze query execution plans and identify optimization opportunities",
				Tools:       []string{"explain_plan", "index_analysis"},
				Heuristics:  []string{"look_for_full_table_scans", "check_join_efficiency"},
			},
			{
				Description: "Implement optimizations starting with highest impact, lowest risk",
				Tools:       []string{"indexing", "query_rewriting"},
				Heuristics:  []string{"test_in_staging_first", "measure_before_after"},
			},
		},
		core.MethodDomainSpecific,
		map[string]interface{}{
			"domain": "database_administration",
			"tech_stack": []string{"postgresql", "mysql", "mongodb"},
		})
	if err != nil {
		t.Fatalf("Failed to create test method 2: %v", err)
	}
	tf.SampleMethods = append(tf.SampleMethods, method2)

	// Method 3: Training delivery method
	method3, err := tf.MethodManager.CreateMethod(ctx,
		"Effective Training Program Delivery",
		"User-specific method for delivering effective training based on past successful programs",
		[]core.ApproachStep{
			{
				Description: "Assess current knowledge levels and learning styles of participants",
				Tools:       []string{"assessment", "survey"},
				Heuristics:  []string{"adapt_to_learning_styles", "identify_knowledge_gaps"},
			},
			{
				Description: "Design interactive curriculum with hands-on exercises",
				Tools:       []string{"curriculum_design", "exercise_creation"},
				Heuristics:  []string{"balance_theory_and_practice", "include_real_scenarios"},
			},
			{
				Description: "Deliver training with regular feedback and adjustment",
				Tools:       []string{"presentation", "facilitation", "feedback_collection"},
				Heuristics:  []string{"check_understanding_frequently", "adapt_pace_to_group"},
			},
			{
				Description: "Follow up with assessment and reinforcement activities",
				Tools:       []string{"testing", "reinforcement_activities"},
				Heuristics:  []string{"measure_retention", "provide_ongoing_support"},
			},
		},
		core.MethodDomainUser,
		map[string]interface{}{
			"user_preference": "interactive_learning",
			"proven_success_rate": 0.92,
		})
	if err != nil {
		t.Fatalf("Failed to create test method 3: %v", err)
	}
	tf.SampleMethods = append(tf.SampleMethods, method3)

	// Sample Objectives
	tf.SampleObjectives = []*core.Objective{}

	// Objective 1: Analysis objective using general method
	objective1, err := tf.ObjectiveManager.CreateObjective(ctx,
		goal1.ID,
		method1.ID,
		"Analyze Customer Satisfaction Issues",
		"Conduct systematic analysis of current customer satisfaction issues to identify root causes and improvement opportunities",
		map[string]interface{}{
			"data_sources":     []string{"survey_responses", "support_tickets", "user_feedback"},
			"analysis_period":  "last_6_months",
			"output_format":    "detailed_report",
		},
		8) // High priority
	if err != nil {
		t.Fatalf("Failed to create test objective 1: %v", err)
	}
	tf.SampleObjectives = append(tf.SampleObjectives, objective1)

	// Objective 2: Database optimization objective
	objective2, err := tf.ObjectiveManager.CreateObjective(ctx,
		goal2.ID,
		method2.ID,
		"Optimize User API Database Queries",
		"Apply database optimization techniques to improve user API response times",
		map[string]interface{}{
			"target_database":     "user_db",
			"target_improvement":  "60%",
			"maintenance_window":  "weekend",
		},
		7)
	if err != nil {
		t.Fatalf("Failed to create test objective 2: %v", err)
	}
	tf.SampleObjectives = append(tf.SampleObjectives, objective2)

	// Objective 3: Training objective
	objective3, err := tf.ObjectiveManager.CreateObjective(ctx,
		goal3.ID,
		method3.ID,
		"Deliver Product Feature Training",
		"Train support team on new product features and escalation procedures",
		map[string]interface{}{
			"training_topics": []string{"feature_overview", "troubleshooting", "escalation_procedures"},
			"participants":    12,
			"format":         "hybrid_online_in_person",
		},
		6)
	if err != nil {
		t.Fatalf("Failed to create test objective 3: %v", err)
	}
	tf.SampleObjectives = append(tf.SampleObjectives, objective3)
}

// GetGoalByTitle finds a sample goal by its title.
func (tf *TestFixtures) GetGoalByTitle(title string) *core.Goal {
	for _, goal := range tf.SampleGoals {
		if goal.Title == title {
			return goal
		}
	}
	return nil
}

// GetMethodByName finds a sample method by its name.
func (tf *TestFixtures) GetMethodByName(name string) *core.Method {
	for _, method := range tf.SampleMethods {
		if method.Name == name {
			return method
		}
	}
	return nil
}

// GetObjectiveByTitle finds a sample objective by its title.
func (tf *TestFixtures) GetObjectiveByTitle(title string) *core.Objective {
	for _, objective := range tf.SampleObjectives {
		if objective.Title == title {
			return objective
		}
	}
	return nil
}

// CreateTestExecutionPlan creates a realistic execution plan for testing.
func (tf *TestFixtures) CreateTestExecutionPlan(objectiveID, methodID string) *core.ExecutionPlan {
	plan := &core.ExecutionPlan{
		ID:          "test-plan-" + time.Now().Format("20060102-150405"),
		ObjectiveID: objectiveID,
		MethodID:    methodID,
		Title:       "Test Execution Plan",
		Tasks: []core.ExecutionTask{
			{
				ID:          "task-1",
				Type:        "data_collection",
				Description: "Gather relevant data and context for analysis",
				Context: core.TaskContext{
					InputRefs:   []string{"data://survey_responses", "data://support_tickets"},
					OutputRef:   "data://collected_data",
					Parameters:  map[string]interface{}{"format": "structured"},
					TokenBudget: 500,
					Priority:    8,
				},
				MethodStepIndex: 0,
				EstimatedTokens: 500,
				CreatedAt:      time.Now(),
			},
			{
				ID:          "task-2",
				Type:        "analysis",
				Description: "Analyze collected data to identify patterns and root causes",
				Context: core.TaskContext{
					InputRefs:   []string{"data://collected_data"},
					OutputRef:   "data://analysis_results",
					Parameters:  map[string]interface{}{"depth": "comprehensive"},
					TokenBudget: 1000,
					Priority:    8,
				},
				MethodStepIndex: 1,
				EstimatedTokens: 1000,
				CreatedAt:      time.Now(),
			},
			{
				ID:          "task-3",
				Type:        "solution_design",
				Description: "Design solution options based on analysis findings",
				Context: core.TaskContext{
					InputRefs:   []string{"data://analysis_results"},
					OutputRef:   "data://solution_options",
					Parameters:  map[string]interface{}{"count": 3},
					TokenBudget: 800,
					Priority:    7,
				},
				MethodStepIndex: 2,
				EstimatedTokens: 800,
				CreatedAt:      time.Now(),
			},
			{
				ID:          "task-4",
				Type:        "implementation_planning",
				Description: "Create detailed implementation plan for chosen solution",
				Context: core.TaskContext{
					InputRefs:   []string{"data://solution_options", "data://analysis_results"},
					OutputRef:   "data://implementation_plan",
					Parameters:  map[string]interface{}{"detail_level": "high"},
					TokenBudget: 700,
					Priority:    7,
				},
				MethodStepIndex: 3,
				EstimatedTokens: 700,
				CreatedAt:      time.Now(),
			},
		},
		Dependencies: []core.TaskDependency{
			{TaskID: "task-2", DependsOnTaskID: "task-1", Reason: "Analysis requires collected data"},
			{TaskID: "task-3", DependsOnTaskID: "task-2", Reason: "Solution design requires analysis results"},
			{TaskID: "task-4", DependsOnTaskID: "task-3", Reason: "Implementation planning requires solution choice"},
		},
		TotalEstimatedTokens: 3000,
		CreatedBy:            "test-contemplative-cursor",
		CreatedAt:            time.Now(),
	}

	return plan
}

// CreateTaskResult creates a realistic task result for testing.
func (tf *TestFixtures) CreateTaskResult(taskID string, status core.TaskStatus, tokensUsed int) *core.TaskResult {
	result := &core.TaskResult{
		TaskID:     taskID,
		Status:     status,
		TokensUsed: tokensUsed,
		Duration:   time.Duration(tokensUsed) * time.Millisecond * 2, // Simulate 2ms per token
		ToolsUsed:  []string{"research", "analysis"},
		Confidence: 0.85,
		CompletedAt: time.Now(),
	}

	switch status {
	case core.TaskStatusCompleted:
		result.Output = map[string]interface{}{
			"result_type": "success",
			"data":       "Test task completed successfully",
			"quality":    0.9,
		}
		result.OutputRef = "data://task_output_" + taskID
	case core.TaskStatusFailed:
		result.ErrorMessage = "Test task failed due to simulated error"
		result.Confidence = 0.2
	}

	return result
}

// CreateExecutionResult creates a realistic execution result for testing.
func (tf *TestFixtures) CreateExecutionResult(planID, objectiveID string, status core.ExecutionStatus, taskResults map[string]*core.TaskResult) *core.ExecutionResult {
	startTime := time.Now().Add(-10 * time.Minute)
	endTime := time.Now()

	totalTokens := 0
	successfulTasks := 0
	failedTasks := 0

	for _, result := range taskResults {
		totalTokens += result.TokensUsed
		if result.Status == core.TaskStatusCompleted {
			successfulTasks++
		} else if result.Status == core.TaskStatusFailed {
			failedTasks++
		}
	}

	return &core.ExecutionResult{
		PlanID:               planID,
		ObjectiveID:          objectiveID,
		Status:               status,
		TaskResults:          taskResults,
		TotalTokensUsed:      totalTokens,
		TotalDuration:        endTime.Sub(startTime),
		StartTime:            startTime,
		EndTime:              endTime,
		SuccessfulTasks:      successfulTasks,
		FailedTasks:          failedTasks,
		MethodRefinementData: map[string]interface{}{
			"success_rate":    float64(successfulTasks) / float64(len(taskResults)) * 100,
			"avg_confidence":  0.85,
			"execution_time":  endTime.Sub(startTime).Seconds(),
		},
	}
}

// SimulateMethodUsage updates method metrics to simulate usage history.
func (tf *TestFixtures) SimulateMethodUsage(ctx context.Context, methodID string, executions int, successRate float64) error {
	for i := 0; i < executions; i++ {
		wasSuccessful := float64(i)/float64(executions) < successRate
		rating := 6.0
		if wasSuccessful {
			rating = 7.5 + (float64(i%3) * 0.5) // Vary rating 7.5-8.5 for successful
		} else {
			rating = 3.0 + (float64(i%3) * 1.0) // Vary rating 3.0-5.0 for failed
		}

		err := tf.MethodManager.UpdateMethodMetrics(ctx, methodID, wasSuccessful, rating)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateExecutionAnalysis creates a test execution analysis.
func (tf *TestFixtures) CreateExecutionAnalysis(outcome core.ExecutionOutcome, complexityLevel int) *core.ExecutionAnalysis {
	return &core.ExecutionAnalysis{
		OverallAssessment:   outcome,
		PrimaryFailureCause: "test-failure-cause",
		MethodPerformanceIssues: []core.PerformanceIssue{
			{
				Category:      core.IssueComplexity,
				Description:   "Method is overly complex for this type of objective",
				AffectedSteps: []int{2, 3},
				Severity:      6,
				SuggestedFix:  "Simplify approach steps",
			},
		},
		SuccessFactors: []string{"good_data_quality", "clear_requirements"},
		ImprovementOpportunities: []string{"reduce_complexity", "improve_tool_usage"},
		ComplexityAssessment: core.ComplexityAnalysis{
			CurrentComplexityLevel:      complexityLevel,
			ComplexityFactors:          []string{"many_steps", "complex_dependencies"},
			SimplificationOpportunities: []string{"combine_similar_steps", "remove_redundancy"},
			OptimalComplexityLevel:     complexityLevel - 2,
		},
		ConfidenceLevel: 0.8,
	}
}

// CreateMethodRefinement creates a test method refinement proposal.
func (tf *TestFixtures) CreateMethodRefinement(refinementType core.RefinementType) *core.MethodRefinement {
	return &core.MethodRefinement{
		Type: refinementType,
		NewApproach: []core.ApproachStep{
			{
				Description: "Simplified first step combining data collection and initial analysis",
				Tools:       []string{"research", "analysis"},
				Heuristics:  []string{"focus_on_key_metrics"},
			},
			{
				Description: "Direct solution implementation based on proven patterns",
				Tools:       []string{"implementation"},
				Heuristics:  []string{"use_established_patterns"},
			},
		},
		Reasoning:                     "Simplify method to reduce complexity and improve success rate",
		ExpectedComplexityChange:      -3,
		ExpectedSuccessRateImprovement: 15.0,
		RequiredVersion:               "2.0.0",
	}
}

// AssertGoalExists verifies a goal exists with expected properties.
func (tf *TestFixtures) AssertGoalExists(t *testing.T, goalID string, expectedTitle string) {
	goal, err := tf.GoalManager.GetGoal(context.Background(), goalID)
	if err != nil {
		t.Fatalf("Expected goal %s to exist, but got error: %v", goalID, err)
	}
	if goal.Title != expectedTitle {
		t.Errorf("Expected goal title %s, got %s", expectedTitle, goal.Title)
	}
}

// AssertMethodExists verifies a method exists with expected properties.
func (tf *TestFixtures) AssertMethodExists(t *testing.T, methodID string, expectedName string) {
	method, err := tf.MethodManager.GetMethod(context.Background(), methodID)
	if err != nil {
		t.Fatalf("Expected method %s to exist, but got error: %v", methodID, err)
	}
	if method.Name != expectedName {
		t.Errorf("Expected method name %s, got %s", expectedName, method.Name)
	}
}

// AssertObjectiveExists verifies an objective exists with expected properties.
func (tf *TestFixtures) AssertObjectiveExists(t *testing.T, objectiveID string, expectedTitle string) {
	objective, err := tf.ObjectiveManager.GetObjective(context.Background(), objectiveID)
	if err != nil {
		t.Fatalf("Expected objective %s to exist, but got error: %v", objectiveID, err)
	}
	if objective.Title != expectedTitle {
		t.Errorf("Expected objective title %s, got %s", expectedTitle, objective.Title)
	}
}

// CleanupFixtures provides manual cleanup for fixtures (useful in some test scenarios).
func (tf *TestFixtures) CleanupFixtures() {
	if tf.TestDataDir != "" {
		os.RemoveAll(tf.TestDataDir)
	}
}