package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/yourusername/ai-work-studio/pkg/core"
)

// TestNewStatusView tests the creation of a new status view
func TestNewStatusView(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	statusView := NewStatusView(app, testWindow)

	if statusView == nil {
		t.Fatal("NewStatusView returned nil")
	}

	if statusView.app != app {
		t.Error("StatusView app field not set correctly")
	}

	if statusView.window != testWindow {
		t.Error("StatusView window field not set correctly")
	}

	// Verify UI components are created
	if statusView.refreshBtn == nil {
		t.Error("Refresh button not created")
	}

	if statusView.autoRefreshCheck == nil {
		t.Error("Auto-refresh checkbox not created")
	}

	if statusView.container == nil {
		t.Error("Container not created")
	}

	// Verify dashboard cards are created
	if statusView.systemHealthCard == nil {
		t.Error("System health card not created")
	}

	if statusView.activityCard == nil {
		t.Error("Activity card not created")
	}

	if statusView.budgetCard == nil {
		t.Error("Budget card not created")
	}

	if statusView.dataStatsCard == nil {
		t.Error("Data stats card not created")
	}

	if statusView.quickActionsCard == nil {
		t.Error("Quick actions card not created")
	}

	if statusView.recentEventsCard == nil {
		t.Error("Recent events card not created")
	}

	// Clean up auto-refresh
	statusView.Cleanup()
}

// TestStatusViewGetContainer tests the GetContainer method
func TestStatusViewGetContainer(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	statusView := NewStatusView(app, testWindow)
	defer statusView.Cleanup()

	container := statusView.GetContainer()

	if container == nil {
		t.Fatal("GetContainer returned nil")
	}

	if container != statusView.container {
		t.Error("GetContainer returned wrong container")
	}
}

// TestStatusViewRefresh tests the refresh functionality
func TestStatusViewRefresh(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	statusView := NewStatusView(app, testWindow)
	defer statusView.Cleanup()

	// Test that refresh doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Refresh() panicked: %v", r)
		}
	}()

	statusView.Refresh()
}

// TestStatusViewCheckDirectoryExists tests directory existence checking
func TestStatusViewCheckDirectoryExists(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	statusView := NewStatusView(app, testWindow)
	defer statusView.Cleanup()

	// Test with existing directory (temp directory should exist)
	tempDir := os.TempDir()
	exists := statusView.checkDirectoryExists(tempDir)
	if !exists {
		t.Errorf("checkDirectoryExists(%s) = false, want true", tempDir)
	}

	// Test with non-existing directory
	nonExistentDir := "/this/path/should/not/exist/12345"
	exists = statusView.checkDirectoryExists(nonExistentDir)
	if exists {
		t.Errorf("checkDirectoryExists(%s) = true, want false", nonExistentDir)
	}
}

// TestStatusViewCheckStorageHealth tests storage health checking
func TestStatusViewCheckStorageHealth(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	statusView := NewStatusView(app, testWindow)
	defer statusView.Cleanup()

	// Test storage health check
	healthy := statusView.checkStorageHealth()
	// Should return true since we have a valid app context
	if !healthy {
		t.Error("checkStorageHealth() = false, want true for valid app")
	}
}

// TestStatusViewCalculateGoalCompletionRate tests goal completion calculation
func TestStatusViewCalculateGoalCompletionRate(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	statusView := NewStatusView(app, testWindow)
	defer statusView.Cleanup()

	// Test with empty goals list
	rate := statusView.calculateGoalCompletionRate([]*core.Goal{})
	if rate != 0 {
		t.Errorf("calculateGoalCompletionRate([]) = %f, want 0", rate)
	}

	// Test with mixed goals
	goals := []*core.Goal{
		{Status: core.GoalStatusActive},
		{Status: core.GoalStatusCompleted},
		{Status: core.GoalStatusCompleted},
		{Status: core.GoalStatusPaused},
	}

	rate = statusView.calculateGoalCompletionRate(goals)
	expected := 50.0 // 2 completed out of 4 = 50%
	if rate != expected {
		t.Errorf("calculateGoalCompletionRate(mixed goals) = %f, want %f", rate, expected)
	}

	// Test with all completed goals
	allCompleted := []*core.Goal{
		{Status: core.GoalStatusCompleted},
		{Status: core.GoalStatusCompleted},
	}

	rate = statusView.calculateGoalCompletionRate(allCompleted)
	expected = 100.0
	if rate != expected {
		t.Errorf("calculateGoalCompletionRate(all completed) = %f, want %f", rate, expected)
	}
}

// TestStatusViewFormatBytes tests byte formatting
func TestStatusViewFormatBytes(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	statusView := NewStatusView(app, testWindow)
	defer statusView.Cleanup()

	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{1536 * 1024, "1.5 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
	}

	for _, test := range tests {
		result := statusView.formatBytes(test.bytes)
		if result != test.expected {
			t.Errorf("formatBytes(%d) = %s, want %s", test.bytes, result, test.expected)
		}
	}
}

// TestStatusViewCalculateDirectoryStats tests directory statistics calculation
func TestStatusViewCalculateDirectoryStats(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	statusView := NewStatusView(app, testWindow)
	defer statusView.Cleanup()

	// Create a temporary directory with some test files
	tempDir, err := os.MkdirTemp("", "status-view-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFile1 := filepath.Join(tempDir, "test1.txt")
	testFile2 := filepath.Join(tempDir, "test2.txt")

	content1 := "Hello, World!"
	content2 := "This is a test file with more content."

	err = os.WriteFile(testFile1, []byte(content1), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file 1: %v", err)
	}

	err = os.WriteFile(testFile2, []byte(content2), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file 2: %v", err)
	}

	// Calculate directory stats
	size, fileCount, err := statusView.calculateDirectoryStats(tempDir)
	if err != nil {
		t.Fatalf("calculateDirectoryStats failed: %v", err)
	}

	expectedFileCount := 2
	if fileCount != expectedFileCount {
		t.Errorf("calculateDirectoryStats file count = %d, want %d", fileCount, expectedFileCount)
	}

	expectedSize := int64(len(content1) + len(content2))
	if size != expectedSize {
		t.Errorf("calculateDirectoryStats size = %d, want %d", size, expectedSize)
	}
}

// TestStatusViewAutoRefresh tests auto-refresh functionality
func TestStatusViewAutoRefresh(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	statusView := NewStatusView(app, testWindow)
	defer statusView.Cleanup()

	// Test that auto-refresh is off by default
	if statusView.refreshTimer != nil {
		t.Error("Auto-refresh timer should not be started by default")
	}

	// Test starting auto-refresh
	statusView.onAutoRefreshToggle(true)

	if statusView.refreshTimer == nil {
		t.Error("Auto-refresh timer should be started after toggle")
	}

	// Test stopping auto-refresh
	statusView.onAutoRefreshToggle(false)

	// Give it a moment to stop
	time.Sleep(10 * time.Millisecond)

	// Timer should be stopped but may still exist (will be cleaned up)
}

// TestStatusViewCleanup tests cleanup functionality
func TestStatusViewCleanup(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	statusView := NewStatusView(app, testWindow)

	// Start auto-refresh to test cleanup
	statusView.onAutoRefreshToggle(true)

	// Verify timer is running
	if statusView.refreshTimer == nil {
		t.Error("Refresh timer should be running after starting auto-refresh")
	}

	// Call cleanup
	statusView.Cleanup()

	// Verify timer is stopped
	if statusView.refreshTimer != nil {
		t.Error("Refresh timer should be stopped after cleanup")
	}
}

// TestStatusViewCreateDashboardCards tests dashboard card creation
func TestStatusViewCreateDashboardCards(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	statusView := NewStatusView(app, testWindow)
	defer statusView.Cleanup()

	// Test that all cards have titles
	expectedCardTitles := map[string]bool{
		"System Health":   false,
		"Current Activity": false,
		"Budget Usage":    false,
		"Data Statistics": false,
		"Quick Actions":   false,
		"Recent Activity": false,
	}

	cards := []*widget.Card{
		statusView.systemHealthCard,
		statusView.activityCard,
		statusView.budgetCard,
		statusView.dataStatsCard,
		statusView.quickActionsCard,
		statusView.recentEventsCard,
	}

	for _, card := range cards {
		if card == nil {
			t.Error("Found nil card")
			continue
		}

		if _, exists := expectedCardTitles[card.Title]; exists {
			expectedCardTitles[card.Title] = true
		} else {
			t.Errorf("Unexpected card title: %s", card.Title)
		}
	}

	// Verify all expected cards were found
	for title, found := range expectedCardTitles {
		if !found {
			t.Errorf("Expected card not found: %s", title)
		}
	}
}

// TestStatusViewGetUptimeString tests uptime string generation
func TestStatusViewGetUptimeString(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	statusView := NewStatusView(app, testWindow)
	defer statusView.Cleanup()

	uptime := statusView.getUptimeString()
	if uptime == "" {
		t.Error("getUptimeString() returned empty string")
	}

	// For now, just verify it returns our placeholder
	expected := "Session active"
	if uptime != expected {
		t.Errorf("getUptimeString() = %s, want %s", uptime, expected)
	}
}

// TestStatusViewGetBackupInfo tests backup info retrieval
func TestStatusViewGetBackupInfo(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	statusView := NewStatusView(app, testWindow)
	defer statusView.Cleanup()

	backupInfo := statusView.getBackupInfo()
	if backupInfo == "" {
		t.Error("getBackupInfo() returned empty string")
	}

	// For now, just verify it returns our placeholder
	expected := "No backup system configured"
	if backupInfo != expected {
		t.Errorf("getBackupInfo() = %s, want %s", backupInfo, expected)
	}
}