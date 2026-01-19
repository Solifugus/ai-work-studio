package storage

import (
	"testing"
	"time"
)

func TestNewEdge(t *testing.T) {
	sourceID := "source-node-id"
	targetID := "target-node-id"
	edgeType := "depends_on"
	data := map[string]interface{}{
		"strength":    0.8,
		"description": "Goal depends on completing this method",
		"critical":    true,
	}

	edge := NewEdge(sourceID, targetID, edgeType, data)

	// Check basic fields
	if edge.ID == "" {
		t.Error("Edge ID should not be empty")
	}
	if edge.SourceID != sourceID {
		t.Errorf("Expected sourceID %s, got %s", sourceID, edge.SourceID)
	}
	if edge.TargetID != targetID {
		t.Errorf("Expected targetID %s, got %s", targetID, edge.TargetID)
	}
	if edge.Type != edgeType {
		t.Errorf("Expected type %s, got %s", edgeType, edge.Type)
	}
	if len(edge.Data) != len(data) {
		t.Errorf("Expected %d data fields, got %d", len(data), len(edge.Data))
	}

	// Check temporal fields
	if edge.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if edge.ValidFrom.IsZero() {
		t.Error("ValidFrom should be set")
	}
	if !edge.ValidUntil.IsZero() {
		t.Error("ValidUntil should be zero for new edge")
	}
	if !edge.IsCurrent() {
		t.Error("New edge should be current")
	}
}

func TestNewEdgeWithID(t *testing.T) {
	id := "test-edge-id"
	sourceID := "source"
	targetID := "target"
	edgeType := "implements"
	data := map[string]interface{}{
		"confidence": 0.95,
	}

	edge := NewEdgeWithID(id, sourceID, targetID, edgeType, data)

	if edge.ID != id {
		t.Errorf("Expected ID %s, got %s", id, edge.ID)
	}
	if edge.SourceID != sourceID {
		t.Errorf("Expected sourceID %s, got %s", sourceID, edge.SourceID)
	}
	if edge.TargetID != targetID {
		t.Errorf("Expected targetID %s, got %s", targetID, edge.TargetID)
	}
	if !edge.IsCurrent() {
		t.Error("New edge with ID should be current")
	}
}

func TestEdgeIsCurrent(t *testing.T) {
	// Test current edge
	currentEdge := NewEdge("source", "target", "test", map[string]interface{}{})
	if !currentEdge.IsCurrent() {
		t.Error("Edge with zero ValidUntil should be current")
	}

	// Test superseded edge
	supersededEdge := NewEdge("source", "target", "test", map[string]interface{}{})
	supersededEdge.ValidUntil = time.Now()
	if supersededEdge.IsCurrent() {
		t.Error("Edge with ValidUntil set should not be current")
	}
}

func TestEdgeIsActiveAt(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Create an edge that was valid from base time to one hour later
	edge := &Edge{
		ID:         "test-edge",
		SourceID:   "source",
		TargetID:   "target",
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
			result := edge.IsActiveAt(tt.timestamp)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for timestamp %v", tt.expected, result, tt.timestamp)
			}
		})
	}

	// Test current edge (ValidUntil is zero)
	currentEdge := NewEdge("source", "target", "test", map[string]interface{}{})
	futureTime := currentEdge.ValidFrom.Add(24 * time.Hour)
	if !currentEdge.IsActiveAt(futureTime) {
		t.Error("Current edge should be active at future time")
	}
}

func TestEdgeSupersede(t *testing.T) {
	edge := NewEdge("source", "target", "test", map[string]interface{}{})
	supersedeTime := time.Now().Add(time.Hour)

	// Initially current
	if !edge.IsCurrent() {
		t.Error("Edge should be current initially")
	}

	// Supersede
	edge.Supersede(supersedeTime)
	if edge.IsCurrent() {
		t.Error("Edge should not be current after superseding")
	}
	if !edge.ValidUntil.Equal(supersedeTime) {
		t.Error("ValidUntil should be set to supersede time")
	}

	// Superseding again should not change ValidUntil
	newTime := time.Now().Add(2 * time.Hour)
	edge.Supersede(newTime)
	if !edge.ValidUntil.Equal(supersedeTime) {
		t.Error("ValidUntil should not change when superseding already superseded edge")
	}
}

func TestEdgeConnectsNode(t *testing.T) {
	edge := NewEdge("node1", "node2", "test", map[string]interface{}{})

	if !edge.ConnectsNode("node1") {
		t.Error("Edge should connect to source node")
	}
	if !edge.ConnectsNode("node2") {
		t.Error("Edge should connect to target node")
	}
	if edge.ConnectsNode("node3") {
		t.Error("Edge should not connect to unrelated node")
	}
}

func TestEdgeConnectsNodes(t *testing.T) {
	edge := NewEdge("node1", "node2", "test", map[string]interface{}{})

	if !edge.ConnectsNodes("node1", "node2") {
		t.Error("Edge should connect node1 and node2 (forward)")
	}
	if !edge.ConnectsNodes("node2", "node1") {
		t.Error("Edge should connect node1 and node2 (reverse)")
	}
	if edge.ConnectsNodes("node1", "node3") {
		t.Error("Edge should not connect node1 and node3")
	}
}

func TestEdgeIsOutgoing(t *testing.T) {
	edge := NewEdge("node1", "node2", "test", map[string]interface{}{})

	if !edge.IsOutgoing("node1") {
		t.Error("Edge should be outgoing from source node")
	}
	if edge.IsOutgoing("node2") {
		t.Error("Edge should not be outgoing from target node")
	}
	if edge.IsOutgoing("node3") {
		t.Error("Edge should not be outgoing from unrelated node")
	}
}

func TestEdgeIsIncoming(t *testing.T) {
	edge := NewEdge("node1", "node2", "test", map[string]interface{}{})

	if edge.IsIncoming("node1") {
		t.Error("Edge should not be incoming to source node")
	}
	if !edge.IsIncoming("node2") {
		t.Error("Edge should be incoming to target node")
	}
	if edge.IsIncoming("node3") {
		t.Error("Edge should not be incoming to unrelated node")
	}
}

func TestEdgeClone(t *testing.T) {
	original := NewEdge("source", "target", "test", map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	})

	clone := original.Clone()

	// Check that all fields are copied
	if clone.ID != original.ID {
		t.Error("Clone should have same ID")
	}
	if clone.SourceID != original.SourceID {
		t.Error("Clone should have same SourceID")
	}
	if clone.TargetID != original.TargetID {
		t.Error("Clone should have same TargetID")
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

func TestEdgeJSON(t *testing.T) {
	original := NewEdge("source", "target", "depends_on", map[string]interface{}{
		"strength":    0.8,
		"description": "Test dependency",
		"critical":    true,
	})

	// Serialize to JSON
	jsonData, err := original.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize edge to JSON: %v", err)
	}

	// Deserialize from JSON
	restored, err := EdgeFromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize edge from JSON: %v", err)
	}

	// Compare fields
	if restored.ID != original.ID {
		t.Error("Restored edge should have same ID")
	}
	if restored.SourceID != original.SourceID {
		t.Error("Restored edge should have same SourceID")
	}
	if restored.TargetID != original.TargetID {
		t.Error("Restored edge should have same TargetID")
	}
	if restored.Type != original.Type {
		t.Error("Restored edge should have same Type")
	}

	// Compare timestamps (with tolerance for JSON precision)
	if !restored.CreatedAt.Truncate(time.Millisecond).Equal(original.CreatedAt.Truncate(time.Millisecond)) {
		t.Error("Restored edge should have same CreatedAt")
	}
	if !restored.ValidFrom.Truncate(time.Millisecond).Equal(original.ValidFrom.Truncate(time.Millisecond)) {
		t.Error("Restored edge should have same ValidFrom")
	}
	if !restored.ValidUntil.Truncate(time.Millisecond).Equal(original.ValidUntil.Truncate(time.Millisecond)) {
		t.Error("Restored edge should have same ValidUntil")
	}

	// Compare data
	if len(restored.Data) != len(original.Data) {
		t.Error("Restored edge should have same data length")
	}
	for k, v := range original.Data {
		if restored.Data[k] != v {
			t.Errorf("Restored data should match original for key %s", k)
		}
	}
}

func TestEdgeHistory(t *testing.T) {
	// Create a history with multiple versions of the SAME edge
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	edgeID := "test-edge"

	// Version 1 (superseded)
	v1 := &Edge{
		ID:         edgeID,
		SourceID:   "node1",
		TargetID:   "node2",
		Type:       "depends_on",
		Data:       map[string]interface{}{"version": 1},
		CreatedAt:  baseTime,
		ValidFrom:  baseTime,
		ValidUntil: baseTime.Add(time.Hour),
	}

	// Version 2 (superseded)
	v2 := &Edge{
		ID:         edgeID,
		SourceID:   "node1",
		TargetID:   "node2",
		Type:       "depends_on",
		Data:       map[string]interface{}{"version": 2},
		CreatedAt:  baseTime.Add(time.Hour),
		ValidFrom:  baseTime.Add(time.Hour),
		ValidUntil: baseTime.Add(2 * time.Hour),
	}

	// Version 3 (current)
	v3 := &Edge{
		ID:         edgeID,
		SourceID:   "node1",
		TargetID:   "node2",
		Type:       "depends_on",
		Data:       map[string]interface{}{"version": 3},
		CreatedAt:  baseTime.Add(2 * time.Hour),
		ValidFrom:  baseTime.Add(2 * time.Hour),
		ValidUntil: time.Time{}, // Current
	}

	history := EdgeHistory{v2, v1, v3} // Intentionally unordered

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
}

func TestEdgeHistoryFiltering(t *testing.T) {
	// Create test edges
	e1 := NewEdge("node1", "node2", "depends_on", map[string]interface{}{})
	e2 := NewEdge("node2", "node3", "implements", map[string]interface{}{})
	e3 := NewEdge("node1", "node3", "depends_on", map[string]interface{}{})
	e4 := NewEdge("node4", "node5", "refines", map[string]interface{}{})

	history := EdgeHistory{e1, e2, e3, e4}

	// Test FilterByNodes
	connectedToNode1 := history.FilterByNodes("node1")
	if len(connectedToNode1) != 2 {
		t.Errorf("Expected 2 edges connected to node1, got %d", len(connectedToNode1))
	}

	connectedToNode2 := history.FilterByNodes("node2")
	if len(connectedToNode2) != 2 {
		t.Errorf("Expected 2 edges connected to node2, got %d", len(connectedToNode2))
	}

	connectedToMultiple := history.FilterByNodes("node1", "node4")
	if len(connectedToMultiple) != 3 {
		t.Errorf("Expected 3 edges connected to node1 or node4, got %d", len(connectedToMultiple))
	}

	// Test FilterByType
	dependsOnEdges := history.FilterByType("depends_on")
	if len(dependsOnEdges) != 2 {
		t.Errorf("Expected 2 'depends_on' edges, got %d", len(dependsOnEdges))
	}

	implementsEdges := history.FilterByType("implements")
	if len(implementsEdges) != 1 {
		t.Errorf("Expected 1 'implements' edge, got %d", len(implementsEdges))
	}

	// Test FilterOutgoing
	outgoingFromNode1 := history.FilterOutgoing("node1")
	if len(outgoingFromNode1) != 2 {
		t.Errorf("Expected 2 edges outgoing from node1, got %d", len(outgoingFromNode1))
	}

	outgoingFromNode2 := history.FilterOutgoing("node2")
	if len(outgoingFromNode2) != 1 {
		t.Errorf("Expected 1 edge outgoing from node2, got %d", len(outgoingFromNode2))
	}

	// Test FilterIncoming
	incomingToNode2 := history.FilterIncoming("node2")
	if len(incomingToNode2) != 1 {
		t.Errorf("Expected 1 edge incoming to node2, got %d", len(incomingToNode2))
	}

	incomingToNode3 := history.FilterIncoming("node3")
	if len(incomingToNode3) != 2 {
		t.Errorf("Expected 2 edges incoming to node3, got %d", len(incomingToNode3))
	}

	// Test chaining filters
	dependsOnFromNode1 := history.FilterOutgoing("node1").FilterByType("depends_on")
	if len(dependsOnFromNode1) != 2 {
		t.Errorf("Expected 2 'depends_on' edges from node1, got %d", len(dependsOnFromNode1))
	}
}