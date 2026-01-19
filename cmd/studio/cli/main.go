// Package main provides the command-line interface for AI Work Studio.
// It offers commands for goal and objective management, status monitoring, and user feedback.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Solifugus/ai-work-studio/internal/config"
	"github.com/Solifugus/ai-work-studio/pkg/core"
	"github.com/Solifugus/ai-work-studio/pkg/llm"
	"github.com/Solifugus/ai-work-studio/pkg/mcp"
	"github.com/Solifugus/ai-work-studio/pkg/storage"
)

// CLI represents the command-line interface with its dependencies.
type CLI struct {
	config           *config.Config
	configPath       string
	store            *storage.Store
	goalManager      *core.GoalManager
	objectiveManager *core.ObjectiveManager
	methodManager    *core.MethodManager
	contextManager   *core.UserContextManager
	ethicalFramework *core.EthicalFramework
	llmRouter        *llm.Router
}

// Command represents a CLI command with its handler function.
type Command struct {
	Name        string
	Description string
	Usage       string
	Handler     func(*CLI, []string) error
}

// getCommands returns the available commands map
func getCommands() map[string]Command {
	return map[string]Command{
	"create-goal": {
		Name:        "create-goal",
		Description: "Create a new goal",
		Usage:       "create-goal <title> [description] [priority]",
		Handler:     (*CLI).createGoal,
	},
	"create-objective": {
		Name:        "create-objective",
		Description: "Create a new objective for a goal",
		Usage:       "create-objective <goal-id> <title> [description] [priority]",
		Handler:     (*CLI).createObjective,
	},
	"list-goals": {
		Name:        "list-goals",
		Description: "List all goals",
		Usage:       "list-goals [status]",
		Handler:     (*CLI).listGoals,
	},
	"list-objectives": {
		Name:        "list-objectives",
		Description: "List objectives for a goal",
		Usage:       "list-objectives [goal-id] [status]",
		Handler:     (*CLI).listObjectives,
	},
	"status": {
		Name:        "status",
		Description: "Show current status and progress",
		Usage:       "status",
		Handler:     (*CLI).showStatus,
	},
	"feedback": {
		Name:        "feedback",
		Description: "Provide feedback on decisions or outcomes",
		Usage:       "feedback <decision-id> <approve|reject> [message]",
		Handler:     (*CLI).provideFeedback,
	},
	"config": {
		Name:        "config",
		Description: "Manage configuration settings",
		Usage:       "config [get|set] [key] [value]",
		Handler:     (*CLI).manageConfig,
	},
	"interactive": {
		Name:        "interactive",
		Description: "Enter interactive conversation mode",
		Usage:       "interactive",
		Handler:     (*CLI).interactiveMode,
	},
	"help": {
		Name:        "help",
		Description: "Show help information",
		Usage:       "help [command]",
		Handler:     (*CLI).showHelp,
	},
	}
}

func main() {
	// Parse global flags
	var configPath string
	var verbose bool
	var dataDir string

	flag.StringVar(&configPath, "config", "", "Configuration file path (default: ~/.ai-work-studio/config.json)")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.StringVar(&dataDir, "data", "", "Data directory path (overrides config)")
	flag.Parse()

	// Get default config path if not specified
	if configPath == "" {
		var err error
		configPath, err = config.GetConfigPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
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
		fmt.Fprintf(os.Stderr, "Error setting up data directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize CLI
	cli, err := NewCLI(cfg, configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing CLI: %v\n", err)
		os.Exit(1)
	}
	defer cli.Close()

	// Get command arguments
	args := flag.Args()

	// If no command provided, show help or enter interactive mode
	if len(args) == 0 {
		if cfg.Preferences.InteractiveMode {
			if err := cli.interactiveMode([]string{}); err != nil {
				fmt.Fprintf(os.Stderr, "Error in interactive mode: %v\n", err)
				os.Exit(1)
			}
		} else {
			cli.showHelp([]string{})
		}
		return
	}

	// Execute command
	commandName := args[0]
	commandArgs := args[1:]

	if err := cli.executeCommand(commandName, commandArgs); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// NewCLI creates a new CLI instance with initialized dependencies.
func NewCLI(cfg *config.Config, configPath string) (*CLI, error) {
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

	// Initialize LLM router (with mock service for now)
	llmRouter := llm.NewRouter(&MockLLMService{})

	// Initialize ethical framework
	ethicalFramework := core.NewEthicalFramework(store, llmRouter, contextManager)

	return &CLI{
		config:           cfg,
		configPath:       configPath,
		store:            store,
		goalManager:      goalManager,
		objectiveManager: objectiveManager,
		methodManager:    methodManager,
		contextManager:   contextManager,
		ethicalFramework: ethicalFramework,
		llmRouter:        llmRouter,
	}, nil
}

// Close cleans up CLI resources.
func (cli *CLI) Close() {
	if cli.store != nil {
		cli.store.Close()
	}
}

// executeCommand executes a CLI command by name.
func (cli *CLI) executeCommand(commandName string, args []string) error {
	command, exists := getCommands()[commandName]
	if !exists {
		return fmt.Errorf("unknown command: %s. Use 'help' to see available commands", commandName)
	}

	return command.Handler(cli, args)
}

// MockLLMService provides a simple mock for testing the CLI.
// TODO: Replace with actual LLM service integration
type MockLLMService struct{}

// Execute implements the LLMServiceInterface for testing.
func (m *MockLLMService) Execute(ctx context.Context, params mcp.ServiceParams) mcp.ServiceResult {
	// Mock response for LLM requests
	response := &mcp.CompletionResponse{
		Text:       "Mock LLM response for CLI testing",
		TokensUsed: 50,
		Model:      "mock-model",
		Provider:   "mock",
		Cost:       0.001,
	}

	return mcp.ServiceResult{
		Success: true,
		Data:    response,
		Metadata: map[string]interface{}{
			"mock": true,
		},
	}
}

// Utility functions

// parseArgs extracts arguments with optional defaults.
func parseArgs(args []string, expectedCount int) []string {
	result := make([]string, expectedCount)
	for i := 0; i < expectedCount && i < len(args); i++ {
		result[i] = args[i]
	}
	return result
}

// parseInt safely parses an integer with a default value.
func parseInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}

	var result int
	if _, err := fmt.Sscanf(s, "%d", &result); err != nil {
		return defaultValue
	}
	return result
}

// readUserInput reads a line of input from the user with a prompt.
func readUserInput(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _, err := reader.ReadLine()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(line)), nil
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// formatTime formats a time in a user-friendly way.
func formatTime(t time.Time) string {
	now := time.Now()

	// If within the last 24 hours, show relative time
	if now.Sub(t) < 24*time.Hour {
		if now.Sub(t) < time.Hour {
			mins := int(now.Sub(t).Minutes())
			if mins < 1 {
				return "just now"
			}
			return fmt.Sprintf("%d minutes ago", mins)
		}
		hours := int(now.Sub(t).Hours())
		return fmt.Sprintf("%d hours ago", hours)
	}

	// Otherwise show the date
	return t.Format("Jan 2, 15:04")
}