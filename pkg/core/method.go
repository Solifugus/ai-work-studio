package core

import (
	"context"
	"fmt"
	"time"

	"github.com/Solifugus/ai-work-studio/pkg/storage"
)

// MethodDomain represents the scope/domain of a method's applicability.
type MethodDomain string

const (
	// MethodDomainGeneral indicates the method applies universally
	MethodDomainGeneral MethodDomain = "general"

	// MethodDomainSpecific indicates the method applies to a specific domain/context
	MethodDomainSpecific MethodDomain = "domain_specific"

	// MethodDomainUser indicates the method is specific to this user's patterns
	MethodDomainUser MethodDomain = "user_specific"
)

// MethodStatus represents the current state of a method.
type MethodStatus string

const (
	// MethodStatusActive indicates the method is actively used
	MethodStatusActive MethodStatus = "active"

	// MethodStatusDeprecated indicates the method is outdated but kept for reference
	MethodStatusDeprecated MethodStatus = "deprecated"

	// MethodStatusSuperseded indicates the method has been replaced by a newer version
	MethodStatusSuperseded MethodStatus = "superseded"
)

// ApproachStep represents a single step in a method's approach.
type ApproachStep struct {
	// Description explains what this step does
	Description string `json:"description"`

	// Tools lists the tools/capabilities needed for this step
	Tools []string `json:"tools,omitempty"`

	// Heuristics contains decision-making guidance for this step
	Heuristics []string `json:"heuristics,omitempty"`

	// Conditions specify when this step should be executed
	Conditions map[string]interface{} `json:"conditions,omitempty"`
}

// SuccessMetrics tracks how well a method performs over time.
type SuccessMetrics struct {
	// ExecutionCount tracks total number of times this method was used
	ExecutionCount int `json:"execution_count"`

	// SuccessCount tracks how many executions were successful
	SuccessCount int `json:"success_count"`

	// LastUsed is when this method was last executed
	LastUsed time.Time `json:"last_used"`

	// AverageRating is the mean user/system rating (1-10) of method effectiveness
	AverageRating float64 `json:"average_rating"`
}

// SuccessRate calculates the success percentage for this method.
func (sm *SuccessMetrics) SuccessRate() float64 {
	if sm.ExecutionCount == 0 {
		return 0.0
	}
	return float64(sm.SuccessCount) / float64(sm.ExecutionCount) * 100.0
}

// Method represents a proven approach for achieving objectives.
// Methods evolve over time through experience and are cached when successful.
type Method struct {
	// ID uniquely identifies this method across all versions
	ID string

	// Name is a short, descriptive title for the method
	Name string

	// Description provides detailed context about what this method accomplishes
	Description string

	// Approach contains the structured steps, tools, and heuristics
	Approach []ApproachStep

	// Domain indicates the scope of applicability
	Domain MethodDomain

	// Version tracks the evolution of this method (semantic versioning style)
	Version string

	// Status indicates whether the method is active, deprecated, or superseded
	Status MethodStatus

	// Metrics tracks success and usage statistics
	Metrics SuccessMetrics

	// UserContext contains additional contextual information specific to the user
	UserContext map[string]interface{}

	// CreatedAt is when this method version was originally created
	CreatedAt time.Time

	// store reference for database operations
	store *storage.Store
}

// MethodManager provides operations for managing methods in the storage system.
type MethodManager struct {
	store *storage.Store
}

// NewMethodManager creates a new manager for method operations.
func NewMethodManager(store *storage.Store) *MethodManager {
	return &MethodManager{
		store: store,
	}
}

// CreateMethod creates a new method and stores it in the system.
func (mm *MethodManager) CreateMethod(ctx context.Context, name, description string, approach []ApproachStep, domain MethodDomain, userContext map[string]interface{}) (*Method, error) {
	if name == "" {
		return nil, fmt.Errorf("method name cannot be empty")
	}
	if !isValidDomain(domain) {
		return nil, fmt.Errorf("invalid method domain: %s", domain)
	}

	now := time.Now()

	// Prepare approach data for storage
	approachData := make([]map[string]interface{}, len(approach))
	for i, step := range approach {
		approachData[i] = map[string]interface{}{
			"description": step.Description,
			"tools":       step.Tools,
			"heuristics":  step.Heuristics,
			"conditions":  step.Conditions,
		}
	}

	// Initialize empty success metrics
	metricsData := map[string]interface{}{
		"execution_count": 0,
		"success_count":   0,
		"last_used":       time.Time{}.Format(time.RFC3339), // Zero time for never used
		"average_rating":  0.0,
	}

	// Prepare data for storage node
	data := map[string]interface{}{
		"name":         name,
		"description":  description,
		"approach":     approachData,
		"domain":       string(domain),
		"version":      "1.0.0", // Initial version
		"status":       string(MethodStatusActive),
		"metrics":      metricsData,
		"user_context": userContext,
		"created_at":   now.Format(time.RFC3339),
	}

	// Create storage node
	node := storage.NewNode("method", data)

	// Store the node
	if err := mm.store.AddNode(ctx, node); err != nil {
		return nil, fmt.Errorf("failed to store method: %w", err)
	}

	// Return method object
	method := &Method{
		ID:          node.ID,
		Name:        name,
		Description: description,
		Approach:    approach,
		Domain:      domain,
		Version:     "1.0.0",
		Status:      MethodStatusActive,
		Metrics: SuccessMetrics{
			ExecutionCount: 0,
			SuccessCount:   0,
			LastUsed:       time.Time{},
			AverageRating:  0.0,
		},
		UserContext: userContext,
		CreatedAt:   now,
		store:       mm.store,
	}

	return method, nil
}

// GetMethod retrieves a method by ID.
func (mm *MethodManager) GetMethod(ctx context.Context, methodID string) (*Method, error) {
	node, err := mm.store.GetNode(ctx, methodID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve method %s: %w", methodID, err)
	}

	if node.Type != "method" {
		return nil, fmt.Errorf("node %s is not a method (type: %s)", methodID, node.Type)
	}

	return mm.nodeToMethod(node)
}

// GetMethodAtTime retrieves the version of a method that was active at the given time.
func (mm *MethodManager) GetMethodAtTime(ctx context.Context, methodID string, timestamp time.Time) (*Method, error) {
	node, err := mm.store.GetNodeAtTime(ctx, methodID, timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve method %s at time %v: %w", methodID, timestamp, err)
	}

	if node.Type != "method" {
		return nil, fmt.Errorf("node %s is not a method (type: %s)", methodID, node.Type)
	}

	return mm.nodeToMethod(node)
}

// UpdateMethod creates a new version of a method with updated information.
func (mm *MethodManager) UpdateMethod(ctx context.Context, methodID string, updates MethodUpdates) (*Method, error) {
	// Get current method to validate and provide defaults
	currentMethod, err := mm.GetMethod(ctx, methodID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current method for update: %w", err)
	}

	// Apply updates with defaults from current method
	name := currentMethod.Name
	if updates.Name != nil {
		name = *updates.Name
		if name == "" {
			return nil, fmt.Errorf("method name cannot be empty")
		}
	}

	description := currentMethod.Description
	if updates.Description != nil {
		description = *updates.Description
	}

	approach := currentMethod.Approach
	if updates.Approach != nil {
		approach = updates.Approach
	}

	domain := currentMethod.Domain
	if updates.Domain != nil {
		domain = *updates.Domain
		if !isValidDomain(domain) {
			return nil, fmt.Errorf("invalid method domain: %s", domain)
		}
	}

	version := currentMethod.Version
	if updates.Version != nil {
		version = *updates.Version
	}

	status := currentMethod.Status
	if updates.Status != nil {
		status = *updates.Status
		if !isValidMethodStatus(status) {
			return nil, fmt.Errorf("invalid method status: %s", status)
		}
	}

	metrics := currentMethod.Metrics
	if updates.Metrics != nil {
		metrics = *updates.Metrics
	}

	userContext := currentMethod.UserContext
	if updates.UserContext != nil {
		userContext = updates.UserContext
	}

	// Prepare approach data for storage
	approachData := make([]map[string]interface{}, len(approach))
	for i, step := range approach {
		approachData[i] = map[string]interface{}{
			"description": step.Description,
			"tools":       step.Tools,
			"heuristics":  step.Heuristics,
			"conditions":  step.Conditions,
		}
	}

	// Prepare metrics data for storage
	lastUsedStr := time.Time{}.Format(time.RFC3339)
	if !metrics.LastUsed.IsZero() {
		lastUsedStr = metrics.LastUsed.Format(time.RFC3339)
	}

	metricsData := map[string]interface{}{
		"execution_count": metrics.ExecutionCount,
		"success_count":   metrics.SuccessCount,
		"last_used":       lastUsedStr,
		"average_rating":  metrics.AverageRating,
	}

	// Prepare updated data
	data := map[string]interface{}{
		"name":         name,
		"description":  description,
		"approach":     approachData,
		"domain":       string(domain),
		"version":      version,
		"status":       string(status),
		"metrics":      metricsData,
		"user_context": userContext,
		"created_at":   currentMethod.CreatedAt.Format(time.RFC3339),
	}

	// Update in storage
	if err := mm.store.UpdateNode(ctx, methodID, data); err != nil {
		return nil, fmt.Errorf("failed to update method: %w", err)
	}

	// Return updated method
	return &Method{
		ID:          methodID,
		Name:        name,
		Description: description,
		Approach:    approach,
		Domain:      domain,
		Version:     version,
		Status:      status,
		Metrics:     metrics,
		UserContext: userContext,
		CreatedAt:   currentMethod.CreatedAt,
		store:       mm.store,
	}, nil
}

// MethodUpdates defines the fields that can be updated in a method.
// All fields are optional pointers to allow partial updates.
type MethodUpdates struct {
	Name        *string
	Description *string
	Approach    []ApproachStep
	Domain      *MethodDomain
	Version     *string
	Status      *MethodStatus
	Metrics     *SuccessMetrics
	UserContext map[string]interface{}
}

// ListMethods returns all methods with optional filtering.
func (mm *MethodManager) ListMethods(ctx context.Context, filter MethodFilter) ([]*Method, error) {
	query := mm.store.Nodes().OfType("method")

	// Apply domain filter if specified
	if filter.Domain != nil {
		query = query.WithData("domain", string(*filter.Domain))
	}

	// Apply status filter if specified
	if filter.Status != nil {
		query = query.WithData("status", string(*filter.Status))
	}

	nodes, err := query.All()
	if err != nil {
		return nil, fmt.Errorf("failed to query methods: %w", err)
	}

	var methods []*Method
	for _, node := range nodes {
		method, err := mm.nodeToMethod(node)
		if err != nil {
			continue // Skip invalid nodes
		}

		// Apply success rate filter in memory
		if filter.MinSuccessRate != nil && method.Metrics.SuccessRate() < *filter.MinSuccessRate {
			continue
		}

		methods = append(methods, method)
	}

	return methods, nil
}

// MethodFilter defines criteria for filtering methods.
type MethodFilter struct {
	Domain         *MethodDomain
	Status         *MethodStatus
	MinSuccessRate *float64 // Percentage (0-100)
}

// CreateMethodEvolution creates a new version of a method and establishes evolution relationship.
func (mm *MethodManager) CreateMethodEvolution(ctx context.Context, oldMethodID string, newMethod *Method, evolutionReason string) error {
	// Store the new method
	node := storage.NewNode("method", mm.methodToNodeData(newMethod))
	newMethod.ID = node.ID

	if err := mm.store.AddNode(ctx, node); err != nil {
		return fmt.Errorf("failed to store evolved method: %w", err)
	}

	// Create evolution edge: new method "evolved_from" old method
	edge := storage.NewEdge(newMethod.ID, oldMethodID, "evolved_from", map[string]interface{}{
		"reason":     evolutionReason,
		"created_at": time.Now().Format(time.RFC3339),
	})

	if err := mm.store.AddEdge(ctx, edge); err != nil {
		return fmt.Errorf("failed to create method evolution relationship: %w", err)
	}

	// Mark old method as superseded if it's still active
	oldMethod, err := mm.GetMethod(ctx, oldMethodID)
	if err == nil && oldMethod.Status == MethodStatusActive {
		updates := MethodUpdates{
			Status: &[]MethodStatus{MethodStatusSuperseded}[0],
		}
		_, err = mm.UpdateMethod(ctx, oldMethodID, updates)
		if err != nil {
			// Log but don't fail the evolution - the relationship is more important
			fmt.Printf("Warning: failed to mark old method as superseded: %v\n", err)
		}
	}

	return nil
}

// GetMethodEvolution returns the evolution chain for a method.
func (mm *MethodManager) GetMethodEvolution(ctx context.Context, methodID string) (*MethodEvolutionChain, error) {
	// Find predecessors (methods this evolved from)
	predecessorEdges, err := mm.store.Edges().OfType("evolved_from").FromNode(methodID).All()
	if err != nil {
		return nil, fmt.Errorf("failed to query method predecessors: %w", err)
	}

	var predecessors []*Method
	for _, edge := range predecessorEdges {
		predecessor, err := mm.GetMethod(ctx, edge.TargetID)
		if err != nil {
			continue // Skip if predecessor no longer exists
		}
		predecessors = append(predecessors, predecessor)
	}

	// Find successors (methods that evolved from this)
	successorEdges, err := mm.store.Edges().OfType("evolved_from").ToNode(methodID).All()
	if err != nil {
		return nil, fmt.Errorf("failed to query method successors: %w", err)
	}

	var successors []*Method
	for _, edge := range successorEdges {
		successor, err := mm.GetMethod(ctx, edge.SourceID)
		if err != nil {
			continue // Skip if successor no longer exists
		}
		successors = append(successors, successor)
	}

	// Get the current method
	currentMethod, err := mm.GetMethod(ctx, methodID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current method: %w", err)
	}

	return &MethodEvolutionChain{
		Current:     currentMethod,
		Predecessors: predecessors,
		Successors:   successors,
	}, nil
}

// MethodEvolutionChain represents the evolution history of a method.
type MethodEvolutionChain struct {
	Current      *Method
	Predecessors []*Method
	Successors   []*Method
}

// UpdateMethodMetrics updates the success metrics for a method based on execution results.
func (mm *MethodManager) UpdateMethodMetrics(ctx context.Context, methodID string, wasSuccessful bool, rating float64) error {
	method, err := mm.GetMethod(ctx, methodID)
	if err != nil {
		return fmt.Errorf("failed to get method for metrics update: %w", err)
	}

	// Update metrics
	newMetrics := method.Metrics
	newMetrics.ExecutionCount++
	if wasSuccessful {
		newMetrics.SuccessCount++
	}
	newMetrics.LastUsed = time.Now()

	// Update average rating using incremental formula
	if rating >= 1.0 && rating <= 10.0 {
		if newMetrics.ExecutionCount == 1 {
			newMetrics.AverageRating = rating
		} else {
			// Incremental average: new_avg = old_avg + (new_value - old_avg) / count
			newMetrics.AverageRating += (rating - newMetrics.AverageRating) / float64(newMetrics.ExecutionCount)
		}
	}

	// Update the method
	updates := MethodUpdates{
		Metrics: &newMetrics,
	}

	_, err = mm.UpdateMethod(ctx, methodID, updates)
	if err != nil {
		return fmt.Errorf("failed to update method metrics: %w", err)
	}

	return nil
}

// nodeToMethod converts a storage node to a Method object.
func (mm *MethodManager) nodeToMethod(node *storage.Node) (*Method, error) {
	if node == nil {
		return nil, fmt.Errorf("node is nil")
	}

	// Extract basic fields
	name, ok := node.Data["name"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing name in method node %s", node.ID)
	}

	description, _ := node.Data["description"].(string)

	domainStr, ok := node.Data["domain"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing domain in method node %s", node.ID)
	}
	domain := MethodDomain(domainStr)

	version, _ := node.Data["version"].(string)

	statusStr, ok := node.Data["status"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing status in method node %s", node.ID)
	}
	status := MethodStatus(statusStr)

	userContext, _ := node.Data["user_context"].(map[string]interface{})

	createdAtStr, ok := node.Data["created_at"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing created_at in method node %s", node.ID)
	}
	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("invalid created_at format in method node %s: %w", node.ID, err)
	}

	// Parse approach data
	var approach []ApproachStep
	if approachData, ok := node.Data["approach"].([]interface{}); ok {
		for _, stepData := range approachData {
			if stepMap, ok := stepData.(map[string]interface{}); ok {
				step := ApproachStep{
					Description: stepMap["description"].(string),
				}
				if tools, ok := stepMap["tools"].([]interface{}); ok {
					step.Tools = interfaceSliceToStringSlice(tools)
				}
				if heuristics, ok := stepMap["heuristics"].([]interface{}); ok {
					step.Heuristics = interfaceSliceToStringSlice(heuristics)
				}
				if conditions, ok := stepMap["conditions"].(map[string]interface{}); ok {
					step.Conditions = conditions
				}
				approach = append(approach, step)
			}
		}
	}

	// Parse metrics data
	var metrics SuccessMetrics
	if metricsData, ok := node.Data["metrics"].(map[string]interface{}); ok {
		// Handle execution count - could be int or float64 from JSON
		if execCountVal := metricsData["execution_count"]; execCountVal != nil {
			switch v := execCountVal.(type) {
			case float64:
				metrics.ExecutionCount = int(v)
			case int:
				metrics.ExecutionCount = v
			}
		}

		// Handle success count - could be int or float64 from JSON
		if successCountVal := metricsData["success_count"]; successCountVal != nil {
			switch v := successCountVal.(type) {
			case float64:
				metrics.SuccessCount = int(v)
			case int:
				metrics.SuccessCount = v
			}
		}

		if avgRating, ok := metricsData["average_rating"].(float64); ok {
			metrics.AverageRating = avgRating
		}
		if lastUsedStr, ok := metricsData["last_used"].(string); ok {
			lastUsed, _ := time.Parse(time.RFC3339, lastUsedStr)
			metrics.LastUsed = lastUsed
		}
	}

	return &Method{
		ID:          node.ID,
		Name:        name,
		Description: description,
		Approach:    approach,
		Domain:      domain,
		Version:     version,
		Status:      status,
		Metrics:     metrics,
		UserContext: userContext,
		CreatedAt:   createdAt,
		store:       mm.store,
	}, nil
}

// methodToNodeData converts a Method object to storage node data.
func (mm *MethodManager) methodToNodeData(method *Method) map[string]interface{} {
	// Prepare approach data for storage
	approachData := make([]map[string]interface{}, len(method.Approach))
	for i, step := range method.Approach {
		approachData[i] = map[string]interface{}{
			"description": step.Description,
			"tools":       step.Tools,
			"heuristics":  step.Heuristics,
			"conditions":  step.Conditions,
		}
	}

	// Prepare metrics data for storage
	lastUsedStr := time.Time{}.Format(time.RFC3339)
	if !method.Metrics.LastUsed.IsZero() {
		lastUsedStr = method.Metrics.LastUsed.Format(time.RFC3339)
	}

	metricsData := map[string]interface{}{
		"execution_count": method.Metrics.ExecutionCount,
		"success_count":   method.Metrics.SuccessCount,
		"last_used":       lastUsedStr,
		"average_rating":  method.Metrics.AverageRating,
	}

	return map[string]interface{}{
		"name":         method.Name,
		"description":  method.Description,
		"approach":     approachData,
		"domain":       string(method.Domain),
		"version":      method.Version,
		"status":       string(method.Status),
		"metrics":      metricsData,
		"user_context": method.UserContext,
		"created_at":   method.CreatedAt.Format(time.RFC3339),
	}
}

// Helper function to convert []interface{} to []string
func interfaceSliceToStringSlice(slice []interface{}) []string {
	strings := make([]string, 0, len(slice))
	for _, item := range slice {
		if str, ok := item.(string); ok {
			strings = append(strings, str)
		}
	}
	return strings
}

// isValidDomain checks if a method domain is valid.
func isValidDomain(domain MethodDomain) bool {
	switch domain {
	case MethodDomainGeneral, MethodDomainSpecific, MethodDomainUser:
		return true
	default:
		return false
	}
}

// isValidMethodStatus checks if a method status is valid.
func isValidMethodStatus(status MethodStatus) bool {
	switch status {
	case MethodStatusActive, MethodStatusDeprecated, MethodStatusSuperseded:
		return true
	default:
		return false
	}
}

// String returns a string representation of the method domain.
func (md MethodDomain) String() string {
	return string(md)
}

// String returns a string representation of the method status.
func (ms MethodStatus) String() string {
	return string(ms)
}

// IsActive returns true if the method is actively available for use.
func (m *Method) IsActive() bool {
	return m.Status == MethodStatusActive
}

// IsDeprecated returns true if the method is deprecated but still available.
func (m *Method) IsDeprecated() bool {
	return m.Status == MethodStatusDeprecated
}

// Update provides a convenient way to update a method through its instance.
func (m *Method) Update(ctx context.Context, updates MethodUpdates) error {
	if m.store == nil {
		return fmt.Errorf("method is not connected to storage")
	}

	mm := &MethodManager{store: m.store}
	updatedMethod, err := mm.UpdateMethod(ctx, m.ID, updates)
	if err != nil {
		return err
	}

	// Update this instance with the new values
	*m = *updatedMethod
	return nil
}

// RecordExecution updates the method's success metrics based on an execution result.
func (m *Method) RecordExecution(ctx context.Context, wasSuccessful bool, rating float64) error {
	if m.store == nil {
		return fmt.Errorf("method is not connected to storage")
	}

	mm := &MethodManager{store: m.store}
	return mm.UpdateMethodMetrics(ctx, m.ID, wasSuccessful, rating)
}