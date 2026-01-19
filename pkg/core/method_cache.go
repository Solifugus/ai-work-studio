package core

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Solifugus/ai-work-studio/pkg/mcp"
	"github.com/Solifugus/ai-work-studio/pkg/storage"
)

// MethodCache provides efficient caching and retrieval of proven methods.
// It supports semantic similarity matching, ranking by multiple factors,
// and session-based performance optimization.
type MethodCache struct {
	store           *storage.Store
	llmService      *mcp.LLMService
	config          CacheConfig
	sessionCache    map[string]*CacheEntry
	embeddingCache  map[string][]float64
	cacheMutex      sync.RWMutex
	embeddingMutex  sync.RWMutex
}

// CacheConfig contains configuration for method cache behavior.
type CacheConfig struct {
	// MinSuccessRate is the minimum success rate (0-100) for methods to be cached
	MinSuccessRate float64

	// MaxCacheSize is the maximum number of methods to keep in session cache
	MaxCacheSize int

	// CacheExpiry is how long session cache entries remain valid
	CacheExpiry time.Duration

	// SimilarityThreshold is the minimum similarity score (0-1) for matches
	SimilarityThreshold float64

	// MaxResults is the maximum number of results to return
	MaxResults int

	// RecencyWeight affects how much recency impacts ranking (0-1)
	RecencyWeight float64

	// SuccessWeight affects how much success rate impacts ranking (0-1)
	SuccessWeight float64

	// SimilarityWeight affects how much similarity impacts ranking (0-1)
	SimilarityWeight float64
}

// DefaultCacheConfig returns sensible defaults for cache configuration.
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		MinSuccessRate:      75.0, // Only cache methods with 75%+ success
		MaxCacheSize:        500,  // Keep up to 500 methods in session cache
		CacheExpiry:         30 * time.Minute,
		SimilarityThreshold: 0.7,  // Require 70% similarity for matches
		MaxResults:          10,   // Return top 10 matches by default
		RecencyWeight:       0.2,  // 20% weight for how recent the method is
		SuccessWeight:       0.4,  // 40% weight for success rate
		SimilarityWeight:    0.4,  // 40% weight for similarity score
	}
}

// CacheEntry represents a cached method with additional metadata for quick retrieval.
type CacheEntry struct {
	Method      *Method
	Embedding   []float64
	CachedAt    time.Time
	LastAccessed time.Time
	AccessCount int
}

// MatchResult represents a method match with confidence scoring.
type MatchResult struct {
	Method           *Method
	SimilarityScore  float64 // 0-1, how similar to the query
	SuccessScore     float64 // 0-1, normalized success rate
	RecencyScore     float64 // 0-1, how recent the method is
	CompositeScore   float64 // 0-1, weighted combination of all scores
	MatchReason      string  // Human-readable explanation of why this matched
}

// CacheQuery provides a fluent interface for building method cache queries.
type CacheQuery struct {
	cache       *MethodCache
	domain      *MethodDomain
	objective   string
	minSuccess  *float64
	maxResults  *int
	similarity  *float64
	createdAfter *time.Time
	excludeIDs  []string
}

// SimilarityMatcher defines the interface for calculating similarity between methods and objectives.
type SimilarityMatcher interface {
	CalculateSimilarity(ctx context.Context, methodDescription string, objective string) (float64, error)
}

// EmbeddingSimilarityMatcher uses vector embeddings to calculate semantic similarity.
type EmbeddingSimilarityMatcher struct {
	llmService *mcp.LLMService
	cache      *MethodCache
}

// NewMethodCache creates a new method cache instance.
func NewMethodCache(store *storage.Store, llmService *mcp.LLMService, config ...CacheConfig) *MethodCache {
	cfg := DefaultCacheConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return &MethodCache{
		store:          store,
		llmService:     llmService,
		config:         cfg,
		sessionCache:   make(map[string]*CacheEntry),
		embeddingCache: make(map[string][]float64),
	}
}

// Query creates a new query builder for retrieving methods from the cache.
func (mc *MethodCache) Query() *CacheQuery {
	return &CacheQuery{
		cache: mc,
	}
}

// WithDomain filters methods by domain.
func (cq *CacheQuery) WithDomain(domain MethodDomain) *CacheQuery {
	cq.domain = &domain
	return cq
}

// WithObjective specifies the objective to find similar methods for.
func (cq *CacheQuery) WithObjective(objective string) *CacheQuery {
	cq.objective = objective
	return cq
}

// WithMinSuccessRate filters methods by minimum success rate.
func (cq *CacheQuery) WithMinSuccessRate(rate float64) *CacheQuery {
	cq.minSuccess = &rate
	return cq
}

// WithMaxResults limits the number of results returned.
func (cq *CacheQuery) WithMaxResults(max int) *CacheQuery {
	cq.maxResults = &max
	return cq
}

// WithMinSimilarity sets minimum similarity threshold for matches.
func (cq *CacheQuery) WithMinSimilarity(similarity float64) *CacheQuery {
	cq.similarity = &similarity
	return cq
}

// CreatedAfter filters methods created after the specified time.
func (cq *CacheQuery) CreatedAfter(after time.Time) *CacheQuery {
	cq.createdAfter = &after
	return cq
}

// ExcludeIDs excludes methods with the specified IDs.
func (cq *CacheQuery) ExcludeIDs(ids ...string) *CacheQuery {
	cq.excludeIDs = append(cq.excludeIDs, ids...)
	return cq
}

// Execute runs the query and returns ranked method matches.
func (cq *CacheQuery) Execute(ctx context.Context) ([]*MatchResult, error) {
	// Start by getting all methods that meet basic criteria
	candidates, err := cq.getCandidateMethods(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get candidate methods: %w", err)
	}

	// If no objective specified, return methods ranked by success rate and recency
	if cq.objective == "" {
		return cq.rankWithoutSimilarity(candidates), nil
	}

	// Calculate similarity scores for each candidate
	results, err := cq.calculateSimilarityScores(ctx, candidates)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate similarity scores: %w", err)
	}

	// Apply similarity threshold
	threshold := cq.cache.config.SimilarityThreshold
	if cq.similarity != nil {
		threshold = *cq.similarity
	}

	var filteredResults []*MatchResult
	for _, result := range results {
		if result.SimilarityScore >= threshold {
			filteredResults = append(filteredResults, result)
		}
	}

	// Sort by composite score
	sort.Slice(filteredResults, func(i, j int) bool {
		return filteredResults[i].CompositeScore > filteredResults[j].CompositeScore
	})

	// Limit results
	maxResults := cq.cache.config.MaxResults
	if cq.maxResults != nil {
		maxResults = *cq.maxResults
	}

	if len(filteredResults) > maxResults {
		filteredResults = filteredResults[:maxResults]
	}

	return filteredResults, nil
}

// getCandidateMethods retrieves methods that meet basic filtering criteria.
func (cq *CacheQuery) getCandidateMethods(ctx context.Context) ([]*Method, error) {
	mm := NewMethodManager(cq.cache.store)

	// Build filter for MethodManager
	filter := MethodFilter{}

	if cq.domain != nil {
		filter.Domain = cq.domain
	}

	// Only include active methods
	status := MethodStatusActive
	filter.Status = &status

	// Apply minimum success rate
	minSuccess := cq.cache.config.MinSuccessRate
	if cq.minSuccess != nil {
		minSuccess = *cq.minSuccess
	}
	filter.MinSuccessRate = &minSuccess

	// Get methods from storage
	methods, err := mm.ListMethods(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Apply additional filters
	var candidates []*Method
	for _, method := range methods {
		// Filter by creation time
		if cq.createdAfter != nil && method.CreatedAt.Before(*cq.createdAfter) {
			continue
		}

		// Exclude specific IDs
		excluded := false
		for _, excludeID := range cq.excludeIDs {
			if method.ID == excludeID {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		candidates = append(candidates, method)
	}

	return candidates, nil
}

// rankWithoutSimilarity ranks methods based only on success rate and recency.
func (cq *CacheQuery) rankWithoutSimilarity(candidates []*Method) []*MatchResult {
	var results []*MatchResult
	now := time.Now()

	for _, method := range candidates {
		successScore := method.Metrics.SuccessRate() / 100.0 // Normalize to 0-1

		// Calculate recency score (methods used more recently score higher)
		daysSinceLastUsed := now.Sub(method.Metrics.LastUsed).Hours() / 24
		recencyScore := math.Exp(-daysSinceLastUsed / 30.0) // Exponential decay over 30 days

		// Composite score without similarity
		compositeScore := (successScore * cq.cache.config.SuccessWeight) +
			(recencyScore * cq.cache.config.RecencyWeight)

		result := &MatchResult{
			Method:         method,
			SimilarityScore: 0.0, // No objective to compare against
			SuccessScore:   successScore,
			RecencyScore:   recencyScore,
			CompositeScore: compositeScore,
			MatchReason:    fmt.Sprintf("Success rate: %.1f%%, Last used: %s",
				method.Metrics.SuccessRate(),
				formatTimeSince(method.Metrics.LastUsed)),
		}

		results = append(results, result)
	}

	// Sort by composite score
	sort.Slice(results, func(i, j int) bool {
		return results[i].CompositeScore > results[j].CompositeScore
	})

	return results
}

// calculateSimilarityScores computes similarity and composite scores for candidates.
func (cq *CacheQuery) calculateSimilarityScores(ctx context.Context, candidates []*Method) ([]*MatchResult, error) {
	matcher := &EmbeddingSimilarityMatcher{
		llmService: cq.cache.llmService,
		cache:      cq.cache,
	}

	var results []*MatchResult
	now := time.Now()

	for _, method := range candidates {
		// Calculate similarity to objective
		similarity, err := matcher.CalculateSimilarity(ctx, method.Description, cq.objective)
		if err != nil {
			// Log error but continue with other methods
			fmt.Printf("Warning: failed to calculate similarity for method %s: %v\n", method.ID, err)
			continue
		}

		// Calculate success score
		successScore := method.Metrics.SuccessRate() / 100.0

		// Calculate recency score
		daysSinceLastUsed := now.Sub(method.Metrics.LastUsed).Hours() / 24
		recencyScore := math.Exp(-daysSinceLastUsed / 30.0)

		// Calculate composite score using weighted combination
		compositeScore := (similarity * cq.cache.config.SimilarityWeight) +
			(successScore * cq.cache.config.SuccessWeight) +
			(recencyScore * cq.cache.config.RecencyWeight)

		// Generate match reason
		matchReason := cq.generateMatchReason(method, similarity, successScore, recencyScore)

		result := &MatchResult{
			Method:         method,
			SimilarityScore: similarity,
			SuccessScore:   successScore,
			RecencyScore:   recencyScore,
			CompositeScore: compositeScore,
			MatchReason:    matchReason,
		}

		results = append(results, result)
	}

	return results, nil
}

// generateMatchReason creates a human-readable explanation for why a method matched.
func (cq *CacheQuery) generateMatchReason(method *Method, similarity, success, recency float64) string {
	reasons := []string{}

	if similarity >= 0.9 {
		reasons = append(reasons, "very high similarity")
	} else if similarity >= 0.8 {
		reasons = append(reasons, "high similarity")
	} else if similarity >= 0.7 {
		reasons = append(reasons, "good similarity")
	}

	if success >= 0.9 {
		reasons = append(reasons, "excellent success rate")
	} else if success >= 0.8 {
		reasons = append(reasons, "high success rate")
	} else if success >= 0.75 {
		reasons = append(reasons, "good success rate")
	}

	if recency >= 0.8 {
		reasons = append(reasons, "recently used")
	}

	if len(reasons) == 0 {
		return fmt.Sprintf("%.1f%% similarity, %.1f%% success rate",
			similarity*100, method.Metrics.SuccessRate())
	}

	return strings.Join(reasons, ", ")
}

// CacheProvenMethod adds a method to the cache if it meets the success criteria.
func (mc *MethodCache) CacheProvenMethod(ctx context.Context, method *Method) error {
	// Check if method meets caching criteria
	if method.Metrics.SuccessRate() < mc.config.MinSuccessRate {
		return nil // Not proven enough to cache
	}

	if !method.IsActive() {
		return nil // Only cache active methods
	}

	// Get or compute embedding for the method
	embedding, err := mc.getMethodEmbedding(ctx, method)
	if err != nil {
		return fmt.Errorf("failed to get method embedding: %w", err)
	}

	// Add to session cache
	mc.cacheMutex.Lock()
	defer mc.cacheMutex.Unlock()

	// Check cache size and evict if necessary
	if len(mc.sessionCache) >= mc.config.MaxCacheSize {
		mc.evictLeastUsed()
	}

	mc.sessionCache[method.ID] = &CacheEntry{
		Method:      method,
		Embedding:   embedding,
		CachedAt:    time.Now(),
		LastAccessed: time.Now(),
		AccessCount: 0,
	}

	return nil
}

// EvictMethod removes a method from the cache.
func (mc *MethodCache) EvictMethod(methodID string) {
	mc.cacheMutex.Lock()
	defer mc.cacheMutex.Unlock()

	delete(mc.sessionCache, methodID)

	// Also remove from embedding cache
	mc.embeddingMutex.Lock()
	defer mc.embeddingMutex.Unlock()
	delete(mc.embeddingCache, methodID)
}

// RefreshCache updates cached methods from storage and evicts expired entries.
func (mc *MethodCache) RefreshCache(ctx context.Context) error {
	mc.cacheMutex.Lock()
	defer mc.cacheMutex.Unlock()

	now := time.Now()
	mm := NewMethodManager(mc.store)

	// Evict expired entries
	for methodID, entry := range mc.sessionCache {
		if now.Sub(entry.CachedAt) > mc.config.CacheExpiry {
			delete(mc.sessionCache, methodID)
		}
	}

	// Update existing entries with fresh data from storage
	for methodID, entry := range mc.sessionCache {
		freshMethod, err := mm.GetMethod(ctx, methodID)
		if err != nil {
			// Method no longer exists, remove from cache
			delete(mc.sessionCache, methodID)
			continue
		}

		// Check if method still meets caching criteria
		if freshMethod.Metrics.SuccessRate() < mc.config.MinSuccessRate || !freshMethod.IsActive() {
			delete(mc.sessionCache, methodID)
			continue
		}

		// Update the cached method
		entry.Method = freshMethod
	}

	return nil
}

// GetCacheStats returns statistics about the current cache state.
func (mc *MethodCache) GetCacheStats() CacheStats {
	mc.cacheMutex.RLock()
	defer mc.cacheMutex.RUnlock()

	stats := CacheStats{
		CachedMethods:    len(mc.sessionCache),
		EmbeddingsCached: len(mc.embeddingCache),
		MaxCacheSize:     mc.config.MaxCacheSize,
	}

	now := time.Now()
	for _, entry := range mc.sessionCache {
		stats.TotalAccessCount += entry.AccessCount
		if now.Sub(entry.LastAccessed) < time.Hour {
			stats.RecentlyAccessedCount++
		}
	}

	return stats
}

// CacheStats provides information about cache performance and usage.
type CacheStats struct {
	CachedMethods         int
	EmbeddingsCached      int
	TotalAccessCount      int
	RecentlyAccessedCount int
	MaxCacheSize          int
}

// evictLeastUsed removes the least recently used entry from the cache.
func (mc *MethodCache) evictLeastUsed() {
	if len(mc.sessionCache) == 0 {
		return
	}

	var oldestID string
	var oldestTime time.Time = time.Now()

	for methodID, entry := range mc.sessionCache {
		if entry.LastAccessed.Before(oldestTime) {
			oldestTime = entry.LastAccessed
			oldestID = methodID
		}
	}

	if oldestID != "" {
		delete(mc.sessionCache, oldestID)
	}
}

// getMethodEmbedding gets or computes the embedding for a method description.
func (mc *MethodCache) getMethodEmbedding(ctx context.Context, method *Method) ([]float64, error) {
	// Check embedding cache first
	mc.embeddingMutex.RLock()
	if embedding, exists := mc.embeddingCache[method.ID]; exists {
		mc.embeddingMutex.RUnlock()
		return embedding, nil
	}
	mc.embeddingMutex.RUnlock()

	// Compute embedding
	text := method.Name + " " + method.Description

	// Use LLM service to get embedding
	params := mcp.ServiceParams{
		"operation": "embed",
		"text":      text,
	}

	result := mc.llmService.Execute(ctx, params)
	if result.Error != nil {
		return nil, fmt.Errorf("embedding generation failed: %w", result.Error)
	}

	// Extract embedding from result
	embeddingResp, ok := result.Data.(*mcp.EmbeddingResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected embedding response type")
	}

	embedding := embeddingResp.Embedding

	// Cache the embedding
	mc.embeddingMutex.Lock()
	mc.embeddingCache[method.ID] = embedding
	mc.embeddingMutex.Unlock()

	return embedding, nil
}

// CalculateSimilarity computes semantic similarity between method description and objective.
func (esm *EmbeddingSimilarityMatcher) CalculateSimilarity(ctx context.Context, methodDescription string, objective string) (float64, error) {
	// Get embedding for objective
	params := mcp.ServiceParams{
		"operation": "embed",
		"text":      objective,
	}

	result := esm.llmService.Execute(ctx, params)
	if result.Error != nil {
		return 0.0, fmt.Errorf("failed to get objective embedding: %w", result.Error)
	}

	embeddingResp, ok := result.Data.(*mcp.EmbeddingResponse)
	if !ok {
		return 0.0, fmt.Errorf("unexpected embedding response type")
	}

	objectiveEmbedding := embeddingResp.Embedding

	// Get embedding for method description
	params["text"] = methodDescription
	result = esm.llmService.Execute(ctx, params)
	if result.Error != nil {
		return 0.0, fmt.Errorf("failed to get method embedding: %w", result.Error)
	}

	embeddingResp, ok = result.Data.(*mcp.EmbeddingResponse)
	if !ok {
		return 0.0, fmt.Errorf("unexpected embedding response type")
	}

	methodEmbedding := embeddingResp.Embedding

	// Calculate cosine similarity
	similarity := cosineSimilarity(objectiveEmbedding, methodEmbedding)
	return similarity, nil
}

// cosineSimilarity calculates cosine similarity between two vectors.
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0.0 || normB == 0.0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// formatTimeSince returns a human-readable string describing time elapsed.
func formatTimeSince(t time.Time) string {
	if t.IsZero() {
		return "never"
	}

	duration := time.Since(t)
	days := duration.Hours() / 24

	if days < 1 {
		if duration.Hours() < 1 {
			return "recently"
		}
		return fmt.Sprintf("%.0f hours ago", duration.Hours())
	}

	if days < 7 {
		return fmt.Sprintf("%.0f days ago", days)
	}

	if days < 30 {
		weeks := days / 7
		return fmt.Sprintf("%.0f weeks ago", weeks)
	}

	months := days / 30
	return fmt.Sprintf("%.0f months ago", months)
}