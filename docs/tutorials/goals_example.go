// Package tutorials provides example code demonstrating AI Work Studio usage.
package tutorials

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Solifugus/ai-work-studio/pkg/core"
	"github.com/Solifugus/ai-work-studio/pkg/storage"
	"github.com/Solifugus/ai-work-studio/pkg/utils"
)

// GoalsExample demonstrates how to create and manage goals using the AI Work Studio API.
func GoalsExample() error {
	fmt.Println("AI Work Studio - Goals Management Example")
	fmt.Println("=========================================")

	// Setup: Create data directory and logger
	dataDir := filepath.Join(os.TempDir(), "ai-work-studio-example")
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}
	defer os.RemoveAll(dataDir) // Cleanup after example

	// Setup logging
	logConfig := utils.DefaultLogConfig("goals-example")
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

	// Example 1: Create a main goal
	fmt.Println("\n1. Creating a main goal...")

	mainGoal := &core.Goal{
		Title:       "Learn AI Work Studio",
		Description: "Master the AI Work Studio system to become a more productive developer",
		Status:      core.GoalStatusActive,
		Priority:    8,
		UserContext: map[string]interface{}{
			"domain":     "software_development",
			"timeframe":  "3_months",
			"difficulty": "intermediate",
		},
	}

	err = store.CreateGoal(ctx, mainGoal)
	if err != nil {
		return fmt.Errorf("failed to create main goal: %w", err)
	}

	fmt.Printf("Created goal: %s (ID: %s)\n", mainGoal.Title, mainGoal.ID)

	// Example 2: Create sub-goals
	fmt.Println("\n2. Creating sub-goals...")

	subGoals := []*core.Goal{
		{
			Title:       "Understand the Storage System",
			Description: "Learn how temporal storage works with nodes and edges",
			Status:      core.GoalStatusActive,
			Priority:    7,
			UserContext: map[string]interface{}{
				"parent_goal": mainGoal.ID,
				"area":        "storage",
			},
		},
		{
			Title:       "Master MCP Services",
			Description: "Learn to create and use MCP services for external integrations",
			Status:      core.GoalStatusActive,
			Priority:    6,
			UserContext: map[string]interface{}{
				"parent_goal": mainGoal.ID,
				"area":        "mcp",
			},
		},
		{
			Title:       "Build Custom UI Components",
			Description: "Create custom Fyne widgets for the AI Work Studio interface",
			Status:      core.GoalStatusPaused, // Start this later
			Priority:    5,
			UserContext: map[string]interface{}{
				"parent_goal": mainGoal.ID,
				"area":        "ui",
				"prerequisite": "storage_and_mcp",
			},
		},
	}

	for _, goal := range subGoals {
		err = store.CreateGoal(ctx, goal)
		if err != nil {
			log.Printf("Warning: failed to create sub-goal %s: %v", goal.Title, err)
			continue
		}
		fmt.Printf("Created sub-goal: %s (ID: %s)\n", goal.Title, goal.ID)
	}

	// Example 3: Query goals
	fmt.Println("\n3. Querying goals...")

	// Get all active goals
	activeGoals, err := store.GetGoalsByStatus(ctx, core.GoalStatusActive)
	if err != nil {
		return fmt.Errorf("failed to query active goals: %w", err)
	}

	fmt.Printf("Found %d active goals:\n", len(activeGoals))
	for _, goal := range activeGoals {
		fmt.Printf("  - %s (Priority: %d)\n", goal.Title, goal.Priority)
	}

	// Example 4: Update a goal
	fmt.Println("\n4. Updating a goal...")

	if len(subGoals) > 0 {
		// Complete the first sub-goal
		firstSubGoal := subGoals[0]
		firstSubGoal.Status = core.GoalStatusCompleted
		firstSubGoal.UserContext["completion_notes"] = "Completed after reading storage documentation and implementing examples"

		err = store.UpdateGoal(ctx, firstSubGoal)
		if err != nil {
			log.Printf("Warning: failed to update goal: %v", err)
		} else {
			fmt.Printf("Updated goal '%s' to completed status\n", firstSubGoal.Title)
		}
	}

	// Example 5: Get goal hierarchy
	fmt.Println("\n5. Displaying goal hierarchy...")

	// This would typically use edges to track parent-child relationships
	// For this example, we'll use the user context to show the concept
	allGoals, err := store.GetAllGoals(ctx)
	if err != nil {
		return fmt.Errorf("failed to get all goals: %w", err)
	}

	fmt.Println("Goal Hierarchy:")
	for _, goal := range allGoals {
		parentID, hasParent := goal.UserContext["parent_goal"]
		if !hasParent {
			// This is a main goal
			fmt.Printf("📋 %s (%s)\n", goal.Title, goal.Status)

			// Find and display sub-goals
			for _, subGoal := range allGoals {
				if subParentID, ok := subGoal.UserContext["parent_goal"]; ok && subParentID == goal.ID {
					status := "✅"
					if subGoal.Status == core.GoalStatusActive {
						status = "🔄"
					} else if subGoal.Status == core.GoalStatusPaused {
						status = "⏸️"
					}
					fmt.Printf("  %s %s (%s)\n", status, subGoal.Title, subGoal.Status)
				}
			}
		}
	}

	// Example 6: Advanced querying with temporal data
	fmt.Println("\n6. Temporal query example...")

	// Get the storage node for the main goal to demonstrate temporal queries
	node, err := store.GetNodeByID(ctx, mainGoal.ID)
	if err != nil {
		log.Printf("Warning: failed to get node for temporal query: %v", err)
	} else {
		fmt.Printf("Goal '%s' was created at: %s\n", mainGoal.Title, node.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Goal is valid from: %s\n", node.ValidFrom.Format("2006-01-02 15:04:05"))
		if !node.ValidUntil.IsZero() {
			fmt.Printf("Goal was superseded at: %s\n", node.ValidUntil.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Println("Goal is currently active (no supersession)")
		}
	}

	fmt.Println("\n✅ Goals example completed successfully!")
	return nil
}

// Helper function to demonstrate goal creation with full context
func createDetailedGoal(store *storage.Store, ctx context.Context) error {
	goal := &core.Goal{
		Title:       "Master Advanced Go Patterns",
		Description: "Learn advanced Go programming patterns including concurrency, interfaces, and performance optimization",
		Status:      core.GoalStatusActive,
		Priority:    9,
		UserContext: map[string]interface{}{
			"domain":          "software_engineering",
			"language":        "go",
			"estimated_hours": 40,
			"resources": []string{
				"Effective Go documentation",
				"Go Concurrency Patterns talks",
				"Performance optimization guides",
			},
			"success_criteria": []string{
				"Can implement worker pools efficiently",
				"Understands interface composition patterns",
				"Can profile and optimize Go applications",
			},
			"tags": []string{"programming", "go", "advanced", "performance"},
		},
	}

	return store.CreateGoal(ctx, goal)
}

// ExampleGoalWorkflow demonstrates a complete workflow with goals, objectives, and methods
func ExampleGoalWorkflow() {
	fmt.Println("Complete Goal Workflow Example")
	fmt.Println("==============================")

	// This example would show:
	// 1. Creating a goal
	// 2. Breaking it down into objectives
	// 3. Using methods to achieve objectives
	// 4. Learning from execution results
	// 5. Evolving the approach based on feedback

	// Implementation would go here, demonstrating the full CC-RTC cycle
	fmt.Println("(This would demonstrate the complete workflow)")
}