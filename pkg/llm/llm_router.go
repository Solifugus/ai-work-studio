package llm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/yourusername/ai-work-studio/pkg/mcp"
)

// LLMServiceInterface defines the interface needed by the router.
type LLMServiceInterface interface {
	Execute(ctx context.Context, params mcp.ServiceParams) mcp.ServiceResult
}

// TaskComplexity represents the complexity level of a task.
type TaskComplexity int

const (
	// TaskComplexitySimple for basic tasks like simple Q&A, formatting
	TaskComplexitySimple TaskComplexity = iota
	// TaskComplexityModerate for tasks requiring reasoning, analysis
	TaskComplexityModerate
	// TaskComplexityComplex for tasks requiring deep reasoning, creativity
	TaskComplexityComplex
)

// QualityRequirement represents the quality level needed for a task.
type QualityRequirement int

const (
	// QualityBasic for tasks where accuracy is less critical
	QualityBasic QualityRequirement = iota
	// QualityStandard for most tasks requiring good accuracy
	QualityStandard
	// QualityPremium for critical tasks requiring highest accuracy
	QualityPremium
)

// TaskRequest represents a request for LLM routing.
type TaskRequest struct {
	// Prompt is the text to be processed
	Prompt string

	// MaxTokens is the maximum number of tokens to generate
	MaxTokens int

	// Temperature controls randomness in generation
	Temperature float64

	// TaskType describes the type of task (e.g., "analysis", "generation", "qa")
	TaskType string

	// QualityRequired specifies the quality level needed
	QualityRequired QualityRequirement

	// BudgetConstraint is the maximum cost willing to spend
	BudgetConstraint *float64

	// PreferredProvider can override automatic selection
	PreferredProvider string

	// Metadata contains additional context about the task
	Metadata map[string]interface{}
}

// TaskAssessment contains the router's assessment of a task.
type TaskAssessment struct {
	// Complexity is the estimated complexity level
	Complexity TaskComplexity

	// EstimatedTokens is the estimated token usage
	EstimatedTokens int

	// QualityNeeded is the assessed quality requirement
	QualityNeeded QualityRequirement

	// RecommendedModels are the models that could handle this task
	RecommendedModels []ModelRecommendation

	// Reasoning explains why this assessment was made
	Reasoning string
}

// ModelRecommendation represents a model recommendation with scoring.
type ModelRecommendation struct {
	// Provider name (e.g., "anthropic", "openai")
	Provider string

	// Model name (e.g., "claude-3-haiku")
	Model string

	// EstimatedCost is the predicted cost for this task
	EstimatedCost float64

	// QualityScore is the expected quality (0-1, higher is better)
	QualityScore float64

	// SpeedScore is the expected speed (0-1, higher is faster)
	SpeedScore float64

	// OverallScore is the weighted combination of factors
	OverallScore float64

	// Reasoning explains why this model was recommended
	Reasoning string
}

// ModelPerformance tracks how well models perform on different task types.
type ModelPerformance struct {
	Provider      string
	Model         string
	TaskType      string
	SuccessRate   float64 // 0-1
	AverageRating float64 // 1-10 user/system rating
	AverageCost   float64
	AverageLatency time.Duration
	SampleCount   int
	LastUpdated   time.Time
}

// Router provides intelligent LLM routing based on task requirements and learning.
type Router struct {
	llmService  LLMServiceInterface
	performance map[string]*ModelPerformance // key: provider_model_tasktype
	mu          sync.RWMutex
	config      RouterConfig
}

// RouterConfig contains configuration for the router.
type RouterConfig struct {
	// DefaultQuality is used when quality requirement is not specified
	DefaultQuality QualityRequirement

	// MaxCostPerRequest is the default budget constraint
	MaxCostPerRequest float64

	// QualityWeight affects how much quality impacts model selection (0-1)
	QualityWeight float64

	// CostWeight affects how much cost impacts model selection (0-1)
	CostWeight float64

	// SpeedWeight affects how much speed impacts model selection (0-1)
	SpeedWeight float64

	// ConservativeBias starts with higher quality models until learning occurs
	ConservativeBias float64

	// MinSampleSize before trusting performance metrics
	MinSampleSize int
}

// DefaultRouterConfig returns sensible defaults for router configuration.
func DefaultRouterConfig() RouterConfig {
	return RouterConfig{
		DefaultQuality:    QualityStandard,
		MaxCostPerRequest: 0.10, // $0.10 per request by default
		QualityWeight:     0.5,  // 50% weight for quality
		CostWeight:        0.3,  // 30% weight for cost
		SpeedWeight:       0.2,  // 20% weight for speed
		ConservativeBias:  0.2,  // Start conservative, prefer quality over cost
		MinSampleSize:     5,    // Need 5 samples before trusting metrics
	}
}

// NewRouter creates a new LLM router.
func NewRouter(llmService LLMServiceInterface, config ...RouterConfig) *Router {
	cfg := DefaultRouterConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return &Router{
		llmService:  llmService,
		performance: make(map[string]*ModelPerformance),
		config:      cfg,
	}
}

// Route selects the best model for a task and executes it.
func (r *Router) Route(ctx context.Context, req TaskRequest) (*RoutingResult, error) {
	// Step 1: Assess the task
	assessment := r.assessTask(req)

	// Step 2: Get available models and their capabilities
	models := r.getAvailableModels()

	// Step 3: Score each model for this task
	recommendations := r.scoreModels(models, assessment, req)

	if len(recommendations) == 0 {
		return nil, fmt.Errorf("no suitable models available for this task")
	}

	// Step 4: Select the best model
	selectedModel := recommendations[0] // Already sorted by score

	// Step 5: Execute the task
	result, err := r.executeTask(ctx, req, selectedModel)
	if err != nil {
		return nil, fmt.Errorf("task execution failed: %w", err)
	}

	return &RoutingResult{
		Assessment:        assessment,
		SelectedModel:     selectedModel,
		AlternativeModels: recommendations[1:],
		ExecutionResult:   result,
		ExecutionTime:     time.Now(),
	}, nil
}

// RoutingResult contains the complete result of routing and execution.
type RoutingResult struct {
	Assessment        TaskAssessment
	SelectedModel     ModelRecommendation
	AlternativeModels []ModelRecommendation
	ExecutionResult   *mcp.CompletionResponse
	ExecutionTime     time.Time
	UserRating        float64 // Set later via feedback
}

// assessTask analyzes a task to determine its complexity and requirements.
func (r *Router) assessTask(req TaskRequest) TaskAssessment {
	// Estimate token usage
	estimatedTokens := r.estimateTokenUsage(req.Prompt, req.MaxTokens)

	// Assess complexity based on prompt characteristics
	complexity := r.assessComplexity(req.Prompt, req.TaskType)

	// Determine quality needed (use provided or infer from task type)
	qualityNeeded := req.QualityRequired
	if qualityNeeded == QualityBasic && r.inferQualityFromTaskType(req.TaskType) > QualityBasic {
		qualityNeeded = r.inferQualityFromTaskType(req.TaskType)
	}

	// Generate reasoning for the assessment
	reasoning := r.generateAssessmentReasoning(complexity, estimatedTokens, qualityNeeded, req.TaskType)

	return TaskAssessment{
		Complexity:      complexity,
		EstimatedTokens: estimatedTokens,
		QualityNeeded:   qualityNeeded,
		Reasoning:       reasoning,
	}
}

// estimateTokenUsage provides a rough estimate of token usage.
func (r *Router) estimateTokenUsage(prompt string, maxTokens int) int {
	// More accurate estimation: 1 token ≈ 3.5 characters for English text
	// Add word count factor for better accuracy
	words := len(strings.Fields(prompt))
	promptTokens := max(len(prompt)/3, words) // Take the larger of char-based or word-based estimate

	// If maxTokens is set, use it; otherwise estimate output length
	outputTokens := maxTokens
	if outputTokens == 0 {
		// Estimate output based on input length with minimum reasonable response
		baseOutput := max(12, int(float64(promptTokens)*2.5)) // Minimum 12 tokens for any response
		outputTokens = baseOutput
	}

	return promptTokens + outputTokens
}

// max returns the larger of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// assessComplexity determines task complexity based on prompt analysis.
func (r *Router) assessComplexity(prompt, taskType string) TaskComplexity {
	prompt = strings.ToLower(prompt)
	taskType = strings.ToLower(taskType)

	// Complexity indicators
	complexityScore := 0

	// Simple indicators (negative points)
	simpleIndicators := []string{
		"what is", "define", "list", "summary", "simple", "quick",
	}
	for _, indicator := range simpleIndicators {
		if strings.Contains(prompt, indicator) {
			complexityScore -= 1
		}
	}

	// Moderate indicators
	moderateIndicators := []string{
		"analyze", "compare", "explain", "describe", "how", "why",
	}
	for _, indicator := range moderateIndicators {
		if strings.Contains(prompt, indicator) {
			complexityScore += 1
		}
	}

	// Complex indicators (extra points)
	complexIndicators := []string{
		"reasoning", "logic", "creative", "novel", "complex", "deep",
		"philosophical", "strategic", "multi-step", "comprehensive",
	}
	for _, indicator := range complexIndicators {
		if strings.Contains(prompt, indicator) {
			complexityScore += 2
		}
	}

	// Task type influence
	switch taskType {
	case "qa", "lookup", "format":
		complexityScore -= 1
	case "analysis", "generation", "reasoning":
		complexityScore += 1
	case "creative", "research", "complex_reasoning":
		complexityScore += 2
	}

	// Length influence (longer prompts tend to be more complex)
	if len(prompt) > 1000 {
		complexityScore += 1
	}
	if len(prompt) > 2000 {
		complexityScore += 1
	}

	// Convert score to complexity level
	if complexityScore <= -1 {
		return TaskComplexitySimple
	} else if complexityScore <= 2 {
		return TaskComplexityModerate
	} else {
		return TaskComplexityComplex
	}
}

// inferQualityFromTaskType infers quality requirements from task type.
func (r *Router) inferQualityFromTaskType(taskType string) QualityRequirement {
	taskType = strings.ToLower(taskType)

	switch taskType {
	case "format", "simple_qa", "list":
		return QualityBasic
	case "analysis", "reasoning", "generation":
		return QualityStandard
	case "creative", "research", "complex_reasoning", "critical":
		return QualityPremium
	default:
		return r.config.DefaultQuality
	}
}

// generateAssessmentReasoning creates human-readable reasoning for the assessment.
func (r *Router) generateAssessmentReasoning(complexity TaskComplexity, tokens int, quality QualityRequirement, taskType string) string {
	parts := []string{}

	// Complexity reasoning
	switch complexity {
	case TaskComplexitySimple:
		parts = append(parts, "simple task requiring basic processing")
	case TaskComplexityModerate:
		parts = append(parts, "moderate task requiring reasoning")
	case TaskComplexityComplex:
		parts = append(parts, "complex task requiring deep analysis")
	}

	// Token reasoning
	if tokens < 500 {
		parts = append(parts, "small token usage")
	} else if tokens < 2000 {
		parts = append(parts, "moderate token usage")
	} else {
		parts = append(parts, "high token usage")
	}

	// Quality reasoning
	switch quality {
	case QualityBasic:
		parts = append(parts, "basic quality sufficient")
	case QualityStandard:
		parts = append(parts, "standard quality needed")
	case QualityPremium:
		parts = append(parts, "premium quality required")
	}

	// Task type
	if taskType != "" {
		parts = append(parts, fmt.Sprintf("task type: %s", taskType))
	}

	return strings.Join(parts, ", ")
}

// getAvailableModels returns the models available from the LLM service.
func (r *Router) getAvailableModels() []ModelInfo {
	// This would need to interface with the LLM service to get available providers and models
	// For now, we'll return a hardcoded set based on the MCP LLM service implementation

	models := []ModelInfo{
		{
			Provider:     "anthropic",
			Model:        "claude-3-sonnet",
			InputCost:    3.0,
			OutputCost:   15.0,
			MaxTokens:    4096,
			ContextSize:  200000,
			QualityTier:  QualityPremium,
			SpeedTier:    2, // 1=fastest, 3=slowest
		},
		{
			Provider:     "anthropic",
			Model:        "claude-3-haiku",
			InputCost:    0.25,
			OutputCost:   1.25,
			MaxTokens:    4096,
			ContextSize:  200000,
			QualityTier:  QualityStandard,
			SpeedTier:    1, // Fastest
		},
		{
			Provider:     "openai",
			Model:        "gpt-4",
			InputCost:    30.0,
			OutputCost:   60.0,
			MaxTokens:    4096,
			ContextSize:  8192,
			QualityTier:  QualityPremium,
			SpeedTier:    3, // Slowest
		},
		{
			Provider:     "openai",
			Model:        "gpt-3.5-turbo",
			InputCost:    0.5,
			OutputCost:   1.5,
			MaxTokens:    4096,
			ContextSize:  16385,
			QualityTier:  QualityStandard,
			SpeedTier:    1, // Fastest
		},
		{
			Provider:     "local",
			Model:        "local-llama",
			InputCost:    0.0,
			OutputCost:   0.0,
			MaxTokens:    4096,
			ContextSize:  4096,
			QualityTier:  QualityBasic,
			SpeedTier:    2, // Medium
		},
	}

	return models
}

// ModelInfo represents information about an available model.
type ModelInfo struct {
	Provider     string
	Model        string
	InputCost    float64 // Cost per 1K tokens
	OutputCost   float64 // Cost per 1K tokens
	MaxTokens    int
	ContextSize  int
	QualityTier  QualityRequirement
	SpeedTier    int // 1=fastest, 3=slowest
}

// scoreModels scores each available model for a given task.
func (r *Router) scoreModels(models []ModelInfo, assessment TaskAssessment, req TaskRequest) []ModelRecommendation {
	var recommendations []ModelRecommendation

	for _, model := range models {
		// Skip models that can't handle the token requirements
		if assessment.EstimatedTokens > model.ContextSize {
			continue
		}

		// Calculate estimated cost
		inputTokens := len(req.Prompt) / 4 // Rough estimate
		outputTokens := assessment.EstimatedTokens - inputTokens
		estimatedCost := (float64(inputTokens)*model.InputCost + float64(outputTokens)*model.OutputCost) / 1000.0

		// Skip models that exceed budget constraint
		if req.BudgetConstraint != nil && estimatedCost > *req.BudgetConstraint {
			continue
		}

		// Calculate quality score (0-1)
		qualityScore := r.calculateQualityScore(model, assessment.QualityNeeded)

		// Calculate speed score (0-1, higher is faster)
		speedScore := float64(4-model.SpeedTier) / 3.0 // Convert 1-3 to 1.0-0.33

		// Get historical performance if available
		perf := r.getPerformance(model.Provider, model.Model, req.TaskType)

		// Apply learning from historical performance
		if perf != nil && perf.SampleCount >= r.config.MinSampleSize {
			// Use learned performance metrics
			qualityScore = (qualityScore + perf.AverageRating/10.0) / 2.0
		} else {
			// Apply conservative bias for unknown models
			if model.QualityTier > assessment.QualityNeeded {
				qualityScore += r.config.ConservativeBias
				if qualityScore > 1.0 {
					qualityScore = 1.0
				}
			}
		}

		// Calculate cost score (0-1, higher is cheaper)
		costScore := r.calculateCostScore(estimatedCost, req.BudgetConstraint)

		// Calculate overall score using weighted combination
		overallScore := (qualityScore * r.config.QualityWeight) +
			(costScore * r.config.CostWeight) +
			(speedScore * r.config.SpeedWeight)

		// Generate reasoning
		reasoning := r.generateRecommendationReasoning(model, qualityScore, costScore, speedScore, estimatedCost)

		recommendation := ModelRecommendation{
			Provider:      model.Provider,
			Model:         model.Model,
			EstimatedCost: estimatedCost,
			QualityScore:  qualityScore,
			SpeedScore:    speedScore,
			OverallScore:  overallScore,
			Reasoning:     reasoning,
		}

		recommendations = append(recommendations, recommendation)
	}

	// Sort by overall score (highest first)
	for i := 0; i < len(recommendations)-1; i++ {
		for j := i + 1; j < len(recommendations); j++ {
			if recommendations[i].OverallScore < recommendations[j].OverallScore {
				recommendations[i], recommendations[j] = recommendations[j], recommendations[i]
			}
		}
	}

	return recommendations
}

// calculateQualityScore calculates how well a model matches quality requirements.
func (r *Router) calculateQualityScore(model ModelInfo, required QualityRequirement) float64 {
	qualityDiff := int(model.QualityTier) - int(required)

	// Perfect match gets 1.0
	if qualityDiff == 0 {
		return 1.0
	}

	// Higher quality than needed gets high score but not perfect (slight cost penalty)
	if qualityDiff > 0 {
		return 0.9
	}

	// Lower quality gets penalty
	if qualityDiff == -1 {
		return 0.6 // One tier down
	} else {
		return 0.3 // Two or more tiers down
	}
}

// calculateCostScore calculates cost efficiency score.
func (r *Router) calculateCostScore(estimatedCost float64, budgetConstraint *float64) float64 {
	maxBudget := r.config.MaxCostPerRequest
	if budgetConstraint != nil {
		maxBudget = *budgetConstraint
	}

	if estimatedCost > maxBudget {
		return 0.0 // Over budget
	}

	// Score based on how much of budget is used (less usage = higher score)
	usageRatio := estimatedCost / maxBudget
	return 1.0 - usageRatio
}

// generateRecommendationReasoning creates human-readable reasoning for model recommendation.
func (r *Router) generateRecommendationReasoning(model ModelInfo, qualityScore, costScore, speedScore, estimatedCost float64) string {
	parts := []string{}

	// Quality reasoning
	if qualityScore >= 0.9 {
		parts = append(parts, "excellent quality match")
	} else if qualityScore >= 0.7 {
		parts = append(parts, "good quality match")
	} else {
		parts = append(parts, "lower quality option")
	}

	// Cost reasoning
	if costScore >= 0.8 {
		parts = append(parts, fmt.Sprintf("very cost-effective ($%.4f)", estimatedCost))
	} else if costScore >= 0.5 {
		parts = append(parts, fmt.Sprintf("reasonably priced ($%.4f)", estimatedCost))
	} else {
		parts = append(parts, fmt.Sprintf("higher cost ($%.4f)", estimatedCost))
	}

	// Speed reasoning
	if speedScore >= 0.8 {
		parts = append(parts, "fast response")
	} else if speedScore >= 0.5 {
		parts = append(parts, "moderate speed")
	} else {
		parts = append(parts, "slower but higher quality")
	}

	return strings.Join(parts, ", ")
}

// executeTask executes the task using the selected model.
func (r *Router) executeTask(ctx context.Context, req TaskRequest, model ModelRecommendation) (*mcp.CompletionResponse, error) {
	// Prepare parameters for the LLM service
	params := mcp.ServiceParams{
		"operation":  "complete",
		"prompt":     req.Prompt,
		"provider":   model.Provider,
		"model":      model.Model,
		"max_tokens": req.MaxTokens,
	}

	if req.Temperature > 0 {
		params["temperature"] = req.Temperature
	}

	// Execute using the LLM service
	result := r.llmService.Execute(ctx, params)
	if result.Error != nil {
		return nil, fmt.Errorf("LLM service execution failed: %w", result.Error)
	}

	// Extract completion response
	completion, ok := result.Data.(*mcp.CompletionResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected response type from LLM service")
	}

	return completion, nil
}

// getPerformance retrieves historical performance data for a model/task combination.
func (r *Router) getPerformance(provider, model, taskType string) *ModelPerformance {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := fmt.Sprintf("%s_%s_%s", provider, model, taskType)
	return r.performance[key]
}

// RecordPerformance records the performance of a model on a task for learning.
func (r *Router) RecordPerformance(provider, model, taskType string, cost float64, rating float64, latency time.Duration, successful bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%s_%s_%s", provider, model, taskType)

	perf, exists := r.performance[key]
	if !exists {
		perf = &ModelPerformance{
			Provider:    provider,
			Model:       model,
			TaskType:    taskType,
			SuccessRate: 0.0,
			SampleCount: 0,
		}
		r.performance[key] = perf
	}

	// Update metrics using incremental formulas
	perf.SampleCount++

	// Update success rate
	if successful {
		perf.SuccessRate = (perf.SuccessRate*float64(perf.SampleCount-1) + 1.0) / float64(perf.SampleCount)
	} else {
		perf.SuccessRate = (perf.SuccessRate*float64(perf.SampleCount-1) + 0.0) / float64(perf.SampleCount)
	}

	// Update average rating (only if rating is provided and valid)
	if rating >= 1.0 && rating <= 10.0 {
		if perf.SampleCount == 1 {
			perf.AverageRating = rating
		} else {
			perf.AverageRating = (perf.AverageRating*float64(perf.SampleCount-1) + rating) / float64(perf.SampleCount)
		}
	}

	// Update average cost
	if perf.SampleCount == 1 {
		perf.AverageCost = cost
	} else {
		perf.AverageCost = (perf.AverageCost*float64(perf.SampleCount-1) + cost) / float64(perf.SampleCount)
	}

	// Update average latency
	if perf.SampleCount == 1 {
		perf.AverageLatency = latency
	} else {
		totalLatency := perf.AverageLatency*time.Duration(perf.SampleCount-1) + latency
		perf.AverageLatency = totalLatency / time.Duration(perf.SampleCount)
	}

	perf.LastUpdated = time.Now()
}

// GetPerformanceStats returns performance statistics for learning analysis.
func (r *Router) GetPerformanceStats() map[string]*ModelPerformance {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	stats := make(map[string]*ModelPerformance)
	for key, perf := range r.performance {
		stats[key] = &ModelPerformance{
			Provider:       perf.Provider,
			Model:         perf.Model,
			TaskType:      perf.TaskType,
			SuccessRate:   perf.SuccessRate,
			AverageRating: perf.AverageRating,
			AverageCost:   perf.AverageCost,
			AverageLatency: perf.AverageLatency,
			SampleCount:   perf.SampleCount,
			LastUpdated:   perf.LastUpdated,
		}
	}

	return stats
}

// EstimateCost provides cost estimation without execution.
func (r *Router) EstimateCost(req TaskRequest) (*CostEstimate, error) {
	assessment := r.assessTask(req)
	models := r.getAvailableModels()
	recommendations := r.scoreModels(models, assessment, req)

	if len(recommendations) == 0 {
		return nil, fmt.Errorf("no suitable models available for cost estimation")
	}

	// Get cost estimates for top 3 recommendations
	estimates := make([]ModelCostEstimate, 0, min(3, len(recommendations)))
	for i := 0; i < min(3, len(recommendations)); i++ {
		rec := recommendations[i]
		estimates = append(estimates, ModelCostEstimate{
			Provider:      rec.Provider,
			Model:         rec.Model,
			EstimatedCost: rec.EstimatedCost,
			QualityScore:  rec.QualityScore,
		})
	}

	return &CostEstimate{
		Assessment: assessment,
		Options:    estimates,
	}, nil
}

// CostEstimate provides cost estimation results.
type CostEstimate struct {
	Assessment TaskAssessment
	Options    []ModelCostEstimate
}

// ModelCostEstimate represents cost estimation for a specific model.
type ModelCostEstimate struct {
	Provider      string
	Model         string
	EstimatedCost float64
	QualityScore  float64
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// String methods for enums
func (tc TaskComplexity) String() string {
	switch tc {
	case TaskComplexitySimple:
		return "simple"
	case TaskComplexityModerate:
		return "moderate"
	case TaskComplexityComplex:
		return "complex"
	default:
		return "unknown"
	}
}

func (qr QualityRequirement) String() string {
	switch qr {
	case QualityBasic:
		return "basic"
	case QualityStandard:
		return "standard"
	case QualityPremium:
		return "premium"
	default:
		return "unknown"
	}
}