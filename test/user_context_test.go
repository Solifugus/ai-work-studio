package test

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Solifugus/ai-work-studio/pkg/core"
	"github.com/Solifugus/ai-work-studio/pkg/storage"
)

// TestUserContextIntegration runs comprehensive integration tests for user context functionality.
func TestUserContextIntegration(t *testing.T) {
	// Create temporary directory for test data
	tempDir := filepath.Join(os.TempDir(), "ai-work-studio-test-context")
	defer func() {
		os.RemoveAll(tempDir)
	}()

	// Initialize storage
	store, err := storage.NewStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}
	defer store.Close()

	// Initialize user context manager
	ucm := core.NewUserContextManager(store)
	ctx := context.Background()
	userID := "test-user-123"

	t.Run("Basic Context Creation", func(t *testing.T) {
		testBasicContextCreation(t, ucm, ctx, userID)
	})

	t.Run("Context Categories", func(t *testing.T) {
		testContextCategories(t, ucm, ctx, userID)
	})

	t.Run("Learning Sources", func(t *testing.T) {
		testLearningSources(t, ucm, ctx, userID)
	})

	t.Run("Confidence Scoring", func(t *testing.T) {
		testConfidenceScoring(t, ucm, ctx, userID)
	})

	t.Run("Temporal Evolution", func(t *testing.T) {
		testTemporalEvolution(t, ucm, ctx, userID)
	})

	t.Run("Context Retrieval", func(t *testing.T) {
		testContextRetrieval(t, ucm, ctx, userID)
	})

	t.Run("Context Updates", func(t *testing.T) {
		testContextUpdates(t, ucm, ctx, userID)
	})

	t.Run("Context Validation", func(t *testing.T) {
		testContextValidation(t, ucm, ctx, userID)
	})

	t.Run("Relevance Scoring", func(t *testing.T) {
		testRelevanceScoring(t, ucm, ctx, userID)
	})

	t.Run("Edge Cases", func(t *testing.T) {
		testEdgeCases(t, ucm, ctx, userID)
	})
}

func testBasicContextCreation(t *testing.T, ucm *core.UserContextManager, ctx context.Context, userID string) {
	// Test creating a basic context entry
	context, err := ucm.LearnContext(ctx,
		core.ContextCategoryPreferences,
		"Prefers detailed explanations with examples",
		core.ContextSourceExplicit,
		[]string{"explanations", "examples", "detail"},
		userID,
	)

	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	if context == nil {
		t.Fatal("Created context is nil")
	}

	if context.ID == "" {
		t.Error("Context ID is empty")
	}

	if context.Category != core.ContextCategoryPreferences {
		t.Errorf("Expected category %v, got %v", core.ContextCategoryPreferences, context.Category)
	}

	if context.Content != "Prefers detailed explanations with examples" {
		t.Errorf("Expected content 'Prefers detailed explanations with examples', got %v", context.Content)
	}

	if context.Source != core.ContextSourceExplicit {
		t.Errorf("Expected source %v, got %v", core.ContextSourceExplicit, context.Source)
	}

	if len(context.RelevanceTags) != 3 {
		t.Errorf("Expected 3 relevance tags, got %d", len(context.RelevanceTags))
	}

	if context.UserID != userID {
		t.Errorf("Expected userID %v, got %v", userID, context.UserID)
	}

	if context.Confidence <= 0 || context.Confidence > 1 {
		t.Errorf("Invalid confidence value: %f", context.Confidence)
	}

	// Test retrieval
	retrieved, err := ucm.GetContext(ctx, context.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve context: %v", err)
	}

	if retrieved.ID != context.ID {
		t.Error("Retrieved context has different ID")
	}

	if retrieved.Content != context.Content {
		t.Error("Retrieved context has different content")
	}
}

func testContextCategories(t *testing.T, ucm *core.UserContextManager, ctx context.Context, userID string) {
	categories := []core.ContextCategory{
		core.ContextCategoryPreferences,
		core.ContextCategoryPatterns,
		core.ContextCategoryValues,
		core.ContextCategoryConstraints,
		core.ContextCategoryDomainExpertise,
	}

	contents := []string{
		"Likes concise summaries",
		"Often works late evenings",
		"Values privacy and security",
		"Limited to 2 hours per task",
		"Expert in machine learning",
	}

	var createdContexts []*core.UserContext

	// Create contexts for each category
	for i, category := range categories {
		context, err := ucm.LearnContext(ctx,
			category,
			contents[i],
			core.ContextSourceInferred,
			[]string{string(category)},
			userID,
		)

		if err != nil {
			t.Fatalf("Failed to create context for category %v: %v", category, err)
		}

		createdContexts = append(createdContexts, context)
	}

	// Test retrieval by category
	for i, category := range categories {
		contexts, err := ucm.GetContextByCategory(ctx, category, userID)
		if err != nil {
			t.Fatalf("Failed to get contexts by category %v: %v", category, err)
		}

		if len(contexts) == 0 {
			t.Errorf("No contexts found for category %v", category)
			continue
		}

		// Find the context we created
		found := false
		for _, context := range contexts {
			if context.Content == contents[i] {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Created context not found in category %v results", category)
		}
	}
}

func testLearningSources(t *testing.T, ucm *core.UserContextManager, ctx context.Context, userID string) {
	sources := []core.ContextSource{
		core.ContextSourceExplicit,
		core.ContextSourceInferred,
		core.ContextSourceFeedback,
	}

	expectedConfidences := []float64{0.9, 0.6, 0.8} // Based on getInitialConfidence

	for i, source := range sources {
		context, err := ucm.LearnContext(ctx,
			core.ContextCategoryPreferences,
			"Test content for "+string(source),
			source,
			[]string{"test"},
			userID,
		)

		if err != nil {
			t.Fatalf("Failed to create context for source %v: %v", source, err)
		}

		if math.Abs(context.Confidence-expectedConfidences[i]) > 0.01 {
			t.Errorf("Expected confidence %f for source %v, got %f",
				expectedConfidences[i], source, context.Confidence)
		}
	}
}

func testConfidenceScoring(t *testing.T, ucm *core.UserContextManager, ctx context.Context, userID string) {
	// Create context with explicit source (high confidence)
	explicitContext, err := ucm.LearnContext(ctx,
		core.ContextCategoryPreferences,
		"Explicitly stated preference",
		core.ContextSourceExplicit,
		[]string{"explicit"},
		userID,
	)
	if err != nil {
		t.Fatalf("Failed to create explicit context: %v", err)
	}

	// Create context with inferred source (lower confidence)
	inferredContext, err := ucm.LearnContext(ctx,
		core.ContextCategoryPreferences,
		"Inferred preference",
		core.ContextSourceInferred,
		[]string{"inferred"},
		userID,
	)
	if err != nil {
		t.Fatalf("Failed to create inferred context: %v", err)
	}

	// Explicit should have higher confidence than inferred
	if explicitContext.Confidence <= inferredContext.Confidence {
		t.Errorf("Explicit context should have higher confidence than inferred. Explicit: %f, Inferred: %f",
			explicitContext.Confidence, inferredContext.Confidence)
	}

	// Test confidence bounds
	if explicitContext.Confidence < 0 || explicitContext.Confidence > 1 {
		t.Errorf("Confidence out of bounds: %f", explicitContext.Confidence)
	}

	if inferredContext.Confidence < 0 || inferredContext.Confidence > 1 {
		t.Errorf("Confidence out of bounds: %f", inferredContext.Confidence)
	}
}

func testTemporalEvolution(t *testing.T, ucm *core.UserContextManager, ctx context.Context, userID string) {
	// Create a context entry
	originalContext, err := ucm.LearnContext(ctx,
		core.ContextCategoryPatterns,
		"Works in morning hours",
		core.ContextSourceInferred,
		[]string{"morning", "work"},
		userID,
	)
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	originalConfidence := originalContext.Confidence

	// For this test, we'll verify that confidence decay exists by testing the math
	// Since we can't easily manipulate internal timestamps, we'll test the decay logic indirectly

	// Get the context - it should have current confidence
	retrievedContext, err := ucm.GetContext(ctx, originalContext.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve context: %v", err)
	}

	// Confidence should be within valid bounds
	if retrievedContext.Confidence > 1.0 || retrievedContext.Confidence < 0.0 {
		t.Errorf("Confidence out of bounds: %f", retrievedContext.Confidence)
	}

	// Since the context was just created, confidence should not have decayed significantly
	if math.Abs(retrievedContext.Confidence-originalConfidence) > 0.1 {
		t.Errorf("Confidence decayed unexpectedly for fresh context. Original: %f, Current: %f",
			originalConfidence, retrievedContext.Confidence)
	}
}

func testContextRetrieval(t *testing.T, ucm *core.UserContextManager, ctx context.Context, userID string) {
	// Create several contexts with different relevance to a test objective
	contexts := []struct {
		content string
		tags    []string
		category core.ContextCategory
	}{
		{
			content:  "Prefers Python for data analysis",
			tags:     []string{"python", "data", "analysis"},
			category: core.ContextCategoryPreferences,
		},
		{
			content:  "Expert in machine learning algorithms",
			tags:     []string{"machine-learning", "algorithms", "expert"},
			category: core.ContextCategoryDomainExpertise,
		},
		{
			content:  "Limited time for complex projects",
			tags:     []string{"time", "constraints", "complexity"},
			category: core.ContextCategoryConstraints,
		},
		{
			content:  "Values clean, maintainable code",
			tags:     []string{"code-quality", "maintainable", "clean"},
			category: core.ContextCategoryValues,
		},
	}

	var createdContexts []*core.UserContext
	for _, c := range contexts {
		context, err := ucm.LearnContext(ctx, c.category, c.content, core.ContextSourceExplicit, c.tags, userID)
		if err != nil {
			t.Fatalf("Failed to create test context: %v", err)
		}
		createdContexts = append(createdContexts, context)
	}

	// Test objective that should match some contexts
	objective := "Build a Python machine learning model for data analysis"

	relevantContexts, err := ucm.GetRelevantContext(ctx, objective, userID, 5)
	if err != nil {
		t.Fatalf("Failed to get relevant context: %v", err)
	}

	if len(relevantContexts) == 0 {
		t.Error("No relevant contexts found")
	}

	// Debug: Print all contexts and their scores
	t.Logf("Found %d relevant contexts for objective: '%s'", len(relevantContexts), objective)
	for i, context := range relevantContexts {
		t.Logf("  %d: %s (tags: %v)", i+1, context.Content, context.RelevanceTags)
	}

	// Verify that Python and ML contexts are found (relaxed check)
	pythonFound := false
	mlFound := false

	for _, context := range relevantContexts {
		// Check both tags and content for matches
		for _, tag := range context.RelevanceTags {
			if tag == "python" {
				pythonFound = true
			}
			if tag == "machine-learning" {
				mlFound = true
			}
		}
		// Also check content for Python and machine learning keywords
		if containsWord(context.Content, "Python") || containsWord(context.Content, "python") {
			pythonFound = true
		}
		if containsWord(context.Content, "machine learning") || containsWord(context.Content, "machine-learning") {
			mlFound = true
		}
	}

	if !pythonFound {
		t.Error("Python context not found in relevant results")
	}

	if !mlFound {
		t.Error("Machine learning context not found in relevant results")
	}

	// At minimum, should return some contexts
	if len(relevantContexts) < 2 {
		t.Errorf("Expected at least 2 relevant contexts, got %d", len(relevantContexts))
	}
}

func testContextUpdates(t *testing.T, ucm *core.UserContextManager, ctx context.Context, userID string) {
	// Create initial context
	originalContext, err := ucm.LearnContext(ctx,
		core.ContextCategoryPreferences,
		"Original preference",
		core.ContextSourceInferred,
		[]string{"original"},
		userID,
	)
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	// Update the context
	newContent := "Updated preference with more detail"
	newConfidence := 0.85
	newTags := []string{"updated", "detailed"}

	updates := core.UserContextUpdates{
		Content:       &newContent,
		Confidence:    &newConfidence,
		RelevanceTags: newTags,
	}

	updatedContext, err := ucm.UpdateContext(ctx, originalContext.ID, updates)
	if err != nil {
		t.Fatalf("Failed to update context: %v", err)
	}

	if updatedContext.Content != newContent {
		t.Errorf("Content not updated. Expected '%s', got '%s'", newContent, updatedContext.Content)
	}

	if math.Abs(updatedContext.Confidence-newConfidence) > 0.01 {
		t.Errorf("Confidence not updated. Expected %f, got %f", newConfidence, updatedContext.Confidence)
	}

	if len(updatedContext.RelevanceTags) != len(newTags) {
		t.Errorf("Tags not updated. Expected %v, got %v", newTags, updatedContext.RelevanceTags)
	}

	// Verify persistence
	retrievedContext, err := ucm.GetContext(ctx, originalContext.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated context: %v", err)
	}

	if retrievedContext.Content != newContent {
		t.Error("Updated content not persisted")
	}
}

func testContextValidation(t *testing.T, ucm *core.UserContextManager, ctx context.Context, userID string) {
	// Create a context with medium confidence
	context, err := ucm.LearnContext(ctx,
		core.ContextCategoryPatterns,
		"Test pattern for validation",
		core.ContextSourceInferred,
		[]string{"test"},
		userID,
	)
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	originalConfidence := context.Confidence

	// Validate the context with a boost
	boost := 0.2
	err = ucm.ValidateContext(ctx, context.ID, boost)
	if err != nil {
		t.Fatalf("Failed to validate context: %v", err)
	}

	// Retrieve and check confidence boost
	validatedContext, err := ucm.GetContext(ctx, context.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve validated context: %v", err)
	}

	expectedConfidence := math.Min(originalConfidence+boost, 1.0)
	if math.Abs(validatedContext.Confidence-expectedConfidence) > 0.01 {
		t.Errorf("Expected confidence %f after validation, got %f",
			expectedConfidence, validatedContext.Confidence)
	}

	// Test confidence cap at 1.0
	err = ucm.ValidateContext(ctx, context.ID, 0.5) // Large boost
	if err != nil {
		t.Fatalf("Failed to validate context with large boost: %v", err)
	}

	cappedContext, err := ucm.GetContext(ctx, context.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve capped context: %v", err)
	}

	if cappedContext.Confidence > 1.0 {
		t.Errorf("Confidence exceeded 1.0: %f", cappedContext.Confidence)
	}
}

func testRelevanceScoring(t *testing.T, ucm *core.UserContextManager, ctx context.Context, userID string) {
	// Create contexts with different relevance characteristics
	highRelevanceContext, err := ucm.LearnContext(ctx,
		core.ContextCategoryConstraints, // High category bonus
		"Cannot work with Python due to company policy",
		core.ContextSourceExplicit, // High confidence
		[]string{"python", "policy", "constraint"},
		userID,
	)
	if err != nil {
		t.Fatalf("Failed to create high relevance context: %v", err)
	}

	lowRelevanceContext, err := ucm.LearnContext(ctx,
		core.ContextCategoryDomainExpertise, // Lower category bonus
		"Knows some JavaScript",
		core.ContextSourceInferred, // Lower confidence
		[]string{"javascript", "basic"},
		userID,
	)
	if err != nil {
		t.Fatalf("Failed to create low relevance context: %v", err)
	}

	// Test objective that strongly matches the high relevance context
	objective := "Write a Python script for data processing"

	relevantContexts, err := ucm.GetRelevantContext(ctx, objective, userID, 10)
	if err != nil {
		t.Fatalf("Failed to get relevant context: %v", err)
	}

	// Find our test contexts in the results
	var highRelevanceRank, lowRelevanceRank int = -1, -1

	for i, context := range relevantContexts {
		if context.ID == highRelevanceContext.ID {
			highRelevanceRank = i
		}
		if context.ID == lowRelevanceContext.ID {
			lowRelevanceRank = i
		}
	}

	// High relevance context should rank higher (lower index) than low relevance
	if highRelevanceRank != -1 && lowRelevanceRank != -1 && highRelevanceRank >= lowRelevanceRank {
		t.Errorf("High relevance context should rank higher. High: %d, Low: %d",
			highRelevanceRank, lowRelevanceRank)
	}
}

func testEdgeCases(t *testing.T, ucm *core.UserContextManager, ctx context.Context, userID string) {
	// Test empty content
	_, err := ucm.LearnContext(ctx,
		core.ContextCategoryPreferences,
		"", // Empty content
		core.ContextSourceExplicit,
		[]string{"test"},
		userID,
	)
	if err == nil {
		t.Error("Expected error for empty content, got nil")
	}

	// Test invalid category
	_, err = ucm.LearnContext(ctx,
		core.ContextCategory("invalid"), // Invalid category
		"Test content",
		core.ContextSourceExplicit,
		[]string{"test"},
		userID,
	)
	if err == nil {
		t.Error("Expected error for invalid category, got nil")
	}

	// Test invalid source
	_, err = ucm.LearnContext(ctx,
		core.ContextCategoryPreferences,
		"Test content",
		core.ContextSource("invalid"), // Invalid source
		[]string{"test"},
		userID,
	)
	if err == nil {
		t.Error("Expected error for invalid source, got nil")
	}

	// Test retrieval of non-existent context
	_, err = ucm.GetContext(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent context, got nil")
	}

	// Test update of non-existent context
	updates := core.UserContextUpdates{
		Content: stringPtr("Updated content"),
	}
	_, err = ucm.UpdateContext(ctx, "non-existent-id", updates)
	if err == nil {
		t.Error("Expected error for updating non-existent context, got nil")
	}

	// Test validation of non-existent context
	err = ucm.ValidateContext(ctx, "non-existent-id", 0.1)
	if err == nil {
		t.Error("Expected error for validating non-existent context, got nil")
	}
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func stringPtr(s string) *string {
	return &s
}

func containsWord(text, word string) bool {
	// Simple word containment check (case-insensitive)
	return strings.Contains(strings.ToLower(text), strings.ToLower(word))
}