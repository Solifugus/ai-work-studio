package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// MainWindow represents the main application window with tab navigation.
type MainWindow struct {
	app    *App
	window fyne.Window

	// UI Components
	tabs         *container.AppTabs
	menuBar      *fyne.MainMenu
	statusBar    *widget.Label
	closeHandler func()

	// Tab views
	goalsTab      fyne.CanvasObject
	objectivesTab fyne.CanvasObject
	methodsTab    fyne.CanvasObject
	statusTab     fyne.CanvasObject
	settingsTab   fyne.CanvasObject
}

// NewMainWindow creates a new main window for the application.
func NewMainWindow(app *App) (*MainWindow, error) {
	if app == nil {
		return nil, fmt.Errorf("app cannot be nil")
	}

	window := app.fyneApp.NewWindow("AI Work Studio")
	window.SetMaster()

	mainWindow := &MainWindow{
		app:    app,
		window: window,
	}

	// Initialize UI components
	mainWindow.setupMenuBar()
	mainWindow.setupTabs()
	mainWindow.setupStatusBar()
	mainWindow.setupContent()
	mainWindow.setupShortcuts()

	// Set minimum window size
	window.SetFixedSize(false)
	window.Resize(fyne.NewSize(1200, 800))

	return mainWindow, nil
}

// setupMenuBar creates the application menu bar.
func (mw *MainWindow) setupMenuBar() {
	// File menu
	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("New Goal", func() {
			mw.showNewGoalDialog()
		}),
		fyne.NewMenuItem("Open Data Directory", func() {
			mw.openDataDirectory()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Preferences...", func() {
			mw.showPreferencesDialog()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Exit", func() {
			mw.app.Stop()
		}),
	)

	// View menu
	viewMenu := fyne.NewMenu("View",
		fyne.NewMenuItem("Goals", func() {
			mw.tabs.SelectTab(mw.tabs.Items[0])
		}),
		fyne.NewMenuItem("Objectives", func() {
			mw.tabs.SelectTab(mw.tabs.Items[1])
		}),
		fyne.NewMenuItem("Methods", func() {
			mw.tabs.SelectTab(mw.tabs.Items[2])
		}),
		fyne.NewMenuItem("Status", func() {
			mw.tabs.SelectTab(mw.tabs.Items[3])
		}),
		fyne.NewMenuItem("Settings", func() {
			mw.tabs.SelectTab(mw.tabs.Items[4])
		}),
	)

	// Help menu
	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("About", func() {
			mw.showAboutDialog()
		}),
		fyne.NewMenuItem("Documentation", func() {
			mw.showDocumentationDialog()
		}),
	)

	mw.menuBar = fyne.NewMainMenu(fileMenu, viewMenu, helpMenu)
	mw.window.SetMainMenu(mw.menuBar)
}

// setupTabs creates the main tab container with all application views.
func (mw *MainWindow) setupTabs() {
	mw.tabs = container.NewAppTabs()
	mw.tabs.SetTabLocation(container.TabLocationTop)

	// Create tab content (placeholder for now)
	mw.goalsTab = mw.createGoalsTab()
	mw.objectivesTab = mw.createObjectivesTab()
	mw.methodsTab = mw.createMethodsTab()
	mw.statusTab = mw.createStatusTab()
	mw.settingsTab = mw.createSettingsTab()

	// Add tabs to container
	mw.tabs.Append(container.NewTabItem("Goals", mw.goalsTab))
	mw.tabs.Append(container.NewTabItem("Objectives", mw.objectivesTab))
	mw.tabs.Append(container.NewTabItem("Methods", mw.methodsTab))
	mw.tabs.Append(container.NewTabItem("Status", mw.statusTab))
	mw.tabs.Append(container.NewTabItem("Settings", mw.settingsTab))
}

// setupStatusBar creates the status bar at the bottom of the window.
func (mw *MainWindow) setupStatusBar() {
	mw.statusBar = widget.NewLabel("Ready")
	mw.statusBar.TextStyle = fyne.TextStyle{Italic: true}
}

// setupContent arranges the window content with tabs and status bar.
func (mw *MainWindow) setupContent() {
	content := container.NewBorder(
		nil,          // top
		mw.statusBar, // bottom
		nil,          // left
		nil,          // right
		mw.tabs,      // center
	)

	mw.window.SetContent(content)
}

// setupShortcuts configures keyboard shortcuts for the application.
func (mw *MainWindow) setupShortcuts() {
	// For now, we'll skip custom shortcuts as Fyne's shortcut API is complex
	// and varies between versions. Basic shortcuts like Ctrl+N, Ctrl+Q work through menus.
	// This can be enhanced in a future version with proper shortcut implementation.
}

// Tab creation methods
func (mw *MainWindow) createGoalsTab() fyne.CanvasObject {
	// Create the full-featured Goals Management View
	goalsView := NewGoalsView(mw.app, mw.window)
	return goalsView.GetContainer()
}

func (mw *MainWindow) createObjectivesTab() fyne.CanvasObject {
	objectivesView := NewObjectivesView(mw.app, mw.window)
	return objectivesView.GetContainer()
}

func (mw *MainWindow) createMethodsTab() fyne.CanvasObject {
	methodsView := NewMethodsView(mw.app, mw.window)
	return methodsView.GetContainer()
}

func (mw *MainWindow) createStatusTab() fyne.CanvasObject {
	statusView := NewStatusView(mw.app, mw.window)
	return statusView.GetContainer()
}

func (mw *MainWindow) createSettingsTab() fyne.CanvasObject {
	// Auto-approve checkbox
	autoApproveCheck := widget.NewCheck("Auto-approve low-risk decisions", func(checked bool) {
		mw.app.config.Preferences.AutoApprove = checked
		mw.app.config.Save(mw.app.configPath)
	})
	autoApproveCheck.SetChecked(mw.app.config.Preferences.AutoApprove)

	// Verbose output checkbox
	verboseCheck := widget.NewCheck("Verbose output", func(checked bool) {
		mw.app.config.Preferences.VerboseOutput = checked
		mw.app.config.Save(mw.app.configPath)
	})
	verboseCheck.SetChecked(mw.app.config.Preferences.VerboseOutput)

	content := container.NewVBox(
		widget.NewLabel("Settings"),
		widget.NewSeparator(),
		widget.NewLabel("User Preferences"),
		autoApproveCheck,
		verboseCheck,
		widget.NewSeparator(),
		widget.NewLabel("More settings will be available in future versions."),
	)
	return container.NewScroll(content)
}

// Dialog methods
func (mw *MainWindow) showNewGoalDialog() {
	titleEntry := widget.NewEntry()
	titleEntry.SetPlaceHolder("Enter goal title...")

	descEntry := widget.NewMultiLineEntry()
	descEntry.SetPlaceHolder("Enter goal description...")

	prioritySelect := widget.NewSelect([]string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}, nil)
	prioritySelect.SetSelected("5") // Default priority

	form := container.NewVBox(
		widget.NewLabel("Create New Goal"),
		widget.NewSeparator(),
		widget.NewLabel("Title:"),
		titleEntry,
		widget.NewLabel("Description:"),
		descEntry,
		widget.NewLabel("Priority (1-10):"),
		prioritySelect,
	)

	dialog.ShowCustomConfirm(
		"New Goal",
		"Create",
		"Cancel",
		form,
		func(confirm bool) {
			if confirm && titleEntry.Text != "" {
				// TODO: Create goal using goal manager
				mw.showInfo("Goal Created", "Goal '"+titleEntry.Text+"' has been created.")
			}
		},
		mw.window,
	)
}

func (mw *MainWindow) showPreferencesDialog() {
	mw.showInfo("Preferences", "Preference editing will be implemented in a future version.")
}

func (mw *MainWindow) openDataDirectory() {
	mw.showInfo("Data Directory", "Data directory: "+mw.app.config.DataDir)
}

func (mw *MainWindow) showAboutDialog() {
	about := fmt.Sprintf(`AI Work Studio v1.0.0

A goal-directed autonomous agent system built in Go.

Data Directory: %s
Configuration: %s

Built with Fyne v2 and Go.`, mw.app.config.DataDir, mw.app.configPath)

	dialog.ShowInformation("About AI Work Studio", about, mw.window)
}

func (mw *MainWindow) showDocumentationDialog() {
	mw.showInfo("Documentation", "Documentation will be available in a future version.")
}

// Public methods for window management
func (mw *MainWindow) Show() {
	mw.window.Show()
}

func (mw *MainWindow) Hide() {
	mw.window.Hide()
}

func (mw *MainWindow) Close() {
	if mw.closeHandler != nil {
		mw.closeHandler()
	}
	mw.window.Close()
}

func (mw *MainWindow) SetCloseIntercept(handler func()) {
	mw.closeHandler = handler
	mw.window.SetCloseIntercept(handler)
}

func (mw *MainWindow) Resize(width, height int) {
	mw.window.Resize(fyne.NewSize(float32(width), float32(height)))
}

func (mw *MainWindow) Move(x, y int) {
	mw.window.SetFixedSize(false) // Allow moving
	// Note: Fyne doesn't have a direct Move method, but we can set position during resize
}

func (mw *MainWindow) GetSize() (int, int) {
	size := mw.window.Canvas().Size()
	return int(size.Width), int(size.Height)
}

func (mw *MainWindow) GetPosition() (int, int) {
	// Fyne doesn't provide direct access to window position
	// Return default values for now
	return 100, 100
}

func (mw *MainWindow) SetActiveTab(index int) {
	items := mw.tabs.Items
	if index >= 0 && index < len(items) {
		mw.tabs.SelectTab(items[index])
	}
}

func (mw *MainWindow) GetActiveTab() int {
	selected := mw.tabs.Selected()
	items := mw.tabs.Items
	for i, item := range items {
		if item == selected {
			return i
		}
	}
	return 0
}

func (mw *MainWindow) UpdateStatus(message string) {
	mw.statusBar.SetText(message)
}

func (mw *MainWindow) ShowError(title, message string) {
	dialog.ShowError(fmt.Errorf("%s", message), mw.window)
}

func (mw *MainWindow) ShowInfo(title, message string) {
	dialog.ShowInformation(title, message, mw.window)
}

func (mw *MainWindow) showInfo(title, message string) {
	dialog.ShowInformation(title, message, mw.window)
}