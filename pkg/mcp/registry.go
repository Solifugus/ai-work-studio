package mcp

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"
)

// ServiceRegistry manages a collection of MCP services and provides
// service discovery and execution capabilities.
type ServiceRegistry struct {
	services map[string]Service
	mutex    sync.RWMutex
	logger   *log.Logger
}

// ServiceInfo contains metadata about a registered service.
type ServiceInfo struct {
	Name        string
	Description string
	Registered  time.Time
}

// NewServiceRegistry creates a new service registry.
func NewServiceRegistry(logger *log.Logger) *ServiceRegistry {
	if logger == nil {
		logger = log.Default()
	}

	return &ServiceRegistry{
		services: make(map[string]Service),
		logger:   logger,
	}
}

// RegisterService adds a service to the registry.
// Returns an error if a service with the same name is already registered.
func (sr *ServiceRegistry) RegisterService(service Service) error {
	if service == nil {
		return fmt.Errorf("service cannot be nil")
	}

	name := service.Name()
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	if _, exists := sr.services[name]; exists {
		return fmt.Errorf("service '%s' is already registered", name)
	}

	sr.services[name] = service
	sr.logger.Printf("MCP Service registered: %s - %s", name, service.Description())

	return nil
}

// UnregisterService removes a service from the registry.
// Returns an error if the service is not found.
func (sr *ServiceRegistry) UnregisterService(name string) error {
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	if _, exists := sr.services[name]; !exists {
		return fmt.Errorf("service '%s' is not registered", name)
	}

	delete(sr.services, name)
	sr.logger.Printf("MCP Service unregistered: %s", name)

	return nil
}

// GetService retrieves a service by name.
// Returns the service and true if found, or nil and false if not found.
func (sr *ServiceRegistry) GetService(name string) (Service, bool) {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	service, exists := sr.services[name]
	return service, exists
}

// ListServices returns information about all registered services.
func (sr *ServiceRegistry) ListServices() []ServiceInfo {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	var services []ServiceInfo
	for name, service := range sr.services {
		services = append(services, ServiceInfo{
			Name:        name,
			Description: service.Description(),
			Registered:  time.Now(), // In a real implementation, we'd track actual registration time
		})
	}

	// Sort services by name for consistent ordering
	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	return services
}

// ServiceExists checks if a service with the given name is registered.
func (sr *ServiceRegistry) ServiceExists(name string) bool {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	_, exists := sr.services[name]
	return exists
}

// CallService executes a service by name with the given parameters.
// This is the primary interface for executing MCP services through the registry.
func (sr *ServiceRegistry) CallService(ctx context.Context, serviceName string, params ServiceParams) ServiceResult {
	// Get the service
	service, exists := sr.GetService(serviceName)
	if !exists {
		return ErrorResult(fmt.Errorf("service '%s' not found in registry", serviceName))
	}

	// Log the service call
	sr.logger.Printf("MCP Service call initiated: %s", serviceName)

	// Execute the service using the framework's CallService function
	result := CallService(ctx, service, params)

	// Add registry metadata
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	result.Metadata["registry_call"] = true
	result.Metadata["called_via_registry"] = serviceName

	return result
}

// FindServices returns services that match the given search criteria.
// It searches both service names and descriptions (case-insensitive).
func (sr *ServiceRegistry) FindServices(searchTerm string) []ServiceInfo {
	if searchTerm == "" {
		return sr.ListServices()
	}

	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	searchLower := strings.ToLower(searchTerm)
	var matchingServices []ServiceInfo

	for name, service := range sr.services {
		nameMatch := strings.Contains(strings.ToLower(name), searchLower)
		descMatch := strings.Contains(strings.ToLower(service.Description()), searchLower)

		if nameMatch || descMatch {
			matchingServices = append(matchingServices, ServiceInfo{
				Name:        name,
				Description: service.Description(),
				Registered:  time.Now(),
			})
		}
	}

	// Sort results by name for consistent ordering
	sort.Slice(matchingServices, func(i, j int) bool {
		return matchingServices[i].Name < matchingServices[j].Name
	})

	return matchingServices
}

// ValidateService checks if a service implementation is valid.
// This is useful for testing and debugging service implementations.
func (sr *ServiceRegistry) ValidateService(service Service) error {
	if service == nil {
		return fmt.Errorf("service is nil")
	}

	name := service.Name()
	if name == "" {
		return fmt.Errorf("service name is empty")
	}

	description := service.Description()
	if description == "" {
		return fmt.Errorf("service description is empty")
	}

	// Test parameter validation with nil parameters
	if err := service.ValidateParams(nil); err == nil {
		return fmt.Errorf("service should reject nil parameters")
	}

	// Test parameter validation with empty parameters
	emptyParams := make(ServiceParams)
	if err := service.ValidateParams(emptyParams); err != nil {
		// This is OK - some services might require parameters
		sr.logger.Printf("Service %s requires parameters: %v", name, err)
	}

	return nil
}

// GetServiceCount returns the number of registered services.
func (sr *ServiceRegistry) GetServiceCount() int {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()
	return len(sr.services)
}

// Clear removes all services from the registry.
// This is primarily useful for testing.
func (sr *ServiceRegistry) Clear() {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	count := len(sr.services)
	sr.services = make(map[string]Service)
	sr.logger.Printf("MCP Service registry cleared (%d services removed)", count)
}

// RegisterMultipleServices is a convenience method to register multiple services at once.
// It returns a list of errors for services that failed to register.
func (sr *ServiceRegistry) RegisterMultipleServices(services ...Service) []error {
	var errors []error

	for _, service := range services {
		if err := sr.RegisterService(service); err != nil {
			errors = append(errors, fmt.Errorf("failed to register service %s: %w",
				service.Name(), err))
		}
	}

	return errors
}

// GetServiceNames returns a sorted list of all registered service names.
func (sr *ServiceRegistry) GetServiceNames() []string {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	names := make([]string, 0, len(sr.services))
	for name := range sr.services {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// ServiceRegistryStats provides statistics about the service registry.
type ServiceRegistryStats struct {
	TotalServices int
	ServiceNames  []string
}

// GetStats returns statistics about the current state of the registry.
func (sr *ServiceRegistry) GetStats() ServiceRegistryStats {
	return ServiceRegistryStats{
		TotalServices: sr.GetServiceCount(),
		ServiceNames:  sr.GetServiceNames(),
	}
}