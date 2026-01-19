package mcp

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ServiceResult represents the result of an MCP service call.
// It provides structured success/failure information with optional data payload.
type ServiceResult struct {
	// Success indicates whether the service call completed successfully
	Success bool

	// Data contains the result payload when the service succeeds
	Data interface{}

	// Error contains error information when the service fails
	Error error

	// Metadata contains additional information about the execution
	Metadata map[string]interface{}
}

// ServiceParams represents parameters passed to an MCP service.
// It provides a flexible way to pass validated parameters.
type ServiceParams map[string]interface{}

// Service defines the interface that all MCP services must implement.
// This follows the project's principle of minimal interfaces with clear contracts.
type Service interface {
	// Name returns the unique name identifying this service
	Name() string

	// Description returns a human-readable description of what this service does
	Description() string

	// ValidateParams checks if the provided parameters are valid for this service.
	// Returns nil if valid, or an error describing the validation failure.
	ValidateParams(params ServiceParams) error

	// Execute performs the service operation with the given parameters.
	// The context allows for cancellation and timeouts.
	Execute(ctx context.Context, params ServiceParams) ServiceResult
}

// BaseService provides a foundation for implementing MCP services.
// It handles common functionality like logging and basic validation.
type BaseService struct {
	name        string
	description string
	logger      *log.Logger
}

// NewBaseService creates a new base service with the given name and description.
func NewBaseService(name, description string, logger *log.Logger) *BaseService {
	if logger == nil {
		logger = log.Default()
	}

	return &BaseService{
		name:        name,
		description: description,
		logger:      logger,
	}
}

// Name returns the service name.
func (bs *BaseService) Name() string {
	return bs.name
}

// Description returns the service description.
func (bs *BaseService) Description() string {
	return bs.description
}

// ValidateParams provides basic parameter validation.
// Concrete services should override this method for specific validation.
func (bs *BaseService) ValidateParams(params ServiceParams) error {
	if params == nil {
		return fmt.Errorf("parameters cannot be nil")
	}
	return nil
}

// Execute is a placeholder that concrete services must override.
func (bs *BaseService) Execute(ctx context.Context, params ServiceParams) ServiceResult {
	return ServiceResult{
		Success: false,
		Error:   fmt.Errorf("execute method not implemented for service %s", bs.name),
	}
}

// LogCall logs a service call for debugging and monitoring purposes.
func (bs *BaseService) LogCall(serviceName string, params ServiceParams, result ServiceResult, duration time.Duration) {
	status := "SUCCESS"
	if !result.Success {
		status = "ERROR"
	}

	bs.logger.Printf("MCP Service Call: %s | Status: %s | Duration: %v | Params: %v",
		serviceName, status, duration, params)

	if result.Error != nil {
		bs.logger.Printf("MCP Service Error: %s | Error: %v", serviceName, result.Error)
	}
}

// CallService executes a service with logging, timing, and error handling.
// This is a helper function that wraps service execution with common functionality.
func CallService(ctx context.Context, service Service, params ServiceParams) ServiceResult {
	start := time.Now()

	// Validate parameters
	if err := service.ValidateParams(params); err != nil {
		result := ServiceResult{
			Success: false,
			Error:   fmt.Errorf("parameter validation failed: %w", err),
			Metadata: map[string]interface{}{
				"validation_error": true,
				"service_name":     service.Name(),
				"timestamp":        start.Format(time.RFC3339),
			},
		}
		return result
	}

	// Execute the service
	result := service.Execute(ctx, params)
	duration := time.Since(start)

	// Add execution metadata
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	result.Metadata["service_name"] = service.Name()
	result.Metadata["execution_time"] = duration
	result.Metadata["timestamp"] = start.Format(time.RFC3339)

	// Log the call if service supports it
	if bs, ok := service.(*BaseService); ok {
		bs.LogCall(service.Name(), params, result, duration)
	}

	return result
}

// SuccessResult creates a successful service result with the given data.
func SuccessResult(data interface{}) ServiceResult {
	return ServiceResult{
		Success:  true,
		Data:     data,
		Metadata: make(map[string]interface{}),
	}
}

// ErrorResult creates a failed service result with the given error.
func ErrorResult(err error) ServiceResult {
	return ServiceResult{
		Success:  false,
		Error:    err,
		Metadata: make(map[string]interface{}),
	}
}

// ValidationError represents a parameter validation error.
type ValidationError struct {
	Parameter string
	Message   string
}

func (ve ValidationError) Error() string {
	return fmt.Sprintf("validation error for parameter '%s': %s", ve.Parameter, ve.Message)
}

// NewValidationError creates a new validation error for a specific parameter.
func NewValidationError(parameter, message string) ValidationError {
	return ValidationError{
		Parameter: parameter,
		Message:   message,
	}
}

// ValidateRequiredParam checks if a required parameter is present and not nil.
func ValidateRequiredParam(params ServiceParams, name string) error {
	value, exists := params[name]
	if !exists {
		return NewValidationError(name, "required parameter is missing")
	}
	if value == nil {
		return NewValidationError(name, "required parameter cannot be nil")
	}
	return nil
}

// ValidateStringParam validates that a parameter is a non-empty string.
func ValidateStringParam(params ServiceParams, name string, required bool) error {
	if !required {
		if _, exists := params[name]; !exists {
			return nil // Optional parameter not provided
		}
	}

	if err := ValidateRequiredParam(params, name); err != nil && required {
		return err
	}

	if value, exists := params[name]; exists && value != nil {
		if str, ok := value.(string); ok {
			if required && str == "" {
				return NewValidationError(name, "string parameter cannot be empty")
			}
		} else {
			return NewValidationError(name, "parameter must be a string")
		}
	}

	return nil
}

// ValidateIntParam validates that a parameter is an integer within optional bounds.
func ValidateIntParam(params ServiceParams, name string, required bool, min, max *int) error {
	if !required {
		if _, exists := params[name]; !exists {
			return nil // Optional parameter not provided
		}
	}

	if err := ValidateRequiredParam(params, name); err != nil && required {
		return err
	}

	if value, exists := params[name]; exists && value != nil {
		var intVal int
		switch v := value.(type) {
		case int:
			intVal = v
		case float64:
			intVal = int(v)
		default:
			return NewValidationError(name, "parameter must be an integer")
		}

		if min != nil && intVal < *min {
			return NewValidationError(name, fmt.Sprintf("value must be >= %d", *min))
		}
		if max != nil && intVal > *max {
			return NewValidationError(name, fmt.Sprintf("value must be <= %d", *max))
		}
	}

	return nil
}