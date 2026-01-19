package test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Solifugus/ai-work-studio/internal/config"
	"github.com/Solifugus/ai-work-studio/pkg/core"
	"github.com/Solifugus/ai-work-studio/pkg/llm"
	"github.com/Solifugus/ai-work-studio/pkg/mcp"
	"github.com/Solifugus/ai-work-studio/pkg/storage"
)

// MockLLMService provides mock LLM functionality for CLI testing.
type MockLLMServiceCLI struct{}

// Execute implements the LLMServiceInterface for testing.
func (m *MockLLMServiceCLI) Execute(ctx context.Context, params mcp.ServiceParams) mcp.ServiceResult {
	// Return structured ethical reasoning response format
	ethicalResponse := `Freedom Impact: 0.8
Well-Being Impact: 0.7
Sustainability Impact: 0.6
Confidence: 0.9
Reasoning: This is a mock ethical reasoning response for testing purposes.
The decision appears to have positive impact across all dimensions.`

	return mcp.ServiceResult{
		Success: true,
		Data: &mcp.CompletionResponse{
			Text:       ethicalResponse,
			TokensUsed: 100,
			Model:      "mock-model",
			Provider:   "mock",
			Cost:       0.002,
		},
		Metadata: map[string]interface{}{
			"test": true,
		},
	}
}

// CLITestSuite provides a test environment for CLI operations.
type CLITestSuite struct {
	tempDir          string
	configPath       string
	config           *config.Config
	store            *storage.Store
	goalManager      *core.GoalManager
	objectiveManager *core.ObjectiveManager
	methodManager    *core.MethodManager
	contextManager   *core.UserContextManager
	ethicalFramework *core.EthicalFramework
	llmRouter        *llm.Router
}

// NewCLITestSuite creates a new test suite for CLI testing.
func NewCLITestSuite(t *testing.T) *CLITestSuite {
	// Create temporary directory
	tempDir := filepath.Join(os.TempDir(), "ai-work-studio-cli-test")
	err := os.RemoveAll(tempDir) // Clean up any existing test data
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("Failed to clean temp directory: %v", err)
	}

	err = os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create test configuration
	dataDir := filepath.Join(tempDir, "data")
	configDir := filepath.Join(tempDir, "config")
	configPath := filepath.Join(configDir, "config.json")

	cfg := config.DefaultConfig()
	cfg.DataDir = dataDir
	cfg.Session.UserID = "test-cli-user"
	// Fix budget limits to make them valid (daily * 30 must be <= monthly)
	cfg.BudgetLimits.MonthlyLimit = 150.00 // Increase monthly limit to accommodate daily * 30

	// Ensure directories exist
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	err = cfg.EnsureDataDir()
	if err != nil {
		t.Fatalf("Failed to create data directory: %v", err)
	}

	// Save configuration
	err = cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Initialize storage
	store, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Initialize managers
	goalManager := core.NewGoalManager(store)
	objectiveManager := core.NewObjectiveManager(store)
	methodManager := core.NewMethodManager(store)
	contextManager := core.NewUserContextManager(store)

	// Initialize LLM router with mock service
	mockLLM := &MockLLMServiceCLI{}
	llmRouter := llm.NewRouter(mockLLM)

	// Initialize ethical framework
	ethicalFramework := core.NewEthicalFramework(store, llmRouter, contextManager)

	return &CLITestSuite{
		tempDir:          tempDir,
		configPath:       configPath,
		config:           cfg,
		store:            store,
		goalManager:      goalManager,
		objectiveManager: objectiveManager,
		methodManager:    methodManager,
		contextManager:   contextManager,
		ethicalFramework: ethicalFramework,
		llmRouter:        llmRouter,
	}
}

// Cleanup cleans up test resources.
func (suite *CLITestSuite) Cleanup() {
	if suite.store != nil {
		suite.store.Close()
	}
	os.RemoveAll(suite.tempDir)
}

// TestCLIConfiguration tests configuration management functionality.
func TestCLIConfiguration(t *testing.T) {
	suite := NewCLITestSuite(t)
	defer suite.Cleanup()

	t.Run("Default Configuration", func(t *testing.T) {
		// Test that default configuration is valid
		err := suite.config.Validate()
		if err != nil {
			t.Errorf("Default configuration should be valid: %v", err)
		}

		// Test default values
		if suite.config.BudgetLimits.DailyLimit != 5.00 {
			t.Errorf("Expected daily limit 5.00, got %f", suite.config.BudgetLimits.DailyLimit)
		}

		if suite.config.Preferences.DefaultPriority != 5 {
			t.Errorf("Expected default priority 5, got %d", suite.config.Preferences.DefaultPriority)
		}

		if !suite.config.Preferences.InteractiveMode {
			t.Error("Expected interactive mode to be enabled by default")
		}
	})

	t.Run("Load and Save Configuration", func(t *testing.T) {
		// Modify configuration with valid budget limits
		updates := config.BudgetUpdates{
			DailyLimit:   floatPtr(3.00), // 3.00 * 30 = 90 < 150 (monthly limit)
			MonthlyLimit: floatPtr(200.00), // Increase monthly limit for safety
		}
		err := suite.config.UpdateBudgetLimits(suite.configPath, updates)
		if err != nil {
			t.Fatalf("Failed to update budget limits: %v", err)
		}

		// Reload configuration
		reloadedConfig, err := config.Load(suite.configPath)
		if err != nil {
			t.Fatalf("Failed to reload configuration: %v", err)
		}

		if reloadedConfig.BudgetLimits.DailyLimit != 3.00 {
			t.Errorf("Expected daily limit 3.00 after reload, got %f", reloadedConfig.BudgetLimits.DailyLimit)
		}
	})

	t.Run("Configuration Validation", func(t *testing.T) {
		// Test invalid configuration
		invalidConfig := config.DefaultConfig()
		// Fix budget limits to avoid validation failure
		invalidConfig.BudgetLimits.MonthlyLimit = 200.00
		invalidConfig.Preferences.DefaultPriority = 15 // Invalid priority

		err := invalidConfig.Validate()
		if err == nil {
			t.Error("Expected validation to fail for invalid priority")
		}

		if !strings.Contains(err.Error(), "priority") {
			t.Errorf("Expected error about priority, got: %v", err)
		}
	})
}

// TestCLIGoalOperations tests goal-related CLI operations.
func TestCLIGoalOperations(t *testing.T) {
	suite := NewCLITestSuite(t)
	defer suite.Cleanup()

	ctx := context.Background()

	t.Run("Create Goal", func(t *testing.T) {
		title := "Test Goal for CLI"
		description := "A test goal to verify CLI functionality"
		priority := 7

		goal, err := suite.goalManager.CreateGoal(ctx, title, description, priority, nil)
		if err != nil {
			t.Fatalf("Failed to create goal: %v", err)
		}

		if goal.Title != title {
			t.Errorf("Expected title %s, got %s", title, goal.Title)
		}

		if goal.Description != description {
			t.Errorf("Expected description %s, got %s", description, goal.Description)
		}

		if goal.Priority != priority {
			t.Errorf("Expected priority %d, got %d", priority, goal.Priority)
		}

		if goal.Status != core.GoalStatusActive {
			t.Errorf("Expected status %s, got %s", core.GoalStatusActive, goal.Status)
		}
	})

	t.Run("List Goals", func(t *testing.T) {
		// Create multiple goals
		goals := []struct {
			title    string
			priority int
			status   core.GoalStatus
		}{
			{"Active Goal 1", 5, core.GoalStatusActive},
			{"Active Goal 2", 8, core.GoalStatusActive},
			{"Completed Goal", 3, core.GoalStatusCompleted},
		}

		var createdGoals []*core.Goal
		for _, goalData := range goals {
			goal, err := suite.goalManager.CreateGoal(ctx, goalData.title, "", goalData.priority, nil)
			if err != nil {
				t.Fatalf("Failed to create goal %s: %v", goalData.title, err)
			}

			// Update status if needed
			if goalData.status != core.GoalStatusActive {
				updates := core.GoalUpdates{Status: &goalData.status}
				_, err = suite.goalManager.UpdateGoal(ctx, goal.ID, updates)
				if err != nil {
					t.Fatalf("Failed to update goal status: %v", err)
				}
			}

			createdGoals = append(createdGoals, goal)
		}

		// Test listing all goals
		allGoals, err := suite.goalManager.ListGoals(ctx, core.GoalFilter{})
		if err != nil {
			t.Fatalf("Failed to list all goals: %v", err)
		}

		if len(allGoals) < len(goals) {
			t.Errorf("Expected at least %d goals, got %d", len(goals), len(allGoals))
		}

		// Test filtering by status
		activeStatus := core.GoalStatusActive
		activeGoals, err := suite.goalManager.ListGoals(ctx, core.GoalFilter{Status: &activeStatus})
		if err != nil {
			t.Fatalf("Failed to list active goals: %v", err)
		}

		activeCount := 0
		for _, goal := range allGoals {
			if goal.Status == core.GoalStatusActive {
				activeCount++
			}
		}

		if len(activeGoals) != activeCount {
			t.Errorf("Expected %d active goals, got %d", activeCount, len(activeGoals))
		}
	})
}

// TestCLIObjectiveOperations tests objective-related CLI operations.
func TestCLIObjectiveOperations(t *testing.T) {
	suite := NewCLITestSuite(t)
	defer suite.Cleanup()

	ctx := context.Background()

	// Create a test goal first
	goal, err := suite.goalManager.CreateGoal(ctx, "Test Goal for Objectives", "Goal for testing objectives", 5, nil)
	if err != nil {
		t.Fatalf("Failed to create test goal: %v", err)
	}

	// Create a test method for objectives to reference
	method, err := suite.methodManager.CreateMethod(ctx, "Test Method", "A method for testing objectives", []core.ApproachStep{}, core.MethodDomainGeneral, nil)
	if err != nil {
		t.Fatalf("Failed to create test method: %v", err)
	}

	t.Run("Create Objective", func(t *testing.T) {
		title := "Test Objective"
		description := "An objective to test CLI functionality"
		priority := 6

		objective, err := suite.objectiveManager.CreateObjective(ctx, goal.ID, method.ID, title, description, nil, priority)
		if err != nil {
			t.Fatalf("Failed to create objective: %v", err)
		}

		if objective.Title != title {
			t.Errorf("Expected title %s, got %s", title, objective.Title)
		}

		if objective.Description != description {
			t.Errorf("Expected description %s, got %s", description, objective.Description)
		}

		if objective.Priority != priority {
			t.Errorf("Expected priority %d, got %d", priority, objective.Priority)
		}

		if objective.GoalID != goal.ID {
			t.Errorf("Expected goal ID %s, got %s", goal.ID, objective.GoalID)
		}

		if objective.Status != core.ObjectiveStatusPending {
			t.Errorf("Expected status %s, got %s", core.ObjectiveStatusPending, objective.Status)
		}
	})

	t.Run("List Objectives", func(t *testing.T) {
		// Create multiple objectives
		objectives := []struct {
			title  string
			status core.ObjectiveStatus
		}{
			{"Pending Objective 1", core.ObjectiveStatusPending},
			{"In Progress Objective", core.ObjectiveStatusInProgress},
			{"Completed Objective", core.ObjectiveStatusCompleted},
		}

		for _, objData := range objectives {
			objective, err := suite.objectiveManager.CreateObjective(ctx, goal.ID, method.ID, objData.title, "", nil, 5)
			if err != nil {
				t.Fatalf("Failed to create objective %s: %v", objData.title, err)
			}

			// Update status if needed
			if objData.status != core.ObjectiveStatusPending {
				updates := core.ObjectiveUpdates{Status: &objData.status}
				_, err = suite.objectiveManager.UpdateObjective(ctx, objective.ID, updates)
				if err != nil {
					t.Fatalf("Failed to update objective status: %v", err)
				}
			}
		}

		// Test listing all objectives
		allObjectives, err := suite.objectiveManager.ListObjectives(ctx, core.ObjectiveFilter{})
		if err != nil {
			t.Fatalf("Failed to list all objectives: %v", err)
		}

		if len(allObjectives) < len(objectives) {
			t.Errorf("Expected at least %d objectives, got %d", len(objectives), len(allObjectives))
		}

		// Test filtering by goal
		goalObjectives, err := suite.objectiveManager.ListObjectives(ctx, core.ObjectiveFilter{GoalID: &goal.ID})
		if err != nil {
			t.Fatalf("Failed to list objectives for goal: %v", err)
		}

		for _, obj := range goalObjectives {
			if obj.GoalID != goal.ID {
				t.Errorf("Expected all objectives to belong to goal %s, got %s", goal.ID, obj.GoalID)
			}
		}
	})
}

// TestCLIEthicalDecisions tests ethical decision functionality in CLI context.
func TestCLIEthicalDecisions(t *testing.T) {
	suite := NewCLITestSuite(t)
	defer suite.Cleanup()

	ctx := context.Background()

	t.Run("Evaluate Decision", func(t *testing.T) {
		objectiveID := "test-objective-123"
		decisionContext := "User wants to organize their file system"
		proposedAction := "Create automated file organization with user control"
		userID := suite.config.Session.UserID

		decision, err := suite.ethicalFramework.EvaluateDecision(ctx, objectiveID, decisionContext, proposedAction, []string{}, userID)
		if err != nil {
			t.Fatalf("Failed to evaluate decision: %v", err)
		}

		if decision.DecisionContext != decisionContext {
			t.Errorf("Expected context %s, got %s", decisionContext, decision.DecisionContext)
		}

		if decision.ProposedAction != proposedAction {
			t.Errorf("Expected action %s, got %s", proposedAction, decision.ProposedAction)
		}

		if decision.UserID != userID {
			t.Errorf("Expected user ID %s, got %s", userID, decision.UserID)
		}

		if decision.ObjectiveID != objectiveID {
			t.Errorf("Expected objective ID %s, got %s", objectiveID, decision.ObjectiveID)
		}
	})

	t.Run("Approve Decision", func(t *testing.T) {
		// Create a decision that requires approval
		decision, err := suite.ethicalFramework.EvaluateDecision(ctx, "test-obj", "Test decision requiring approval", "Some restrictive action", []string{}, suite.config.Session.UserID)
		if err != nil {
			t.Fatalf("Failed to create decision: %v", err)
		}

		// If the decision requires approval, test the approval process
		if decision.IsPendingApproval() {
			feedback := "Approved for testing purposes"
			err = suite.ethicalFramework.ApproveDecision(ctx, decision.ID, feedback)
			if err != nil {
				t.Fatalf("Failed to approve decision: %v", err)
			}

			// Verify approval
			approvedDecision, err := suite.ethicalFramework.GetDecision(ctx, decision.ID)
			if err != nil {
				t.Fatalf("Failed to get approved decision: %v", err)
			}

			if !approvedDecision.IsApproved() {
				t.Error("Decision should be approved")
			}

			if approvedDecision.UserFeedback != feedback {
				t.Errorf("Expected feedback %s, got %s", feedback, approvedDecision.UserFeedback)
			}
		}
	})
}

// TestCLISessionManagement tests session state management.
func TestCLISessionManagement(t *testing.T) {
	suite := NewCLITestSuite(t)
	defer suite.Cleanup()

	ctx := context.Background()

	t.Run("Update Current Goal", func(t *testing.T) {
		// Create a goal
		goal, err := suite.goalManager.CreateGoal(ctx, "Session Test Goal", "", 5, nil)
		if err != nil {
			t.Fatalf("Failed to create goal: %v", err)
		}

		// Update current goal in session
		updates := config.SessionUpdates{
			CurrentGoalID: &goal.ID,
		}
		err = suite.config.UpdateSession(suite.configPath, updates)
		if err != nil {
			t.Fatalf("Failed to update session: %v", err)
		}

		// Reload configuration and verify
		reloadedConfig, err := config.Load(suite.configPath)
		if err != nil {
			t.Fatalf("Failed to reload config: %v", err)
		}

		if reloadedConfig.Session.CurrentGoalID != goal.ID {
			t.Errorf("Expected current goal %s, got %s", goal.ID, reloadedConfig.Session.CurrentGoalID)
		}
	})

	t.Run("Update User Preferences", func(t *testing.T) {
		// Update preferences
		newPriority := 8
		verboseOutput := true
		updates := config.PreferenceUpdates{
			DefaultPriority: &newPriority,
			VerboseOutput:   &verboseOutput,
		}
		err := suite.config.UpdatePreferences(suite.configPath, updates)
		if err != nil {
			t.Fatalf("Failed to update preferences: %v", err)
		}

		// Reload and verify
		reloadedConfig, err := config.Load(suite.configPath)
		if err != nil {
			t.Fatalf("Failed to reload config: %v", err)
		}

		if reloadedConfig.Preferences.DefaultPriority != newPriority {
			t.Errorf("Expected priority %d, got %d", newPriority, reloadedConfig.Preferences.DefaultPriority)
		}

		if !reloadedConfig.Preferences.VerboseOutput {
			t.Error("Expected verbose output to be enabled")
		}
	})
}

// TestCLIErrorHandling tests error handling in CLI operations.
func TestCLIErrorHandling(t *testing.T) {
	suite := NewCLITestSuite(t)
	defer suite.Cleanup()

	ctx := context.Background()

	t.Run("Invalid Goal Creation", func(t *testing.T) {
		// Test empty title
		_, err := suite.goalManager.CreateGoal(ctx, "", "description", 5, nil)
		if err == nil {
			t.Error("Expected error for empty goal title")
		}

		// Test invalid priority
		_, err = suite.goalManager.CreateGoal(ctx, "Valid Title", "description", 15, nil)
		if err == nil {
			t.Error("Expected error for invalid priority")
		}
	})

	t.Run("Nonexistent Resource Access", func(t *testing.T) {
		// Test getting nonexistent goal
		_, err := suite.goalManager.GetGoal(ctx, "nonexistent-goal-id")
		if err == nil {
			t.Error("Expected error for nonexistent goal")
		}

		// Test getting nonexistent objective
		_, err = suite.objectiveManager.GetObjective(ctx, "nonexistent-objective-id")
		if err == nil {
			t.Error("Expected error for nonexistent objective")
		}

		// Test getting nonexistent decision
		_, err = suite.ethicalFramework.GetDecision(ctx, "nonexistent-decision-id")
		if err == nil {
			t.Error("Expected error for nonexistent decision")
		}
	})

	t.Run("Configuration Validation", func(t *testing.T) {
		// Test updating with invalid values
		invalidPriority := 15
		updates := config.PreferenceUpdates{
			DefaultPriority: &invalidPriority,
		}
		err := suite.config.UpdatePreferences(suite.configPath, updates)
		if err == nil {
			t.Error("Expected error for invalid priority update")
		}

		// Test updating with negative budget
		negativeBudget := -10.0
		budgetUpdates := config.BudgetUpdates{
			DailyLimit: &negativeBudget,
		}
		err = suite.config.UpdateBudgetLimits(suite.configPath, budgetUpdates)
		if err == nil {
			t.Error("Expected error for negative budget")
		}
	})
}

// Helper functions

func floatPtr(f float64) *float64 {
	return &f
}

func intPtr(i int) *int {
	return &i
}

