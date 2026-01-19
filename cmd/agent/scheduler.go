package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/yourusername/ai-work-studio/internal/config"
	"github.com/yourusername/ai-work-studio/pkg/core"
)

// Scheduler manages background monitoring and execution of objectives.
type Scheduler struct {
	config           SchedulerConfig
	runningObjectives sync.Map // map[string]context.CancelFunc for tracking running objectives
	mutex            sync.RWMutex
	executionCount   int
}

// SchedulerConfig defines configuration for the scheduler.
type SchedulerConfig struct {
	// CheckInterval determines how often to check for new objectives
	CheckInterval time.Duration

	// MaxConcurrentObjectives limits parallel execution
	MaxConcurrentObjectives int

	// DryRun mode simulates execution without making changes
	DryRun bool
}

// SchedulerDependencies contains all external dependencies the scheduler needs.
type SchedulerDependencies struct {
	ObjectiveManager *core.ObjectiveManager
	LearningLoop     *core.LearningLoop
	EthicalFramework *core.EthicalFramework
	ContextManager   *core.UserContextManager
	Config           *config.Config
	Logger           *ActivityLogger
}

// ExecutionContext tracks the context of a running objective.
type ExecutionContext struct {
	ObjectiveID   string
	StartTime     time.Time
	Cancel        context.CancelFunc
	DryRun        bool
}

// NewScheduler creates a new scheduler instance.
func NewScheduler(config SchedulerConfig) (*Scheduler, error) {
	if config.CheckInterval <= 0 {
		return nil, fmt.Errorf("check interval must be positive")
	}
	if config.MaxConcurrentObjectives <= 0 {
		config.MaxConcurrentObjectives = 3 // Default to 3
	}

	return &Scheduler{
		config: config,
	}, nil
}

// Start begins the scheduler's monitoring loop.
func (s *Scheduler) Start(ctx context.Context, deps *SchedulerDependencies) {
	ticker := time.NewTicker(s.config.CheckInterval)
	defer ticker.Stop()

	log.Printf("Scheduler started with %d max concurrent objectives", s.config.MaxConcurrentObjectives)

	for {
		select {
		case <-ctx.Done():
			log.Println("Scheduler stopping due to context cancellation")
			s.stopAllRunningObjectives()
			return
		case <-ticker.C:
			s.checkAndExecuteObjectives(ctx, deps)
		}
	}
}

// checkAndExecuteObjectives checks for pending objectives and executes them if appropriate.
func (s *Scheduler) checkAndExecuteObjectives(ctx context.Context, deps *SchedulerDependencies) {
	s.mutex.Lock()
	runningCount := s.getRunningObjectiveCount()
	s.mutex.Unlock()

	if runningCount >= s.config.MaxConcurrentObjectives {
		if deps.Config.Preferences.VerboseOutput {
			log.Printf("At max capacity (%d objectives running), skipping check", runningCount)
		}
		return
	}

	// Get pending objectives
	status := core.ObjectiveStatusPending
	filter := core.ObjectiveFilter{
		Status: &status,
	}

	objectives, err := deps.ObjectiveManager.ListObjectives(ctx, filter)
	if err != nil {
		log.Printf("Error listing objectives: %v", err)
		deps.Logger.LogActivity("error", map[string]interface{}{
			"error":   err.Error(),
			"context": "listing_objectives",
		})
		return
	}

	if len(objectives) == 0 && deps.Config.Preferences.VerboseOutput {
		log.Println("No pending objectives found")
		return
	}

	// Process each pending objective
	for _, objective := range objectives {
		if s.shouldExecuteObjective(ctx, objective, deps) {
			if err := s.startObjectiveExecution(ctx, objective, deps); err != nil {
				log.Printf("Error starting objective %s: %v", objective.ID, err)
				deps.Logger.LogActivity("execution_error", map[string]interface{}{
					"objective_id": objective.ID,
					"error":        err.Error(),
				})
			}
		}
	}
}

// shouldExecuteObjective determines if an objective should be executed based on
// ethical framework, user context, and system state.
func (s *Scheduler) shouldExecuteObjective(ctx context.Context, objective *core.Objective, deps *SchedulerDependencies) bool {
	// Check if already running
	if _, isRunning := s.runningObjectives.Load(objective.ID); isRunning {
		return false
	}

	// If auto-approval is disabled, require manual intervention
	if !deps.Config.Preferences.AutoApprove {
		if deps.Config.Preferences.VerboseOutput {
			log.Printf("Objective %s requires manual approval (auto-approve disabled)", objective.ID)
		}
		deps.Logger.LogActivity("approval_required", map[string]interface{}{
			"objective_id": objective.ID,
			"title":        objective.Title,
			"reason":       "auto_approve_disabled",
		})
		return false
	}

	// Evaluate ethical implications for background execution
	decision, err := deps.EthicalFramework.EvaluateDecision(ctx, objective.ID, "execute_objective", "Execute pending objective automatically", []string{"wait_for_approval", "schedule_later"}, deps.Config.Session.UserID)
	if err != nil {
		log.Printf("Error evaluating ethical decision for objective %s: %v", objective.ID, err)
		deps.Logger.LogActivity("ethical_evaluation_error", map[string]interface{}{
			"objective_id": objective.ID,
			"error":        err.Error(),
		})
		return false
	}

	// Check if execution should be interrupted
	shouldInterrupt := s.shouldInterruptExecution(decision, objective, deps)
	if shouldInterrupt {
		if deps.Config.Preferences.VerboseOutput {
			log.Printf("Objective %s execution interrupted by ethical framework", objective.ID)
		}
		deps.Logger.LogActivity("execution_interrupted", map[string]interface{}{
			"objective_id": objective.ID,
			"decision_id":  decision.ID,
			"reasoning":    decision.Impact.Reasoning,
		})
		return false
	}

	// Auto-approve if ethical evaluation passed
	if err := deps.EthicalFramework.ApproveDecision(ctx, decision.ID, "Automatically approved by background agent"); err != nil {
		log.Printf("Error approving decision for objective %s: %v", objective.ID, err)
		return false
	}

	return true
}

// shouldInterruptExecution determines if execution should be interrupted based on
// ethical impact and user context.
func (s *Scheduler) shouldInterruptExecution(decision *core.EthicalDecision, objective *core.Objective, deps *SchedulerDependencies) bool {
	impact := decision.Impact

	// Interrupt if confidence is too low
	if impact.ConfidenceScore < 0.7 {
		return true
	}

	// Interrupt if any impact dimension is negative
	if impact.FreedomImpact < 0 || impact.WellBeingImpact < 0 || impact.SustainabilityImpact < 0 {
		return true
	}

	// Interrupt if overall impact is too low
	averageImpact := (impact.FreedomImpact + impact.WellBeingImpact + impact.SustainabilityImpact) / 3.0
	if averageImpact < 0.3 {
		return true
	}

	// Check if this is a high-priority objective that might need user input
	if objective.Priority >= 8 {
		if deps.Config.Preferences.VerboseOutput {
			log.Printf("High-priority objective %s requires user attention", objective.ID)
		}
		return true
	}

	return false
}

// startObjectiveExecution begins execution of an objective in a separate goroutine.
func (s *Scheduler) startObjectiveExecution(ctx context.Context, objective *core.Objective, deps *SchedulerDependencies) error {
	// Create cancellable context for this execution
	execCtx, cancel := context.WithCancel(ctx)

	// Track the running objective
	execContext := &ExecutionContext{
		ObjectiveID: objective.ID,
		StartTime:   time.Now(),
		Cancel:      cancel,
		DryRun:      s.config.DryRun,
	}
	s.runningObjectives.Store(objective.ID, execContext)

	s.mutex.Lock()
	s.executionCount++
	execNumber := s.executionCount
	s.mutex.Unlock()

	log.Printf("Starting execution #%d of objective: %s (%s)", execNumber, objective.ID, objective.Title)

	// Log execution start
	deps.Logger.LogActivity("execution_start", map[string]interface{}{
		"objective_id":    objective.ID,
		"objective_title": objective.Title,
		"priority":        objective.Priority,
		"execution_number": execNumber,
		"dry_run":         s.config.DryRun,
	})

	// Start execution in background
	go s.executeObjective(execCtx, objective, deps, execNumber)

	return nil
}

// executeObjective performs the actual execution of an objective.
func (s *Scheduler) executeObjective(ctx context.Context, objective *core.Objective, deps *SchedulerDependencies, execNumber int) {
	defer func() {
		// Clean up tracking
		s.runningObjectives.Delete(objective.ID)
		log.Printf("Execution #%d of objective %s completed", execNumber, objective.ID)
	}()

	startTime := time.Now()

	// In dry-run mode, simulate execution
	if s.config.DryRun {
		s.simulateExecution(ctx, objective, deps, execNumber)
		return
	}

	// Execute the objective using the learning loop
	if deps.LearningLoop == nil {
		// Simulate execution for basic daemon functionality
		time.Sleep(2 * time.Second)
		log.Printf("Mock execution of objective %s completed", objective.ID)
		return
	}
	result, err := deps.LearningLoop.ExecuteObjective(ctx, objective.ID)

	executionTime := time.Since(startTime)

	if err != nil {
		log.Printf("Execution #%d of objective %s failed: %v", execNumber, objective.ID, err)
		deps.Logger.LogActivity("execution_failure", map[string]interface{}{
			"objective_id":     objective.ID,
			"execution_number": execNumber,
			"error":           err.Error(),
			"execution_time":   executionTime.String(),
		})
		return
	}

	// Log successful execution
	log.Printf("Execution #%d of objective %s succeeded in %v", execNumber, objective.ID, executionTime)
	deps.Logger.LogActivity("execution_success", map[string]interface{}{
		"objective_id":     objective.ID,
		"execution_number": execNumber,
		"execution_time":   executionTime.String(),
		"attempts":         len(result.ExecutionAttempts),
		"final_outcome":    string(result.FinalOutcome),
	})
}

// simulateExecution simulates objective execution in dry-run mode.
func (s *Scheduler) simulateExecution(ctx context.Context, objective *core.Objective, deps *SchedulerDependencies, execNumber int) {
	log.Printf("DRY-RUN: Simulating execution #%d of objective %s", execNumber, objective.ID)

	// Simulate some processing time
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Duration(2+execNumber%8) * time.Second):
		// Simulate variable execution time
	}

	deps.Logger.LogActivity("simulation_complete", map[string]interface{}{
		"objective_id":     objective.ID,
		"execution_number": execNumber,
		"simulated_time":   "2-10 seconds",
	})

	log.Printf("DRY-RUN: Simulation #%d of objective %s completed", execNumber, objective.ID)
}

// getRunningObjectiveCount returns the number of currently running objectives.
func (s *Scheduler) getRunningObjectiveCount() int {
	count := 0
	s.runningObjectives.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// stopAllRunningObjectives cancels all running objective executions.
func (s *Scheduler) stopAllRunningObjectives() {
	log.Println("Stopping all running objectives...")

	s.runningObjectives.Range(func(key, value interface{}) bool {
		if execContext, ok := value.(*ExecutionContext); ok {
			log.Printf("Stopping objective %s", execContext.ObjectiveID)
			execContext.Cancel()
		}
		return true
	})

	// Give some time for graceful shutdown
	time.Sleep(1 * time.Second)

	log.Println("All objectives stopped")
}

// GetStatus returns the current status of the scheduler.
func (s *Scheduler) GetStatus() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	runningObjectives := make([]map[string]interface{}, 0)
	s.runningObjectives.Range(func(key, value interface{}) bool {
		if execContext, ok := value.(*ExecutionContext); ok {
			runningObjectives = append(runningObjectives, map[string]interface{}{
				"objective_id": execContext.ObjectiveID,
				"start_time":   execContext.StartTime,
				"duration":     time.Since(execContext.StartTime).String(),
				"dry_run":      execContext.DryRun,
			})
		}
		return true
	})

	return map[string]interface{}{
		"check_interval":           s.config.CheckInterval.String(),
		"max_concurrent":           s.config.MaxConcurrentObjectives,
		"running_count":            len(runningObjectives),
		"running_objectives":       runningObjectives,
		"total_executions":         s.executionCount,
		"dry_run":                 s.config.DryRun,
	}
}