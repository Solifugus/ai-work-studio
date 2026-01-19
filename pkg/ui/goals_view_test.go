package ui

import (
	"context"
	"testing"
	"time"

	"github.com/Solifugus/ai-work-studio/internal/config"
	"github.com/Solifugus/ai-work-studio/pkg/core"
)

// setupTestApp creates a test application with temporary storage
func setupTestApp(t *testing.T) *App {
	tempDir := t.TempDir()

	cfg := &config.Config{
		DataDir: tempDir,
		Session: config.SessionState{
			UserID: "test-user",
		},
	}

	app, err := NewApp(cfg, tempDir+"/config.json")
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}

	return app
}

// createTestGoals creates a set of test goals with hierarchical relationships
func createTestGoals(t *testing.T, app *App) []*core.Goal {
	ctx := context.Background()
	gm := app.GetGoalManager()

	// Create parent goals
	parentGoal1, err := gm.CreateGoal(ctx, "Complete Project Alpha", "Major project milestone", 2,
		map[string]interface{}{"category": "work"})
	if err != nil {
		t.Fatalf("Failed to create parent goal 1: %v", err)
	}

	parentGoal2, err := gm.CreateGoal(ctx, "Personal Development", "Self-improvement goals", 4,
		map[string]interface{}{"category": "personal"})
	if err != nil {
		t.Fatalf("Failed to create parent goal 2: %v", err)
	}

	// Create child goals
	childGoal1, err := gm.CreateGoal(ctx, "Design Architecture", "Design system architecture", 3,
		map[string]interface{}{"category": "work"})
	if err != nil {
		t.Fatalf("Failed to create child goal 1: %v", err)
	}

	childGoal2, err := gm.CreateGoal(ctx, "Write Tests", "Implement comprehensive tests", 5,
		map[string]interface{}{"category": "work"})
	if err != nil {
		t.Fatalf("Failed to create child goal 2: %v", err)
	}

	childGoal3, err := gm.CreateGoal(ctx, "Learn Go", "Master Go programming language", 6,
		map[string]interface{}{"category": "personal"})
	if err != nil {
		t.Fatalf("Failed to create child goal 3: %v", err)
	}

	// Create goal hierarchy
	if err := gm.AddSubGoal(ctx, parentGoal1.ID, childGoal1.ID); err != nil {
		t.Fatalf("Failed to add subgoal 1: %v", err)
	}
	if err := gm.AddSubGoal(ctx, parentGoal1.ID, childGoal2.ID); err != nil {
		t.Fatalf("Failed to add subgoal 2: %v", err)
	}
	if err := gm.AddSubGoal(ctx, parentGoal2.ID, childGoal3.ID); err != nil {
		t.Fatalf("Failed to add subgoal 3: %v", err)
	}

	// Update some goal statuses
	pausedStatus := core.GoalStatusPaused
	if _, err := gm.UpdateGoal(ctx, childGoal2.ID, core.GoalUpdates{Status: &pausedStatus}); err != nil {
		t.Fatalf("Failed to update goal status: %v", err)
	}

	return []*core.Goal{parentGoal1, parentGoal2, childGoal1, childGoal2, childGoal3}
}

func TestGoalsView_Creation(t *testing.T) {
	app := setupTestApp(t)
	defer app.Stop()

	// Create a mock parent window (nil is acceptable for testing)
	gv := NewGoalsView(app, nil)

	if gv == nil {
		t.Fatal("NewGoalsView returned nil")
	}

	if gv.app != app {
		t.Error("GoalsView app reference not set correctly")
	}

	if gv.goalNodes == nil {
		t.Error("GoalsView goalNodes map not initialized")
	}

	if gv.container == nil {
		t.Error("GoalsView container not created")
	}

	if gv.goalsTree == nil {
		t.Error("GoalsView tree widget not created")
	}
}

func TestGoalsView_TreeStructureBuilding(t *testing.T) {
	app := setupTestApp(t)
	defer app.Stop()

	// Create test goals
	goals := createTestGoals(t, app)

	// Create goals view
	gv := NewGoalsView(app, nil)

	// Wait a moment for async operations
	time.Sleep(100 * time.Millisecond)

	// Test that goal nodes were created
	if len(gv.goalNodes) != len(goals) {
		t.Errorf("Expected %d goal nodes, got %d", len(goals), len(gv.goalNodes))
	}

	// Test that root goals were identified
	if len(gv.rootGoals) != 2 {
		t.Errorf("Expected 2 root goals, got %d", len(gv.rootGoals))
	}

	// Test hierarchical relationships
	var parentGoal1ID string
	for _, goal := range goals {
		if goal.Title == "Complete Project Alpha" {
			parentGoal1ID = goal.ID
			break
		}
	}

	if parentGoal1ID == "" {
		t.Fatal("Could not find parent goal 1")
	}

	parentNode, exists := gv.goalNodes[parentGoal1ID]
	if !exists {
		t.Fatal("Parent goal node not found in goalNodes map")
	}

	if len(parentNode.Children) != 2 {
		t.Errorf("Expected parent goal to have 2 children, got %d", len(parentNode.Children))
	}
}

func TestGoalsView_SearchFiltering(t *testing.T) {
	app := setupTestApp(t)
	defer app.Stop()

	createTestGoals(t, app)
	gv := NewGoalsView(app, nil)

	// Wait for initial data load
	time.Sleep(100 * time.Millisecond)

	tests := []struct {
		name         string
		searchFilter string
		expectedMin  int // Minimum expected results
		expectedMax  int // Maximum expected results
	}{
		{
			name:         "empty search returns all",
			searchFilter: "",
			expectedMin:  5,
			expectedMax:  5,
		},
		{
			name:         "search for 'project'",
			searchFilter: "project",
			expectedMin:  1,
			expectedMax:  3, // Could match multiple goals
		},
		{
			name:         "search for 'tests'",
			searchFilter: "tests",
			expectedMin:  1,
			expectedMax:  1,
		},
		{
			name:         "search for nonexistent",
			searchFilter: "nonexistent",
			expectedMin:  0,
			expectedMax:  0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Apply search filter
			gv.searchFilter = test.searchFilter
			gv.applyFiltersAndSort()

			// Count visible goals by checking root goals and their children
			visibleCount := len(gv.rootGoals)
			for _, rootID := range gv.rootGoals {
				if node, exists := gv.goalNodes[rootID]; exists {
					visibleCount += len(node.Children)
				}
			}

			if visibleCount < test.expectedMin || visibleCount > test.expectedMax {
				t.Errorf("Search '%s': expected %d-%d results, got %d",
					test.searchFilter, test.expectedMin, test.expectedMax, visibleCount)
			}
		})
	}
}

func TestGoalsView_StatusFiltering(t *testing.T) {
	app := setupTestApp(t)
	defer app.Stop()

	createTestGoals(t, app)
	gv := NewGoalsView(app, nil)

	// Wait for initial data load
	time.Sleep(100 * time.Millisecond)

	tests := []struct {
		name         string
		statusFilter core.GoalStatus
		expectedMin  int
		expectedMax  int
	}{
		{
			name:         "filter active goals",
			statusFilter: core.GoalStatusActive,
			expectedMin:  4,
			expectedMax:  4,
		},
		{
			name:         "filter paused goals",
			statusFilter: core.GoalStatusPaused,
			expectedMin:  1,
			expectedMax:  1,
		},
		{
			name:         "filter completed goals",
			statusFilter: core.GoalStatusCompleted,
			expectedMin:  0,
			expectedMax:  0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Apply status filter
			gv.statusFilter = test.statusFilter
			gv.applyFiltersAndSort()

			// Count filtered goals
			filteredCount := len(gv.rootGoals)
			for _, rootID := range gv.rootGoals {
				if node, exists := gv.goalNodes[rootID]; exists {
					filteredCount += len(node.Children)
				}
			}

			if filteredCount < test.expectedMin || filteredCount > test.expectedMax {
				t.Errorf("Status filter '%s': expected %d-%d results, got %d",
					test.statusFilter, test.expectedMin, test.expectedMax, filteredCount)
			}
		})
	}
}

func TestGoalsView_Sorting(t *testing.T) {
	app := setupTestApp(t)
	defer app.Stop()

	createTestGoals(t, app)
	gv := NewGoalsView(app, nil)

	// Wait for initial data load
	time.Sleep(100 * time.Millisecond)

	tests := []struct {
		name     string
		sortMode string
	}{
		{"sort by priority", "priority"},
		{"sort by title", "title"},
		{"sort by created", "created"},
		{"sort by status", "status"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Apply sort
			gv.sortMode = test.sortMode
			gv.applyFiltersAndSort()

			// Verify that root goals are in some order (specific order testing would
			// require more complex validation)
			if len(gv.rootGoals) == 0 {
				t.Error("No root goals after sorting")
			}

			// Ensure all root goals still exist in goalNodes
			for _, rootID := range gv.rootGoals {
				if _, exists := gv.goalNodes[rootID]; !exists {
					t.Errorf("Root goal %s not found in goalNodes after sorting", rootID)
				}
			}
		})
	}
}

func TestGoalsView_DataRefresh(t *testing.T) {
	app := setupTestApp(t)
	defer app.Stop()

	gv := NewGoalsView(app, nil)

	// Initial state should be empty
	if len(gv.goals) > 0 {
		t.Error("Goals view should start with no goals")
	}

	// Create goals after view creation
	createTestGoals(t, app)

	// Refresh data
	gv.refreshData()

	// Wait for async operations
	time.Sleep(100 * time.Millisecond)

	// Verify goals were loaded
	if len(gv.goals) == 0 {
		t.Error("No goals loaded after refresh")
	}

	if len(gv.goalNodes) == 0 {
		t.Error("No goal nodes created after refresh")
	}

	if len(gv.rootGoals) == 0 {
		t.Error("No root goals identified after refresh")
	}
}

func TestGoalsView_UpdateStatusBar(t *testing.T) {
	app := setupTestApp(t)
	defer app.Stop()

	gv := NewGoalsView(app, nil)

	// Test default status
	gv.updateStatusBar()
	if gv.statusLabel.Text != "Ready" {
		t.Errorf("Expected default status 'Ready', got '%s'", gv.statusLabel.Text)
	}

	// Test custom message
	gv.updateStatusBar("Loading...")
	if gv.statusLabel.Text != "Loading..." {
		t.Errorf("Expected status 'Loading...', got '%s'", gv.statusLabel.Text)
	}

	// Test with selected goal (need to create goals first)
	goals := createTestGoals(t, app)
	gv.refreshData()
	time.Sleep(100 * time.Millisecond)

	gv.selectedGoalID = goals[0].ID
	gv.updateStatusBar()

	if gv.statusLabel.Text == "Ready" || gv.statusLabel.Text == "" {
		t.Error("Status bar should show goal details when goal is selected")
	}
}

func TestGoalsView_StatusIconMapping(t *testing.T) {
	app := setupTestApp(t)
	defer app.Stop()

	gv := NewGoalsView(app, nil)

	tests := []struct {
		status       core.GoalStatus
		expectNonNil bool
	}{
		{core.GoalStatusActive, true},
		{core.GoalStatusPaused, true},
		{core.GoalStatusCompleted, true},
		{core.GoalStatusArchived, true},
	}

	for _, test := range tests {
		t.Run(string(test.status), func(t *testing.T) {
			icon := gv.getStatusIcon(test.status)
			if test.expectNonNil && icon == nil {
				t.Errorf("Expected non-nil icon for status %s", test.status)
			}
		})
	}
}

// TestGoalsView_Integration performs a more comprehensive integration test
func TestGoalsView_Integration(t *testing.T) {
	app := setupTestApp(t)
	defer app.Stop()

	// Create goals view
	gv := NewGoalsView(app, nil)

	// Create test data
	goals := createTestGoals(t, app)

	// Refresh to load the data
	gv.refreshData()
	time.Sleep(100 * time.Millisecond)

	// Verify initial state
	if len(gv.goals) != 5 {
		t.Errorf("Expected 5 goals, got %d", len(gv.goals))
	}

	if len(gv.rootGoals) != 2 {
		t.Errorf("Expected 2 root goals, got %d", len(gv.rootGoals))
	}

	// Test filtering and sorting combination
	gv.searchFilter = "goal"
	gv.statusFilter = core.GoalStatusActive
	gv.sortMode = "priority"
	gv.applyFiltersAndSort()

	// Should still have valid structure
	if len(gv.goalNodes) == 0 {
		t.Error("No goal nodes after combined filtering")
	}

	// Test goal selection
	if len(goals) > 0 {
		gv.selectedGoalID = goals[0].ID
		gv.updateStatusBar()

		if gv.statusLabel.Text == "Ready" {
			t.Error("Status bar should show goal information when goal is selected")
		}
	}

	// Reset filters
	gv.searchFilter = ""
	gv.statusFilter = ""
	gv.applyFiltersAndSort()

	// Should return to original state
	if len(gv.rootGoals) != 2 {
		t.Errorf("Expected 2 root goals after reset, got %d", len(gv.rootGoals))
	}
}

// Benchmark tests
func BenchmarkGoalsView_RefreshData(b *testing.B) {
	tempDir := b.TempDir()
	cfg := &config.Config{
		DataDir: tempDir,
		Session: config.SessionState{
			UserID: "test-user",
		},
	}
	app, err := NewApp(cfg, tempDir+"/config.json")
	if err != nil {
		b.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	gv := NewGoalsView(app, nil)

	// Create test goals
	ctx := context.Background()
	gm := app.GetGoalManager()
	goal1, _ := gm.CreateGoal(ctx, "Test Goal 1", "Description 1", 1, nil)
	goal2, _ := gm.CreateGoal(ctx, "Test Goal 2", "Description 2", 2, nil)
	gm.AddSubGoal(ctx, goal1.ID, goal2.ID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gv.refreshData()
	}
}

func BenchmarkGoalsView_ApplyFiltersAndSort(b *testing.B) {
	tempDir := b.TempDir()
	cfg := &config.Config{
		DataDir: tempDir,
		Session: config.SessionState{
			UserID: "test-user",
		},
	}
	app, err := NewApp(cfg, tempDir+"/config.json")
	if err != nil {
		b.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	gv := NewGoalsView(app, nil)

	// Create test goals
	ctx := context.Background()
	gm := app.GetGoalManager()
	goal1, _ := gm.CreateGoal(ctx, "Test Goal 1", "Description 1", 1, nil)
	goal2, _ := gm.CreateGoal(ctx, "Test Goal 2", "Description 2", 2, nil)
	gm.AddSubGoal(ctx, goal1.ID, goal2.ID)

	gv.refreshData()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gv.applyFiltersAndSort()
	}
}