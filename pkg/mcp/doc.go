// Package mcp provides the Model Context Protocol service framework for AI Work Studio.
//
// This package implements the MCP (Model Context Protocol) service framework, which allows
// the Real-Time Cursor (RTC) and other system components to interact with external tools
// and capabilities through a standardized interface.
//
// Key Components:
//
//   - Service: Interface that all MCP services must implement
//   - ServiceRegistry: Manages service instances and provides discovery
//   - BaseService: Foundation for implementing MCP services with common functionality
//   - ServiceResult: Structured result format for service execution
//
// The framework follows the project's core principles:
//   - Simplicity over complexity: Minimal interface, clear contracts
//   - Judgment over rules: Flexible parameter validation and error handling
//   - Single user focus: No multi-tenancy concerns, optimized for personal AI assistant
//   - Minimal context design: Services receive only necessary parameters
//
// Example Usage:
//
//	// Create a registry
//	registry := mcp.NewServiceRegistry(logger)
//
//	// Register a service
//	service := &MyService{}
//	err := registry.RegisterService(service)
//
//	// Call a service
//	params := mcp.ServiceParams{"input": "hello"}
//	result := registry.CallService(ctx, "my-service", params)
//
// Service Implementation:
//
//	type MyService struct {
//		*mcp.BaseService
//	}
//
//	func NewMyService() *MyService {
//		base := mcp.NewBaseService("my-service", "Example service", nil)
//		return &MyService{BaseService: base}
//	}
//
//	func (s *MyService) ValidateParams(params mcp.ServiceParams) error {
//		return mcp.ValidateStringParam(params, "input", true)
//	}
//
//	func (s *MyService) Execute(ctx context.Context, params mcp.ServiceParams) mcp.ServiceResult {
//		input := params["input"].(string)
//		return mcp.SuccessResult(map[string]string{"output": "processed: " + input})
//	}
//
// The framework provides built-in support for:
//   - Parameter validation with helpful error messages
//   - Structured error handling and logging
//   - Service discovery and metadata
//   - Execution timing and monitoring
//   - Thread-safe service registry operations
//
// This enables the RTC to execute methods that require external tool capabilities
// while maintaining the system's principles of simplicity and reliability.
package mcp