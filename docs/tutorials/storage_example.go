// Package tutorials provides example code demonstrating AI Work Studio usage.
package tutorials

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Solifugus/ai-work-studio/pkg/storage"
	"github.com/Solifugus/ai-work-studio/pkg/utils"
)

// StorageExample demonstrates the temporal storage system capabilities.
func StorageExample() error {
	fmt.Println("AI Work Studio - Storage System Example")
	fmt.Println("=======================================")

	// Setup: Create data directory and logger
	dataDir := filepath.Join(os.TempDir(), "ai-work-studio-storage-example")
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}
	defer os.RemoveAll(dataDir) // Cleanup after example

	// Setup logging
	logConfig := utils.DefaultLogConfig("storage-example")
	logger, err := utils.NewLogger(logConfig)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	defer logger.Close()

	// Initialize storage
	storeConfig := storage.Config{
		DataDir: dataDir,
		Logger:  logger,
	}
	store, err := storage.NewStore(storeConfig)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Example 1: Create and store nodes
	fmt.Println("\n1. Creating and storing nodes...")

	// Create a node representing a goal
	goalData := map[string]interface{}{
		"title":       "Learn Temporal Storage",
		"description": "Understand how the temporal storage system works",
		"status":      "active",
		"priority":    8,
		"created_by":  "user",
	}

	goalNode := storage.NewNode("goal", goalData)
	err = store.StoreNode(ctx, goalNode)
	if err != nil {
		return fmt.Errorf("failed to store goal node: %w", err)
	}

	fmt.Printf("Created goal node: %s (Type: %s)\n", goalNode.ID, goalNode.Type)
	fmt.Printf("Node data: %+v\n", goalNode.Data)

	// Create a method node
	methodData := map[string]interface{}{
		"name":        "Read Documentation",
		"description": "Carefully read the storage system documentation",
		"goal_id":     goalNode.ID,
		"parameters": map[string]interface{}{
			"source": "docs/api/overview.md",
			"focus":  "temporal_features",
		},
		"estimated_duration": "30 minutes",
	}

	methodNode := storage.NewNode("method", methodData)
	err = store.StoreNode(ctx, methodNode)
	if err != nil {
		return fmt.Errorf("failed to store method node: %w", err)
	}

	fmt.Printf("Created method node: %s\n", methodNode.ID)

	// Example 2: Create edges to represent relationships
	fmt.Println("\n2. Creating relationships with edges...")

	// Create an edge from goal to method (goal "uses" method)
	edgeData := map[string]interface{}{
		"relationship": "uses_method",
		"confidence":   0.9,
		"created_by":   "learning_loop",
	}

	edge := storage.NewEdge(goalNode.ID, methodNode.ID, "uses", edgeData)
	err = store.StoreEdge(ctx, edge)
	if err != nil {
		return fmt.Errorf("failed to store edge: %w", err)
	}

	fmt.Printf("Created edge: %s -> %s (Type: %s)\n", edge.SourceID, edge.TargetID, edge.Type)

	// Example 3: Query nodes and edges
	fmt.Println("\n3. Querying stored data...")

	// Get node by ID
	retrievedGoal, err := store.GetNodeByID(ctx, goalNode.ID)
	if err != nil {
		return fmt.Errorf("failed to retrieve goal: %w", err)
	}
	fmt.Printf("Retrieved goal: %s\n", retrievedGoal.Data["title"])

	// Query nodes by type
	goalNodes, err := store.QueryNodes(ctx, storage.Query{
		Type: "goal",
	})
	if err != nil {
		return fmt.Errorf("failed to query goal nodes: %w", err)
	}
	fmt.Printf("Found %d goal nodes\n", len(goalNodes))

	// Query edges by source
	goalEdges, err := store.QueryEdges(ctx, storage.Query{
		SourceID: goalNode.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to query edges: %w", err)
	}
	fmt.Printf("Found %d edges from goal\n", len(goalEdges))

	// Example 4: Temporal operations - updating and versioning
	fmt.Println("\n4. Demonstrating temporal versioning...")

	// Wait a moment to ensure distinct timestamps
	time.Sleep(10 * time.Millisecond)

	// Update the goal (this creates a new version)
	fmt.Printf("Original goal status: %s\n", goalNode.Data["status"])

	updatedGoalData := make(map[string]interface{})
	for k, v := range goalData {
		updatedGoalData[k] = v
	}
	updatedGoalData["status"] = "in_progress"
	updatedGoalData["progress_notes"] = "Started reading documentation"
	updatedGoalData["last_updated"] = time.Now().Format(time.RFC3339)

	updatedGoalNode := storage.NewNodeWithID(goalNode.ID, "goal", updatedGoalData)
	err = store.StoreNode(ctx, updatedGoalNode)
	if err != nil {
		return fmt.Errorf("failed to store updated goal: %w", err)
	}

	fmt.Printf("Updated goal status: %s\n", updatedGoalNode.Data["status"])

	// Query for current version
	currentGoal, err := store.GetNodeByID(ctx, goalNode.ID)
	if err != nil {
		return fmt.Errorf("failed to get current goal: %w", err)
	}
	fmt.Printf("Current goal status: %s\n", currentGoal.Data["status"])

	// Example 5: Point-in-time queries
	fmt.Println("\n5. Point-in-time queries...")

	// Get the goal as it existed before the update
	beforeUpdate := updatedGoalNode.ValidFrom.Add(-1 * time.Second)
	historicalGoal, err := store.GetNodeAtTime(ctx, goalNode.ID, beforeUpdate)
	if err != nil {
		log.Printf("Warning: failed to get historical node: %v", err)
	} else {
		fmt.Printf("Goal status before update: %s\n", historicalGoal.Data["status"])
	}

	// Example 6: Advanced querying with filters
	fmt.Println("\n6. Advanced querying...")

	// Query active goals
	activeGoals, err := store.QueryNodes(ctx, storage.Query{
		Type: "goal",
		DataFilter: map[string]interface{}{
			"status": "in_progress",
		},
	})
	if err != nil {
		log.Printf("Warning: failed to query active goals: %v", err)
	} else {
		fmt.Printf("Found %d in-progress goals\n", len(activeGoals))
	}

	// Query high-priority goals
	highPriorityGoals, err := store.QueryNodes(ctx, storage.Query{
		Type: "goal",
		DataFilter: map[string]interface{}{
			"priority": map[string]interface{}{
				"$gte": 7, // Priority >= 7
			},
		},
	})
	if err != nil {
		log.Printf("Warning: failed to query high-priority goals: %v", err)
	} else {
		fmt.Printf("Found %d high-priority goals\n", len(highPriorityGoals))
	}

	// Example 7: Backup and recovery
	fmt.Println("\n7. Backup and recovery...")

	backupDir := filepath.Join(dataDir, "backups")
	err = os.MkdirAll(backupDir, 0755)
	if err != nil {
		log.Printf("Warning: failed to create backup directory: %v", err)
	} else {
		// Create a backup
		backupPath := filepath.Join(backupDir, "storage_backup.json")
		err = store.CreateBackup(ctx, backupPath)
		if err != nil {
			log.Printf("Warning: failed to create backup: %v", err)
		} else {
			fmt.Printf("Created backup at: %s\n", backupPath)

			// Verify backup exists
			if stat, err := os.Stat(backupPath); err == nil {
				fmt.Printf("Backup size: %d bytes\n", stat.Size())
			}
		}
	}

	// Example 8: Complex graph traversal
	fmt.Println("\n8. Graph traversal example...")

	// Create additional nodes to form a more complex graph
	objectiveData := map[string]interface{}{
		"title":       "Read Storage Documentation",
		"description": "Complete reading of the temporal storage docs",
		"goal_id":     goalNode.ID,
		"method_id":   methodNode.ID,
		"status":      "pending",
	}

	objectiveNode := storage.NewNode("objective", objectiveData)
	err = store.StoreNode(ctx, objectiveNode)
	if err != nil {
		log.Printf("Warning: failed to store objective: %v", err)
	} else {
		// Create edges: goal -> objective, objective -> method
		goalToObjective := storage.NewEdge(goalNode.ID, objectiveNode.ID, "has_objective", map[string]interface{}{
			"priority": 1,
		})
		objectiveToMethod := storage.NewEdge(objectiveNode.ID, methodNode.ID, "uses_method", map[string]interface{}{
			"application": "direct",
		})

		store.StoreEdge(ctx, goalToObjective)
		store.StoreEdge(ctx, objectiveToMethod)

		// Traverse: find all methods used by this goal (direct and indirect)
		allGoalEdges, err := store.QueryEdges(ctx, storage.Query{
			SourceID: goalNode.ID,
		})
		if err != nil {
			log.Printf("Warning: failed to query goal edges: %v", err)
		} else {
			fmt.Printf("Goal has %d direct relationships\n", len(allGoalEdges))

			// Follow the graph to find all connected methods
			var methods []string
			visited := make(map[string]bool)

			var traverse func(nodeID string)
			traverse = func(nodeID string) {
				if visited[nodeID] {
					return
				}
				visited[nodeID] = true

				// Get node to check type
				if node, err := store.GetNodeByID(ctx, nodeID); err == nil {
					if node.Type == "method" {
						methods = append(methods, nodeID)
						fmt.Printf("  Found method: %s\n", node.Data["name"])
					}
				}

				// Get edges from this node
				if edges, err := store.QueryEdges(ctx, storage.Query{SourceID: nodeID}); err == nil {
					for _, edge := range edges {
						traverse(edge.TargetID)
					}
				}
			}

			traverse(goalNode.ID)
			fmt.Printf("Total methods connected to goal: %d\n", len(methods))
		}
	}

	fmt.Println("\n✅ Storage example completed successfully!")
	return nil
}

// TemporalQueryExample demonstrates advanced temporal queries.
func TemporalQueryExample() error {
	fmt.Println("Advanced Temporal Query Example")
	fmt.Println("===============================")

	// This would demonstrate:
	// 1. Querying data at specific timestamps
	// 2. Finding changes between time periods
	// 3. Reconstructing historical state
	// 4. Temporal analytics and trends

	fmt.Println("(This would demonstrate advanced temporal patterns)")
	return nil
}

// PerformanceExample demonstrates storage performance optimization.
func PerformanceExample() error {
	fmt.Println("Storage Performance Example")
	fmt.Println("==========================")

	// This would demonstrate:
	// 1. Bulk operations
	// 2. Query optimization
	// 3. Index usage
	// 4. Memory management

	fmt.Println("(This would demonstrate performance optimization)")
	return nil
}

// ValidationExample demonstrates data validation and integrity.
func ValidationExample() error {
	fmt.Println("Data Validation Example")
	fmt.Println("======================")

	// Setup (simplified for example)
	dataDir := filepath.Join(os.TempDir(), "ai-work-studio-validation-example")
	defer os.RemoveAll(dataDir)

	logConfig := utils.DefaultLogConfig("validation-example")
	logger, err := utils.NewLogger(logConfig)
	if err != nil {
		return err
	}
	defer logger.Close()

	storeConfig := storage.Config{
		DataDir: dataDir,
		Logger:  logger,
	}
	store, err := storage.NewStore(storeConfig)
	if err != nil {
		return err
	}
	defer store.Close()

	ctx := context.Background()

	fmt.Println("\n1. Testing data validation...")

	// Example of valid data
	validData := map[string]interface{}{
		"title":    "Valid Goal",
		"priority": 5,
		"status":   "active",
	}

	validNode := storage.NewNode("goal", validData)
	err = store.StoreNode(ctx, validNode)
	if err != nil {
		fmt.Printf("❌ Unexpected error with valid data: %v\n", err)
	} else {
		fmt.Printf("✅ Valid data stored successfully\n")
	}

	// Test data validation (if implemented)
	fmt.Println("\n2. Testing edge validation...")

	// Create a valid edge
	validEdge := storage.NewEdge(validNode.ID, validNode.ID, "self_reference", map[string]interface{}{
		"type": "test",
	})

	err = store.StoreEdge(ctx, validEdge)
	if err != nil {
		fmt.Printf("❌ Error storing valid edge: %v\n", err)
	} else {
		fmt.Printf("✅ Valid edge stored successfully\n")
	}

	fmt.Println("\n✅ Validation example completed!")
	return nil
}