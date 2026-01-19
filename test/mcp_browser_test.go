package test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Solifugus/ai-work-studio/pkg/mcp"
)

// TestBrowserServiceIntegration runs integration tests for the browser service.
// These tests require a real Chrome browser to be available.
func TestBrowserServiceIntegration(t *testing.T) {
	// Skip integration tests if running in CI or if no browser available
	if os.Getenv("SKIP_BROWSER_TESTS") == "true" {
		t.Skip("Skipping browser integration tests")
	}

	logger := log.New(os.Stdout, "[BROWSER_TEST] ", log.LstdFlags)
	service := mcp.NewBrowserService(logger)
	defer service.Close()

	ctx := context.Background()

	t.Run("ValidateParams", func(t *testing.T) {
		tests := []struct {
			name    string
			params  mcp.ServiceParams
			wantErr bool
		}{
			{
				name:    "missing operation",
				params:  mcp.ServiceParams{},
				wantErr: true,
			},
			{
				name:    "invalid operation",
				params:  mcp.ServiceParams{"operation": "invalid"},
				wantErr: true,
			},
			{
				name:    "navigate without url",
				params:  mcp.ServiceParams{"operation": "navigate"},
				wantErr: true,
			},
			{
				name:    "valid navigate",
				params:  mcp.ServiceParams{"operation": "navigate", "url": "https://example.com"},
				wantErr: false,
			},
			{
				name:    "valid get_text",
				params:  mcp.ServiceParams{"operation": "get_text"},
				wantErr: false,
			},
			{
				name:    "get_element without selector",
				params:  mcp.ServiceParams{"operation": "get_element"},
				wantErr: true,
			},
			{
				name:    "valid get_element",
				params:  mcp.ServiceParams{"operation": "get_element", "selector": "body"},
				wantErr: false,
			},
			{
				name:    "click without selector",
				params:  mcp.ServiceParams{"operation": "click"},
				wantErr: true,
			},
			{
				name:    "valid click",
				params:  mcp.ServiceParams{"operation": "click", "selector": "button"},
				wantErr: false,
			},
			{
				name:    "valid screenshot",
				params:  mcp.ServiceParams{"operation": "screenshot"},
				wantErr: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := service.ValidateParams(tt.params)
				if (err != nil) != tt.wantErr {
					t.Errorf("ValidateParams() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})

	t.Run("NavigateToRealSite", func(t *testing.T) {
		params := mcp.ServiceParams{
			"operation": "navigate",
			"url":       "https://httpbin.org/html",
		}

		result := mcp.CallService(ctx, service, params)
		if !result.Success {
			t.Fatalf("Navigate failed: %v", result.Error)
		}

		browserResult, ok := result.Data.(mcp.BrowserResult)
		if !ok {
			t.Fatalf("Expected BrowserResult, got %T", result.Data)
		}

		if browserResult.Operation != "navigate" {
			t.Errorf("Expected operation 'navigate', got '%s'", browserResult.Operation)
		}

		if browserResult.URL == "" {
			t.Error("Expected URL to be set")
		}
	})

	t.Run("GetTextFromPage", func(t *testing.T) {
		// First navigate to a page
		navParams := mcp.ServiceParams{
			"operation": "navigate",
			"url":       "https://httpbin.org/html",
		}

		navResult := mcp.CallService(ctx, service, navParams)
		if !navResult.Success {
			t.Fatalf("Navigate failed: %v", navResult.Error)
		}

		// Then get text
		textParams := mcp.ServiceParams{
			"operation": "get_text",
		}

		result := mcp.CallService(ctx, service, textParams)
		if !result.Success {
			t.Fatalf("GetText failed: %v", result.Error)
		}

		browserResult, ok := result.Data.(mcp.BrowserResult)
		if !ok {
			t.Fatalf("Expected BrowserResult, got %T", result.Data)
		}

		if browserResult.Text == "" {
			t.Error("Expected text to be extracted")
		}

		if browserResult.Operation != "get_text" {
			t.Errorf("Expected operation 'get_text', got '%s'", browserResult.Operation)
		}
	})
}

// TestBrowserServiceWithMockServer tests browser service against a controlled test server.
func TestBrowserServiceWithMockServer(t *testing.T) {
	// Skip if browser tests are disabled
	if os.Getenv("SKIP_BROWSER_TESTS") == "true" {
		t.Skip("Skipping browser integration tests")
	}

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `
<!DOCTYPE html>
<html>
<head>
	<title>Test Page</title>
</head>
<body>
	<h1 id="title">Hello World</h1>
	<p class="content">This is a test page for browser automation.</p>
	<button id="test-button">Click Me</button>
	<div id="hidden" style="display: none;">Hidden content</div>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, html)
	}))
	defer server.Close()

	logger := log.New(os.Stdout, "[BROWSER_TEST] ", log.LstdFlags)
	service := mcp.NewBrowserService(logger)
	defer service.Close()

	ctx := context.Background()

	t.Run("NavigateToTestServer", func(t *testing.T) {
		params := mcp.ServiceParams{
			"operation": "navigate",
			"url":       server.URL,
		}

		result := mcp.CallService(ctx, service, params)
		if !result.Success {
			t.Fatalf("Navigate failed: %v", result.Error)
		}

		browserResult, ok := result.Data.(mcp.BrowserResult)
		if !ok {
			t.Fatalf("Expected BrowserResult, got %T", result.Data)
		}

		if !browserResult.Success {
			t.Errorf("Expected navigation to succeed")
		}
	})

	t.Run("GetElementById", func(t *testing.T) {
		// Navigate first
		navParams := mcp.ServiceParams{
			"operation": "navigate",
			"url":       server.URL,
		}
		navResult := mcp.CallService(ctx, service, navParams)
		if !navResult.Success {
			t.Fatalf("Navigate failed: %v", navResult.Error)
		}

		// Get element by ID
		params := mcp.ServiceParams{
			"operation": "get_element",
			"selector":  "#title",
		}

		result := mcp.CallService(ctx, service, params)
		if !result.Success {
			t.Fatalf("GetElement failed: %v", result.Error)
		}

		browserResult, ok := result.Data.(mcp.BrowserResult)
		if !ok {
			t.Fatalf("Expected BrowserResult, got %T", result.Data)
		}

		if !browserResult.ElementFound {
			t.Error("Expected element to be found")
		}

		if browserResult.ElementText != "Hello World" {
			t.Errorf("Expected 'Hello World', got '%s'", browserResult.ElementText)
		}
	})

	t.Run("GetElementByClass", func(t *testing.T) {
		// Navigate first
		navParams := mcp.ServiceParams{
			"operation": "navigate",
			"url":       server.URL,
		}
		mcp.CallService(ctx, service, navParams)

		// Get element by class
		params := mcp.ServiceParams{
			"operation": "get_element",
			"selector":  ".content",
		}

		result := mcp.CallService(ctx, service, params)
		if !result.Success {
			t.Fatalf("GetElement failed: %v", result.Error)
		}

		browserResult := result.Data.(mcp.BrowserResult)
		if !browserResult.ElementFound {
			t.Error("Expected element to be found")
		}

		expected := "This is a test page for browser automation."
		if browserResult.ElementText != expected {
			t.Errorf("Expected '%s', got '%s'", expected, browserResult.ElementText)
		}
	})

	t.Run("ClickElement", func(t *testing.T) {
		// Navigate first
		navParams := mcp.ServiceParams{
			"operation": "navigate",
			"url":       server.URL,
		}
		mcp.CallService(ctx, service, navParams)

		// Click button
		params := mcp.ServiceParams{
			"operation": "click",
			"selector":  "#test-button",
		}

		result := mcp.CallService(ctx, service, params)
		if !result.Success {
			t.Fatalf("Click failed: %v", result.Error)
		}

		browserResult := result.Data.(mcp.BrowserResult)
		if !browserResult.Success {
			t.Error("Expected click to succeed")
		}
	})

	t.Run("TakeScreenshot", func(t *testing.T) {
		// Navigate first
		navParams := mcp.ServiceParams{
			"operation": "navigate",
			"url":       server.URL,
		}
		mcp.CallService(ctx, service, navParams)

		// Take screenshot
		params := mcp.ServiceParams{
			"operation": "screenshot",
		}

		result := mcp.CallService(ctx, service, params)
		if !result.Success {
			t.Fatalf("Screenshot failed: %v", result.Error)
		}

		browserResult := result.Data.(mcp.BrowserResult)
		if browserResult.Screenshot == "" {
			t.Error("Expected screenshot data to be present")
		}

		// Screenshot should be base64 encoded
		if len(browserResult.Screenshot) < 100 {
			t.Error("Screenshot data seems too small to be a valid image")
		}
	})

	t.Run("ElementNotFound", func(t *testing.T) {
		// Navigate first
		navParams := mcp.ServiceParams{
			"operation": "navigate",
			"url":       server.URL,
		}
		mcp.CallService(ctx, service, navParams)

		// Try to get non-existent element
		params := mcp.ServiceParams{
			"operation": "get_element",
			"selector":  "#nonexistent",
		}

		result := mcp.CallService(ctx, service, params)
		if result.Success {
			t.Error("Expected get_element to fail for non-existent element")
		}

		browserResult := result.Data.(mcp.BrowserResult)
		if browserResult.ElementFound {
			t.Error("Expected element not to be found")
		}
	})
}

// TestBrowserServiceTimeout tests timeout handling.
func TestBrowserServiceTimeout(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") == "true" {
		t.Skip("Skipping browser integration tests")
	}

	logger := log.New(os.Stdout, "[BROWSER_TEST] ", log.LstdFlags)
	service := mcp.NewBrowserService(logger)
	defer service.Close()

	// Set a very short timeout for this test
	service.SetTimeout(1 * time.Millisecond)

	ctx := context.Background()

	params := mcp.ServiceParams{
		"operation": "navigate",
		"url":       "https://httpbin.org/delay/5", // This will take 5 seconds
	}

	result := mcp.CallService(ctx, service, params)
	if result.Success {
		t.Error("Expected navigation to fail due to timeout")
	}

	if result.Error == nil {
		t.Error("Expected an error due to timeout")
	}
}

// TestBrowserServiceRateLimit tests rate limiting between navigations.
func TestBrowserServiceRateLimit(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") == "true" {
		t.Skip("Skipping browser integration tests")
	}

	logger := log.New(os.Stdout, "[BROWSER_TEST] ", log.LstdFlags)
	service := mcp.NewBrowserService(logger)
	defer service.Close()

	// Set rate limit to 2 seconds for testing
	service.SetNavigationInterval(2 * time.Second)

	ctx := context.Background()

	start := time.Now()

	// First navigation
	params1 := mcp.ServiceParams{
		"operation": "navigate",
		"url":       "https://httpbin.org/html",
	}
	mcp.CallService(ctx, service, params1)

	// Second navigation should be rate limited
	params2 := mcp.ServiceParams{
		"operation": "navigate",
		"url":       "https://example.com",
	}
	mcp.CallService(ctx, service, params2)

	elapsed := time.Since(start)

	// Should have taken at least 2 seconds due to rate limiting
	if elapsed < 2*time.Second {
		t.Errorf("Expected rate limiting to enforce 2 second delay, but took %v", elapsed)
	}
}

// BenchmarkBrowserOperations benchmarks different browser operations.
func BenchmarkBrowserOperations(b *testing.B) {
	if os.Getenv("SKIP_BROWSER_TESTS") == "true" {
		b.Skip("Skipping browser integration tests")
	}

	logger := log.New(os.Stdout, "[BROWSER_BENCH] ", log.LstdFlags)
	service := mcp.NewBrowserService(logger)
	defer service.Close()

	ctx := context.Background()

	// Setup: Navigate to a test page once
	navParams := mcp.ServiceParams{
		"operation": "navigate",
		"url":       "https://httpbin.org/html",
	}
	mcp.CallService(ctx, service, navParams)

	b.Run("GetText", func(b *testing.B) {
		params := mcp.ServiceParams{
			"operation": "get_text",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			mcp.CallService(ctx, service, params)
		}
	})

	b.Run("GetElement", func(b *testing.B) {
		params := mcp.ServiceParams{
			"operation": "get_element",
			"selector":  "body",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			mcp.CallService(ctx, service, params)
		}
	})

	b.Run("Screenshot", func(b *testing.B) {
		params := mcp.ServiceParams{
			"operation": "screenshot",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			mcp.CallService(ctx, service, params)
		}
	})
}