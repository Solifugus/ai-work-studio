package storage

import (
	"context"
	"testing"
	"time"
)

// setupTestStore creates a test store with sample data
func setupTestStore(t *testing.T) *Store {
	// Create temporary directory for test data
	tempDir := t.TempDir()

	store, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}

	// Create test nodes
	goal1 := NewNode("Goal", map[string]interface{}{
		"title":    "Learn Go",
		"priority": "high",
	})

	goal2 := NewNode("Goal", map[string]interface{}{
		"title":    "Build AI System",
		"priority": "medium",
	})

	method1 := NewNode("Method", map[string]interface{}{
		"name":        "Study documentation",
		"effectiveness": 0.8,
	})

	method2 := NewNode("Method", map[string]interface{}{
		"name":        "Build projects",
		"effectiveness": 0.9,
	})

	objective1 := NewNode("Objective", map[string]interface{}{
		"description": "Read Go tour",
		"status":      "pending",
	})

	// Add nodes to store
	ctx := context.Background()
	if err := store.AddNode(ctx, goal1); err != nil {
		t.Fatalf("Failed to add goal1: %v", err)
	}
	if err := store.AddNode(ctx, goal2); err != nil {
		t.Fatalf("Failed to add goal2: %v", err)
	}
	if err := store.AddNode(ctx, method1); err != nil {
		t.Fatalf("Failed to add method1: %v", err)
	}
	if err := store.AddNode(ctx, method2); err != nil {
		t.Fatalf("Failed to add method2: %v", err)
	}
	if err := store.AddNode(ctx, objective1); err != nil {
		t.Fatalf("Failed to add objective1: %v", err)
	}

	// Create test edges
	edge1 := NewEdge(goal1.ID, method1.ID, "uses", map[string]interface{}{
		"confidence": 0.7,
	})

	edge2 := NewEdge(goal1.ID, method2.ID, "uses", map[string]interface{}{
		"confidence": 0.9,
	})

	edge3 := NewEdge(method1.ID, objective1.ID, "implements", map[string]interface{}{
		"order": 1,
	})

	edge4 := NewEdge(goal2.ID, method2.ID, "depends_on", map[string]interface{}{
		"reason": "foundational",
	})

	// Add edges to store
	if err := store.AddEdge(ctx, edge1); err != nil {
		t.Fatalf("Failed to add edge1: %v", err)
	}
	if err := store.AddEdge(ctx, edge2); err != nil {
		t.Fatalf("Failed to add edge2: %v", err)
	}
	if err := store.AddEdge(ctx, edge3); err != nil {
		t.Fatalf("Failed to add edge3: %v", err)
	}
	if err := store.AddEdge(ctx, edge4); err != nil {
		t.Fatalf("Failed to add edge4: %v", err)
	}

	return store
}

func TestNodeQuery_OfType(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	// Test querying nodes by type
	goals, err := store.Nodes().OfType("Goal").All()
	if err != nil {
		t.Fatalf("Failed to query goals: %v", err)
	}

	if len(goals) != 2 {
		t.Errorf("Expected 2 goals, got %d", len(goals))
	}

	for _, goal := range goals {
		if goal.Type != "Goal" {
			t.Errorf("Expected type 'Goal', got '%s'", goal.Type)
		}
	}

	// Test querying methods
	methods, err := store.Nodes().OfType("Method").All()
	if err != nil {
		t.Fatalf("Failed to query methods: %v", err)
	}

	if len(methods) != 2 {
		t.Errorf("Expected 2 methods, got %d", len(methods))
	}

	// Test querying non-existent type
	nonExistent, err := store.Nodes().OfType("NonExistent").All()
	if err != nil {
		t.Fatalf("Failed to query non-existent type: %v", err)
	}

	if len(nonExistent) != 0 {
		t.Errorf("Expected 0 nodes for non-existent type, got %d", len(nonExistent))
	}
}

func TestNodeQuery_WithData(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	// Test querying nodes with specific data values
	highPriority, err := store.Nodes().WithData("priority", "high").All()
	if err != nil {
		t.Fatalf("Failed to query high priority nodes: %v", err)
	}

	if len(highPriority) != 1 {
		t.Errorf("Expected 1 high priority node, got %d", len(highPriority))
	}

	if highPriority[0].Data["title"] != "Learn Go" {
		t.Errorf("Expected 'Learn Go', got '%v'", highPriority[0].Data["title"])
	}

	// Test querying with non-existent data key
	nonExistent, err := store.Nodes().WithData("nonexistent", "value").All()
	if err != nil {
		t.Fatalf("Failed to query with non-existent key: %v", err)
	}

	if len(nonExistent) != 0 {
		t.Errorf("Expected 0 nodes for non-existent key, got %d", len(nonExistent))
	}
}

func TestNodeQuery_Chaining(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	// Test chaining multiple filters
	result, err := store.Nodes().OfType("Goal").WithData("priority", "high").All()
	if err != nil {
		t.Fatalf("Failed to execute chained query: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 result from chained query, got %d", len(result))
	}

	if result[0].Data["title"] != "Learn Go" {
		t.Errorf("Expected 'Learn Go', got '%v'", result[0].Data["title"])
	}

	// Test chaining that results in no matches
	noMatch, err := store.Nodes().OfType("Goal").WithData("priority", "low").All()
	if err != nil {
		t.Fatalf("Failed to execute chained query with no matches: %v", err)
	}

	if len(noMatch) != 0 {
		t.Errorf("Expected 0 results from chained query, got %d", len(noMatch))
	}
}

func TestNodeQuery_First(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	// Test First() method
	firstGoal, err := store.Nodes().OfType("Goal").First()
	if err != nil {
		t.Fatalf("Failed to get first goal: %v", err)
	}

	if firstGoal == nil {
		t.Fatalf("Expected first goal to be non-nil")
	}

	if firstGoal.Type != "Goal" {
		t.Errorf("Expected type 'Goal', got '%s'", firstGoal.Type)
	}

	// Test First() with no matches
	noMatch, err := store.Nodes().OfType("NonExistent").First()
	if err != nil {
		t.Fatalf("Failed to execute First() with no matches: %v", err)
	}

	if noMatch != nil {
		t.Errorf("Expected nil for no matches, got %v", noMatch)
	}
}

func TestNodeQuery_Count(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	// Test Count() method
	goalCount, err := store.Nodes().OfType("Goal").Count()
	if err != nil {
		t.Fatalf("Failed to count goals: %v", err)
	}

	if goalCount != 2 {
		t.Errorf("Expected 2 goals, got %d", goalCount)
	}

	// Test Count() with no matches
	noMatchCount, err := store.Nodes().OfType("NonExistent").Count()
	if err != nil {
		t.Fatalf("Failed to count non-existent type: %v", err)
	}

	if noMatchCount != 0 {
		t.Errorf("Expected 0 count for non-existent type, got %d", noMatchCount)
	}
}

func TestEdgeQuery_OfType(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	// Test querying edges by type
	usesEdges, err := store.Edges().OfType("uses").All()
	if err != nil {
		t.Fatalf("Failed to query uses edges: %v", err)
	}

	if len(usesEdges) != 2 {
		t.Errorf("Expected 2 uses edges, got %d", len(usesEdges))
	}

	for _, edge := range usesEdges {
		if edge.Type != "uses" {
			t.Errorf("Expected type 'uses', got '%s'", edge.Type)
		}
	}

	// Test querying implements edges
	implementsEdges, err := store.Edges().OfType("implements").All()
	if err != nil {
		t.Fatalf("Failed to query implements edges: %v", err)
	}

	if len(implementsEdges) != 1 {
		t.Errorf("Expected 1 implements edge, got %d", len(implementsEdges))
	}
}

func TestEdgeQuery_ConnectingNode(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	// Get a node to test with
	goal, err := store.Nodes().OfType("Goal").First()
	if err != nil || goal == nil {
		t.Fatalf("Failed to get test goal")
	}

	// Test querying edges connecting to a specific node
	connectingEdges, err := store.Edges().ConnectingNode(goal.ID).All()
	if err != nil {
		t.Fatalf("Failed to query connecting edges: %v", err)
	}

	// Should find edges where goal is either source or target
	if len(connectingEdges) == 0 {
		t.Errorf("Expected at least 1 connecting edge, got 0")
	}

	for _, edge := range connectingEdges {
		if !edge.ConnectsNode(goal.ID) {
			t.Errorf("Edge should connect to goal %s", goal.ID)
		}
	}
}

func TestEdgeQuery_FromAndToNode(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	// Get a goal node
	goal, err := store.Nodes().OfType("Goal").First()
	if err != nil || goal == nil {
		t.Fatalf("Failed to get test goal")
	}

	// Test FromNode query
	outgoingEdges, err := store.Edges().FromNode(goal.ID).All()
	if err != nil {
		t.Fatalf("Failed to query outgoing edges: %v", err)
	}

	for _, edge := range outgoingEdges {
		if edge.SourceID != goal.ID {
			t.Errorf("Expected source ID %s, got %s", goal.ID, edge.SourceID)
		}
	}

	// Get a method node for incoming edge test
	method, err := store.Nodes().OfType("Method").First()
	if err != nil || method == nil {
		t.Fatalf("Failed to get test method")
	}

	// Test ToNode query
	incomingEdges, err := store.Edges().ToNode(method.ID).All()
	if err != nil {
		t.Fatalf("Failed to query incoming edges: %v", err)
	}

	for _, edge := range incomingEdges {
		if edge.TargetID != method.ID {
			t.Errorf("Expected target ID %s, got %s", method.ID, edge.TargetID)
		}
	}
}

func TestEdgeQuery_WithData(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	// Test querying edges with specific data values
	highConfidence, err := store.Edges().WithData("confidence", 0.9).All()
	if err != nil {
		t.Fatalf("Failed to query high confidence edges: %v", err)
	}

	if len(highConfidence) != 1 {
		t.Errorf("Expected 1 high confidence edge, got %d", len(highConfidence))
	}

	if highConfidence[0].Data["confidence"] != 0.9 {
		t.Errorf("Expected confidence 0.9, got %v", highConfidence[0].Data["confidence"])
	}
}

func TestTemporalQueries_AsOf(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()

	// Get a node to test temporal queries with
	goal, err := store.Nodes().OfType("Goal").First()
	if err != nil || goal == nil {
		t.Fatalf("Failed to get test goal")
	}

	// Record the time before update
	beforeUpdate := time.Now()
	time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps

	// Update the node to create a new version
	err = store.UpdateNode(ctx, goal.ID, map[string]interface{}{
		"title":    goal.Data["title"],
		"priority": "updated",
		"version":  2,
	})
	if err != nil {
		t.Fatalf("Failed to update node: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	afterUpdate := time.Now()

	// Query for the node as it was before the update
	oldVersion, err := store.Nodes().WithID(goal.ID).AsOf(beforeUpdate).First()
	if err != nil {
		t.Fatalf("Failed to query node AsOf before update: %v", err)
	}

	if oldVersion == nil {
		t.Fatalf("Expected old version to be found")
	}

	if oldVersion.Data["priority"] == "updated" {
		t.Errorf("Old version should not have updated priority")
	}

	// Query for the node as it is after the update
	newVersion, err := store.Nodes().WithID(goal.ID).AsOf(afterUpdate).First()
	if err != nil {
		t.Fatalf("Failed to query node AsOf after update: %v", err)
	}

	if newVersion == nil {
		t.Fatalf("Expected new version to be found")
	}

	if newVersion.Data["priority"] != "updated" {
		t.Errorf("New version should have updated priority")
	}
}

func TestTemporalQueries_Between(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()

	// Get a node to test with
	goal, err := store.Nodes().OfType("Goal").First()
	if err != nil || goal == nil {
		t.Fatalf("Failed to get test goal")
	}

	// Record time points
	beforeAll := time.Now().Add(-1 * time.Hour)
	beforeUpdate := time.Now()
	time.Sleep(10 * time.Millisecond)

	// Update the node
	err = store.UpdateNode(ctx, goal.ID, map[string]interface{}{
		"title":    goal.Data["title"],
		"priority": "updated",
	})
	if err != nil {
		t.Fatalf("Failed to update node: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	afterUpdate := time.Now()

	// Query for nodes between before and after the update
	betweenResults, err := store.Nodes().WithID(goal.ID).Between(beforeUpdate, afterUpdate).All()
	if err != nil {
		t.Fatalf("Failed to query nodes Between: %v", err)
	}

	// Should find at least one version (the one that was active during this period)
	if len(betweenResults) == 0 {
		t.Errorf("Expected at least 1 result from Between query, got 0")
	}

	// Query for a time period before the node existed
	veryOld := beforeAll.Add(-1 * time.Hour)
	oldResults, err := store.Nodes().WithID(goal.ID).Between(veryOld, beforeAll).All()
	if err != nil {
		t.Fatalf("Failed to query nodes in old time period: %v", err)
	}

	// Should find no results in this old time period
	if len(oldResults) != 0 {
		t.Errorf("Expected 0 results from old time period, got %d", len(oldResults))
	}
}

func TestEdgeTemporalQueries(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()

	// Get an edge to test temporal queries with
	edge, err := store.Edges().OfType("uses").First()
	if err != nil || edge == nil {
		t.Fatalf("Failed to get test edge")
	}

	// Record the time before update
	beforeUpdate := time.Now()
	time.Sleep(10 * time.Millisecond)

	// Update the edge to create a new version
	err = store.UpdateEdge(ctx, edge.ID, map[string]interface{}{
		"confidence": 0.95,
		"updated":    true,
	})
	if err != nil {
		t.Fatalf("Failed to update edge: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	afterUpdate := time.Now()

	// Test AsOf query for edges
	oldEdge, err := store.Edges().OfType("uses").AsOf(beforeUpdate).First()
	if err != nil {
		t.Fatalf("Failed to query edge AsOf: %v", err)
	}

	if oldEdge == nil {
		t.Fatalf("Expected old edge to be found")
	}

	// The old version should not have the "updated" field
	if oldEdge.Data["updated"] == true {
		t.Errorf("Old edge version should not have 'updated' field")
	}

	// Test Between query for edges
	betweenEdges, err := store.Edges().OfType("uses").Between(beforeUpdate, afterUpdate).All()
	if err != nil {
		t.Fatalf("Failed to query edges Between: %v", err)
	}

	// Should find at least the edge versions that were active during this period
	if len(betweenEdges) == 0 {
		t.Errorf("Expected at least 1 edge result from Between query, got 0")
	}
}

// Note: Testing the Neighbors method requires a more complex setup
// since the current implementation has some issues with the neighbor traversal logic.
// This test demonstrates the intended usage but may need refinement.
func TestNodeQuery_NeighborsBasic(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	// Get a goal node that should have neighbors through "uses" edges
	goal, err := store.Nodes().OfType("Goal").First()
	if err != nil || goal == nil {
		t.Fatalf("Failed to get test goal")
	}

	// Test basic neighbor functionality through the store's GetNeighbors method
	neighbors, err := store.GetNeighbors(context.Background(), goal.ID)
	if err != nil {
		t.Fatalf("Failed to get neighbors: %v", err)
	}

	if len(neighbors) == 0 {
		t.Errorf("Expected goal to have neighbors, got 0")
	}

	// Verify that the neighbors are connected to the goal through edges
	for _, neighbor := range neighbors {
		connected := false
		edges, err := store.Edges().ConnectingNode(goal.ID).All()
		if err != nil {
			t.Fatalf("Failed to get connecting edges: %v", err)
		}

		for _, edge := range edges {
			if edge.ConnectsNodes(goal.ID, neighbor.ID) {
				connected = true
				break
			}
		}

		if !connected {
			t.Errorf("Neighbor %s should be connected to goal %s", neighbor.ID, goal.ID)
		}
	}
}

func TestQueryBuilderPatterns(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	// Test that queries can be built step by step
	baseQuery := store.Nodes().OfType("Goal")

	// Should be able to continue building the query
	refinedQuery := baseQuery.WithData("priority", "high")

	// Executing the query should work
	results, err := refinedQuery.All()
	if err != nil {
		t.Fatalf("Failed to execute refined query: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result from refined query, got %d", len(results))
	}

	// Test that the original query object wasn't modified
	allGoals, err := baseQuery.All()
	if err != nil {
		t.Fatalf("Failed to execute base query: %v", err)
	}

	if len(allGoals) != 2 {
		t.Errorf("Base query should still return 2 goals, got %d", len(allGoals))
	}
}

// Benchmark tests
func BenchmarkNodeQuery_OfType(b *testing.B) {
	store := setupBenchmarkStore(b)
	defer store.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.Nodes().OfType("Goal").All()
		if err != nil {
			b.Fatalf("Query failed: %v", err)
		}
	}
}

func BenchmarkEdgeQuery_ConnectingNode(b *testing.B) {
	store := setupBenchmarkStore(b)
	defer store.Close()

	// Get a node ID to test with
	goal, err := store.Nodes().OfType("Goal").First()
	if err != nil || goal == nil {
		b.Fatalf("Failed to get test goal")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.Edges().ConnectingNode(goal.ID).All()
		if err != nil {
			b.Fatalf("Query failed: %v", err)
		}
	}
}

// setupBenchmarkStore creates a store with more data for benchmarking
func setupBenchmarkStore(b *testing.B) *Store {
	tempDir := b.TempDir()

	store, err := NewStore(tempDir)
	if err != nil {
		b.Fatalf("Failed to create benchmark store: %v", err)
	}

	ctx := context.Background()

	// Create more test data for meaningful benchmarks
	for i := 0; i < 100; i++ {
		goal := NewNode("Goal", map[string]interface{}{
			"title":    "Goal " + string(rune(i)),
			"priority": "medium",
		})

		method := NewNode("Method", map[string]interface{}{
			"name":        "Method " + string(rune(i)),
			"effectiveness": 0.5,
		})

		if err := store.AddNode(ctx, goal); err != nil {
			b.Fatalf("Failed to add goal: %v", err)
		}
		if err := store.AddNode(ctx, method); err != nil {
			b.Fatalf("Failed to add method: %v", err)
		}

		// Add edge connecting them
		edge := NewEdge(goal.ID, method.ID, "uses", map[string]interface{}{
			"confidence": 0.7,
		})
		if err := store.AddEdge(ctx, edge); err != nil {
			b.Fatalf("Failed to add edge: %v", err)
		}
	}

	return store
}