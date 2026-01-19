package storage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Test helper to create a temporary directory for testing
func createTempDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "ai-work-studio-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}

func TestNewStore(t *testing.T) {
	tempDir := createTempDir(t)

	store, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	if store.dataDir != tempDir {
		t.Errorf("Expected dataDir %s, got %s", tempDir, store.dataDir)
	}

	// Verify directory structure was created
	nodesDir := filepath.Join(tempDir, "nodes")
	edgesDir := filepath.Join(tempDir, "edges")

	if _, err := os.Stat(nodesDir); os.IsNotExist(err) {
		t.Errorf("Nodes directory was not created")
	}
	if _, err := os.Stat(edgesDir); os.IsNotExist(err) {
		t.Errorf("Edges directory was not created")
	}

	// Clean up
	store.Close()
}

func TestNodeOperations(t *testing.T) {
	tempDir := createTempDir(t)
	store, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Test data
	nodeData := map[string]interface{}{
		"name":        "test-goal",
		"description": "A test goal",
		"priority":    10,
	}

	t.Run("AddNode", func(t *testing.T) {
		node := NewNode("goal", nodeData)

		err := store.AddNode(ctx, node)
		if err != nil {
			t.Fatalf("Failed to add node: %v", err)
		}

		// Verify node exists in memory
		if _, exists := store.nodes[node.ID]; !exists {
			t.Error("Node not found in memory after adding")
		}

		// Verify file was created
		filePath := filepath.Join(tempDir, "nodes", "goal", node.ID+".json")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Error("Node file was not created")
		}
	})

	t.Run("GetNode", func(t *testing.T) {
		node := NewNode("goal", nodeData)
		store.AddNode(ctx, node)

		retrieved, err := store.GetNode(ctx, node.ID)
		if err != nil {
			t.Fatalf("Failed to get node: %v", err)
		}

		if retrieved.ID != node.ID {
			t.Errorf("Expected ID %s, got %s", node.ID, retrieved.ID)
		}
		if retrieved.Type != node.Type {
			t.Errorf("Expected Type %s, got %s", node.Type, retrieved.Type)
		}

		// Check data
		if retrieved.Data["name"] != nodeData["name"] {
			t.Errorf("Data mismatch: expected %v, got %v", nodeData["name"], retrieved.Data["name"])
		}
	})

	t.Run("UpdateNode", func(t *testing.T) {
		node := NewNode("goal", nodeData)
		store.AddNode(ctx, node)

		// Update data
		updatedData := map[string]interface{}{
			"name":        "updated-goal",
			"description": "An updated test goal",
			"priority":    20,
		}

		err := store.UpdateNode(ctx, node.ID, updatedData)
		if err != nil {
			t.Fatalf("Failed to update node: %v", err)
		}

		// Get updated node
		retrieved, err := store.GetNode(ctx, node.ID)
		if err != nil {
			t.Fatalf("Failed to get updated node: %v", err)
		}

		if retrieved.Data["name"] != "updated-goal" {
			t.Errorf("Update failed: expected 'updated-goal', got %v", retrieved.Data["name"])
		}
		if retrieved.Data["priority"] != 20 {
			t.Errorf("Update failed: expected priority 20, got %v", retrieved.Data["priority"])
		}

		// Verify that history has two versions
		history := store.nodes[node.ID]
		if len(history) != 2 {
			t.Errorf("Expected 2 versions in history, got %d", len(history))
		}
	})

	t.Run("GetNodesByType", func(t *testing.T) {
		// Add multiple nodes of same type
		node1 := NewNode("method", map[string]interface{}{"name": "method1"})
		node2 := NewNode("method", map[string]interface{}{"name": "method2"})
		node3 := NewNode("goal", map[string]interface{}{"name": "goal1"})

		store.AddNode(ctx, node1)
		store.AddNode(ctx, node2)
		store.AddNode(ctx, node3)

		methods, err := store.GetNodesByType(ctx, "method")
		if err != nil {
			t.Fatalf("Failed to get nodes by type: %v", err)
		}

		if len(methods) != 2 {
			t.Errorf("Expected 2 method nodes, got %d", len(methods))
		}
	})

	t.Run("GetNodeAtTime", func(t *testing.T) {
		node := NewNode("goal", nodeData)
		creationTime := node.ValidFrom // Use the node's actual creation time
		store.AddNode(ctx, node)

		// Update after a small delay
		time.Sleep(10 * time.Millisecond)
		updatedData := map[string]interface{}{
			"name":        "updated-goal",
			"description": "An updated test goal",
		}
		store.UpdateNode(ctx, node.ID, updatedData)
		updateTime := time.Now() // Capture time after update

		// Get original version
		original, err := store.GetNodeAtTime(ctx, node.ID, creationTime)
		if err != nil {
			t.Fatalf("Failed to get node at creation time: %v", err)
		}

		if original.Data["name"] != "test-goal" {
			t.Errorf("Expected original name, got %v", original.Data["name"])
		}

		// Get updated version
		updated, err := store.GetNodeAtTime(ctx, node.ID, updateTime)
		if err != nil {
			t.Fatalf("Failed to get node at update time: %v", err)
		}

		if updated.Data["name"] != "updated-goal" {
			t.Errorf("Expected updated name, got %v", updated.Data["name"])
		}
	})

	t.Run("NodeNotFound", func(t *testing.T) {
		_, err := store.GetNode(ctx, "non-existent-id")
		if err == nil {
			t.Error("Expected error for non-existent node")
		}
	})
}

func TestEdgeOperations(t *testing.T) {
	tempDir := createTempDir(t)
	store, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create test nodes first
	sourceNode := NewNode("goal", map[string]interface{}{"name": "source"})
	targetNode := NewNode("method", map[string]interface{}{"name": "target"})

	store.AddNode(ctx, sourceNode)
	store.AddNode(ctx, targetNode)

	t.Run("AddEdge", func(t *testing.T) {
		edgeData := map[string]interface{}{
			"strength": 0.8,
			"notes":    "test relationship",
		}

		edge := NewEdge(sourceNode.ID, targetNode.ID, "implements", edgeData)

		err := store.AddEdge(ctx, edge)
		if err != nil {
			t.Fatalf("Failed to add edge: %v", err)
		}

		// Verify edge exists in memory
		if _, exists := store.edges[edge.ID]; !exists {
			t.Error("Edge not found in memory after adding")
		}

		// Verify file was created
		filePath := filepath.Join(tempDir, "edges", edge.ID+".json")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Error("Edge file was not created")
		}
	})

	t.Run("GetEdge", func(t *testing.T) {
		edgeData := map[string]interface{}{"strength": 0.9}
		edge := NewEdge(sourceNode.ID, targetNode.ID, "depends_on", edgeData)
		store.AddEdge(ctx, edge)

		retrieved, err := store.GetEdge(ctx, edge.ID)
		if err != nil {
			t.Fatalf("Failed to get edge: %v", err)
		}

		if retrieved.ID != edge.ID {
			t.Errorf("Expected ID %s, got %s", edge.ID, retrieved.ID)
		}
		if retrieved.SourceID != sourceNode.ID {
			t.Errorf("Expected SourceID %s, got %s", sourceNode.ID, retrieved.SourceID)
		}
		if retrieved.TargetID != targetNode.ID {
			t.Errorf("Expected TargetID %s, got %s", targetNode.ID, retrieved.TargetID)
		}
	})

	t.Run("UpdateEdge", func(t *testing.T) {
		edgeData := map[string]interface{}{"strength": 0.5}
		edge := NewEdge(sourceNode.ID, targetNode.ID, "refines", edgeData)
		store.AddEdge(ctx, edge)

		// Update edge data
		updatedData := map[string]interface{}{
			"strength": 0.95,
			"notes":    "updated relationship",
		}

		err := store.UpdateEdge(ctx, edge.ID, updatedData)
		if err != nil {
			t.Fatalf("Failed to update edge: %v", err)
		}

		// Get updated edge
		retrieved, err := store.GetEdge(ctx, edge.ID)
		if err != nil {
			t.Fatalf("Failed to get updated edge: %v", err)
		}

		if retrieved.Data["strength"] != 0.95 {
			t.Errorf("Update failed: expected strength 0.95, got %v", retrieved.Data["strength"])
		}
	})

	t.Run("GetEdgesByType", func(t *testing.T) {
		// Create a fresh store to avoid interference from other subtests
		tempDir2 := createTempDir(t)
		store2, err := NewStore(tempDir2)
		if err != nil {
			t.Fatalf("Failed to create second store: %v", err)
		}
		defer store2.Close()

		// Add nodes to second store
		store2.AddNode(ctx, sourceNode)
		store2.AddNode(ctx, targetNode)

		// Add multiple edges of same type
		edge1 := NewEdge(sourceNode.ID, targetNode.ID, "implements", map[string]interface{}{})
		edge2 := NewEdge(targetNode.ID, sourceNode.ID, "implements", map[string]interface{}{})
		edge3 := NewEdge(sourceNode.ID, targetNode.ID, "depends_on", map[string]interface{}{})

		store2.AddEdge(ctx, edge1)
		store2.AddEdge(ctx, edge2)
		store2.AddEdge(ctx, edge3)

		implementsEdges, err := store2.GetEdgesByType(ctx, "implements")
		if err != nil {
			t.Fatalf("Failed to get edges by type: %v", err)
		}

		if len(implementsEdges) != 2 {
			t.Errorf("Expected 2 implements edges, got %d", len(implementsEdges))
		}
	})

	t.Run("GetEdgeAtTime", func(t *testing.T) {
		edgeData := map[string]interface{}{"strength": 0.6}
		edge := NewEdge(sourceNode.ID, targetNode.ID, "test", edgeData)
		creationTime := edge.ValidFrom // Use the edge's actual creation time
		store.AddEdge(ctx, edge)

		// Update after a small delay
		time.Sleep(10 * time.Millisecond)
		updatedData := map[string]interface{}{"strength": 0.9}
		store.UpdateEdge(ctx, edge.ID, updatedData)
		updateTime := time.Now() // Capture time after update

		// Get original version
		original, err := store.GetEdgeAtTime(ctx, edge.ID, creationTime)
		if err != nil {
			t.Fatalf("Failed to get edge at creation time: %v", err)
		}

		if original.Data["strength"] != 0.6 {
			t.Errorf("Expected original strength 0.6, got %v", original.Data["strength"])
		}

		// Get updated version
		updated, err := store.GetEdgeAtTime(ctx, edge.ID, updateTime)
		if err != nil {
			t.Fatalf("Failed to get edge at update time: %v", err)
		}

		if updated.Data["strength"] != 0.9 {
			t.Errorf("Expected updated strength 0.9, got %v", updated.Data["strength"])
		}
	})

	t.Run("EdgeWithNonExistentNodes", func(t *testing.T) {
		edge := NewEdge("non-existent-1", "non-existent-2", "test", map[string]interface{}{})

		err := store.AddEdge(ctx, edge)
		if err == nil {
			t.Error("Expected error when adding edge with non-existent nodes")
		}
	})
}

func TestGraphTraversal(t *testing.T) {
	tempDir := createTempDir(t)
	store, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create a small graph: goal -> method1 -> objective
	//                             -> method2
	goalNode := NewNode("goal", map[string]interface{}{"name": "test-goal"})
	method1Node := NewNode("method", map[string]interface{}{"name": "method1"})
	method2Node := NewNode("method", map[string]interface{}{"name": "method2"})
	objectiveNode := NewNode("objective", map[string]interface{}{"name": "test-objective"})

	store.AddNode(ctx, goalNode)
	store.AddNode(ctx, method1Node)
	store.AddNode(ctx, method2Node)
	store.AddNode(ctx, objectiveNode)

	// Add edges
	edge1 := NewEdge(goalNode.ID, method1Node.ID, "implements", map[string]interface{}{})
	edge2 := NewEdge(goalNode.ID, method2Node.ID, "implements", map[string]interface{}{})
	edge3 := NewEdge(method1Node.ID, objectiveNode.ID, "produces", map[string]interface{}{})

	store.AddEdge(ctx, edge1)
	store.AddEdge(ctx, edge2)
	store.AddEdge(ctx, edge3)

	t.Run("GetNeighbors", func(t *testing.T) {
		// Get neighbors of goal node
		neighbors, err := store.GetNeighbors(ctx, goalNode.ID)
		if err != nil {
			t.Fatalf("Failed to get neighbors: %v", err)
		}

		if len(neighbors) != 2 {
			t.Errorf("Expected 2 neighbors for goal node, got %d", len(neighbors))
		}

		// Verify both methods are neighbors
		neighborNames := make(map[string]bool)
		for _, neighbor := range neighbors {
			if name, ok := neighbor.Data["name"]; ok {
				neighborNames[name.(string)] = true
			}
		}

		if !neighborNames["method1"] || !neighborNames["method2"] {
			t.Error("Expected method1 and method2 as neighbors of goal")
		}

		// Get neighbors of method1 node
		method1Neighbors, err := store.GetNeighbors(ctx, method1Node.ID)
		if err != nil {
			t.Fatalf("Failed to get method1 neighbors: %v", err)
		}

		if len(method1Neighbors) != 2 {
			t.Errorf("Expected 2 neighbors for method1 node, got %d", len(method1Neighbors))
		}
	})
}

func TestPersistenceAndLoading(t *testing.T) {
	tempDir := createTempDir(t)

	// Create store and add some data
	store1, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create first store: %v", err)
	}

	ctx := context.Background()

	// Add test data
	node1 := NewNode("goal", map[string]interface{}{"name": "persistent-goal"})
	node2 := NewNode("method", map[string]interface{}{"name": "persistent-method"})

	store1.AddNode(ctx, node1)
	store1.AddNode(ctx, node2)

	edge1 := NewEdge(node1.ID, node2.ID, "implements", map[string]interface{}{"strength": 0.8})
	store1.AddEdge(ctx, edge1)

	store1.Close()

	// Create new store instance and verify data is loaded
	store2, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create second store: %v", err)
	}
	defer store2.Close()

	// Verify nodes are loaded
	retrievedNode1, err := store2.GetNode(ctx, node1.ID)
	if err != nil {
		t.Fatalf("Failed to get persistent node: %v", err)
	}

	if retrievedNode1.Data["name"] != "persistent-goal" {
		t.Errorf("Node data not persisted correctly: expected 'persistent-goal', got %v", retrievedNode1.Data["name"])
	}

	// Verify edges are loaded
	retrievedEdge1, err := store2.GetEdge(ctx, edge1.ID)
	if err != nil {
		t.Fatalf("Failed to get persistent edge: %v", err)
	}

	if retrievedEdge1.Data["strength"] != 0.8 {
		t.Errorf("Edge data not persisted correctly: expected 0.8, got %v", retrievedEdge1.Data["strength"])
	}
}

func TestConcurrentAccess(t *testing.T) {
	tempDir := createTempDir(t)
	store, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create initial nodes for edge testing
	sourceNode := NewNode("goal", map[string]interface{}{"name": "source"})
	targetNode := NewNode("method", map[string]interface{}{"name": "target"})
	store.AddNode(ctx, sourceNode)
	store.AddNode(ctx, targetNode)

	const numGoroutines = 10
	const numOpsPerGoroutine = 5

	// Test concurrent node operations
	t.Run("ConcurrentNodeOps", func(t *testing.T) {
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer func() { done <- true }()

				for j := 0; j < numOpsPerGoroutine; j++ {
					// Add node
					node := NewNode("concurrent", map[string]interface{}{
						"goroutine": id,
						"operation": j,
					})
					store.AddNode(ctx, node)

					// Get node
					store.GetNode(ctx, node.ID)

					// Update node
					store.UpdateNode(ctx, node.ID, map[string]interface{}{
						"updated": true,
					})
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})

	// Test concurrent edge operations
	t.Run("ConcurrentEdgeOps", func(t *testing.T) {
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer func() { done <- true }()

				for j := 0; j < numOpsPerGoroutine; j++ {
					// Add edge
					edge := NewEdge(sourceNode.ID, targetNode.ID, "concurrent", map[string]interface{}{
						"goroutine": id,
						"operation": j,
					})
					store.AddEdge(ctx, edge)

					// Get edge
					store.GetEdge(ctx, edge.ID)

					// Update edge
					store.UpdateEdge(ctx, edge.ID, map[string]interface{}{
						"updated": true,
					})
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})
}

func TestErrorHandling(t *testing.T) {
	tempDir := createTempDir(t)
	store, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	t.Run("NilNode", func(t *testing.T) {
		err := store.AddNode(ctx, nil)
		if err == nil {
			t.Error("Expected error when adding nil node")
		}
	})

	t.Run("NilEdge", func(t *testing.T) {
		err := store.AddEdge(ctx, nil)
		if err == nil {
			t.Error("Expected error when adding nil edge")
		}
	})

	t.Run("UpdateNonExistentNode", func(t *testing.T) {
		err := store.UpdateNode(ctx, "non-existent", map[string]interface{}{})
		if err == nil {
			t.Error("Expected error when updating non-existent node")
		}
	})

	t.Run("UpdateNonExistentEdge", func(t *testing.T) {
		err := store.UpdateEdge(ctx, "non-existent", map[string]interface{}{})
		if err == nil {
			t.Error("Expected error when updating non-existent edge")
		}
	})
}

func TestFileFormat(t *testing.T) {
	tempDir := createTempDir(t)
	store, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Add a node and verify file format
	node := NewNode("goal", map[string]interface{}{"test": "data"})
	store.AddNode(ctx, node)

	// Update the node to create version history
	store.UpdateNode(ctx, node.ID, map[string]interface{}{"test": "updated-data"})

	// Read the file directly and verify it's valid JSON array
	filePath := filepath.Join(tempDir, "nodes", "goal", node.ID+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read node file: %v", err)
	}

	var history []Node
	if err := json.Unmarshal(data, &history); err != nil {
		t.Fatalf("Node file is not valid JSON array: %v", err)
	}

	if len(history) != 2 {
		t.Errorf("Expected 2 versions in file, got %d", len(history))
	}

	// Verify the versions are ordered correctly (should be in temporal order)
	if !history[1].ValidFrom.After(history[0].ValidFrom) {
		t.Error("Versions not ordered correctly in file")
	}
}