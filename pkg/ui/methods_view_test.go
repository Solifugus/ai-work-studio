package ui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/test"

	"github.com/yourusername/ai-work-studio/internal/config"
	"github.com/yourusername/ai-work-studio/pkg/core"
)

// createTestApp creates a test application instance for testing
func createTestApp() (*App, error) {
	cfg := &config.Config{
		DataDir: "/tmp/test-ai-work-studio",
	}

	return NewApp(cfg, "/tmp/test-config.json")
}

// TestNewMethodsView tests the creation of a new methods view
func TestNewMethodsView(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	methodsView := NewMethodsView(app, testWindow)

	if methodsView == nil {
		t.Fatal("NewMethodsView returned nil")
	}

	if methodsView.app != app {
		t.Error("MethodsView app field not set correctly")
	}

	if methodsView.window != testWindow {
		t.Error("MethodsView window field not set correctly")
	}

	// Verify UI components are created
	if methodsView.searchEntry == nil {
		t.Error("Search entry not created")
	}

	if methodsView.domainSelect == nil {
		t.Error("Domain select not created")
	}

	if methodsView.statusSelect == nil {
		t.Error("Status select not created")
	}

	if methodsView.methodsList == nil {
		t.Error("Methods list not created")
	}

	if methodsView.detailsView == nil {
		t.Error("Details view not created")
	}

	if methodsView.container == nil {
		t.Error("Container not created")
	}
}

// TestMethodsViewGetContainer tests the GetContainer method
func TestMethodsViewGetContainer(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	methodsView := NewMethodsView(app, testWindow)
	container := methodsView.GetContainer()

	if container == nil {
		t.Fatal("GetContainer returned nil")
	}

	if container != methodsView.container {
		t.Error("GetContainer returned wrong container")
	}
}

// TestMethodsViewDomainSelectToEnum tests domain selection conversion
func TestMethodsViewDomainSelectToEnum(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	methodsView := NewMethodsView(app, testWindow)

	tests := []struct {
		selection string
		expected  core.MethodDomain
	}{
		{"General", core.MethodDomainGeneral},
		{"Domain Specific", core.MethodDomainSpecific},
		{"User Specific", core.MethodDomainUser},
		{"Invalid", core.MethodDomainGeneral}, // Should default to General
	}

	for _, test := range tests {
		result := methodsView.domainSelectToEnum(test.selection)
		if result != test.expected {
			t.Errorf("domainSelectToEnum(%s) = %v, want %v", test.selection, result, test.expected)
		}
	}
}

// TestMethodsViewStatusSelectToEnum tests status selection conversion
func TestMethodsViewStatusSelectToEnum(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	methodsView := NewMethodsView(app, testWindow)

	tests := []struct {
		selection string
		expected  core.MethodStatus
	}{
		{"Active", core.MethodStatusActive},
		{"Deprecated", core.MethodStatusDeprecated},
		{"Superseded", core.MethodStatusSuperseded},
		{"Invalid", core.MethodStatusActive}, // Should default to Active
	}

	for _, test := range tests {
		result := methodsView.statusSelectToEnum(test.selection)
		if result != test.expected {
			t.Errorf("statusSelectToEnum(%s) = %v, want %v", test.selection, result, test.expected)
		}
	}
}

// TestMethodsViewFormatLastUsed tests time formatting
func TestMethodsViewFormatLastUsed(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	methodsView := NewMethodsView(app, testWindow)

	// Test zero time
	result := methodsView.formatLastUsed(time.Time{})
	expected := "Never"
	if result != expected {
		t.Errorf("formatLastUsed(zero time) = %s, want %s", result, expected)
	}

	// Test valid time
	testTime := time.Date(2023, 12, 25, 14, 30, 0, 0, time.UTC)
	result = methodsView.formatLastUsed(testTime)
	expected = "Dec 25, 2023 14:30"
	if result != expected {
		t.Errorf("formatLastUsed(%v) = %s, want %s", testTime, result, expected)
	}
}

// TestMethodsViewGetTimeSinceLastUsed tests time since calculation
func TestMethodsViewGetTimeSinceLastUsed(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	methodsView := NewMethodsView(app, testWindow)

	// Test zero time
	result := methodsView.getTimeSinceLastUsed(time.Time{})
	expected := "Never used"
	if result != expected {
		t.Errorf("getTimeSinceLastUsed(zero time) = %s, want %s", result, expected)
	}

	// Test recent time (hours ago)
	recentTime := time.Now().Add(-2 * time.Hour)
	result = methodsView.getTimeSinceLastUsed(recentTime)
	if result == "" || result == "Never used" {
		t.Errorf("getTimeSinceLastUsed(2 hours ago) should not be empty or 'Never used', got: %s", result)
	}
}

// TestMethodsViewCreateUI tests UI component creation
func TestMethodsViewCreateUI(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	methodsView := NewMethodsView(app, testWindow)

	// Test that all UI components are properly initialized
	if methodsView.searchEntry.PlaceHolder != "Search methods by name or description..." {
		t.Error("Search entry placeholder not set correctly")
	}

	// Test domain select options
	domainOptions := methodsView.domainSelect.Options
	expectedDomains := []string{"All Domains", "General", "Domain Specific", "User Specific"}
	if len(domainOptions) != len(expectedDomains) {
		t.Errorf("Domain select options count = %d, want %d", len(domainOptions), len(expectedDomains))
	}

	for i, expected := range expectedDomains {
		if i >= len(domainOptions) || domainOptions[i] != expected {
			t.Errorf("Domain option %d = %s, want %s", i, domainOptions[i], expected)
		}
	}

	// Test status select options
	statusOptions := methodsView.statusSelect.Options
	expectedStatuses := []string{"All Status", "Active", "Deprecated", "Superseded"}
	if len(statusOptions) != len(expectedStatuses) {
		t.Errorf("Status select options count = %d, want %d", len(statusOptions), len(expectedStatuses))
	}

	for i, expected := range expectedStatuses {
		if i >= len(statusOptions) || statusOptions[i] != expected {
			t.Errorf("Status option %d = %s, want %s", i, statusOptions[i], expected)
		}
	}

	// Test details view tabs
	if len(methodsView.detailsView.Items) != 4 {
		t.Errorf("Details view tabs count = %d, want 4", len(methodsView.detailsView.Items))
	}

	expectedTabs := []string{"Overview", "Approach", "History", "Metrics"}
	for i, expected := range expectedTabs {
		if i >= len(methodsView.detailsView.Items) {
			break
		}
		if methodsView.detailsView.Items[i].Text != expected {
			t.Errorf("Tab %d text = %s, want %s", i, methodsView.detailsView.Items[i].Text, expected)
		}
	}
}

// TestMethodsViewRefresh tests the refresh functionality
func TestMethodsViewRefresh(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	methodsView := NewMethodsView(app, testWindow)

	// Test that refresh doesn't panic
	// Note: This will likely result in an error due to storage not being set up,
	// but it shouldn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Refresh() panicked: %v", r)
		}
	}()

	methodsView.Refresh()
}

// TestMethodsViewMethodsListLength tests the list length calculation
func TestMethodsViewMethodsListLength(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	methodsView := NewMethodsView(app, testWindow)

	// Initially should be 0
	length := methodsView.methodsListLength()
	if length != 0 {
		t.Errorf("Initial methods list length = %d, want 0", length)
	}

	// Add some test filtered methods
	methodsView.filteredMethods = []*core.Method{
		{Name: "Test Method 1"},
		{Name: "Test Method 2"},
	}

	length = methodsView.methodsListLength()
	if length != 2 {
		t.Errorf("Methods list length after adding items = %d, want 2", length)
	}
}

// TestMethodsViewSearchFunctionality tests search functionality
func TestMethodsViewSearchFunctionality(t *testing.T) {
	app, err := createTestApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}
	defer app.Stop()

	testApp := test.NewApp()
	testWindow := testApp.NewWindow("Test")

	methodsView := NewMethodsView(app, testWindow)

	// Mock some methods
	methodsView.methods = []*core.Method{
		{
			Name:        "Data Processing Method",
			Description: "Processes incoming data streams",
			Domain:      core.MethodDomainGeneral,
			Status:      core.MethodStatusActive,
		},
		{
			Name:        "Email Handler",
			Description: "Handles email notifications",
			Domain:      core.MethodDomainSpecific,
			Status:      core.MethodStatusActive,
		},
		{
			Name:        "Archive Tool",
			Description: "Archives old data",
			Domain:      core.MethodDomainUser,
			Status:      core.MethodStatusDeprecated,
		},
	}

	// Test search functionality by triggering filter application
	methodsView.applyFilters()

	// Initially all methods should be visible
	if len(methodsView.filteredMethods) != 3 {
		t.Errorf("Initial filtered methods count = %d, want 3", len(methodsView.filteredMethods))
	}

	// Simulate search for "data"
	methodsView.searchEntry.SetText("data")
	methodsView.applyFilters()

	// Should match "Data Processing Method" and "Archive Tool" (archives old data)
	if len(methodsView.filteredMethods) != 2 {
		t.Errorf("Filtered methods count after search = %d, want 2", len(methodsView.filteredMethods))
	}
}