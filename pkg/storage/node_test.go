package storage

import (
	"testing"
	"time"
)

func TestNewNode(t *testing.T) {
	nodeType := "goal"
	data := map[string]interface{}{
		"title":       "Complete project",
		"description": "Finish the AI Work Studio implementation",
		"priority":    5,
	}

	node := NewNode(nodeType, data)

	// Check basic fields
	if node.ID == "" {
		t.Error("Node ID should not be empty")
	}
	if node.Type != nodeType {
		t.Errorf("Expected type %s, got %s", nodeType, node.Type)
	}
	if len(node.Data) != len(data) {
		t.Errorf("Expected %d data fields, got %d", len(data), len(node.Data))
	}

	// Check temporal fields
	if node.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if node.ValidFrom.IsZero() {
		t.Error("ValidFrom should be set")
	}
	if !node.ValidUntil.IsZero() {
		t.Error("ValidUntil should be zero for new node")
	}
	if !node.IsCurrent() {
		t.Error("New node should be current")
	}
}

func TestNewNodeWithID(t *testing.T) {
	id := "test-node-id"
	nodeType := "method"
	data := map[string]interface{}{
		"name": "file_organizer",
		"success_rate": 0.85,
	}

	node := NewNodeWithID(id, nodeType, data)

	if node.ID != id {
		t.Errorf("Expected ID %s, got %s", id, node.ID)
	}
	if node.Type != nodeType {
		t.Errorf("Expected type %s, got %s", nodeType, node.Type)
	}
	if !node.IsCurrent() {
		t.Error("New node with ID should be current")
	}
}

func TestNodeIsCurrent(t *testing.T) {
	// Test current node
	currentNode := NewNode("test", map[string]interface{}{})
	if !currentNode.IsCurrent() {
		t.Error("Node with zero ValidUntil should be current")
	}

	// Test superseded node
	supersededNode := NewNode("test", map[string]interface{}{})
	supersededNode.ValidUntil = time.Now()
	if supersededNode.IsCurrent() {
		t.Error("Node with ValidUntil set should not be current")
	}
}

func TestNodeIsActiveAt(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Create a node that was valid from base time to one hour later
	node := &Node{
		ID:         "test-node",
		Type:       "test",
		Data:       map[string]interface{}{},
		CreatedAt:  baseTime,
		ValidFrom:  baseTime,
		ValidUntil: baseTime.Add(time.Hour),
	}

	tests := []struct {
		name      string
		timestamp time.Time
		expected  bool
	}{
		{
			name:      "Before ValidFrom",
			timestamp: baseTime.Add(-time.Minute),
			expected:  false,
		},
		{
			name:      "At ValidFrom",
			timestamp: baseTime,
			expected:  true,
		},
		{
			name:      "During validity period",
			timestamp: baseTime.Add(30 * time.Minute),
			expected:  true,
		},
		{
			name:      "At ValidUntil",
			timestamp: baseTime.Add(time.Hour),
			expected:  false, // ValidUntil is exclusive
		},
		{
			name:      "After ValidUntil",
			timestamp: baseTime.Add(2 * time.Hour),
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := node.IsActiveAt(tt.timestamp)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for timestamp %v", tt.expected, result, tt.timestamp)
			}
		})
	}

	// Test current node (ValidUntil is zero)
	currentNode := NewNode("test", map[string]interface{}{})
	futureTime := currentNode.ValidFrom.Add(24 * time.Hour)
	if !currentNode.IsActiveAt(futureTime) {
		t.Error("Current node should be active at future time")
	}
}

func TestNodeSupersede(t *testing.T) {
	node := NewNode("test", map[string]interface{}{})
	supersedeTime := time.Now().Add(time.Hour)

	// Initially current
	if !node.IsCurrent() {
		t.Error("Node should be current initially")
	}

	// Supersede
	node.Supersede(supersedeTime)
	if node.IsCurrent() {
		t.Error("Node should not be current after superseding")
	}
	if !node.ValidUntil.Equal(supersedeTime) {
		t.Error("ValidUntil should be set to supersede time")
	}

	// Superseding again should not change ValidUntil
	newTime := time.Now().Add(2 * time.Hour)
	node.Supersede(newTime)
	if !node.ValidUntil.Equal(supersedeTime) {
		t.Error("ValidUntil should not change when superseding already superseded node")
	}
}

func TestNodeClone(t *testing.T) {
	original := NewNode("test", map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	})

	clone := original.Clone()

	// Check that all fields are copied
	if clone.ID != original.ID {
		t.Error("Clone should have same ID")
	}
	if clone.Type != original.Type {
		t.Error("Clone should have same Type")
	}
	if clone.CreatedAt != original.CreatedAt {
		t.Error("Clone should have same CreatedAt")
	}
	if clone.ValidFrom != original.ValidFrom {
		t.Error("Clone should have same ValidFrom")
	}
	if clone.ValidUntil != original.ValidUntil {
		t.Error("Clone should have same ValidUntil")
	}

	// Check that data is deep copied
	if len(clone.Data) != len(original.Data) {
		t.Error("Clone should have same data length")
	}
	for k, v := range original.Data {
		if clone.Data[k] != v {
			t.Errorf("Clone data should match original for key %s", k)
		}
	}

	// Modify clone data and ensure original is unchanged
	clone.Data["new_key"] = "new_value"
	if _, exists := original.Data["new_key"]; exists {
		t.Error("Modifying clone data should not affect original")
	}
}

func TestNodeJSON(t *testing.T) {
	original := NewNode("goal", map[string]interface{}{
		"title":       "Test Goal",
		"priority":    3,
		"completed":   false,
		"score":       4.5,
	})

	// Serialize to JSON
	jsonData, err := original.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize node to JSON: %v", err)
	}

	// Deserialize from JSON
	restored, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize node from JSON: %v", err)
	}

	// Compare fields
	if restored.ID != original.ID {
		t.Error("Restored node should have same ID")
	}
	if restored.Type != original.Type {
		t.Error("Restored node should have same Type")
	}

	// Compare timestamps (with tolerance for JSON precision)
	if !restored.CreatedAt.Truncate(time.Millisecond).Equal(original.CreatedAt.Truncate(time.Millisecond)) {
		t.Error("Restored node should have same CreatedAt")
	}
	if !restored.ValidFrom.Truncate(time.Millisecond).Equal(original.ValidFrom.Truncate(time.Millisecond)) {
		t.Error("Restored node should have same ValidFrom")
	}
	if !restored.ValidUntil.Truncate(time.Millisecond).Equal(original.ValidUntil.Truncate(time.Millisecond)) {
		t.Error("Restored node should have same ValidUntil")
	}

	// Compare data - handle JSON type conversions
	if len(restored.Data) != len(original.Data) {
		t.Error("Restored node should have same data length")
	}

	// Check specific values with type-aware comparison
	if restored.Data["title"] != "Test Goal" {
		t.Errorf("Expected title 'Test Goal', got %v", restored.Data["title"])
	}

	// JSON unmarshaling converts numbers to float64
	if priority, ok := restored.Data["priority"].(float64); !ok || priority != 3.0 {
		t.Errorf("Expected priority 3.0, got %v (%T)", restored.Data["priority"], restored.Data["priority"])
	}

	if completed, ok := restored.Data["completed"].(bool); !ok || completed != false {
		t.Errorf("Expected completed false, got %v", restored.Data["completed"])
	}

	if score, ok := restored.Data["score"].(float64); !ok || score != 4.5 {
		t.Errorf("Expected score 4.5, got %v", restored.Data["score"])
	}
}

func TestNodeHistory(t *testing.T) {
	// Create a history with multiple versions
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Version 1 (superseded)
	v1 := &Node{
		ID:         "test-node",
		Type:       "goal",
		Data:       map[string]interface{}{"version": 1},
		CreatedAt:  baseTime,
		ValidFrom:  baseTime,
		ValidUntil: baseTime.Add(time.Hour),
	}

	// Version 2 (superseded)
	v2 := &Node{
		ID:         "test-node",
		Type:       "goal",
		Data:       map[string]interface{}{"version": 2},
		CreatedAt:  baseTime.Add(time.Hour),
		ValidFrom:  baseTime.Add(time.Hour),
		ValidUntil: baseTime.Add(2 * time.Hour),
	}

	// Version 3 (current)
	v3 := &Node{
		ID:         "test-node",
		Type:       "goal",
		Data:       map[string]interface{}{"version": 3},
		CreatedAt:  baseTime.Add(2 * time.Hour),
		ValidFrom:  baseTime.Add(2 * time.Hour),
		ValidUntil: time.Time{}, // Current version
	}

	history := NodeHistory{v2, v1, v3} // Intentionally unordered

	// Test GetVersionAt
	versionAtStart := history.GetVersionAt(baseTime.Add(30 * time.Minute))
	if versionAtStart == nil || versionAtStart.Data["version"] != 1 {
		t.Error("Should get version 1 at start time")
	}

	versionAtMiddle := history.GetVersionAt(baseTime.Add(90 * time.Minute))
	if versionAtMiddle == nil || versionAtMiddle.Data["version"] != 2 {
		t.Error("Should get version 2 at middle time")
	}

	versionAtEnd := history.GetVersionAt(baseTime.Add(3 * time.Hour))
	if versionAtEnd == nil || versionAtEnd.Data["version"] != 3 {
		t.Error("Should get version 3 at end time")
	}

	// Test GetCurrentVersion
	current := history.GetCurrentVersion()
	if current == nil || current.Data["version"] != 3 {
		t.Error("Should get version 3 as current")
	}

	// Test GetAllVersions (should be sorted by ValidFrom)
	allVersions := history.GetAllVersions()
	if len(allVersions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(allVersions))
	}
	if allVersions[0].Data["version"] != 1 {
		t.Error("First version should be version 1")
	}
	if allVersions[1].Data["version"] != 2 {
		t.Error("Second version should be version 2")
	}
	if allVersions[2].Data["version"] != 3 {
		t.Error("Third version should be version 3")
	}

	// Test with no version at timestamp
	noVersion := history.GetVersionAt(baseTime.Add(-time.Hour))
	if noVersion != nil {
		t.Error("Should return nil for timestamp before all versions")
	}
}