package core

import (
	"context"
	"fmt"
	"time"

	"github.com/Solifugus/ai-work-studio/pkg/storage"
)

// GoalStatus represents the current state of a goal.
type GoalStatus string

const (
	// GoalStatusActive indicates the goal is actively being pursued
	GoalStatusActive GoalStatus = "active"

	// GoalStatusPaused indicates the goal is temporarily paused
	GoalStatusPaused GoalStatus = "paused"

	// GoalStatusCompleted indicates the goal has been achieved
	GoalStatusCompleted GoalStatus = "completed"

	// GoalStatusArchived indicates the goal is no longer relevant
	GoalStatusArchived GoalStatus = "archived"
)

// Goal represents a user's objective that the AI Work Studio serves.
// Goals form hierarchical relationships where sub-goals serve parent goals.
// They evolve over time through the temporal storage system.
type Goal struct {
	// ID uniquely identifies this goal
	ID string

	// Title is a short, descriptive name for the goal
	Title string

	// Description provides detailed context about what this goal entails
	Description string

	// Status indicates the current state of the goal
	Status GoalStatus

	// Priority is a numeric value (1-10) indicating relative importance.
	// The system learns what priority levels mean through experience rather
	// than hardcoded rules. Higher numbers typically indicate higher priority.
	Priority int

	// UserContext contains additional contextual information specific to the user.
	// This map allows for flexible, evolving understanding of the goal.
	UserContext map[string]interface{}

	// CreatedAt is when this goal was originally created
	CreatedAt time.Time

	// store reference for database operations
	store *storage.Store
}

// GoalManager provides operations for managing goals in the storage system.
type GoalManager struct {
	store *storage.Store
}

// NewGoalManager creates a new manager for goal operations.
func NewGoalManager(store *storage.Store) *GoalManager {
	return &GoalManager{
		store: store,
	}
}

// CreateGoal creates a new goal and stores it in the system.
func (gm *GoalManager) CreateGoal(ctx context.Context, title, description string, priority int, userContext map[string]interface{}) (*Goal, error) {
	if title == "" {
		return nil, fmt.Errorf("goal title cannot be empty")
	}
	if priority < 1 || priority > 10 {
		return nil, fmt.Errorf("priority must be between 1 and 10, got %d", priority)
	}

	now := time.Now()

	// Prepare data for storage node
	data := map[string]interface{}{
		"title":       title,
		"description": description,
		"status":      string(GoalStatusActive), // New goals start as active
		"priority":    priority,
		"user_context": userContext,
		"created_at":  now.Format(time.RFC3339),
	}

	// Create storage node
	node := storage.NewNode("goal", data)

	// Store the node
	if err := gm.store.AddNode(ctx, node); err != nil {
		return nil, fmt.Errorf("failed to store goal: %w", err)
	}

	// Return goal object
	goal := &Goal{
		ID:          node.ID,
		Title:       title,
		Description: description,
		Status:      GoalStatusActive,
		Priority:    priority,
		UserContext: userContext,
		CreatedAt:   now,
		store:       gm.store,
	}

	return goal, nil
}

// GetGoal retrieves a goal by ID.
func (gm *GoalManager) GetGoal(ctx context.Context, goalID string) (*Goal, error) {
	node, err := gm.store.GetNode(ctx, goalID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve goal %s: %w", goalID, err)
	}

	if node.Type != "goal" {
		return nil, fmt.Errorf("node %s is not a goal (type: %s)", goalID, node.Type)
	}

	return gm.nodeToGoal(node)
}

// GetGoalAtTime retrieves the version of a goal that was active at the given time.
func (gm *GoalManager) GetGoalAtTime(ctx context.Context, goalID string, timestamp time.Time) (*Goal, error) {
	node, err := gm.store.GetNodeAtTime(ctx, goalID, timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve goal %s at time %v: %w", goalID, timestamp, err)
	}

	if node.Type != "goal" {
		return nil, fmt.Errorf("node %s is not a goal (type: %s)", goalID, node.Type)
	}

	return gm.nodeToGoal(node)
}

// UpdateGoal creates a new version of a goal with updated information.
func (gm *GoalManager) UpdateGoal(ctx context.Context, goalID string, updates GoalUpdates) (*Goal, error) {
	// Get current goal to validate and provide defaults
	currentGoal, err := gm.GetGoal(ctx, goalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current goal for update: %w", err)
	}

	// Apply updates with defaults from current goal
	title := currentGoal.Title
	if updates.Title != nil {
		title = *updates.Title
		if title == "" {
			return nil, fmt.Errorf("goal title cannot be empty")
		}
	}

	description := currentGoal.Description
	if updates.Description != nil {
		description = *updates.Description
	}

	status := currentGoal.Status
	if updates.Status != nil {
		status = *updates.Status
		if !isValidStatus(status) {
			return nil, fmt.Errorf("invalid goal status: %s", status)
		}
	}

	priority := currentGoal.Priority
	if updates.Priority != nil {
		priority = *updates.Priority
		if priority < 1 || priority > 10 {
			return nil, fmt.Errorf("priority must be between 1 and 10, got %d", priority)
		}
	}

	userContext := currentGoal.UserContext
	if updates.UserContext != nil {
		userContext = updates.UserContext
	}

	// Prepare updated data
	data := map[string]interface{}{
		"title":        title,
		"description":  description,
		"status":       string(status),
		"priority":     priority,
		"user_context": userContext,
		"created_at":   currentGoal.CreatedAt.Format(time.RFC3339),
	}

	// Update in storage
	if err := gm.store.UpdateNode(ctx, goalID, data); err != nil {
		return nil, fmt.Errorf("failed to update goal: %w", err)
	}

	// Return updated goal
	return &Goal{
		ID:          goalID,
		Title:       title,
		Description: description,
		Status:      status,
		Priority:    priority,
		UserContext: userContext,
		CreatedAt:   currentGoal.CreatedAt,
		store:       gm.store,
	}, nil
}

// GoalUpdates defines the fields that can be updated in a goal.
// All fields are optional pointers to allow partial updates.
type GoalUpdates struct {
	Title       *string
	Description *string
	Status      *GoalStatus
	Priority    *int
	UserContext map[string]interface{}
}

// ListGoals returns all goals with optional filtering.
func (gm *GoalManager) ListGoals(ctx context.Context, filter GoalFilter) ([]*Goal, error) {
	query := gm.store.Nodes().OfType("goal")

	// Apply status filter if specified
	if filter.Status != nil {
		query = query.WithData("status", string(*filter.Status))
	}

	// Apply priority filter if specified
	if filter.MinPriority != nil {
		// Note: This requires custom filtering since WithData does exact matches
		// For now, we'll get all and filter in memory
	}

	nodes, err := query.All()
	if err != nil {
		return nil, fmt.Errorf("failed to query goals: %w", err)
	}

	var goals []*Goal
	for _, node := range nodes {
		goal, err := gm.nodeToGoal(node)
		if err != nil {
			continue // Skip invalid nodes
		}

		// Apply priority filter in memory (custom filtering)
		if filter.MinPriority != nil && goal.Priority < *filter.MinPriority {
			continue
		}
		if filter.MaxPriority != nil && goal.Priority > *filter.MaxPriority {
			continue
		}

		goals = append(goals, goal)
	}

	return goals, nil
}

// GoalFilter defines criteria for filtering goals.
type GoalFilter struct {
	Status      *GoalStatus
	MinPriority *int
	MaxPriority *int
}

// AddSubGoal creates a hierarchical relationship where the subgoal serves the parent goal.
func (gm *GoalManager) AddSubGoal(ctx context.Context, parentGoalID, subGoalID string) error {
	// Verify both goals exist
	_, err := gm.GetGoal(ctx, parentGoalID)
	if err != nil {
		return fmt.Errorf("parent goal not found: %w", err)
	}

	_, err = gm.GetGoal(ctx, subGoalID)
	if err != nil {
		return fmt.Errorf("sub goal not found: %w", err)
	}

	// Create edge: sub-goal "serves" parent goal
	edge := storage.NewEdge(subGoalID, parentGoalID, "serves", map[string]interface{}{
		"relationship": "sub_goal_serves_parent",
		"created_at":   time.Now().Format(time.RFC3339),
	})

	if err := gm.store.AddEdge(ctx, edge); err != nil {
		return fmt.Errorf("failed to create goal hierarchy relationship: %w", err)
	}

	return nil
}

// GetSubGoals returns all goals that serve the given parent goal.
func (gm *GoalManager) GetSubGoals(ctx context.Context, parentGoalID string) ([]*Goal, error) {
	// Find all edges of type "serves" targeting the parent goal
	edges, err := gm.store.Edges().OfType("serves").ToNode(parentGoalID).All()
	if err != nil {
		return nil, fmt.Errorf("failed to query sub-goal relationships: %w", err)
	}

	var subGoals []*Goal
	for _, edge := range edges {
		subGoal, err := gm.GetGoal(ctx, edge.SourceID)
		if err != nil {
			continue // Skip if sub-goal no longer exists
		}
		subGoals = append(subGoals, subGoal)
	}

	return subGoals, nil
}

// GetParentGoals returns all goals that this goal serves.
func (gm *GoalManager) GetParentGoals(ctx context.Context, subGoalID string) ([]*Goal, error) {
	// Find all edges of type "serves" originating from the sub-goal
	edges, err := gm.store.Edges().OfType("serves").FromNode(subGoalID).All()
	if err != nil {
		return nil, fmt.Errorf("failed to query parent-goal relationships: %w", err)
	}

	var parentGoals []*Goal
	for _, edge := range edges {
		parentGoal, err := gm.GetGoal(ctx, edge.TargetID)
		if err != nil {
			continue // Skip if parent goal no longer exists
		}
		parentGoals = append(parentGoals, parentGoal)
	}

	return parentGoals, nil
}

// RemoveSubGoal removes a hierarchical relationship between goals.
func (gm *GoalManager) RemoveSubGoal(ctx context.Context, parentGoalID, subGoalID string) error {
	// Find the edge representing this relationship
	edges, err := gm.store.Edges().OfType("serves").FromNode(subGoalID).ToNode(parentGoalID).All()
	if err != nil {
		return fmt.Errorf("failed to query goal relationship: %w", err)
	}

	if len(edges) == 0 {
		return fmt.Errorf("no relationship found between goals %s and %s", subGoalID, parentGoalID)
	}

	// For now, we don't have a delete operation in the store.
	// We could implement this by creating a new version of the edge
	// with ValidUntil set to now, effectively "deleting" it.
	// For this initial implementation, we'll return an error suggesting
	// the relationship should be updated rather than deleted.
	return fmt.Errorf("relationship removal not implemented - consider updating goal status instead")
}

// nodeToGoal converts a storage node to a Goal object.
func (gm *GoalManager) nodeToGoal(node *storage.Node) (*Goal, error) {
	if node == nil {
		return nil, fmt.Errorf("node is nil")
	}

	// Extract fields from node data
	title, ok := node.Data["title"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing title in goal node %s", node.ID)
	}

	description, _ := node.Data["description"].(string) // Optional field

	statusStr, ok := node.Data["status"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing status in goal node %s", node.ID)
	}
	status := GoalStatus(statusStr)

	// Handle priority conversion - it could be int or float64 depending on JSON unmarshaling
	var priority int
	switch v := node.Data["priority"].(type) {
	case float64:
		priority = int(v)
	case int:
		priority = v
	default:
		return nil, fmt.Errorf("invalid or missing priority in goal node %s", node.ID)
	}

	userContext, _ := node.Data["user_context"].(map[string]interface{}) // Optional field

	createdAtStr, ok := node.Data["created_at"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing created_at in goal node %s", node.ID)
	}
	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("invalid created_at format in goal node %s: %w", node.ID, err)
	}

	return &Goal{
		ID:          node.ID,
		Title:       title,
		Description: description,
		Status:      status,
		Priority:    int(priority),
		UserContext: userContext,
		CreatedAt:   createdAt,
		store:       gm.store,
	}, nil
}

// isValidStatus checks if a goal status is valid.
func isValidStatus(status GoalStatus) bool {
	switch status {
	case GoalStatusActive, GoalStatusPaused, GoalStatusCompleted, GoalStatusArchived:
		return true
	default:
		return false
	}
}

// String returns a string representation of the goal status.
func (gs GoalStatus) String() string {
	return string(gs)
}

// IsActive returns true if the goal is actively being pursued.
func (g *Goal) IsActive() bool {
	return g.Status == GoalStatusActive
}

// IsCompleted returns true if the goal has been achieved.
func (g *Goal) IsCompleted() bool {
	return g.Status == GoalStatusCompleted
}

// Update provides a convenient way to update a goal through its instance.
func (g *Goal) Update(ctx context.Context, updates GoalUpdates) error {
	if g.store == nil {
		return fmt.Errorf("goal is not connected to storage")
	}

	gm := &GoalManager{store: g.store}
	updatedGoal, err := gm.UpdateGoal(ctx, g.ID, updates)
	if err != nil {
		return err
	}

	// Update this instance with the new values
	*g = *updatedGoal
	return nil
}