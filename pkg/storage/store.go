package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store provides file-based temporal storage for nodes and edges.
// It maintains in-memory indexes for fast access and persists changes to disk.
// The storage layout is:
//   data/nodes/{type}/{id}.json - Node history files
//   data/edges/{id}.json - Edge history files
type Store struct {
	// Base directory for all data files
	dataDir string

	// In-memory indexes for fast access
	nodes map[string]NodeHistory // map[nodeID]versions
	edges map[string]EdgeHistory // map[edgeID]versions

	// Concurrent access protection
	mu sync.RWMutex

	// Node type index for faster queries
	nodesByType map[string]map[string]NodeHistory // map[type]map[nodeID]versions

	// Edge type index for faster queries (only current versions)
	edgesByType map[string][]*Edge // map[type]current_edges
}

// NewStore creates a new file-based storage instance.
// It creates the necessary directory structure if it doesn't exist.
func NewStore(dataDir string) (*Store, error) {
	// Ensure directory structure exists
	nodesDir := filepath.Join(dataDir, "nodes")
	edgesDir := filepath.Join(dataDir, "edges")

	if err := os.MkdirAll(nodesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create nodes directory: %w", err)
	}
	if err := os.MkdirAll(edgesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create edges directory: %w", err)
	}

	store := &Store{
		dataDir:     dataDir,
		nodes:       make(map[string]NodeHistory),
		edges:       make(map[string]EdgeHistory),
		nodesByType: make(map[string]map[string]NodeHistory),
		edgesByType: make(map[string][]*Edge),
	}

	// Load all existing data into memory
	if err := store.loadAll(); err != nil {
		return nil, fmt.Errorf("failed to load existing data: %w", err)
	}

	return store, nil
}

// AddNode adds a new node to the store.
// If a node with this ID already exists, creates a new version.
func (s *Store) AddNode(ctx context.Context, node *Node) error {
	if node == nil {
		return fmt.Errorf("node cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if node ID already exists
	if history, exists := s.nodes[node.ID]; exists {
		// Supersede the current version
		currentVersion := history.GetCurrentVersion()
		if currentVersion != nil {
			currentVersion.Supersede(time.Now())
		}

		// Add new version
		s.nodes[node.ID] = append(history, node)
	} else {
		// Create new node history
		s.nodes[node.ID] = NodeHistory{node}
	}

	// Update type index
	if s.nodesByType[node.Type] == nil {
		s.nodesByType[node.Type] = make(map[string]NodeHistory)
	}
	s.nodesByType[node.Type][node.ID] = s.nodes[node.ID]

	// Persist to disk
	return s.saveNodeFile(node.ID)
}

// UpdateNode creates a new version of an existing node.
func (s *Store) UpdateNode(ctx context.Context, nodeID string, data map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	history, exists := s.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	currentVersion := history.GetCurrentVersion()
	if currentVersion == nil {
		return fmt.Errorf("no current version found for node %s", nodeID)
	}

	// Create new version with updated data
	newVersion := NewNodeWithID(nodeID, currentVersion.Type, data)

	// Supersede current version
	currentVersion.Supersede(time.Now())

	// Add new version
	s.nodes[nodeID] = append(history, newVersion)
	s.nodesByType[newVersion.Type][nodeID] = s.nodes[nodeID]

	// Persist to disk
	return s.saveNodeFile(nodeID)
}

// GetNode returns the current version of a node by ID.
func (s *Store) GetNode(ctx context.Context, nodeID string) (*Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history, exists := s.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node %s not found", nodeID)
	}

	current := history.GetCurrentVersion()
	if current == nil {
		return nil, fmt.Errorf("no current version found for node %s", nodeID)
	}

	return current, nil
}

// GetNodeAtTime returns the version of a node that was active at the given time.
func (s *Store) GetNodeAtTime(ctx context.Context, nodeID string, timestamp time.Time) (*Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history, exists := s.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node %s not found", nodeID)
	}

	version := history.GetVersionAt(timestamp)
	if version == nil {
		return nil, fmt.Errorf("no version found for node %s at time %v", nodeID, timestamp)
	}

	return version, nil
}

// AddEdge adds a new edge to the store.
// If an edge with this ID already exists, creates a new version.
func (s *Store) AddEdge(ctx context.Context, edge *Edge) error {
	if edge == nil {
		return fmt.Errorf("edge cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify that source and target nodes exist
	if _, exists := s.nodes[edge.SourceID]; !exists {
		return fmt.Errorf("source node %s not found", edge.SourceID)
	}
	if _, exists := s.nodes[edge.TargetID]; !exists {
		return fmt.Errorf("target node %s not found", edge.TargetID)
	}

	// Check if edge ID already exists
	if history, exists := s.edges[edge.ID]; exists {
		// Supersede the current version
		currentVersion := history.GetCurrentVersion()
		if currentVersion != nil {
			currentVersion.Supersede(time.Now())
		}

		// Add new version
		s.edges[edge.ID] = append(history, edge)
	} else {
		// Create new edge history
		s.edges[edge.ID] = EdgeHistory{edge}
	}

	// Update type index (only store current version)
	s.updateEdgeTypeIndex(edge)

	// Persist to disk
	return s.saveEdgeFile(edge.ID)
}

// UpdateEdge creates a new version of an existing edge.
func (s *Store) UpdateEdge(ctx context.Context, edgeID string, data map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	history, exists := s.edges[edgeID]
	if !exists {
		return fmt.Errorf("edge %s not found", edgeID)
	}

	currentVersion := history.GetCurrentVersion()
	if currentVersion == nil {
		return fmt.Errorf("no current version found for edge %s", edgeID)
	}

	// Create new version with updated data
	newVersion := NewEdgeWithID(edgeID, currentVersion.SourceID, currentVersion.TargetID, currentVersion.Type, data)

	// Supersede current version
	currentVersion.Supersede(time.Now())

	// Add new version
	s.edges[edgeID] = append(history, newVersion)

	// Update type index (remove old version, add new version)
	s.removeFromEdgeTypeIndex(currentVersion)
	s.updateEdgeTypeIndex(newVersion)

	// Persist to disk
	return s.saveEdgeFile(edgeID)
}

// GetEdge returns the current version of an edge by ID.
func (s *Store) GetEdge(ctx context.Context, edgeID string) (*Edge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history, exists := s.edges[edgeID]
	if !exists {
		return nil, fmt.Errorf("edge %s not found", edgeID)
	}

	current := history.GetCurrentVersion()
	if current == nil {
		return nil, fmt.Errorf("no current version found for edge %s", edgeID)
	}

	return current, nil
}

// GetEdgeAtTime returns the version of an edge that was active at the given time.
func (s *Store) GetEdgeAtTime(ctx context.Context, edgeID string, timestamp time.Time) (*Edge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history, exists := s.edges[edgeID]
	if !exists {
		return nil, fmt.Errorf("edge %s not found", edgeID)
	}

	version := history.GetVersionAt(timestamp)
	if version == nil {
		return nil, fmt.Errorf("no version found for edge %s at time %v", edgeID, timestamp)
	}

	return version, nil
}

// GetNeighbors returns all nodes connected to the given node ID through current edges.
func (s *Store) GetNeighbors(ctx context.Context, nodeID string) ([]*Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var neighbors []*Node
	neighborIDs := make(map[string]bool) // To avoid duplicates

	// Find all edges connected to this node
	for _, history := range s.edges {
		current := history.GetCurrentVersion()
		if current != nil && current.ConnectsNode(nodeID) {
			var neighborID string
			if current.SourceID == nodeID {
				neighborID = current.TargetID
			} else {
				neighborID = current.SourceID
			}

			// Avoid duplicates
			if neighborIDs[neighborID] {
				continue
			}
			neighborIDs[neighborID] = true

			// Get the neighbor node
			if neighborHistory, exists := s.nodes[neighborID]; exists {
				if neighbor := neighborHistory.GetCurrentVersion(); neighbor != nil {
					neighbors = append(neighbors, neighbor)
				}
			}
		}
	}

	return neighbors, nil
}

// GetEdgesByType returns all current edges of the given type.
func (s *Store) GetEdgesByType(ctx context.Context, edgeType string) ([]*Edge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	edges := make([]*Edge, len(s.edgesByType[edgeType]))
	copy(edges, s.edgesByType[edgeType])

	return edges, nil
}

// GetNodesByType returns all current nodes of the given type.
func (s *Store) GetNodesByType(ctx context.Context, nodeType string) ([]*Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var nodes []*Node
	if typeMap, exists := s.nodesByType[nodeType]; exists {
		for _, history := range typeMap {
			if current := history.GetCurrentVersion(); current != nil {
				nodes = append(nodes, current)
			}
		}
	}

	return nodes, nil
}

// saveNodeFile persists a node's history to disk using atomic writes.
func (s *Store) saveNodeFile(nodeID string) error {
	history, exists := s.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found in memory", nodeID)
	}

	// Get node type for directory structure
	current := history.GetCurrentVersion()
	if current == nil {
		return fmt.Errorf("no current version for node %s", nodeID)
	}

	// Ensure type directory exists
	typeDir := filepath.Join(s.dataDir, "nodes", current.Type)
	if err := os.MkdirAll(typeDir, 0755); err != nil {
		return fmt.Errorf("failed to create type directory: %w", err)
	}

	// File path
	filePath := filepath.Join(typeDir, nodeID+".json")
	tempPath := filePath + ".tmp"

	// Serialize all versions
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize node history: %w", err)
	}

	// Atomic write: write to temp file, then rename
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath) // Clean up on failure
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// saveEdgeFile persists an edge's history to disk using atomic writes.
func (s *Store) saveEdgeFile(edgeID string) error {
	history, exists := s.edges[edgeID]
	if !exists {
		return fmt.Errorf("edge %s not found in memory", edgeID)
	}

	// File path
	filePath := filepath.Join(s.dataDir, "edges", edgeID+".json")
	tempPath := filePath + ".tmp"

	// Serialize all versions
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize edge history: %w", err)
	}

	// Atomic write: write to temp file, then rename
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath) // Clean up on failure
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// loadAll loads all existing nodes and edges from disk into memory.
func (s *Store) loadAll() error {
	// Load nodes
	if err := s.loadNodes(); err != nil {
		return fmt.Errorf("failed to load nodes: %w", err)
	}

	// Load edges
	if err := s.loadEdges(); err != nil {
		return fmt.Errorf("failed to load edges: %w", err)
	}

	return nil
}

// loadNodes loads all node files from disk.
func (s *Store) loadNodes() error {
	nodesDir := filepath.Join(s.dataDir, "nodes")

	// Walk through type directories
	return filepath.Walk(nodesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-JSON files
		if info.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		// Read and parse node history file
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read node file %s: %w", path, err)
		}

		var history NodeHistory
		if err := json.Unmarshal(data, &history); err != nil {
			return fmt.Errorf("failed to unmarshal node file %s: %w", path, err)
		}

		if len(history) == 0 {
			return nil // Skip empty history
		}

		// Store in memory
		nodeID := history[0].ID
		s.nodes[nodeID] = history

		// Update type index
		if current := history.GetCurrentVersion(); current != nil {
			if s.nodesByType[current.Type] == nil {
				s.nodesByType[current.Type] = make(map[string]NodeHistory)
			}
			s.nodesByType[current.Type][nodeID] = history
		}

		return nil
	})
}

// loadEdges loads all edge files from disk.
func (s *Store) loadEdges() error {
	edgesDir := filepath.Join(s.dataDir, "edges")

	// Walk through edge files
	return filepath.Walk(edgesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-JSON files
		if info.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		// Read and parse edge history file
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read edge file %s: %w", path, err)
		}

		var history EdgeHistory
		if err := json.Unmarshal(data, &history); err != nil {
			return fmt.Errorf("failed to unmarshal edge file %s: %w", path, err)
		}

		if len(history) == 0 {
			return nil // Skip empty history
		}

		// Store in memory
		edgeID := history[0].ID
		s.edges[edgeID] = history

		// Update type index (only add current version)
		if current := history.GetCurrentVersion(); current != nil {
			s.updateEdgeTypeIndex(current)
		}

		return nil
	})
}

// updateEdgeTypeIndex adds an edge to the type index.
func (s *Store) updateEdgeTypeIndex(edge *Edge) {
	s.edgesByType[edge.Type] = append(s.edgesByType[edge.Type], edge)
}

// removeFromEdgeTypeIndex removes an edge from the type index.
func (s *Store) removeFromEdgeTypeIndex(edge *Edge) {
	edges := s.edgesByType[edge.Type]
	for i, e := range edges {
		if e.ID == edge.ID {
			// Remove this edge from the slice
			s.edgesByType[edge.Type] = append(edges[:i], edges[i+1:]...)
			break
		}
	}
}

// Close safely shuts down the store.
// Currently a no-op, but provides future hook for cleanup.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Future: Could sync any pending writes, close file handles, etc.
	return nil
}