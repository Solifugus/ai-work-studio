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

// MockService is a simple test service for registry testing
type MockService struct {
	*BaseService
	shouldFail bool
}

func NewMockService(name, description string, shouldFail bool) *MockService {
	return &MockService{
		BaseService: NewBaseService(name, description, log.New(os.Stdout, "", 0)),
		shouldFail:  shouldFail,
	}
}

func (ms *MockService) ValidateParams(params ServiceParams) error {
	if err := ms.BaseService.ValidateParams(params); err != nil {
		return err
	}
	if ms.shouldFail {
		return NewValidationError("test", "mock validation failure")
	}
	return nil
}

func (ms *MockService) Execute(ctx context.Context, params ServiceParams) ServiceResult {
	if ms.shouldFail {
		return ErrorResult(fmt.Errorf("mock execution failure"))
	}
	return SuccessResult(map[string]string{"service": ms.Name(), "status": "executed"})
}

func TestNewServiceRegistry(t *testing.T) {
	t.Run("WithLogger", func(t *testing.T) {
		logger := log.New(os.Stdout, "", 0)
		registry := NewServiceRegistry(logger)
		if registry == nil {
			t.Error("expected non-nil registry")
		}
		if registry.services == nil {
			t.Error("expected services map to be initialized")
		}
		if registry.logger != logger {
			t.Error("expected logger to be set")
		}
	})

	t.Run("WithNilLogger", func(t *testing.T) {
		registry := NewServiceRegistry(nil)
		if registry == nil {
			t.Error("expected non-nil registry")
		}
		if registry.logger == nil {
			t.Error("expected default logger to be set")
		}
	})
}

func TestServiceRegistry_RegisterService(t *testing.T) {
	registry := NewServiceRegistry(nil)

	t.Run("Success", func(t *testing.T) {
		service := NewMockService("test-service", "Test service", false)
		err := registry.RegisterService(service)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !registry.ServiceExists("test-service") {
			t.Error("service should exist after registration")
		}
	})

	t.Run("NilService", func(t *testing.T) {
		err := registry.RegisterService(nil)
		if err == nil {
			t.Error("expected error for nil service")
		}
	})

	t.Run("EmptyName", func(t *testing.T) {
		service := NewMockService("", "Empty name service", false)
		err := registry.RegisterService(service)
		if err == nil {
			t.Error("expected error for empty service name")
		}
	})

	t.Run("DuplicateName", func(t *testing.T) {
		service1 := NewMockService("duplicate", "First service", false)
		service2 := NewMockService("duplicate", "Second service", false)

		err := registry.RegisterService(service1)
		if err != nil {
			t.Errorf("unexpected error registering first service: %v", err)
		}

		err = registry.RegisterService(service2)
		if err == nil {
			t.Error("expected error for duplicate service name")
		}
	})
}

func TestServiceRegistry_UnregisterService(t *testing.T) {
	registry := NewServiceRegistry(nil)
	service := NewMockService("test-service", "Test service", false)
	registry.RegisterService(service)

	t.Run("Success", func(t *testing.T) {
		err := registry.UnregisterService("test-service")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if registry.ServiceExists("test-service") {
			t.Error("service should not exist after unregistration")
		}
	})

	t.Run("EmptyName", func(t *testing.T) {
		err := registry.UnregisterService("")
		if err == nil {
			t.Error("expected error for empty service name")
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		err := registry.UnregisterService("non-existent")
		if err == nil {
			t.Error("expected error for non-existent service")
		}
	})
}

func TestServiceRegistry_GetService(t *testing.T) {
	registry := NewServiceRegistry(nil)
	service := NewMockService("test-service", "Test service", false)
	registry.RegisterService(service)

	t.Run("Found", func(t *testing.T) {
		retrieved, found := registry.GetService("test-service")
		if !found {
			t.Error("expected service to be found")
		}
		if retrieved.Name() != "test-service" {
			t.Errorf("expected service name 'test-service', got %s", retrieved.Name())
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		retrieved, found := registry.GetService("non-existent")
		if found {
			t.Error("expected service not to be found")
		}
		if retrieved != nil {
			t.Error("expected nil service for not found")
		}
	})
}

func TestServiceRegistry_ListServices(t *testing.T) {
	registry := NewServiceRegistry(nil)

	t.Run("Empty", func(t *testing.T) {
		services := registry.ListServices()
		if len(services) != 0 {
			t.Errorf("expected 0 services, got %d", len(services))
		}
	})

	t.Run("WithServices", func(t *testing.T) {
		service1 := NewMockService("service-a", "Service A", false)
		service2 := NewMockService("service-b", "Service B", false)
		registry.RegisterService(service1)
		registry.RegisterService(service2)

		services := registry.ListServices()
		if len(services) != 2 {
			t.Errorf("expected 2 services, got %d", len(services))
		}

		// Check that services are sorted by name
		if services[0].Name != "service-a" || services[1].Name != "service-b" {
			t.Error("services should be sorted by name")
		}
	})
}

func TestServiceRegistry_CallService(t *testing.T) {
	registry := NewServiceRegistry(nil)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		service := NewMockService("test-service", "Test service", false)
		registry.RegisterService(service)

		params := ServiceParams{}
		result := registry.CallService(ctx, "test-service", params)

		if !result.Success {
			t.Errorf("expected success, got error: %v", result.Error)
		}
		if result.Metadata["registry_call"] != true {
			t.Error("expected registry_call metadata to be true")
		}
	})

	t.Run("ServiceNotFound", func(t *testing.T) {
		params := ServiceParams{}
		result := registry.CallService(ctx, "non-existent", params)

		if result.Success {
			t.Error("expected failure for non-existent service")
		}
		if result.Error == nil {
			t.Error("expected error for non-existent service")
		}
	})

	t.Run("ServiceExecutionError", func(t *testing.T) {
		service := NewMockService("failing-service", "Failing service", true)
		registry.RegisterService(service)

		params := ServiceParams{}
		result := registry.CallService(ctx, "failing-service", params)

		if result.Success {
			t.Error("expected failure for failing service")
		}
	})
}

func TestServiceRegistry_FindServices(t *testing.T) {
	registry := NewServiceRegistry(nil)

	service1 := NewMockService("file-reader", "Reads files from disk", false)
	service2 := NewMockService("data-processor", "Processes data files", false)
	service3 := NewMockService("api-client", "Makes API calls", false)

	registry.RegisterService(service1)
	registry.RegisterService(service2)
	registry.RegisterService(service3)

	t.Run("EmptySearch", func(t *testing.T) {
		results := registry.FindServices("")
		if len(results) != 3 {
			t.Errorf("expected 3 results for empty search, got %d", len(results))
		}
	})

	t.Run("NameMatch", func(t *testing.T) {
		results := registry.FindServices("file-reader")
		if len(results) != 1 {
			t.Errorf("expected 1 result for 'file-reader' search, got %d", len(results))
		}
		if results[0].Name != "file-reader" {
			t.Errorf("expected 'file-reader', got %s", results[0].Name)
		}
	})

	t.Run("DescriptionMatch", func(t *testing.T) {
		results := registry.FindServices("files")
		if len(results) != 2 {
			t.Errorf("expected 2 results for 'files' search, got %d", len(results))
		}
	})

	t.Run("CaseInsensitive", func(t *testing.T) {
		results := registry.FindServices("API")
		if len(results) != 1 {
			t.Errorf("expected 1 result for 'API' search, got %d", len(results))
		}
	})

	t.Run("NoMatch", func(t *testing.T) {
		results := registry.FindServices("xyz")
		if len(results) != 0 {
			t.Errorf("expected 0 results for 'xyz' search, got %d", len(results))
		}
	})
}

func TestServiceRegistry_ValidateService(t *testing.T) {
	registry := NewServiceRegistry(nil)

	t.Run("ValidService", func(t *testing.T) {
		service := NewMockService("test", "Test service", false)
		err := registry.ValidateService(service)
		if err != nil {
			t.Errorf("unexpected error for valid service: %v", err)
		}
	})

	t.Run("NilService", func(t *testing.T) {
		err := registry.ValidateService(nil)
		if err == nil {
			t.Error("expected error for nil service")
		}
	})

	t.Run("EmptyName", func(t *testing.T) {
		service := NewMockService("", "Test service", false)
		err := registry.ValidateService(service)
		if err == nil {
			t.Error("expected error for empty name")
		}
	})

	t.Run("EmptyDescription", func(t *testing.T) {
		service := NewMockService("test", "", false)
		err := registry.ValidateService(service)
		if err == nil {
			t.Error("expected error for empty description")
		}
	})
}

func TestServiceRegistry_GetServiceCount(t *testing.T) {
	registry := NewServiceRegistry(nil)

	if registry.GetServiceCount() != 0 {
		t.Error("expected 0 services initially")
	}

	service1 := NewMockService("service1", "Service 1", false)
	service2 := NewMockService("service2", "Service 2", false)

	registry.RegisterService(service1)
	if registry.GetServiceCount() != 1 {
		t.Error("expected 1 service after first registration")
	}

	registry.RegisterService(service2)
	if registry.GetServiceCount() != 2 {
		t.Error("expected 2 services after second registration")
	}

	registry.UnregisterService("service1")
	if registry.GetServiceCount() != 1 {
		t.Error("expected 1 service after unregistration")
	}
}

func TestServiceRegistry_Clear(t *testing.T) {
	registry := NewServiceRegistry(nil)

	service1 := NewMockService("service1", "Service 1", false)
	service2 := NewMockService("service2", "Service 2", false)

	registry.RegisterService(service1)
	registry.RegisterService(service2)

	if registry.GetServiceCount() != 2 {
		t.Error("expected 2 services before clear")
	}

	registry.Clear()

	if registry.GetServiceCount() != 0 {
		t.Error("expected 0 services after clear")
	}

	if registry.ServiceExists("service1") || registry.ServiceExists("service2") {
		t.Error("services should not exist after clear")
	}
}

func TestServiceRegistry_RegisterMultipleServices(t *testing.T) {
	registry := NewServiceRegistry(nil)

	service1 := NewMockService("service1", "Service 1", false)
	service2 := NewMockService("service2", "Service 2", false)
	service3 := NewMockService("service1", "Duplicate name", false) // Duplicate name

	errors := registry.RegisterMultipleServices(service1, service2, service3)

	if len(errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(errors))
	}

	if registry.GetServiceCount() != 2 {
		t.Errorf("expected 2 services registered, got %d", registry.GetServiceCount())
	}

	if !strings.Contains(errors[0].Error(), "service1") {
		t.Errorf("error should mention duplicate service name: %v", errors[0])
	}
}

func TestServiceRegistry_GetServiceNames(t *testing.T) {
	registry := NewServiceRegistry(nil)

	t.Run("Empty", func(t *testing.T) {
		names := registry.GetServiceNames()
		if len(names) != 0 {
			t.Errorf("expected 0 names, got %d", len(names))
		}
	})

	t.Run("WithServices", func(t *testing.T) {
		service1 := NewMockService("zebra", "Service Z", false)
		service2 := NewMockService("alpha", "Service A", false)
		service3 := NewMockService("beta", "Service B", false)

		registry.RegisterService(service1)
		registry.RegisterService(service2)
		registry.RegisterService(service3)

		names := registry.GetServiceNames()
		if len(names) != 3 {
			t.Errorf("expected 3 names, got %d", len(names))
		}

		// Check that names are sorted
		expected := []string{"alpha", "beta", "zebra"}
		for i, name := range names {
			if name != expected[i] {
				t.Errorf("expected name %s at index %d, got %s", expected[i], i, name)
			}
		}
	})
}

func TestServiceRegistry_GetStats(t *testing.T) {
	registry := NewServiceRegistry(nil)

	service1 := NewMockService("service1", "Service 1", false)
	service2 := NewMockService("service2", "Service 2", false)

	registry.RegisterService(service1)
	registry.RegisterService(service2)

	stats := registry.GetStats()

	if stats.TotalServices != 2 {
		t.Errorf("expected 2 total services, got %d", stats.TotalServices)
	}

	if len(stats.ServiceNames) != 2 {
		t.Errorf("expected 2 service names, got %d", len(stats.ServiceNames))
	}

	// Names should be sorted
	if stats.ServiceNames[0] != "service1" || stats.ServiceNames[1] != "service2" {
		t.Errorf("service names not sorted correctly: %v", stats.ServiceNames)
	}
}

func TestServiceRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewServiceRegistry(nil)

	// Test concurrent registration and retrieval
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 10; i++ {
			service := NewMockService(
				fmt.Sprintf("service-%d", i),
				fmt.Sprintf("Service %d", i),
				false,
			)
			registry.RegisterService(service)
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			registry.GetService(fmt.Sprintf("service-%d", i))
			registry.ServiceExists(fmt.Sprintf("service-%d", i))
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Wait for both goroutines to finish
	<-done
	<-done

	// Verify final state
	if registry.GetServiceCount() != 10 {
		t.Errorf("expected 10 services after concurrent operations, got %d", registry.GetServiceCount())
	}
}