package core

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/Solifugus/ai-work-studio/pkg/storage"
)

// ContextCategory represents the type of user context being stored.
type ContextCategory string

const (
	// ContextCategoryPreferences tracks user's preferred ways of working
	ContextCategoryPreferences ContextCategory = "preferences"

	// ContextCategoryPatterns tracks user's recurring behavioral patterns
	ContextCategoryPatterns ContextCategory = "patterns"

	// ContextCategoryValues tracks user's core values and principles
	ContextCategoryValues ContextCategory = "values"

	// ContextCategoryConstraints tracks user's limitations and boundaries
	ContextCategoryConstraints ContextCategory = "constraints"

	// ContextCategoryDomainExpertise tracks user's knowledge areas and skill levels
	ContextCategoryDomainExpertise ContextCategory = "domain_expertise"
)

// ContextSource represents how the context was learned.
type ContextSource string

const (
	// ContextSourceExplicit means user directly stated this context
	ContextSourceExplicit ContextSource = "explicit"

	// ContextSourceInferred means we deduced this from user's behavior
	ContextSourceInferred ContextSource = "inferred"

	// ContextSourceFeedback means we learned this from user corrections/feedback
	ContextSourceFeedback ContextSource = "feedback"
)

// UserContext represents learned information about the user that informs
// system judgment and method selection. Context evolves temporally and
// includes confidence scoring for reliability.
type UserContext struct {
	// ID uniquely identifies this context entry
	ID string

	// Category classifies the type of context
	Category ContextCategory

	// Content contains the actual context information
	Content string

	// Source indicates how this context was learned
	Source ContextSource

	// Confidence represents how certain we are about this context (0.0-1.0)
	Confidence float64

	// RelevanceTags are keywords used for context retrieval and matching
	RelevanceTags []string

	// LastValidated tracks when this context was last confirmed or used
	LastValidated time.Time

	// CreatedAt is when this context was originally learned
	CreatedAt time.Time

	// UserID identifies which user this context belongs to (for future multi-user)
	UserID string

	// store reference for database operations
	store *storage.Store
}

// UserContextManager provides operations for managing user context in the storage system.
type UserContextManager struct {
	store *storage.Store

	// Configuration for temporal confidence decay
	confidenceDecayRate float64 // How much confidence decreases per day
	minConfidence      float64 // Minimum confidence before context is considered stale
}

// NewUserContextManager creates a new manager for user context operations.
func NewUserContextManager(store *storage.Store) *UserContextManager {
	return &UserContextManager{
		store:               store,
		confidenceDecayRate: 0.01, // 1% decay per day
		minConfidence:      0.1,   // 10% minimum confidence
	}
}

// LearnContext creates a new context entry and stores it in the system.
func (ucm *UserContextManager) LearnContext(ctx context.Context, category ContextCategory, content string, source ContextSource, relevanceTags []string, userID string) (*UserContext, error) {
	if content == "" {
		return nil, fmt.Errorf("context content cannot be empty")
	}

	if !isValidCategory(category) {
		return nil, fmt.Errorf("invalid context category: %s", category)
	}

	if !isValidSource(source) {
		return nil, fmt.Errorf("invalid context source: %s", source)
	}

	now := time.Now()

	// Initial confidence based on source reliability
	confidence := ucm.getInitialConfidence(source)

	// Prepare data for storage node
	data := map[string]interface{}{
		"category":       string(category),
		"content":        content,
		"source":         string(source),
		"confidence":     confidence,
		"relevance_tags": relevanceTags,
		"last_validated": now.Format(time.RFC3339),
		"created_at":     now.Format(time.RFC3339),
		"user_id":        userID,
	}

	// Create storage node
	node := storage.NewNode("user_context", data)

	// Store the node
	if err := ucm.store.AddNode(ctx, node); err != nil {
		return nil, fmt.Errorf("failed to store user context: %w", err)
	}

	// Return context object
	userContext := &UserContext{
		ID:            node.ID,
		Category:      category,
		Content:       content,
		Source:        source,
		Confidence:    confidence,
		RelevanceTags: relevanceTags,
		LastValidated: now,
		CreatedAt:     now,
		UserID:        userID,
		store:         ucm.store,
	}

	return userContext, nil
}

// GetContext retrieves a context entry by ID.
func (ucm *UserContextManager) GetContext(ctx context.Context, contextID string) (*UserContext, error) {
	node, err := ucm.store.GetNode(ctx, contextID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve context %s: %w", contextID, err)
	}

	if node.Type != "user_context" {
		return nil, fmt.Errorf("node %s is not a user context (type: %s)", contextID, node.Type)
	}

	userContext, err := ucm.nodeToUserContext(node)
	if err != nil {
		return nil, err
	}

	// Apply temporal confidence decay
	userContext.Confidence = ucm.applyConfidenceDecay(userContext.Confidence, userContext.LastValidated)

	return userContext, nil
}

// UpdateContext creates a new version of a context entry with updated information.
func (ucm *UserContextManager) UpdateContext(ctx context.Context, contextID string, updates UserContextUpdates) (*UserContext, error) {
	// Get current context to validate and provide defaults
	currentContext, err := ucm.GetContext(ctx, contextID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current context for update: %w", err)
	}

	// Apply updates with defaults from current context
	category := currentContext.Category
	if updates.Category != nil {
		category = *updates.Category
		if !isValidCategory(category) {
			return nil, fmt.Errorf("invalid context category: %s", category)
		}
	}

	content := currentContext.Content
	if updates.Content != nil {
		content = *updates.Content
		if content == "" {
			return nil, fmt.Errorf("context content cannot be empty")
		}
	}

	source := currentContext.Source
	if updates.Source != nil {
		source = *updates.Source
		if !isValidSource(source) {
			return nil, fmt.Errorf("invalid context source: %s", source)
		}
	}

	confidence := currentContext.Confidence
	if updates.Confidence != nil {
		confidence = *updates.Confidence
		if confidence < 0.0 || confidence > 1.0 {
			return nil, fmt.Errorf("confidence must be between 0.0 and 1.0, got %f", confidence)
		}
	}

	relevanceTags := currentContext.RelevanceTags
	if updates.RelevanceTags != nil {
		relevanceTags = updates.RelevanceTags
	}

	// Update validation time
	now := time.Now()

	// Prepare updated data
	data := map[string]interface{}{
		"category":       string(category),
		"content":        content,
		"source":         string(source),
		"confidence":     confidence,
		"relevance_tags": relevanceTags,
		"last_validated": now.Format(time.RFC3339),
		"created_at":     currentContext.CreatedAt.Format(time.RFC3339),
		"user_id":        currentContext.UserID,
	}

	// Update in storage
	if err := ucm.store.UpdateNode(ctx, contextID, data); err != nil {
		return nil, fmt.Errorf("failed to update context: %w", err)
	}

	// Return updated context
	return &UserContext{
		ID:            contextID,
		Category:      category,
		Content:       content,
		Source:        source,
		Confidence:    confidence,
		RelevanceTags: relevanceTags,
		LastValidated: now,
		CreatedAt:     currentContext.CreatedAt,
		UserID:        currentContext.UserID,
		store:         ucm.store,
	}, nil
}

// UserContextUpdates defines the fields that can be updated in a user context.
// All fields are optional pointers to allow partial updates.
type UserContextUpdates struct {
	Category      *ContextCategory
	Content       *string
	Source        *ContextSource
	Confidence    *float64
	RelevanceTags []string
}

// GetRelevantContext retrieves context entries relevant to the given objective.
// Results are ranked by relevance score combining confidence and tag matching.
func (ucm *UserContextManager) GetRelevantContext(ctx context.Context, objectiveText string, userID string, limit int) ([]*UserContext, error) {
	if limit <= 0 {
		limit = 10 // Default limit
	}

	// Get all user contexts for this user
	query := ucm.store.Nodes().OfType("user_context")
	if userID != "" {
		query = query.WithData("user_id", userID)
	}

	nodes, err := query.All()
	if err != nil {
		return nil, fmt.Errorf("failed to query user contexts: %w", err)
	}

	var contexts []*UserContext
	for _, node := range nodes {
		userContext, err := ucm.nodeToUserContext(node)
		if err != nil {
			continue // Skip invalid nodes
		}

		// Apply temporal confidence decay
		userContext.Confidence = ucm.applyConfidenceDecay(userContext.Confidence, userContext.LastValidated)

		// Skip contexts with very low confidence
		if userContext.Confidence < ucm.minConfidence {
			continue
		}

		contexts = append(contexts, userContext)
	}

	// Score and sort by relevance
	scoredContexts := ucm.scoreContexts(contexts, objectiveText)

	// Sort by relevance score (descending)
	sort.Slice(scoredContexts, func(i, j int) bool {
		return scoredContexts[i].RelevanceScore > scoredContexts[j].RelevanceScore
	})

	// Return top results
	var results []*UserContext
	for i, sc := range scoredContexts {
		if i >= limit {
			break
		}
		results = append(results, sc.Context)
	}

	return results, nil
}

// ScoredContext holds a context and its calculated relevance score.
type ScoredContext struct {
	Context        *UserContext
	RelevanceScore float64
}

// GetContextByCategory retrieves all context entries of a specific category for a user.
func (ucm *UserContextManager) GetContextByCategory(ctx context.Context, category ContextCategory, userID string) ([]*UserContext, error) {
	query := ucm.store.Nodes().OfType("user_context").WithData("category", string(category))
	if userID != "" {
		query = query.WithData("user_id", userID)
	}

	nodes, err := query.All()
	if err != nil {
		return nil, fmt.Errorf("failed to query contexts by category: %w", err)
	}

	var contexts []*UserContext
	for _, node := range nodes {
		userContext, err := ucm.nodeToUserContext(node)
		if err != nil {
			continue // Skip invalid nodes
		}

		// Apply temporal confidence decay
		userContext.Confidence = ucm.applyConfidenceDecay(userContext.Confidence, userContext.LastValidated)

		contexts = append(contexts, userContext)
	}

	// Sort by confidence (descending)
	sort.Slice(contexts, func(i, j int) bool {
		return contexts[i].Confidence > contexts[j].Confidence
	})

	return contexts, nil
}

// ValidateContext marks a context entry as recently validated, boosting its confidence.
func (ucm *UserContextManager) ValidateContext(ctx context.Context, contextID string, boost float64) error {
	currentContext, err := ucm.GetContext(ctx, contextID)
	if err != nil {
		return fmt.Errorf("failed to get context for validation: %w", err)
	}

	// Apply confidence boost, capped at 1.0
	newConfidence := math.Min(currentContext.Confidence+boost, 1.0)

	updates := UserContextUpdates{
		Confidence: &newConfidence,
	}

	_, err = ucm.UpdateContext(ctx, contextID, updates)
	return err
}

// scoreContexts calculates relevance scores for contexts against an objective.
func (ucm *UserContextManager) scoreContexts(contexts []*UserContext, objectiveText string) []ScoredContext {
	objectiveWords := strings.Fields(strings.ToLower(objectiveText))

	var scoredContexts []ScoredContext

	for _, context := range contexts {
		score := ucm.calculateRelevanceScore(context, objectiveWords)
		scoredContexts = append(scoredContexts, ScoredContext{
			Context:        context,
			RelevanceScore: score,
		})
	}

	return scoredContexts
}

// calculateRelevanceScore computes a relevance score based on confidence and keyword matching.
func (ucm *UserContextManager) calculateRelevanceScore(context *UserContext, objectiveWords []string) float64 {
	// Base score is the confidence
	score := context.Confidence

	// Content matching score
	contextWords := strings.Fields(strings.ToLower(context.Content))
	contentMatchScore := ucm.calculateWordMatchScore(contextWords, objectiveWords)

	// Tag matching score (tags are more specific, so weight higher)
	tagWords := make([]string, 0)
	for _, tag := range context.RelevanceTags {
		tagWords = append(tagWords, strings.ToLower(tag))
	}
	tagMatchScore := ucm.calculateWordMatchScore(tagWords, objectiveWords) * 2.0

	// Category bonus - some categories are more immediately relevant
	categoryBonus := ucm.getCategoryRelevanceBonus(context.Category)

	// Combine scores: confidence * (1 + content_match + tag_match + category_bonus)
	totalScore := score * (1.0 + contentMatchScore + tagMatchScore + categoryBonus)

	return totalScore
}

// calculateWordMatchScore computes how well two sets of words match.
func (ucm *UserContextManager) calculateWordMatchScore(contextWords, objectiveWords []string) float64 {
	if len(contextWords) == 0 || len(objectiveWords) == 0 {
		return 0.0
	}

	matches := 0
	for _, objWord := range objectiveWords {
		for _, ctxWord := range contextWords {
			// Exact match or substring match
			if objWord == ctxWord || strings.Contains(ctxWord, objWord) || strings.Contains(objWord, ctxWord) {
				matches++
				break
			}
		}
	}

	return float64(matches) / float64(len(objectiveWords))
}

// getCategoryRelevanceBonus provides category-specific relevance weighting.
func (ucm *UserContextManager) getCategoryRelevanceBonus(category ContextCategory) float64 {
	switch category {
	case ContextCategoryConstraints:
		return 0.3 // Constraints are very relevant - need to respect limits
	case ContextCategoryPreferences:
		return 0.2 // Preferences are highly relevant for method selection
	case ContextCategoryPatterns:
		return 0.15 // Patterns inform approach selection
	case ContextCategoryValues:
		return 0.1 // Values guide decision-making
	case ContextCategoryDomainExpertise:
		return 0.1 // Expertise informs complexity of approach
	default:
		return 0.0
	}
}

// getInitialConfidence returns initial confidence based on source reliability.
func (ucm *UserContextManager) getInitialConfidence(source ContextSource) float64 {
	switch source {
	case ContextSourceExplicit:
		return 0.9 // High confidence - user directly stated this
	case ContextSourceFeedback:
		return 0.8 // High confidence - user corrected us about this
	case ContextSourceInferred:
		return 0.6 // Medium confidence - we deduced this from behavior
	default:
		return 0.5 // Default medium confidence
	}
}

// applyConfidenceDecay reduces confidence over time since last validation.
func (ucm *UserContextManager) applyConfidenceDecay(currentConfidence float64, lastValidated time.Time) float64 {
	daysSinceValidation := time.Since(lastValidated).Hours() / 24
	decay := daysSinceValidation * ucm.confidenceDecayRate
	newConfidence := math.Max(currentConfidence-decay, 0.0)
	return newConfidence
}

// nodeToUserContext converts a storage node to a UserContext object.
func (ucm *UserContextManager) nodeToUserContext(node *storage.Node) (*UserContext, error) {
	if node == nil {
		return nil, fmt.Errorf("node is nil")
	}

	// Extract fields from node data
	categoryStr, ok := node.Data["category"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing category in context node %s", node.ID)
	}
	category := ContextCategory(categoryStr)

	content, ok := node.Data["content"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing content in context node %s", node.ID)
	}

	sourceStr, ok := node.Data["source"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing source in context node %s", node.ID)
	}
	source := ContextSource(sourceStr)

	confidence, ok := node.Data["confidence"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid or missing confidence in context node %s", node.ID)
	}

	// Handle relevance tags (could be stored as interface{})
	var relevanceTags []string
	if tags, exists := node.Data["relevance_tags"]; exists {
		if tagSlice, ok := tags.([]interface{}); ok {
			for _, tag := range tagSlice {
				if tagStr, ok := tag.(string); ok {
					relevanceTags = append(relevanceTags, tagStr)
				}
			}
		}
	}

	lastValidatedStr, ok := node.Data["last_validated"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing last_validated in context node %s", node.ID)
	}
	lastValidated, err := time.Parse(time.RFC3339, lastValidatedStr)
	if err != nil {
		return nil, fmt.Errorf("invalid last_validated format in context node %s: %w", node.ID, err)
	}

	createdAtStr, ok := node.Data["created_at"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing created_at in context node %s", node.ID)
	}
	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("invalid created_at format in context node %s: %w", node.ID, err)
	}

	userID, _ := node.Data["user_id"].(string) // Optional field

	return &UserContext{
		ID:            node.ID,
		Category:      category,
		Content:       content,
		Source:        source,
		Confidence:    confidence,
		RelevanceTags: relevanceTags,
		LastValidated: lastValidated,
		CreatedAt:     createdAt,
		UserID:        userID,
		store:         ucm.store,
	}, nil
}

// isValidCategory checks if a context category is valid.
func isValidCategory(category ContextCategory) bool {
	switch category {
	case ContextCategoryPreferences, ContextCategoryPatterns, ContextCategoryValues,
		 ContextCategoryConstraints, ContextCategoryDomainExpertise:
		return true
	default:
		return false
	}
}

// isValidSource checks if a context source is valid.
func isValidSource(source ContextSource) bool {
	switch source {
	case ContextSourceExplicit, ContextSourceInferred, ContextSourceFeedback:
		return true
	default:
		return false
	}
}

// String returns a string representation of the context category.
func (cc ContextCategory) String() string {
	return string(cc)
}

// String returns a string representation of the context source.
func (cs ContextSource) String() string {
	return string(cs)
}

// Update provides a convenient way to update a context through its instance.
func (uc *UserContext) Update(ctx context.Context, updates UserContextUpdates) error {
	if uc.store == nil {
		return fmt.Errorf("context is not connected to storage")
	}

	ucm := &UserContextManager{store: uc.store}
	updatedContext, err := ucm.UpdateContext(ctx, uc.ID, updates)
	if err != nil {
		return err
	}

	// Update this instance with the new values
	*uc = *updatedContext
	return nil
}

// Validate provides a convenient way to validate a context through its instance.
func (uc *UserContext) Validate(ctx context.Context, boost float64) error {
	if uc.store == nil {
		return fmt.Errorf("context is not connected to storage")
	}

	ucm := &UserContextManager{store: uc.store}
	return ucm.ValidateContext(ctx, uc.ID, boost)
}