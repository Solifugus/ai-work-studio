// Package ui provides the Fyne-based graphical user interface for AI Work Studio.
//
// This package implements the main application window with tabbed navigation
// and all UI components needed to interact with goals, objectives, methods,
// and system status.
//
// Key Components:
//   - App: Application lifecycle management and configuration
//   - MainWindow: Main application window with tab navigation
//   - Tab Views: Individual views for Goals, Objectives, Methods, Status, Settings
//
// The UI follows the design principles of simplicity and minimal context,
// displaying only essential information and loading details on demand.
//
// Architecture:
//   - Uses Fyne's TabContainer for main navigation
//   - Persists window preferences in configuration
//   - Integrates with core components through minimal interfaces
//   - Supports keyboard shortcuts for common actions
//
// Example Usage:
//
//	config := config.DefaultConfig()
//	app := ui.NewApp(config)
//	app.Run() // Blocks until application closes
package ui