package core

import (
	"context"
	"testing"
	"time"
)

func TestMethodManager_CreateMethod(t *testing.T) {
	store := setupTestStore(t)
	mm := NewMethodManager(store)
	ctx := context.Background()

	tests := []struct {
		name        string
		methodName  string
		description string
		approach    []ApproachStep
		domain      MethodDomain
		userContext map[string]interface{}
		expectError bool
	}{
		{
			name:        "valid method",
			methodName:  "File Processing",
			description: "Process files in a directory",
			approach: []ApproachStep{
				{
					Description: "Read directory",
					Tools:       []string{"file_reader", "directory_scanner"},
					Heuristics:  []string{"check permissions first"},
					Conditions:  map[string]interface{}{"min_files": 1},
				},
				{
					Description: "Process each file",
					Tools:       []string{"file_processor"},
				},
			},
			domain:      MethodDomainGeneral,
			userContext: map[string]interface{}{"category": "file_management"},
			expectError: false,
		},
		{
			name:        "empty name",
			methodName:  "",
			description: "Some description",
			approach:    []ApproachStep{},
			domain:      MethodDomainGeneral,
			expectError: true,
		},
		{
			name:        "invalid domain",
			methodName:  "Test method",
			description: "Test description",
			approach:    []ApproachStep{},
			domain:      MethodDomain("invalid_domain"),
			expectError: true,
		},
		{
			name:        "minimal valid method",
			methodName:  "Simple method",
			description: "",
			approach:    []ApproachStep{},
			domain:      MethodDomainUser,
			userContext: nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, err := mm.CreateMethod(ctx, tt.methodName, tt.description, tt.approach, tt.domain, tt.userContext)

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

			// Verify method properties
			if method.Name != tt.methodName {
				t.Errorf("Expected name %q, got %q", tt.methodName, method.Name)
			}
			if method.Description != tt.description {
				t.Errorf("Expected description %q, got %q", tt.description, method.Description)
			}
			if method.Domain != tt.domain {
				t.Errorf("Expected domain %v, got %v", tt.domain, method.Domain)
			}
			if method.Status != MethodStatusActive {
				t.Errorf("Expected status %v, got %v", MethodStatusActive, method.Status)
			}
			if method.Version != "1.0.0" {
				t.Errorf("Expected version 1.0.0, got %s", method.Version)
			}

			// Verify approach steps
			if len(method.Approach) != len(tt.approach) {
				t.Errorf("Expected %d approach steps, got %d", len(tt.approach), len(method.Approach))
			}

			// Verify initial metrics
			if method.Metrics.ExecutionCount != 0 {
				t.Errorf("Expected execution count 0, got %d", method.Metrics.ExecutionCount)
			}
			if method.Metrics.SuccessCount != 0 {
				t.Errorf("Expected success count 0, got %d", method.Metrics.SuccessCount)
			}
			if !method.Metrics.LastUsed.IsZero() {
				t.Errorf("Expected zero LastUsed time, got %v", method.Metrics.LastUsed)
			}
		})
	}
}

func TestMethodManager_GetMethod(t *testing.T) {
	store := setupTestStore(t)
	mm := NewMethodManager(store)
	ctx := context.Background()

	// Create a test method
	approach := []ApproachStep{
		{Description: "Step 1", Tools: []string{"tool1"}},
	}
	method, err := mm.CreateMethod(ctx, "Test Method", "Test description", approach, MethodDomainGeneral, nil)
	if err != nil {
		t.Fatalf("Failed to create test method: %v", err)
	}

	// Test getting the method
	retrieved, err := mm.GetMethod(ctx, method.ID)
	if err != nil {
		t.Errorf("Failed to get method: %v", err)
		return
	}

	// Verify retrieved method
	if retrieved.ID != method.ID {
		t.Errorf("Expected ID %s, got %s", method.ID, retrieved.ID)
	}
	if retrieved.Name != method.Name {
		t.Errorf("Expected name %q, got %q", method.Name, retrieved.Name)
	}

	// Test getting non-existent method
	_, err = mm.GetMethod(ctx, "non-existent-id")
	if err == nil {
		t.Errorf("Expected error when getting non-existent method")
	}
}

func TestMethodManager_UpdateMethod(t *testing.T) {
	store := setupTestStore(t)
	mm := NewMethodManager(store)
	ctx := context.Background()

	// Create a test method
	approach := []ApproachStep{
		{Description: "Original step", Tools: []string{"tool1"}},
	}
	method, err := mm.CreateMethod(ctx, "Original Method", "Original description", approach, MethodDomainGeneral, nil)
	if err != nil {
		t.Fatalf("Failed to create test method: %v", err)
	}

	// Test updates
	newName := "Updated Method"
	newDescription := "Updated description"
	newStatus := MethodStatusDeprecated
	newApproach := []ApproachStep{
		{Description: "Updated step", Tools: []string{"tool1", "tool2"}},
		{Description: "New step", Heuristics: []string{"new heuristic"}},
	}

	updates := MethodUpdates{
		Name:        &newName,
		Description: &newDescription,
		Status:      &newStatus,
		Approach:    newApproach,
	}

	updated, err := mm.UpdateMethod(ctx, method.ID, updates)
	if err != nil {
		t.Errorf("Failed to update method: %v", err)
		return
	}

	// Verify updates
	if updated.Name != newName {
		t.Errorf("Expected updated name %q, got %q", newName, updated.Name)
	}
	if updated.Description != newDescription {
		t.Errorf("Expected updated description %q, got %q", newDescription, updated.Description)
	}
	if updated.Status != newStatus {
		t.Errorf("Expected updated status %v, got %v", newStatus, updated.Status)
	}
	if len(updated.Approach) != len(newApproach) {
		t.Errorf("Expected %d approach steps, got %d", len(newApproach), len(updated.Approach))
	}

	// Test invalid updates
	invalidName := ""
	invalidUpdates := MethodUpdates{
		Name: &invalidName,
	}
	_, err = mm.UpdateMethod(ctx, method.ID, invalidUpdates)
	if err == nil {
		t.Errorf("Expected error when updating with empty name")
	}
}

func TestMethodManager_UpdateMethodMetrics(t *testing.T) {
	store := setupTestStore(t)
	mm := NewMethodManager(store)
	ctx := context.Background()

	// Create a test method
	method, err := mm.CreateMethod(ctx, "Test Method", "Test description", []ApproachStep{}, MethodDomainGeneral, nil)
	if err != nil {
		t.Fatalf("Failed to create test method: %v", err)
	}

	// Record successful execution
	err = mm.UpdateMethodMetrics(ctx, method.ID, true, 8.5)
	if err != nil {
		t.Errorf("Failed to update metrics: %v", err)
		return
	}

	// Get updated method
	updated, err := mm.GetMethod(ctx, method.ID)
	if err != nil {
		t.Fatalf("Failed to get updated method: %v", err)
	}

	// Verify metrics
	if updated.Metrics.ExecutionCount != 1 {
		t.Errorf("Expected execution count 1, got %d", updated.Metrics.ExecutionCount)
	}
	if updated.Metrics.SuccessCount != 1 {
		t.Errorf("Expected success count 1, got %d", updated.Metrics.SuccessCount)
	}
	if updated.Metrics.SuccessRate() != 100.0 {
		t.Errorf("Expected success rate 100.0, got %f", updated.Metrics.SuccessRate())
	}
	if updated.Metrics.AverageRating != 8.5 {
		t.Errorf("Expected average rating 8.5, got %f", updated.Metrics.AverageRating)
	}
	if updated.Metrics.LastUsed.IsZero() {
		t.Errorf("Expected LastUsed to be set, but it was zero")
	}

	// Record failed execution
	err = mm.UpdateMethodMetrics(ctx, method.ID, false, 6.0)
	if err != nil {
		t.Errorf("Failed to update metrics for failed execution: %v", err)
		return
	}

	// Get updated method again
	updated2, err := mm.GetMethod(ctx, method.ID)
	if err != nil {
		t.Fatalf("Failed to get updated method: %v", err)
	}

	// Verify updated metrics
	if updated2.Metrics.ExecutionCount != 2 {
		t.Errorf("Expected execution count 2, got %d", updated2.Metrics.ExecutionCount)
	}
	if updated2.Metrics.SuccessCount != 1 {
		t.Errorf("Expected success count 1, got %d", updated2.Metrics.SuccessCount)
	}
	if updated2.Metrics.SuccessRate() != 50.0 {
		t.Errorf("Expected success rate 50.0, got %f", updated2.Metrics.SuccessRate())
	}

	// Verify incremental average calculation (8.5 + 6.0) / 2 = 7.25
	expectedAvg := 7.25
	if updated2.Metrics.AverageRating != expectedAvg {
		t.Errorf("Expected average rating %f, got %f", expectedAvg, updated2.Metrics.AverageRating)
	}
}

func TestMethodManager_ListMethods(t *testing.T) {
	store := setupTestStore(t)
	mm := NewMethodManager(store)
	ctx := context.Background()

	// Create test methods
	method1, err := mm.CreateMethod(ctx, "Method 1", "Description 1", []ApproachStep{}, MethodDomainGeneral, nil)
	if err != nil {
		t.Fatalf("Failed to create method 1: %v", err)
	}

	method2, err := mm.CreateMethod(ctx, "Method 2", "Description 2", []ApproachStep{}, MethodDomainUser, nil)
	if err != nil {
		t.Fatalf("Failed to create method 2: %v", err)
	}

	// Update method2 to deprecated
	deprecatedStatus := MethodStatusDeprecated
	updates := MethodUpdates{Status: &deprecatedStatus}
	_, err = mm.UpdateMethod(ctx, method2.ID, updates)
	if err != nil {
		t.Fatalf("Failed to update method 2 status: %v", err)
	}

	// Test listing all methods
	methods, err := mm.ListMethods(ctx, MethodFilter{})
	if err != nil {
		t.Errorf("Failed to list methods: %v", err)
		return
	}
	if len(methods) != 2 {
		t.Errorf("Expected 2 methods, got %d", len(methods))
	}

	// Test filtering by domain
	generalDomain := MethodDomainGeneral
	generalMethods, err := mm.ListMethods(ctx, MethodFilter{Domain: &generalDomain})
	if err != nil {
		t.Errorf("Failed to list methods by domain: %v", err)
		return
	}
	if len(generalMethods) != 1 {
		t.Errorf("Expected 1 general method, got %d", len(generalMethods))
	}
	if generalMethods[0].ID != method1.ID {
		t.Errorf("Expected method 1 ID, got %s", generalMethods[0].ID)
	}

	// Test filtering by status
	activeStatus := MethodStatusActive
	activeMethods, err := mm.ListMethods(ctx, MethodFilter{Status: &activeStatus})
	if err != nil {
		t.Errorf("Failed to list methods by status: %v", err)
		return
	}
	if len(activeMethods) != 1 {
		t.Errorf("Expected 1 active method, got %d", len(activeMethods))
	}
}

func TestMethodManager_CreateMethodEvolution(t *testing.T) {
	store := setupTestStore(t)
	mm := NewMethodManager(store)
	ctx := context.Background()

	// Create original method
	approach := []ApproachStep{
		{Description: "Original step", Tools: []string{"tool1"}},
	}
	originalMethod, err := mm.CreateMethod(ctx, "Original Method", "Original description", approach, MethodDomainGeneral, nil)
	if err != nil {
		t.Fatalf("Failed to create original method: %v", err)
	}

	// Create evolved method
	newApproach := []ApproachStep{
		{Description: "Improved step", Tools: []string{"tool1", "tool2"}},
		{Description: "New step", Heuristics: []string{"optimization"}},
	}
	evolvedMethod := &Method{
		Name:        "Evolved Method",
		Description: "Evolved description",
		Approach:    newApproach,
		Domain:      MethodDomainGeneral,
		Version:     "2.0.0",
		Status:      MethodStatusActive,
		UserContext: map[string]interface{}{"evolution": true},
		CreatedAt:   time.Now(),
		store:       store,
	}

	// Create evolution relationship
	err = mm.CreateMethodEvolution(ctx, originalMethod.ID, evolvedMethod, "Improved performance and added new capabilities")
	if err != nil {
		t.Errorf("Failed to create method evolution: %v", err)
		return
	}

	// Verify evolution chain
	evolutionChain, err := mm.GetMethodEvolution(ctx, evolvedMethod.ID)
	if err != nil {
		t.Errorf("Failed to get method evolution: %v", err)
		return
	}

	if len(evolutionChain.Predecessors) != 1 {
		t.Errorf("Expected 1 predecessor, got %d", len(evolutionChain.Predecessors))
	}
	if evolutionChain.Predecessors[0].ID != originalMethod.ID {
		t.Errorf("Expected predecessor ID %s, got %s", originalMethod.ID, evolutionChain.Predecessors[0].ID)
	}

	// Verify original method was superseded
	updatedOriginal, err := mm.GetMethod(ctx, originalMethod.ID)
	if err != nil {
		t.Errorf("Failed to get original method after evolution: %v", err)
		return
	}
	if updatedOriginal.Status != MethodStatusSuperseded {
		t.Errorf("Expected original method to be superseded, got status %v", updatedOriginal.Status)
	}
}

func TestSuccessMetrics_SuccessRate(t *testing.T) {
	tests := []struct {
		name           string
		metrics        SuccessMetrics
		expectedRate   float64
	}{
		{
			name: "perfect success rate",
			metrics: SuccessMetrics{
				ExecutionCount: 10,
				SuccessCount:   10,
			},
			expectedRate: 100.0,
		},
		{
			name: "no successes",
			metrics: SuccessMetrics{
				ExecutionCount: 5,
				SuccessCount:   0,
			},
			expectedRate: 0.0,
		},
		{
			name: "partial success rate",
			metrics: SuccessMetrics{
				ExecutionCount: 10,
				SuccessCount:   7,
			},
			expectedRate: 70.0,
		},
		{
			name: "no executions",
			metrics: SuccessMetrics{
				ExecutionCount: 0,
				SuccessCount:   0,
			},
			expectedRate: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rate := tt.metrics.SuccessRate()
			if rate != tt.expectedRate {
				t.Errorf("Expected success rate %f, got %f", tt.expectedRate, rate)
			}
		})
	}
}

func TestMethod_IsActive(t *testing.T) {
	method := &Method{Status: MethodStatusActive}
	if !method.IsActive() {
		t.Errorf("Expected method to be active")
	}

	method.Status = MethodStatusDeprecated
	if method.IsActive() {
		t.Errorf("Expected method not to be active when deprecated")
	}
}

func TestMethod_IsDeprecated(t *testing.T) {
	method := &Method{Status: MethodStatusDeprecated}
	if !method.IsDeprecated() {
		t.Errorf("Expected method to be deprecated")
	}

	method.Status = MethodStatusActive
	if method.IsDeprecated() {
		t.Errorf("Expected method not to be deprecated when active")
	}
}

func TestMethod_Update(t *testing.T) {
	store := setupTestStore(t)
	mm := NewMethodManager(store)
	ctx := context.Background()

	// Create a test method
	method, err := mm.CreateMethod(ctx, "Test Method", "Test description", []ApproachStep{}, MethodDomainGeneral, nil)
	if err != nil {
		t.Fatalf("Failed to create test method: %v", err)
	}

	// Update through instance method
	newName := "Updated Method Name"
	updates := MethodUpdates{Name: &newName}
	err = method.Update(ctx, updates)
	if err != nil {
		t.Errorf("Failed to update method through instance: %v", err)
		return
	}

	// Verify update
	if method.Name != newName {
		t.Errorf("Expected updated name %q, got %q", newName, method.Name)
	}
}

func TestMethod_RecordExecution(t *testing.T) {
	store := setupTestStore(t)
	mm := NewMethodManager(store)
	ctx := context.Background()

	// Create a test method
	method, err := mm.CreateMethod(ctx, "Test Method", "Test description", []ApproachStep{}, MethodDomainGeneral, nil)
	if err != nil {
		t.Fatalf("Failed to create test method: %v", err)
	}

	// Record execution through instance method
	err = method.RecordExecution(ctx, true, 9.0)
	if err != nil {
		t.Errorf("Failed to record execution through instance: %v", err)
		return
	}

	// Refresh method to get updated metrics
	updatedMethod, err := mm.GetMethod(ctx, method.ID)
	if err != nil {
		t.Fatalf("Failed to get updated method: %v", err)
	}

	// Verify metrics were updated
	if updatedMethod.Metrics.ExecutionCount != 1 {
		t.Errorf("Expected execution count 1, got %d", updatedMethod.Metrics.ExecutionCount)
	}
	if updatedMethod.Metrics.SuccessCount != 1 {
		t.Errorf("Expected success count 1, got %d", updatedMethod.Metrics.SuccessCount)
	}
}

func TestMethodDomain_String(t *testing.T) {
	domain := MethodDomainGeneral
	if domain.String() != "general" {
		t.Errorf("Expected domain string 'general', got %q", domain.String())
	}
}

func TestMethodStatus_String(t *testing.T) {
	status := MethodStatusActive
	if status.String() != "active" {
		t.Errorf("Expected status string 'active', got %q", status.String())
	}
}

func TestMethodManager_GetMethodAtTime(t *testing.T) {
	store := setupTestStore(t)
	mm := NewMethodManager(store)
	ctx := context.Background()

	// Create a method
	method, err := mm.CreateMethod(ctx, "Test Method", "Original description", []ApproachStep{}, MethodDomainGeneral, nil)
	if err != nil {
		t.Fatalf("Failed to create method: %v", err)
	}

	// Store creation time
	creationTime := time.Now()

	// Wait a bit and update the method
	time.Sleep(time.Millisecond)
	newDesc := "Updated description"
	updates := MethodUpdates{Description: &newDesc}
	_, err = mm.UpdateMethod(ctx, method.ID, updates)
	if err != nil {
		t.Fatalf("Failed to update method: %v", err)
	}

	// Get method at creation time
	historicalMethod, err := mm.GetMethodAtTime(ctx, method.ID, creationTime)
	if err != nil {
		t.Errorf("Failed to get method at time: %v", err)
		return
	}

	// Should have original description
	if historicalMethod.Description != "Original description" {
		t.Errorf("Expected original description, got %q", historicalMethod.Description)
	}

	// Get current method
	currentMethod, err := mm.GetMethod(ctx, method.ID)
	if err != nil {
		t.Fatalf("Failed to get current method: %v", err)
	}

	// Should have updated description
	if currentMethod.Description != "Updated description" {
		t.Errorf("Expected updated description, got %q", currentMethod.Description)
	}
}