package storage

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Node represents a temporal entity in the storage system.
// Each node maintains full version history through temporal metadata,
// supporting queries at any point in time.
type Node struct {
	// ID uniquely identifies this node across all versions
	ID string `json:"id"`

	// Type categorizes the node (e.g., "goal", "method", "objective")
	Type string `json:"type"`

	// Data contains the node's payload as a JSON-serializable map
	Data map[string]interface{} `json:"data"`

	// CreatedAt is when this version was created
	CreatedAt time.Time `json:"created_at"`

	// ValidFrom is when this version became active
	ValidFrom time.Time `json:"valid_from"`

	// ValidUntil is when this version was superseded.
	// Zero time (time.Time{}) indicates the current active version.
	ValidUntil time.Time `json:"valid_until"`
}

// NewNode creates a new node with the given type and data.
// The node is created as the current active version.
func NewNode(nodeType string, data map[string]interface{}) *Node {
	now := time.Now()
	return &Node{
		ID:         uuid.New().String(),
		Type:       nodeType,
		Data:       data,
		CreatedAt:  now,
		ValidFrom:  now,
		ValidUntil: time.Time{}, // Zero time means current version
	}
}

// NewNodeWithID creates a new node with a specific ID.
// This is useful when creating new versions of existing nodes.
func NewNodeWithID(id, nodeType string, data map[string]interface{}) *Node {
	now := time.Now()
	return &Node{
		ID:         id,
		Type:       nodeType,
		Data:       data,
		CreatedAt:  now,
		ValidFrom:  now,
		ValidUntil: time.Time{},
	}
}

// IsCurrent returns true if this node version is currently active.
// A node is current if ValidUntil is the zero time.
func (n *Node) IsCurrent() bool {
	return n.ValidUntil.IsZero()
}

// IsActiveAt returns true if this node version was active at the given time.
// A version is active if the timestamp is between ValidFrom (inclusive) and ValidUntil (exclusive).
func (n *Node) IsActiveAt(timestamp time.Time) bool {
	// Active if timestamp >= ValidFrom AND (ValidUntil is zero OR timestamp < ValidUntil)
	if timestamp.Before(n.ValidFrom) {
		return false
	}

	// If ValidUntil is zero (current version), it's active
	if n.ValidUntil.IsZero() {
		return true
	}

	// Otherwise, check if timestamp is before ValidUntil
	return timestamp.Before(n.ValidUntil)
}

// Supersede marks this node version as superseded at the given time.
// This is called when a new version is created.
func (n *Node) Supersede(at time.Time) {
	if !n.IsCurrent() {
		return // Already superseded
	}
	n.ValidUntil = at
}

// Clone creates a deep copy of the node.
// This is useful for creating new versions while preserving the original.
func (n *Node) Clone() *Node {
	// Deep copy the data map
	dataCopy := make(map[string]interface{})
	for k, v := range n.Data {
		dataCopy[k] = v
	}

	return &Node{
		ID:         n.ID,
		Type:       n.Type,
		Data:       dataCopy,
		CreatedAt:  n.CreatedAt,
		ValidFrom:  n.ValidFrom,
		ValidUntil: n.ValidUntil,
	}
}

// ToJSON serializes the node to JSON bytes.
func (n *Node) ToJSON() ([]byte, error) {
	return json.Marshal(n)
}

// FromJSON deserializes a node from JSON bytes.
func FromJSON(data []byte) (*Node, error) {
	var node Node
	err := json.Unmarshal(data, &node)
	if err != nil {
		return nil, err
	}
	return &node, nil
}

// NodeHistory represents a collection of node versions for temporal queries.
type NodeHistory []*Node

// GetVersionAt returns the node version that was active at the given timestamp.
// Returns nil if no version was active at that time.
func (history NodeHistory) GetVersionAt(timestamp time.Time) *Node {
	for _, node := range history {
		if node.IsActiveAt(timestamp) {
			return node
		}
	}
	return nil
}

// GetCurrentVersion returns the currently active version of the node.
// Returns nil if there is no current version.
func (history NodeHistory) GetCurrentVersion() *Node {
	for _, node := range history {
		if node.IsCurrent() {
			return node
		}
	}
	return nil
}

// GetAllVersions returns all versions sorted by ValidFrom time (oldest first).
func (history NodeHistory) GetAllVersions() []*Node {
	// Create a copy to avoid modifying the original slice
	versions := make([]*Node, len(history))
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