# AI Work Studio API Documentation

This document provides a comprehensive overview of the AI Work Studio API, designed for developers who want to extend or integrate with the system.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Core Packages](#core-packages)
- [Storage System](#storage-system)
- [MCP Service Framework](#mcp-service-framework)
- [LLM Router and Budget Management](#llm-router-and-budget-management)
- [User Interface](#user-interface)
- [Utilities](#utilities)
- [Examples](#examples)

## Architecture Overview

AI Work Studio is a goal-directed autonomous agent system built around two complementary processes:

- **Contemplative Cursor (CC)**: Strategic planner that designs methods through reasoning
- **Real-Time Cursor (RTC)**: Tactical executor that tests methods through execution

The system follows five core design principles:

1. **Simplicity Over Complexity**: Minimal dependencies, general solutions
2. **Judgment Over Rules**: Contextual decisions, not rigid logic
3. **Single User Focus**: Optimized for one user with personalization through learning
4. **Theory Educated by Practice**: Methods learn from execution results
5. **Minimal Context Design**: Pass references, not full data

## Core Packages

### pkg/core

The core package provides fundamental data models and business logic:

#### Key Types

```go
// Goal represents a user objective with hierarchical relationships
type Goal struct {
    ID          string
    Title       string
    Description string
    Status      GoalStatus
    Priority    int
    UserContext map[string]interface{}
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// Method represents a proven approach for achieving objectives
type Method struct {
    ID          string
    GoalID      string
    Name        string
    Description string
    Parameters  []MethodParameter
    // ... (implementation details in source)
}

// Objective represents a specific task with minimal context references
type Objective struct {
    ID       string
    GoalID   string
    Title    string
    Context  ObjectiveContext
    Status   ObjectiveStatus
    // ... (implementation details in source)
}
```

#### Key Components

- **Contemplative Cursor**: Strategic planning component
- **Real-Time Cursor**: Execution component
- **Learning Loop**: Feedback mechanism that improves methods
- **Method Cache**: Stores and retrieves proven methods

### Storage System

The storage package provides temporal data structures with full version history.

#### Key Types

```go
// Node represents a temporal entity with version history
type Node struct {
    ID         string
    Type       string
    Data       map[string]interface{}
    CreatedAt  time.Time
    ValidFrom  time.Time
    ValidUntil time.Time
}

// Edge represents relationships between nodes with temporal metadata
type Edge struct {
    ID         string
    SourceID   string
    TargetID   string
    Type       string
    // ... (temporal metadata)
}
```

#### Features

- **Temporal Versioning**: Every modification creates a new version
- **Point-in-Time Queries**: Query data as it existed at any timestamp
- **Immutable History**: Nothing is ever deleted, only superseded
- **Migration Ready**: Designed for future AmorphDB integration

### MCP Service Framework

The mcp package provides the Model Context Protocol service framework.

#### Core Interface

```go
// Service interface for all MCP services
type Service interface {
    Name() string
    Description() string
    ValidateParams(params ServiceParams) error
    Execute(ctx context.Context, params ServiceParams) ServiceResult
}
```

#### Built-in Services

- **FileSystemService**: File operations and management
- **LLMService**: Language model interactions
- **BrowserService**: Web automation capabilities
- **CommandService**: System command execution

#### Service Registry

```go
// ServiceRegistry manages service instances and discovery
registry := mcp.NewServiceRegistry(logger)
err := registry.RegisterService(service)
result := registry.CallService(ctx, "service-name", params)
```

### LLM Router and Budget Management

The llm package provides intelligent routing and budget controls.

#### Router

```go
// Router for intelligent model selection
router := llm.NewRouter(llmService)
result, err := router.Route(ctx, llm.TaskRequest{
    Prompt: "Analyze this data...",
    TaskType: "analysis",
    QualityRequired: llm.QualityStandard,
    MaxTokens: 1000,
})
```

#### Budget Manager

```go
// BudgetManager for tracking and controlling costs
budgetManager, err := llm.NewBudgetManager("/data/budget",
    llm.DefaultBudgetConfig(), logger)

// Check affordability before execution
affordability, err := budgetManager.CanAfford(0.05)
if !affordability.Affordable {
    // Handle budget constraint
}

// Record usage after execution
transaction := llm.Transaction{
    Provider: "anthropic",
    Model: "claude-3-haiku",
    Cost: 0.05,
    // ... other fields
}
err = budgetManager.RecordUsage(ctx, transaction)
```

## User Interface

The ui package provides a Fyne-based cross-platform GUI.

### Key Components

- **App**: Application lifecycle management
- **MainWindow**: Main window with tabbed navigation
- **Views**: Goals, Objectives, Methods, Status, Settings views

### Example Usage

```go
config := config.DefaultConfig()
app := ui.NewApp(config)
app.Run() // Blocks until application closes
```

## Utilities

The utils package provides logging and common utilities.

### Structured Logging

```go
config := utils.DefaultLogConfig("my-component")
logger, err := utils.NewLogger(config)
defer logger.Close()

ctx := utils.LogContext{
    GoalID:      "goal-123",
    ObjectiveID: "obj-456",
    Component:   "my-component",
}

logger.Info(ctx, "Processing started")
logger.Error(ctx, "An error occurred", map[string]interface{}{
    "error_code": 500,
    "details":    "Connection failed",
})
```

### Logger Manager

```go
manager := utils.NewLoggerManager()
defer manager.Close()

agentLogger, err := manager.GetLogger("agent")
mcpLogger, err := manager.GetLogger("mcp_services")
```

## Examples

See the [tutorials](../tutorials/) directory for complete examples:

- [Creating and Managing Goals](../tutorials/goals_example.go)
- [Implementing Custom MCP Services](../tutorials/mcp_service_example.go)
- [Using the Storage System](../tutorials/storage_example.go)
- [Building UI Components](../tutorials/ui_example.go)

## Best Practices

### Storage

- Use temporal versioning for all entities that change over time
- Store references (IDs) rather than full objects when linking entities
- Query data at specific points in time when needed for analysis

### MCP Services

- Implement parameter validation in `ValidateParams()`
- Use structured error handling in service results
- Keep services focused on single responsibilities
- Log service execution for debugging and monitoring

### Core Components

- Design goals and objectives with minimal context
- Let methods evolve through the learning loop
- Use the method cache for performance optimization
- Implement proper error handling and recovery

### UI Development

- Follow Fyne's design patterns and widgets
- Keep views lightweight and data-driven
- Use the application's configuration for persistence
- Implement keyboard shortcuts for common actions

## Performance Considerations

- The system is optimized for single-user scenarios
- Storage operations are file-based with eventual AmorphDB migration
- LLM routing considers cost, speed, and quality trade-offs
- UI updates use efficient data binding and minimal refreshes

## Security

- All external tool access goes through the MCP framework
- User data is stored locally in the configured data directory
- No multi-tenant concerns - each installation is user-specific
- LLM interactions respect configured budget limits and providers

For detailed API reference documentation, run:
```bash
godoc -http=:8080
```
Then visit http://localhost:8080/pkg/github.com/yourusername/ai-work-studio/
