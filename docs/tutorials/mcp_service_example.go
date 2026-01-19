// Package tutorials provides example code demonstrating AI Work Studio usage.
package tutorials

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Solifugus/ai-work-studio/pkg/mcp"
	"github.com/Solifugus/ai-work-studio/pkg/utils"
)

// TextProcessingService demonstrates how to create a custom MCP service.
// This service provides various text processing capabilities.
type TextProcessingService struct {
	*mcp.BaseService
}

// NewTextProcessingService creates a new text processing service.
func NewTextProcessingService() *TextProcessingService {
	base := mcp.NewBaseService(
		"text-processor",
		"Provides text processing capabilities like word count, sentiment analysis, and formatting",
		nil, // No logger for this example
	)
	return &TextProcessingService{BaseService: base}
}

// ValidateParams validates the parameters for text processing operations.
func (s *TextProcessingService) ValidateParams(params mcp.ServiceParams) error {
	// Validate required 'operation' parameter
	operation, exists := params["operation"]
	if !exists {
		return fmt.Errorf("missing required parameter: operation")
	}

	operationStr, ok := operation.(string)
	if !ok {
		return fmt.Errorf("operation parameter must be a string")
	}

	// Validate operation type
	validOps := []string{"word_count", "sentiment", "format", "extract_keywords"}
	valid := false
	for _, op := range validOps {
		if operationStr == op {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("unsupported operation: %s (valid: %s)", operationStr, strings.Join(validOps, ", "))
	}

	// Validate required 'text' parameter
	return mcp.ValidateStringParam(params, "text", true)
}

// Execute performs the text processing operation.
func (s *TextProcessingService) Execute(ctx context.Context, params mcp.ServiceParams) mcp.ServiceResult {
	operation := params["operation"].(string)
	text := params["text"].(string)

	switch operation {
	case "word_count":
		return s.wordCount(text)
	case "sentiment":
		return s.sentimentAnalysis(text)
	case "format":
		return s.formatText(text, params)
	case "extract_keywords":
		return s.extractKeywords(text)
	default:
		return mcp.ErrorResult(fmt.Errorf("unsupported operation: %s", operation))
	}
}

// wordCount counts words, characters, and lines in the text.
func (s *TextProcessingService) wordCount(text string) mcp.ServiceResult {
	words := strings.Fields(text)
	lines := strings.Split(text, "\n")

	result := map[string]interface{}{
		"words":      len(words),
		"characters": len(text),
		"lines":      len(lines),
		"paragraphs": len(strings.Split(strings.TrimSpace(text), "\n\n")),
	}

	return mcp.SuccessResult(result)
}

// sentimentAnalysis provides basic sentiment analysis.
func (s *TextProcessingService) sentimentAnalysis(text string) mcp.ServiceResult {
	// Simple sentiment analysis based on keyword matching
	positiveWords := []string{"good", "great", "excellent", "amazing", "wonderful", "fantastic", "love", "like", "happy", "pleased"}
	negativeWords := []string{"bad", "terrible", "awful", "horrible", "hate", "dislike", "sad", "angry", "frustrated", "disappointed"}

	textLower := strings.ToLower(text)
	positiveCount := 0
	negativeCount := 0

	for _, word := range positiveWords {
		positiveCount += strings.Count(textLower, word)
	}

	for _, word := range negativeWords {
		negativeCount += strings.Count(textLower, word)
	}

	var sentiment string
	var score float64

	if positiveCount > negativeCount {
		sentiment = "positive"
		score = float64(positiveCount) / float64(positiveCount+negativeCount)
	} else if negativeCount > positiveCount {
		sentiment = "negative"
		score = float64(negativeCount) / float64(positiveCount+negativeCount)
	} else {
		sentiment = "neutral"
		score = 0.5
	}

	result := map[string]interface{}{
		"sentiment":       sentiment,
		"confidence":      score,
		"positive_words":  positiveCount,
		"negative_words":  negativeCount,
		"total_analyzed":  positiveCount + negativeCount,
	}

	return mcp.SuccessResult(result)
}

// formatText applies basic text formatting.
func (s *TextProcessingService) formatText(text string, params mcp.ServiceParams) mcp.ServiceResult {
	format, exists := params["format"]
	if !exists {
		format = "clean" // default format
	}

	formatStr, ok := format.(string)
	if !ok {
		return mcp.ErrorResult(fmt.Errorf("format parameter must be a string"))
	}

	var result string

	switch formatStr {
	case "clean":
		// Remove extra whitespace
		result = strings.TrimSpace(text)
		result = strings.ReplaceAll(result, "\t", " ")
		// Replace multiple spaces with single space
		for strings.Contains(result, "  ") {
			result = strings.ReplaceAll(result, "  ", " ")
		}

	case "uppercase":
		result = strings.ToUpper(text)

	case "lowercase":
		result = strings.ToLower(text)

	case "title":
		result = strings.Title(strings.ToLower(text))

	case "remove_punctuation":
		punctuation := ".,!?;:\"'()[]{}/"
		result = text
		for _, p := range punctuation {
			result = strings.ReplaceAll(result, string(p), "")
		}

	default:
		return mcp.ErrorResult(fmt.Errorf("unsupported format: %s", formatStr))
	}

	return mcp.SuccessResult(map[string]interface{}{
		"formatted_text": result,
		"original_text":  text,
		"format_applied": formatStr,
	})
}

// extractKeywords extracts potential keywords from text.
func (s *TextProcessingService) extractKeywords(text string) mcp.ServiceResult {
	// Simple keyword extraction based on word frequency
	words := strings.Fields(strings.ToLower(text))

	// Remove common stop words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "up": true, "about": true,
		"into": true, "through": true, "during": true, "before": true, "after": true,
		"above": true, "below": true, "over": true, "under": true, "is": true,
		"was": true, "were": true, "been": true, "be": true, "have": true,
		"has": true, "had": true, "do": true, "does": true, "did": true,
		"will": true, "would": true, "could": true, "should": true, "may": true,
		"might": true, "must": true, "can": true, "this": true, "that": true,
		"these": true, "those": true, "i": true, "you": true, "he": true,
		"she": true, "it": true, "we": true, "they": true, "them": true,
	}

	wordCount := make(map[string]int)
	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,!?;:\"'()[]{}/-")
		if len(word) > 2 && !stopWords[word] { // Only count words longer than 2 chars
			wordCount[word]++
		}
	}

	// Find most frequent words
	type wordFreq struct {
		Word  string `json:"word"`
		Count int    `json:"count"`
	}

	var keywords []wordFreq
	for word, count := range wordCount {
		if count > 1 { // Only include words that appear more than once
			keywords = append(keywords, wordFreq{Word: word, Count: count})
		}
	}

	// Sort by frequency (simple bubble sort for demo)
	for i := 0; i < len(keywords)-1; i++ {
		for j := 0; j < len(keywords)-i-1; j++ {
			if keywords[j].Count < keywords[j+1].Count {
				keywords[j], keywords[j+1] = keywords[j+1], keywords[j]
			}
		}
	}

	// Limit to top 10 keywords
	if len(keywords) > 10 {
		keywords = keywords[:10]
	}

	result := map[string]interface{}{
		"keywords":    keywords,
		"total_words": len(words),
		"unique_words": len(wordCount),
	}

	return mcp.SuccessResult(result)
}

// MCPServiceExample demonstrates how to create, register, and use MCP services.
func MCPServiceExample() error {
	fmt.Println("AI Work Studio - MCP Service Framework Example")
	fmt.Println("==============================================")

	// Setup logging
	logConfig := utils.DefaultLogConfig("mcp-example")
	logger, err := utils.NewLogger(logConfig)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	defer logger.Close()

	// Example 1: Create and register services
	fmt.Println("\n1. Creating service registry...")
	registry := mcp.NewServiceRegistry(logger)

	// Register our custom text processing service
	textService := NewTextProcessingService()
	err = registry.RegisterService(textService)
	if err != nil {
		return fmt.Errorf("failed to register text service: %w", err)
	}

	// Register built-in services
	fileService := mcp.NewFileSystemService()
	err = registry.RegisterService(fileService)
	if err != nil {
		log.Printf("Warning: failed to register file service: %v", err)
	}

	fmt.Printf("Registered services: %v\n", registry.ListServices())

	// Example 2: Call text processing service
	fmt.Println("\n2. Using text processing service...")

	sampleText := `
	AI Work Studio is a fantastic goal-directed autonomous agent system.
	It provides excellent capabilities for managing goals, objectives, and methods.
	The system is built with Go and uses a wonderful temporal storage approach.
	I love how it balances simplicity with powerful functionality.
	This is truly an amazing project that will help developers be more productive.
	`

	ctx := context.Background()

	// Word count operation
	params := mcp.ServiceParams{
		"operation": "word_count",
		"text":      sampleText,
	}

	result := registry.CallService(ctx, "text-processor", params)
	if result.Success {
		fmt.Printf("Word count result: %+v\n", result.Data)
	} else {
		log.Printf("Word count failed: %v", result.Error)
	}

	// Sentiment analysis
	params["operation"] = "sentiment"
	result = registry.CallService(ctx, "text-processor", params)
	if result.Success {
		fmt.Printf("Sentiment analysis: %+v\n", result.Data)
	} else {
		log.Printf("Sentiment analysis failed: %v", result.Error)
	}

	// Text formatting
	params = mcp.ServiceParams{
		"operation": "format",
		"text":      "  this   is    POORLY    formatted   TEXT!!!  ",
		"format":    "clean",
	}
	result = registry.CallService(ctx, "text-processor", params)
	if result.Success {
		fmt.Printf("Formatted text: %+v\n", result.Data)
	} else {
		log.Printf("Formatting failed: %v", result.Error)
	}

	// Keyword extraction
	params = mcp.ServiceParams{
		"operation": "extract_keywords",
		"text":      sampleText,
	}
	result = registry.CallService(ctx, "text-processor", params)
	if result.Success {
		fmt.Printf("Keywords: %+v\n", result.Data)
	} else {
		log.Printf("Keyword extraction failed: %v", result.Error)
	}

	// Example 3: Error handling
	fmt.Println("\n3. Demonstrating error handling...")

	// Try an invalid operation
	invalidParams := mcp.ServiceParams{
		"operation": "invalid_operation",
		"text":      "test text",
	}
	result = registry.CallService(ctx, "text-processor", invalidParams)
	if !result.Success {
		fmt.Printf("Expected error for invalid operation: %v\n", result.Error)
	}

	// Try missing parameters
	missingParams := mcp.ServiceParams{
		"operation": "word_count",
		// Missing "text" parameter
	}
	result = registry.CallService(ctx, "text-processor", missingParams)
	if !result.Success {
		fmt.Printf("Expected error for missing parameter: %v\n", result.Error)
	}

	// Example 4: Service metadata and introspection
	fmt.Println("\n4. Service introspection...")

	services := registry.ListServices()
	for _, serviceName := range services {
		service, exists := registry.GetService(serviceName)
		if exists {
			fmt.Printf("Service: %s - %s\n", service.Name(), service.Description())
		}
	}

	fmt.Println("\n✅ MCP Service example completed successfully!")
	return nil
}

// AsyncServiceExample demonstrates asynchronous service execution.
func AsyncServiceExample() {
	fmt.Println("Async MCP Service Example")
	fmt.Println("=========================")

	// This example would demonstrate:
	// 1. Long-running service operations
	// 2. Context cancellation
	// 3. Timeout handling
	// 4. Progress reporting

	fmt.Println("(This would demonstrate async service patterns)")
}

// ServiceChainExample demonstrates chaining multiple service calls.
func ServiceChainExample() {
	fmt.Println("Service Chain Example")
	fmt.Println("====================")

	// This example would show:
	// 1. Using output from one service as input to another
	// 2. Error handling in service chains
	// 3. Conditional execution based on service results
	// 4. Parallel vs sequential service execution

	fmt.Println("(This would demonstrate service chaining patterns)")
}

// BenchmarkService provides a simple benchmark for service performance.
type BenchmarkService struct {
	*mcp.BaseService
}

// NewBenchmarkService creates a service for performance testing.
func NewBenchmarkService() *BenchmarkService {
	base := mcp.NewBaseService(
		"benchmark",
		"Provides performance benchmarking capabilities",
		nil,
	)
	return &BenchmarkService{BaseService: base}
}

// ValidateParams validates benchmark parameters.
func (s *BenchmarkService) ValidateParams(params mcp.ServiceParams) error {
	return mcp.ValidateStringParam(params, "operation", true)
}

// Execute performs benchmark operations.
func (s *BenchmarkService) Execute(ctx context.Context, params mcp.ServiceParams) mcp.ServiceResult {
	operation := params["operation"].(string)
	start := time.Now()

	switch operation {
	case "cpu_intensive":
		// Simulate CPU-intensive work
		sum := 0
		for i := 0; i < 1000000; i++ {
			sum += i * i
		}

		elapsed := time.Since(start)
		return mcp.SuccessResult(map[string]interface{}{
			"operation": "cpu_intensive",
			"result":    sum,
			"duration":  elapsed.String(),
		})

	case "sleep":
		// Simulate I/O wait
		duration := 100 * time.Millisecond
		if d, exists := params["duration"]; exists {
			if dStr, ok := d.(string); ok {
				if parsed, err := time.ParseDuration(dStr); err == nil {
					duration = parsed
				}
			}
		}

		time.Sleep(duration)
		elapsed := time.Since(start)
		return mcp.SuccessResult(map[string]interface{}{
			"operation": "sleep",
			"requested": duration.String(),
			"actual":    elapsed.String(),
		})

	default:
		return mcp.ErrorResult(fmt.Errorf("unsupported benchmark operation: %s", operation))
	}
}