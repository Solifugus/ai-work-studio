// Package llm provides intelligent LLM routing and budget management for the AI Work Studio.
//
// This package implements two main components:
//
// 1. Router: Intelligent task assessment and model selection
//    - Analyzes task complexity, token requirements, and quality needs
//    - Selects the most cost-effective model meeting requirements
//    - Learns from historical performance to improve routing decisions
//    - Provides cost estimation before execution
//
// 2. BudgetManager: Comprehensive budget tracking and alerts
//    - Tracks spending across daily, weekly, and monthly periods
//    - Triggers alerts at configurable thresholds (75%, 90%, 100%)
//    - Provides detailed ROI analysis for different providers and models
//    - Enforces budget limits with optional grace periods
//
// The router uses a multi-factor scoring algorithm that balances:
//   - Quality requirements vs model capabilities
//   - Cost constraints and budget limits
//   - Speed requirements
//   - Historical performance data (when available)
//
// Example usage:
//
//	// Create router with LLM service
//	router := llm.NewRouter(llmService)
//
//	// Route a task
//	result, err := router.Route(ctx, llm.TaskRequest{
//	    Prompt: "Analyze this financial data...",
//	    TaskType: "analysis",
//	    QualityRequired: llm.QualityStandard,
//	    MaxTokens: 1000,
//	})
//
//	// Create budget manager
//	budgetManager, err := llm.NewBudgetManager("/data/budget",
//	    llm.DefaultBudgetConfig(), logger)
//
//	// Check affordability before execution
//	affordability, err := budgetManager.CanAfford(0.05)
//	if !affordability.Affordable {
//	    // Handle budget constraint
//	}
//
//	// Record usage after execution
//	transaction := llm.Transaction{
//	    Provider: "anthropic",
//	    Model: "claude-3-haiku",
//	    Cost: 0.05,
//	    TokensUsed: 1000,
//	    Success: true,
//	    Quality: 8.5,
//	}
//	err = budgetManager.RecordUsage(ctx, transaction)
//
// The package is designed to work seamlessly with the existing MCP LLM service
// while providing enhanced routing intelligence and budget controls.
package llm