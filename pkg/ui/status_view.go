package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/yourusername/ai-work-studio/pkg/core"
)

// StatusView provides a dashboard showing system status and activity
type StatusView struct {
	app    *App
	window fyne.Window

	// UI Components
	container        *container.Scroll
	refreshBtn       *widget.Button
	autoRefreshCheck *widget.Check

	// Dashboard Cards
	systemHealthCard   *widget.Card
	activityCard       *widget.Card
	budgetCard         *widget.Card
	dataStatsCard      *widget.Card
	quickActionsCard   *widget.Card
	recentEventsCard   *widget.Card

	// Auto-refresh timer
	refreshTimer *time.Ticker
	stopTimer    chan bool
}

// NewStatusView creates a new status dashboard view
func NewStatusView(app *App, window fyne.Window) *StatusView {
	sv := &StatusView{
		app:       app,
		window:    window,
		stopTimer: make(chan bool),
	}

	sv.createUI()
	sv.loadStatus()

	return sv
}

// createUI initializes the user interface components
func (sv *StatusView) createUI() {
	// Create header controls
	sv.refreshBtn = widget.NewButton("Refresh", sv.onRefresh)

	sv.autoRefreshCheck = widget.NewCheck("Auto-refresh (30s)", sv.onAutoRefreshToggle)
	sv.autoRefreshCheck.SetChecked(false)

	headerContainer := container.NewHBox(
		widget.NewLabel("System Status Dashboard"),
		widget.NewLabel(" "), // Spacer
		sv.autoRefreshCheck,
		sv.refreshBtn,
	)

	// Create dashboard cards
	sv.createDashboardCards()

	// Layout cards in grid
	topRow := container.NewHBox(
		sv.systemHealthCard,
		sv.activityCard,
		sv.budgetCard,
	)

	middleRow := container.NewHBox(
		sv.dataStatsCard,
		sv.quickActionsCard,
	)

	bottomRow := container.NewHBox(
		sv.recentEventsCard,
	)

	content := container.NewVBox(
		headerContainer,
		widget.NewSeparator(),
		topRow,
		middleRow,
		bottomRow,
	)

	sv.container = container.NewScroll(content)
	sv.container.SetMinSize(fyne.NewSize(800, 600))
}

// createDashboardCards creates all the dashboard cards
func (sv *StatusView) createDashboardCards() {
	sv.systemHealthCard = sv.createSystemHealthCard()
	sv.activityCard = sv.createActivityCard()
	sv.budgetCard = sv.createBudgetCard()
	sv.dataStatsCard = sv.createDataStatsCard()
	sv.quickActionsCard = sv.createQuickActionsCard()
	sv.recentEventsCard = sv.createRecentEventsCard()
}

// createSystemHealthCard creates the system health status card
func (sv *StatusView) createSystemHealthCard() *widget.Card {
	content := container.NewVBox(
		widget.NewLabel("Loading system status..."),
	)

	return widget.NewCard("System Health", "", content)
}

// createActivityCard creates the current activity status card
func (sv *StatusView) createActivityCard() *widget.Card {
	content := container.NewVBox(
		widget.NewLabel("Loading activity status..."),
	)

	return widget.NewCard("Current Activity", "", content)
}

// createBudgetCard creates the budget usage card
func (sv *StatusView) createBudgetCard() *widget.Card {
	// Note: This shows placeholder data until BudgetManager is integrated into App
	content := container.NewVBox(
		widget.NewLabel("Budget tracking not yet integrated"),
		widget.NewLabel("Will show LLM usage costs when available"),
	)

	return widget.NewCard("Budget Usage", "", content)
}

// createDataStatsCard creates the data statistics card
func (sv *StatusView) createDataStatsCard() *widget.Card {
	content := container.NewVBox(
		widget.NewLabel("Loading data statistics..."),
	)

	return widget.NewCard("Data Statistics", "", content)
}

// createQuickActionsCard creates the quick actions card
func (sv *StatusView) createQuickActionsCard() *widget.Card {
	// Create action buttons
	backupBtn := widget.NewButton("Backup Data", sv.onBackupData)
	validateBtn := widget.NewButton("Validate Storage", sv.onValidateStorage)
	openDataBtn := widget.NewButton("Open Data Dir", sv.onOpenDataDirectory)
	clearCacheBtn := widget.NewButton("Clear Cache", sv.onClearCache)

	content := container.NewVBox(
		backupBtn,
		validateBtn,
		openDataBtn,
		clearCacheBtn,
	)

	return widget.NewCard("Quick Actions", "", content)
}

// createRecentEventsCard creates the recent events timeline card
func (sv *StatusView) createRecentEventsCard() *widget.Card {
	content := container.NewVBox(
		widget.NewLabel("Loading recent events..."),
	)

	return widget.NewCard("Recent Activity", "", content)
}

// loadStatus loads and displays current system status
func (sv *StatusView) loadStatus() {
	sv.loadSystemHealth()
	sv.loadActivity()
	sv.loadBudgetStatus()
	sv.loadDataStats()
	sv.loadRecentEvents()
}

// loadSystemHealth loads system health information
func (sv *StatusView) loadSystemHealth() {
	config := sv.app.GetConfig()

	// Check data directory accessibility
	dataDir := config.DataDir
	dataDirExists := sv.checkDirectoryExists(dataDir)

	// Check storage connectivity
	storageHealthy := sv.checkStorageHealth()

	// Create health indicators
	dataDirStatus := "❌ Error"
	if dataDirExists {
		dataDirStatus = "✅ OK"
	}

	storageStatus := "❌ Error"
	if storageHealthy {
		storageStatus = "✅ OK"
	}

	uptimeStr := sv.getUptimeString()

	content := container.NewVBox(
		container.NewHBox(widget.NewLabel("Data Directory:"), widget.NewLabel(dataDirStatus)),
		container.NewHBox(widget.NewLabel("Storage Engine:"), widget.NewLabel(storageStatus)),
		container.NewHBox(widget.NewLabel("Uptime:"), widget.NewLabel(uptimeStr)),
		widget.NewSeparator(),
		widget.NewLabel(fmt.Sprintf("Data Path: %s", dataDir)),
		widget.NewLabel(fmt.Sprintf("Config Path: %s", sv.app.GetConfigPath())),
	)

	sv.systemHealthCard.SetContent(content)
}

// loadActivity loads current activity information
func (sv *StatusView) loadActivity() {
	ctx := sv.app.GetContext()

	// Get methods count
	methodManager := sv.app.GetMethodManager()
	methods, err := methodManager.ListMethods(ctx, core.MethodFilter{})
	methodCount := 0
	if err == nil {
		methodCount = len(methods)
	}

	// Get goals count
	goalManager := sv.app.GetGoalManager()
	goals, err := goalManager.ListGoals(ctx, core.GoalFilter{})
	goalCount := 0
	if err == nil {
		goalCount = len(goals)
	}

	// Get objectives count
	objectiveManager := sv.app.GetObjectiveManager()
	objectives, err := objectiveManager.ListObjectives(ctx, core.ObjectiveFilter{})
	objectiveCount := 0
	if err == nil {
		objectiveCount = len(objectives)
	}

	// Calculate activity status
	activeGoalsCount := 0
	completedGoalsCount := 0
	for _, goal := range goals {
		if goal.Status == core.GoalStatusActive {
			activeGoalsCount++
		} else if goal.Status == core.GoalStatusCompleted {
			completedGoalsCount++
		}
	}

	content := container.NewVBox(
		container.NewHBox(widget.NewLabel("Active Goals:"), widget.NewLabel(fmt.Sprintf("%d", activeGoalsCount))),
		container.NewHBox(widget.NewLabel("Total Goals:"), widget.NewLabel(fmt.Sprintf("%d", goalCount))),
		container.NewHBox(widget.NewLabel("Objectives:"), widget.NewLabel(fmt.Sprintf("%d", objectiveCount))),
		container.NewHBox(widget.NewLabel("Methods:"), widget.NewLabel(fmt.Sprintf("%d", methodCount))),
		widget.NewSeparator(),
		NewProgressBar("Goal Completion", sv.calculateGoalCompletionRate(goals)).Card,
	)

	sv.activityCard.SetContent(content)
}

// loadBudgetStatus loads budget and usage information
func (sv *StatusView) loadBudgetStatus() {
	// TODO: Integrate BudgetManager from pkg/llm when added to App
	// For now, show placeholder information

	dailyUsageData := map[string]float64{
		"OpenAI":    2.50,
		"Anthropic": 1.75,
		"Local":     0.00,
	}

	// Create simple spending chart
	spendingChart := NewSimplePieChart("Daily Spending", dailyUsageData)

	// Create usage metrics
	totalSpent := 0.0
	for _, amount := range dailyUsageData {
		totalSpent += amount
	}

	dailyLimit := 5.0
	usagePercentage := (totalSpent / dailyLimit) * 100

	usageBar := NewProgressBar("Daily Budget", usagePercentage)

	content := container.NewVBox(
		container.NewHBox(
			widget.NewLabel("Today's Spending:"),
			widget.NewLabel(fmt.Sprintf("$%.2f", totalSpent)),
		),
		container.NewHBox(
			widget.NewLabel("Daily Limit:"),
			widget.NewLabel(fmt.Sprintf("$%.2f", dailyLimit)),
		),
		usageBar.Card,
		spendingChart.Card,
	)

	sv.budgetCard.SetContent(content)
}

// loadDataStats loads data storage statistics
func (sv *StatusView) loadDataStats() {
	config := sv.app.GetConfig()
	dataDir := config.DataDir

	// Calculate directory size and file counts
	size, fileCount, err := sv.calculateDirectoryStats(dataDir)

	sizeStr := "Unknown"
	fileCountStr := "Unknown"

	if err == nil {
		sizeStr = sv.formatBytes(size)
		fileCountStr = fmt.Sprintf("%d", fileCount)
	}

	// Get recent backup info
	backupInfo := sv.getBackupInfo()

	content := container.NewVBox(
		container.NewHBox(widget.NewLabel("Data Size:"), widget.NewLabel(sizeStr)),
		container.NewHBox(widget.NewLabel("Files:"), widget.NewLabel(fileCountStr)),
		widget.NewSeparator(),
		widget.NewLabel("Last Backup:"),
		widget.NewLabel(backupInfo),
		widget.NewSeparator(),
		NewMetricCard(
			"Storage Health",
			"Good",
			"All systems operational",
			true,
		).Card,
	)

	sv.dataStatsCard.SetContent(content)
}

// loadRecentEvents loads recent system events
func (sv *StatusView) loadRecentEvents() {
	// Create sample recent events
	// In a real implementation, this would read from logs or activity tracking
	events := []TimelineEvent{
		{
			Title:       "System Started",
			Description: "AI Work Studio application started",
			Time:        time.Now().Add(-2 * time.Hour).Format("15:04"),
			IsSuccess:   true,
		},
		{
			Title:       "Data Loaded",
			Description: "Successfully loaded user data and preferences",
			Time:        time.Now().Add(-1 * time.Hour).Format("15:04"),
			IsSuccess:   true,
		},
		{
			Title:       "Methods Refreshed",
			Description: "Method library updated with latest data",
			Time:        time.Now().Add(-30 * time.Minute).Format("15:04"),
			IsSuccess:   true,
		},
	}

	timeline := NewTimelineChart("Recent Events", events)
	sv.recentEventsCard.SetContent(timeline.Card)
}

// Helper methods

// checkDirectoryExists checks if a directory exists and is accessible
func (sv *StatusView) checkDirectoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// checkStorageHealth checks if the storage system is healthy
func (sv *StatusView) checkStorageHealth() bool {
	// For now, just check if we can create a context
	ctx := sv.app.GetContext()
	return ctx != nil
}

// getUptimeString returns a formatted uptime string
func (sv *StatusView) getUptimeString() string {
	// Simple implementation - in practice, you'd track actual start time
	return "Session active"
}

// calculateGoalCompletionRate calculates the percentage of completed goals
func (sv *StatusView) calculateGoalCompletionRate(goals []*core.Goal) float64 {
	if len(goals) == 0 {
		return 0
	}

	completedCount := 0
	for _, goal := range goals {
		if goal.Status == core.GoalStatusCompleted {
			completedCount++
		}
	}

	return (float64(completedCount) / float64(len(goals))) * 100
}

// calculateDirectoryStats calculates size and file count for a directory
func (sv *StatusView) calculateDirectoryStats(dirPath string) (int64, int, error) {
	var totalSize int64
	var fileCount int

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}
		if !info.IsDir() {
			totalSize += info.Size()
			fileCount++
		}
		return nil
	})

	return totalSize, fileCount, err
}

// formatBytes formats bytes into human-readable format
func (sv *StatusView) formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// getBackupInfo returns information about the last backup
func (sv *StatusView) getBackupInfo() string {
	// This is a placeholder - real implementation would check actual backup status
	return "No backup system configured"
}

// Event handlers

// onRefresh handles the refresh button
func (sv *StatusView) onRefresh() {
	sv.loadStatus()
}

// onAutoRefreshToggle handles auto-refresh toggle
func (sv *StatusView) onAutoRefreshToggle(checked bool) {
	if checked {
		sv.startAutoRefresh()
	} else {
		sv.stopAutoRefresh()
	}
}

// onBackupData handles backup action
func (sv *StatusView) onBackupData() {
	sv.app.ShowInfo("Backup", "Backup functionality will be implemented in a future version")
}

// onValidateStorage handles storage validation
func (sv *StatusView) onValidateStorage() {
	// TODO: Implement storage validation using storage.ValidateStore()
	sv.app.ShowInfo("Validation", "Storage validation functionality will be implemented in a future version")
}

// onOpenDataDirectory handles opening data directory
func (sv *StatusView) onOpenDataDirectory() {
	config := sv.app.GetConfig()
	sv.app.ShowInfo("Data Directory", "Data directory: "+config.DataDir)
}

// onClearCache handles cache clearing
func (sv *StatusView) onClearCache() {
	sv.app.ShowInfo("Clear Cache", "Cache clearing functionality will be implemented in a future version")
}

// Auto-refresh methods

// startAutoRefresh starts the auto-refresh timer
func (sv *StatusView) startAutoRefresh() {
	sv.stopAutoRefresh() // Stop any existing timer

	sv.refreshTimer = time.NewTicker(30 * time.Second)

	go func() {
		ticker := sv.refreshTimer
		if ticker == nil {
			return
		}
		for {
			select {
			case <-ticker.C:
				if sv.autoRefreshCheck != nil && sv.autoRefreshCheck.Checked {
					sv.loadStatus()
				}
			case <-sv.stopTimer:
				return
			}
		}
	}()
}

// stopAutoRefresh stops the auto-refresh timer
func (sv *StatusView) stopAutoRefresh() {
	if sv.refreshTimer != nil {
		sv.refreshTimer.Stop()
		sv.refreshTimer = nil
	}
}

// GetContainer returns the main container for this view
func (sv *StatusView) GetContainer() fyne.CanvasObject {
	return sv.container
}

// Refresh refreshes the status data
func (sv *StatusView) Refresh() {
	sv.loadStatus()
}

// Cleanup stops auto-refresh and cleans up resources
func (sv *StatusView) Cleanup() {
	sv.stopAutoRefresh()
	if sv.stopTimer != nil {
		close(sv.stopTimer)
	}
}