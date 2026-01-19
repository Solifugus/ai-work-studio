package core

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/ai-work-studio/pkg/storage"
)

// ObjectiveStatus represents the current state of an objective.
type ObjectiveStatus string

const (
	// ObjectiveStatusPending indicates the objective is created but not yet started
	ObjectiveStatusPending ObjectiveStatus = "pending"

	// ObjectiveStatusInProgress indicates the objective is currently being worked on
	ObjectiveStatusInProgress ObjectiveStatus = "in_progress"

	// ObjectiveStatusCompleted indicates the objective has been successfully achieved
	ObjectiveStatusCompleted ObjectiveStatus = "completed"

	// ObjectiveStatusFailed indicates the objective failed and cannot be completed
	ObjectiveStatusFailed ObjectiveStatus = "failed"

	// ObjectiveStatusPaused indicates the objective is temporarily paused
	ObjectiveStatusPaused ObjectiveStatus = "paused"
)

// ObjectiveResult captures the outcome when an objective completes.
type ObjectiveResult struct {
	// Success indicates whether the objective was achieved
	Success bool `json:"success"`

	// Message provides a human-readable description of the outcome
	Message string `json:"message"`

	// Data contains structured output data from the objective execution
	Data map[string]interface{} `json:"data,omitempty"`

	// TokensUsed tracks LLM token consumption for budget management
	TokensUsed int `json:"tokens_used"`

	// ExecutionTime is how long the objective took to complete
	ExecutionTime time.Duration `json:"execution_time"`

	// CompletedAt is when the objective finished
	CompletedAt time.Time `json:"completed_at"`
}

// Objective represents a specific task to achieve a goal using a proven method.
// Objectives track their lifecycle from creation to completion and maintain minimal
// context for execution efficiency.
type Objective struct {
	// ID uniquely identifies this objective
	ID string

	// GoalID links this objective to the goal it serves
	GoalID string

	// MethodID links this objective to the method it uses
	MethodID string

	// Title is a short, descriptive name for the objective
	Title string

	// Description provides detailed context about what this objective accomplishes
	Description string

	// Status indicates the current state of the objective
	Status ObjectiveStatus

	// Context contains minimal necessary data for execution.
	// Prefer references to data rather than embedding full data.
	Context map[string]interface{}

	// Result contains the outcome when the objective completes
	Result *ObjectiveResult

	// Priority is a numeric value (1-10) indicating relative importance.
	// Inherited from the goal but can be adjusted for task-specific urgency.
	Priority int

	// CreatedAt is when this objective was originally created
	CreatedAt time.Time

	// StartedAt is when work on this objective began
	StartedAt *time.Time

	// CompletedAt is when this objective finished (success or failure)
	CompletedAt *time.Time

	// store reference for database operations
	store *storage.Store
}

// ObjectiveManager provides operations for managing objectives in the storage system.
type ObjectiveManager struct {
	store *storage.Store
}

// NewObjectiveManager creates a new manager for objective operations.
func NewObjectiveManager(store *storage.Store) *ObjectiveManager {
	return &ObjectiveManager{
		store: store,
	}
}

// CreateObjective creates a new objective and stores it in the system.
// It also establishes the relationships to the goal and method via edges.
func (om *ObjectiveManager) CreateObjective(ctx context.Context, goalID, methodID, title, description string, context map[string]interface{}, priority int) (*Objective, error) {
	if title == "" {
		return nil, fmt.Errorf("objective title cannot be empty")
	}
	if goalID == "" {
		return nil, fmt.Errorf("goal ID cannot be empty")
	}
	if methodID == "" {
		return nil, fmt.Errorf("method ID cannot be empty")
	}
	if priority < 1 || priority > 10 {
		return nil, fmt.Errorf("priority must be between 1 and 10, got %d", priority)
	}

	now := time.Now()

	// Ensure context is not nil
	if context == nil {
		context = make(map[string]interface{})
	}

	// Prepare data for storage node
	data := map[string]interface{}{
		"goal_id":     goalID,
		"method_id":   methodID,
		"title":       title,
		"description": description,
		"status":      string(ObjectiveStatusPending), // New objectives start as pending
		"context":     context,
		"priority":    priority,
		"created_at":  now.Format(time.RFC3339),
		"started_at":  nil,
		"completed_at": nil,
		"result":      nil,
	}

	// Create storage node
	node := storage.NewNode("objective", data)

	// Store the node
	if err := om.store.AddNode(ctx, node); err != nil {
		return nil, fmt.Errorf("failed to store objective: %w", err)
	}

	// Create relationships via edges
	// Objective "serves" Goal
	servesEdge := storage.NewEdge(node.ID, goalID, "serves", map[string]interface{}{
		"relationship": "objective_serves_goal",
		"created_at":   now.Format(time.RFC3339),
	})
	if err := om.store.AddEdge(ctx, servesEdge); err != nil {
		return nil, fmt.Errorf("failed to create objective-goal relationship: %w", err)
	}

	// Objective "uses" Method
	usesEdge := storage.NewEdge(node.ID, methodID, "uses", map[string]interface{}{
		"relationship": "objective_uses_method",
		"created_at":   now.Format(time.RFC3339),
	})
	if err := om.store.AddEdge(ctx, usesEdge); err != nil {
		return nil, fmt.Errorf("failed to create objective-method relationship: %w", err)
	}

	// Return objective object
	objective := &Objective{
		ID:          node.ID,
		GoalID:      goalID,
		MethodID:    methodID,
		Title:       title,
		Description: description,
		Status:      ObjectiveStatusPending,
		Context:     context,
		Priority:    priority,
		CreatedAt:   now,
		store:       om.store,
	}

	return objective, nil
}

// GetObjective retrieves an objective by ID.
func (om *ObjectiveManager) GetObjective(ctx context.Context, objectiveID string) (*Objective, error) {
	node, err := om.store.GetNode(ctx, objectiveID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve objective %s: %w", objectiveID, err)
	}

	if node.Type != "objective" {
		return nil, fmt.Errorf("node %s is not an objective (type: %s)", objectiveID, node.Type)
	}

	return om.nodeToObjective(node)
}

// GetObjectiveAtTime retrieves the version of an objective that was active at the given time.
func (om *ObjectiveManager) GetObjectiveAtTime(ctx context.Context, objectiveID string, timestamp time.Time) (*Objective, error) {
	node, err := om.store.GetNodeAtTime(ctx, objectiveID, timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve objective %s at time %v: %w", objectiveID, timestamp, err)
	}

	if node.Type != "objective" {
		return nil, fmt.Errorf("node %s is not an objective (type: %s)", objectiveID, node.Type)
	}

	return om.nodeToObjective(node)
}

// UpdateObjective creates a new version of an objective with updated information.
func (om *ObjectiveManager) UpdateObjective(ctx context.Context, objectiveID string, updates ObjectiveUpdates) (*Objective, error) {
	// Get current objective to validate and provide defaults
	currentObjective, err := om.GetObjective(ctx, objectiveID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current objective for update: %w", err)
	}

	// Apply updates with defaults from current objective
	goalID := currentObjective.GoalID
	if updates.GoalID != nil {
		goalID = *updates.GoalID
		if goalID == "" {
			return nil, fmt.Errorf("goal ID cannot be empty")
		}
	}

	methodID := currentObjective.MethodID
	if updates.MethodID != nil {
		methodID = *updates.MethodID
		if methodID == "" {
			return nil, fmt.Errorf("method ID cannot be empty")
		}
	}

	title := currentObjective.Title
	if updates.Title != nil {
		title = *updates.Title
		if title == "" {
			return nil, fmt.Errorf("objective title cannot be empty")
		}
	}

	description := currentObjective.Description
	if updates.Description != nil {
		description = *updates.Description
	}

	status := currentObjective.Status
	if updates.Status != nil {
		status = *updates.Status
		if !isValidObjectiveStatus(status) {
			return nil, fmt.Errorf("invalid objective status: %s", status)
		}
	}

	context := currentObjective.Context
	if updates.Context != nil {
		context = updates.Context
	}

	result := currentObjective.Result
	if updates.Result != nil {
		result = updates.Result
	}

	priority := currentObjective.Priority
	if updates.Priority != nil {
		priority = *updates.Priority
		if priority < 1 || priority > 10 {
			return nil, fmt.Errorf("priority must be between 1 and 10, got %d", priority)
		}
	}

	startedAt := currentObjective.StartedAt
	if updates.StartedAt != nil {
		startedAt = updates.StartedAt
	}

	completedAt := currentObjective.CompletedAt
	if updates.CompletedAt != nil {
		completedAt = updates.CompletedAt
	}

	// Prepare result data for storage
	var resultData map[string]interface{}
	if result != nil {
		resultData = map[string]interface{}{
			"success":        result.Success,
			"message":        result.Message,
			"data":          result.Data,
			"tokens_used":    result.TokensUsed,
			"execution_time": result.ExecutionTime.String(),
			"completed_at":   result.CompletedAt.Format(time.RFC3339),
		}
	}

	// Prepare time fields
	var startedAtStr *string
	if startedAt != nil {
		str := startedAt.Format(time.RFC3339)
		startedAtStr = &str
	}

	var completedAtStr *string
	if completedAt != nil {
		str := completedAt.Format(time.RFC3339)
		completedAtStr = &str
	}

	// Prepare updated data
	data := map[string]interface{}{
		"goal_id":      goalID,
		"method_id":    methodID,
		"title":        title,
		"description":  description,
		"status":       string(status),
		"context":      context,
		"priority":     priority,
		"result":       resultData,
		"created_at":   currentObjective.CreatedAt.Format(time.RFC3339),
		"started_at":   startedAtStr,
		"completed_at": completedAtStr,
	}

	// Update in storage
	if err := om.store.UpdateNode(ctx, objectiveID, data); err != nil {
		return nil, fmt.Errorf("failed to update objective: %w", err)
	}

	// Return updated objective
	return &Objective{
		ID:          objectiveID,
		GoalID:      goalID,
		MethodID:    methodID,
		Title:       title,
		Description: description,
		Status:      status,
		Context:     context,
		Result:      result,
		Priority:    priority,
		CreatedAt:   currentObjective.CreatedAt,
		StartedAt:   startedAt,
		CompletedAt: completedAt,
		store:       om.store,
	}, nil
}

// ObjectiveUpdates defines the fields that can be updated in an objective.
// All fields are optional pointers to allow partial updates.
type ObjectiveUpdates struct {
	GoalID      *string
	MethodID    *string
	Title       *string
	Description *string
	Status      *ObjectiveStatus
	Context     map[string]interface{}
	Result      *ObjectiveResult
	Priority    *int
	StartedAt   *time.Time
	CompletedAt *time.Time
}

// ListObjectives returns all objectives with optional filtering.
func (om *ObjectiveManager) ListObjectives(ctx context.Context, filter ObjectiveFilter) ([]*Objective, error) {
	query := om.store.Nodes().OfType("objective")

	// Apply status filter if specified
	if filter.Status != nil {
		query = query.WithData("status", string(*filter.Status))
	}

	// Apply goal filter if specified
	if filter.GoalID != nil {
		query = query.WithData("goal_id", *filter.GoalID)
	}

	// Apply method filter if specified
	if filter.MethodID != nil {
		query = query.WithData("method_id", *filter.MethodID)
	}

	nodes, err := query.All()
	if err != nil {
		return nil, fmt.Errorf("failed to query objectives: %w", err)
	}

	var objectives []*Objective
	for _, node := range nodes {
		objective, err := om.nodeToObjective(node)
		if err != nil {
			continue // Skip invalid nodes
		}

		// Apply priority filter in memory
		if filter.MinPriority != nil && objective.Priority < *filter.MinPriority {
			continue
		}
		if filter.MaxPriority != nil && objective.Priority > *filter.MaxPriority {
			continue
		}

		objectives = append(objectives, objective)
	}

	return objectives, nil
}

// ObjectiveFilter defines criteria for filtering objectives.
type ObjectiveFilter struct {
	Status      *ObjectiveStatus
	GoalID      *string
	MethodID    *string
	MinPriority *int
	MaxPriority *int
}

// StartObjective begins work on an objective by changing its status to in_progress.
func (om *ObjectiveManager) StartObjective(ctx context.Context, objectiveID string) (*Objective, error) {
	objective, err := om.GetObjective(ctx, objectiveID)
	if err != nil {
		return nil, fmt.Errorf("failed to get objective: %w", err)
	}

	if objective.Status != ObjectiveStatusPending {
		return nil, fmt.Errorf("can only start pending objectives, current status: %s", objective.Status)
	}

	now := time.Now()
	updates := ObjectiveUpdates{
		Status:    &[]ObjectiveStatus{ObjectiveStatusInProgress}[0],
		StartedAt: &now,
	}

	return om.UpdateObjective(ctx, objectiveID, updates)
}

// CompleteObjective marks an objective as completed with the given result.
func (om *ObjectiveManager) CompleteObjective(ctx context.Context, objectiveID string, result ObjectiveResult) (*Objective, error) {
	objective, err := om.GetObjective(ctx, objectiveID)
	if err != nil {
		return nil, fmt.Errorf("failed to get objective: %w", err)
	}

	if objective.Status != ObjectiveStatusInProgress {
		return nil, fmt.Errorf("can only complete in-progress objectives, current status: %s", objective.Status)
	}

	now := time.Now()
	result.CompletedAt = now

	// Calculate execution time if objective was started
	if objective.StartedAt != nil {
		result.ExecutionTime = now.Sub(*objective.StartedAt)
	}

	status := ObjectiveStatusCompleted
	if !result.Success {
		status = ObjectiveStatusFailed
	}

	updates := ObjectiveUpdates{
		Status:      &status,
		Result:      &result,
		CompletedAt: &now,
	}

	return om.UpdateObjective(ctx, objectiveID, updates)
}

// FailObjective marks an objective as failed with the given error information.
func (om *ObjectiveManager) FailObjective(ctx context.Context, objectiveID string, errorMessage string, tokensUsed int) (*Objective, error) {
	result := ObjectiveResult{
		Success:    false,
		Message:    errorMessage,
		TokensUsed: tokensUsed,
	}

	return om.CompleteObjective(ctx, objectiveID, result)
}

// PauseObjective temporarily pauses work on an objective.
func (om *ObjectiveManager) PauseObjective(ctx context.Context, objectiveID string) (*Objective, error) {
	objective, err := om.GetObjective(ctx, objectiveID)
	if err != nil {
		return nil, fmt.Errorf("failed to get objective: %w", err)
	}

	if objective.Status != ObjectiveStatusInProgress {
		return nil, fmt.Errorf("can only pause in-progress objectives, current status: %s", objective.Status)
	}

	status := ObjectiveStatusPaused
	updates := ObjectiveUpdates{
		Status: &status,
	}

	return om.UpdateObjective(ctx, objectiveID, updates)
}

// ResumeObjective resumes work on a paused objective.
func (om *ObjectiveManager) ResumeObjective(ctx context.Context, objectiveID string) (*Objective, error) {
	objective, err := om.GetObjective(ctx, objectiveID)
	if err != nil {
		return nil, fmt.Errorf("failed to get objective: %w", err)
	}

	if objective.Status != ObjectiveStatusPaused {
		return nil, fmt.Errorf("can only resume paused objectives, current status: %s", objective.Status)
	}

	status := ObjectiveStatusInProgress
	updates := ObjectiveUpdates{
		Status: &status,
	}

	return om.UpdateObjective(ctx, objectiveID, updates)
}

// GetObjectivesForGoal returns all objectives that serve the given goal.
func (om *ObjectiveManager) GetObjectivesForGoal(ctx context.Context, goalID string) ([]*Objective, error) {
	// Find all edges of type "serves" targeting the goal
	edges, err := om.store.Edges().OfType("serves").ToNode(goalID).All()
	if err != nil {
		return nil, fmt.Errorf("failed to query objective-goal relationships: %w", err)
	}

	var objectives []*Objective
	for _, edge := range edges {
		// Check if the source is an objective
		node, err := om.store.GetNode(ctx, edge.SourceID)
		if err != nil || node.Type != "objective" {
			continue // Skip if not an objective or doesn't exist
		}

		objective, err := om.nodeToObjective(node)
		if err != nil {
			continue // Skip invalid objectives
		}
		objectives = append(objectives, objective)
	}

	return objectives, nil
}

// GetObjectivesUsingMethod returns all objectives that use the given method.
func (om *ObjectiveManager) GetObjectivesUsingMethod(ctx context.Context, methodID string) ([]*Objective, error) {
	// Find all edges of type "uses" targeting the method
	edges, err := om.store.Edges().OfType("uses").ToNode(methodID).All()
	if err != nil {
		return nil, fmt.Errorf("failed to query objective-method relationships: %w", err)
	}

	var objectives []*Objective
	for _, edge := range edges {
		// Check if the source is an objective
		node, err := om.store.GetNode(ctx, edge.SourceID)
		if err != nil || node.Type != "objective" {
			continue // Skip if not an objective or doesn't exist
		}

		objective, err := om.nodeToObjective(node)
		if err != nil {
			continue // Skip invalid objectives
		}
		objectives = append(objectives, objective)
	}

	return objectives, nil
}

// nodeToObjective converts a storage node to an Objective object.
func (om *ObjectiveManager) nodeToObjective(node *storage.Node) (*Objective, error) {
	if node == nil {
		return nil, fmt.Errorf("node is nil")
	}

	// Extract basic required fields
	goalID, ok := node.Data["goal_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing goal_id in objective node %s", node.ID)
	}

	methodID, ok := node.Data["method_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing method_id in objective node %s", node.ID)
	}

	title, ok := node.Data["title"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing title in objective node %s", node.ID)
	}

	statusStr, ok := node.Data["status"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing status in objective node %s", node.ID)
	}
	status := ObjectiveStatus(statusStr)

	createdAtStr, ok := node.Data["created_at"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing created_at in objective node %s", node.ID)
	}
	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("invalid created_at format in objective node %s: %w", node.ID, err)
	}

	// Extract optional fields
	description, _ := node.Data["description"].(string)

	// Handle priority - could be int or float64 from JSON
	var priority int
	switch v := node.Data["priority"].(type) {
	case float64:
		priority = int(v)
	case int:
		priority = v
	default:
		priority = 5 // Default priority if missing
	}

	context, _ := node.Data["context"].(map[string]interface{})
	if context == nil {
		context = make(map[string]interface{})
	}

	// Parse result data if present
	var result *ObjectiveResult
	if resultData, ok := node.Data["result"].(map[string]interface{}); ok && resultData != nil {
		result = &ObjectiveResult{}

		if success, ok := resultData["success"].(bool); ok {
			result.Success = success
		}

		if message, ok := resultData["message"].(string); ok {
			result.Message = message
		}

		if data, ok := resultData["data"].(map[string]interface{}); ok {
			result.Data = data
		}

		if tokensUsed, ok := resultData["tokens_used"].(float64); ok {
			result.TokensUsed = int(tokensUsed)
		} else if tokensUsed, ok := resultData["tokens_used"].(int); ok {
			result.TokensUsed = tokensUsed
		}

		if executionTimeStr, ok := resultData["execution_time"].(string); ok {
			if duration, err := time.ParseDuration(executionTimeStr); err == nil {
				result.ExecutionTime = duration
			}
		}

		if completedAtStr, ok := resultData["completed_at"].(string); ok {
			if completedAt, err := time.Parse(time.RFC3339, completedAtStr); err == nil {
				result.CompletedAt = completedAt
			}
		}
	}

	// Parse optional time fields
	var startedAt *time.Time
	if startedAtStr, ok := node.Data["started_at"].(string); ok && startedAtStr != "" {
		if parsed, err := time.Parse(time.RFC3339, startedAtStr); err == nil {
			startedAt = &parsed
		}
	}

	var completedAt *time.Time
	if completedAtStr, ok := node.Data["completed_at"].(string); ok && completedAtStr != "" {
		if parsed, err := time.Parse(time.RFC3339, completedAtStr); err == nil {
			completedAt = &parsed
		}
	}

	return &Objective{
		ID:          node.ID,
		GoalID:      goalID,
		MethodID:    methodID,
		Title:       title,
		Description: description,
		Status:      status,
		Context:     context,
		Result:      result,
		Priority:    priority,
		CreatedAt:   createdAt,
		StartedAt:   startedAt,
		CompletedAt: completedAt,
		store:       om.store,
	}, nil
}

// isValidObjectiveStatus checks if an objective status is valid.
func isValidObjectiveStatus(status ObjectiveStatus) bool {
	switch status {
	case ObjectiveStatusPending, ObjectiveStatusInProgress, ObjectiveStatusCompleted, ObjectiveStatusFailed, ObjectiveStatusPaused:
		return true
	default:
		return false
	}
}

// String returns a string representation of the objective status.
func (os ObjectiveStatus) String() string {
	return string(os)
}

// IsPending returns true if the objective has not started yet.
func (o *Objective) IsPending() bool {
	return o.Status == ObjectiveStatusPending
}

// IsInProgress returns true if the objective is actively being worked on.
func (o *Objective) IsInProgress() bool {
	return o.Status == ObjectiveStatusInProgress
}

// IsCompleted returns true if the objective has been successfully achieved.
func (o *Objective) IsCompleted() bool {
	return o.Status == ObjectiveStatusCompleted
}

// IsFailed returns true if the objective failed and cannot be completed.
func (o *Objective) IsFailed() bool {
	return o.Status == ObjectiveStatusFailed
}

// IsPaused returns true if the objective is temporarily paused.
func (o *Objective) IsPaused() bool {
	return o.Status == ObjectiveStatusPaused
}

// IsFinished returns true if the objective has completed (either success or failure).
func (o *Objective) IsFinished() bool {
	return o.Status == ObjectiveStatusCompleted || o.Status == ObjectiveStatusFailed
}

// Update provides a convenient way to update an objective through its instance.
func (o *Objective) Update(ctx context.Context, updates ObjectiveUpdates) error {
	if o.store == nil {
		return fmt.Errorf("objective is not connected to storage")
	}

	om := &ObjectiveManager{store: o.store}
	updatedObjective, err := om.UpdateObjective(ctx, o.ID, updates)
	if err != nil {
		return err
	}

	// Update this instance with the new values
	*o = *updatedObjective
	return nil
}

// Start begins work on this objective.
func (o *Objective) Start(ctx context.Context) error {
	if o.store == nil {
		return fmt.Errorf("objective is not connected to storage")
	}

	om := &ObjectiveManager{store: o.store}
	updatedObjective, err := om.StartObjective(ctx, o.ID)
	if err != nil {
		return err
	}

	// Update this instance with the new values
	*o = *updatedObjective
	return nil
}

// Complete marks this objective as finished with the given result.
func (o *Objective) Complete(ctx context.Context, result ObjectiveResult) error {
	if o.store == nil {
		return fmt.Errorf("objective is not connected to storage")
	}

	om := &ObjectiveManager{store: o.store}
	updatedObjective, err := om.CompleteObjective(ctx, o.ID, result)
	if err != nil {
		return err
	}

	// Update this instance with the new values
	*o = *updatedObjective
	return nil
}

// Fail marks this objective as failed with the given error information.
func (o *Objective) Fail(ctx context.Context, errorMessage string, tokensUsed int) error {
	if o.store == nil {
		return fmt.Errorf("objective is not connected to storage")
	}

	om := &ObjectiveManager{store: o.store}
	updatedObjective, err := om.FailObjective(ctx, o.ID, errorMessage, tokensUsed)
	if err != nil {
		return err
	}

	// Update this instance with the new values
	*o = *updatedObjective
	return nil
}

// Pause temporarily pauses work on this objective.
func (o *Objective) Pause(ctx context.Context) error {
	if o.store == nil {
		return fmt.Errorf("objective is not connected to storage")
	}

	om := &ObjectiveManager{store: o.store}
	updatedObjective, err := om.PauseObjective(ctx, o.ID)
	if err != nil {
		return err
	}

	// Update this instance with the new values
	*o = *updatedObjective
	return nil
}

// Resume resumes work on this paused objective.
func (o *Objective) Resume(ctx context.Context) error {
	if o.store == nil {
		return fmt.Errorf("objective is not connected to storage")
	}

	om := &ObjectiveManager{store: o.store}
	updatedObjective, err := om.ResumeObjective(ctx, o.ID)
	if err != nil {
		return err
	}

	// Update this instance with the new values
	*o = *updatedObjective
	return nil
}