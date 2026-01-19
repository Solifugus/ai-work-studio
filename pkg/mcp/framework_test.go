package mcp

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

// TestService is a concrete implementation for testing purposes
type TestService struct {
	*BaseService
	executeFunc func(ctx context.Context, params ServiceParams) ServiceResult
}

func NewTestService(name, description string, executeFunc func(ctx context.Context, params ServiceParams) ServiceResult) *TestService {
	return &TestService{
		BaseService: NewBaseService(name, description, log.New(os.Stdout, "", 0)),
		executeFunc: executeFunc,
	}
}

func (ts *TestService) Execute(ctx context.Context, params ServiceParams) ServiceResult {
	if ts.executeFunc != nil {
		return ts.executeFunc(ctx, params)
	}
	return SuccessResult("test executed")
}

func (ts *TestService) ValidateParams(params ServiceParams) error {
	if err := ts.BaseService.ValidateParams(params); err != nil {
		return err
	}
	// Test service requires "input" parameter
	return ValidateStringParam(params, "input", true)
}

func TestBaseService(t *testing.T) {
	logger := log.New(os.Stdout, "", 0)
	service := NewBaseService("test-service", "A test service", logger)

	t.Run("Name", func(t *testing.T) {
		if service.Name() != "test-service" {
			t.Errorf("expected name 'test-service', got %s", service.Name())
		}
	})

	t.Run("Description", func(t *testing.T) {
		if service.Description() != "A test service" {
			t.Errorf("expected description 'A test service', got %s", service.Description())
		}
	})

	t.Run("ValidateParams_Nil", func(t *testing.T) {
		err := service.ValidateParams(nil)
		if err == nil {
			t.Error("expected error for nil parameters")
		}
	})

	t.Run("ValidateParams_Empty", func(t *testing.T) {
		err := service.ValidateParams(ServiceParams{})
		if err != nil {
			t.Errorf("unexpected error for empty parameters: %v", err)
		}
	})

	t.Run("Execute_NotImplemented", func(t *testing.T) {
		result := service.Execute(context.Background(), ServiceParams{})
		if result.Success {
			t.Error("expected failure for unimplemented execute")
		}
		if result.Error == nil {
			t.Error("expected error for unimplemented execute")
		}
	})
}

func TestTestService(t *testing.T) {
	service := NewTestService("test", "Test service", nil)
	ctx := context.Background()

	t.Run("ValidateParams_Valid", func(t *testing.T) {
		params := ServiceParams{"input": "test"}
		err := service.ValidateParams(params)
		if err != nil {
			t.Errorf("unexpected validation error: %v", err)
		}
	})

	t.Run("ValidateParams_MissingRequired", func(t *testing.T) {
		params := ServiceParams{}
		err := service.ValidateParams(params)
		if err == nil {
			t.Error("expected validation error for missing required parameter")
		}
		if !strings.Contains(err.Error(), "input") {
			t.Errorf("error should mention 'input' parameter: %v", err)
		}
	})

	t.Run("Execute_Success", func(t *testing.T) {
		params := ServiceParams{"input": "test"}
		result := service.Execute(ctx, params)
		if !result.Success {
			t.Errorf("expected success, got error: %v", result.Error)
		}
	})
}

func TestCallService(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		service := NewTestService("test", "Test service", func(ctx context.Context, params ServiceParams) ServiceResult {
			return SuccessResult("success")
		})

		params := ServiceParams{"input": "test"}
		result := CallService(ctx, service, params)

		if !result.Success {
			t.Errorf("expected success, got error: %v", result.Error)
		}
		if result.Data != "success" {
			t.Errorf("expected data 'success', got %v", result.Data)
		}
		if result.Metadata == nil {
			t.Error("expected metadata to be set")
		}
		if result.Metadata["service_name"] != "test" {
			t.Errorf("expected service_name 'test', got %v", result.Metadata["service_name"])
		}
	})

	t.Run("ValidationError", func(t *testing.T) {
		service := NewTestService("test", "Test service", nil)

		params := ServiceParams{} // Missing required "input"
		result := CallService(ctx, service, params)

		if result.Success {
			t.Error("expected failure for invalid parameters")
		}
		if result.Error == nil {
			t.Error("expected error for invalid parameters")
		}
		if !strings.Contains(result.Error.Error(), "validation") {
			t.Errorf("error should mention validation: %v", result.Error)
		}
	})

	t.Run("ExecutionError", func(t *testing.T) {
		service := NewTestService("test", "Test service", func(ctx context.Context, params ServiceParams) ServiceResult {
			return ErrorResult(fmt.Errorf("execution failed"))
		})

		params := ServiceParams{"input": "test"}
		result := CallService(ctx, service, params)

		if result.Success {
			t.Error("expected failure for execution error")
		}
		if result.Error == nil {
			t.Error("expected error for execution failure")
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		service := NewTestService("test", "Test service", func(ctx context.Context, params ServiceParams) ServiceResult {
			// Simulate long-running operation
			select {
			case <-time.After(50 * time.Millisecond):
				return SuccessResult("completed")
			case <-ctx.Done():
				return ErrorResult(ctx.Err())
			}
		})

		params := ServiceParams{"input": "test"}
		result := CallService(ctx, service, params)

		if result.Success {
			t.Error("expected failure due to timeout")
		}
		if result.Error == nil {
			t.Error("expected timeout error")
		}
	})
}

func TestSuccessResult(t *testing.T) {
	data := "test data"
	result := SuccessResult(data)

	if !result.Success {
		t.Error("expected success to be true")
	}
	if result.Data != data {
		t.Errorf("expected data %v, got %v", data, result.Data)
	}
	if result.Error != nil {
		t.Errorf("expected error to be nil, got %v", result.Error)
	}
	if result.Metadata == nil {
		t.Error("expected metadata to be initialized")
	}
}

func TestErrorResult(t *testing.T) {
	err := fmt.Errorf("execution failed")
	result := ErrorResult(err)

	if result.Success {
		t.Error("expected success to be false")
	}
	if result.Data != nil {
		t.Errorf("expected data to be nil, got %v", result.Data)
	}
	if result.Error != err {
		t.Errorf("expected error %v, got %v", err, result.Error)
	}
	if result.Metadata == nil {
		t.Error("expected metadata to be initialized")
	}
}

func TestValidationError(t *testing.T) {
	ve := NewValidationError("test_param", "test message")

	expectedMsg := "validation error for parameter 'test_param': test message"
	if ve.Error() != expectedMsg {
		t.Errorf("expected error message '%s', got '%s'", expectedMsg, ve.Error())
	}

	if ve.Parameter != "test_param" {
		t.Errorf("expected parameter 'test_param', got '%s'", ve.Parameter)
	}

	if ve.Message != "test message" {
		t.Errorf("expected message 'test message', got '%s'", ve.Message)
	}
}

func TestValidateRequiredParam(t *testing.T) {
	t.Run("Present", func(t *testing.T) {
		params := ServiceParams{"key": "value"}
		err := ValidateRequiredParam(params, "key")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Missing", func(t *testing.T) {
		params := ServiceParams{}
		err := ValidateRequiredParam(params, "key")
		if err == nil {
			t.Error("expected error for missing parameter")
		}
		ve, ok := err.(ValidationError)
		if !ok {
			t.Errorf("expected ValidationError, got %T", err)
		}
		if ve.Parameter != "key" {
			t.Errorf("expected parameter 'key', got '%s'", ve.Parameter)
		}
	})

	t.Run("Nil", func(t *testing.T) {
		params := ServiceParams{"key": nil}
		err := ValidateRequiredParam(params, "key")
		if err == nil {
			t.Error("expected error for nil parameter")
		}
	})
}

func TestValidateStringParam(t *testing.T) {
	t.Run("ValidRequired", func(t *testing.T) {
		params := ServiceParams{"key": "value"}
		err := ValidateStringParam(params, "key", true)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("ValidOptional", func(t *testing.T) {
		params := ServiceParams{"key": "value"}
		err := ValidateStringParam(params, "key", false)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("OptionalMissing", func(t *testing.T) {
		params := ServiceParams{}
		err := ValidateStringParam(params, "key", false)
		if err != nil {
			t.Errorf("unexpected error for missing optional parameter: %v", err)
		}
	})

	t.Run("RequiredMissing", func(t *testing.T) {
		params := ServiceParams{}
		err := ValidateStringParam(params, "key", true)
		if err == nil {
			t.Error("expected error for missing required parameter")
		}
	})

	t.Run("EmptyString", func(t *testing.T) {
		params := ServiceParams{"key": ""}
		err := ValidateStringParam(params, "key", true)
		if err == nil {
			t.Error("expected error for empty string")
		}
	})

	t.Run("WrongType", func(t *testing.T) {
		params := ServiceParams{"key": 123}
		err := ValidateStringParam(params, "key", true)
		if err == nil {
			t.Error("expected error for wrong type")
		}
	})
}

func TestValidateIntParam(t *testing.T) {
	min := 1
	max := 10

	t.Run("ValidRequired", func(t *testing.T) {
		params := ServiceParams{"key": 5}
		err := ValidateIntParam(params, "key", true, &min, &max)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("ValidFloat", func(t *testing.T) {
		params := ServiceParams{"key": 5.0}
		err := ValidateIntParam(params, "key", true, &min, &max)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("OptionalMissing", func(t *testing.T) {
		params := ServiceParams{}
		err := ValidateIntParam(params, "key", false, &min, &max)
		if err != nil {
			t.Errorf("unexpected error for missing optional parameter: %v", err)
		}
	})

	t.Run("TooSmall", func(t *testing.T) {
		params := ServiceParams{"key": 0}
		err := ValidateIntParam(params, "key", true, &min, &max)
		if err == nil {
			t.Error("expected error for value below minimum")
		}
	})

	t.Run("TooLarge", func(t *testing.T) {
		params := ServiceParams{"key": 15}
		err := ValidateIntParam(params, "key", true, &min, &max)
		if err == nil {
			t.Error("expected error for value above maximum")
		}
	})

	t.Run("WrongType", func(t *testing.T) {
		params := ServiceParams{"key": "not an int"}
		err := ValidateIntParam(params, "key", true, &min, &max)
		if err == nil {
			t.Error("expected error for wrong type")
		}
	})

	t.Run("NoBounds", func(t *testing.T) {
		params := ServiceParams{"key": 100}
		err := ValidateIntParam(params, "key", true, nil, nil)
		if err != nil {
			t.Errorf("unexpected error without bounds: %v", err)
		}
	})
}