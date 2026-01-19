package utils

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// RetryableFunc represents a function that can be retried.
type RetryableFunc func(ctx context.Context) error

// Operation represents a resilient operation with context.
type Operation struct {
	Name        string
	ServiceID   string
	Context     LogContext
	MaxRetries  int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	JitterMax   time.Duration
	Timeout     time.Duration
}

// ExecutionResult represents the result of executing a resilient operation.
type ExecutionResult struct {
	Success     bool
	Error       error
	Attempts    int
	Duration    time.Duration
	Recovered   bool
	Degraded    bool
}

// CircuitBreakerState represents the state of a circuit breaker.
type CircuitBreakerState int

const (
	// CircuitClosed - normal operation, requests allowed
	CircuitClosed CircuitBreakerState = iota
	// CircuitOpen - failing fast, requests not allowed
	CircuitOpen
	// CircuitHalfOpen - testing if service recovered, limited requests allowed
	CircuitHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	mu                sync.RWMutex
	serviceID         string
	state             CircuitBreakerState
	failureCount      int
	lastFailureTime   time.Time
	lastSuccessTime   time.Time
	config            CircuitBreakerConfig
	logger            Logger
	context           LogContext
}

// CircuitBreakerConfig holds configuration for circuit breaker.
type CircuitBreakerConfig struct {
	FailureThreshold   int           // Number of failures before opening
	SuccessThreshold   int           // Number of successes needed to close from half-open
	Timeout           time.Duration // Time to wait before moving to half-open
	MaxRequests       int           // Max requests allowed in half-open state
}

// Checkpoint represents a point in execution that can be resumed from.
type Checkpoint struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Operation string    `json:"operation"`
	State     map[string]interface{} `json:"state"`
	Context   LogContext `json:"context"`
}

// ResilienceManager manages resilience patterns and recovery.
type ResilienceManager struct {
	circuitBreakers map[string]*CircuitBreaker
	checkpoints     map[string]*Checkpoint
	logger          Logger
	mu              sync.RWMutex
}

// DefaultRetryConfig returns default retry configuration.
func DefaultRetryConfig() Operation {
	return Operation{
		MaxRetries: 3,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   30 * time.Second,
		JitterMax:  50 * time.Millisecond,
		Timeout:    5 * time.Minute,
	}
}

// DefaultCircuitBreakerConfig returns default circuit breaker configuration.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold:  5,
		SuccessThreshold:  3,
		Timeout:          60 * time.Second,
		MaxRequests:      10,
	}
}

// NewResilienceManager creates a new resilience manager.
func NewResilienceManager(logger Logger) *ResilienceManager {
	return &ResilienceManager{
		circuitBreakers: make(map[string]*CircuitBreaker),
		checkpoints:     make(map[string]*Checkpoint),
		logger:          logger,
	}
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(serviceID string, config CircuitBreakerConfig, logger Logger, context LogContext) *CircuitBreaker {
	return &CircuitBreaker{
		serviceID: serviceID,
		state:     CircuitClosed,
		config:    config,
		logger:    logger,
		context:   context,
	}
}

// ExecuteWithResilience executes a function with retry logic and circuit breaking.
func (rm *ResilienceManager) ExecuteWithResilience(op Operation, fn RetryableFunc) ExecutionResult {
	startTime := time.Now()

	// Get or create circuit breaker for this service
	circuitBreaker := rm.getOrCreateCircuitBreaker(op.ServiceID, op.Context)

	// Check circuit breaker state
	if !circuitBreaker.AllowRequest() {
		rm.logger.Warning(op.Context, "Circuit breaker open, request rejected", map[string]interface{}{
			"service_id": op.ServiceID,
			"operation":  op.Name,
		})

		return ExecutionResult{
			Success:  false,
			Error:    errors.New("circuit breaker open"),
			Attempts: 0,
			Duration: time.Since(startTime),
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), op.Timeout)
	defer cancel()

	var lastError error
	attempts := 0

	for attempts <= op.MaxRetries {
		attempts++

		// Execute the function
		err := fn(ctx)

		if err == nil {
			// Success - record with circuit breaker
			circuitBreaker.RecordSuccess()

			rm.logger.Debug(op.Context, "Operation succeeded", map[string]interface{}{
				"service_id": op.ServiceID,
				"operation":  op.Name,
				"attempts":   attempts,
				"duration":   time.Since(startTime),
			})

			return ExecutionResult{
				Success:  true,
				Error:    nil,
				Attempts: attempts,
				Duration: time.Since(startTime),
			}
		}

		lastError = err

		// Check if error is retryable
		if !IsRetryableError(err) {
			rm.logger.Warning(op.Context, "Non-retryable error encountered", map[string]interface{}{
				"service_id": op.ServiceID,
				"operation":  op.Name,
				"error":      err.Error(),
				"attempts":   attempts,
			})
			break
		}

		// Don't retry on the last attempt
		if attempts > op.MaxRetries {
			break
		}

		// Calculate delay with exponential backoff and jitter
		delay := calculateDelay(attempts, op.BaseDelay, op.MaxDelay, op.JitterMax)

		rm.logger.Debug(op.Context, "Retrying operation after delay", map[string]interface{}{
			"service_id": op.ServiceID,
			"operation":  op.Name,
			"attempt":    attempts,
			"error":      err.Error(),
			"delay_ms":   delay.Milliseconds(),
		})

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			lastError = ctx.Err()
			break
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// Record failure with circuit breaker
	circuitBreaker.RecordFailure()

	rm.logger.Error(op.Context, "Operation failed after all retries", map[string]interface{}{
		"service_id": op.ServiceID,
		"operation":  op.Name,
		"attempts":   attempts,
		"error":      lastError.Error(),
		"duration":   time.Since(startTime),
	})

	return ExecutionResult{
		Success:  false,
		Error:    lastError,
		Attempts: attempts,
		Duration: time.Since(startTime),
	}
}

// AllowRequest checks if a request should be allowed based on circuit breaker state.
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		return cb.shouldAttemptReset()
	case CircuitHalfOpen:
		return true // Allow request in half-open state
	default:
		return false
	}
}

// RecordSuccess records a successful request.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastSuccessTime = time.Now()

	switch cb.state {
	case CircuitClosed:
		cb.failureCount = 0
	case CircuitHalfOpen:
		cb.failureCount = 0
		if cb.shouldClose() {
			cb.state = CircuitClosed
			cb.logger.Info(cb.context, "Circuit breaker closed", map[string]interface{}{
				"service_id": cb.serviceID,
			})
		}
	}
}

// RecordFailure records a failed request.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitClosed:
		if cb.failureCount >= cb.config.FailureThreshold {
			cb.state = CircuitOpen
			cb.logger.Warning(cb.context, "Circuit breaker opened", map[string]interface{}{
				"service_id":     cb.serviceID,
				"failure_count":  cb.failureCount,
				"threshold":      cb.config.FailureThreshold,
			})
		}
	case CircuitHalfOpen:
		cb.state = CircuitOpen
		cb.logger.Warning(cb.context, "Circuit breaker opened from half-open", map[string]interface{}{
			"service_id": cb.serviceID,
		})
	}
}

// CreateCheckpoint creates a checkpoint for state recovery.
func (rm *ResilienceManager) CreateCheckpoint(id, operation string, state map[string]interface{}, context LogContext) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	checkpoint := &Checkpoint{
		ID:        id,
		Timestamp: time.Now(),
		Operation: operation,
		State:     state,
		Context:   context,
	}

	rm.checkpoints[id] = checkpoint

	rm.logger.Debug(context, "Checkpoint created", map[string]interface{}{
		"checkpoint_id": id,
		"operation":     operation,
	})
}

// RestoreFromCheckpoint restores state from a checkpoint.
func (rm *ResilienceManager) RestoreFromCheckpoint(id string) (*Checkpoint, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	checkpoint, exists := rm.checkpoints[id]
	if !exists {
		return nil, fmt.Errorf("checkpoint %s not found", id)
	}

	rm.logger.Info(checkpoint.Context, "Restoring from checkpoint", map[string]interface{}{
		"checkpoint_id": id,
		"operation":     checkpoint.Operation,
		"age_seconds":   time.Since(checkpoint.Timestamp).Seconds(),
	})

	return checkpoint, nil
}

// IsRetryableError determines if an error should be retried.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for common retryable error types
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return true
	case errors.Is(err, context.Canceled):
		return false // Don't retry cancelled operations
	default:
		// Check error message for common retryable patterns
		errMsg := err.Error()
		retryablePatterns := []string{
			"connection refused",
			"timeout",
			"temporary failure",
			"service unavailable",
			"too many requests",
			"rate limit",
			"network error",
		}

		for _, pattern := range retryablePatterns {
			if len(errMsg) > 0 && errMsg == pattern {
				return true
			}
		}
	}

	return false
}

// calculateDelay calculates retry delay with exponential backoff and jitter.
func calculateDelay(attempt int, baseDelay, maxDelay, jitterMax time.Duration) time.Duration {
	// Exponential backoff: baseDelay * 2^(attempt-1)
	backoff := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt-1)))

	// Cap at maxDelay
	if backoff > maxDelay {
		backoff = maxDelay
	}

	// Add jitter to prevent thundering herd
	jitter := time.Duration(rand.Int63n(int64(jitterMax)))

	return backoff + jitter
}

// getOrCreateCircuitBreaker gets an existing circuit breaker or creates a new one.
func (rm *ResilienceManager) getOrCreateCircuitBreaker(serviceID string, context LogContext) *CircuitBreaker {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if cb, exists := rm.circuitBreakers[serviceID]; exists {
		return cb
	}

	config := DefaultCircuitBreakerConfig()
	cb := NewCircuitBreaker(serviceID, config, rm.logger, context)
	rm.circuitBreakers[serviceID] = cb

	rm.logger.Debug(context, "Created new circuit breaker", map[string]interface{}{
		"service_id": serviceID,
	})

	return cb
}

// shouldAttemptReset checks if circuit breaker should move to half-open.
func (cb *CircuitBreaker) shouldAttemptReset() bool {
	return time.Since(cb.lastFailureTime) >= cb.config.Timeout
}

// shouldClose checks if circuit breaker should close from half-open.
func (cb *CircuitBreaker) shouldClose() bool {
	// Simple heuristic: if we haven't had recent failures, close
	return cb.failureCount == 0 && time.Since(cb.lastSuccessTime) < time.Minute
}

// GetCircuitBreakerState returns the current state of a circuit breaker.
func (rm *ResilienceManager) GetCircuitBreakerState(serviceID string) (CircuitBreakerState, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if cb, exists := rm.circuitBreakers[serviceID]; exists {
		cb.mu.RLock()
		state := cb.state
		cb.mu.RUnlock()
		return state, true
	}

	return CircuitClosed, false
}

// ClearCheckpoint removes a checkpoint after successful completion.
func (rm *ResilienceManager) ClearCheckpoint(id string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	delete(rm.checkpoints, id)
}