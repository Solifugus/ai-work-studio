// Package main provides the background agent daemon for AI Work Studio.
// It monitors for pending objectives, executes them autonomously, and handles
// interruption logic based on ethical framework and user context.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Solifugus/ai-work-studio/internal/config"
	"github.com/Solifugus/ai-work-studio/pkg/core"
	"github.com/Solifugus/ai-work-studio/pkg/llm"
	"github.com/Solifugus/ai-work-studio/pkg/mcp"
	"github.com/Solifugus/ai-work-studio/pkg/storage"
)

// Agent represents the background daemon with all its dependencies.
type Agent struct {
	config            *config.Config
	configPath        string
	store             *storage.Store
	goalManager       *core.GoalManager
	objectiveManager  *core.ObjectiveManager
	methodManager     *core.MethodManager
	contextManager    *core.UserContextManager
	ethicalFramework  *core.EthicalFramework
	learningLoop      *core.LearningLoop
	scheduler         *Scheduler
	llmRouter         *llm.Router
	logger            *ActivityLogger
	ctx               context.Context
	cancel            context.CancelFunc
}

// AgentConfig contains configuration specific to the background agent.
type AgentConfig struct {
	// CheckInterval determines how often to check for new objectives (in seconds)
	CheckInterval int

	// MaxConcurrentObjectives limits parallel objective execution
	MaxConcurrentObjectives int

	// EnableAutoExecution controls whether objectives are executed automatically
	EnableAutoExecution bool

	// LogLevel controls verbosity (debug, info, warn, error)
	LogLevel string

	// ActivityLogRetentionDays controls how long to keep activity logs
	ActivityLogRetentionDays int
}

// DefaultAgentConfig returns sensible defaults for the agent.
func DefaultAgentConfig() *AgentConfig {
	return &AgentConfig{
		CheckInterval:            30, // Check every 30 seconds
		MaxConcurrentObjectives:  3,  // Max 3 concurrent objectives
		EnableAutoExecution:      true,
		LogLevel:                 "info",
		ActivityLogRetentionDays: 30,
	}
}

// MockLLMService provides a simple mock for the agent's LLM needs.
// TODO: Replace with actual LLM service integration
type MockLLMService struct{}

// MockLearningAgent provides a simple mock for learning agent functionality.
type MockLearningAgent struct{}

// MockLLMReasoner implements the LLMReasoner interface for ContemplativeCursor.
type MockLLMReasoner struct {
	router *llm.Router
}

// MockTaskExecutor implements the TaskExecutor interface for RealTimeCursor.
type MockTaskExecutor struct {
	router *llm.Router
}

// MockContextLoader implements the ContextLoader interface for RealTimeCursor.
type MockContextLoader struct {
	manager *core.UserContextManager
}

// AnalyzeObjective implements LLMReasoner interface.
func (m *MockLLMReasoner) AnalyzeObjective(ctx context.Context, objective *core.Objective, goalContext *core.Goal) (*core.ObjectiveAnalysis, error) {
	return &core.ObjectiveAnalysis{
		ComplexityLevel:      3,
		RequiredCapabilities: []string{"basic_execution"},
		KeyChallenges:        []string{"mock_challenge"},
	}, nil
}

// SelectMethod implements LLMReasoner interface.
func (m *MockLLMReasoner) SelectMethod(ctx context.Context, analysis *core.ObjectiveAnalysis, methods []*core.Method) (*core.MethodSelection, error) {
	// Return nil for now since we're not using the learning loop
	return nil, fmt.Errorf("mock implementation - not used")
}

// DecomposePlan implements LLMReasoner interface.
func (m *MockLLMReasoner) DecomposePlan(ctx context.Context, objective *core.Objective, method *core.Method) (*core.ExecutionPlan, error) {
	return &core.ExecutionPlan{
		ID:          "mock-plan",
		ObjectiveID: objective.ID,
		MethodID:    method.ID,
	}, nil
}

// EstimateTokenUsage implements TaskExecutor interface.
func (m *MockTaskExecutor) EstimateTokenUsage(ctx context.Context, task *core.ExecutionTask) (int, error) {
	return 50, nil // Mock estimate
}

// ExecuteTask implements TaskExecutor interface.
func (m *MockTaskExecutor) ExecuteTask(ctx context.Context, task *core.ExecutionTask) (*core.TaskResult, error) {
	// Return nil for now since we're not using the learning loop
	return nil, fmt.Errorf("mock implementation - not used")
}

// LoadObjectiveContext implements ContextLoader interface.
func (m *MockContextLoader) LoadObjectiveContext(ctx context.Context, objectiveID string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"mock": "context loaded for daemon execution",
	}, nil
}

// LoadTaskContext implements ContextLoader interface.
func (m *MockContextLoader) LoadTaskContext(ctx context.Context, task *core.ExecutionTask) (map[string]interface{}, error) {
	return map[string]interface{}{
		"mock": "task context loaded",
	}, nil
}

// AnalyzeExecutionOutcome implements the LearningAgent interface.
func (m *MockLearningAgent) AnalyzeExecutionOutcome(ctx context.Context, result *core.ExecutionResult, plan *core.ExecutionPlan, method *core.Method) (*core.ExecutionAnalysis, error) {
	return &core.ExecutionAnalysis{
		OverallAssessment:   core.OutcomeSuccess,
		PrimaryFailureCause: "",
		SuccessFactors:      []string{"Mock execution completed successfully"},
		ConfidenceLevel:     0.8,
	}, nil
}

// ProposeMethodRefinement implements the LearningAgent interface.
func (m *MockLearningAgent) ProposeMethodRefinement(ctx context.Context, analysis *core.ExecutionAnalysis, method *core.Method) (*core.MethodRefinement, error) {
	return &core.MethodRefinement{
		Type:                          core.RefinementNone,
		Reasoning:                     "Mock refinement suggestion",
		ExpectedComplexityChange:      0,
		ExpectedSuccessRateImprovement: 0,
	}, nil
}

// EvaluateRefinement implements the LearningAgent interface.
func (m *MockLearningAgent) EvaluateRefinement(ctx context.Context, original *core.Method, refinement *core.MethodRefinement) (*core.RefinementEvaluation, error) {
	return &core.RefinementEvaluation{
		IsImprovement:     false,
		ReducesComplexity: false,
		QualityScore:      5.0,
	}, nil
}

// Execute implements the LLMServiceInterface for the agent.
func (m *MockLLMService) Execute(ctx context.Context, params mcp.ServiceParams) mcp.ServiceResult {
	// Debug: log what we receive
	if operation, ok := params["operation"]; ok {
		log.Printf("[DEBUG] MockLLMService received operation: %v", operation)
	}
	if prompt, ok := params["prompt"]; ok {
		log.Printf("[DEBUG] MockLLMService received prompt: %v", prompt)
	}

	// Return structured responses for different types of requests
	switch operation := params["operation"].(string); operation {
	case "complete":
		// Check if this is an ethical analysis based on the prompt content
		if prompt, ok := params["prompt"].(string); ok {
			// Check for various indicators this is an ethical evaluation
			isEthical := strings.Contains(strings.ToLower(prompt), "ethical") ||
				strings.Contains(strings.ToLower(prompt), "freedom") ||
				strings.Contains(strings.ToLower(prompt), "well-being") ||
				strings.Contains(strings.ToLower(prompt), "sustainability") ||
				strings.Contains(strings.ToLower(prompt), "impact")

			if isEthical {
				response := `Freedom Impact: 0.8
Well-Being Impact: 0.7
Sustainability Impact: 0.6
Confidence: 0.9
Reasoning: Automated execution appears to have positive impact across all dimensions.`

				return mcp.ServiceResult{
					Success: true,
					Data: &mcp.CompletionResponse{
						Text:       response,
						TokensUsed: 50,
						Model:      "mock-agent-model",
						Provider:   "mock",
						Cost:       0.001,
					},
					Metadata: map[string]interface{}{
						"agent": true,
					},
				}
			}
		}
		fallthrough
	default:
		return mcp.ServiceResult{
			Success: true,
			Data: &mcp.CompletionResponse{
				Text:       "Mock agent response",
				TokensUsed: 25,
				Model:      "mock-agent-model",
				Provider:   "mock",
				Cost:       0.001,
			},
			Metadata: map[string]interface{}{
				"agent": true,
			},
		}
	}
}

func main() {
	// Parse command line arguments
	var configPath string
	var dataDir string
	var verbose bool
	var checkInterval int
	var dryRun bool

	flag.StringVar(&configPath, "config", "", "Configuration file path")
	flag.StringVar(&dataDir, "data", "", "Data directory path (overrides config)")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.IntVar(&checkInterval, "interval", 30, "Check interval in seconds")
	flag.BoolVar(&dryRun, "dry-run", false, "Simulate execution without making changes")
	flag.Parse()

	// Get default config path if not specified
	if configPath == "" {
		var err error
		configPath, err = config.GetConfigPath()
		if err != nil {
			log.Fatalf("Error getting config path: %v", err)
		}
	}

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// Override data directory if specified
	if dataDir != "" {
		cfg.DataDir = dataDir
	}

	// Override verbose setting if specified
	if verbose {
		cfg.Preferences.VerboseOutput = true
	}

	// Ensure data directory exists
	if err := cfg.EnsureDataDir(); err != nil {
		log.Fatalf("Error setting up data directory: %v", err)
	}

	// Initialize agent
	agent, err := NewAgent(cfg, configPath, checkInterval, dryRun)
	if err != nil {
		log.Fatalf("Error initializing agent: %v", err)
	}
	defer agent.Close()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the agent
	if err := agent.Start(); err != nil {
		log.Fatalf("Error starting agent: %v", err)
	}

	log.Printf("AI Work Studio Agent started (PID: %d)", os.Getpid())
	log.Printf("Data directory: %s", cfg.DataDir)
	log.Printf("Check interval: %d seconds", checkInterval)
	if dryRun {
		log.Printf("Running in dry-run mode (no actual execution)")
	}

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutdown signal received, stopping agent...")

	// Graceful shutdown
	agent.Stop()
	log.Println("Agent stopped successfully")
}

// NewAgent creates a new background agent instance with all dependencies.
func NewAgent(cfg *config.Config, configPath string, checkInterval int, dryRun bool) (*Agent, error) {
	// Initialize storage
	store, err := storage.NewStore(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize managers
	goalManager := core.NewGoalManager(store)
	objectiveManager := core.NewObjectiveManager(store)
	methodManager := core.NewMethodManager(store)
	contextManager := core.NewUserContextManager(store)

	// Initialize LLM router
	llmRouter := llm.NewRouter(&MockLLMService{})

	// Initialize ethical framework
	ethicalFramework := core.NewEthicalFramework(store, llmRouter, contextManager)

	// Initialize learning loop components
	// TODO: Implement proper learning loop integration
	// For now, use nil to focus on basic daemon functionality
	var learningLoop *core.LearningLoop = nil

	// Initialize activity logger
	logger, err := NewActivityLogger(store, cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize activity logger: %w", err)
	}

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Create agent configuration
	agentConfig := DefaultAgentConfig()
	agentConfig.CheckInterval = checkInterval

	// Initialize scheduler
	scheduler, err := NewScheduler(SchedulerConfig{
		CheckInterval:           time.Duration(checkInterval) * time.Second,
		MaxConcurrentObjectives: agentConfig.MaxConcurrentObjectives,
		DryRun:                  dryRun,
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize scheduler: %w", err)
	}

	return &Agent{
		config:           cfg,
		configPath:       configPath,
		store:            store,
		goalManager:      goalManager,
		objectiveManager: objectiveManager,
		methodManager:    methodManager,
		contextManager:   contextManager,
		ethicalFramework: ethicalFramework,
		learningLoop:     learningLoop,
		scheduler:        scheduler,
		llmRouter:        llmRouter,
		logger:           logger,
		ctx:              ctx,
		cancel:           cancel,
	}, nil
}

// Start begins the background monitoring and execution.
func (a *Agent) Start() error {
	// Log agent startup
	a.logger.LogActivity("agent_startup", map[string]interface{}{
		"pid":            os.Getpid(),
		"data_directory": a.config.DataDir,
		"auto_approve":   a.config.Preferences.AutoApprove,
		"check_interval": a.scheduler.config.CheckInterval,
	})

	// Start the scheduler
	go a.scheduler.Start(a.ctx, &SchedulerDependencies{
		ObjectiveManager: a.objectiveManager,
		LearningLoop:     a.learningLoop,
		EthicalFramework: a.ethicalFramework,
		ContextManager:   a.contextManager,
		Config:           a.config,
		Logger:           a.logger,
	})

	return nil
}

// Stop performs graceful shutdown of the agent.
func (a *Agent) Stop() {
	// Cancel the context to stop all goroutines
	a.cancel()

	// Give some time for graceful shutdown
	time.Sleep(2 * time.Second)

	// Log agent shutdown
	a.logger.LogActivity("agent_shutdown", map[string]interface{}{
		"shutdown_time": time.Now(),
	})
}

// Close cleans up agent resources.
func (a *Agent) Close() {
	if a.store != nil {
		a.store.Close()
	}
}