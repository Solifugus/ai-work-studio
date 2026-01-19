package mcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

// BrowserService provides web browser automation capabilities through MCP.
// It supports navigation, text extraction, element interaction, and screenshots.
type BrowserService struct {
	*BaseService
	allocatorCtx context.Context
	cancel       context.CancelFunc
	timeout      time.Duration
	lastNav      time.Time // For rate limiting
	navInterval  time.Duration
}

// BrowserResult represents the result of a browser operation.
type BrowserResult struct {
	// Operation performed (navigate, get_text, etc.)
	Operation string `json:"operation"`

	// URL of the current page
	URL string `json:"url"`

	// Text content (for get_text operations)
	Text string `json:"text,omitempty"`

	// Element found (for get_element operations)
	ElementFound bool `json:"element_found,omitempty"`

	// Element text content
	ElementText string `json:"element_text,omitempty"`

	// Screenshot data (base64 encoded PNG)
	Screenshot string `json:"screenshot,omitempty"`

	// Whether operation was successful
	Success bool `json:"success"`

	// Any error message
	Message string `json:"message,omitempty"`
}

// NewBrowserService creates a new browser automation service.
func NewBrowserService(logger *log.Logger) *BrowserService {
	base := NewBaseService("browser", "Web browser automation service", logger)

	// Create allocator context for Chrome browser
	allocatorCtx, cancel := chromedp.NewExecAllocator(context.Background(),
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("no-sandbox", true),
	)

	return &BrowserService{
		BaseService:  base,
		allocatorCtx: allocatorCtx,
		cancel:       cancel,
		timeout:      30 * time.Second,
		navInterval:  1 * time.Second, // Rate limiting: 1 second between navigations
	}
}

// ValidateParams validates parameters for browser operations.
func (bs *BrowserService) ValidateParams(params ServiceParams) error {
	if err := bs.BaseService.ValidateParams(params); err != nil {
		return err
	}

	operation, exists := params["operation"]
	if !exists {
		return NewValidationError("operation", "required parameter is missing")
	}

	op, ok := operation.(string)
	if !ok {
		return NewValidationError("operation", "must be a string")
	}

	switch op {
	case "navigate":
		return ValidateStringParam(params, "url", true)
	case "get_text":
		// No additional params required for get_text
		return nil
	case "get_element":
		return ValidateStringParam(params, "selector", true)
	case "click":
		return ValidateStringParam(params, "selector", true)
	case "screenshot":
		// No additional params required for screenshot
		return nil
	default:
		return NewValidationError("operation",
			"must be one of: navigate, get_text, get_element, click, screenshot")
	}
}

// Execute performs the browser automation operation.
func (bs *BrowserService) Execute(ctx context.Context, params ServiceParams) ServiceResult {
	operation := params["operation"].(string)

	// Apply timeout to context
	ctx, cancel := context.WithTimeout(ctx, bs.timeout)
	defer cancel()

	// Create browser context
	browserCtx, cancel := chromedp.NewContext(bs.allocatorCtx)
	defer cancel()

	switch operation {
	case "navigate":
		url := params["url"].(string)
		return bs.navigate(browserCtx, url)
	case "get_text":
		return bs.getText(browserCtx)
	case "get_element":
		selector := params["selector"].(string)
		return bs.getElement(browserCtx, selector)
	case "click":
		selector := params["selector"].(string)
		return bs.click(browserCtx, selector)
	case "screenshot":
		return bs.screenshot(browserCtx)
	default:
		return ErrorResult(fmt.Errorf("unsupported operation: %s", operation))
	}
}

// navigate navigates to the specified URL with rate limiting.
func (bs *BrowserService) navigate(ctx context.Context, url string) ServiceResult {
	// Rate limiting
	if time.Since(bs.lastNav) < bs.navInterval {
		time.Sleep(bs.navInterval - time.Since(bs.lastNav))
	}
	bs.lastNav = time.Now()

	// Validate URL
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	var currentURL string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Location(&currentURL),
	)

	result := BrowserResult{
		Operation: "navigate",
		URL:       currentURL,
		Success:   err == nil,
	}

	if err != nil {
		result.Message = err.Error()
		return ServiceResult{
			Success: false,
			Error:   err,
			Data:    result,
		}
	}

	return SuccessResult(result)
}

// getText extracts all visible text from the current page.
func (bs *BrowserService) getText(ctx context.Context) ServiceResult {
	var text string
	var currentURL string

	err := chromedp.Run(ctx,
		chromedp.Text("body", &text, chromedp.ByQuery),
		chromedp.Location(&currentURL),
	)

	result := BrowserResult{
		Operation: "get_text",
		URL:       currentURL,
		Text:      strings.TrimSpace(text),
		Success:   err == nil,
	}

	if err != nil {
		result.Message = err.Error()
		return ServiceResult{
			Success: false,
			Error:   err,
			Data:    result,
		}
	}

	return SuccessResult(result)
}

// getElement finds an element by CSS selector and returns its text content.
func (bs *BrowserService) getElement(ctx context.Context, selector string) ServiceResult {
	var elementText string
	var currentURL string
	var elementFound bool

	err := chromedp.Run(ctx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.Text(selector, &elementText, chromedp.ByQuery),
		chromedp.Location(&currentURL),
	)

	if err == nil {
		elementFound = true
	} else {
		// Try to check if element exists without waiting
		var nodes []*cdp.Node
		chromedp.Run(ctx, chromedp.Nodes(selector, &nodes, chromedp.ByQuery))
		elementFound = len(nodes) > 0
		if elementFound && elementText == "" {
			chromedp.Run(ctx, chromedp.Text(selector, &elementText, chromedp.ByQuery))
		}
	}

	result := BrowserResult{
		Operation:    "get_element",
		URL:          currentURL,
		ElementFound: elementFound,
		ElementText:  strings.TrimSpace(elementText),
		Success:      elementFound,
	}

	if !elementFound {
		result.Message = fmt.Sprintf("element not found: %s", selector)
		return ServiceResult{
			Success: false,
			Error:   fmt.Errorf("element not found: %s", selector),
			Data:    result,
		}
	}

	return SuccessResult(result)
}

// click clicks on an element specified by CSS selector.
func (bs *BrowserService) click(ctx context.Context, selector string) ServiceResult {
	var currentURL string

	err := chromedp.Run(ctx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.Click(selector, chromedp.ByQuery),
		chromedp.Location(&currentURL),
	)

	result := BrowserResult{
		Operation: "click",
		URL:       currentURL,
		Success:   err == nil,
	}

	if err != nil {
		result.Message = err.Error()
		return ServiceResult{
			Success: false,
			Error:   err,
			Data:    result,
		}
	}

	return SuccessResult(result)
}

// screenshot captures a full page screenshot.
func (bs *BrowserService) screenshot(ctx context.Context) ServiceResult {
	var buf []byte
	var currentURL string

	err := chromedp.Run(ctx,
		chromedp.FullScreenshot(&buf, 90),
		chromedp.Location(&currentURL),
	)

	result := BrowserResult{
		Operation: "screenshot",
		URL:       currentURL,
		Success:   err == nil,
	}

	if err != nil {
		result.Message = err.Error()
		return ServiceResult{
			Success: false,
			Error:   err,
			Data:    result,
		}
	}

	// Encode screenshot as base64
	result.Screenshot = base64.StdEncoding.EncodeToString(buf)

	return SuccessResult(result)
}

// Close cleans up the browser service resources.
func (bs *BrowserService) Close() {
	if bs.cancel != nil {
		bs.cancel()
	}
}

// SetTimeout configures the timeout for browser operations.
func (bs *BrowserService) SetTimeout(timeout time.Duration) {
	bs.timeout = timeout
}

// SetNavigationInterval configures the rate limiting interval between navigations.
func (bs *BrowserService) SetNavigationInterval(interval time.Duration) {
	bs.navInterval = interval
}