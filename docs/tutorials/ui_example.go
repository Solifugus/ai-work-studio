// Package tutorials provides example code demonstrating AI Work Studio usage.
package tutorials

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/Solifugus/ai-work-studio/pkg/core"
	"github.com/Solifugus/ai-work-studio/pkg/ui"
)

// CustomGoalWidget demonstrates how to create custom widgets for AI Work Studio.
type CustomGoalWidget struct {
	widget.BaseWidget
	goal     *core.Goal
	onUpdate func(*core.Goal)
	onDelete func(string)

	titleLabel    *widget.Label
	statusBadge   *widget.Label
	priorityBar   *widget.ProgressBar
	editButton    *widget.Button
	deleteButton  *widget.Button
}

// NewCustomGoalWidget creates a new custom goal widget.
func NewCustomGoalWidget(goal *core.Goal, onUpdate func(*core.Goal), onDelete func(string)) *CustomGoalWidget {
	w := &CustomGoalWidget{
		goal:     goal,
		onUpdate: onUpdate,
		onDelete: onDelete,
	}
	w.ExtendBaseWidget(w)
	w.initUI()
	return w
}

// initUI initializes the widget's UI components.
func (w *CustomGoalWidget) initUI() {
	w.titleLabel = widget.NewLabel(w.goal.Title)
	w.titleLabel.Wrapping = fyne.TextWrapWord

	w.statusBadge = widget.NewLabel(string(w.goal.Status))
	w.updateStatusBadge()

	w.priorityBar = widget.NewProgressBar()
	w.priorityBar.Min = 0
	w.priorityBar.Max = 10
	w.priorityBar.SetValue(float64(w.goal.Priority))

	w.editButton = widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), w.onEditClicked)
	w.deleteButton = widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), w.onDeleteClicked)
}

// updateStatusBadge updates the status badge appearance based on goal status.
func (w *CustomGoalWidget) updateStatusBadge() {
	switch w.goal.Status {
	case core.GoalStatusActive:
		w.statusBadge.Text = "🟢 Active"
	case core.GoalStatusPaused:
		w.statusBadge.Text = "⏸️ Paused"
	case core.GoalStatusCompleted:
		w.statusBadge.Text = "✅ Completed"
	case core.GoalStatusArchived:
		w.statusBadge.Text = "📦 Archived"
	default:
		w.statusBadge.Text = string(w.goal.Status)
	}
	w.statusBadge.Refresh()
}

// onEditClicked handles the edit button click.
func (w *CustomGoalWidget) onEditClicked() {
	// In a real implementation, this would open an edit dialog
	fmt.Printf("Edit goal: %s\n", w.goal.Title)

	// Example: Toggle status for demo purposes
	switch w.goal.Status {
	case core.GoalStatusActive:
		w.goal.Status = core.GoalStatusPaused
	case core.GoalStatusPaused:
		w.goal.Status = core.GoalStatusActive
	default:
		w.goal.Status = core.GoalStatusActive
	}

	w.updateStatusBadge()
	if w.onUpdate != nil {
		w.onUpdate(w.goal)
	}
}

// onDeleteClicked handles the delete button click.
func (w *CustomGoalWidget) onDeleteClicked() {
	if w.onDelete != nil {
		w.onDelete(w.goal.ID)
	}
}

// CreateRenderer creates the widget renderer.
func (w *CustomGoalWidget) CreateRenderer() fyne.WidgetRenderer {
	headerContainer := container.NewHBox(
		w.titleLabel,
		widget.NewSeparator(),
		w.statusBadge,
	)

	priorityContainer := container.NewHBox(
		widget.NewLabel("Priority:"),
		w.priorityBar,
		widget.NewLabel(fmt.Sprintf("%d/10", w.goal.Priority)),
	)

	buttonsContainer := container.NewHBox(
		w.editButton,
		w.deleteButton,
	)

	content := container.NewVBox(
		headerContainer,
		priorityContainer,
		buttonsContainer,
	)

	return widget.NewSimpleRenderer(content)
}

// UIExample demonstrates building custom UI components for AI Work Studio.
func UIExample() error {
	fmt.Println("AI Work Studio - UI Components Example")
	fmt.Println("======================================")

	// Create a Fyne app
	myApp := app.New()
	myApp.Settings().SetTheme(theme.DefaultTheme())

	window := myApp.NewWindow("AI Work Studio - UI Example")
	window.Resize(fyne.NewSize(800, 600))

	// Example 1: Custom goal widget
	fmt.Println("\n1. Creating custom goal widgets...")

	sampleGoals := []*core.Goal{
		{
			ID:          "goal-1",
			Title:       "Learn Fyne UI Development",
			Description: "Master the Fyne framework for cross-platform GUI development",
			Status:      core.GoalStatusActive,
			Priority:    8,
		},
		{
			ID:          "goal-2",
			Title:       "Implement Custom Widgets",
			Description: "Create reusable custom widgets for the AI Work Studio interface",
			Status:      core.GoalStatusActive,
			Priority:    7,
		},
		{
			ID:          "goal-3",
			Title:       "Optimize UI Performance",
			Description: "Ensure the UI remains responsive with large datasets",
			Status:      core.GoalStatusPaused,
			Priority:    6,
		},
	}

	// Create a container for goals
	goalsContainer := container.NewVBox()

	// Callback functions
	onGoalUpdate := func(goal *core.Goal) {
		fmt.Printf("Goal updated: %s (Status: %s)\n", goal.Title, goal.Status)
	}

	onGoalDelete := func(goalID string) {
		fmt.Printf("Goal deleted: %s\n", goalID)
		// In a real app, this would remove the widget from the container
	}

	// Add goal widgets to container
	for _, goal := range sampleGoals {
		goalWidget := NewCustomGoalWidget(goal, onGoalUpdate, onGoalDelete)
		goalsContainer.Add(goalWidget)
		goalsContainer.Add(widget.NewSeparator()) // Visual separation
	}

	// Example 2: Advanced layout with tabs
	fmt.Println("\n2. Creating tabbed interface...")

	// Goals tab
	goalsScrollable := container.NewScroll(goalsContainer)
	goalsScrollable.SetMinSize(fyne.NewSize(400, 300))

	// Objectives tab (placeholder)
	objectivesContent := container.NewVBox(
		widget.NewLabel("Objectives View"),
		widget.NewButton("Add Objective", func() {
			fmt.Println("Add objective clicked")
		}),
		widget.NewList(
			func() int { return 5 },
			func() fyne.CanvasObject {
				return container.NewHBox(
					widget.NewLabel("Objective"),
					widget.NewButton("Complete", nil),
				)
			},
			func(id widget.ListItemID, item fyne.CanvasObject) {
				item.(*fyne.Container).Objects[0].(*widget.Label).SetText(
					fmt.Sprintf("Objective %d", id+1))
			},
		),
	)

	// Methods tab (placeholder)
	methodsContent := container.NewVBox(
		widget.NewLabel("Methods View"),
		widget.NewTable(
			func() (int, int) { return 3, 3 }, // 3 rows, 3 columns
			func() fyne.CanvasObject {
				return widget.NewLabel("Cell")
			},
			func(id widget.TableCellID, cell fyne.CanvasObject) {
				label := cell.(*widget.Label)
				if id.Row == 0 {
					switch id.Col {
					case 0:
						label.SetText("Method")
					case 1:
						label.SetText("Success Rate")
					case 2:
						label.SetText("Last Used")
					}
				} else {
					switch id.Col {
					case 0:
						label.SetText(fmt.Sprintf("Method %d", id.Row))
					case 1:
						label.SetText(fmt.Sprintf("%.1f%%", 85.5+float64(id.Row)*5))
					case 2:
						label.SetText("2024-01-15")
					}
				}
			},
		),
	)

	// Status tab with charts (simplified)
	statusContent := container.NewVBox(
		widget.NewLabel("System Status"),
		container.NewGridWithColumns(2,
			widget.NewCard("Active Goals", "", widget.NewLabel("3")),
			widget.NewCard("Completed Today", "", widget.NewLabel("2")),
			widget.NewCard("Methods Learned", "", widget.NewLabel("15")),
			widget.NewCard("Success Rate", "", widget.NewLabel("87.3%")),
		),
		widget.NewSeparator(),
		widget.NewProgressBarInfinite(), // Simulating activity
	)

	// Create tabs
	tabs := container.NewAppTabs(
		container.NewTabItem("Goals", goalsScrollable),
		container.NewTabItem("Objectives", objectivesContent),
		container.NewTabItem("Methods", methodsContent),
		container.NewTabItem("Status", statusContent),
	)

	// Example 3: Menu and toolbar
	fmt.Println("\n3. Adding menu and toolbar...")

	// Create menu
	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("New Goal", func() {
			showNewGoalDialog(window)
		}),
		fyne.NewMenuItem("Import", func() {
			fmt.Println("Import clicked")
		}),
		fyne.NewMenuItem("Export", func() {
			fmt.Println("Export clicked")
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Exit", func() {
			myApp.Quit()
		}),
	)

	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("Documentation", func() {
			fmt.Println("Opening documentation...")
		}),
		fyne.NewMenuItem("About", func() {
			showAboutDialog(window)
		}),
	)

	mainMenu := fyne.NewMainMenu(fileMenu, helpMenu)
	window.SetMainMenu(mainMenu)

	// Create toolbar
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentAddIcon(), func() {
			showNewGoalDialog(window)
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.ViewRefreshIcon(), func() {
			fmt.Println("Refresh clicked")
		}),
		widget.NewToolbarAction(theme.SettingsIcon(), func() {
			showSettingsDialog(window)
		}),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(theme.HelpIcon(), func() {
			showAboutDialog(window)
		}),
	)

	// Combine everything
	content := container.NewBorder(
		toolbar, // Top
		nil,     // Bottom
		nil,     // Left
		nil,     // Right
		tabs,    // Center
	)

	window.SetContent(content)

	// Show keyboard shortcuts info
	fmt.Println("\n4. Available keyboard shortcuts:")
	fmt.Println("   Ctrl+N - New Goal")
	fmt.Println("   Ctrl+R - Refresh")
	fmt.Println("   F1 - Help")

	// Note: In a real application, you would set up actual keyboard shortcuts here

	fmt.Println("\n✅ UI example setup complete!")
	fmt.Println("Close the window to continue...")

	// Show and run (this blocks until window is closed)
	window.ShowAndRun()

	return nil
}

// showNewGoalDialog displays a dialog for creating a new goal.
func showNewGoalDialog(parent fyne.Window) {
	titleEntry := widget.NewEntry()
	titleEntry.SetPlaceHolder("Goal title...")

	descEntry := widget.NewMultiLineEntry()
	descEntry.SetPlaceHolder("Goal description...")

	prioritySlider := widget.NewSlider(1, 10)
	prioritySlider.Value = 5
	prioritySlider.Step = 1

	statusSelect := widget.NewSelect(
		[]string{"active", "paused", "completed", "archived"},
		func(value string) {
			fmt.Printf("Status selected: %s\n", value)
		},
	)
	statusSelect.SetSelected("active")

	form := container.NewVBox(
		widget.NewLabel("Create New Goal"),
		widget.NewSeparator(),
		widget.NewLabel("Title:"),
		titleEntry,
		widget.NewLabel("Description:"),
		descEntry,
		container.NewHBox(
			widget.NewLabel("Priority:"),
			prioritySlider,
			widget.NewLabel(fmt.Sprintf("%.0f", prioritySlider.Value)),
		),
		container.NewHBox(
			widget.NewLabel("Status:"),
			statusSelect,
		),
	)

	dialog := dialog.NewCustomConfirm(
		"New Goal",
		"Create",
		"Cancel",
		form,
		func(confirmed bool) {
			if confirmed {
				fmt.Printf("Creating goal: %s (Priority: %.0f, Status: %s)\n",
					titleEntry.Text, prioritySlider.Value, statusSelect.Selected)
			}
		},
		parent,
	)

	dialog.Show()
}

// showAboutDialog displays an about dialog.
func showAboutDialog(parent fyne.Window) {
	content := container.NewVBox(
		widget.NewLabel("AI Work Studio"),
		widget.NewLabel("Version 1.0.0"),
		widget.NewSeparator(),
		widget.NewLabel("A goal-directed autonomous agent system"),
		widget.NewLabel("built with simplicity and effectiveness in mind."),
		widget.NewSeparator(),
		widget.NewLabel("Built with Go and Fyne"),
		widget.NewHyperlink("Documentation", nil),
		widget.NewHyperlink("GitHub Repository", nil),
	)

	dialog := dialog.NewCustom("About AI Work Studio", "Close", content, parent)
	dialog.Show()
}

// showSettingsDialog displays a settings dialog.
func showSettingsDialog(parent fyne.Window) {
	themeSelect := widget.NewSelect(
		[]string{"Light", "Dark", "Auto"},
		func(value string) {
			fmt.Printf("Theme selected: %s\n", value)
		},
	)
	themeSelect.SetSelected("Auto")

	autoSaveCheck := widget.NewCheck("Enable auto-save", func(checked bool) {
		fmt.Printf("Auto-save: %v\n", checked)
	})
	autoSaveCheck.SetChecked(true)

	logLevelSelect := widget.NewSelect(
		[]string{"DEBUG", "INFO", "WARNING", "ERROR"},
		func(value string) {
			fmt.Printf("Log level: %s\n", value)
		},
	)
	logLevelSelect.SetSelected("INFO")

	form := container.NewVBox(
		widget.NewLabel("Settings"),
		widget.NewSeparator(),
		container.NewHBox(
			widget.NewLabel("Theme:"),
			themeSelect,
		),
		autoSaveCheck,
		container.NewHBox(
			widget.NewLabel("Log Level:"),
			logLevelSelect,
		),
		widget.NewSeparator(),
		widget.NewButton("Reset to Defaults", func() {
			fmt.Println("Reset to defaults")
		}),
	)

	dialog := dialog.NewCustomConfirm(
		"Settings",
		"Apply",
		"Cancel",
		form,
		func(confirmed bool) {
			if confirmed {
				fmt.Println("Settings applied")
			}
		},
		parent,
	)

	dialog.Show()
}

// ResponsiveLayoutExample demonstrates responsive UI layouts.
func ResponsiveLayoutExample() {
	fmt.Println("Responsive Layout Example")
	fmt.Println("========================")

	// This would demonstrate:
	// 1. Adaptive layouts for different screen sizes
	// 2. Collapsible panels and sidebars
	// 3. Dynamic content arrangement
	// 4. Mobile-friendly interactions

	fmt.Println("(This would demonstrate responsive design patterns)")
}

// CustomThemeExample demonstrates custom theming.
func CustomThemeExample() {
	fmt.Println("Custom Theme Example")
	fmt.Println("===================")

	// This would demonstrate:
	// 1. Creating custom color schemes
	// 2. Custom fonts and typography
	// 3. Icon customization
	// 4. Brand-specific styling

	fmt.Println("(This would demonstrate custom theming)")
}

// DataVisualizationExample demonstrates charts and data visualization.
func DataVisualizationExample() {
	fmt.Println("Data Visualization Example")
	fmt.Println("==========================")

	// This would demonstrate:
	// 1. Goal progress charts
	// 2. Method effectiveness graphs
	// 3. Timeline visualizations
	// 4. Interactive data exploration

	fmt.Println("(This would demonstrate data visualization widgets)")
}