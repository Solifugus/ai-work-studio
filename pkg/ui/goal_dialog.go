package ui

import (
	"fmt"
	"log"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/yourusername/ai-work-studio/pkg/core"
)

// GoalDialogMode represents the mode of the goal dialog.
type GoalDialogMode int

const (
	// GoalDialogModeCreate indicates dialog is for creating a new goal
	GoalDialogModeCreate GoalDialogMode = iota

	// GoalDialogModeEdit indicates dialog is for editing an existing goal
	GoalDialogModeEdit
)

// GoalDialog represents a modal dialog for creating or editing goals.
type GoalDialog struct {
	app    *App
	parent fyne.Window
	mode   GoalDialogMode
	goal   *core.Goal // nil for create mode, existing goal for edit mode

	// Dialog
	dialog *dialog.ConfirmDialog

	// Form fields
	titleEntry       *widget.Entry
	descriptionEntry *widget.Entry
	prioritySlider   *widget.Slider
	statusSelect     *widget.Select
	parentSelect     *widget.Select
	contextEntry     *widget.Entry

	// Labels
	priorityLabel *widget.Label
	errorLabel    *widget.Label

	// Data
	availableParents []*core.Goal

	// Callbacks
	OnGoalSaved func(goal *core.Goal)
	OnCancelled func()
}

// NewGoalDialog creates a new goal dialog.
func NewGoalDialog(app *App, parent fyne.Window, mode GoalDialogMode, goal *core.Goal) *GoalDialog {
	gd := &GoalDialog{
		app:    app,
		parent: parent,
		mode:   mode,
		goal:   goal,
	}

	gd.buildDialog()
	gd.loadAvailableParents()
	gd.populateFields()

	return gd
}

// buildDialog constructs the dialog and form components.
func (gd *GoalDialog) buildDialog() {
	// Create form fields
	gd.buildFormFields()

	// Create form content
	content := gd.buildFormContent()

	// Create dialog title
	var title string
	if gd.mode == GoalDialogModeCreate {
		title = "Create New Goal"
	} else {
		title = "Edit Goal"
	}

	// Create buttons
	var submitText string
	if gd.mode == GoalDialogModeCreate {
		submitText = "Create"
	} else {
		submitText = "Update"
	}

	// Create the dialog
	gd.dialog = dialog.NewCustomConfirm(
		title,
		submitText,
		"Cancel",
		content,
		func(confirmed bool) {
			if confirmed {
				gd.handleSubmit()
			} else {
				gd.handleCancel()
			}
		},
		gd.parent,
	)

	// Set dialog size
	gd.dialog.Resize(fyne.NewSize(500, 600))
}

// buildFormFields creates all the form input fields.
func (gd *GoalDialog) buildFormFields() {
	// Title entry
	gd.titleEntry = widget.NewEntry()
	gd.titleEntry.SetPlaceHolder("Enter goal title...")
	gd.titleEntry.Validator = func(text string) error {
		if strings.TrimSpace(text) == "" {
			return fmt.Errorf("title is required")
		}
		if len(text) > 100 {
			return fmt.Errorf("title must be 100 characters or less")
		}
		return nil
	}

	// Description entry
	gd.descriptionEntry = widget.NewMultiLineEntry()
	gd.descriptionEntry.SetPlaceHolder("Enter goal description...")
	gd.descriptionEntry.Wrapping = fyne.TextWrapWord

	// Priority slider (1-10, where 1 is highest priority)
	gd.prioritySlider = widget.NewSlider(1, 10)
	gd.prioritySlider.Step = 1
	gd.prioritySlider.Value = 5 // Default to medium priority
	gd.prioritySlider.OnChanged = func(value float64) {
		gd.priorityLabel.SetText(fmt.Sprintf("Priority: %d", int(value)))
	}

	// Priority label
	gd.priorityLabel = widget.NewLabel("Priority: 5")

	// Status select
	statusOptions := []string{"Active", "Paused", "Completed", "Archived"}
	gd.statusSelect = widget.NewSelect(statusOptions, nil)
	gd.statusSelect.SetSelected("Active") // Default to active

	// Parent goal select
	gd.parentSelect = widget.NewSelect([]string{"None"}, nil)
	gd.parentSelect.SetSelected("None")

	// Context entry for additional metadata
	gd.contextEntry = widget.NewMultiLineEntry()
	gd.contextEntry.SetPlaceHolder("Additional context (optional, JSON format)")

	// Error label
	gd.errorLabel = widget.NewLabel("")
	gd.errorLabel.Importance = widget.DangerImportance
	gd.errorLabel.Hide()
}

// buildFormContent arranges the form fields in a layout.
func (gd *GoalDialog) buildFormContent() *fyne.Container {
	// Title section
	titleSection := container.NewVBox(
		widget.NewLabel("Title *"),
		gd.titleEntry,
	)

	// Description section
	descSection := container.NewVBox(
		widget.NewLabel("Description"),
		container.NewWithoutLayout(gd.descriptionEntry),
	)
	// Set fixed height for description
	gd.descriptionEntry.Resize(fyne.NewSize(460, 80))

	// Priority section
	prioritySection := container.NewVBox(
		gd.priorityLabel,
		gd.prioritySlider,
		widget.NewLabel("1 = Highest Priority, 10 = Lowest Priority"),
	)

	// Status section
	statusSection := container.NewVBox(
		widget.NewLabel("Status"),
		gd.statusSelect,
	)

	// Parent goal section
	parentSection := container.NewVBox(
		widget.NewLabel("Parent Goal"),
		gd.parentSelect,
		widget.NewLabel("Select a parent goal to create a hierarchy"),
	)

	// Context section
	contextSection := container.NewVBox(
		widget.NewLabel("Additional Context (Optional)"),
		container.NewWithoutLayout(gd.contextEntry),
		widget.NewLabel("Enter additional metadata as JSON, e.g., {\"tags\": [\"work\", \"urgent\"]}"),
	)
	// Set fixed height for context
	gd.contextEntry.Resize(fyne.NewSize(460, 60))

	// Error section
	errorSection := container.NewVBox(gd.errorLabel)

	// Combine all sections
	return container.NewVBox(
		titleSection,
		widget.NewSeparator(),
		descSection,
		widget.NewSeparator(),
		prioritySection,
		widget.NewSeparator(),
		statusSection,
		widget.NewSeparator(),
		parentSection,
		widget.NewSeparator(),
		contextSection,
		widget.NewSeparator(),
		errorSection,
	)
}

// loadAvailableParents loads the list of goals that can be selected as parent goals.
func (gd *GoalDialog) loadAvailableParents() {
	ctx := gd.app.GetContext()

	// Get all goals except the current one (for edit mode)
	goals, err := gd.app.GetGoalManager().ListGoals(ctx, core.GoalFilter{})
	if err != nil {
		log.Printf("Failed to load available parent goals: %v", err)
		return
	}

	// Filter out the current goal (if editing) and completed/archived goals
	var availableParents []*core.Goal
	for _, goal := range goals {
		// Skip the current goal (prevent self-reference)
		if gd.goal != nil && goal.ID == gd.goal.ID {
			continue
		}

		// Skip completed/archived goals as they shouldn't be parent goals
		if goal.Status == core.GoalStatusCompleted || goal.Status == core.GoalStatusArchived {
			continue
		}

		availableParents = append(availableParents, goal)
	}

	gd.availableParents = availableParents

	// Update parent select options
	parentOptions := []string{"None"}
	for _, parent := range availableParents {
		parentOptions = append(parentOptions, fmt.Sprintf("%s (P%d)", parent.Title, parent.Priority))
	}

	gd.parentSelect.Options = parentOptions
	gd.parentSelect.SetSelected("None")
	gd.parentSelect.Refresh()
}

// populateFields fills the form fields based on the current mode and goal data.
func (gd *GoalDialog) populateFields() {
	if gd.mode == GoalDialogModeEdit && gd.goal != nil {
		// Populate fields with existing goal data
		gd.titleEntry.SetText(gd.goal.Title)
		gd.descriptionEntry.SetText(gd.goal.Description)
		gd.prioritySlider.SetValue(float64(gd.goal.Priority))

		// Set status
		switch gd.goal.Status {
		case core.GoalStatusActive:
			gd.statusSelect.SetSelected("Active")
		case core.GoalStatusPaused:
			gd.statusSelect.SetSelected("Paused")
		case core.GoalStatusCompleted:
			gd.statusSelect.SetSelected("Completed")
		case core.GoalStatusArchived:
			gd.statusSelect.SetSelected("Archived")
		}

		// Set parent goal
		gd.setCurrentParentGoal()

		// Set context if available
		if gd.goal.UserContext != nil && len(gd.goal.UserContext) > 0 {
			contextText := gd.formatUserContext(gd.goal.UserContext)
			gd.contextEntry.SetText(contextText)
		}
	}
}

// setCurrentParentGoal sets the parent goal select to the current parent (if any).
func (gd *GoalDialog) setCurrentParentGoal() {
	if gd.goal == nil {
		return
	}

	ctx := gd.app.GetContext()
	parentGoals, err := gd.app.GetGoalManager().GetParentGoals(ctx, gd.goal.ID)
	if err != nil {
		log.Printf("Failed to get parent goals: %v", err)
		return
	}

	if len(parentGoals) > 0 {
		// Find the parent in our available parents list
		parentGoal := parentGoals[0] // Take the first parent
		for i, available := range gd.availableParents {
			if available.ID == parentGoal.ID {
				// Select this parent (add 1 to skip "None" option)
				gd.parentSelect.SetSelectedIndex(i + 1)
				break
			}
		}
	}
}

// formatUserContext converts the user context map to a readable string.
func (gd *GoalDialog) formatUserContext(context map[string]interface{}) string {
	if len(context) == 0 {
		return ""
	}

	var parts []string
	for key, value := range context {
		parts = append(parts, fmt.Sprintf("\"%s\": \"%v\"", key, value))
	}

	return fmt.Sprintf("{%s}", strings.Join(parts, ", "))
}

// parseUserContext parses the user context string back to a map.
func (gd *GoalDialog) parseUserContext(text string) map[string]interface{} {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	// Simple JSON-like parsing - just handle basic key-value pairs
	// For a production system, you'd want proper JSON parsing
	context := make(map[string]interface{})

	// Remove outer braces
	text = strings.Trim(text, "{}")

	// Split by commas (simple approach)
	pairs := strings.Split(text, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) == 2 {
			key := strings.Trim(strings.TrimSpace(parts[0]), "\"")
			value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
			context[key] = value
		}
	}

	return context
}

// Show displays the dialog.
func (gd *GoalDialog) Show() {
	gd.dialog.Show()
}

// Hide hides the dialog.
func (gd *GoalDialog) Hide() {
	gd.dialog.Hide()
}

// handleSubmit processes the form submission.
func (gd *GoalDialog) handleSubmit() {
	// Validate form
	if err := gd.validateForm(); err != nil {
		gd.showError(err.Error())
		return
	}

	// Clear any previous errors
	gd.hideError()

	// Prepare goal data
	title := strings.TrimSpace(gd.titleEntry.Text)
	description := strings.TrimSpace(gd.descriptionEntry.Text)
	priority := int(gd.prioritySlider.Value)
	status := gd.getSelectedStatus()
	userContext := gd.parseUserContext(gd.contextEntry.Text)

	ctx := gd.app.GetContext()

	if gd.mode == GoalDialogModeCreate {
		// Create new goal
		goal, err := gd.app.GetGoalManager().CreateGoal(ctx, title, description, priority, userContext)
		if err != nil {
			gd.showError(fmt.Sprintf("Failed to create goal: %v", err))
			return
		}

		// Set parent relationship if selected
		if err := gd.setParentRelationship(goal.ID); err != nil {
			log.Printf("Warning: Failed to set parent relationship: %v", err)
			// Continue anyway - the goal was created successfully
		}

		// Set status if not active
		if status != core.GoalStatusActive {
			updates := core.GoalUpdates{Status: &status}
			goal, err = gd.app.GetGoalManager().UpdateGoal(ctx, goal.ID, updates)
			if err != nil {
				log.Printf("Warning: Failed to set initial status: %v", err)
				// Continue anyway
			}
		}

		// Call success callback
		if gd.OnGoalSaved != nil {
			gd.OnGoalSaved(goal)
		}

	} else {
		// Update existing goal
		updates := core.GoalUpdates{
			Title:       &title,
			Description: &description,
			Priority:    &priority,
			Status:      &status,
			UserContext: userContext,
		}

		goal, err := gd.app.GetGoalManager().UpdateGoal(ctx, gd.goal.ID, updates)
		if err != nil {
			gd.showError(fmt.Sprintf("Failed to update goal: %v", err))
			return
		}

		// Update parent relationship if changed
		if err := gd.updateParentRelationship(goal.ID); err != nil {
			log.Printf("Warning: Failed to update parent relationship: %v", err)
			// Continue anyway
		}

		// Call success callback
		if gd.OnGoalSaved != nil {
			gd.OnGoalSaved(goal)
		}
	}

	// Close dialog
	gd.dialog.Hide()
}

// handleCancel processes the form cancellation.
func (gd *GoalDialog) handleCancel() {
	if gd.OnCancelled != nil {
		gd.OnCancelled()
	}
}

// validateForm validates all form fields.
func (gd *GoalDialog) validateForm() error {
	// Validate title
	if err := gd.titleEntry.Validate(); err != nil {
		return fmt.Errorf("title: %v", err)
	}

	// Validate context if provided
	if contextText := strings.TrimSpace(gd.contextEntry.Text); contextText != "" {
		// Simple validation - just check that it's balanced braces
		if strings.Count(contextText, "{") != strings.Count(contextText, "}") {
			return fmt.Errorf("context: invalid JSON format (unbalanced braces)")
		}
	}

	return nil
}

// getSelectedStatus returns the selected goal status.
func (gd *GoalDialog) getSelectedStatus() core.GoalStatus {
	switch gd.statusSelect.Selected {
	case "Active":
		return core.GoalStatusActive
	case "Paused":
		return core.GoalStatusPaused
	case "Completed":
		return core.GoalStatusCompleted
	case "Archived":
		return core.GoalStatusArchived
	default:
		return core.GoalStatusActive
	}
}

// setParentRelationship sets the parent relationship for a new goal.
func (gd *GoalDialog) setParentRelationship(goalID string) error {
	if gd.parentSelect.Selected == "None" || gd.parentSelect.SelectedIndex() <= 0 {
		return nil // No parent selected
	}

	// Get the selected parent
	parentIndex := gd.parentSelect.SelectedIndex() - 1 // Subtract 1 for "None" option
	if parentIndex < 0 || parentIndex >= len(gd.availableParents) {
		return fmt.Errorf("invalid parent selection")
	}

	parentGoal := gd.availableParents[parentIndex]
	ctx := gd.app.GetContext()

	return gd.app.GetGoalManager().AddSubGoal(ctx, parentGoal.ID, goalID)
}

// updateParentRelationship updates the parent relationship for an existing goal.
func (gd *GoalDialog) updateParentRelationship(goalID string) error {
	// For now, we'll skip this complex operation since the storage layer
	// doesn't support relationship deletion/modification easily.
	// In a future version, this could be implemented by removing old relationships
	// and adding new ones.
	return nil
}

// showError displays an error message in the dialog.
func (gd *GoalDialog) showError(message string) {
	gd.errorLabel.SetText(message)
	gd.errorLabel.Show()
}

// hideError hides the error message.
func (gd *GoalDialog) hideError() {
	gd.errorLabel.Hide()
}