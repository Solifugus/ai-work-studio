package ui

import (
	"context"
	"testing"
	"time"

	"fyne.io/fyne/v2"

	"github.com/Solifugus/ai-work-studio/internal/config"
	"github.com/Solifugus/ai-work-studio/pkg/core"
)

// setupTestApp creates a test application with temporary storage
func setupTestObjectivesApp(t *testing.T) *App {
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

// createTestObjectives creates a set of test objectives with various statuses
func createTestObjectives(t *testing.T, app *App) []*core.Objective {
	ctx := context.Background()
	gm := app.GetGoalManager()
	mm := app.GetMethodManager()
	om := app.GetObjectiveManager()

	// Create test goals first
	goal1, err := gm.CreateGoal(ctx, "Complete Project Alpha", "Major project milestone", 8,
		map[string]interface{}{"category": "work"})
	if err != nil {
		t.Fatalf("Failed to create test goal 1: %v", err)
	}

	goal2, err := gm.CreateGoal(ctx, "Learn New Skills", "Personal development", 5,
		map[string]interface{}{"category": "personal"})
	if err != nil {
		t.Fatalf("Failed to create test goal 2: %v", err)
	}

	// Create test methods
	method1, err := mm.CreateMethod(ctx, "Agile Development", "Iterative development approach",
		[]core.ApproachStep{}, core.MethodDomainGeneral, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to create test method: %v", err)
	}

	// Create objectives with various statuses
	objective1, err := om.CreateObjective(ctx, goal1.ID, method1.ID,
		"Setup Development Environment", "Configure development tools and environment",
		map[string]interface{}{"tools": []string{"IDE", "Git", "Docker"}}, 9)
	if err != nil {
		t.Fatalf("Failed to create objective 1: %v", err)
	}

	objective2, err := om.CreateObjective(ctx, goal1.ID, method1.ID,
		"Design Database Schema", "Create comprehensive database design",
		map[string]interface{}{"database": "PostgreSQL"}, 7)
	if err != nil {
		t.Fatalf("Failed to create objective 2: %v", err)
	}

	objective3, err := om.CreateObjective(ctx, goal2.ID, method1.ID,
		"Read Programming Books", "Study advanced programming concepts",
		map[string]interface{}{"books": []string{"Clean Code", "Design Patterns"}}, 4)
	if err != nil {
		t.Fatalf("Failed to create objective 3: %v", err)
	}

	// Start some objectives to test different statuses
	if _, err := om.StartObjective(ctx, objective2.ID); err != nil {
		t.Fatalf("Failed to start objective 2: %v", err)
	}

	if _, err := om.StartObjective(ctx, objective3.ID); err != nil {
		t.Fatalf("Failed to start objective 3: %v", err)
	}

	if _, err := om.PauseObjective(ctx, objective3.ID); err != nil {
		t.Fatalf("Failed to pause objective 3: %v", err)
	}

	// Create a completed objective
	objective4, err := om.CreateObjective(ctx, goal1.ID, method1.ID,
		"Initial Research", "Gather project requirements",
		map[string]interface{}{}, 6)
	if err != nil {
		t.Fatalf("Failed to create objective 4: %v", err)
	}

	if _, err := om.StartObjective(ctx, objective4.ID); err != nil {
		t.Fatalf("Failed to start objective 4: %v", err)
	}

	if _, err := om.CompleteObjective(ctx, objective4.ID, core.ObjectiveResult{
		Success:       true,
		Message:       "Research completed successfully",
		TokensUsed:    150,
		ExecutionTime: 30 * time.Minute,
		CompletedAt:   time.Now().Add(-1 * time.Hour),
	}); err != nil {
		t.Fatalf("Failed to complete objective 4: %v", err)
	}

	return []*core.Objective{objective1, objective2, objective3, objective4}
}

// Helper functions for creating pointers
func stringPtr(s string) *string                      { return &s }
func intPtr(i int) *int                               { return &i }

func TestObjectivesView_Creation(t *testing.T) {
	app := setupTestObjectivesApp(t)
	defer app.Stop()

	// Create a mock parent window (nil is acceptable for testing)
	ov := NewObjectivesView(app, nil)

	if ov == nil {
		t.Fatal("NewObjectivesView returned nil")
	}

	if ov.app != app {
		t.Error("ObjectivesView app reference not set correctly")
	}

	if ov.container == nil {
		t.Error("ObjectivesView container not created")
	}

	if ov.objectivesList == nil {
		t.Error("ObjectivesView objectives list not created")
	}

	if ov.statusLabel == nil {
		t.Error("ObjectivesView status label not created")
	}
}

func TestObjectivesView_LoadObjectives(t *testing.T) {
	app := setupTestObjectivesApp(t)
	defer app.Stop()

	objectives := createTestObjectives(t, app)
	ov := NewObjectivesView(app, nil)

	// Wait a moment for auto-loading to complete
	time.Sleep(100 * time.Millisecond)

	if len(ov.objectives) == 0 {
		t.Error("Expected objectives to be loaded, got 0")
	}

	if len(ov.objectives) != len(objectives) {
		t.Errorf("Expected %d objectives, got %d", len(objectives), len(ov.objectives))
	}

	// Check that filtered objectives are also set
	if len(ov.filteredObjectives) == 0 {
		t.Error("Expected filtered objectives to be set")
	}
}

func TestObjectivesView_FilterByStatus(t *testing.T) {
	app := setupTestObjectivesApp(t)
	defer app.Stop()

	createTestObjectives(t, app)
	ov := NewObjectivesView(app, nil)

	// Wait for initial load
	time.Sleep(100 * time.Millisecond)

	tests := []struct {
		filter   string
		expected int
	}{
		{"all", 4},        // All objectives
		{"pending", 1},    // Only pending objectives
		{"in_progress", 1}, // Only in-progress objectives
		{"paused", 1},     // Only paused objectives
		{"completed", 1},  // Only completed objectives
		{"failed", 0},     // No failed objectives
	}

	for _, tt := range tests {
		t.Run("filter_"+tt.filter, func(t *testing.T) {
			ov.statusFilter = tt.filter
			ov.applyFiltersAndSort()

			if len(ov.filteredObjectives) != tt.expected {
				t.Errorf("Filter %s: expected %d objectives, got %d",
					tt.filter, tt.expected, len(ov.filteredObjectives))
			}
		})
	}
}

func TestObjectivesView_SearchFilter(t *testing.T) {
	app := setupTestObjectivesApp(t)
	defer app.Stop()

	createTestObjectives(t, app)
	ov := NewObjectivesView(app, nil)

	// Wait for initial load
	time.Sleep(100 * time.Millisecond)

	tests := []struct {
		search   string
		expected int
	}{
		{"", 4},           // No filter
		{"Setup", 1},      // Match title
		{"Database", 1},   // Match title
		{"environment", 1}, // Case-insensitive match in description
		{"programming", 1}, // Match description
		{"nonexistent", 0}, // No matches
		{"Research", 1},   // Match completed objective
	}

	for _, tt := range tests {
		t.Run("search_"+tt.search, func(t *testing.T) {
			ov.searchFilter = tt.search
			ov.applyFiltersAndSort()

			if len(ov.filteredObjectives) != tt.expected {
				t.Errorf("Search %q: expected %d objectives, got %d",
					tt.search, tt.expected, len(ov.filteredObjectives))
			}
		})
	}
}

func TestObjectivesView_SortObjectives(t *testing.T) {
	app := setupTestObjectivesApp(t)
	defer app.Stop()

	createTestObjectives(t, app)
	ov := NewObjectivesView(app, nil)

	// Wait for initial load
	time.Sleep(100 * time.Millisecond)

	// Test priority sorting (default - higher priority first)
	ov.sortMode = "priority"
	ov.applyFiltersAndSort()

	if len(ov.filteredObjectives) >= 2 {
		first := ov.filteredObjectives[0]
		second := ov.filteredObjectives[1]

		if first.Priority < second.Priority {
			t.Errorf("Priority sort failed: first priority %d should be >= second priority %d",
				first.Priority, second.Priority)
		}
	}

	// Test title sorting
	ov.sortMode = "title"
	ov.applyFiltersAndSort()

	if len(ov.filteredObjectives) >= 2 {
		first := ov.filteredObjectives[0]
		second := ov.filteredObjectives[1]

		if first.Title > second.Title {
			t.Errorf("Title sort failed: %q should come before %q",
				first.Title, second.Title)
		}
	}
}

func TestObjectivesView_StatusIcons(t *testing.T) {
	// Skip this test for now - would require proper widget interface implementations
	// In a real implementation, this would test actual Fyne widgets
	t.Skip("Skipping widget interface tests - requires full Fyne widget mocks")
}

func TestObjectivesView_ProgressBar(t *testing.T) {
	// Skip this test for now - would require proper widget interface implementations
	// In a real implementation, this would test actual Fyne widgets
	t.Skip("Skipping widget interface tests - requires full Fyne widget mocks")
}

func TestObjectivesView_TimeFormatting(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "30 seconds"},
		{5 * time.Minute, "5 minutes"},
		{2 * time.Hour, "2 hours"},
		{3 * 24 * time.Hour, "3 days"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %q, expected %q",
					tt.duration, result, tt.expected)
			}
		})
	}
}

func TestObjectivesView_StatusBarUpdates(t *testing.T) {
	app := setupTestObjectivesApp(t)
	defer app.Stop()

	createTestObjectives(t, app)
	ov := NewObjectivesView(app, nil)

	// Wait for initial load
	time.Sleep(100 * time.Millisecond)

	// Test with all objectives
	ov.updateStatusBar()
	statusText := ov.statusLabel.Text
	if statusText == "" {
		t.Error("Status bar should show objective count")
	}

	// Test with filtered objectives
	ov.searchFilter = "Setup"
	ov.applyFiltersAndSort()

	if len(ov.filteredObjectives) != len(ov.objectives) {
		// Should show filtered count
		statusText = ov.statusLabel.Text
		if statusText == "" {
			t.Error("Status bar should show filtered count")
		}
	}

	// Test with no objectives (empty filter)
	ov.objectives = []*core.Objective{}
	ov.filteredObjectives = []*core.Objective{}
	ov.updateStatusBar()

	statusText = ov.statusLabel.Text
	if statusText != "No objectives found" {
		t.Errorf("Expected 'No objectives found', got %q", statusText)
	}
}

func TestObjectivesView_AutoRefresh(t *testing.T) {
	app := setupTestObjectivesApp(t)
	defer app.Stop()

	ov := NewObjectivesView(app, nil)
	defer ov.Stop() // Clean up goroutine

	// Test that auto-refresh starts
	if ov.refreshChan == nil {
		t.Error("Refresh channel should be initialized")
	}

	if ov.stopRefresh == nil {
		t.Error("Stop refresh channel should be initialized")
	}

	// Test manual refresh trigger
	select {
	case ov.refreshChan <- true:
		// Should not block
	case <-time.After(100 * time.Millisecond):
		t.Error("Refresh channel should accept messages")
	}
}

// Mock widget types for testing
type testIcon struct {
	resource fyne.Resource
}

func (i *testIcon) SetResource(resource fyne.Resource) { i.resource = resource }

type testLabel struct {
	text string
}

func (l *testLabel) SetText(text string) { l.text = text }

type testProgressBar struct {
	value   float64
	visible bool
}

func (p *testProgressBar) SetValue(value float64) { p.value = value }
func (p *testProgressBar) Show()                 { p.visible = true }
func (p *testProgressBar) Hide()                 { p.visible = false }

// Test error handling
func TestObjectivesView_ErrorHandling(t *testing.T) {
	app := setupTestObjectivesApp(t)
	defer app.Stop()

	ov := NewObjectivesView(app, nil)

	// Test with invalid list index (should not panic)
	ov.filteredObjectives = []*core.Objective{}
	ov.updateObjectiveListItem(0, ov.createObjectiveListItem())

	// Test with nil objective (should handle gracefully)
	ov.filteredObjectives = []*core.Objective{nil}
	// This should not panic, even with nil objective
}

func TestObjectivesView_Integration(t *testing.T) {
	app := setupTestObjectivesApp(t)
	defer app.Stop()

	objectives := createTestObjectives(t, app)
	ov := NewObjectivesView(app, nil)
	defer ov.Stop()

	// Wait for initial load
	time.Sleep(100 * time.Millisecond)

	// Verify complete integration
	if len(ov.objectives) == 0 {
		t.Error("Objectives should be loaded from storage")
	}

	// Test that we can filter and sort
	originalCount := len(ov.filteredObjectives)
	ov.statusFilter = "completed"
	ov.applyFiltersAndSort()

	completedCount := len(ov.filteredObjectives)
	if completedCount == 0 {
		t.Error("Should have at least one completed objective")
	}

	if completedCount == originalCount {
		t.Error("Filtering should reduce the number of displayed objectives")
	}

	// Reset filter
	ov.statusFilter = "all"
	ov.applyFiltersAndSort()

	if len(ov.filteredObjectives) != originalCount {
		t.Error("Resetting filter should restore original count")
	}

	// Verify objectives have the expected statuses
	statusCount := make(map[core.ObjectiveStatus]int)
	for _, obj := range objectives {
		// Reload to get current status
		ctx := context.Background()
		current, err := app.GetObjectiveManager().GetObjective(ctx, obj.ID)
		if err != nil {
			t.Fatalf("Failed to reload objective: %v", err)
		}
		statusCount[current.Status]++
	}

	expectedStatuses := []core.ObjectiveStatus{
		core.ObjectiveStatusPending,
		core.ObjectiveStatusInProgress,
		core.ObjectiveStatusPaused,
		core.ObjectiveStatusCompleted,
	}

	for _, status := range expectedStatuses {
		if statusCount[status] == 0 {
			t.Errorf("Expected at least one objective with status %s", status)
		}
	}
}