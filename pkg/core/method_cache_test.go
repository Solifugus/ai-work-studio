package core

import (
	"context"
	"log"
	"math"
	"testing"
	"time"

	"github.com/Solifugus/ai-work-studio/pkg/mcp"
	"github.com/Solifugus/ai-work-studio/pkg/storage"
)

// MockLLMProvider provides predictable embeddings for testing.
type MockLLMProvider struct {
	embeddings map[string][]float64
}

func (m *MockLLMProvider) Name() string {
	return "Mock Provider"
}

func (m *MockLLMProvider) Complete(ctx context.Context, request mcp.CompletionRequest) (*mcp.CompletionResponse, error) {
	return &mcp.CompletionResponse{
		Text:       "Mock completion",
		TokensUsed: 10,
		Model:      request.Model,
		Provider:   "mock",
		Cost:       0.0,
	}, nil
}

func (m *MockLLMProvider) Embed(ctx context.Context, request mcp.EmbeddingRequest) (*mcp.EmbeddingResponse, error) {
	embedding, exists := m.embeddings[request.Text]
	if !exists {
		// Generate a simple hash-based embedding for unknown text
		hash := simpleHash(request.Text)
		embedding = make([]float64, 384) // Standard embedding size
		for i := range embedding {
			embedding[i] = math.Sin(float64(hash+i)) * 0.1
		}
		m.embeddings[request.Text] = embedding
	}

	return &mcp.EmbeddingResponse{
		Embedding:  embedding,
		TokensUsed: len(request.Text) / 4,
		Model:      request.Model,
		Provider:   "mock",
		Cost:       0.0,
	}, nil
}

func (m *MockLLMProvider) CalculateCost(tokens int, operation string) float64 {
	return 0.0 // Mock is free
}

// simpleHash generates a simple hash for testing purposes.
func simpleHash(s string) int {
	hash := 0
	for _, char := range s {
		hash = 31*hash + int(char)
	}
	return hash
}

// setupTestMethodCache creates a method cache with mocked LLM service.
func setupTestMethodCache(t *testing.T, config ...CacheConfig) (*MethodCache, *storage.Store, *MethodManager) {
	store := setupTestStore(t)

	// Create LLM service with mock provider
	llmService := mcp.NewLLMService(log.New(&testLogWriter{}, "test: ", log.LstdFlags))

	// Create mock provider with predictable embeddings
	mockProvider := &MockLLMProvider{
		embeddings: map[string][]float64{
			"file processing": createTestEmbedding(0.1, 0.8, 0.3),
			"data analysis":   createTestEmbedding(0.9, 0.2, 0.7),
			"api integration": createTestEmbedding(0.5, 0.9, 0.1),
			"process files":   createTestEmbedding(0.2, 0.7, 0.4), // Similar to "file processing"
			"analyze data":    createTestEmbedding(0.8, 0.3, 0.6), // Similar to "data analysis"
		},
	}

	llmService.SetProvider("mock", mockProvider)

	// Create cache with custom config if provided
	var cache *MethodCache
	if len(config) > 0 {
		cache = NewMethodCache(store, llmService, config[0])
	} else {
		cache = NewMethodCache(store, llmService)
	}

	mm := NewMethodManager(store)

	return cache, store, mm
}

// createTestEmbedding generates a normalized test embedding vector.
func createTestEmbedding(x, y, z float64) []float64 {
	embedding := make([]float64, 384)
	for i := range embedding {
		switch i % 3 {
		case 0:
			embedding[i] = x
		case 1:
			embedding[i] = y
		case 2:
			embedding[i] = z
		}
	}

	// Normalize the vector
	norm := 0.0
	for _, val := range embedding {
		norm += val * val
	}
	norm = math.Sqrt(norm)

	for i := range embedding {
		embedding[i] /= norm
	}

	return embedding
}

type testLogWriter struct{}

func (w *testLogWriter) Write(p []byte) (n int, err error) {
	return len(p), nil // Discard log output during tests
}

// createTestMethodWithMetrics creates a method with specified success metrics.
func createTestMethodWithMetrics(t *testing.T, mm *MethodManager, name, description string, domain MethodDomain, successRate float64, lastUsed time.Time) *Method {
	ctx := context.Background()

	method, err := mm.CreateMethod(ctx, name, description, []ApproachStep{}, domain, nil)
	if err != nil {
		t.Fatalf("Failed to create test method: %v", err)
	}

	// Update metrics to reflect desired success rate
	if successRate > 0 {
		execCount := 10
		successCount := int(float64(execCount) * (successRate / 100.0))

		metrics := SuccessMetrics{
			ExecutionCount: execCount,
			SuccessCount:   successCount,
			LastUsed:       lastUsed,
			AverageRating:  successRate / 10.0, // Convert to 1-10 scale
		}

		updates := MethodUpdates{Metrics: &metrics}
		_, err = mm.UpdateMethod(ctx, method.ID, updates)
		if err != nil {
			t.Fatalf("Failed to update method metrics: %v", err)
		}

		// Get the updated method
		method, err = mm.GetMethod(ctx, method.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve updated method: %v", err)
		}
	}

	return method
}

func TestMethodCache_CacheProvenMethod(t *testing.T) {
	tests := []struct {
		name           string
		successRate    float64
		methodStatus   MethodStatus
		expectCached   bool
	}{
		{
			name:         "high success rate active method",
			successRate:  85.0,
			methodStatus: MethodStatusActive,
			expectCached: true,
		},
		{
			name:         "low success rate method",
			successRate:  60.0,
			methodStatus: MethodStatusActive,
			expectCached: false,
		},
		{
			name:         "deprecated method",
			successRate:  90.0,
			methodStatus: MethodStatusDeprecated,
			expectCached: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh cache and manager for each test
			cache, _, mm := setupTestMethodCache(t)
			ctx := context.Background()

			// Record initial cache count
			initialStats := cache.GetCacheStats()
			initialCount := initialStats.CachedMethods

			method := createTestMethodWithMetrics(t, mm, tt.name, "Test method", MethodDomainGeneral, tt.successRate, time.Now())

			// Set method status
			if tt.methodStatus != MethodStatusActive {
				status := tt.methodStatus
				updates := MethodUpdates{Status: &status}
				_, err := mm.UpdateMethod(ctx, method.ID, updates)
				if err != nil {
					t.Fatalf("Failed to update method status: %v", err)
				}
				method, _ = mm.GetMethod(ctx, method.ID)
			}

			// Try to cache the method
			err := cache.CacheProvenMethod(ctx, method)
			if err != nil {
				t.Fatalf("CacheProvenMethod failed: %v", err)
			}

			// Check if method was cached by comparing counts
			finalStats := cache.GetCacheStats()
			methodWasCached := finalStats.CachedMethods > initialCount

			if methodWasCached != tt.expectCached {
				t.Errorf("Expected cached=%v, got cached=%v for method with success rate %.1f%% and status %s",
					tt.expectCached, methodWasCached, tt.successRate, tt.methodStatus)
			}
		})
	}
}

func TestMethodCache_QueryByDomain(t *testing.T) {
	cache, _, mm := setupTestMethodCache(t)
	ctx := context.Background()

	// Create test methods in different domains
	method1 := createTestMethodWithMetrics(t, mm, "General Method", "General purpose method", MethodDomainGeneral, 80.0, time.Now())
	method2 := createTestMethodWithMetrics(t, mm, "Specific Method", "Domain specific method", MethodDomainSpecific, 85.0, time.Now())
	method3 := createTestMethodWithMetrics(t, mm, "User Method", "User specific method", MethodDomainUser, 90.0, time.Now())

	// Cache the methods
	cache.CacheProvenMethod(ctx, method1)
	cache.CacheProvenMethod(ctx, method2)
	cache.CacheProvenMethod(ctx, method3)

	// Query by specific domain
	results, err := cache.Query().WithDomain(MethodDomainSpecific).Execute(ctx)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Method.ID != method2.ID {
		t.Errorf("Expected method2, got method %s", results[0].Method.ID)
	}
}

func TestMethodCache_QueryBySimilarity(t *testing.T) {
	cache, _, mm := setupTestMethodCache(t)
	ctx := context.Background()

	// Create methods with known descriptions
	method1 := createTestMethodWithMetrics(t, mm, "File Processor", "file processing", MethodDomainGeneral, 80.0, time.Now())
	method2 := createTestMethodWithMetrics(t, mm, "Data Analyzer", "data analysis", MethodDomainGeneral, 85.0, time.Now())
	method3 := createTestMethodWithMetrics(t, mm, "API Connector", "api integration", MethodDomainGeneral, 90.0, time.Now())

	// Cache the methods
	cache.CacheProvenMethod(ctx, method1)
	cache.CacheProvenMethod(ctx, method2)
	cache.CacheProvenMethod(ctx, method3)

	// Query for similar objective
	results, err := cache.Query().
		WithObjective("process files").
		WithMinSimilarity(0.5).
		WithMaxResults(2).
		Execute(ctx)

	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	// First result should be the most similar (file processing)
	if results[0].Method.ID != method1.ID {
		t.Errorf("Expected method1 (file processing) to be most similar, got %s", results[0].Method.Name)
	}

	// Check that similarity score is reasonable
	if results[0].SimilarityScore < 0.5 {
		t.Errorf("Similarity score too low: %f", results[0].SimilarityScore)
	}
}

func TestMethodCache_QueryWithRanking(t *testing.T) {
	cache, _, mm := setupTestMethodCache(t)
	ctx := context.Background()

	// Create methods with different success rates and recency
	oldTime := time.Now().Add(-30 * 24 * time.Hour) // 30 days ago
	recentTime := time.Now().Add(-1 * time.Hour)    // 1 hour ago

	method1 := createTestMethodWithMetrics(t, mm, "Old High Success", "analyze data", MethodDomainGeneral, 95.0, oldTime)
	method2 := createTestMethodWithMetrics(t, mm, "Recent Medium Success", "analyze data", MethodDomainGeneral, 80.0, recentTime)

	// Cache the methods
	cache.CacheProvenMethod(ctx, method1)
	cache.CacheProvenMethod(ctx, method2)

	// Query with similar objective
	results, err := cache.Query().
		WithObjective("analyze data").
		Execute(ctx)

	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// Results should be ordered by composite score
	if results[0].CompositeScore <= results[1].CompositeScore {
		t.Error("Results should be ordered by composite score (descending)")
	}

	// Check that scores are calculated correctly
	for _, result := range results {
		if result.SuccessScore < 0 || result.SuccessScore > 1 {
			t.Errorf("Success score out of range: %f", result.SuccessScore)
		}
		if result.RecencyScore < 0 || result.RecencyScore > 1 {
			t.Errorf("Recency score out of range: %f", result.RecencyScore)
		}
		if result.SimilarityScore < 0 || result.SimilarityScore > 1 {
			t.Errorf("Similarity score out of range: %f", result.SimilarityScore)
		}
	}
}

func TestMethodCache_QueryWithFilters(t *testing.T) {
	cache, _, mm := setupTestMethodCache(t)
	ctx := context.Background()

	// Create methods with different properties - use more spaced out times
	baseTime := time.Now()

	// Create methods with guaranteed different creation times by manipulating the data directly
	method1 := createTestMethodWithMetrics(t, mm, "Recent High", "file processing", MethodDomainGeneral, 90.0, baseTime.Add(-1*time.Hour))

	// Wait a bit to ensure different creation times
	time.Sleep(10 * time.Millisecond)
	method2 := createTestMethodWithMetrics(t, mm, "Recent Medium", "file processing", MethodDomainGeneral, 80.0, baseTime.Add(-2*time.Hour))

	time.Sleep(10 * time.Millisecond)
	method3 := createTestMethodWithMetrics(t, mm, "Old High", "file processing", MethodDomainGeneral, 85.0, baseTime.Add(-30*24*time.Hour))

	// Cache the methods
	cache.CacheProvenMethod(ctx, method1)
	cache.CacheProvenMethod(ctx, method2)
	cache.CacheProvenMethod(ctx, method3)

	// Test that we can find methods by different criteria
	t.Run("basic query", func(t *testing.T) {
		results, err := cache.Query().WithObjective("file processing").Execute(ctx)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) < 3 {
			t.Errorf("Expected at least 3 results, got %d", len(results))
		}

		t.Logf("All methods found:")
		for i, result := range results {
			t.Logf("  %d: %s - Success: %.1f%%, Created: %s",
				i, result.Method.Name, result.Method.Metrics.SuccessRate(), result.Method.CreatedAt)
		}
	})

	t.Run("exclude specific method", func(t *testing.T) {
		results, err := cache.Query().
			WithObjective("file processing").
			ExcludeIDs(method1.ID).
			Execute(ctx)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		// Should have 2 methods (method2 and method3)
		if len(results) != 2 {
			t.Errorf("Expected 2 results after excluding method1, got %d", len(results))
			for i, result := range results {
				t.Logf("  %d: %s (ID: %s)", i, result.Method.Name, result.Method.ID)
			}
		}

		// Verify method1 is not in results
		for _, result := range results {
			if result.Method.ID == method1.ID {
				t.Error("method1 should have been excluded")
			}
		}
	})
}

func TestMethodCache_CacheEviction(t *testing.T) {
	// Create cache with small size for testing eviction
	config := DefaultCacheConfig()
	config.MaxCacheSize = 2

	cache, _, mm := setupTestMethodCache(t, config)
	ctx := context.Background()

	// Create methods
	method1 := createTestMethodWithMetrics(t, mm, "Method 1", "First method", MethodDomainGeneral, 80.0, time.Now())
	method2 := createTestMethodWithMetrics(t, mm, "Method 2", "Second method", MethodDomainGeneral, 85.0, time.Now())
	method3 := createTestMethodWithMetrics(t, mm, "Method 3", "Third method", MethodDomainGeneral, 90.0, time.Now())

	// Cache methods one by one
	err := cache.CacheProvenMethod(ctx, method1)
	if err != nil {
		t.Fatalf("Failed to cache method1: %v", err)
	}

	err = cache.CacheProvenMethod(ctx, method2)
	if err != nil {
		t.Fatalf("Failed to cache method2: %v", err)
	}

	// Cache should be at capacity
	stats := cache.GetCacheStats()
	if stats.CachedMethods != 2 {
		t.Fatalf("Expected 2 cached methods, got %d", stats.CachedMethods)
	}

	// Cache third method, should evict the least recently used
	err = cache.CacheProvenMethod(ctx, method3)
	if err != nil {
		t.Fatalf("Failed to cache method3: %v", err)
	}

	// Should still have 2 methods, but method1 should be evicted
	stats = cache.GetCacheStats()
	if stats.CachedMethods != 2 {
		t.Errorf("Expected 2 cached methods after eviction, got %d", stats.CachedMethods)
	}
}

func TestMethodCache_RefreshCache(t *testing.T) {
	cache, _, mm := setupTestMethodCache(t)
	ctx := context.Background()

	// Create and cache a method
	method := createTestMethodWithMetrics(t, mm, "Test Method", "Test description", MethodDomainGeneral, 80.0, time.Now())
	err := cache.CacheProvenMethod(ctx, method)
	if err != nil {
		t.Fatalf("Failed to cache method: %v", err)
	}

	// Verify method is cached
	stats := cache.GetCacheStats()
	if stats.CachedMethods != 1 {
		t.Fatalf("Expected 1 cached method, got %d", stats.CachedMethods)
	}

	// Update the method to have low success rate
	lowSuccessMetrics := SuccessMetrics{
		ExecutionCount: 10,
		SuccessCount:   5, // 50% success rate
		LastUsed:       time.Now(),
		AverageRating:  5.0,
	}
	updates := MethodUpdates{Metrics: &lowSuccessMetrics}
	_, err = mm.UpdateMethod(ctx, method.ID, updates)
	if err != nil {
		t.Fatalf("Failed to update method: %v", err)
	}

	// Refresh cache - method should be evicted due to low success rate
	err = cache.RefreshCache(ctx)
	if err != nil {
		t.Fatalf("RefreshCache failed: %v", err)
	}

	// Method should no longer be cached
	stats = cache.GetCacheStats()
	if stats.CachedMethods != 0 {
		t.Errorf("Expected 0 cached methods after refresh, got %d", stats.CachedMethods)
	}
}

func TestMethodCache_EvictMethod(t *testing.T) {
	cache, _, mm := setupTestMethodCache(t)
	ctx := context.Background()

	// Create and cache methods
	method1 := createTestMethodWithMetrics(t, mm, "Method 1", "First method", MethodDomainGeneral, 80.0, time.Now())
	method2 := createTestMethodWithMetrics(t, mm, "Method 2", "Second method", MethodDomainGeneral, 85.0, time.Now())

	cache.CacheProvenMethod(ctx, method1)
	cache.CacheProvenMethod(ctx, method2)

	// Verify both methods are cached
	stats := cache.GetCacheStats()
	if stats.CachedMethods != 2 {
		t.Fatalf("Expected 2 cached methods, got %d", stats.CachedMethods)
	}

	// Evict one method
	cache.EvictMethod(method1.ID)

	// Verify only one method remains
	stats = cache.GetCacheStats()
	if stats.CachedMethods != 1 {
		t.Errorf("Expected 1 cached method after eviction, got %d", stats.CachedMethods)
	}
}

func TestMethodCache_GetCacheStats(t *testing.T) {
	cache, _, mm := setupTestMethodCache(t)
	ctx := context.Background()

	// Initially empty
	stats := cache.GetCacheStats()
	if stats.CachedMethods != 0 || stats.EmbeddingsCached != 0 {
		t.Errorf("Expected empty cache, got %+v", stats)
	}

	// Add a method
	method := createTestMethodWithMetrics(t, mm, "Test Method", "Test description", MethodDomainGeneral, 80.0, time.Now())
	err := cache.CacheProvenMethod(ctx, method)
	if err != nil {
		t.Fatalf("Failed to cache method: %v", err)
	}

	// Check stats
	stats = cache.GetCacheStats()
	if stats.CachedMethods != 1 {
		t.Errorf("Expected 1 cached method, got %d", stats.CachedMethods)
	}

	if stats.EmbeddingsCached == 0 {
		t.Error("Expected embeddings to be cached")
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float64{1.0, 0.0, 0.0},
			b:        []float64{1.0, 0.0, 0.0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float64{1.0, 0.0, 0.0},
			b:        []float64{0.0, 1.0, 0.0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float64{1.0, 0.0, 0.0},
			b:        []float64{-1.0, 0.0, 0.0},
			expected: -1.0,
		},
		{
			name:     "different lengths",
			a:        []float64{1.0, 0.0},
			b:        []float64{1.0, 0.0, 0.0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			if math.Abs(result-tt.expected) > 1e-6 {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestCacheConfigValidation(t *testing.T) {
	config := DefaultCacheConfig()

	// Check that weights sum to approximately 1.0
	totalWeight := config.RecencyWeight + config.SuccessWeight + config.SimilarityWeight
	if math.Abs(totalWeight-1.0) > 0.01 {
		t.Errorf("Weights should sum to 1.0, got %f", totalWeight)
	}

	// Check reasonable defaults
	if config.MinSuccessRate < 0 || config.MinSuccessRate > 100 {
		t.Errorf("MinSuccessRate should be 0-100, got %f", config.MinSuccessRate)
	}

	if config.SimilarityThreshold < 0 || config.SimilarityThreshold > 1 {
		t.Errorf("SimilarityThreshold should be 0-1, got %f", config.SimilarityThreshold)
	}
}

// Benchmarks

func BenchmarkMethodCache_Query(b *testing.B) {
	// Create a test instance for setup
	t := &testing.T{}
	cache, _, mm := setupTestMethodCache(t)
	ctx := context.Background()

	// Create and cache multiple methods
	for i := 0; i < 100; i++ {
		method := createTestMethodWithMetrics(t, mm, "Method", "file processing method", MethodDomainGeneral, 80.0, time.Now())
		cache.CacheProvenMethod(ctx, method)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, _ := cache.Query().WithObjective("process files").Execute(ctx)
		_ = results
	}
}

func BenchmarkCosineSimilarity(b *testing.B) {
	a := make([]float64, 384)
	bVec := make([]float64, 384)

	for i := range a {
		a[i] = float64(i) / 384.0
		bVec[i] = float64(384-i) / 384.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cosineSimilarity(a, bVec)
	}
}