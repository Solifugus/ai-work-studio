package storage

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Edge represents a temporal relationship between two nodes in the storage system.
// Each edge maintains full version history through temporal metadata,
// supporting queries at any point in time.
type Edge struct {
	// ID uniquely identifies this edge across all versions
	ID string `json:"id"`

	// SourceID is the ID of the source node in this relationship
	SourceID string `json:"source_id"`

	// TargetID is the ID of the target node in this relationship
	TargetID string `json:"target_id"`

	// Type categorizes the edge relationship (e.g., "depends_on", "implements", "refines")
	Type string `json:"type"`

	// Data contains the edge's payload as a JSON-serializable map
	Data map[string]interface{} `json:"data"`

	// CreatedAt is when this version was created
	CreatedAt time.Time `json:"created_at"`

	// ValidFrom is when this version became active
	ValidFrom time.Time `json:"valid_from"`

	// ValidUntil is when this version was superseded.
	// Zero time (time.Time{}) indicates the current active version.
	ValidUntil time.Time `json:"valid_until"`
}

// NewEdge creates a new edge between the given source and target nodes.
// The edge is created as the current active version.
func NewEdge(sourceID, targetID, edgeType string, data map[string]interface{}) *Edge {
	now := time.Now()
	return &Edge{
		ID:         uuid.New().String(),
		SourceID:   sourceID,
		TargetID:   targetID,
		Type:       edgeType,
		Data:       data,
		CreatedAt:  now,
		ValidFrom:  now,
		ValidUntil: time.Time{}, // Zero time means current version
	}
}

// NewEdgeWithID creates a new edge with a specific ID.
// This is useful when creating new versions of existing edges.
func NewEdgeWithID(id, sourceID, targetID, edgeType string, data map[string]interface{}) *Edge {
	now := time.Now()
	return &Edge{
		ID:         id,
		SourceID:   sourceID,
		TargetID:   targetID,
		Type:       edgeType,
		Data:       data,
		CreatedAt:  now,
		ValidFrom:  now,
		ValidUntil: time.Time{},
	}
}

// IsCurrent returns true if this edge version is currently active.
// An edge is current if ValidUntil is the zero time.
func (e *Edge) IsCurrent() bool {
	return e.ValidUntil.IsZero()
}

// IsActiveAt returns true if this edge version was active at the given time.
// A version is active if the timestamp is between ValidFrom (inclusive) and ValidUntil (exclusive).
func (e *Edge) IsActiveAt(timestamp time.Time) bool {
	// Active if timestamp >= ValidFrom AND (ValidUntil is zero OR timestamp < ValidUntil)
	if timestamp.Before(e.ValidFrom) {
		return false
	}

	// If ValidUntil is zero (current version), it's active
	if e.ValidUntil.IsZero() {
		return true
	}

	// Otherwise, check if timestamp is before ValidUntil
	return timestamp.Before(e.ValidUntil)
}

// Supersede marks this edge version as superseded at the given time.
// This is called when a new version is created.
func (e *Edge) Supersede(at time.Time) {
	if !e.IsCurrent() {
		return // Already superseded
	}
	e.ValidUntil = at
}

// ConnectsNode returns true if this edge connects to the given node ID
// (either as source or target).
func (e *Edge) ConnectsNode(nodeID string) bool {
	return e.SourceID == nodeID || e.TargetID == nodeID
}

// ConnectsNodes returns true if this edge connects the two given node IDs
// in either direction.
func (e *Edge) ConnectsNodes(nodeID1, nodeID2 string) bool {
	return (e.SourceID == nodeID1 && e.TargetID == nodeID2) ||
		(e.SourceID == nodeID2 && e.TargetID == nodeID1)
}

// IsOutgoing returns true if this edge goes FROM the given node ID.
func (e *Edge) IsOutgoing(nodeID string) bool {
	return e.SourceID == nodeID
}

// IsIncoming returns true if this edge comes TO the given node ID.
func (e *Edge) IsIncoming(nodeID string) bool {
	return e.TargetID == nodeID
}

// Clone creates a deep copy of the edge.
// This is useful for creating new versions while preserving the original.
func (e *Edge) Clone() *Edge {
	// Deep copy the data map
	dataCopy := make(map[string]interface{})
	for k, v := range e.Data {
		dataCopy[k] = v
	}

	return &Edge{
		ID:         e.ID,
		SourceID:   e.SourceID,
		TargetID:   e.TargetID,
		Type:       e.Type,
		Data:       dataCopy,
		CreatedAt:  e.CreatedAt,
		ValidFrom:  e.ValidFrom,
		ValidUntil: e.ValidUntil,
	}
}

// ToJSON serializes the edge to JSON bytes.
func (e *Edge) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// EdgeFromJSON deserializes an edge from JSON bytes.
func EdgeFromJSON(data []byte) (*Edge, error) {
	var edge Edge
	err := json.Unmarshal(data, &edge)
	if err != nil {
		return nil, err
	}
	return &edge, nil
}

// EdgeHistory represents a collection of edge versions for temporal queries.
type EdgeHistory []*Edge

// GetVersionAt returns the edge version that was active at the given timestamp.
// Returns nil if no version was active at that time.
func (history EdgeHistory) GetVersionAt(timestamp time.Time) *Edge {
	for _, edge := range history {
		if edge.IsActiveAt(timestamp) {
			return edge
		}
	}
	return nil
}

// GetCurrentVersion returns the currently active version of the edge.
// Returns nil if there is no current version.
func (history EdgeHistory) GetCurrentVersion() *Edge {
	for _, edge := range history {
		if edge.IsCurrent() {
			return edge
		}
	}
	return nil
}

// GetAllVersions returns all versions sorted by ValidFrom time (oldest first).
func (history EdgeHistory) GetAllVersions() []*Edge {
	// Create a copy to avoid modifying the original slice
	versions := make([]*Edge, len(history))
	copy(versions, history)

	// Sort by ValidFrom time (bubble sort for simplicity)
	for i := 0; i < len(versions)-1; i++ {
		for j := 0; j < len(versions)-i-1; j++ {
			if versions[j].ValidFrom.After(versions[j+1].ValidFrom) {
				versions[j], versions[j+1] = versions[j+1], versions[j]
			}
		}
	}

	return versions
}

// FilterByNodes returns edges that connect to any of the given node IDs.
func (history EdgeHistory) FilterByNodes(nodeIDs ...string) EdgeHistory {
	var filtered []*Edge
	for _, edge := range history {
		for _, nodeID := range nodeIDs {
			if edge.ConnectsNode(nodeID) {
				filtered = append(filtered, edge)
				break
			}
		}
	}
	return EdgeHistory(filtered)
}

// FilterByType returns edges of the given type.
func (history EdgeHistory) FilterByType(edgeType string) EdgeHistory {
	var filtered []*Edge
	for _, edge := range history {
		if edge.Type == edgeType {
			filtered = append(filtered, edge)
		}
	}
	return EdgeHistory(filtered)
}

// FilterOutgoing returns edges that are outgoing from the given node ID.
func (history EdgeHistory) FilterOutgoing(nodeID string) EdgeHistory {
	var filtered []*Edge
	for _, edge := range history {
		if edge.IsOutgoing(nodeID) {
			filtered = append(filtered, edge)
		}
	}
	return EdgeHistory(filtered)
}

// FilterIncoming returns edges that are incoming to the given node ID.
func (history EdgeHistory) FilterIncoming(nodeID string) EdgeHistory {
	var filtered []*Edge
	for _, edge := range history {
		if edge.IsIncoming(nodeID) {
			filtered = append(filtered, edge)
		}
	}
	return EdgeHistory(filtered)
}