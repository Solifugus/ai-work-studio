package ui

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/yourusername/ai-work-studio/pkg/core"
)

// ObjectivesView represents the objectives monitoring interface with status indicators,
// progress tracking, real-time updates, and comprehensive filtering capabilities.
type ObjectivesView struct {
	app    *App
	parent fyne.Window

	// Main container
	container *fyne.Container

	// UI Components
	toolbar       *fyne.Container
	searchEntry   *widget.Entry
	filterSelect  *widget.Select
	sortSelect    *widget.Select
	objectivesList *widget.List
	statusLabel   *widget.Label
	refreshButton *widget.Button

	// Data
	objectives     []*core.Objective
	filteredObjectives []*core.Objective

	// State
	searchFilter   string
	statusFilter   string // "all", "active", "completed", "failed", "pending", "paused"
	sortMode       string
	selectedObjectiveID string

	// Real-time updates
	updateTimer *time.Timer
	refreshChan chan bool
	stopRefresh chan bool
}

// ObjectiveListItem represents an objective display item with progress and status indicators.
type ObjectiveListItem struct {
	Objective   *core.Objective
	Container   *fyne.Container
	StatusIcon  *widget.Icon
	TitleLabel  *widget.Label
	StatusLabel *widget.Label
	ProgressBar *widget.ProgressBar
	DetailsButton *widget.Button
}

// NewObjectivesView creates a new objectives monitoring interface.
func NewObjectivesView(app *App, parent fyne.Window) *ObjectivesView {
	ov := &ObjectivesView{
		app:         app,
		parent:      parent,
		refreshChan: make(chan bool, 1),
		stopRefresh: make(chan bool, 1),
	}

	ov.buildUI()
	ov.loadObjectives()
	ov.startAutoRefresh()

	return ov
}

// GetContainer returns the main container for this view.
func (ov *ObjectivesView) GetContainer() *fyne.Container {
	return ov.container
}

// buildUI constructs the complete objectives view interface.
func (ov *ObjectivesView) buildUI() {
	ov.buildStatusBar()
	ov.buildToolbar()
	ov.buildObjectivesList()

	// Main layout using border container
	ov.container = container.NewBorder(
		ov.toolbar,    // top
		ov.statusLabel, // bottom
		nil,           // left
		nil,           // right
		container.NewScroll(ov.objectivesList), // center
	)
}

// buildToolbar creates the search, filter, and action controls.
func (ov *ObjectivesView) buildToolbar() {
	// Search entry
	ov.searchEntry = widget.NewEntry()
	ov.searchEntry.SetPlaceHolder("Search objectives...")
	ov.searchEntry.OnChanged = func(text string) {
		ov.searchFilter = text
		ov.applyFiltersAndSort()
	}

	// Status filter dropdown
	ov.filterSelect = widget.NewSelect([]string{
		"all",
		"active",
		"pending",
		"in_progress",
		"completed",
		"failed",
		"paused",
	}, func(value string) {
		ov.statusFilter = value
		ov.applyFiltersAndSort()
	})
	ov.filterSelect.SetSelected("all")

	// Sort dropdown
	ov.sortSelect = widget.NewSelect([]string{
		"priority",
		"title",
		"status",
		"created",
		"updated",
	}, func(value string) {
		ov.sortMode = value
		ov.applyFiltersAndSort()
	})
	ov.sortSelect.SetSelected("priority")

	// Refresh button
	ov.refreshButton = widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		ov.loadObjectives()
	})

	// New objective button
	newObjectiveButton := widget.NewButtonWithIcon("New Objective", theme.ContentAddIcon(), func() {
		ov.showNewObjectiveDialog()
	})

	// Search container
	searchContainer := container.NewBorder(
		nil, nil,
		widget.NewLabel("Search:"), nil,
		ov.searchEntry,
	)

	// Filter container
	filterContainer := container.NewBorder(
		nil, nil,
		widget.NewLabel("Filter:"), nil,
		ov.filterSelect,
	)

	// Sort container
	sortContainer := container.NewBorder(
		nil, nil,
		widget.NewLabel("Sort:"), nil,
		ov.sortSelect,
	)

	// Combine into toolbar
	ov.toolbar = container.NewVBox(
		container.NewHBox(
			searchContainer,
			widget.NewSeparator(),
			filterContainer,
			widget.NewSeparator(),
			sortContainer,
			widget.NewSeparator(),
			ov.refreshButton,
			newObjectiveButton,
		),
		widget.NewSeparator(),
	)
}

// buildObjectivesList creates the main objectives display list.
func (ov *ObjectivesView) buildObjectivesList() {
	ov.objectivesList = widget.NewList(
		func() int {
			return len(ov.filteredObjectives)
		},
		func() fyne.CanvasObject {
			return ov.createObjectiveListItem()
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			ov.updateObjectiveListItem(id, obj)
		},
	)

	ov.objectivesList.OnSelected = func(id widget.ListItemID) {
		if id >= 0 && id < len(ov.filteredObjectives) {
			objective := ov.filteredObjectives[id]
			ov.selectedObjectiveID = objective.ID
			ov.showObjectiveDetails(objective)
		}
	}
}

// createObjectiveListItem creates a template for objective list items.
func (ov *ObjectivesView) createObjectiveListItem() fyne.CanvasObject {
	// Status icon
	statusIcon := widget.NewIcon(theme.InfoIcon())
	statusIcon.Resize(fyne.NewSize(16, 16))

	// Title label
	titleLabel := widget.NewLabel("Objective Title")
	titleLabel.TextStyle.Bold = true

	// Status label
	statusLabel := widget.NewLabel("Status")
	statusLabel.TextStyle.Italic = true

	// Progress bar (for in-progress objectives)
	progressBar := widget.NewProgressBar()
	progressBar.Hide() // Hidden by default

	// Details button
	detailsButton := widget.NewButtonWithIcon("", theme.InfoIcon(), nil)
	detailsButton.Resize(fyne.NewSize(24, 24))

	// Priority indicator
	priorityLabel := widget.NewLabel("P1")
	priorityLabel.TextStyle.Bold = true

	// Time labels
	timeLabel := widget.NewLabel("Created: 2 hours ago")

	// Layout: icon + content + actions
	contentBox := container.NewVBox(
		container.NewHBox(titleLabel, priorityLabel),
		statusLabel,
		timeLabel,
		progressBar,
	)

	itemContainer := container.NewBorder(
		nil, nil,
		statusIcon,
		detailsButton,
		contentBox,
	)

	return container.NewVBox(
		itemContainer,
		widget.NewSeparator(),
	)
}

// updateObjectiveListItem updates a list item with objective data.
func (ov *ObjectivesView) updateObjectiveListItem(id widget.ListItemID, obj fyne.CanvasObject) {
	if id >= len(ov.filteredObjectives) {
		return
	}

	objective := ov.filteredObjectives[id]
	container := obj.(*fyne.Container).Objects[0].(*fyne.Container)

	// Get components
	statusIcon := container.Objects[0].(*widget.Icon)
	contentBox := container.Objects[1].(*fyne.Container)
	detailsButton := container.Objects[2].(*widget.Button)

	// Get content components
	titleContainer := contentBox.Objects[0].(*fyne.Container)
	titleLabel := titleContainer.Objects[0].(*widget.Label)
	priorityLabel := titleContainer.Objects[1].(*widget.Label)
	statusLabel := contentBox.Objects[1].(*widget.Label)
	timeLabel := contentBox.Objects[2].(*widget.Label)
	progressBar := contentBox.Objects[3].(*widget.ProgressBar)

	// Update title and priority
	titleLabel.SetText(objective.Title)
	priorityLabel.SetText(fmt.Sprintf("P%d", objective.Priority))

	// Update status icon and label
	ov.updateStatusIconAndLabel(statusIcon, statusLabel, objective.Status)

	// Update progress bar for in-progress objectives
	ov.updateProgressBar(progressBar, objective)

	// Update time information
	ov.updateTimeLabel(timeLabel, objective)

	// Set details button action
	detailsButton.OnTapped = func() {
		ov.showObjectiveDetails(objective)
	}
}

// updateStatusIconAndLabel updates the status icon and label based on objective status.
func (ov *ObjectivesView) updateStatusIconAndLabel(icon *widget.Icon, label *widget.Label, status core.ObjectiveStatus) {
	switch status {
	case core.ObjectiveStatusPending:
		icon.SetResource(theme.MediaPauseIcon())
		label.SetText("Pending")
	case core.ObjectiveStatusInProgress:
		icon.SetResource(theme.MediaPlayIcon())
		label.SetText("In Progress")
	case core.ObjectiveStatusCompleted:
		icon.SetResource(theme.ConfirmIcon())
		label.SetText("Completed")
	case core.ObjectiveStatusFailed:
		icon.SetResource(theme.ErrorIcon())
		label.SetText("Failed")
	case core.ObjectiveStatusPaused:
		icon.SetResource(theme.MediaPauseIcon())
		label.SetText("Paused")
	default:
		icon.SetResource(theme.InfoIcon())
		label.SetText("Unknown")
	}
}

// updateProgressBar shows progress for in-progress objectives.
func (ov *ObjectivesView) updateProgressBar(progressBar *widget.ProgressBar, objective *core.Objective) {
	if objective.Status == core.ObjectiveStatusInProgress {
		// Calculate progress based on time elapsed (simple heuristic)
		// In a real implementation, this could be based on actual progress metrics
		if objective.StartedAt != nil {
			elapsed := time.Since(*objective.StartedAt)
			// Assume 1 hour is 100% for demo purposes
			progress := elapsed.Hours()
			if progress > 1.0 {
				progress = 1.0
			}
			progressBar.SetValue(progress)
			progressBar.Show()
		} else {
			progressBar.SetValue(0.1) // Just started
			progressBar.Show()
		}
	} else {
		progressBar.Hide()
	}
}

// updateTimeLabel shows relevant time information for the objective.
func (ov *ObjectivesView) updateTimeLabel(label *widget.Label, objective *core.Objective) {
	now := time.Now()

	switch objective.Status {
	case core.ObjectiveStatusInProgress:
		if objective.StartedAt != nil {
			elapsed := now.Sub(*objective.StartedAt)
			label.SetText(fmt.Sprintf("Started: %s ago", formatDuration(elapsed)))
		} else {
			label.SetText("Started: just now")
		}
	case core.ObjectiveStatusCompleted, core.ObjectiveStatusFailed:
		if objective.CompletedAt != nil {
			completed := now.Sub(*objective.CompletedAt)
			label.SetText(fmt.Sprintf("Finished: %s ago", formatDuration(completed)))
		} else {
			label.SetText("Finished: recently")
		}
	default:
		created := now.Sub(objective.CreatedAt)
		label.SetText(fmt.Sprintf("Created: %s ago", formatDuration(created)))
	}
}

// formatDuration formats a duration into a human-readable string.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d hours", int(d.Hours()))
	}
	return fmt.Sprintf("%d days", int(d.Hours()/24))
}

// buildStatusBar creates the bottom status information bar.
func (ov *ObjectivesView) buildStatusBar() {
	ov.statusLabel = widget.NewLabel("Ready")
	ov.updateStatusBar()
}

// updateStatusBar updates the status bar with current objective counts.
func (ov *ObjectivesView) updateStatusBar() {
	// Check if statusLabel is initialized
	if ov.statusLabel == nil {
		return
	}

	total := len(ov.objectives)
	filtered := len(ov.filteredObjectives)

	if total == 0 {
		ov.statusLabel.SetText("No objectives found")
		return
	}

	if filtered == total {
		ov.statusLabel.SetText(fmt.Sprintf("%d objectives", total))
	} else {
		ov.statusLabel.SetText(fmt.Sprintf("Showing %d of %d objectives", filtered, total))
	}
}

// loadObjectives loads all objectives from storage.
func (ov *ObjectivesView) loadObjectives() {
	ctx := ov.app.GetContext()
	manager := ov.app.GetObjectiveManager()

	objectives, err := manager.ListObjectives(ctx, core.ObjectiveFilter{})
	if err != nil {
		log.Printf("Error loading objectives: %v", err)
		ov.statusLabel.SetText("Error loading objectives")
		return
	}

	ov.objectives = objectives
	ov.applyFiltersAndSort()
}

// applyFiltersAndSort filters and sorts objectives based on current settings.
func (ov *ObjectivesView) applyFiltersAndSort() {
	filtered := make([]*core.Objective, 0, len(ov.objectives))

	for _, obj := range ov.objectives {
		// Apply search filter
		if ov.searchFilter != "" {
			searchLower := strings.ToLower(ov.searchFilter)
			titleLower := strings.ToLower(obj.Title)
			descLower := strings.ToLower(obj.Description)

			if !strings.Contains(titleLower, searchLower) && !strings.Contains(descLower, searchLower) {
				continue
			}
		}

		// Apply status filter
		if ov.statusFilter != "all" {
			switch ov.statusFilter {
			case "active":
				if obj.Status != core.ObjectiveStatusInProgress && obj.Status != core.ObjectiveStatusPending {
					continue
				}
			case "pending":
				if obj.Status != core.ObjectiveStatusPending {
					continue
				}
			case "in_progress":
				if obj.Status != core.ObjectiveStatusInProgress {
					continue
				}
			case "completed":
				if obj.Status != core.ObjectiveStatusCompleted {
					continue
				}
			case "failed":
				if obj.Status != core.ObjectiveStatusFailed {
					continue
				}
			case "paused":
				if obj.Status != core.ObjectiveStatusPaused {
					continue
				}
			}
		}

		filtered = append(filtered, obj)
	}

	// Apply sorting
	ov.sortObjectives(filtered)

	ov.filteredObjectives = filtered
	ov.updateStatusBar()

	// Only refresh if the list is initialized
	if ov.objectivesList != nil {
		ov.objectivesList.Refresh()
	}
}

// sortObjectives sorts objectives based on the selected sort mode.
func (ov *ObjectivesView) sortObjectives(objectives []*core.Objective) {
	sort.Slice(objectives, func(i, j int) bool {
		switch ov.sortMode {
		case "priority":
			// Higher priority first
			return objectives[i].Priority > objectives[j].Priority
		case "title":
			return objectives[i].Title < objectives[j].Title
		case "status":
			return string(objectives[i].Status) < string(objectives[j].Status)
		case "created":
			return objectives[i].CreatedAt.After(objectives[j].CreatedAt)
		case "updated":
			// Use started time if available, otherwise created time
			iTime := objectives[i].CreatedAt
			if objectives[i].StartedAt != nil {
				iTime = *objectives[i].StartedAt
			}
			jTime := objectives[j].CreatedAt
			if objectives[j].StartedAt != nil {
				jTime = *objectives[j].StartedAt
			}
			return iTime.After(jTime)
		default:
			return objectives[i].Priority > objectives[j].Priority
		}
	})
}

// showObjectiveDetails displays detailed information about an objective.
func (ov *ObjectivesView) showObjectiveDetails(objective *core.Objective) {
	content := ov.buildObjectiveDetailsContent(objective)

	d := dialog.NewCustom(
		fmt.Sprintf("Objective: %s", objective.Title),
		"Close",
		content,
		ov.parent,
	)
	d.Resize(fyne.NewSize(600, 400))
	d.Show()
}

// buildObjectiveDetailsContent creates the detailed view content for an objective.
func (ov *ObjectivesView) buildObjectiveDetailsContent(objective *core.Objective) *fyne.Container {
	// Basic information
	infoGrid := container.NewGridWithColumns(2,
		widget.NewLabel("ID:"), widget.NewLabel(objective.ID),
		widget.NewLabel("Title:"), widget.NewLabel(objective.Title),
		widget.NewLabel("Status:"), widget.NewLabel(string(objective.Status)),
		widget.NewLabel("Priority:"), widget.NewLabel(strconv.Itoa(objective.Priority)),
		widget.NewLabel("Goal ID:"), widget.NewLabel(objective.GoalID),
		widget.NewLabel("Method ID:"), widget.NewLabel(objective.MethodID),
		widget.NewLabel("Created:"), widget.NewLabel(objective.CreatedAt.Format("2006-01-02 15:04:05")),
	)

	// Description
	descEntry := widget.NewMultiLineEntry()
	descEntry.SetText(objective.Description)
	descEntry.Disable()

	// Context
	contextText := "None"
	if len(objective.Context) > 0 {
		var contextParts []string
		for k, v := range objective.Context {
			contextParts = append(contextParts, fmt.Sprintf("%s: %v", k, v))
		}
		contextText = strings.Join(contextParts, "\n")
	}
	contextEntry := widget.NewMultiLineEntry()
	contextEntry.SetText(contextText)
	contextEntry.Disable()

	// Results (if completed)
	var resultContainer *fyne.Container
	if objective.Result != nil {
		result := objective.Result
		resultGrid := container.NewGridWithColumns(2,
			widget.NewLabel("Success:"), widget.NewLabel(strconv.FormatBool(result.Success)),
			widget.NewLabel("Tokens Used:"), widget.NewLabel(strconv.Itoa(result.TokensUsed)),
			widget.NewLabel("Execution Time:"), widget.NewLabel(result.ExecutionTime.String()),
			widget.NewLabel("Completed:"), widget.NewLabel(result.CompletedAt.Format("2006-01-02 15:04:05")),
		)

		resultMessage := widget.NewMultiLineEntry()
		resultMessage.SetText(result.Message)
		resultMessage.Disable()

		resultContainer = container.NewVBox(
			widget.NewCard("Result", "", resultGrid),
			widget.NewCard("Message", "", resultMessage),
		)
	} else {
		resultContainer = container.NewVBox(
			widget.NewLabel("No results yet"),
		)
	}

	// Action buttons
	var actionButtons *fyne.Container
	switch objective.Status {
	case core.ObjectiveStatusPending:
		startButton := widget.NewButtonWithIcon("Start", theme.MediaPlayIcon(), func() {
			ov.startObjective(objective)
		})
		actionButtons = container.NewHBox(startButton)
	case core.ObjectiveStatusInProgress:
		pauseButton := widget.NewButtonWithIcon("Pause", theme.MediaPauseIcon(), func() {
			ov.pauseObjective(objective)
		})
		actionButtons = container.NewHBox(pauseButton)
	case core.ObjectiveStatusPaused:
		resumeButton := widget.NewButtonWithIcon("Resume", theme.MediaPlayIcon(), func() {
			ov.resumeObjective(objective)
		})
		actionButtons = container.NewHBox(resumeButton)
	default:
		editButton := widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() {
			ov.showEditObjectiveDialog(objective)
		})
		actionButtons = container.NewHBox(editButton)
	}

	// Create tabs for organized view
	tabs := container.NewAppTabs(
		container.NewTabItem("Info", infoGrid),
		container.NewTabItem("Description", descEntry),
		container.NewTabItem("Context", contextEntry),
		container.NewTabItem("Results", resultContainer),
	)

	return container.NewBorder(
		nil,           // top
		actionButtons, // bottom
		nil,           // left
		nil,           // right
		tabs,          // center
	)
}

// showNewObjectiveDialog displays the dialog for creating a new objective.
func (ov *ObjectivesView) showNewObjectiveDialog() {
	dialog := NewObjectiveDialog(ov.app, ov.parent, nil) // nil for new objective
	dialog.OnObjectiveSaved = func(objective *core.Objective) {
		ov.loadObjectives() // Refresh the list
	}
	dialog.Show()
}

// showEditObjectiveDialog displays the dialog for editing an existing objective.
func (ov *ObjectivesView) showEditObjectiveDialog(objective *core.Objective) {
	dialog := NewObjectiveDialog(ov.app, ov.parent, objective)
	dialog.OnObjectiveSaved = func(updatedObjective *core.Objective) {
		ov.loadObjectives() // Refresh the list
	}
	dialog.Show()
}

// startObjective starts a pending objective.
func (ov *ObjectivesView) startObjective(objective *core.Objective) {
	ctx := ov.app.GetContext()
	manager := ov.app.GetObjectiveManager()

	_, err := manager.StartObjective(ctx, objective.ID)
	if err != nil {
		dialog.ShowError(err, ov.parent)
		return
	}

	ov.loadObjectives()
}

// pauseObjective pauses an in-progress objective.
func (ov *ObjectivesView) pauseObjective(objective *core.Objective) {
	ctx := ov.app.GetContext()
	manager := ov.app.GetObjectiveManager()

	_, err := manager.PauseObjective(ctx, objective.ID)
	if err != nil {
		dialog.ShowError(err, ov.parent)
		return
	}

	ov.loadObjectives()
}

// resumeObjective resumes a paused objective.
func (ov *ObjectivesView) resumeObjective(objective *core.Objective) {
	ctx := ov.app.GetContext()
	manager := ov.app.GetObjectiveManager()

	_, err := manager.ResumeObjective(ctx, objective.ID)
	if err != nil {
		dialog.ShowError(err, ov.parent)
		return
	}

	ov.loadObjectives()
}

// startAutoRefresh begins the automatic refresh timer for real-time updates.
func (ov *ObjectivesView) startAutoRefresh() {
	go func() {
		ticker := time.NewTicker(5 * time.Second) // Refresh every 5 seconds
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Only refresh if there are active objectives
				hasActive := false
				for _, obj := range ov.objectives {
					if obj.Status == core.ObjectiveStatusInProgress {
						hasActive = true
						break
					}
				}
				if hasActive {
					ov.refreshChan <- true
				}
			case <-ov.refreshChan:
				ov.loadObjectives()
			case <-ov.stopRefresh:
				return
			}
		}
	}()
}

// Stop gracefully stops the auto-refresh timer.
func (ov *ObjectivesView) Stop() {
	select {
	case ov.stopRefresh <- true:
	default:
	}
}