# AI Work Studio - Code Tutorials

This directory contains comprehensive code examples demonstrating how to use the AI Work Studio API. Each tutorial is a complete, runnable Go program that showcases different aspects of the system.

## Available Tutorials

### 1. Goals Management (`goals_example.go`)
Learn how to create, manage, and organize goals using the core package:
- Creating main goals and sub-goals
- Updating goal status and properties
- Querying goals by status and criteria
- Understanding goal hierarchies
- Working with temporal data

**Key Concepts**: Goal lifecycle, hierarchical relationships, temporal queries

### 2. MCP Service Framework (`mcp_service_example.go`)
Understand how to create and use MCP (Model Context Protocol) services:
- Implementing custom services
- Service registration and discovery
- Parameter validation and error handling
- Service chaining and composition
- Performance monitoring

**Key Concepts**: Service interfaces, parameter validation, structured results

### 3. Storage System (`storage_example.go`)
Master the temporal storage system with nodes and edges:
- Creating and storing temporal nodes
- Establishing relationships with edges
- Point-in-time queries
- Version history management
- Graph traversal and analysis

**Key Concepts**: Temporal versioning, graph relationships, data integrity

### 4. User Interface (`ui_example.go`)
Build custom UI components using Fyne:
- Creating custom widgets
- Tabbed interfaces and layouts
- Menus, toolbars, and dialogs
- Data binding and updates
- Responsive design patterns

**Key Concepts**: Widget composition, event handling, responsive layouts

## Running the Tutorials

Each tutorial can be run independently. To execute a tutorial:

```bash
# Make sure you're in the project root
cd /path/to/ai-work-studio

# Run a specific tutorial
go run docs/tutorials/goals_example.go
go run docs/tutorials/mcp_service_example.go
go run docs/tutorials/storage_example.go
go run docs/tutorials/ui_example.go
```

**Note**: The UI example will open a GUI window. Close the window to complete the tutorial.

## Integration with Main System

These tutorials demonstrate the same APIs used by the main AI Work Studio application. You can:

1. **Copy patterns**: Use these patterns in your own extensions
2. **Extend examples**: Build upon these examples for your specific needs
3. **Test changes**: Modify tutorials to test API changes before implementing

## Prerequisites

- Go 1.21 or later
- Fyne dependencies (for UI examples):
  ```bash
  go mod tidy
  ```

## Tutorial Structure

Each tutorial follows a consistent structure:

1. **Setup**: Initialize storage, logging, and dependencies
2. **Basic Operations**: Demonstrate core functionality
3. **Advanced Features**: Show sophisticated usage patterns
4. **Best Practices**: Highlight recommended approaches
5. **Cleanup**: Proper resource management

## Error Handling

Tutorials demonstrate proper error handling patterns:
- Graceful degradation when optional features fail
- Clear error messages for debugging
- Resource cleanup in error scenarios
- Logging for troubleshooting

## Extension Points

Each tutorial includes comments about extension opportunities:
- Performance optimizations
- Additional features
- Integration patterns
- Customization options

## Common Patterns

Across all tutorials, you'll see these consistent patterns:

### Resource Management
```go
// Always clean up resources
defer logger.Close()
defer store.Close()
```

### Error Handling
```go
if err != nil {
    return fmt.Errorf("descriptive error: %w", err)
}
```

### Context Usage
```go
ctx := context.Background()
// Pass context to all operations
result, err := service.Execute(ctx, params)
```

### Structured Data
```go
data := map[string]interface{}{
    "field": "value",
    // Use interface{} for flexible data structures
}
```

## Related Documentation

- [API Overview](../api/overview.md) - Complete API reference
- [Installation Guide](../installation.md) - Setup instructions
- Package Documentation (via `godoc`) - Detailed function references

## Contributing

To add new tutorials:

1. Follow the existing naming convention: `topic_example.go`
2. Include comprehensive comments
3. Add error handling and resource cleanup
4. Update this README with the new tutorial
5. Ensure the tutorial is self-contained and runnable

## Best Practices for Tutorial Development

- **Start Simple**: Begin with basic functionality before advanced features
- **Real-World Examples**: Use realistic data and scenarios
- **Progressive Complexity**: Build complexity gradually
- **Error Scenarios**: Show what happens when things go wrong
- **Performance Notes**: Mention performance considerations where relevant
- **Testing**: Include validation that operations succeeded

## Troubleshooting

Common issues when running tutorials:

1. **Import Errors**: Run `go mod tidy` to ensure dependencies are available
2. **Permission Errors**: Ensure write access to temp directories used by examples
3. **UI Issues**: For Fyne tutorials, ensure display is available (not SSH without X11)
4. **Memory Issues**: Examples create temporary data - ensure sufficient memory

## Getting Help

If you encounter issues with tutorials:

1. Check the console output for specific error messages
2. Verify all dependencies are installed (`go mod tidy`)
3. Review the [API documentation](../api/overview.md) for function details
4. Check the main project's issue tracker for known problems
5. Look at the test files (`*_test.go`) for additional usage examples