package ui

import (
	"context"
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/yourusername/ai-work-studio/internal/config"
	"github.com/yourusername/ai-work-studio/pkg/core"
	"github.com/yourusername/ai-work-studio/pkg/storage"
)

// App represents the main AI Work Studio application.
type App struct {
	fyneApp    fyne.App
	config     *config.Config
	configPath string
	mainWindow *MainWindow

	// Core components
	store            *storage.Store
	goalManager      *core.GoalManager
	objectiveManager *core.ObjectiveManager
	methodManager    *core.MethodManager
	contextManager   *core.UserContextManager

	// Application state
	ctx    context.Context
	cancel context.CancelFunc
}

// NewApp creates a new AI Work Studio application with the given configuration.
func NewApp(cfg *config.Config, configPath string) (*App, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	// Create Fyne application
	fyneApp := app.NewWithID("ai.work.studio")

	// Initialize storage
	store, err := storage.NewStore(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize core managers
	goalManager := core.NewGoalManager(store)
	objectiveManager := core.NewObjectiveManager(store)
	methodManager := core.NewMethodManager(store)
	contextManager := core.NewUserContextManager(store)

	// Create cancellable context for the application
	ctx, cancel := context.WithCancel(context.Background())

	return &App{
		fyneApp:          fyneApp,
		config:           cfg,
		configPath:       configPath,
		store:            store,
		goalManager:      goalManager,
		objectiveManager: objectiveManager,
		methodManager:    methodManager,
		contextManager:   contextManager,
		ctx:              ctx,
		cancel:           cancel,
	}, nil
}

// Run starts the application and blocks until it exits.
func (a *App) Run() error {
	// Ensure data directory exists
	if err := a.config.EnsureDataDir(); err != nil {
		return fmt.Errorf("failed to setup data directory: %w", err)
	}

	// Create and configure the main window
	mainWindow, err := NewMainWindow(a)
	if err != nil {
		return fmt.Errorf("failed to create main window: %w", err)
	}
	a.mainWindow = mainWindow

	// Apply window preferences from configuration
	a.applyWindowPreferences()

	// Set up graceful shutdown
	a.setupGracefulShutdown()

	// Show the main window
	a.mainWindow.Show()

	// Run the application (blocks until window is closed)
	a.fyneApp.Run()

	return nil
}

// Stop gracefully shuts down the application.
func (a *App) Stop() {
	// Save current window preferences before closing
	if err := a.saveWindowPreferences(); err != nil {
		log.Printf("Warning: Failed to save window preferences: %v", err)
	}

	// Cancel context to stop any background operations
	a.cancel()

	// Close storage
	if a.store != nil {
		a.store.Close()
	}

	// Quit the application
	a.fyneApp.Quit()
}

// GetConfig returns the application configuration.
func (a *App) GetConfig() *config.Config {
	return a.config
}

// GetConfigPath returns the path to the configuration file.
func (a *App) GetConfigPath() string {
	return a.configPath
}

// GetContext returns the application context.
func (a *App) GetContext() context.Context {
	return a.ctx
}

// GetGoalManager returns the goal manager.
func (a *App) GetGoalManager() *core.GoalManager {
	return a.goalManager
}

// GetObjectiveManager returns the objective manager.
func (a *App) GetObjectiveManager() *core.ObjectiveManager {
	return a.objectiveManager
}

// GetMethodManager returns the method manager.
func (a *App) GetMethodManager() *core.MethodManager {
	return a.methodManager
}

// GetUserContextManager returns the user context manager.
func (a *App) GetUserContextManager() *core.UserContextManager {
	return a.contextManager
}

// applyWindowPreferences applies saved window preferences to the main window.
func (a *App) applyWindowPreferences() {
	if a.mainWindow == nil {
		return
	}

	prefs := a.config.WindowPrefs

	// Set window size and position
	a.mainWindow.Resize(prefs.Width, prefs.Height)
	a.mainWindow.Move(prefs.X, prefs.Y)

	// Set maximized state if supported (simplified for now)
	if prefs.Maximized {
		// Note: Fyne doesn't have direct window maximization API
		// This could be implemented with platform-specific code if needed
	}

	// Set active tab
	a.mainWindow.SetActiveTab(prefs.ActiveTab)
}

// saveWindowPreferences saves current window state to configuration.
func (a *App) saveWindowPreferences() error {
	if a.mainWindow == nil {
		return nil
	}

	// Get current window state
	width, height := a.mainWindow.GetSize()
	x, y := a.mainWindow.GetPosition()
	maximized := false // We'll implement maximized detection if needed
	activeTab := a.mainWindow.GetActiveTab()

	// Update configuration with current window state
	updates := config.WindowUpdates{
		Width:     &width,
		Height:    &height,
		X:         &x,
		Y:         &y,
		Maximized: &maximized,
		ActiveTab: &activeTab,
	}

	return a.config.UpdateWindowPreferences(a.configPath, updates)
}

// setupGracefulShutdown configures graceful shutdown handlers.
func (a *App) setupGracefulShutdown() {
	// Set close intercept to save preferences before closing
	a.mainWindow.SetCloseIntercept(func() {
		a.Stop()
	})
}

// ShowError displays an error dialog to the user.
func (a *App) ShowError(title, message string) {
	if a.mainWindow != nil {
		a.mainWindow.ShowError(title, message)
	} else {
		log.Printf("ERROR: %s: %s", title, message)
	}
}

// ShowInfo displays an information dialog to the user.
func (a *App) ShowInfo(title, message string) {
	if a.mainWindow != nil {
		a.mainWindow.ShowInfo(title, message)
	} else {
		log.Printf("INFO: %s: %s", title, message)
	}
}