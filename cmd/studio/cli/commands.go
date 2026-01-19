package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/yourusername/ai-work-studio/internal/config"
	"github.com/yourusername/ai-work-studio/pkg/core"
)

// createGoal creates a new goal with the given parameters.
func (cli *CLI) createGoal(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: create-goal <title> [description] [priority]")
	}

	parsed := parseArgs(args, 3)
	title := parsed[0]
	description := parsed[1]
	priority := parseInt(parsed[2], cli.config.Preferences.DefaultPriority)

	if priority < 1 || priority > 10 {
		return fmt.Errorf("priority must be between 1 and 10, got %d", priority)
	}

	ctx := context.Background()

	// Create the goal
	goal, err := cli.goalManager.CreateGoal(ctx, title, description, priority, nil)
	if err != nil {
		return fmt.Errorf("failed to create goal: %w", err)
	}

	// Update current goal in session if this is the first goal
	if cli.config.Session.CurrentGoalID == "" {
		updates := config.SessionUpdates{
			CurrentGoalID: &goal.ID,
		}
		if err := cli.config.UpdateSession(cli.configPath, updates); err != nil {
			// Log warning but don't fail
			fmt.Printf("Warning: failed to update session: %v\n", err)
		}
	}

	if cli.config.Preferences.VerboseOutput {
		fmt.Printf("✓ Created goal: %s\n", goal.ID)
		fmt.Printf("  Title: %s\n", goal.Title)
		if goal.Description != "" {
			fmt.Printf("  Description: %s\n", goal.Description)
		}
		fmt.Printf("  Priority: %d\n", goal.Priority)
		fmt.Printf("  Status: %s\n", goal.Status)
		fmt.Printf("  Created: %s\n", formatTime(goal.CreatedAt))
	} else {
		fmt.Printf("✓ Created goal: %s (%s)\n", goal.Title, goal.ID)
	}

	return nil
}

// createObjective creates a new objective for a goal.
func (cli *CLI) createObjective(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: create-objective <goal-id> <title> [description] [priority]")
	}

	parsed := parseArgs(args, 4)
	goalID := parsed[0]
	title := parsed[1]
	description := parsed[2]
	priority := parseInt(parsed[3], cli.config.Preferences.DefaultPriority)

	if priority < 1 || priority > 10 {
		return fmt.Errorf("priority must be between 1 and 10, got %d", priority)
	}

	ctx := context.Background()

	// Verify goal exists
	goal, err := cli.goalManager.GetGoal(ctx, goalID)
	if err != nil {
		return fmt.Errorf("goal not found: %w", err)
	}

	// For now, use a placeholder method ID
	// TODO: Integrate with method selection when method system is ready
	methodID := "placeholder-method"

	// Create the objective
	objective, err := cli.objectiveManager.CreateObjective(ctx, goalID, methodID, title, description, nil, priority)
	if err != nil {
		return fmt.Errorf("failed to create objective: %w", err)
	}

	if cli.config.Preferences.VerboseOutput {
		fmt.Printf("✓ Created objective: %s\n", objective.ID)
		fmt.Printf("  Title: %s\n", objective.Title)
		if objective.Description != "" {
			fmt.Printf("  Description: %s\n", objective.Description)
		}
		fmt.Printf("  Goal: %s (%s)\n", goal.Title, goalID)
		fmt.Printf("  Priority: %d\n", objective.Priority)
		fmt.Printf("  Status: %s\n", objective.Status)
		fmt.Printf("  Created: %s\n", formatTime(objective.CreatedAt))
	} else {
		fmt.Printf("✓ Created objective: %s for goal %s\n", objective.Title, goal.Title)
	}

	return nil
}

// listGoals lists all goals, optionally filtered by status.
func (cli *CLI) listGoals(args []string) error {
	var statusFilter *core.GoalStatus

	if len(args) > 0 {
		status := core.GoalStatus(args[0])
		statusFilter = &status
	}

	ctx := context.Background()

	// Build filter
	filter := core.GoalFilter{}
	if statusFilter != nil {
		filter.Status = statusFilter
	}

	// Get goals
	goals, err := cli.goalManager.ListGoals(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list goals: %w", err)
	}

	if len(goals) == 0 {
		if statusFilter != nil {
			fmt.Printf("No goals found with status: %s\n", *statusFilter)
		} else {
			fmt.Printf("No goals found. Use 'create-goal' to create your first goal.\n")
		}
		return nil
	}

	// Display goals in a table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	if cli.config.Preferences.VerboseOutput {
		fmt.Fprintln(w, "ID\tTitle\tStatus\tPriority\tCreated\tDescription")
		fmt.Fprintln(w, "---\t-----\t------\t--------\t-------\t-----------")

		for _, goal := range goals {
			description := goal.Description
			if len(description) > 50 {
				description = description[:47] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
				goal.ID[:8], goal.Title, goal.Status, goal.Priority,
				formatTime(goal.CreatedAt), description)
		}
	} else {
		fmt.Fprintln(w, "Title\tStatus\tPriority\tCreated")
		fmt.Fprintln(w, "-----\t------\t--------\t-------")

		for _, goal := range goals {
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\n",
				goal.Title, goal.Status, goal.Priority, formatTime(goal.CreatedAt))
		}
	}

	return nil
}

// listObjectives lists objectives, optionally filtered by goal and status.
func (cli *CLI) listObjectives(args []string) error {
	var goalIDFilter string
	var statusFilter *core.ObjectiveStatus

	if len(args) > 0 {
		goalIDFilter = args[0]
	}
	if len(args) > 1 {
		status := core.ObjectiveStatus(args[1])
		statusFilter = &status
	}

	ctx := context.Background()

	// Build filter
	filter := core.ObjectiveFilter{}
	if goalIDFilter != "" {
		filter.GoalID = &goalIDFilter
	}
	if statusFilter != nil {
		filter.Status = statusFilter
	}

	// Get objectives
	objectives, err := cli.objectiveManager.ListObjectives(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list objectives: %w", err)
	}

	if len(objectives) == 0 {
		if goalIDFilter != "" && statusFilter != nil {
			fmt.Printf("No objectives found for goal %s with status %s\n", goalIDFilter, *statusFilter)
		} else if goalIDFilter != "" {
			fmt.Printf("No objectives found for goal %s\n", goalIDFilter)
		} else if statusFilter != nil {
			fmt.Printf("No objectives found with status: %s\n", *statusFilter)
		} else {
			fmt.Printf("No objectives found. Use 'create-objective' to create objectives for your goals.\n")
		}
		return nil
	}

	// Display objectives in a table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	if cli.config.Preferences.VerboseOutput {
		fmt.Fprintln(w, "ID\tTitle\tGoal ID\tStatus\tPriority\tCreated\tDescription")
		fmt.Fprintln(w, "---\t-----\t-------\t------\t--------\t-------\t-----------")

		for _, objective := range objectives {
			description := objective.Description
			if len(description) > 40 {
				description = description[:37] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
				objective.ID[:8], objective.Title, objective.GoalID[:8],
				objective.Status, objective.Priority, formatTime(objective.CreatedAt), description)
		}
	} else {
		fmt.Fprintln(w, "Title\tGoal ID\tStatus\tPriority\tCreated")
		fmt.Fprintln(w, "-----\t-------\t------\t--------\t-------")

		for _, objective := range objectives {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
				objective.Title, objective.GoalID[:8], objective.Status,
				objective.Priority, formatTime(objective.CreatedAt))
		}
	}

	return nil
}

// showStatus displays current system status and progress.
func (cli *CLI) showStatus(args []string) error {
	ctx := context.Background()

	fmt.Println("🎯 AI Work Studio Status")
	fmt.Println()

	// Show current goal if set
	if cli.config.Session.CurrentGoalID != "" {
		goal, err := cli.goalManager.GetGoal(ctx, cli.config.Session.CurrentGoalID)
		if err != nil {
			fmt.Printf("⚠️  Current goal (%s) not found\n", cli.config.Session.CurrentGoalID)
		} else {
			fmt.Printf("📋 Current Goal: %s\n", goal.Title)
			fmt.Printf("   Status: %s | Priority: %d\n", goal.Status, goal.Priority)
			if goal.Description != "" {
				fmt.Printf("   %s\n", goal.Description)
			}
		}
		fmt.Println()
	}

	// Show active goals summary
	activeFilter := core.GoalFilter{Status: &[]core.GoalStatus{core.GoalStatusActive}[0]}
	activeGoals, err := cli.goalManager.ListGoals(ctx, activeFilter)
	if err != nil {
		return fmt.Errorf("failed to get active goals: %w", err)
	}

	fmt.Printf("📊 Active Goals: %d\n", len(activeGoals))

	// Show in-progress objectives summary
	inProgressFilter := core.ObjectiveFilter{Status: &[]core.ObjectiveStatus{core.ObjectiveStatusInProgress}[0]}
	inProgressObjectives, err := cli.objectiveManager.ListObjectives(ctx, inProgressFilter)
	if err != nil {
		return fmt.Errorf("failed to get in-progress objectives: %w", err)
	}

	fmt.Printf("⚡ In Progress: %d objectives\n", len(inProgressObjectives))

	// Show recent completions
	completedFilter := core.ObjectiveFilter{Status: &[]core.ObjectiveStatus{core.ObjectiveStatusCompleted}[0]}
	completedObjectives, err := cli.objectiveManager.ListObjectives(ctx, completedFilter)
	if err != nil {
		return fmt.Errorf("failed to get completed objectives: %w", err)
	}

	// Count recent completions (last 24 hours)
	recentCompletions := 0
	for _, obj := range completedObjectives {
		if obj.CompletedAt != nil && time.Since(*obj.CompletedAt) < 24*time.Hour {
			recentCompletions++
		}
	}

	fmt.Printf("✅ Recent Completions: %d (last 24h)\n", recentCompletions)

	// Show budget status if configured
	if cli.config.BudgetLimits.DailyLimit > 0 {
		fmt.Println()
		fmt.Printf("💰 Budget Limits:\n")
		fmt.Printf("   Daily: $%.2f | Monthly: $%.2f | Per Request: $%.2f\n",
			cli.config.BudgetLimits.DailyLimit, cli.config.BudgetLimits.MonthlyLimit,
			cli.config.BudgetLimits.PerRequestLimit)
	}

	// Show data directory info
	if cli.config.Preferences.VerboseOutput {
		fmt.Println()
		fmt.Printf("📁 Data Directory: %s\n", cli.config.DataDir)
		fmt.Printf("👤 User ID: %s\n", cli.config.Session.UserID)
	}

	return nil
}

// provideFeedback handles user feedback on decisions or outcomes.
func (cli *CLI) provideFeedback(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: feedback <decision-id> <approve|reject> [message]")
	}

	decisionID := args[0]
	action := strings.ToLower(args[1])
	var message string
	if len(args) > 2 {
		message = strings.Join(args[2:], " ")
	}

	ctx := context.Background()

	// Validate action
	if action != "approve" && action != "reject" {
		return fmt.Errorf("action must be 'approve' or 'reject', got '%s'", action)
	}

	// Get the decision
	decision, err := cli.ethicalFramework.GetDecision(ctx, decisionID)
	if err != nil {
		return fmt.Errorf("decision not found: %w", err)
	}

	// Provide feedback
	if action == "approve" {
		if message == "" {
			message = "Approved via CLI"
		}
		err = cli.ethicalFramework.ApproveDecision(ctx, decisionID, message)
		if err != nil {
			return fmt.Errorf("failed to approve decision: %w", err)
		}
		fmt.Printf("✓ Approved decision: %s\n", decision.DecisionContext)
	} else {
		if message == "" {
			message = "Rejected via CLI"
		}
		err = cli.ethicalFramework.RejectDecision(ctx, decisionID, message)
		if err != nil {
			return fmt.Errorf("failed to reject decision: %w", err)
		}
		fmt.Printf("✗ Rejected decision: %s\n", decision.DecisionContext)
	}

	if cli.config.Preferences.VerboseOutput {
		fmt.Printf("  Feedback: %s\n", message)
		fmt.Printf("  Impact scores: Freedom=%.1f, Well-being=%.1f, Sustainability=%.1f\n",
			decision.Impact.FreedomImpact, decision.Impact.WellBeingImpact, decision.Impact.SustainabilityImpact)
	}

	return nil
}

// manageConfig handles configuration management commands.
func (cli *CLI) manageConfig(args []string) error {
	if len(args) == 0 {
		return cli.showConfig()
	}

	action := args[0]
	switch action {
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: config get <key>")
		}
		return cli.getConfigValue(args[1])
	case "set":
		if len(args) < 3 {
			return fmt.Errorf("usage: config set <key> <value>")
		}
		return cli.setConfigValue(args[1], args[2])
	default:
		return fmt.Errorf("unknown config action: %s. Use 'get' or 'set'", action)
	}
}

// showConfig displays current configuration.
func (cli *CLI) showConfig() error {
	fmt.Println("🔧 Configuration Settings")
	fmt.Println()

	fmt.Printf("Data Directory: %s\n", cli.config.DataDir)
	fmt.Println()

	fmt.Printf("Budget Limits:\n")
	fmt.Printf("  daily-limit: $%.2f\n", cli.config.BudgetLimits.DailyLimit)
	fmt.Printf("  monthly-limit: $%.2f\n", cli.config.BudgetLimits.MonthlyLimit)
	fmt.Printf("  per-request-limit: $%.2f\n", cli.config.BudgetLimits.PerRequestLimit)
	fmt.Println()

	fmt.Printf("Preferences:\n")
	fmt.Printf("  auto-approve: %t\n", cli.config.Preferences.AutoApprove)
	fmt.Printf("  verbose-output: %t\n", cli.config.Preferences.VerboseOutput)
	fmt.Printf("  default-priority: %d\n", cli.config.Preferences.DefaultPriority)
	fmt.Printf("  interactive-mode: %t\n", cli.config.Preferences.InteractiveMode)
	fmt.Println()

	fmt.Printf("Session:\n")
	fmt.Printf("  current-goal-id: %s\n", cli.config.Session.CurrentGoalID)
	fmt.Printf("  user-id: %s\n", cli.config.Session.UserID)

	return nil
}

// getConfigValue retrieves and displays a specific configuration value.
func (cli *CLI) getConfigValue(key string) error {
	switch key {
	case "data-dir":
		fmt.Println(cli.config.DataDir)
	case "daily-limit":
		fmt.Printf("%.2f\n", cli.config.BudgetLimits.DailyLimit)
	case "monthly-limit":
		fmt.Printf("%.2f\n", cli.config.BudgetLimits.MonthlyLimit)
	case "per-request-limit":
		fmt.Printf("%.2f\n", cli.config.BudgetLimits.PerRequestLimit)
	case "auto-approve":
		fmt.Printf("%t\n", cli.config.Preferences.AutoApprove)
	case "verbose-output":
		fmt.Printf("%t\n", cli.config.Preferences.VerboseOutput)
	case "default-priority":
		fmt.Printf("%d\n", cli.config.Preferences.DefaultPriority)
	case "interactive-mode":
		fmt.Printf("%t\n", cli.config.Preferences.InteractiveMode)
	case "current-goal-id":
		fmt.Println(cli.config.Session.CurrentGoalID)
	case "user-id":
		fmt.Println(cli.config.Session.UserID)
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

// setConfigValue updates a configuration value.
func (cli *CLI) setConfigValue(key, value string) error {
	switch key {
	case "daily-limit":
		limit, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid daily limit: %s", value)
		}
		updates := config.BudgetUpdates{DailyLimit: &limit}
		return cli.config.UpdateBudgetLimits(cli.configPath, updates)

	case "monthly-limit":
		limit, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid monthly limit: %s", value)
		}
		updates := config.BudgetUpdates{MonthlyLimit: &limit}
		return cli.config.UpdateBudgetLimits(cli.configPath, updates)

	case "per-request-limit":
		limit, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid per-request limit: %s", value)
		}
		updates := config.BudgetUpdates{PerRequestLimit: &limit}
		return cli.config.UpdateBudgetLimits(cli.configPath, updates)

	case "auto-approve":
		autoApprove, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %s", value)
		}
		updates := config.PreferenceUpdates{AutoApprove: &autoApprove}
		return cli.config.UpdatePreferences(cli.configPath, updates)

	case "verbose-output":
		verboseOutput, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %s", value)
		}
		updates := config.PreferenceUpdates{VerboseOutput: &verboseOutput}
		return cli.config.UpdatePreferences(cli.configPath, updates)

	case "default-priority":
		priority, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid priority: %s", value)
		}
		updates := config.PreferenceUpdates{DefaultPriority: &priority}
		return cli.config.UpdatePreferences(cli.configPath, updates)

	case "interactive-mode":
		interactiveMode, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %s", value)
		}
		updates := config.PreferenceUpdates{InteractiveMode: &interactiveMode}
		return cli.config.UpdatePreferences(cli.configPath, updates)

	case "current-goal-id":
		updates := config.SessionUpdates{CurrentGoalID: &value}
		return cli.config.UpdateSession(cli.configPath, updates)

	case "user-id":
		updates := config.SessionUpdates{UserID: &value}
		return cli.config.UpdateSession(cli.configPath, updates)

	default:
		return fmt.Errorf("unknown or read-only config key: %s", key)
	}
}

// interactiveMode enters conversation-like interactive mode.
func (cli *CLI) interactiveMode(args []string) error {
	fmt.Println("🤖 AI Work Studio - Interactive Mode")
	fmt.Println("Type 'help' for commands, 'exit' to quit")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("ai-work-studio> ")
		line, _, err := reader.ReadLine()
		if err != nil {
			break
		}

		input := strings.TrimSpace(string(line))
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("👋 Goodbye!")
			break
		}

		// Parse command and arguments
		parts := strings.Fields(input)
		if len(parts) == 0 {
			continue
		}

		commandName := parts[0]
		commandArgs := parts[1:]

		// Execute command
		if err := cli.executeCommand(commandName, commandArgs); err != nil {
			fmt.Printf("Error: %v\n", err)
		}

		fmt.Println()
	}

	return nil
}

// showHelp displays help information.
func (cli *CLI) showHelp(args []string) error {
	if len(args) > 0 {
		// Show help for specific command
		commandName := args[0]
		command, exists := getCommands()[commandName]
		if !exists {
			return fmt.Errorf("unknown command: %s", commandName)
		}

		fmt.Printf("Command: %s\n", command.Name)
		fmt.Printf("Description: %s\n", command.Description)
		fmt.Printf("Usage: %s\n", command.Usage)
		return nil
	}

	// Show general help
	fmt.Println("🎯 AI Work Studio - Command Line Interface")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  ai-work-studio [global-options] <command> [command-options]")
	fmt.Println()
	fmt.Println("GLOBAL OPTIONS:")
	fmt.Println("  -config <path>    Configuration file path")
	fmt.Println("  -data <path>      Data directory path")
	fmt.Println("  -verbose          Enable verbose output")
	fmt.Println()
	fmt.Println("COMMANDS:")

	// Display commands in a table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "  Command\tDescription")
	fmt.Fprintln(w, "  -------\t-----------")

	for _, command := range getCommands() {
		fmt.Fprintf(w, "  %s\t%s\n", command.Name, command.Description)
	}

	fmt.Println()
	fmt.Println("Use 'help <command>' for detailed information about a specific command.")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  ai-work-studio create-goal \"Learn Go programming\" \"Master Go for backend development\" 8")
	fmt.Println("  ai-work-studio list-goals active")
	fmt.Println("  ai-work-studio status")
	fmt.Println("  ai-work-studio interactive")

	return nil
}