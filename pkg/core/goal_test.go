package core

import (
	"context"
	"testing"
	"time"

	"github.com/Solifugus/ai-work-studio/pkg/storage"
)

// setupTestStore creates a temporary storage for testing
func setupTestStore(t *testing.T) *storage.Store {
	tempDir := t.TempDir()
	store, err := storage.NewStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}
	return store
}

func TestGoalManager_CreateGoal(t *testing.T) {
	store := setupTestStore(t)
	gm := NewGoalManager(store)
	ctx := context.Background()

	tests := []struct {
		name        string
		title       string
		description string
		priority    int
		userContext map[string]interface{}
		expectError bool
	}{
		{
			name:        "valid goal",
			title:       "Complete project",
			description: "Finish the AI Work Studio project",
			priority:    5,
			userContext: map[string]interface{}{"tags": []string{"work", "important"}},
			expectError: false,
		},
		{
			name:        "empty title",
			title:       "",
			description: "Some description",
			priority:    5,
			expectError: true,
		},
		{
			name:        "invalid priority too low",
			title:       "Test goal",
			description: "Test description",
			priority:    0,
			expectError: true,
		},
		{
			name:        "invalid priority too high",
			title:       "Test goal",
			description: "Test description",
			priority:    11,
			expectError: true,
		},
		{
			name:        "minimal valid goal",
			title:       "Simple goal",
			description: "",
			priority:    1,
			userContext: nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goal, err := gm.CreateGoal(ctx, tt.title, tt.description, tt.priority, tt.userContext)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify goal properties
			if goal.Title != tt.title {
				t.Errorf("Expected title %q, got %q", tt.title, goal.Title)
			}
			if goal.Description != tt.description {
				t.Errorf("Expected description %q, got %q", tt.description, goal.Description)
			}
			if goal.Priority != tt.priority {
				t.Errorf("Expected priority %d, got %d", tt.priority, goal.Priority)
			}
			if goal.Status != GoalStatusActive {
				t.Errorf("Expected status %v, got %v", GoalStatusActive, goal.Status)
			}
			if goal.ID == "" {
				t.Errorf("Goal ID should not be empty")
			}
			if goal.CreatedAt.IsZero() {
				t.Errorf("CreatedAt should be set")
			}

			// Verify user context (if provided)
			if tt.userContext != nil {
				if goal.UserContext == nil {
					t.Errorf("Expected user context to be set")
				}
			}
		})
	}
}

func TestGoalManager_GetGoal(t *testing.T) {
	store := setupTestStore(t)
	gm := NewGoalManager(store)
	ctx := context.Background()

	// Create a test goal
	originalGoal, err := gm.CreateGoal(ctx, "Test Goal", "Test Description", 5, map[string]interface{}{"test": true})
	if err != nil {
		t.Fatalf("Failed to create test goal: %v", err)
	}

	// Test retrieving the goal
	retrievedGoal, err := gm.GetGoal(ctx, originalGoal.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve goal: %v", err)
	}

	// Verify all fields match
	if retrievedGoal.ID != originalGoal.ID {
		t.Errorf("Expected ID %q, got %q", originalGoal.ID, retrievedGoal.ID)
	}
	if retrievedGoal.Title != originalGoal.Title {
		t.Errorf("Expected title %q, got %q", originalGoal.Title, retrievedGoal.Title)
	}
	if retrievedGoal.Description != originalGoal.Description {
		t.Errorf("Expected description %q, got %q", originalGoal.Description, retrievedGoal.Description)
	}
	if retrievedGoal.Status != originalGoal.Status {
		t.Errorf("Expected status %v, got %v", originalGoal.Status, retrievedGoal.Status)
	}
	if retrievedGoal.Priority != originalGoal.Priority {
		t.Errorf("Expected priority %d, got %d", originalGoal.Priority, retrievedGoal.Priority)
	}

	// Test retrieving non-existent goal
	_, err = gm.GetGoal(ctx, "non-existent-id")
	if err == nil {
		t.Errorf("Expected error when retrieving non-existent goal")
	}
}

func TestGoalManager_UpdateGoal(t *testing.T) {
	store := setupTestStore(t)
	gm := NewGoalManager(store)
	ctx := context.Background()

	// Create a test goal
	originalGoal, err := gm.CreateGoal(ctx, "Original Title", "Original Description", 5, map[string]interface{}{"original": true})
	if err != nil {
		t.Fatalf("Failed to create test goal: %v", err)
	}

	// Test updating different fields
	tests := []struct {
		name    string
		updates GoalUpdates
		verify  func(t *testing.T, goal *Goal)
	}{
		{
			name: "update title",
			updates: GoalUpdates{
				Title: stringPtr("Updated Title"),
			},
			verify: func(t *testing.T, goal *Goal) {
				if goal.Title != "Updated Title" {
					t.Errorf("Expected title to be updated to 'Updated Title', got %q", goal.Title)
				}
				// Other fields should remain unchanged
				if goal.Description != "Original Description" {
					t.Errorf("Description should not have changed")
				}
			},
		},
		{
			name: "update status",
			updates: GoalUpdates{
				Status: statusPtr(GoalStatusCompleted),
			},
			verify: func(t *testing.T, goal *Goal) {
				if goal.Status != GoalStatusCompleted {
					t.Errorf("Expected status to be completed, got %v", goal.Status)
				}
			},
		},
		{
			name: "update priority",
			updates: GoalUpdates{
				Priority: intPtr(8),
			},
			verify: func(t *testing.T, goal *Goal) {
				if goal.Priority != 8 {
					t.Errorf("Expected priority to be 8, got %d", goal.Priority)
				}
			},
		},
		{
			name: "update multiple fields",
			updates: GoalUpdates{
				Title:       stringPtr("Multi Update"),
				Description: stringPtr("Updated Description"),
				Priority:    intPtr(10),
			},
			verify: func(t *testing.T, goal *Goal) {
				if goal.Title != "Multi Update" {
					t.Errorf("Expected title to be 'Multi Update', got %q", goal.Title)
				}
				if goal.Description != "Updated Description" {
					t.Errorf("Expected description to be 'Updated Description', got %q", goal.Description)
				}
				if goal.Priority != 10 {
					t.Errorf("Expected priority to be 10, got %d", goal.Priority)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updatedGoal, err := gm.UpdateGoal(ctx, originalGoal.ID, tt.updates)
			if err != nil {
				t.Fatalf("Failed to update goal: %v", err)
			}

			tt.verify(t, updatedGoal)

			// Verify the goal is actually updated in storage
			retrievedGoal, err := gm.GetGoal(ctx, originalGoal.ID)
			if err != nil {
				t.Fatalf("Failed to retrieve updated goal: %v", err)
			}
			tt.verify(t, retrievedGoal)

			// Update original goal reference for next test
			originalGoal = updatedGoal
		})
	}
}

func TestGoalManager_UpdateGoal_InvalidInputs(t *testing.T) {
	store := setupTestStore(t)
	gm := NewGoalManager(store)
	ctx := context.Background()

	// Create a test goal
	goal, err := gm.CreateGoal(ctx, "Test Goal", "Test Description", 5, nil)
	if err != nil {
		t.Fatalf("Failed to create test goal: %v", err)
	}

	tests := []struct {
		name    string
		updates GoalUpdates
	}{
		{
			name: "empty title",
			updates: GoalUpdates{
				Title: stringPtr(""),
			},
		},
		{
			name: "invalid priority too low",
			updates: GoalUpdates{
				Priority: intPtr(0),
			},
		},
		{
			name: "invalid priority too high",
			updates: GoalUpdates{
				Priority: intPtr(11),
			},
		},
		{
			name: "invalid status",
			updates: GoalUpdates{
				Status: statusPtr(GoalStatus("invalid")),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := gm.UpdateGoal(ctx, goal.ID, tt.updates)
			if err == nil {
				t.Errorf("Expected error for invalid update")
			}
		})
	}
}

func TestGoalManager_ListGoals(t *testing.T) {
	store := setupTestStore(t)
	gm := NewGoalManager(store)
	ctx := context.Background()

	// Create test goals with different properties
	goals := []struct {
		title    string
		status   GoalStatus
		priority int
	}{
		{"Active Goal 1", GoalStatusActive, 5},
		{"Active Goal 2", GoalStatusActive, 8},
		{"Completed Goal", GoalStatusCompleted, 3},
		{"Paused Goal", GoalStatusPaused, 7},
		{"High Priority Goal", GoalStatusActive, 10},
	}

	var createdGoals []*Goal
	for _, g := range goals {
		goal, err := gm.CreateGoal(ctx, g.title, "Description", g.priority, nil)
		if err != nil {
			t.Fatalf("Failed to create test goal %s: %v", g.title, err)
		}

		// Update status if not active (since goals are created as active by default)
		if g.status != GoalStatusActive {
			_, err = gm.UpdateGoal(ctx, goal.ID, GoalUpdates{Status: &g.status})
			if err != nil {
				t.Fatalf("Failed to update goal status: %v", err)
			}
		}

		createdGoals = append(createdGoals, goal)
	}

	tests := []struct {
		name           string
		filter         GoalFilter
		expectedCount  int
		verifyResults  func(t *testing.T, goals []*Goal)
	}{
		{
			name:          "all goals",
			filter:        GoalFilter{},
			expectedCount: 5,
		},
		{
			name:          "active goals only",
			filter:        GoalFilter{Status: statusPtr(GoalStatusActive)},
			expectedCount: 3,
			verifyResults: func(t *testing.T, goals []*Goal) {
				for _, goal := range goals {
					if goal.Status != GoalStatusActive {
						t.Errorf("Expected all goals to be active, found %v", goal.Status)
					}
				}
			},
		},
		{
			name:          "completed goals only",
			filter:        GoalFilter{Status: statusPtr(GoalStatusCompleted)},
			expectedCount: 1,
		},
		{
			name:          "high priority goals",
			filter:        GoalFilter{MinPriority: intPtr(8)},
			expectedCount: 2,
			verifyResults: func(t *testing.T, goals []*Goal) {
				for _, goal := range goals {
					if goal.Priority < 8 {
						t.Errorf("Expected priority >= 8, got %d", goal.Priority)
					}
				}
			},
		},
		{
			name:          "priority range",
			filter:        GoalFilter{MinPriority: intPtr(5), MaxPriority: intPtr(8)},
			expectedCount: 3,
			verifyResults: func(t *testing.T, goals []*Goal) {
				for _, goal := range goals {
					if goal.Priority < 5 || goal.Priority > 8 {
						t.Errorf("Expected priority between 5-8, got %d", goal.Priority)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := gm.ListGoals(ctx, tt.filter)
			if err != nil {
				t.Fatalf("Failed to list goals: %v", err)
			}

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d goals, got %d", tt.expectedCount, len(results))
			}

			if tt.verifyResults != nil {
				tt.verifyResults(t, results)
			}
		})
	}
}

func TestGoalManager_GoalHierarchy(t *testing.T) {
	store := setupTestStore(t)
	gm := NewGoalManager(store)
	ctx := context.Background()

	// Create test goals
	parentGoal, err := gm.CreateGoal(ctx, "Parent Goal", "Main objective", 10, nil)
	if err != nil {
		t.Fatalf("Failed to create parent goal: %v", err)
	}

	subGoal1, err := gm.CreateGoal(ctx, "Sub Goal 1", "First sub-objective", 8, nil)
	if err != nil {
		t.Fatalf("Failed to create sub goal 1: %v", err)
	}

	subGoal2, err := gm.CreateGoal(ctx, "Sub Goal 2", "Second sub-objective", 6, nil)
	if err != nil {
		t.Fatalf("Failed to create sub goal 2: %v", err)
	}

	// Test adding sub-goals
	err = gm.AddSubGoal(ctx, parentGoal.ID, subGoal1.ID)
	if err != nil {
		t.Fatalf("Failed to add sub goal 1: %v", err)
	}

	err = gm.AddSubGoal(ctx, parentGoal.ID, subGoal2.ID)
	if err != nil {
		t.Fatalf("Failed to add sub goal 2: %v", err)
	}

	// Test retrieving sub-goals
	subGoals, err := gm.GetSubGoals(ctx, parentGoal.ID)
	if err != nil {
		t.Fatalf("Failed to get sub goals: %v", err)
	}

	if len(subGoals) != 2 {
		t.Errorf("Expected 2 sub goals, got %d", len(subGoals))
	}

	// Verify sub-goal IDs
	subGoalIDs := make(map[string]bool)
	for _, sg := range subGoals {
		subGoalIDs[sg.ID] = true
	}
	if !subGoalIDs[subGoal1.ID] {
		t.Errorf("Sub goal 1 not found in results")
	}
	if !subGoalIDs[subGoal2.ID] {
		t.Errorf("Sub goal 2 not found in results")
	}

	// Test retrieving parent goals
	parentGoals, err := gm.GetParentGoals(ctx, subGoal1.ID)
	if err != nil {
		t.Fatalf("Failed to get parent goals: %v", err)
	}

	if len(parentGoals) != 1 {
		t.Errorf("Expected 1 parent goal, got %d", len(parentGoals))
	}

	if parentGoals[0].ID != parentGoal.ID {
		t.Errorf("Expected parent goal ID %s, got %s", parentGoal.ID, parentGoals[0].ID)
	}

	// Test error cases
	err = gm.AddSubGoal(ctx, "non-existent", subGoal1.ID)
	if err == nil {
		t.Errorf("Expected error when adding sub-goal to non-existent parent")
	}

	err = gm.AddSubGoal(ctx, parentGoal.ID, "non-existent")
	if err == nil {
		t.Errorf("Expected error when adding non-existent sub-goal")
	}
}

func TestGoalManager_TemporalQueries(t *testing.T) {
	store := setupTestStore(t)
	gm := NewGoalManager(store)
	ctx := context.Background()

	// Create a goal and track time points
	timeBeforeCreation := time.Now().Add(-1 * time.Second)

	goal, err := gm.CreateGoal(ctx, "Temporal Test Goal", "Testing temporal features", 5, nil)
	if err != nil {
		t.Fatalf("Failed to create goal: %v", err)
	}

	// Take timestamp that's definitely after creation but before update
	timeAfterCreation := time.Now()
	time.Sleep(50 * time.Millisecond) // Ensure clear separation

	// Update the goal
	_, err = gm.UpdateGoal(ctx, goal.ID, GoalUpdates{
		Title: stringPtr("Updated Temporal Goal"),
	})
	if err != nil {
		t.Fatalf("Failed to update goal: %v", err)
	}

	timeAfterUpdate := time.Now()

	// Test retrieving goal at different times (use timestamp taken before the update)
	goalAtCreation, err := gm.GetGoalAtTime(ctx, goal.ID, timeAfterCreation)
	if err != nil {
		t.Fatalf("Failed to get goal at creation time: %v", err)
	}

	if goalAtCreation.Title != "Temporal Test Goal" {
		t.Errorf("Expected original title at creation time, got %q", goalAtCreation.Title)
	}

	// Test retrieving goal before it existed should fail
	_, err = gm.GetGoalAtTime(ctx, goal.ID, timeBeforeCreation)
	if err == nil {
		t.Errorf("Expected error when retrieving goal before it existed")
	}

	// Get current goal should have updated title
	currentGoal, err := gm.GetGoal(ctx, goal.ID)
	if err != nil {
		t.Fatalf("Failed to get current goal: %v", err)
	}

	if currentGoal.Title != "Updated Temporal Goal" {
		t.Errorf("Expected updated title, got %q", currentGoal.Title)
	}

	_ = timeAfterUpdate // Silence unused variable
}

func TestGoal_InstanceMethods(t *testing.T) {
	store := setupTestStore(t)
	gm := NewGoalManager(store)
	ctx := context.Background()

	// Create test goals with different statuses
	goal, err := gm.CreateGoal(ctx, "Test Goal", "Description", 5, nil)
	if err != nil {
		t.Fatalf("Failed to create goal: %v", err)
	}

	// Test IsActive method
	if !goal.IsActive() {
		t.Errorf("New goal should be active")
	}

	if goal.IsCompleted() {
		t.Errorf("New goal should not be completed")
	}

	// Update goal to completed and test
	err = goal.Update(ctx, GoalUpdates{Status: statusPtr(GoalStatusCompleted)})
	if err != nil {
		t.Fatalf("Failed to update goal through instance method: %v", err)
	}

	if goal.IsActive() {
		t.Errorf("Completed goal should not be active")
	}

	if !goal.IsCompleted() {
		t.Errorf("Goal should be completed")
	}

	// Verify the status was actually updated
	if goal.Status != GoalStatusCompleted {
		t.Errorf("Expected status to be completed, got %v", goal.Status)
	}
}

func TestGoalStatus_String(t *testing.T) {
	tests := []struct {
		status   GoalStatus
		expected string
	}{
		{GoalStatusActive, "active"},
		{GoalStatusPaused, "paused"},
		{GoalStatusCompleted, "completed"},
		{GoalStatusArchived, "archived"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if tt.status.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.status.String())
			}
		})
	}
}

// Helper functions for creating pointers to values (for optional updates)
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func statusPtr(s GoalStatus) *GoalStatus {
	return &s
}