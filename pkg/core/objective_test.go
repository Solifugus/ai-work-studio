package core

import (
	"context"
	"testing"
	"time"
)

func TestObjectiveManager_CreateObjective(t *testing.T) {
	store := setupTestStore(t)
	gm := NewGoalManager(store)
	mm := NewMethodManager(store)
	om := NewObjectiveManager(store)
	ctx := context.Background()

	// Create test goal and method first
	goal, err := gm.CreateGoal(ctx, "Test Goal", "A goal for testing", 5, nil)
	if err != nil {
		t.Fatalf("Failed to create test goal: %v", err)
	}

	method, err := mm.CreateMethod(ctx, "Test Method", "A method for testing", []ApproachStep{
		{Description: "Step 1", Tools: []string{"tool1"}},
	}, MethodDomainGeneral, nil)
	if err != nil {
		t.Fatalf("Failed to create test method: %v", err)
	}

	tests := []struct {
		name        string
		goalID      string
		methodID    string
		title       string
		description string
		context     map[string]interface{}
		priority    int
		expectError bool
	}{
		{
			name:        "valid objective",
			goalID:      goal.ID,
			methodID:    method.ID,
			title:       "Complete task",
			description: "Complete a specific task using the test method",
			context:     map[string]interface{}{"input": "test data", "ref": "file://data.json"},
			priority:    7,
			expectError: false,
		},
		{
			name:        "empty title",
			goalID:      goal.ID,
			methodID:    method.ID,
			title:       "",
			description: "Some description",
			priority:    5,
			expectError: true,
		},
		{
			name:        "empty goal ID",
			goalID:      "",
			methodID:    method.ID,
			title:       "Test objective",
			description: "Test description",
			priority:    5,
			expectError: true,
		},
		{
			name:        "empty method ID",
			goalID:      goal.ID,
			methodID:    "",
			title:       "Test objective",
			description: "Test description",
			priority:    5,
			expectError: true,
		},
		{
			name:        "invalid priority too low",
			goalID:      goal.ID,
			methodID:    method.ID,
			title:       "Test objective",
			description: "Test description",
			priority:    0,
			expectError: true,
		},
		{
			name:        "invalid priority too high",
			goalID:      goal.ID,
			methodID:    method.ID,
			title:       "Test objective",
			description: "Test description",
			priority:    11,
			expectError: true,
		},
		{
			name:        "minimal valid objective",
			goalID:      goal.ID,
			methodID:    method.ID,
			title:       "Simple objective",
			description: "",
			context:     nil,
			priority:    1,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objective, err := om.CreateObjective(ctx, tt.goalID, tt.methodID, tt.title, tt.description, tt.context, tt.priority)

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

			// Verify objective properties
			if objective.GoalID != tt.goalID {
				t.Errorf("Expected goalID %q, got %q", tt.goalID, objective.GoalID)
			}
			if objective.MethodID != tt.methodID {
				t.Errorf("Expected methodID %q, got %q", tt.methodID, objective.MethodID)
			}
			if objective.Title != tt.title {
				t.Errorf("Expected title %q, got %q", tt.title, objective.Title)
			}
			if objective.Description != tt.description {
				t.Errorf("Expected description %q, got %q", tt.description, objective.Description)
			}
			if objective.Priority != tt.priority {
				t.Errorf("Expected priority %d, got %d", tt.priority, objective.Priority)
			}
			if objective.Status != ObjectiveStatusPending {
				t.Errorf("Expected status %v, got %v", ObjectiveStatusPending, objective.Status)
			}
			if objective.ID == "" {
				t.Error("Expected non-empty objective ID")
			}
			if objective.CreatedAt.IsZero() {
				t.Error("Expected non-zero CreatedAt time")
			}
			if objective.StartedAt != nil {
				t.Error("Expected nil StartedAt for pending objective")
			}
			if objective.CompletedAt != nil {
				t.Error("Expected nil CompletedAt for pending objective")
			}
			if objective.Result != nil {
				t.Error("Expected nil Result for pending objective")
			}
		})
	}
}

func TestObjectiveManager_GetObjective(t *testing.T) {
	store := setupTestStore(t)
	gm := NewGoalManager(store)
	mm := NewMethodManager(store)
	om := NewObjectiveManager(store)
	ctx := context.Background()

	// Create test goal and method
	goal, _ := gm.CreateGoal(ctx, "Test Goal", "A goal for testing", 5, nil)
	method, _ := mm.CreateMethod(ctx, "Test Method", "A method for testing", []ApproachStep{}, MethodDomainGeneral, nil)

	// Create test objective
	originalObjective, err := om.CreateObjective(ctx, goal.ID, method.ID, "Test Objective", "Test description", map[string]interface{}{"key": "value"}, 8)
	if err != nil {
		t.Fatalf("Failed to create test objective: %v", err)
	}

	// Test getting the objective
	retrievedObjective, err := om.GetObjective(ctx, originalObjective.ID)
	if err != nil {
		t.Fatalf("Failed to get objective: %v", err)
	}

	// Verify properties match
	if retrievedObjective.ID != originalObjective.ID {
		t.Errorf("Expected ID %q, got %q", originalObjective.ID, retrievedObjective.ID)
	}
	if retrievedObjective.Title != originalObjective.Title {
		t.Errorf("Expected title %q, got %q", originalObjective.Title, retrievedObjective.Title)
	}
	if retrievedObjective.GoalID != originalObjective.GoalID {
		t.Errorf("Expected goalID %q, got %q", originalObjective.GoalID, retrievedObjective.GoalID)
	}
	if retrievedObjective.MethodID != originalObjective.MethodID {
		t.Errorf("Expected methodID %q, got %q", originalObjective.MethodID, retrievedObjective.MethodID)
	}

	// Test getting non-existent objective
	_, err = om.GetObjective(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error when getting non-existent objective")
	}
}

func TestObjectiveLifecycle(t *testing.T) {
	store := setupTestStore(t)
	gm := NewGoalManager(store)
	mm := NewMethodManager(store)
	om := NewObjectiveManager(store)
	ctx := context.Background()

	// Create test dependencies
	goal, _ := gm.CreateGoal(ctx, "Test Goal", "A goal for testing", 5, nil)
	method, _ := mm.CreateMethod(ctx, "Test Method", "A method for testing", []ApproachStep{}, MethodDomainGeneral, nil)

	// Create objective
	objective, err := om.CreateObjective(ctx, goal.ID, method.ID, "Lifecycle Test", "Test objective lifecycle", nil, 5)
	if err != nil {
		t.Fatalf("Failed to create objective: %v", err)
	}

	// Test initial state
	if objective.Status != ObjectiveStatusPending {
		t.Errorf("Expected initial status %v, got %v", ObjectiveStatusPending, objective.Status)
	}
	if !objective.IsPending() {
		t.Error("Expected IsPending() to be true")
	}

	// Test starting objective
	startedObjective, err := om.StartObjective(ctx, objective.ID)
	if err != nil {
		t.Fatalf("Failed to start objective: %v", err)
	}
	if startedObjective.Status != ObjectiveStatusInProgress {
		t.Errorf("Expected status after start %v, got %v", ObjectiveStatusInProgress, startedObjective.Status)
	}
	if !startedObjective.IsInProgress() {
		t.Error("Expected IsInProgress() to be true")
	}
	if startedObjective.StartedAt == nil {
		t.Error("Expected StartedAt to be set after starting")
	}

	// Test pausing objective
	pausedObjective, err := om.PauseObjective(ctx, startedObjective.ID)
	if err != nil {
		t.Fatalf("Failed to pause objective: %v", err)
	}
	if pausedObjective.Status != ObjectiveStatusPaused {
		t.Errorf("Expected status after pause %v, got %v", ObjectiveStatusPaused, pausedObjective.Status)
	}
	if !pausedObjective.IsPaused() {
		t.Error("Expected IsPaused() to be true")
	}

	// Test resuming objective
	resumedObjective, err := om.ResumeObjective(ctx, pausedObjective.ID)
	if err != nil {
		t.Fatalf("Failed to resume objective: %v", err)
	}
	if resumedObjective.Status != ObjectiveStatusInProgress {
		t.Errorf("Expected status after resume %v, got %v", ObjectiveStatusInProgress, resumedObjective.Status)
	}

	// Test completing objective successfully
	result := ObjectiveResult{
		Success:    true,
		Message:    "Objective completed successfully",
		Data:       map[string]interface{}{"output": "success_data"},
		TokensUsed: 150,
	}
	completedObjective, err := om.CompleteObjective(ctx, resumedObjective.ID, result)
	if err != nil {
		t.Fatalf("Failed to complete objective: %v", err)
	}
	if completedObjective.Status != ObjectiveStatusCompleted {
		t.Errorf("Expected status after completion %v, got %v", ObjectiveStatusCompleted, completedObjective.Status)
	}
	if !completedObjective.IsCompleted() {
		t.Error("Expected IsCompleted() to be true")
	}
	if !completedObjective.IsFinished() {
		t.Error("Expected IsFinished() to be true")
	}
	if completedObjective.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set after completion")
	}
	if completedObjective.Result == nil {
		t.Error("Expected Result to be set after completion")
	}
	if completedObjective.Result.Success != true {
		t.Error("Expected Result.Success to be true")
	}
	if completedObjective.Result.TokensUsed != 150 {
		t.Errorf("Expected Result.TokensUsed to be 150, got %d", completedObjective.Result.TokensUsed)
	}
}

func TestObjectiveFailure(t *testing.T) {
	store := setupTestStore(t)
	gm := NewGoalManager(store)
	mm := NewMethodManager(store)
	om := NewObjectiveManager(store)
	ctx := context.Background()

	// Create test dependencies
	goal, _ := gm.CreateGoal(ctx, "Test Goal", "A goal for testing", 5, nil)
	method, _ := mm.CreateMethod(ctx, "Test Method", "A method for testing", []ApproachStep{}, MethodDomainGeneral, nil)

	// Create and start objective
	objective, _ := om.CreateObjective(ctx, goal.ID, method.ID, "Failure Test", "Test objective failure", nil, 5)
	startedObjective, _ := om.StartObjective(ctx, objective.ID)

	// Test failing objective
	failedObjective, err := om.FailObjective(ctx, startedObjective.ID, "Something went wrong", 75)
	if err != nil {
		t.Fatalf("Failed to fail objective: %v", err)
	}
	if failedObjective.Status != ObjectiveStatusFailed {
		t.Errorf("Expected status after failure %v, got %v", ObjectiveStatusFailed, failedObjective.Status)
	}
	if !failedObjective.IsFailed() {
		t.Error("Expected IsFailed() to be true")
	}
	if !failedObjective.IsFinished() {
		t.Error("Expected IsFinished() to be true")
	}
	if failedObjective.Result == nil {
		t.Error("Expected Result to be set after failure")
	}
	if failedObjective.Result.Success != false {
		t.Error("Expected Result.Success to be false")
	}
	if failedObjective.Result.Message != "Something went wrong" {
		t.Errorf("Expected error message, got %q", failedObjective.Result.Message)
	}
	if failedObjective.Result.TokensUsed != 75 {
		t.Errorf("Expected tokens used 75, got %d", failedObjective.Result.TokensUsed)
	}
}

func TestObjectiveInvalidTransitions(t *testing.T) {
	store := setupTestStore(t)
	gm := NewGoalManager(store)
	mm := NewMethodManager(store)
	om := NewObjectiveManager(store)
	ctx := context.Background()

	// Create test dependencies
	goal, _ := gm.CreateGoal(ctx, "Test Goal", "A goal for testing", 5, nil)
	method, _ := mm.CreateMethod(ctx, "Test Method", "A method for testing", []ApproachStep{}, MethodDomainGeneral, nil)
	objective, _ := om.CreateObjective(ctx, goal.ID, method.ID, "Transition Test", "Test invalid transitions", nil, 5)

	// Test starting non-pending objective
	_, err := om.StartObjective(ctx, objective.ID)
	if err != nil {
		t.Fatalf("Failed to start objective: %v", err)
	}

	// Try to start already started objective
	_, err = om.StartObjective(ctx, objective.ID)
	if err == nil {
		t.Error("Expected error when starting already started objective")
	}

	// Test pausing non-in-progress objective
	pendingObjective, _ := om.CreateObjective(ctx, goal.ID, method.ID, "Pending Test", "Test pausing pending", nil, 5)
	_, err = om.PauseObjective(ctx, pendingObjective.ID)
	if err == nil {
		t.Error("Expected error when pausing pending objective")
	}

	// Test resuming non-paused objective
	_, err = om.ResumeObjective(ctx, objective.ID)
	if err == nil {
		t.Error("Expected error when resuming non-paused objective")
	}

	// Test completing non-in-progress objective
	result := ObjectiveResult{Success: true, Message: "Test"}
	_, err = om.CompleteObjective(ctx, pendingObjective.ID, result)
	if err == nil {
		t.Error("Expected error when completing non-in-progress objective")
	}
}

func TestObjectiveManager_ListObjectives(t *testing.T) {
	store := setupTestStore(t)
	gm := NewGoalManager(store)
	mm := NewMethodManager(store)
	om := NewObjectiveManager(store)
	ctx := context.Background()

	// Create test dependencies
	goal1, _ := gm.CreateGoal(ctx, "Goal 1", "First goal", 5, nil)
	goal2, _ := gm.CreateGoal(ctx, "Goal 2", "Second goal", 5, nil)
	method, _ := mm.CreateMethod(ctx, "Test Method", "A method for testing", []ApproachStep{}, MethodDomainGeneral, nil)

	// Create test objectives
	obj1, _ := om.CreateObjective(ctx, goal1.ID, method.ID, "Objective 1", "First objective", nil, 3)
	obj2, _ := om.CreateObjective(ctx, goal1.ID, method.ID, "Objective 2", "Second objective", nil, 7)
	obj3, _ := om.CreateObjective(ctx, goal2.ID, method.ID, "Objective 3", "Third objective", nil, 5)

	// Start one objective
	om.StartObjective(ctx, obj2.ID)

	tests := []struct {
		name           string
		filter         ObjectiveFilter
		expectedCount  int
		expectedTitles []string
	}{
		{
			name:          "no filter",
			filter:        ObjectiveFilter{},
			expectedCount: 3,
		},
		{
			name:           "filter by status pending",
			filter:         ObjectiveFilter{Status: &[]ObjectiveStatus{ObjectiveStatusPending}[0]},
			expectedCount:  2,
			expectedTitles: []string{"Objective 1", "Objective 3"},
		},
		{
			name:           "filter by status in progress",
			filter:         ObjectiveFilter{Status: &[]ObjectiveStatus{ObjectiveStatusInProgress}[0]},
			expectedCount:  1,
			expectedTitles: []string{"Objective 2"},
		},
		{
			name:          "filter by goal ID",
			filter:        ObjectiveFilter{GoalID: &goal1.ID},
			expectedCount: 2,
		},
		{
			name:          "filter by method ID",
			filter:        ObjectiveFilter{MethodID: &method.ID},
			expectedCount: 3,
		},
		{
			name:          "filter by min priority",
			filter:        ObjectiveFilter{MinPriority: &[]int{5}[0]},
			expectedCount: 2,
		},
		{
			name:          "filter by max priority",
			filter:        ObjectiveFilter{MaxPriority: &[]int{5}[0]},
			expectedCount: 2,
		},
		{
			name:          "filter by priority range",
			filter:        ObjectiveFilter{MinPriority: &[]int{4}[0], MaxPriority: &[]int{6}[0]},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objectives, err := om.ListObjectives(ctx, tt.filter)
			if err != nil {
				t.Fatalf("Failed to list objectives: %v", err)
			}

			if len(objectives) != tt.expectedCount {
				t.Errorf("Expected %d objectives, got %d", tt.expectedCount, len(objectives))
			}

			if tt.expectedTitles != nil {
				titles := make([]string, len(objectives))
				for i, obj := range objectives {
					titles[i] = obj.Title
				}
				for _, expectedTitle := range tt.expectedTitles {
					found := false
					for _, title := range titles {
						if title == expectedTitle {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected to find objective with title %q", expectedTitle)
					}
				}
			}
		})
	}

	// Clean up by using the objective variables
	_ = obj1
	_ = obj3
}

func TestObjectiveManager_GetObjectivesForGoal(t *testing.T) {
	store := setupTestStore(t)
	gm := NewGoalManager(store)
	mm := NewMethodManager(store)
	om := NewObjectiveManager(store)
	ctx := context.Background()

	// Create test dependencies
	goal1, _ := gm.CreateGoal(ctx, "Goal 1", "First goal", 5, nil)
	goal2, _ := gm.CreateGoal(ctx, "Goal 2", "Second goal", 5, nil)
	method, _ := mm.CreateMethod(ctx, "Test Method", "A method for testing", []ApproachStep{}, MethodDomainGeneral, nil)

	// Create objectives for different goals
	obj1, _ := om.CreateObjective(ctx, goal1.ID, method.ID, "Objective for Goal 1", "First", nil, 5)
	obj2, _ := om.CreateObjective(ctx, goal1.ID, method.ID, "Another for Goal 1", "Second", nil, 5)
	obj3, _ := om.CreateObjective(ctx, goal2.ID, method.ID, "Objective for Goal 2", "Third", nil, 5)

	// Test getting objectives for goal 1
	objectivesForGoal1, err := om.GetObjectivesForGoal(ctx, goal1.ID)
	if err != nil {
		t.Fatalf("Failed to get objectives for goal 1: %v", err)
	}
	if len(objectivesForGoal1) != 2 {
		t.Errorf("Expected 2 objectives for goal 1, got %d", len(objectivesForGoal1))
	}

	// Verify the objectives belong to goal 1
	for _, obj := range objectivesForGoal1 {
		if obj.GoalID != goal1.ID {
			t.Errorf("Expected objective to belong to goal 1, but got goal %s", obj.GoalID)
		}
	}

	// Test getting objectives for goal 2
	objectivesForGoal2, err := om.GetObjectivesForGoal(ctx, goal2.ID)
	if err != nil {
		t.Fatalf("Failed to get objectives for goal 2: %v", err)
	}
	if len(objectivesForGoal2) != 1 {
		t.Errorf("Expected 1 objective for goal 2, got %d", len(objectivesForGoal2))
	}

	// Test getting objectives for non-existent goal
	objectivesForNonExistent, err := om.GetObjectivesForGoal(ctx, "non-existent-goal")
	if err != nil {
		t.Fatalf("Failed to get objectives for non-existent goal: %v", err)
	}
	if len(objectivesForNonExistent) != 0 {
		t.Errorf("Expected 0 objectives for non-existent goal, got %d", len(objectivesForNonExistent))
	}

	// Clean up by using the objective variables
	_ = obj1
	_ = obj2
	_ = obj3
}

func TestObjectiveManager_GetObjectivesUsingMethod(t *testing.T) {
	store := setupTestStore(t)
	gm := NewGoalManager(store)
	mm := NewMethodManager(store)
	om := NewObjectiveManager(store)
	ctx := context.Background()

	// Create test dependencies
	goal, _ := gm.CreateGoal(ctx, "Test Goal", "A goal for testing", 5, nil)
	method1, _ := mm.CreateMethod(ctx, "Method 1", "First method", []ApproachStep{}, MethodDomainGeneral, nil)
	method2, _ := mm.CreateMethod(ctx, "Method 2", "Second method", []ApproachStep{}, MethodDomainGeneral, nil)

	// Create objectives using different methods
	obj1, _ := om.CreateObjective(ctx, goal.ID, method1.ID, "Objective using Method 1", "First", nil, 5)
	obj2, _ := om.CreateObjective(ctx, goal.ID, method1.ID, "Another using Method 1", "Second", nil, 5)
	obj3, _ := om.CreateObjective(ctx, goal.ID, method2.ID, "Objective using Method 2", "Third", nil, 5)

	// Test getting objectives for method 1
	objectivesUsingMethod1, err := om.GetObjectivesUsingMethod(ctx, method1.ID)
	if err != nil {
		t.Fatalf("Failed to get objectives using method 1: %v", err)
	}
	if len(objectivesUsingMethod1) != 2 {
		t.Errorf("Expected 2 objectives using method 1, got %d", len(objectivesUsingMethod1))
	}

	// Verify the objectives use method 1
	for _, obj := range objectivesUsingMethod1 {
		if obj.MethodID != method1.ID {
			t.Errorf("Expected objective to use method 1, but uses method %s", obj.MethodID)
		}
	}

	// Test getting objectives for method 2
	objectivesUsingMethod2, err := om.GetObjectivesUsingMethod(ctx, method2.ID)
	if err != nil {
		t.Fatalf("Failed to get objectives using method 2: %v", err)
	}
	if len(objectivesUsingMethod2) != 1 {
		t.Errorf("Expected 1 objective using method 2, got %d", len(objectivesUsingMethod2))
	}

	// Clean up by using the objective variables
	_ = obj1
	_ = obj2
	_ = obj3
}

func TestObjectiveManager_UpdateObjective(t *testing.T) {
	store := setupTestStore(t)
	gm := NewGoalManager(store)
	mm := NewMethodManager(store)
	om := NewObjectiveManager(store)
	ctx := context.Background()

	// Create test dependencies
	goal1, _ := gm.CreateGoal(ctx, "Goal 1", "First goal", 5, nil)
	goal2, _ := gm.CreateGoal(ctx, "Goal 2", "Second goal", 5, nil)
	method, _ := mm.CreateMethod(ctx, "Test Method", "A method for testing", []ApproachStep{}, MethodDomainGeneral, nil)

	// Create test objective
	objective, _ := om.CreateObjective(ctx, goal1.ID, method.ID, "Original Title", "Original description", map[string]interface{}{"key": "value"}, 5)

	// Test updating various fields
	newTitle := "Updated Title"
	newDescription := "Updated description"
	newPriority := 8
	newGoalID := goal2.ID
	newContext := map[string]interface{}{"new_key": "new_value"}

	updates := ObjectiveUpdates{
		Title:       &newTitle,
		Description: &newDescription,
		Priority:    &newPriority,
		GoalID:      &newGoalID,
		Context:     newContext,
	}

	updatedObjective, err := om.UpdateObjective(ctx, objective.ID, updates)
	if err != nil {
		t.Fatalf("Failed to update objective: %v", err)
	}

	// Verify updates
	if updatedObjective.Title != newTitle {
		t.Errorf("Expected title %q, got %q", newTitle, updatedObjective.Title)
	}
	if updatedObjective.Description != newDescription {
		t.Errorf("Expected description %q, got %q", newDescription, updatedObjective.Description)
	}
	if updatedObjective.Priority != newPriority {
		t.Errorf("Expected priority %d, got %d", newPriority, updatedObjective.Priority)
	}
	if updatedObjective.GoalID != newGoalID {
		t.Errorf("Expected goalID %q, got %q", newGoalID, updatedObjective.GoalID)
	}
	if updatedObjective.Context["new_key"] != "new_value" {
		t.Error("Expected context to be updated")
	}

	// Test invalid updates
	emptyTitle := ""
	invalidUpdates := ObjectiveUpdates{
		Title: &emptyTitle,
	}
	_, err = om.UpdateObjective(ctx, objective.ID, invalidUpdates)
	if err == nil {
		t.Error("Expected error when updating with empty title")
	}

	invalidPriority := 15
	invalidUpdates = ObjectiveUpdates{
		Priority: &invalidPriority,
	}
	_, err = om.UpdateObjective(ctx, objective.ID, invalidUpdates)
	if err == nil {
		t.Error("Expected error when updating with invalid priority")
	}
}

func TestObjectiveStatusMethods(t *testing.T) {
	// Test each status
	tests := []struct {
		status   ObjectiveStatus
		isPending bool
		isInProgress bool
		isCompleted bool
		isFailed bool
		isPaused bool
		isFinished bool
	}{
		{ObjectiveStatusPending, true, false, false, false, false, false},
		{ObjectiveStatusInProgress, false, true, false, false, false, false},
		{ObjectiveStatusCompleted, false, false, true, false, false, true},
		{ObjectiveStatusFailed, false, false, false, true, false, true},
		{ObjectiveStatusPaused, false, false, false, false, true, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			objective := &Objective{Status: tt.status}

			if objective.IsPending() != tt.isPending {
				t.Errorf("Expected IsPending() %v, got %v", tt.isPending, objective.IsPending())
			}
			if objective.IsInProgress() != tt.isInProgress {
				t.Errorf("Expected IsInProgress() %v, got %v", tt.isInProgress, objective.IsInProgress())
			}
			if objective.IsCompleted() != tt.isCompleted {
				t.Errorf("Expected IsCompleted() %v, got %v", tt.isCompleted, objective.IsCompleted())
			}
			if objective.IsFailed() != tt.isFailed {
				t.Errorf("Expected IsFailed() %v, got %v", tt.isFailed, objective.IsFailed())
			}
			if objective.IsPaused() != tt.isPaused {
				t.Errorf("Expected IsPaused() %v, got %v", tt.isPaused, objective.IsPaused())
			}
			if objective.IsFinished() != tt.isFinished {
				t.Errorf("Expected IsFinished() %v, got %v", tt.isFinished, objective.IsFinished())
			}
		})
	}
}

func TestObjectiveInstanceMethods(t *testing.T) {
	store := setupTestStore(t)
	gm := NewGoalManager(store)
	mm := NewMethodManager(store)
	om := NewObjectiveManager(store)
	ctx := context.Background()

	// Create test dependencies
	goal, _ := gm.CreateGoal(ctx, "Test Goal", "A goal for testing", 5, nil)
	method, _ := mm.CreateMethod(ctx, "Test Method", "A method for testing", []ApproachStep{}, MethodDomainGeneral, nil)

	// Create test objective
	objective, _ := om.CreateObjective(ctx, goal.ID, method.ID, "Instance Test", "Test instance methods", nil, 5)

	// Test Start method
	err := objective.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start objective via instance method: %v", err)
	}
	if !objective.IsInProgress() {
		t.Error("Expected objective to be in progress after Start()")
	}

	// Test Pause method
	err = objective.Pause(ctx)
	if err != nil {
		t.Fatalf("Failed to pause objective via instance method: %v", err)
	}
	if !objective.IsPaused() {
		t.Error("Expected objective to be paused after Pause()")
	}

	// Test Resume method
	err = objective.Resume(ctx)
	if err != nil {
		t.Fatalf("Failed to resume objective via instance method: %v", err)
	}
	if !objective.IsInProgress() {
		t.Error("Expected objective to be in progress after Resume()")
	}

	// Test Complete method
	result := ObjectiveResult{
		Success:    true,
		Message:    "Success via instance method",
		TokensUsed: 100,
	}
	err = objective.Complete(ctx, result)
	if err != nil {
		t.Fatalf("Failed to complete objective via instance method: %v", err)
	}
	if !objective.IsCompleted() {
		t.Error("Expected objective to be completed after Complete()")
	}
	if objective.Result.Message != result.Message {
		t.Error("Expected result message to be set")
	}

	// Test Fail method (create a new objective for this)
	failObjective, _ := om.CreateObjective(ctx, goal.ID, method.ID, "Fail Test", "Test fail method", nil, 5)
	failObjective.Start(ctx)

	err = failObjective.Fail(ctx, "Test failure", 50)
	if err != nil {
		t.Fatalf("Failed to fail objective via instance method: %v", err)
	}
	if !failObjective.IsFailed() {
		t.Error("Expected objective to be failed after Fail()")
	}
	if failObjective.Result.Success != false {
		t.Error("Expected result success to be false")
	}
}

func TestObjectiveTemporalQueries(t *testing.T) {
	store := setupTestStore(t)
	gm := NewGoalManager(store)
	mm := NewMethodManager(store)
	om := NewObjectiveManager(store)
	ctx := context.Background()

	// Create test dependencies
	goal, _ := gm.CreateGoal(ctx, "Test Goal", "A goal for testing", 5, nil)
	method, _ := mm.CreateMethod(ctx, "Test Method", "A method for testing", []ApproachStep{}, MethodDomainGeneral, nil)

	// Create objective
	originalTitle := "Original Title"
	objective, _ := om.CreateObjective(ctx, goal.ID, method.ID, originalTitle, "Original description", nil, 5)

	// Record time after creation
	creationTime := time.Now()
	time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps

	// Update the objective
	newTitle := "Updated Title"
	updates := ObjectiveUpdates{Title: &newTitle}
	_, err := om.UpdateObjective(ctx, objective.ID, updates)
	if err != nil {
		t.Fatalf("Failed to update objective: %v", err)
	}

	// Test getting objective at creation time
	objectiveAtCreation, err := om.GetObjectiveAtTime(ctx, objective.ID, creationTime)
	if err != nil {
		t.Fatalf("Failed to get objective at creation time: %v", err)
	}
	if objectiveAtCreation.Title != originalTitle {
		t.Errorf("Expected title at creation time %q, got %q", originalTitle, objectiveAtCreation.Title)
	}

	// Test getting current objective (should have updated title)
	currentObjective, err := om.GetObjective(ctx, objective.ID)
	if err != nil {
		t.Fatalf("Failed to get current objective: %v", err)
	}
	if currentObjective.Title != newTitle {
		t.Errorf("Expected current title %q, got %q", newTitle, currentObjective.Title)
	}
}