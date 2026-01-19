// Package utils provides utility functions and shared services for the AI Work Studio.
//
// The utils package contains common functionality used across different components
// of the AI Work Studio system, including:
//
//   - Structured logging system with JSON output, multiple destinations, and automatic rotation
//   - Configuration management utilities
//   - Common helper functions
//
// # Logging System
//
// The logging system provides structured, contextual logging with the following features:
//
//   - Multiple log levels: DEBUG, INFO, WARNING, ERROR
//   - JSON formatted output for structured parsing
//   - Configurable destinations: console, file, or both
//   - Automatic log file rotation with size and age limits
//   - Contextual logging with goal, objective, and method IDs
//   - Component-based logger management
//
// Example usage:
//
//	config := utils.DefaultLogConfig("my-component")
//	logger, err := utils.NewLogger(config)
//	if err != nil {
//	    panic(err)
//	}
//	defer logger.Close()
//
//	ctx := utils.LogContext{
//	    GoalID:      "goal-123",
//	    ObjectiveID: "obj-456",
//	    Component:   "my-component",
//	}
//
//	logger.Info(ctx, "Processing started")
//	logger.Error(ctx, "An error occurred", map[string]interface{}{
//	    "error_code": 500,
//	    "details":    "Connection failed",
//	})
//
// # Logger Manager
//
// For managing multiple component loggers, use LoggerManager:
//
//	manager := utils.NewLoggerManager()
//	defer manager.Close()
//
//	agentLogger, err := manager.GetLogger("agent")
//	mcpLogger, err := manager.GetLogger("mcp_services")
//	dbLogger, err := manager.GetLogger("amorphdb")
//
// # Design Philosophy
//
// The utilities in this package follow the AI Work Studio design principles:
//
//   - Simplicity over complexity: Minimal dependencies, focused functionality
//   - Judgment over rules: Configurable behavior rather than hardcoded logic
//   - Single user focus: Optimized for single-user scenarios
//   - Theory educated by practice: Learning and adapting from usage patterns
//   - Minimal context design: Efficient data passing with references
package utils