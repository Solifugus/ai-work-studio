package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ValidationError represents a specific validation failure.
type ValidationError struct {
	Type     string // "node" or "edge"
	ID       string // Entity ID if available
	FilePath string // File path where error occurred
	Issue    string // Human-readable description of the issue
	Cause    error  // Underlying error if any
}

func (ve ValidationError) Error() string {
	if ve.ID != "" {
		return fmt.Sprintf("validation failed for %s %s in %s: %s", ve.Type, ve.ID, ve.FilePath, ve.Issue)
	}
	return fmt.Sprintf("validation failed for %s in %s: %s", ve.Type, ve.FilePath, ve.Issue)
}

// ValidationResult contains the results of a validation check.
type ValidationResult struct {
	FilePath   string            // File that was validated
	Valid      bool              // Whether validation passed
	Errors     []ValidationError // List of validation errors
	Warnings   []string          // Non-fatal issues worth noting
	EntityType string            // "node", "edge", or "unknown"
	EntityID   string            // Entity ID if successfully parsed
}

// AddError adds a validation error to the result.
func (vr *ValidationResult) AddError(issue string, cause error) {
	vr.Valid = false
	vr.Errors = append(vr.Errors, ValidationError{
		Type:     vr.EntityType,
		ID:       vr.EntityID,
		FilePath: vr.FilePath,
		Issue:    issue,
		Cause:    cause,
	})
}

// AddWarning adds a non-fatal warning to the result.
func (vr *ValidationResult) AddWarning(warning string) {
	vr.Warnings = append(vr.Warnings, warning)
}

// ValidateFile validates a single JSON file containing node or edge history.
// It performs structural, semantic, and temporal validation.
func ValidateFile(filePath string) *ValidationResult {
	result := &ValidationResult{
		FilePath:   filePath,
		Valid:      true,
		EntityType: "unknown",
	}

	// Read and parse the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		result.AddError(fmt.Sprintf("failed to read file: %v", err), err)
		return result
	}

	// Detect entity type from file path and content
	if strings.Contains(filePath, "/nodes/") {
		result.EntityType = "node"
		validateNodeFile(data, result)
	} else if strings.Contains(filePath, "/edges/") {
		result.EntityType = "edge"
		validateEdgeFile(data, result)
	} else {
		result.AddError("unknown file type - not in nodes/ or edges/ directory", nil)
	}

	return result
}

// validateNodeFile validates a node history JSON file.
func validateNodeFile(data []byte, result *ValidationResult) {
	// Parse JSON structure
	var history NodeHistory
	if err := json.Unmarshal(data, &history); err != nil {
		result.AddError(fmt.Sprintf("invalid JSON structure: %v", err), err)
		return
	}

	if len(history) == 0 {
		result.AddWarning("empty node history - no versions found")
		return
	}

	// Validate each node version
	seenVersions := make(map[time.Time]bool)
	var currentVersions int
	var nodeID string

	for i, node := range history {
		if node == nil {
			result.AddError(fmt.Sprintf("nil node at index %d", i), nil)
			continue
		}

		// Set entity ID from first valid node
		if nodeID == "" && node.ID != "" {
			nodeID = node.ID
			result.EntityID = nodeID
		}

		// Validate individual node
		validateNode(node, i, result)

		// Check for duplicate ValidFrom timestamps
		if seenVersions[node.ValidFrom] {
			result.AddError(fmt.Sprintf("duplicate ValidFrom timestamp %v at index %d", node.ValidFrom, i), nil)
		}
		seenVersions[node.ValidFrom] = true

		// Count current versions
		if node.IsCurrent() {
			currentVersions++
		}

		// Validate temporal consistency
		validateNodeTemporal(node, i, result)
	}

	// Validate history-level constraints
	if nodeID != "" {
		// All nodes should have same ID
		for i, node := range history {
			if node != nil && node.ID != nodeID {
				result.AddError(fmt.Sprintf("inconsistent node ID at index %d: expected %s, got %s", i, nodeID, node.ID), nil)
			}
		}
	}

	// Should have exactly one current version
	if currentVersions == 0 {
		result.AddWarning("no current version found (all versions have ValidUntil set)")
	} else if currentVersions > 1 {
		result.AddError(fmt.Sprintf("multiple current versions found: %d (should be exactly 1)", currentVersions), nil)
	}
}

// validateEdgeFile validates an edge history JSON file.
func validateEdgeFile(data []byte, result *ValidationResult) {
	// Parse JSON structure
	var history EdgeHistory
	if err := json.Unmarshal(data, &history); err != nil {
		result.AddError(fmt.Sprintf("invalid JSON structure: %v", err), err)
		return
	}

	if len(history) == 0 {
		result.AddWarning("empty edge history - no versions found")
		return
	}

	// Validate each edge version
	seenVersions := make(map[time.Time]bool)
	var currentVersions int
	var edgeID string

	for i, edge := range history {
		if edge == nil {
			result.AddError(fmt.Sprintf("nil edge at index %d", i), nil)
			continue
		}

		// Set entity ID from first valid edge
		if edgeID == "" && edge.ID != "" {
			edgeID = edge.ID
			result.EntityID = edgeID
		}

		// Validate individual edge
		validateEdge(edge, i, result)

		// Check for duplicate ValidFrom timestamps
		if seenVersions[edge.ValidFrom] {
			result.AddError(fmt.Sprintf("duplicate ValidFrom timestamp %v at index %d", edge.ValidFrom, i), nil)
		}
		seenVersions[edge.ValidFrom] = true

		// Count current versions
		if edge.IsCurrent() {
			currentVersions++
		}

		// Validate temporal consistency
		validateEdgeTemporal(edge, i, result)
	}

	// Validate history-level constraints
	if edgeID != "" {
		// All edges should have same ID
		for i, edge := range history {
			if edge != nil && edge.ID != edgeID {
				result.AddError(fmt.Sprintf("inconsistent edge ID at index %d: expected %s, got %s", i, edgeID, edge.ID), nil)
			}
		}
	}

	// Should have exactly one current version
	if currentVersions == 0 {
		result.AddWarning("no current version found (all versions have ValidUntil set)")
	} else if currentVersions > 1 {
		result.AddError(fmt.Sprintf("multiple current versions found: %d (should be exactly 1)", currentVersions), nil)
	}
}

// validateNode validates a single node version.
func validateNode(node *Node, index int, result *ValidationResult) {
	// Required fields validation
	if node.ID == "" {
		result.AddError(fmt.Sprintf("missing or empty ID at index %d", index), nil)
	}
	if node.Type == "" {
		result.AddError(fmt.Sprintf("missing or empty Type at index %d", index), nil)
	}
	if node.Data == nil {
		result.AddError(fmt.Sprintf("missing Data field at index %d", index), nil)
	}

	// Time fields validation
	if node.CreatedAt.IsZero() {
		result.AddError(fmt.Sprintf("missing or zero CreatedAt at index %d", index), nil)
	}
	if node.ValidFrom.IsZero() {
		result.AddError(fmt.Sprintf("missing or zero ValidFrom at index %d", index), nil)
	}

	// Validate time relationships
	if !node.CreatedAt.IsZero() && !node.ValidFrom.IsZero() {
		if node.CreatedAt.After(node.ValidFrom) {
			result.AddError(fmt.Sprintf("CreatedAt (%v) is after ValidFrom (%v) at index %d", node.CreatedAt, node.ValidFrom, index), nil)
		}
	}

	if !node.ValidUntil.IsZero() && !node.ValidFrom.IsZero() {
		if node.ValidUntil.Before(node.ValidFrom) || node.ValidUntil.Equal(node.ValidFrom) {
			result.AddError(fmt.Sprintf("ValidUntil (%v) is not after ValidFrom (%v) at index %d", node.ValidUntil, node.ValidFrom, index), nil)
		}
	}
}

// validateEdge validates a single edge version.
func validateEdge(edge *Edge, index int, result *ValidationResult) {
	// Required fields validation
	if edge.ID == "" {
		result.AddError(fmt.Sprintf("missing or empty ID at index %d", index), nil)
	}
	if edge.SourceID == "" {
		result.AddError(fmt.Sprintf("missing or empty SourceID at index %d", index), nil)
	}
	if edge.TargetID == "" {
		result.AddError(fmt.Sprintf("missing or empty TargetID at index %d", index), nil)
	}
	if edge.Type == "" {
		result.AddError(fmt.Sprintf("missing or empty Type at index %d", index), nil)
	}
	if edge.Data == nil {
		result.AddError(fmt.Sprintf("missing Data field at index %d", index), nil)
	}

	// Validate source != target
	if edge.SourceID == edge.TargetID {
		result.AddError(fmt.Sprintf("SourceID and TargetID are identical (%s) at index %d", edge.SourceID, index), nil)
	}

	// Time fields validation
	if edge.CreatedAt.IsZero() {
		result.AddError(fmt.Sprintf("missing or zero CreatedAt at index %d", index), nil)
	}
	if edge.ValidFrom.IsZero() {
		result.AddError(fmt.Sprintf("missing or zero ValidFrom at index %d", index), nil)
	}

	// Validate time relationships
	if !edge.CreatedAt.IsZero() && !edge.ValidFrom.IsZero() {
		if edge.CreatedAt.After(edge.ValidFrom) {
			result.AddError(fmt.Sprintf("CreatedAt (%v) is after ValidFrom (%v) at index %d", edge.CreatedAt, edge.ValidFrom, index), nil)
		}
	}

	if !edge.ValidUntil.IsZero() && !edge.ValidFrom.IsZero() {
		if edge.ValidUntil.Before(edge.ValidFrom) || edge.ValidUntil.Equal(edge.ValidFrom) {
			result.AddError(fmt.Sprintf("ValidUntil (%v) is not after ValidFrom (%v) at index %d", edge.ValidUntil, edge.ValidFrom, index), nil)
		}
	}
}

// validateNodeTemporal validates temporal consistency for a node.
func validateNodeTemporal(node *Node, index int, result *ValidationResult) {
	// Additional temporal validation can be added here
	// For example, checking that ValidFrom is reasonable (not too far in future)
	now := time.Now()
	if node.ValidFrom.After(now.Add(24 * time.Hour)) {
		result.AddWarning(fmt.Sprintf("ValidFrom is more than 24 hours in the future at index %d", index))
	}
}

// validateEdgeTemporal validates temporal consistency for an edge.
func validateEdgeTemporal(edge *Edge, index int, result *ValidationResult) {
	// Additional temporal validation can be added here
	now := time.Now()
	if edge.ValidFrom.After(now.Add(24 * time.Hour)) {
		result.AddWarning(fmt.Sprintf("ValidFrom is more than 24 hours in the future at index %d", index))
	}
}

// ValidateDataDirectory performs a comprehensive validation of the entire data directory.
// It validates all node and edge files and returns a summary of results.
func ValidateDataDirectory(dataDir string) ([]ValidationResult, error) {
	var results []ValidationResult

	// Validate all node files
	nodesDir := filepath.Join(dataDir, "nodes")
	if _, err := os.Stat(nodesDir); err == nil {
		err := filepath.Walk(nodesDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip directories and non-JSON files
			if info.IsDir() || filepath.Ext(path) != ".json" {
				return nil
			}

			result := ValidateFile(path)
			results = append(results, *result)
			return nil
		})
		if err != nil {
			return results, fmt.Errorf("error walking nodes directory: %w", err)
		}
	}

	// Validate all edge files
	edgesDir := filepath.Join(dataDir, "edges")
	if _, err := os.Stat(edgesDir); err == nil {
		err := filepath.Walk(edgesDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip directories and non-JSON files
			if info.IsDir() || filepath.Ext(path) != ".json" {
				return nil
			}

			result := ValidateFile(path)
			results = append(results, *result)
			return nil
		})
		if err != nil {
			return results, fmt.Errorf("error walking edges directory: %w", err)
		}
	}

	return results, nil
}

// HealthCheck performs a quick validation check and returns summary statistics.
// This is designed for periodic health monitoring.
func HealthCheck(dataDir string) (map[string]interface{}, error) {
	results, err := ValidateDataDirectory(dataDir)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_files":    len(results),
		"valid_files":    0,
		"invalid_files":  0,
		"warnings_count": 0,
		"errors_count":   0,
		"node_files":     0,
		"edge_files":     0,
	}

	for _, result := range results {
		if result.Valid {
			stats["valid_files"] = stats["valid_files"].(int) + 1
		} else {
			stats["invalid_files"] = stats["invalid_files"].(int) + 1
		}

		stats["warnings_count"] = stats["warnings_count"].(int) + len(result.Warnings)
		stats["errors_count"] = stats["errors_count"].(int) + len(result.Errors)

		if result.EntityType == "node" {
			stats["node_files"] = stats["node_files"].(int) + 1
		} else if result.EntityType == "edge" {
			stats["edge_files"] = stats["edge_files"].(int) + 1
		}
	}

	return stats, nil
}