package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// LLMService provides language model access as an MCP service.
// It supports multiple providers, budget tracking, and error handling with retries.
type LLMService struct {
	*BaseService
	providers    map[string]LLMProvider
	budgetTracker *BudgetTracker
	httpClient   *http.Client
	retryConfig  RetryConfig
}

// LLMProvider defines the interface for different LLM providers.
type LLMProvider interface {
	Name() string
	Complete(ctx context.Context, request CompletionRequest) (*CompletionResponse, error)
	Embed(ctx context.Context, request EmbeddingRequest) (*EmbeddingResponse, error)
	CalculateCost(tokens int, operation string) float64
}

// CompletionRequest represents a text completion request.
type CompletionRequest struct {
	Model       string            `json:"model"`
	Prompt      string            `json:"prompt"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	StopWords   []string          `json:"stop_words,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// CompletionResponse represents a text completion response.
type CompletionResponse struct {
	Text         string                 `json:"text"`
	TokensUsed   int                    `json:"tokens_used"`
	Model        string                 `json:"model"`
	Provider     string                 `json:"provider"`
	Cost         float64                `json:"cost"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// EmbeddingRequest represents an embedding request.
type EmbeddingRequest struct {
	Model    string            `json:"model"`
	Text     string            `json:"text"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// EmbeddingResponse represents an embedding response.
type EmbeddingResponse struct {
	Embedding    []float64              `json:"embedding"`
	TokensUsed   int                    `json:"tokens_used"`
	Model        string                 `json:"model"`
	Provider     string                 `json:"provider"`
	Cost         float64                `json:"cost"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// BudgetTracker tracks token usage and costs across providers.
type BudgetTracker struct {
	TotalTokens int                        `json:"total_tokens"`
	TotalCost   float64                    `json:"total_cost"`
	ByProvider  map[string]ProviderUsage   `json:"by_provider"`
	ByOperation map[string]OperationUsage  `json:"by_operation"`
	DailyLimit  float64                    `json:"daily_limit"`
	StartTime   time.Time                  `json:"start_time"`
}

// ProviderUsage tracks usage for a specific provider.
type ProviderUsage struct {
	Tokens int     `json:"tokens"`
	Cost   float64 `json:"cost"`
	Calls  int     `json:"calls"`
}

// OperationUsage tracks usage for a specific operation.
type OperationUsage struct {
	Tokens int     `json:"tokens"`
	Cost   float64 `json:"cost"`
	Calls  int     `json:"calls"`
}

// RetryConfig defines retry behavior for failed requests.
type RetryConfig struct {
	MaxRetries  int           `json:"max_retries"`
	BaseDelay   time.Duration `json:"base_delay"`
	MaxDelay    time.Duration `json:"max_delay"`
	BackoffRate float64       `json:"backoff_rate"`
}

// AnthropicProvider implements the Anthropic Claude API.
type AnthropicProvider struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
	Models     map[string]ModelConfig
}

// OpenAIProvider implements the OpenAI API.
type OpenAIProvider struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
	Models     map[string]ModelConfig
}

// LocalProvider implements local HuggingFace model access.
type LocalProvider struct {
	ServerURL  string
	HTTPClient *http.Client
	Models     map[string]ModelConfig
}

// ModelConfig contains configuration for a specific model.
type ModelConfig struct {
	Name         string  `json:"name"`
	InputCost    float64 `json:"input_cost"`    // Cost per 1K tokens
	OutputCost   float64 `json:"output_cost"`   // Cost per 1K tokens
	MaxTokens    int     `json:"max_tokens"`
	ContextSize  int     `json:"context_size"`
	SupportsChat bool    `json:"supports_chat"`
	SupportsEmbed bool   `json:"supports_embed"`
}

// NewLLMService creates a new LLM MCP service.
func NewLLMService(logger *log.Logger) *LLMService {
	base := NewBaseService(
		"llm",
		"Language model access with multiple providers, budget tracking, and error handling",
		logger,
	)

	service := &LLMService{
		BaseService: base,
		providers:   make(map[string]LLMProvider),
		budgetTracker: &BudgetTracker{
			ByProvider:  make(map[string]ProviderUsage),
			ByOperation: make(map[string]OperationUsage),
			DailyLimit:  100.0, // $100 daily limit by default
			StartTime:   time.Now(),
		},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		retryConfig: RetryConfig{
			MaxRetries:  3,
			BaseDelay:   1 * time.Second,
			MaxDelay:    10 * time.Second,
			BackoffRate: 2.0,
		},
	}

	// Initialize providers based on available credentials
	service.initializeProviders()

	return service
}

// initializeProviders sets up available LLM providers based on environment variables.
func (llm *LLMService) initializeProviders() {
	// Anthropic Claude API
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		anthropic := &AnthropicProvider{
			APIKey:     apiKey,
			BaseURL:    "https://api.anthropic.com",
			HTTPClient: llm.httpClient,
			Models: map[string]ModelConfig{
				"claude-3-sonnet": {
					Name:         "claude-3-sonnet-20240229",
					InputCost:    3.0,   // $3 per 1M tokens
					OutputCost:   15.0,  // $15 per 1M tokens
					MaxTokens:    4096,
					ContextSize:  200000,
					SupportsChat: true,
					SupportsEmbed: false,
				},
				"claude-3-haiku": {
					Name:         "claude-3-haiku-20240307",
					InputCost:    0.25,  // $0.25 per 1M tokens
					OutputCost:   1.25,  // $1.25 per 1M tokens
					MaxTokens:    4096,
					ContextSize:  200000,
					SupportsChat: true,
					SupportsEmbed: false,
				},
			},
		}
		llm.providers["anthropic"] = anthropic
	}

	// OpenAI API
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		openai := &OpenAIProvider{
			APIKey:     apiKey,
			BaseURL:    "https://api.openai.com",
			HTTPClient: llm.httpClient,
			Models: map[string]ModelConfig{
				"gpt-4": {
					Name:         "gpt-4",
					InputCost:    30.0,  // $30 per 1M tokens
					OutputCost:   60.0,  // $60 per 1M tokens
					MaxTokens:    4096,
					ContextSize:  8192,
					SupportsChat: true,
					SupportsEmbed: false,
				},
				"gpt-3.5-turbo": {
					Name:         "gpt-3.5-turbo",
					InputCost:    0.5,   // $0.5 per 1M tokens
					OutputCost:   1.5,   // $1.5 per 1M tokens
					MaxTokens:    4096,
					ContextSize:  16385,
					SupportsChat: true,
					SupportsEmbed: false,
				},
				"text-embedding-ada-002": {
					Name:         "text-embedding-ada-002",
					InputCost:    0.1,   // $0.1 per 1M tokens
					OutputCost:   0.0,   // No output cost for embeddings
					MaxTokens:    0,     // Not applicable
					ContextSize:  8191,
					SupportsChat: false,
					SupportsEmbed: true,
				},
			},
		}
		llm.providers["openai"] = openai
	}

	// Local HuggingFace models
	if serverURL := os.Getenv("LOCAL_LLM_URL"); serverURL != "" {
		local := &LocalProvider{
			ServerURL:  serverURL,
			HTTPClient: llm.httpClient,
			Models: map[string]ModelConfig{
				"local-llama": {
					Name:         "llama-2-7b-chat",
					InputCost:    0.0,   // Free for local models
					OutputCost:   0.0,
					MaxTokens:    4096,
					ContextSize:  4096,
					SupportsChat: true,
					SupportsEmbed: false,
				},
			},
		}
		llm.providers["local"] = local
	}
}

// ValidateParams validates parameters for LLM operations.
func (llm *LLMService) ValidateParams(params ServiceParams) error {
	if err := llm.BaseService.ValidateParams(params); err != nil {
		return err
	}

	operation, exists := params["operation"]
	if !exists {
		return NewValidationError("operation", "operation parameter is required")
	}

	operationStr, ok := operation.(string)
	if !ok {
		return NewValidationError("operation", "operation must be a string")
	}

	// Validate operation-specific parameters
	switch operationStr {
	case "complete":
		return llm.validateCompleteParams(params)
	case "embed":
		return llm.validateEmbedParams(params)
	case "list_providers":
		return nil // No additional parameters needed
	case "get_budget":
		return nil // No additional parameters needed
	case "reset_budget":
		return nil // No additional parameters needed
	default:
		return NewValidationError("operation", fmt.Sprintf("unsupported operation: %s", operationStr))
	}
}

// validateCompleteParams validates parameters for complete operation.
func (llm *LLMService) validateCompleteParams(params ServiceParams) error {
	if err := ValidateStringParam(params, "prompt", true); err != nil {
		return err
	}

	if err := ValidateStringParam(params, "provider", false); err != nil {
		return err
	}

	if err := ValidateStringParam(params, "model", false); err != nil {
		return err
	}

	// Validate provider exists if specified
	if providerName, exists := params["provider"]; exists {
		providerStr := providerName.(string)
		if _, exists := llm.providers[providerStr]; !exists {
			return NewValidationError("provider", "specified provider '"+providerStr+"' is not available")
		}
	}

	// Optional numeric parameters
	minTokens := 1
	maxTokens := 8192
	if err := ValidateIntParam(params, "max_tokens", false, &minTokens, &maxTokens); err != nil {
		return err
	}

	// Temperature validation (0.0 to 2.0)
	if temp, exists := params["temperature"]; exists {
		if tempFloat, ok := temp.(float64); ok {
			if tempFloat < 0.0 || tempFloat > 2.0 {
				return NewValidationError("temperature", "temperature must be between 0.0 and 2.0")
			}
		} else {
			return NewValidationError("temperature", "temperature must be a number")
		}
	}

	return nil
}

// validateEmbedParams validates parameters for embed operation.
func (llm *LLMService) validateEmbedParams(params ServiceParams) error {
	if err := ValidateStringParam(params, "text", true); err != nil {
		return err
	}

	if err := ValidateStringParam(params, "provider", false); err != nil {
		return err
	}

	if err := ValidateStringParam(params, "model", false); err != nil {
		return err
	}

	// Validate provider exists if specified
	if providerName, exists := params["provider"]; exists {
		providerStr := providerName.(string)
		if _, exists := llm.providers[providerStr]; !exists {
			return NewValidationError("provider", "specified provider '"+providerStr+"' is not available")
		}
	}

	return nil
}

// Execute performs the requested LLM operation.
func (llm *LLMService) Execute(ctx context.Context, params ServiceParams) ServiceResult {
	operation := params["operation"].(string)

	switch operation {
	case "complete":
		return llm.complete(ctx, params)
	case "embed":
		return llm.embed(ctx, params)
	case "list_providers":
		return llm.listProviders(ctx, params)
	case "get_budget":
		return llm.getBudget(ctx, params)
	case "reset_budget":
		return llm.resetBudget(ctx, params)
	default:
		return ErrorResult(fmt.Errorf("unsupported operation: %s", operation))
	}
}

// complete performs text completion with automatic provider selection.
func (llm *LLMService) complete(ctx context.Context, params ServiceParams) ServiceResult {
	prompt := params["prompt"].(string)

	// Select provider and model
	providerName, modelName, err := llm.selectProvider(params, "complete")
	if err != nil {
		return ErrorResult(fmt.Errorf("provider selection failed: %w", err))
	}

	provider, exists := llm.providers[providerName]
	if !exists {
		return ErrorResult(fmt.Errorf("provider '%s' not available", providerName))
	}

	// Build completion request
	request := CompletionRequest{
		Model:  modelName,
		Prompt: prompt,
	}

	// Set optional parameters
	if maxTokens, exists := params["max_tokens"]; exists {
		request.MaxTokens = maxTokens.(int)
	}

	if temperature, exists := params["temperature"]; exists {
		request.Temperature = temperature.(float64)
	}

	if stopWords, exists := params["stop_words"]; exists {
		if words, ok := stopWords.([]interface{}); ok {
			request.StopWords = make([]string, len(words))
			for i, word := range words {
				request.StopWords[i] = word.(string)
			}
		}
	}

	// Check budget before making request
	if err := llm.checkBudget(); err != nil {
		return ErrorResult(fmt.Errorf("budget check failed: %w", err))
	}

	// Execute with retries
	response, err := llm.executeWithRetry(ctx, func() (interface{}, error) {
		return provider.Complete(ctx, request)
	})

	if err != nil {
		return ErrorResult(fmt.Errorf("completion failed: %w", err))
	}

	completionResp := response.(*CompletionResponse)

	// Update budget tracking
	llm.updateBudget(providerName, "complete", completionResp.TokensUsed, completionResp.Cost)

	return SuccessResult(completionResp)
}

// embed performs text embedding.
func (llm *LLMService) embed(ctx context.Context, params ServiceParams) ServiceResult {
	text := params["text"].(string)

	// Select provider and model for embeddings
	providerName, modelName, err := llm.selectProvider(params, "embed")
	if err != nil {
		return ErrorResult(fmt.Errorf("provider selection failed: %w", err))
	}

	provider, exists := llm.providers[providerName]
	if !exists {
		return ErrorResult(fmt.Errorf("provider '%s' not available", providerName))
	}

	// Build embedding request
	request := EmbeddingRequest{
		Model: modelName,
		Text:  text,
	}

	// Check budget before making request
	if err := llm.checkBudget(); err != nil {
		return ErrorResult(fmt.Errorf("budget check failed: %w", err))
	}

	// Execute with retries
	response, err := llm.executeWithRetry(ctx, func() (interface{}, error) {
		return provider.Embed(ctx, request)
	})

	if err != nil {
		return ErrorResult(fmt.Errorf("embedding failed: %w", err))
	}

	embeddingResp := response.(*EmbeddingResponse)

	// Update budget tracking
	llm.updateBudget(providerName, "embed", embeddingResp.TokensUsed, embeddingResp.Cost)

	return SuccessResult(embeddingResp)
}

// listProviders returns information about available providers.
func (llm *LLMService) listProviders(ctx context.Context, params ServiceParams) ServiceResult {
	result := map[string]interface{}{
		"providers": make([]map[string]interface{}, 0, len(llm.providers)),
	}

	for name, provider := range llm.providers {
		providerInfo := map[string]interface{}{
			"name": name,
			"provider_name": provider.Name(),
		}
		result["providers"] = append(result["providers"].([]map[string]interface{}), providerInfo)
	}

	return SuccessResult(result)
}

// getBudget returns current budget tracking information.
func (llm *LLMService) getBudget(ctx context.Context, params ServiceParams) ServiceResult {
	return SuccessResult(llm.budgetTracker)
}

// resetBudget resets the budget tracking counters.
func (llm *LLMService) resetBudget(ctx context.Context, params ServiceParams) ServiceResult {
	llm.budgetTracker = &BudgetTracker{
		ByProvider:  make(map[string]ProviderUsage),
		ByOperation: make(map[string]OperationUsage),
		DailyLimit:  llm.budgetTracker.DailyLimit,
		StartTime:   time.Now(),
	}

	result := map[string]interface{}{
		"message": "Budget tracking reset successfully",
		"reset_time": time.Now().Format(time.RFC3339),
	}

	return SuccessResult(result)
}

// selectProvider chooses the best provider and model for the operation.
func (llm *LLMService) selectProvider(params ServiceParams, operation string) (string, string, error) {
	// If provider explicitly specified, use it
	if providerName, exists := params["provider"]; exists {
		providerStr := providerName.(string)
		if _, exists := llm.providers[providerStr]; !exists {
			return "", "", fmt.Errorf("specified provider '%s' not available", providerStr)
		}

		// Get model for this provider
		modelName := llm.getModelForProvider(providerStr, operation, params)
		return providerStr, modelName, nil
	}

	// Auto-select based on operation and cost
	switch operation {
	case "complete":
		// Prefer local, then anthropic (haiku), then openai
		if _, exists := llm.providers["local"]; exists {
			return "local", llm.getModelForProvider("local", operation, params), nil
		}
		if _, exists := llm.providers["anthropic"]; exists {
			return "anthropic", "claude-3-haiku", nil
		}
		if _, exists := llm.providers["openai"]; exists {
			return "openai", "gpt-3.5-turbo", nil
		}
	case "embed":
		// For embeddings, prefer OpenAI as it has dedicated embedding models
		if _, exists := llm.providers["openai"]; exists {
			return "openai", "text-embedding-ada-002", nil
		}
	}

	return "", "", fmt.Errorf("no suitable provider available for operation '%s'", operation)
}

// getModelForProvider returns the appropriate model for a provider and operation.
func (llm *LLMService) getModelForProvider(providerName, operation string, params ServiceParams) string {
	// If model explicitly specified, use it
	if modelName, exists := params["model"]; exists {
		return modelName.(string)
	}

	// Return default models based on provider and operation
	switch providerName {
	case "anthropic":
		if operation == "complete" {
			return "claude-3-haiku" // Default to cheaper model
		}
	case "openai":
		if operation == "complete" {
			return "gpt-3.5-turbo"
		}
		if operation == "embed" {
			return "text-embedding-ada-002"
		}
	case "local":
		return "local-llama"
	}

	return ""
}

// checkBudget verifies that the daily budget limit hasn't been exceeded.
func (llm *LLMService) checkBudget() error {
	if llm.budgetTracker.TotalCost >= llm.budgetTracker.DailyLimit {
		return fmt.Errorf("daily budget limit of $%.2f exceeded (current: $%.2f)",
			llm.budgetTracker.DailyLimit, llm.budgetTracker.TotalCost)
	}
	return nil
}

// updateBudget updates budget tracking with usage information.
func (llm *LLMService) updateBudget(provider, operation string, tokens int, cost float64) {
	// Update totals
	llm.budgetTracker.TotalTokens += tokens
	llm.budgetTracker.TotalCost += cost

	// Update provider usage
	providerUsage := llm.budgetTracker.ByProvider[provider]
	providerUsage.Tokens += tokens
	providerUsage.Cost += cost
	providerUsage.Calls++
	llm.budgetTracker.ByProvider[provider] = providerUsage

	// Update operation usage
	operationUsage := llm.budgetTracker.ByOperation[operation]
	operationUsage.Tokens += tokens
	operationUsage.Cost += cost
	operationUsage.Calls++
	llm.budgetTracker.ByOperation[operation] = operationUsage
}

// executeWithRetry executes a function with exponential backoff retry logic.
func (llm *LLMService) executeWithRetry(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	var lastErr error
	delay := llm.retryConfig.BaseDelay

	for attempt := 0; attempt <= llm.retryConfig.MaxRetries; attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if this is a retryable error
		if !llm.isRetryableError(err) {
			break
		}

		// Don't retry on the last attempt
		if attempt == llm.retryConfig.MaxRetries {
			break
		}

		// Wait before retrying
		select {
		case <-time.After(delay):
			// Continue to retry
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled during retry: %w", ctx.Err())
		}

		// Exponential backoff
		delay = time.Duration(float64(delay) * llm.retryConfig.BackoffRate)
		if delay > llm.retryConfig.MaxDelay {
			delay = llm.retryConfig.MaxDelay
		}
	}

	return nil, fmt.Errorf("operation failed after %d retries: %w", llm.retryConfig.MaxRetries, lastErr)
}

// isRetryableError determines if an error should trigger a retry.
func (llm *LLMService) isRetryableError(err error) bool {
	errStr := strings.ToLower(err.Error())

	// Rate limiting errors
	if strings.Contains(errStr, "rate limit") ||
	   strings.Contains(errStr, "too many requests") ||
	   strings.Contains(errStr, "429") {
		return true
	}

	// Temporary network errors
	if strings.Contains(errStr, "timeout") ||
	   strings.Contains(errStr, "connection") ||
	   strings.Contains(errStr, "500") ||
	   strings.Contains(errStr, "502") ||
	   strings.Contains(errStr, "503") ||
	   strings.Contains(errStr, "504") {
		return true
	}

	return false
}

// Provider implementations

// Name returns the provider name for AnthropicProvider.
func (ap *AnthropicProvider) Name() string {
	return "Anthropic Claude API"
}

// Complete performs text completion using the Anthropic Claude API.
func (ap *AnthropicProvider) Complete(ctx context.Context, request CompletionRequest) (*CompletionResponse, error) {
	// Build Anthropic API request
	anthropicRequest := map[string]interface{}{
		"model":      request.Model,
		"max_tokens": request.MaxTokens,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": request.Prompt,
			},
		},
	}

	if request.Temperature > 0 {
		anthropicRequest["temperature"] = request.Temperature
	}

	if len(request.StopWords) > 0 {
		anthropicRequest["stop_sequences"] = request.StopWords
	}

	// Marshal request
	requestBody, err := json.Marshal(anthropicRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", ap.BaseURL+"/v1/messages", strings.NewReader(string(requestBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ap.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Execute request
	resp, err := ap.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var anthropicResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Handle API errors
	if resp.StatusCode >= 400 {
		errMsg := "unknown error"
		if errData, exists := anthropicResp["error"]; exists {
			if errMap, ok := errData.(map[string]interface{}); ok {
				if msg, ok := errMap["message"].(string); ok {
					errMsg = msg
				}
			}
		}
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, errMsg)
	}

	// Extract content and usage
	var text string
	var tokensUsed int

	if content, exists := anthropicResp["content"]; exists {
		if contentArray, ok := content.([]interface{}); ok && len(contentArray) > 0 {
			if firstContent, ok := contentArray[0].(map[string]interface{}); ok {
				if textContent, ok := firstContent["text"].(string); ok {
					text = textContent
				}
			}
		}
	}

	if usage, exists := anthropicResp["usage"]; exists {
		if usageMap, ok := usage.(map[string]interface{}); ok {
			if inputTokens, ok := usageMap["input_tokens"].(float64); ok {
				tokensUsed += int(inputTokens)
			}
			if outputTokens, ok := usageMap["output_tokens"].(float64); ok {
				tokensUsed += int(outputTokens)
			}
		}
	}

	// Calculate cost
	cost := ap.CalculateCost(tokensUsed, "complete")

	return &CompletionResponse{
		Text:       text,
		TokensUsed: tokensUsed,
		Model:      request.Model,
		Provider:   "anthropic",
		Cost:       cost,
		Metadata: map[string]interface{}{
			"api_version": "2023-06-01",
		},
	}, nil
}

// Embed returns an error as Anthropic doesn't provide embedding models.
func (ap *AnthropicProvider) Embed(ctx context.Context, request EmbeddingRequest) (*EmbeddingResponse, error) {
	return nil, fmt.Errorf("Anthropic provider does not support embeddings")
}

// CalculateCost calculates the cost for Anthropic API usage.
func (ap *AnthropicProvider) CalculateCost(tokens int, operation string) float64 {
	// Cost is typically split between input and output tokens
	// For simplicity, we'll use average cost (this could be refined with actual input/output split)
	modelConfig, exists := ap.Models["claude-3-haiku"] // Default model for cost calculation
	if !exists {
		return 0.0
	}

	avgCost := (modelConfig.InputCost + modelConfig.OutputCost) / 2.0
	return float64(tokens) * avgCost / 1000000.0 // Convert to cost per token
}

// Name returns the provider name for OpenAIProvider.
func (op *OpenAIProvider) Name() string {
	return "OpenAI API"
}

// Complete performs text completion using the OpenAI API.
func (op *OpenAIProvider) Complete(ctx context.Context, request CompletionRequest) (*CompletionResponse, error) {
	// Build OpenAI API request
	openaiRequest := map[string]interface{}{
		"model": request.Model,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": request.Prompt,
			},
		},
	}

	if request.MaxTokens > 0 {
		openaiRequest["max_tokens"] = request.MaxTokens
	}

	if request.Temperature > 0 {
		openaiRequest["temperature"] = request.Temperature
	}

	if len(request.StopWords) > 0 {
		openaiRequest["stop"] = request.StopWords
	}

	// Marshal request
	requestBody, err := json.Marshal(openaiRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", op.BaseURL+"/v1/chat/completions", strings.NewReader(string(requestBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+op.APIKey)

	// Execute request
	resp, err := op.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var openaiResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Handle API errors
	if resp.StatusCode >= 400 {
		errMsg := "unknown error"
		if errData, exists := openaiResp["error"]; exists {
			if errMap, ok := errData.(map[string]interface{}); ok {
				if msg, ok := errMap["message"].(string); ok {
					errMsg = msg
				}
			}
		}
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, errMsg)
	}

	// Extract content and usage
	var text string
	var tokensUsed int

	if choices, exists := openaiResp["choices"]; exists {
		if choicesArray, ok := choices.([]interface{}); ok && len(choicesArray) > 0 {
			if firstChoice, ok := choicesArray[0].(map[string]interface{}); ok {
				if message, ok := firstChoice["message"].(map[string]interface{}); ok {
					if content, ok := message["content"].(string); ok {
						text = content
					}
				}
			}
		}
	}

	if usage, exists := openaiResp["usage"]; exists {
		if usageMap, ok := usage.(map[string]interface{}); ok {
			if totalTokens, ok := usageMap["total_tokens"].(float64); ok {
				tokensUsed = int(totalTokens)
			}
		}
	}

	// Calculate cost
	cost := op.CalculateCost(tokensUsed, "complete")

	return &CompletionResponse{
		Text:       text,
		TokensUsed: tokensUsed,
		Model:      request.Model,
		Provider:   "openai",
		Cost:       cost,
		Metadata: map[string]interface{}{
			"api_version": "v1",
		},
	}, nil
}

// Embed performs text embedding using the OpenAI API.
func (op *OpenAIProvider) Embed(ctx context.Context, request EmbeddingRequest) (*EmbeddingResponse, error) {
	// Build OpenAI embedding request
	openaiRequest := map[string]interface{}{
		"model": request.Model,
		"input": request.Text,
	}

	// Marshal request
	requestBody, err := json.Marshal(openaiRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", op.BaseURL+"/v1/embeddings", strings.NewReader(string(requestBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+op.APIKey)

	// Execute request
	resp, err := op.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var openaiResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Handle API errors
	if resp.StatusCode >= 400 {
		errMsg := "unknown error"
		if errData, exists := openaiResp["error"]; exists {
			if errMap, ok := errData.(map[string]interface{}); ok {
				if msg, ok := errMap["message"].(string); ok {
					errMsg = msg
				}
			}
		}
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, errMsg)
	}

	// Extract embedding and usage
	var embedding []float64
	var tokensUsed int

	if data, exists := openaiResp["data"]; exists {
		if dataArray, ok := data.([]interface{}); ok && len(dataArray) > 0 {
			if firstData, ok := dataArray[0].(map[string]interface{}); ok {
				if embeddingData, ok := firstData["embedding"].([]interface{}); ok {
					embedding = make([]float64, len(embeddingData))
					for i, val := range embeddingData {
						if floatVal, ok := val.(float64); ok {
							embedding[i] = floatVal
						}
					}
				}
			}
		}
	}

	if usage, exists := openaiResp["usage"]; exists {
		if usageMap, ok := usage.(map[string]interface{}); ok {
			if totalTokens, ok := usageMap["total_tokens"].(float64); ok {
				tokensUsed = int(totalTokens)
			}
		}
	}

	// Calculate cost
	cost := op.CalculateCost(tokensUsed, "embed")

	return &EmbeddingResponse{
		Embedding:  embedding,
		TokensUsed: tokensUsed,
		Model:      request.Model,
		Provider:   "openai",
		Cost:       cost,
		Metadata: map[string]interface{}{
			"api_version": "v1",
		},
	}, nil
}

// CalculateCost calculates the cost for OpenAI API usage.
func (op *OpenAIProvider) CalculateCost(tokens int, operation string) float64 {
	var cost float64

	// Find appropriate model cost
	for _, modelConfig := range op.Models {
		if operation == "complete" && modelConfig.SupportsChat {
			avgCost := (modelConfig.InputCost + modelConfig.OutputCost) / 2.0
			cost = float64(tokens) * avgCost / 1000000.0
			break
		} else if operation == "embed" && modelConfig.SupportsEmbed {
			cost = float64(tokens) * modelConfig.InputCost / 1000000.0
			break
		}
	}

	return cost
}

// Name returns the provider name for LocalProvider.
func (lp *LocalProvider) Name() string {
	return "Local HuggingFace Models"
}

// Complete performs text completion using local models.
func (lp *LocalProvider) Complete(ctx context.Context, request CompletionRequest) (*CompletionResponse, error) {
	// Build local API request (compatible with text-generation-webui format)
	localRequest := map[string]interface{}{
		"prompt":      request.Prompt,
		"max_tokens":  request.MaxTokens,
		"temperature": request.Temperature,
	}

	if len(request.StopWords) > 0 {
		localRequest["stop"] = request.StopWords
	}

	// Marshal request
	requestBody, err := json.Marshal(localRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request to local server
	req, err := http.NewRequestWithContext(ctx, "POST", lp.ServerURL+"/api/v1/generate", strings.NewReader(string(requestBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := lp.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("local API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var localResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&localResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Handle API errors
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("local API error (status %d): %v", resp.StatusCode, localResp)
	}

	// Extract generated text
	var text string
	if results, exists := localResp["results"]; exists {
		if resultsArray, ok := results.([]interface{}); ok && len(resultsArray) > 0 {
			if firstResult, ok := resultsArray[0].(map[string]interface{}); ok {
				if generatedText, ok := firstResult["text"].(string); ok {
					text = generatedText
				}
			}
		}
	}

	// Estimate tokens (rough approximation: 1 token ≈ 4 characters)
	tokensUsed := len(request.Prompt+text) / 4

	return &CompletionResponse{
		Text:       text,
		TokensUsed: tokensUsed,
		Model:      request.Model,
		Provider:   "local",
		Cost:       0.0, // Local models are free
		Metadata: map[string]interface{}{
			"server_url": lp.ServerURL,
		},
	}, nil
}

// Embed returns an error as local embeddings would need a separate implementation.
func (lp *LocalProvider) Embed(ctx context.Context, request EmbeddingRequest) (*EmbeddingResponse, error) {
	return nil, fmt.Errorf("local provider does not currently support embeddings")
}

// CalculateCost returns 0.0 for local providers since they're free to use.
func (lp *LocalProvider) CalculateCost(tokens int, operation string) float64 {
	return 0.0 // Local models are free
}

// Testing helper methods

// SetProvider manually sets a provider for testing purposes.
func (llm *LLMService) SetProvider(name string, provider LLMProvider) {
	llm.providers[name] = provider
}

// SetRetryConfig sets the retry configuration for testing.
func (llm *LLMService) SetRetryConfig(config RetryConfig) {
	llm.retryConfig = config
}

// SetBudgetLimit sets the daily budget limit for testing.
func (llm *LLMService) SetBudgetLimit(limit float64) {
	llm.budgetTracker.DailyLimit = limit
}

// UpdateBudgetForTest manually updates budget for testing purposes.
func (llm *LLMService) UpdateBudgetForTest(provider, operation string, tokens int, cost float64) {
	llm.updateBudget(provider, operation, tokens, cost)
}

// GetProviderCount returns the number of registered providers.
func (llm *LLMService) GetProviderCount() int {
	return len(llm.providers)
}