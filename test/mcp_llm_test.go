package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Solifugus/ai-work-studio/pkg/mcp"
)

// TestLLMService tests the LLM MCP service implementation.
func TestLLMService(t *testing.T) {
	// Create a test LLM service
	service := mcp.NewLLMService(nil)

	// Test service implementation
	if service.Name() != "llm" {
		t.Errorf("Expected service name 'llm', got %s", service.Name())
	}

	expectedDesc := "Language model access with multiple providers, budget tracking, and error handling"
	if service.Description() != expectedDesc {
		t.Errorf("Expected service description '%s', got %s", expectedDesc, service.Description())
	}
}

// TestLLMValidateParams tests parameter validation.
func TestLLMValidateParams(t *testing.T) {
	service := mcp.NewLLMService(nil)

	tests := []struct {
		name     string
		params   mcp.ServiceParams
		hasError bool
	}{
		{
			name:     "nil parameters",
			params:   nil,
			hasError: true,
		},
		{
			name:     "missing operation",
			params:   mcp.ServiceParams{},
			hasError: true,
		},
		{
			name: "invalid operation",
			params: mcp.ServiceParams{
				"operation": "invalid_op",
			},
			hasError: true,
		},
		{
			name: "complete operation - valid",
			params: mcp.ServiceParams{
				"operation": "complete",
				"prompt":    "Hello, world!",
			},
			hasError: false,
		},
		{
			name: "complete operation - missing prompt",
			params: mcp.ServiceParams{
				"operation": "complete",
			},
			hasError: true,
		},
		{
			name: "complete operation - invalid temperature",
			params: mcp.ServiceParams{
				"operation":   "complete",
				"prompt":      "Hello, world!",
				"temperature": 3.0, // Too high
			},
			hasError: true,
		},
		{
			name: "embed operation - valid",
			params: mcp.ServiceParams{
				"operation": "embed",
				"text":      "Hello, world!",
			},
			hasError: false,
		},
		{
			name: "embed operation - missing text",
			params: mcp.ServiceParams{
				"operation": "embed",
			},
			hasError: true,
		},
		{
			name: "list_providers - valid",
			params: mcp.ServiceParams{
				"operation": "list_providers",
			},
			hasError: false,
		},
		{
			name: "get_budget - valid",
			params: mcp.ServiceParams{
				"operation": "get_budget",
			},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateParams(tt.params)
			if tt.hasError && err == nil {
				t.Errorf("Expected validation error, got nil")
			}
			if !tt.hasError && err != nil {
				t.Errorf("Expected no validation error, got %v", err)
			}
		})
	}
}

// TestLLMListProviders tests listing available providers.
func TestLLMListProviders(t *testing.T) {
	// Set up environment variables to initialize providers
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer func() {
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("OPENAI_API_KEY")
	}()

	service := mcp.NewLLMService(nil)

	params := mcp.ServiceParams{
		"operation": "list_providers",
	}

	result := service.Execute(context.Background(), params)
	if !result.Success {
		t.Fatalf("Expected success, got error: %v", result.Error)
	}

	// Check that providers are listed
	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result.Data)
	}

	providers, exists := data["providers"]
	if !exists {
		t.Fatalf("Expected providers field in result")
	}

	providerList, ok := providers.([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected providers to be a list, got %T", providers)
	}

	if len(providerList) < 2 {
		t.Errorf("Expected at least 2 providers (Anthropic and OpenAI), got %d", len(providerList))
	}
}

// TestLLMBudgetTracking tests budget tracking functionality.
func TestLLMBudgetTracking(t *testing.T) {
	service := mcp.NewLLMService(nil)

	// Test getting initial budget
	params := mcp.ServiceParams{
		"operation": "get_budget",
	}

	result := service.Execute(context.Background(), params)
	if !result.Success {
		t.Fatalf("Expected success, got error: %v", result.Error)
	}

	// Test resetting budget
	params = mcp.ServiceParams{
		"operation": "reset_budget",
	}

	result = service.Execute(context.Background(), params)
	if !result.Success {
		t.Fatalf("Expected success, got error: %v", result.Error)
	}

	// Verify budget was reset
	resetData, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result.Data)
	}

	if _, exists := resetData["message"]; !exists {
		t.Errorf("Expected reset message in result")
	}

	if _, exists := resetData["reset_time"]; !exists {
		t.Errorf("Expected reset_time in result")
	}
}

// mockAnthropicServer creates a mock Anthropic API server for testing.
func mockAnthropicServer(t *testing.T, response map[string]interface{}, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Invalid authorization header")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
}

// mockOpenAIServer creates a mock OpenAI API server for testing.
func mockOpenAIServer(t *testing.T, endpoint string, response map[string]interface{}, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1/" + endpoint
		if r.URL.Path != expectedPath {
			t.Errorf("Unexpected path: %s, expected: %s", r.URL.Path, expectedPath)
		}

		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Invalid authorization header")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
}

// TestLLMAnthropicProvider tests the Anthropic provider implementation.
func TestLLMAnthropicProvider(t *testing.T) {
	// Create mock server
	response := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": "Hello! How can I help you today?",
			},
		},
		"usage": map[string]interface{}{
			"input_tokens":  10.0,
			"output_tokens": 15.0,
		},
	}

	server := mockAnthropicServer(t, response, 200)
	defer server.Close()

	// Create provider
	provider := &mcp.AnthropicProvider{
		APIKey:  "test-key",
		BaseURL: server.URL,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		Models: map[string]mcp.ModelConfig{
			"claude-3-haiku": {
				Name:       "claude-3-haiku-20240307",
				InputCost:  0.25,
				OutputCost: 1.25,
			},
		},
	}

	// Test completion
	request := mcp.CompletionRequest{
		Model:       "claude-3-haiku",
		Prompt:      "Hello!",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	ctx := context.Background()
	result, err := provider.Complete(ctx, request)
	if err != nil {
		t.Fatalf("Completion failed: %v", err)
	}

	if result.Text != "Hello! How can I help you today?" {
		t.Errorf("Unexpected response text: %s", result.Text)
	}

	if result.TokensUsed != 25 {
		t.Errorf("Expected 25 tokens used, got %d", result.TokensUsed)
	}

	if result.Provider != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got %s", result.Provider)
	}

	// Test embedding (should fail)
	embeddingRequest := mcp.EmbeddingRequest{
		Model: "claude-3-haiku",
		Text:  "Hello!",
	}

	_, err = provider.Embed(ctx, embeddingRequest)
	if err == nil {
		t.Errorf("Expected embedding to fail for Anthropic provider")
	}
}

// TestLLMOpenAIProvider tests the OpenAI provider implementation.
func TestLLMOpenAIProvider(t *testing.T) {
	// Test completion
	t.Run("completion", func(t *testing.T) {
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "Hello! How can I assist you?",
					},
				},
			},
			"usage": map[string]interface{}{
				"total_tokens": 20.0,
			},
		}

		server := mockOpenAIServer(t, "chat/completions", response, 200)
		defer server.Close()

		provider := &mcp.OpenAIProvider{
			APIKey:  "test-key",
			BaseURL: server.URL,
			HTTPClient: &http.Client{
				Timeout: 5 * time.Second,
			},
			Models: map[string]mcp.ModelConfig{
				"gpt-3.5-turbo": {
					Name:          "gpt-3.5-turbo",
					InputCost:     0.5,
					OutputCost:    1.5,
					SupportsChat:  true,
					SupportsEmbed: false,
				},
			},
		}

		request := mcp.CompletionRequest{
			Model:       "gpt-3.5-turbo",
			Prompt:      "Hello!",
			MaxTokens:   100,
			Temperature: 0.7,
		}

		ctx := context.Background()
		result, err := provider.Complete(ctx, request)
		if err != nil {
			t.Fatalf("Completion failed: %v", err)
		}

		if result.Text != "Hello! How can I assist you?" {
			t.Errorf("Unexpected response text: %s", result.Text)
		}

		if result.TokensUsed != 20 {
			t.Errorf("Expected 20 tokens used, got %d", result.TokensUsed)
		}

		if result.Provider != "openai" {
			t.Errorf("Expected provider 'openai', got %s", result.Provider)
		}
	})

	// Test embedding
	t.Run("embedding", func(t *testing.T) {
		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"embedding": []interface{}{0.1, 0.2, 0.3, 0.4, 0.5},
				},
			},
			"usage": map[string]interface{}{
				"total_tokens": 5.0,
			},
		}

		server := mockOpenAIServer(t, "embeddings", response, 200)
		defer server.Close()

		provider := &mcp.OpenAIProvider{
			APIKey:  "test-key",
			BaseURL: server.URL,
			HTTPClient: &http.Client{
				Timeout: 5 * time.Second,
			},
			Models: map[string]mcp.ModelConfig{
				"text-embedding-ada-002": {
					Name:          "text-embedding-ada-002",
					InputCost:     0.1,
					OutputCost:    0.0,
					SupportsChat:  false,
					SupportsEmbed: true,
				},
			},
		}

		request := mcp.EmbeddingRequest{
			Model: "text-embedding-ada-002",
			Text:  "Hello!",
		}

		ctx := context.Background()
		result, err := provider.Embed(ctx, request)
		if err != nil {
			t.Fatalf("Embedding failed: %v", err)
		}

		expectedEmbedding := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
		if len(result.Embedding) != len(expectedEmbedding) {
			t.Errorf("Expected embedding length %d, got %d", len(expectedEmbedding), len(result.Embedding))
		}

		for i, val := range expectedEmbedding {
			if result.Embedding[i] != val {
				t.Errorf("Expected embedding[%d] = %f, got %f", i, val, result.Embedding[i])
			}
		}

		if result.TokensUsed != 5 {
			t.Errorf("Expected 5 tokens used, got %d", result.TokensUsed)
		}

		if result.Provider != "openai" {
			t.Errorf("Expected provider 'openai', got %s", result.Provider)
		}
	})
}

// TestLLMLocalProvider tests the local provider implementation.
func TestLLMLocalProvider(t *testing.T) {
	// Create mock local server
	response := map[string]interface{}{
		"results": []map[string]interface{}{
			{
				"text": " I'd be happy to help you with that!",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/generate" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	provider := &mcp.LocalProvider{
		ServerURL: server.URL,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		Models: map[string]mcp.ModelConfig{
			"local-llama": {
				Name:      "llama-2-7b-chat",
				InputCost: 0.0,
			},
		},
	}

	// Test completion
	request := mcp.CompletionRequest{
		Model:       "local-llama",
		Prompt:      "Hello!",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	ctx := context.Background()
	result, err := provider.Complete(ctx, request)
	if err != nil {
		t.Fatalf("Completion failed: %v", err)
	}

	if result.Text != " I'd be happy to help you with that!" {
		t.Errorf("Unexpected response text: %s", result.Text)
	}

	if result.Cost != 0.0 {
		t.Errorf("Expected cost 0.0 for local provider, got %f", result.Cost)
	}

	if result.Provider != "local" {
		t.Errorf("Expected provider 'local', got %s", result.Provider)
	}

	// Test embedding (should fail)
	embeddingRequest := mcp.EmbeddingRequest{
		Model: "local-llama",
		Text:  "Hello!",
	}

	_, err = provider.Embed(ctx, embeddingRequest)
	if err == nil {
		t.Errorf("Expected embedding to fail for local provider")
	}
}

// TestLLMErrorHandling tests error handling and retry logic.
func TestLLMErrorHandling(t *testing.T) {
	// Test rate limiting retry
	t.Run("rate_limiting", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount <= 2 {
				// Return rate limit error for first two calls
				w.WriteHeader(429)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"message": "Rate limit exceeded",
					},
				})
				return
			}

			// Succeed on third call
			w.WriteHeader(200)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": "Success after retries!",
					},
				},
				"usage": map[string]interface{}{
					"input_tokens":  5.0,
					"output_tokens": 10.0,
				},
			})
		}))
		defer server.Close()

		provider := &mcp.AnthropicProvider{
			APIKey:  "test-key",
			BaseURL: server.URL,
			HTTPClient: &http.Client{
				Timeout: 5 * time.Second,
			},
			Models: map[string]mcp.ModelConfig{
				"claude-3-haiku": {
					Name:       "claude-3-haiku",
					InputCost:  0.25,
					OutputCost: 1.25,
				},
			},
		}

		// Create LLM service with fast retry for testing
		service := mcp.NewLLMService(nil)

		// Manually set the provider for testing
		service.SetProvider("anthropic", provider)
		service.SetRetryConfig(mcp.RetryConfig{
			MaxRetries:  3,
			BaseDelay:   10 * time.Millisecond,
			MaxDelay:    100 * time.Millisecond,
			BackoffRate: 2.0,
		})

		params := mcp.ServiceParams{
			"operation": "complete",
			"prompt":    "Hello!",
			"provider":  "anthropic",
			"model":     "claude-3-haiku",
		}

		result := service.Execute(context.Background(), params)
		if !result.Success {
			t.Fatalf("Expected success after retries, got error: %v", result.Error)
		}

		completionResp, ok := result.Data.(*mcp.CompletionResponse)
		if !ok {
			t.Fatalf("Expected CompletionResponse, got %T", result.Data)
		}

		if completionResp.Text != "Success after retries!" {
			t.Errorf("Unexpected response text: %s", completionResp.Text)
		}

		// Verify that retries happened (should be 3 calls total)
		if callCount != 3 {
			t.Errorf("Expected 3 API calls (2 failed + 1 success), got %d", callCount)
		}
	})

	// Test non-retryable error
	t.Run("non_retryable_error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Return authentication error (non-retryable)
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Invalid API key",
				},
			})
		}))
		defer server.Close()

		provider := &mcp.AnthropicProvider{
			APIKey:  "invalid-key",
			BaseURL: server.URL,
			HTTPClient: &http.Client{
				Timeout: 5 * time.Second,
			},
		}

		request := mcp.CompletionRequest{
			Model:  "claude-3-haiku",
			Prompt: "Hello!",
		}

		ctx := context.Background()
		_, err := provider.Complete(ctx, request)
		if err == nil {
			t.Errorf("Expected error for invalid API key")
		}

		if !strings.Contains(err.Error(), "API error (status 401)") {
			t.Errorf("Expected authentication error, got: %v", err)
		}
	})
}

// TestLLMBudgetLimits tests budget limit enforcement.
func TestLLMBudgetLimits(t *testing.T) {
	// Create service with environment to have at least one provider
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	service := mcp.NewLLMService(nil)

	// Set a very low budget limit for testing
	service.SetBudgetLimit(0.01) // $0.01 limit

	// Simulate exceeding the budget
	service.UpdateBudgetForTest("test", "complete", 1000, 0.02) // $0.02 cost

	// Try to make a request that should be blocked
	params := mcp.ServiceParams{
		"operation": "complete",
		"prompt":    "Hello!",
		"provider":  "anthropic", // Explicitly specify provider
		"model":     "claude-3-haiku",
	}

	result := service.Execute(context.Background(), params)
	if result.Success {
		t.Errorf("Expected budget limit error, got success")
	}

	if !strings.Contains(result.Error.Error(), "budget limit") {
		t.Errorf("Expected budget limit error, got: %v", result.Error)
	}
}

// TestLLMProviderSelection tests automatic provider selection logic.
func TestLLMProviderSelection(t *testing.T) {
	// Set up multiple providers
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	os.Setenv("OPENAI_API_KEY", "test-key")
	os.Setenv("LOCAL_LLM_URL", "http://localhost:5000")
	defer func() {
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("LOCAL_LLM_URL")
	}()

	service := mcp.NewLLMService(nil)

	// Check that we have the expected number of providers
	expectedProviders := 3 // anthropic, openai, local
	if service.GetProviderCount() != expectedProviders {
		t.Fatalf("Expected %d providers, got %d", expectedProviders, service.GetProviderCount())
	}

	tests := []struct {
		name             string
		operation        string
		explicitProvider string
		shouldValidate   bool
	}{
		{
			name:           "completion auto-select",
			operation:      "complete",
			shouldValidate: true,
		},
		{
			name:             "completion explicit anthropic",
			operation:        "complete",
			explicitProvider: "anthropic",
			shouldValidate:   true,
		},
		{
			name:           "embedding auto-select",
			operation:      "embed",
			shouldValidate: true,
		},
		{
			name:             "embedding explicit openai",
			operation:        "embed",
			explicitProvider: "openai",
			shouldValidate:   true,
		},
		{
			name:             "invalid provider",
			operation:        "complete",
			explicitProvider: "invalid",
			shouldValidate:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := mcp.ServiceParams{
				"operation": tt.operation,
			}

			if tt.operation == "complete" {
				params["prompt"] = "Hello!"
			} else if tt.operation == "embed" {
				params["text"] = "Hello!"
			}

			if tt.explicitProvider != "" {
				params["provider"] = tt.explicitProvider
			}

			// Test parameter validation (which includes provider selection validation)
			err := service.ValidateParams(params)
			if tt.shouldValidate && err != nil {
				t.Errorf("Validation failed: %v", err)
			}
			if !tt.shouldValidate && err == nil {
				t.Errorf("Expected validation to fail, but it succeeded")
			}
		})
	}
}

// TestLLMIntegration tests the LLM service with the MCP framework.
func TestLLMIntegration(t *testing.T) {
	// Create service registry
	registry := mcp.NewServiceRegistry(nil)

	// Create and register LLM service
	llmService := mcp.NewLLMService(nil)
	err := registry.RegisterService(llmService)
	if err != nil {
		t.Fatalf("Failed to register LLM service: %v", err)
	}

	// Test service discovery
	services := registry.ListServices()
	found := false
	for _, service := range services {
		if service.Name == "llm" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("LLM service not found in registry")
	}

	// Test service call through registry
	params := mcp.ServiceParams{
		"operation": "list_providers",
	}

	result := registry.CallService(context.Background(), "llm", params)
	if !result.Success {
		t.Fatalf("Service call failed: %v", result.Error)
	}

	// Verify registry metadata was added
	if result.Metadata == nil {
		t.Errorf("Expected metadata in result")
	}

	if registryCall, exists := result.Metadata["registry_call"]; !exists || !registryCall.(bool) {
		t.Errorf("Expected registry_call metadata")
	}
}