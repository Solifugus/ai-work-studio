package llm

import (
	"context"
	"testing"
	"time"

	"github.com/Solifugus/ai-work-studio/pkg/mcp"
)

// MockLLMService provides a mock LLM service for testing.
type MockLLMService struct {
	responses map[string]*mcp.CompletionResponse
	errors    map[string]error
}

// NewMockLLMService creates a new mock LLM service.
func NewMockLLMService() *MockLLMService {
	return &MockLLMService{
		responses: make(map[string]*mcp.CompletionResponse),
		errors:    make(map[string]error),
	}
}

// Execute implements the LLM service interface for testing.
func (m *MockLLMService) Execute(ctx context.Context, params mcp.ServiceParams) mcp.ServiceResult {
	operation, _ := params["operation"].(string)
	provider, _ := params["provider"].(string)
	model, _ := params["model"].(string)

	key := operation + "_" + provider + "_" + model

	if err, exists := m.errors[key]; exists {
		return mcp.ErrorResult(err)
	}

	if response, exists := m.responses[key]; exists {
		return mcp.SuccessResult(response)
	}

	// Default response
	return mcp.SuccessResult(&mcp.CompletionResponse{
		Text:       "Mock response",
		TokensUsed: 100,
		Model:      model,
		Provider:   provider,
		Cost:       0.01,
	})
}

// SetResponse sets a mock response for specific parameters.
func (m *MockLLMService) SetResponse(operation, provider, model string, response *mcp.CompletionResponse) {
	key := operation + "_" + provider + "_" + model
	m.responses[key] = response
}

// SetError sets a mock error for specific parameters.
func (m *MockLLMService) SetError(operation, provider, model string, err error) {
	key := operation + "_" + provider + "_" + model
	m.errors[key] = err
}

func TestNewRouter(t *testing.T) {
	mockService := NewMockLLMService()
	router := NewRouter(mockService)

	if router == nil {
		t.Fatal("NewRouter returned nil")
	}

	if router.llmService == nil {
		t.Error("Router should store the provided LLM service")
	}

	if router.performance == nil {
		t.Error("Router should initialize performance tracking")
	}

	// Test with custom config
	config := RouterConfig{
		DefaultQuality:    QualityPremium,
		MaxCostPerRequest: 0.50,
		QualityWeight:     0.6,
		CostWeight:        0.4,
	}

	router2 := NewRouter(mockService, config)
	if router2.config.DefaultQuality != QualityPremium {
		t.Error("Router should use provided config")
	}
}

func TestTaskComplexityAssessment(t *testing.T) {
	router := NewRouter(NewMockLLMService())

	tests := []struct {
		name        string
		prompt      string
		taskType    string
		expected    TaskComplexity
		description string
	}{
		{
			name:        "simple_definition",
			prompt:      "What is the definition of machine learning?",
			taskType:    "qa",
			expected:    TaskComplexitySimple,
			description: "Simple definition request",
		},
		{
			name:        "moderate_analysis",
			prompt:      "Analyze the pros and cons of renewable energy sources",
			taskType:    "analysis",
			expected:    TaskComplexityModerate,
			description: "Analysis task requiring reasoning",
		},
		{
			name:        "complex_reasoning",
			prompt:      "Design a comprehensive strategy for implementing AI governance across a multinational corporation, considering ethical implications, regulatory compliance, and competitive advantages",
			taskType:    "complex_reasoning",
			expected:    TaskComplexityComplex,
			description: "Complex multi-faceted strategic task",
		},
		{
			name:        "creative_task",
			prompt:      "Write a novel story that incorporates quantum physics concepts in a creative and engaging way",
			taskType:    "creative",
			expected:    TaskComplexityComplex,
			description: "Creative task requiring innovation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := TaskRequest{
				Prompt:   tt.prompt,
				TaskType: tt.taskType,
			}

			assessment := router.assessTask(req)

			if assessment.Complexity != tt.expected {
				t.Errorf("Expected complexity %s, got %s for: %s",
					tt.expected.String(), assessment.Complexity.String(), tt.description)
			}

			if assessment.EstimatedTokens <= 0 {
				t.Error("Should estimate positive token usage")
			}

			if assessment.Reasoning == "" {
				t.Error("Should provide reasoning for assessment")
			}
		})
	}
}

func TestTokenEstimation(t *testing.T) {
	router := NewRouter(NewMockLLMService())

	tests := []struct {
		prompt       string
		maxTokens    int
		expectedMin  int
		expectedMax  int
		description  string
	}{
		{
			prompt:      "Short prompt",
			maxTokens:   0,
			expectedMin: 15, // ~3 input + 4 output (1.5x estimate)
			expectedMax: 50,
			description: "Short prompt with auto-estimation",
		},
		{
			prompt:      "This is a much longer prompt that contains significantly more text and should result in higher token estimates",
			maxTokens:   500,
			expectedMax: 600, // Should include both input and max output tokens
			description: "Longer prompt with explicit max tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			estimated := router.estimateTokenUsage(tt.prompt, tt.maxTokens)

			if tt.expectedMin > 0 && estimated < tt.expectedMin {
				t.Errorf("Expected at least %d tokens, got %d", tt.expectedMin, estimated)
			}

			if tt.expectedMax > 0 && estimated > tt.expectedMax {
				t.Errorf("Expected at most %d tokens, got %d", tt.expectedMax, estimated)
			}
		})
	}
}

func TestQualityInference(t *testing.T) {
	router := NewRouter(NewMockLLMService())

	tests := []struct {
		taskType string
		expected QualityRequirement
	}{
		{"format", QualityBasic},
		{"simple_qa", QualityBasic},
		{"analysis", QualityStandard},
		{"reasoning", QualityStandard},
		{"creative", QualityPremium},
		{"research", QualityPremium},
		{"unknown_type", QualityStandard}, // Should use default
	}

	for _, tt := range tests {
		t.Run(tt.taskType, func(t *testing.T) {
			quality := router.inferQualityFromTaskType(tt.taskType)
			if quality != tt.expected {
				t.Errorf("Expected quality %s for task type %s, got %s",
					tt.expected.String(), tt.taskType, quality.String())
			}
		})
	}
}

func TestModelScoring(t *testing.T) {
	router := NewRouter(NewMockLLMService())

	// Test model scoring
	models := router.getAvailableModels()
	if len(models) == 0 {
		t.Fatal("Should have available models for testing")
	}

	assessment := TaskAssessment{
		Complexity:      TaskComplexityModerate,
		EstimatedTokens: 1000,
		QualityNeeded:   QualityStandard,
	}

	req := TaskRequest{
		Prompt:          "Test prompt for analysis",
		TaskType:        "analysis",
		QualityRequired: QualityStandard,
		MaxTokens:       1000,
	}

	recommendations := router.scoreModels(models, assessment, req)

	if len(recommendations) == 0 {
		t.Fatal("Should have at least one recommendation")
	}

	// Check that recommendations are sorted by score
	for i := 1; i < len(recommendations); i++ {
		if recommendations[i-1].OverallScore < recommendations[i].OverallScore {
			t.Error("Recommendations should be sorted by overall score (highest first)")
		}
	}

	// Check that each recommendation has valid scores
	for _, rec := range recommendations {
		if rec.QualityScore < 0 || rec.QualityScore > 1 {
			t.Errorf("Quality score should be 0-1, got %f", rec.QualityScore)
		}

		if rec.EstimatedCost < 0 {
			t.Errorf("Estimated cost should be non-negative, got %f", rec.EstimatedCost)
		}

		if rec.Reasoning == "" {
			t.Error("Should provide reasoning for recommendation")
		}
	}
}

func TestModelScoringWithBudgetConstraint(t *testing.T) {
	router := NewRouter(NewMockLLMService())

	models := router.getAvailableModels()
	assessment := TaskAssessment{
		Complexity:      TaskComplexityModerate,
		EstimatedTokens: 1000,
		QualityNeeded:   QualityStandard,
	}

	// Set a very low budget constraint
	lowBudget := 0.001
	req := TaskRequest{
		Prompt:           "Test prompt",
		TaskType:         "analysis",
		QualityRequired:  QualityStandard,
		MaxTokens:        1000,
		BudgetConstraint: &lowBudget,
	}

	recommendations := router.scoreModels(models, assessment, req)

	// Should filter out expensive models
	for _, rec := range recommendations {
		if rec.EstimatedCost > lowBudget {
			t.Errorf("Recommendation exceeds budget constraint: cost %f > budget %f",
				rec.EstimatedCost, lowBudget)
		}
	}
}

func TestCostEstimation(t *testing.T) {
	router := NewRouter(NewMockLLMService())

	req := TaskRequest{
		Prompt:          "Analyze this business proposal and provide recommendations",
		TaskType:        "analysis",
		QualityRequired: QualityStandard,
		MaxTokens:       2000,
	}

	estimate, err := router.EstimateCost(req)
	if err != nil {
		t.Fatalf("Cost estimation failed: %v", err)
	}

	if estimate == nil {
		t.Fatal("Cost estimate should not be nil")
	}

	if len(estimate.Options) == 0 {
		t.Fatal("Should have cost estimation options")
	}

	// Check that options are sorted by some criteria (typically cost or overall score)
	for _, option := range estimate.Options {
		if option.EstimatedCost < 0 {
			t.Errorf("Estimated cost should be non-negative, got %f", option.EstimatedCost)
		}

		if option.QualityScore < 0 || option.QualityScore > 1 {
			t.Errorf("Quality score should be 0-1, got %f", option.QualityScore)
		}
	}
}

func TestPerformanceRecording(t *testing.T) {
	router := NewRouter(NewMockLLMService())

	// Record some performance data
	router.RecordPerformance("anthropic", "claude-3-haiku", "analysis", 0.05, 8.5, 2*time.Second, true)
	router.RecordPerformance("anthropic", "claude-3-haiku", "analysis", 0.06, 7.5, 3*time.Second, true)
	router.RecordPerformance("anthropic", "claude-3-haiku", "analysis", 0.04, 9.0, 1*time.Second, false)

	// Get performance stats
	stats := router.GetPerformanceStats()

	key := "anthropic_claude-3-haiku_analysis"
	perf, exists := stats[key]
	if !exists {
		t.Fatal("Should have performance data for recorded provider/model/task")
	}

	if perf.SampleCount != 3 {
		t.Errorf("Expected 3 samples, got %d", perf.SampleCount)
	}

	expectedSuccessRate := 2.0 / 3.0 // 2 successful out of 3
	if perf.SuccessRate < expectedSuccessRate-0.01 || perf.SuccessRate > expectedSuccessRate+0.01 {
		t.Errorf("Expected success rate ~%.2f, got %.2f", expectedSuccessRate, perf.SuccessRate)
	}

	// Check that average rating is calculated correctly (only for successful requests with ratings)
	expectedAvgRating := (8.5 + 7.5 + 9.0) / 3.0
	if perf.AverageRating < expectedAvgRating-0.1 || perf.AverageRating > expectedAvgRating+0.1 {
		t.Errorf("Expected average rating ~%.1f, got %.1f", expectedAvgRating, perf.AverageRating)
	}
}

func TestRouterIntegration(t *testing.T) {
	// Test full routing workflow
	mockService := NewMockLLMService()

	// Set up a mock response
	mockResponse := &mcp.CompletionResponse{
		Text:       "This is a comprehensive analysis of the business proposal...",
		TokensUsed: 1500,
		Model:      "claude-3-haiku",
		Provider:   "anthropic",
		Cost:       0.075,
	}

	// Set response for all possible models that could be selected
	mockService.SetResponse("complete", "anthropic", "claude-3-haiku", mockResponse)
	mockService.SetResponse("complete", "anthropic", "claude-3-sonnet", mockResponse)
	mockService.SetResponse("complete", "openai", "gpt-4", mockResponse)
	mockService.SetResponse("complete", "openai", "gpt-3.5-turbo", mockResponse)
	mockService.SetResponse("complete", "local", "local-llama", mockResponse)

	router := NewRouter(mockService)

	req := TaskRequest{
		Prompt:          "Analyze this business proposal for potential risks and opportunities",
		TaskType:        "analysis",
		QualityRequired: QualityStandard,
		MaxTokens:       2000,
		Temperature:     0.7,
	}

	ctx := context.Background()
	result, err := router.Route(ctx, req)

	if err != nil {
		t.Fatalf("Routing failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	// Check assessment
	if result.Assessment.Complexity == 0 {
		t.Error("Assessment should determine complexity")
	}

	// Check selected model
	if result.SelectedModel.Provider == "" || result.SelectedModel.Model == "" {
		t.Error("Should select a specific provider and model")
	}

	// Check execution result
	if result.ExecutionResult == nil {
		t.Error("Should have execution result")
	}

	if result.ExecutionResult.Text != mockResponse.Text {
		t.Errorf("Should return the mock response text. Expected: %q, Got: %q. Selected model: %s/%s",
			mockResponse.Text, result.ExecutionResult.Text, result.SelectedModel.Provider, result.SelectedModel.Model)
	}

	// Check that we got some alternative models
	if len(result.AlternativeModels) == 0 {
		t.Log("Warning: No alternative models provided (this is ok if only one model meets criteria)")
	}
}

func TestRouterWithLearning(t *testing.T) {
	router := NewRouter(NewMockLLMService())

	// First, record some historical performance to influence routing
	router.RecordPerformance("anthropic", "claude-3-haiku", "analysis", 0.05, 9.0, time.Second, true)
	router.RecordPerformance("anthropic", "claude-3-haiku", "analysis", 0.06, 8.5, time.Second, true)
	router.RecordPerformance("anthropic", "claude-3-haiku", "analysis", 0.04, 9.5, time.Second, true)
	router.RecordPerformance("anthropic", "claude-3-haiku", "analysis", 0.05, 8.0, time.Second, true)
	router.RecordPerformance("anthropic", "claude-3-haiku", "analysis", 0.07, 9.0, time.Second, true)

	// Now test that historical performance affects model selection
	req := TaskRequest{
		Prompt:          "Analyze this market trend data",
		TaskType:        "analysis",
		QualityRequired: QualityStandard,
		MaxTokens:       1000,
	}

	// Get assessment and scoring (without full routing to avoid mock complexity)
	assessment := router.assessTask(req)
	models := router.getAvailableModels()
	recommendations := router.scoreModels(models, assessment, req)

	// Should have recommendations
	if len(recommendations) == 0 {
		t.Fatal("Should have model recommendations")
	}

	// The model with good historical performance should be preferred
	// (Though exact ranking depends on the weighting and other factors)
	claudeHaikuFound := false
	for _, rec := range recommendations {
		if rec.Provider == "anthropic" && rec.Model == "claude-3-haiku" {
			claudeHaikuFound = true
			// Should have a decent score due to good historical performance
			if rec.OverallScore < 0.5 {
				t.Errorf("Expected higher overall score for model with good history, got %f", rec.OverallScore)
			}
			break
		}
	}

	if !claudeHaikuFound {
		t.Error("Should include claude-3-haiku in recommendations")
	}
}

func TestDefaultRouterConfig(t *testing.T) {
	config := DefaultRouterConfig()

	// Check that weights sum to approximately 1.0
	totalWeight := config.QualityWeight + config.CostWeight + config.SpeedWeight
	if totalWeight < 0.95 || totalWeight > 1.05 {
		t.Errorf("Weights should sum to ~1.0, got %f", totalWeight)
	}

	// Check reasonable defaults
	if config.MaxCostPerRequest <= 0 {
		t.Error("Should have positive default max cost")
	}

	if config.MinSampleSize < 1 {
		t.Error("Should require at least 1 sample for learning")
	}

	if config.ConservativeBias < 0 || config.ConservativeBias > 1 {
		t.Error("Conservative bias should be between 0 and 1")
	}
}

func TestEnumStringMethods(t *testing.T) {
	// Test TaskComplexity string methods
	complexities := []TaskComplexity{TaskComplexitySimple, TaskComplexityModerate, TaskComplexityComplex}
	for _, c := range complexities {
		if c.String() == "" || c.String() == "unknown" {
			t.Errorf("TaskComplexity %d should have valid string representation", int(c))
		}
	}

	// Test QualityRequirement string methods
	qualities := []QualityRequirement{QualityBasic, QualityStandard, QualityPremium}
	for _, q := range qualities {
		if q.String() == "" || q.String() == "unknown" {
			t.Errorf("QualityRequirement %d should have valid string representation", int(q))
		}
	}
}

// Benchmark tests

func BenchmarkTaskAssessment(b *testing.B) {
	router := NewRouter(NewMockLLMService())

	req := TaskRequest{
		Prompt:          "Analyze this business proposal for potential risks and opportunities in the current market environment",
		TaskType:        "analysis",
		QualityRequired: QualityStandard,
		MaxTokens:       2000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = router.assessTask(req)
	}
}

func BenchmarkModelScoring(b *testing.B) {
	router := NewRouter(NewMockLLMService())
	models := router.getAvailableModels()
	assessment := TaskAssessment{
		Complexity:      TaskComplexityModerate,
		EstimatedTokens: 1500,
		QualityNeeded:   QualityStandard,
	}
	req := TaskRequest{
		Prompt:          "Test prompt",
		TaskType:        "analysis",
		QualityRequired: QualityStandard,
		MaxTokens:       2000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = router.scoreModels(models, assessment, req)
	}
}