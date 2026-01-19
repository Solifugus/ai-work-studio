package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateFile(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	tests := []struct {
		name           string
		createFile     func(string) string
		expectValid    bool
		expectErrors   int
		expectWarnings int
		entityType     string
	}{
		{
			name: "valid node file",
			createFile: func(dir string) string {
				nodeDir := filepath.Join(dir, "nodes", "goal")
				os.MkdirAll(nodeDir, 0755)
				filePath := filepath.Join(nodeDir, "test_node.json")

				node := NewNode("goal", map[string]interface{}{"title": "Test Goal"})
				history := NodeHistory{node}
				data, _ := json.MarshalIndent(history, "", "  ")
				os.WriteFile(filePath, data, 0644)
				return filePath
			},
			expectValid:  true,
			expectErrors: 0,
			entityType:   "node",
		},
		{
			name: "valid edge file",
			createFile: func(dir string) string {
				edgeDir := filepath.Join(dir, "edges")
				os.MkdirAll(edgeDir, 0755)
				filePath := filepath.Join(edgeDir, "test_edge.json")

				edge := NewEdge("source-id", "target-id", "depends_on", map[string]interface{}{"weight": 1.0})
				history := EdgeHistory{edge}
				data, _ := json.MarshalIndent(history, "", "  ")
				os.WriteFile(filePath, data, 0644)
				return filePath
			},
			expectValid:  true,
			expectErrors: 0,
			entityType:   "edge",
		},
		{
			name: "invalid JSON",
			createFile: func(dir string) string {
				nodeDir := filepath.Join(dir, "nodes", "goal")
				os.MkdirAll(nodeDir, 0755)
				filePath := filepath.Join(nodeDir, "invalid.json")
				os.WriteFile(filePath, []byte("{invalid json"), 0644)
				return filePath
			},
			expectValid:  false,
			expectErrors: 1,
			entityType:   "node",
		},
		{
			name: "node with missing fields",
			createFile: func(dir string) string {
				nodeDir := filepath.Join(dir, "nodes", "goal")
				os.MkdirAll(nodeDir, 0755)
				filePath := filepath.Join(nodeDir, "missing_fields.json")

				// Create node with missing required fields
				invalidNode := &Node{
					ID:   "", // Missing ID
					Type: "", // Missing Type
					Data: nil, // Missing Data
				}
				history := NodeHistory{invalidNode}
				data, _ := json.MarshalIndent(history, "", "  ")
				os.WriteFile(filePath, data, 0644)
				return filePath
			},
			expectValid:  false,
			expectErrors: 5, // Missing ID, Type, Data, CreatedAt, ValidFrom
			entityType:   "node",
		},
		{
			name: "edge with same source and target",
			createFile: func(dir string) string {
				edgeDir := filepath.Join(dir, "edges")
				os.MkdirAll(edgeDir, 0755)
				filePath := filepath.Join(edgeDir, "self_edge.json")

				edge := NewEdge("same-id", "same-id", "self_ref", map[string]interface{}{})
				history := EdgeHistory{edge}
				data, _ := json.MarshalIndent(history, "", "  ")
				os.WriteFile(filePath, data, 0644)
				return filePath
			},
			expectValid:  false,
			expectErrors: 1, // SourceID == TargetID
			entityType:   "edge",
		},
		{
			name: "node with invalid time relationships",
			createFile: func(dir string) string {
				nodeDir := filepath.Join(dir, "nodes", "goal")
				os.MkdirAll(nodeDir, 0755)
				filePath := filepath.Join(nodeDir, "invalid_times.json")

				now := time.Now()
				node := &Node{
					ID:         uuid.New().String(),
					Type:       "goal",
					Data:       map[string]interface{}{"title": "Test"},
					CreatedAt:  now.Add(time.Hour),  // CreatedAt after ValidFrom
					ValidFrom:  now,                  // ValidFrom
					ValidUntil: now.Add(-time.Hour), // ValidUntil before ValidFrom
				}
				history := NodeHistory{node}
				data, _ := json.MarshalIndent(history, "", "  ")
				os.WriteFile(filePath, data, 0644)
				return filePath
			},
			expectValid:    false,
			expectErrors:   2, // CreatedAt after ValidFrom, ValidUntil before ValidFrom
			expectWarnings: 1, // No current version found (ValidUntil is in the past)
			entityType:     "node",
		},
		{
			name: "multiple current versions",
			createFile: func(dir string) string {
				nodeDir := filepath.Join(dir, "nodes", "goal")
				os.MkdirAll(nodeDir, 0755)
				filePath := filepath.Join(nodeDir, "multiple_current.json")

				id := uuid.New().String()
				now := time.Now()

				// Create two versions, both current (ValidUntil is zero)
				node1 := &Node{
					ID:         id,
					Type:       "goal",
					Data:       map[string]interface{}{"version": 1},
					CreatedAt:  now,
					ValidFrom:  now,
					ValidUntil: time.Time{}, // Current
				}
				node2 := &Node{
					ID:         id,
					Type:       "goal",
					Data:       map[string]interface{}{"version": 2},
					CreatedAt:  now.Add(time.Minute),
					ValidFrom:  now.Add(time.Minute),
					ValidUntil: time.Time{}, // Also current - this is invalid
				}
				history := NodeHistory{node1, node2}
				data, _ := json.MarshalIndent(history, "", "  ")
				os.WriteFile(filePath, data, 0644)
				return filePath
			},
			expectValid:  false,
			expectErrors: 1, // Multiple current versions
			entityType:   "node",
		},
		{
			name: "empty history with warning",
			createFile: func(dir string) string {
				nodeDir := filepath.Join(dir, "nodes", "goal")
				os.MkdirAll(nodeDir, 0755)
				filePath := filepath.Join(nodeDir, "empty.json")

				history := NodeHistory{}
				data, _ := json.MarshalIndent(history, "", "  ")
				os.WriteFile(filePath, data, 0644)
				return filePath
			},
			expectValid:    true,
			expectErrors:   0,
			expectWarnings: 1, // Empty history warning
			entityType:     "node",
		},
		{
			name: "unknown file type",
			createFile: func(dir string) string {
				unknownDir := filepath.Join(dir, "unknown")
				os.MkdirAll(unknownDir, 0755)
				filePath := filepath.Join(unknownDir, "unknown.json")
				os.WriteFile(filePath, []byte("{}"), 0644)
				return filePath
			},
			expectValid:  false,
			expectErrors: 1, // Unknown file type
			entityType:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := tt.createFile(tempDir)
			result := ValidateFile(filePath)

			if result.Valid != tt.expectValid {
				t.Errorf("expected Valid=%v, got %v", tt.expectValid, result.Valid)
			}

			if len(result.Errors) != tt.expectErrors {
				t.Errorf("expected %d errors, got %d: %v", tt.expectErrors, len(result.Errors), result.Errors)
			}

			if len(result.Warnings) != tt.expectWarnings {
				t.Errorf("expected %d warnings, got %d: %v", tt.expectWarnings, len(result.Warnings), result.Warnings)
			}

			if result.EntityType != tt.entityType {
				t.Errorf("expected EntityType=%s, got %s", tt.entityType, result.EntityType)
			}
		})
	}
}

func TestValidateDataDirectory(t *testing.T) {
	tempDir := t.TempDir()

	// Create valid data structure
	nodesDir := filepath.Join(tempDir, "nodes", "goal")
	edgesDir := filepath.Join(tempDir, "edges")
	os.MkdirAll(nodesDir, 0755)
	os.MkdirAll(edgesDir, 0755)

	// Create some valid files
	node1 := NewNode("goal", map[string]interface{}{"title": "Goal 1"})
	node2 := NewNode("goal", map[string]interface{}{"title": "Goal 2"})
	edge1 := NewEdge(node1.ID, node2.ID, "depends_on", map[string]interface{}{})

	// Write node files
	node1History := NodeHistory{node1}
	node1Data, _ := json.MarshalIndent(node1History, "", "  ")
	os.WriteFile(filepath.Join(nodesDir, node1.ID+".json"), node1Data, 0644)

	node2History := NodeHistory{node2}
	node2Data, _ := json.MarshalIndent(node2History, "", "  ")
	os.WriteFile(filepath.Join(nodesDir, node2.ID+".json"), node2Data, 0644)

	// Write edge file
	edge1History := EdgeHistory{edge1}
	edge1Data, _ := json.MarshalIndent(edge1History, "", "  ")
	os.WriteFile(filepath.Join(edgesDir, edge1.ID+".json"), edge1Data, 0644)

	// Create one invalid file
	os.WriteFile(filepath.Join(nodesDir, "invalid.json"), []byte("{invalid"), 0644)

	// Validate directory
	results, err := ValidateDataDirectory(tempDir)
	if err != nil {
		t.Fatalf("ValidateDataDirectory failed: %v", err)
	}

	// Should have 4 results: 2 valid nodes, 1 valid edge, 1 invalid node
	if len(results) != 4 {
		t.Errorf("expected 4 results, got %d", len(results))
	}

	// Count valid and invalid results
	validCount := 0
	invalidCount := 0
	for _, result := range results {
		if result.Valid {
			validCount++
		} else {
			invalidCount++
		}
	}

	if validCount != 3 {
		t.Errorf("expected 3 valid results, got %d", validCount)
	}
	if invalidCount != 1 {
		t.Errorf("expected 1 invalid result, got %d", invalidCount)
	}
}

func TestHealthCheck(t *testing.T) {
	tempDir := t.TempDir()

	// Create test data
	nodesDir := filepath.Join(tempDir, "nodes", "goal")
	edgesDir := filepath.Join(tempDir, "edges")
	os.MkdirAll(nodesDir, 0755)
	os.MkdirAll(edgesDir, 0755)

	// Create valid node
	node := NewNode("goal", map[string]interface{}{"title": "Test Goal"})
	nodeHistory := NodeHistory{node}
	nodeData, _ := json.MarshalIndent(nodeHistory, "", "  ")
	os.WriteFile(filepath.Join(nodesDir, node.ID+".json"), nodeData, 0644)

	// Create valid edge
	edge := NewEdge("source", "target", "depends_on", map[string]interface{}{})
	edgeHistory := EdgeHistory{edge}
	edgeData, _ := json.MarshalIndent(edgeHistory, "", "  ")
	os.WriteFile(filepath.Join(edgesDir, edge.ID+".json"), edgeData, 0644)

	// Create invalid file
	os.WriteFile(filepath.Join(nodesDir, "invalid.json"), []byte("{broken"), 0644)

	// Run health check
	stats, err := HealthCheck(tempDir)
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	// Verify statistics
	expectedStats := map[string]int{
		"total_files":    3,
		"valid_files":    2,
		"invalid_files":  1,
		"node_files":     2, // 1 valid + 1 invalid
		"edge_files":     1,
	}

	for key, expected := range expectedStats {
		if actual, ok := stats[key].(int); !ok || actual != expected {
			t.Errorf("expected %s=%d, got %v", key, expected, stats[key])
		}
	}

	// Should have some errors from the invalid file
	if errorsCount, ok := stats["errors_count"].(int); !ok || errorsCount == 0 {
		t.Errorf("expected some errors, got %v", stats["errors_count"])
	}
}

func TestValidationError(t *testing.T) {
	err := ValidationError{
		Type:     "node",
		ID:       "test-id",
		FilePath: "/path/to/file.json",
		Issue:    "missing required field",
		Cause:    nil,
	}

	expected := "validation failed for node test-id in /path/to/file.json: missing required field"
	if err.Error() != expected {
		t.Errorf("expected error message: %s, got: %s", expected, err.Error())
	}

	// Test error without ID
	err2 := ValidationError{
		Type:     "edge",
		FilePath: "/path/to/file.json",
		Issue:    "invalid format",
	}

	expected2 := "validation failed for edge in /path/to/file.json: invalid format"
	if err2.Error() != expected2 {
		t.Errorf("expected error message: %s, got: %s", expected2, err2.Error())
	}
}

func TestValidationResultMethods(t *testing.T) {
	result := &ValidationResult{
		FilePath:   "/test/file.json",
		Valid:      true,
		EntityType: "node",
		EntityID:   "test-id",
	}

	// Test AddError
	result.AddError("test error", nil)
	if result.Valid {
		t.Error("expected Valid to be false after adding error")
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0].Issue != "test error" {
		t.Errorf("expected error issue 'test error', got '%s'", result.Errors[0].Issue)
	}

	// Test AddWarning
	result.AddWarning("test warning")
	if len(result.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(result.Warnings))
	}
	if result.Warnings[0] != "test warning" {
		t.Errorf("expected warning 'test warning', got '%s'", result.Warnings[0])
	}
}

func TestValidateNodeWithFutureValidFrom(t *testing.T) {
	tempDir := t.TempDir()
	nodeDir := filepath.Join(tempDir, "nodes", "goal")
	os.MkdirAll(nodeDir, 0755)
	filePath := filepath.Join(nodeDir, "future_node.json")

	// Create node with ValidFrom far in the future
	now := time.Now()
	futureTime := now.Add(48 * time.Hour) // 2 days in future
	node := &Node{
		ID:         uuid.New().String(),
		Type:       "goal",
		Data:       map[string]interface{}{"title": "Future Goal"},
		CreatedAt:  now,
		ValidFrom:  futureTime,
		ValidUntil: time.Time{},
	}

	history := NodeHistory{node}
	data, _ := json.MarshalIndent(history, "", "  ")
	os.WriteFile(filePath, data, 0644)

	result := ValidateFile(filePath)

	// Should be valid but have a warning about future ValidFrom
	if !result.Valid {
		t.Errorf("expected valid result, got invalid with errors: %v", result.Errors)
	}

	if len(result.Warnings) == 0 {
		t.Error("expected warning about future ValidFrom")
	}

	// Check that the warning mentions future ValidFrom
	foundWarning := false
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "future") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Errorf("expected warning about future ValidFrom, got warnings: %v", result.Warnings)
	}
}

// Helper function to create a temporary data directory with test files
func createTestDataDir(t *testing.T) string {
	tempDir := t.TempDir()

	// Create directory structure
	nodesDir := filepath.Join(tempDir, "nodes", "goal")
	edgesDir := filepath.Join(tempDir, "edges")
	os.MkdirAll(nodesDir, 0755)
	os.MkdirAll(edgesDir, 0755)

	// Create some test data
	node1 := NewNode("goal", map[string]interface{}{"title": "Test Goal 1"})
	node2 := NewNode("goal", map[string]interface{}{"title": "Test Goal 2"})
	edge := NewEdge(node1.ID, node2.ID, "depends_on", map[string]interface{}{})

	// Write files
	writeNodeHistory(t, nodesDir, node1.ID, NodeHistory{node1})
	writeNodeHistory(t, nodesDir, node2.ID, NodeHistory{node2})
	writeEdgeHistory(t, edgesDir, edge.ID, EdgeHistory{edge})

	return tempDir
}

func writeNodeHistory(t *testing.T, dir, nodeID string, history NodeHistory) {
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal node history: %v", err)
	}
	err = os.WriteFile(filepath.Join(dir, nodeID+".json"), data, 0644)
	if err != nil {
		t.Fatalf("failed to write node file: %v", err)
	}
}

func writeEdgeHistory(t *testing.T, dir, edgeID string, history EdgeHistory) {
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal edge history: %v", err)
	}
	err = os.WriteFile(filepath.Join(dir, edgeID+".json"), data, 0644)
	if err != nil {
		t.Fatalf("failed to write edge file: %v", err)
	}
}