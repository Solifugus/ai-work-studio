package storage

import (
	"context"
	"time"
)

// NodeQuery provides a fluent interface for querying nodes.
// It uses the builder pattern to construct filters and executes them lazily.
type NodeQuery struct {
	store     *Store
	filters   []NodeFilter
	timeQuery *TimeQuery
}

// EdgeQuery provides a fluent interface for querying edges.
// It uses the builder pattern to construct filters and executes them lazily.
type EdgeQuery struct {
	store     *Store
	filters   []EdgeFilter
	timeQuery *TimeQuery
}

// NodeFilter is a function that filters nodes based on criteria.
type NodeFilter func(*Node) bool

// EdgeFilter is a function that filters edges based on criteria.
type EdgeFilter func(*Edge) bool

// TimeQuery holds temporal query parameters.
type TimeQuery struct {
	asOf      *time.Time
	rangeFrom *time.Time
	rangeTo   *time.Time
}

// Nodes returns a new NodeQuery for fluent querying.
func (s *Store) Nodes() *NodeQuery {
	return &NodeQuery{
		store:   s,
		filters: make([]NodeFilter, 0),
	}
}

// Edges returns a new EdgeQuery for fluent querying.
func (s *Store) Edges() *EdgeQuery {
	return &EdgeQuery{
		store:   s,
		filters: make([]EdgeFilter, 0),
	}
}

// OfType filters nodes by type.
func (nq *NodeQuery) OfType(nodeType string) *NodeQuery {
	// Create a new query to avoid modifying the original
	newFilters := make([]NodeFilter, len(nq.filters), len(nq.filters)+1)
	copy(newFilters, nq.filters)
	newFilters = append(newFilters, func(n *Node) bool {
		return n.Type == nodeType
	})

	return &NodeQuery{
		store:     nq.store,
		filters:   newFilters,
		timeQuery: nq.timeQuery, // Shallow copy is OK for timeQuery
	}
}

// WithData filters nodes that have specific data values.
// The dataKey is the key in the node's Data map, and expectedValue is the expected value.
func (nq *NodeQuery) WithData(dataKey string, expectedValue interface{}) *NodeQuery {
	// Create a new query to avoid modifying the original
	newFilters := make([]NodeFilter, len(nq.filters), len(nq.filters)+1)
	copy(newFilters, nq.filters)
	newFilters = append(newFilters, func(n *Node) bool {
		if n.Data == nil {
			return false
		}
		value, exists := n.Data[dataKey]
		if !exists {
			return false
		}
		return value == expectedValue
	})

	return &NodeQuery{
		store:     nq.store,
		filters:   newFilters,
		timeQuery: nq.timeQuery,
	}
}

// WithID filters nodes by specific ID.
func (nq *NodeQuery) WithID(nodeID string) *NodeQuery {
	// Create a new query to avoid modifying the original
	newFilters := make([]NodeFilter, len(nq.filters), len(nq.filters)+1)
	copy(newFilters, nq.filters)
	newFilters = append(newFilters, func(n *Node) bool {
		return n.ID == nodeID
	})

	return &NodeQuery{
		store:     nq.store,
		filters:   newFilters,
		timeQuery: nq.timeQuery,
	}
}

// AsOf sets the temporal query to a specific timestamp.
// Returns nodes that were active at that time.
func (nq *NodeQuery) AsOf(timestamp time.Time) *NodeQuery {
	// Create a new query to avoid modifying the original
	newFilters := make([]NodeFilter, len(nq.filters))
	copy(newFilters, nq.filters)

	newTimeQuery := &TimeQuery{
		asOf: &timestamp,
	}

	return &NodeQuery{
		store:     nq.store,
		filters:   newFilters,
		timeQuery: newTimeQuery,
	}
}

// Between sets the temporal query to a time range.
// Returns nodes that were active during any part of the range.
func (nq *NodeQuery) Between(start, end time.Time) *NodeQuery {
	// Create a new query to avoid modifying the original
	newFilters := make([]NodeFilter, len(nq.filters))
	copy(newFilters, nq.filters)

	newTimeQuery := &TimeQuery{
		rangeFrom: &start,
		rangeTo:   &end,
	}

	return &NodeQuery{
		store:     nq.store,
		filters:   newFilters,
		timeQuery: newTimeQuery,
	}
}

// Neighbors finds nodes connected to the current query results through edges of the given type.
// This performs relationship traversal.
func (nq *NodeQuery) Neighbors(edgeType string) *NodeQuery {
	// Create a new query to avoid modifying the original
	newFilters := make([]NodeFilter, len(nq.filters), len(nq.filters)+2)
	copy(newFilters, nq.filters)

	// This is a more complex filter that requires executing the current query first
	// and then finding neighbors of the results
	newFilters = append(newFilters, func(n *Node) bool {
		// This will be handled specially in the execution logic
		return true // Placeholder - actual logic in All() method
	})
	// Mark this query as needing neighbor traversal
	newFilters = append(newFilters, func(n *Node) bool {
		// Special marker filter for neighbor traversal
		return n.Data != nil && n.Data["__neighbor_traversal__"] == edgeType
	})

	return &NodeQuery{
		store:     nq.store,
		filters:   newFilters,
		timeQuery: nq.timeQuery,
	}
}

// All executes the query and returns all matching nodes.
func (nq *NodeQuery) All() ([]*Node, error) {
	nq.store.mu.RLock()
	defer nq.store.mu.RUnlock()

	var results []*Node

	// Check if this is a neighbor traversal query (special case)
	if nq.hasNeighborTraversal() {
		return nq.executeNeighborTraversal()
	}

	// Handle temporal queries
	if nq.timeQuery != nil && nq.timeQuery.asOf != nil {
		return nq.executeAsOfQuery()
	}
	if nq.timeQuery != nil && nq.timeQuery.rangeFrom != nil && nq.timeQuery.rangeTo != nil {
		return nq.executeBetweenQuery()
	}

	// Regular query - iterate through all nodes
	for _, history := range nq.store.nodes {
		node := history.GetCurrentVersion()
		if node != nil && nq.matchesAllFilters(node) {
			results = append(results, node)
		}
	}

	return results, nil
}

// First executes the query and returns the first matching node, or nil if none found.
func (nq *NodeQuery) First() (*Node, error) {
	results, err := nq.All()
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results[0], nil
}

// Count executes the query and returns the number of matching nodes.
func (nq *NodeQuery) Count() (int, error) {
	results, err := nq.All()
	if err != nil {
		return 0, err
	}
	return len(results), nil
}

// OfType filters edges by type.
func (eq *EdgeQuery) OfType(edgeType string) *EdgeQuery {
	// Create a new query to avoid modifying the original
	newFilters := make([]EdgeFilter, len(eq.filters), len(eq.filters)+1)
	copy(newFilters, eq.filters)
	newFilters = append(newFilters, func(e *Edge) bool {
		return e.Type == edgeType
	})

	return &EdgeQuery{
		store:     eq.store,
		filters:   newFilters,
		timeQuery: eq.timeQuery,
	}
}

// ConnectingNode filters edges that connect to the given node (either as source or target).
func (eq *EdgeQuery) ConnectingNode(nodeID string) *EdgeQuery {
	// Create a new query to avoid modifying the original
	newFilters := make([]EdgeFilter, len(eq.filters), len(eq.filters)+1)
	copy(newFilters, eq.filters)
	newFilters = append(newFilters, func(e *Edge) bool {
		return e.ConnectsNode(nodeID)
	})

	return &EdgeQuery{
		store:     eq.store,
		filters:   newFilters,
		timeQuery: eq.timeQuery,
	}
}

// FromNode filters edges that originate from the given node.
func (eq *EdgeQuery) FromNode(nodeID string) *EdgeQuery {
	// Create a new query to avoid modifying the original
	newFilters := make([]EdgeFilter, len(eq.filters), len(eq.filters)+1)
	copy(newFilters, eq.filters)
	newFilters = append(newFilters, func(e *Edge) bool {
		return e.SourceID == nodeID
	})

	return &EdgeQuery{
		store:     eq.store,
		filters:   newFilters,
		timeQuery: eq.timeQuery,
	}
}

// ToNode filters edges that target the given node.
func (eq *EdgeQuery) ToNode(nodeID string) *EdgeQuery {
	// Create a new query to avoid modifying the original
	newFilters := make([]EdgeFilter, len(eq.filters), len(eq.filters)+1)
	copy(newFilters, eq.filters)
	newFilters = append(newFilters, func(e *Edge) bool {
		return e.TargetID == nodeID
	})

	return &EdgeQuery{
		store:     eq.store,
		filters:   newFilters,
		timeQuery: eq.timeQuery,
	}
}

// WithData filters edges that have specific data values.
func (eq *EdgeQuery) WithData(dataKey string, expectedValue interface{}) *EdgeQuery {
	// Create a new query to avoid modifying the original
	newFilters := make([]EdgeFilter, len(eq.filters), len(eq.filters)+1)
	copy(newFilters, eq.filters)
	newFilters = append(newFilters, func(e *Edge) bool {
		if e.Data == nil {
			return false
		}
		value, exists := e.Data[dataKey]
		if !exists {
			return false
		}
		return value == expectedValue
	})

	return &EdgeQuery{
		store:     eq.store,
		filters:   newFilters,
		timeQuery: eq.timeQuery,
	}
}

// AsOf sets the temporal query to a specific timestamp.
func (eq *EdgeQuery) AsOf(timestamp time.Time) *EdgeQuery {
	// Create a new query to avoid modifying the original
	newFilters := make([]EdgeFilter, len(eq.filters))
	copy(newFilters, eq.filters)

	newTimeQuery := &TimeQuery{
		asOf: &timestamp,
	}

	return &EdgeQuery{
		store:     eq.store,
		filters:   newFilters,
		timeQuery: newTimeQuery,
	}
}

// Between sets the temporal query to a time range.
func (eq *EdgeQuery) Between(start, end time.Time) *EdgeQuery {
	// Create a new query to avoid modifying the original
	newFilters := make([]EdgeFilter, len(eq.filters))
	copy(newFilters, eq.filters)

	newTimeQuery := &TimeQuery{
		rangeFrom: &start,
		rangeTo:   &end,
	}

	return &EdgeQuery{
		store:     eq.store,
		filters:   newFilters,
		timeQuery: newTimeQuery,
	}
}

// All executes the query and returns all matching edges.
func (eq *EdgeQuery) All() ([]*Edge, error) {
	eq.store.mu.RLock()
	defer eq.store.mu.RUnlock()

	var results []*Edge

	// Handle temporal queries
	if eq.timeQuery != nil && eq.timeQuery.asOf != nil {
		return eq.executeAsOfQuery()
	}
	if eq.timeQuery != nil && eq.timeQuery.rangeFrom != nil && eq.timeQuery.rangeTo != nil {
		return eq.executeBetweenQuery()
	}

	// Regular query - iterate through all edges
	for _, history := range eq.store.edges {
		edge := history.GetCurrentVersion()
		if edge != nil && eq.matchesAllFilters(edge) {
			results = append(results, edge)
		}
	}

	return results, nil
}

// First executes the query and returns the first matching edge, or nil if none found.
func (eq *EdgeQuery) First() (*Edge, error) {
	results, err := eq.All()
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results[0], nil
}

// Count executes the query and returns the number of matching edges.
func (eq *EdgeQuery) Count() (int, error) {
	results, err := eq.All()
	if err != nil {
		return 0, err
	}
	return len(results), nil
}

// Helper methods for NodeQuery

// matchesAllFilters checks if a node matches all the applied filters.
func (nq *NodeQuery) matchesAllFilters(node *Node) bool {
	for _, filter := range nq.filters {
		// Skip special neighbor traversal markers
		if nq.isNeighborTraversalFilter(filter) {
			continue
		}
		if !filter(node) {
			return false
		}
	}
	return true
}

// hasNeighborTraversal checks if this query includes neighbor traversal.
func (nq *NodeQuery) hasNeighborTraversal() bool {
	for _, filter := range nq.filters {
		if nq.isNeighborTraversalFilter(filter) {
			return true
		}
	}
	return false
}

// isNeighborTraversalFilter identifies neighbor traversal marker filters.
func (nq *NodeQuery) isNeighborTraversalFilter(filter NodeFilter) bool {
	// Test with a dummy node that has the marker
	testNode := &Node{
		Data: map[string]interface{}{
			"__neighbor_traversal__": "test",
		},
	}
	return filter(testNode)
}

// executeNeighborTraversal performs relationship traversal.
func (nq *NodeQuery) executeNeighborTraversal() ([]*Node, error) {
	// First, execute the query without neighbor traversal to get base nodes
	baseQuery := &NodeQuery{
		store:     nq.store,
		filters:   make([]NodeFilter, 0),
		timeQuery: nq.timeQuery,
	}

	// Add all non-neighbor-traversal filters
	for _, filter := range nq.filters {
		if !nq.isNeighborTraversalFilter(filter) {
			baseQuery.filters = append(baseQuery.filters, filter)
		}
	}

	baseNodes, err := baseQuery.All()
	if err != nil {
		return nil, err
	}

	// Extract edge type from the neighbor traversal filter
	edgeType := ""
	for _, filter := range nq.filters {
		if nq.isNeighborTraversalFilter(filter) {
			// Find the edge type from the marker
			testNode := &Node{Data: map[string]interface{}{}}
			for testType := range nq.store.edgesByType {
				testNode.Data["__neighbor_traversal__"] = testType
				if filter(testNode) {
					edgeType = testType
					break
				}
			}
			break
		}
	}

	// Find all neighbors of the base nodes through edges of the specified type
	neighborMap := make(map[string]*Node) // Use map to avoid duplicates
	for _, baseNode := range baseNodes {
		neighbors, err := nq.store.GetNeighbors(context.Background(), baseNode.ID)
		if err != nil {
			continue
		}

		// Filter neighbors by edge type
		for _, neighbor := range neighbors {
			// Check if there's an edge of the specified type between baseNode and neighbor
			if nq.hasEdgeOfType(baseNode.ID, neighbor.ID, edgeType) {
				neighborMap[neighbor.ID] = neighbor
			}
		}
	}

	// Convert map to slice
	var result []*Node
	for _, neighbor := range neighborMap {
		result = append(result, neighbor)
	}

	return result, nil
}

// hasEdgeOfType checks if there's an edge of the given type between two nodes.
func (nq *NodeQuery) hasEdgeOfType(nodeID1, nodeID2, edgeType string) bool {
	for _, history := range nq.store.edges {
		edge := history.GetCurrentVersion()
		if edge != nil && edge.Type == edgeType && edge.ConnectsNodes(nodeID1, nodeID2) {
			return true
		}
	}
	return false
}

// executeAsOfQuery executes a temporal query for a specific timestamp.
func (nq *NodeQuery) executeAsOfQuery() ([]*Node, error) {
	var results []*Node
	timestamp := *nq.timeQuery.asOf

	for _, history := range nq.store.nodes {
		node := history.GetVersionAt(timestamp)
		if node != nil && nq.matchesAllFilters(node) {
			results = append(results, node)
		}
	}

	return results, nil
}

// executeBetweenQuery executes a temporal query for a time range.
func (nq *NodeQuery) executeBetweenQuery() ([]*Node, error) {
	var results []*Node
	start := *nq.timeQuery.rangeFrom
	end := *nq.timeQuery.rangeTo

	for _, history := range nq.store.nodes {
		// Check if any version of this node was active during the range
		found := false
		for _, version := range history {
			if nq.isActiveInRange(version, start, end) && nq.matchesAllFilters(version) {
				results = append(results, version)
				found = true
				break // Only include one version per node
			}
		}
		_ = found // Silence unused variable warning
	}

	return results, nil
}

// isActiveInRange checks if a node version was active during any part of the time range.
func (nq *NodeQuery) isActiveInRange(node *Node, start, end time.Time) bool {
	// Node is active in range if its valid period overlaps with [start, end]
	nodeStart := node.ValidFrom
	nodeEnd := node.ValidUntil
	if nodeEnd.IsZero() {
		nodeEnd = time.Now() // Current version is active until now
	}

	// Check for overlap: nodeStart < end AND start < nodeEnd
	return nodeStart.Before(end) && start.Before(nodeEnd)
}

// Helper methods for EdgeQuery

// matchesAllFilters checks if an edge matches all the applied filters.
func (eq *EdgeQuery) matchesAllFilters(edge *Edge) bool {
	for _, filter := range eq.filters {
		if !filter(edge) {
			return false
		}
	}
	return true
}

// executeAsOfQuery executes a temporal query for a specific timestamp.
func (eq *EdgeQuery) executeAsOfQuery() ([]*Edge, error) {
	var results []*Edge
	timestamp := *eq.timeQuery.asOf

	for _, history := range eq.store.edges {
		edge := history.GetVersionAt(timestamp)
		if edge != nil && eq.matchesAllFilters(edge) {
			results = append(results, edge)
		}
	}

	return results, nil
}

// executeBetweenQuery executes a temporal query for a time range.
func (eq *EdgeQuery) executeBetweenQuery() ([]*Edge, error) {
	var results []*Edge
	start := *eq.timeQuery.rangeFrom
	end := *eq.timeQuery.rangeTo

	for _, history := range eq.store.edges {
		// Check if any version of this edge was active during the range
		found := false
		for _, version := range history {
			if eq.isActiveInRange(version, start, end) && eq.matchesAllFilters(version) {
				results = append(results, version)
				found = true
				break // Only include one version per edge
			}
		}
		_ = found // Silence unused variable warning
	}

	return results, nil
}

// isActiveInRange checks if an edge version was active during any part of the time range.
func (eq *EdgeQuery) isActiveInRange(edge *Edge, start, end time.Time) bool {
	// Edge is active in range if its valid period overlaps with [start, end]
	edgeStart := edge.ValidFrom
	edgeEnd := edge.ValidUntil
	if edgeEnd.IsZero() {
		edgeEnd = time.Now() // Current version is active until now
	}

	// Check for overlap: edgeStart < end AND start < edgeEnd
	return edgeStart.Before(end) && start.Before(edgeEnd)
}