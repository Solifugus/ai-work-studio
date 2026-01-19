package ui

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/Solifugus/ai-work-studio/pkg/core"
)

// ObjectiveDialogMode represents the mode of the objective dialog.
type ObjectiveDialogMode int

const (
	// ObjectiveDialogModeCreate indicates dialog is for creating a new objective
	ObjectiveDialogModeCreate ObjectiveDialogMode = iota

	// ObjectiveDialogModeEdit indicates dialog is for editing an existing objective
	ObjectiveDialogModeEdit
)

// ObjectiveDialog represents a modal dialog for creating or editing objectives.
type ObjectiveDialog struct {
	app       *App
	parent    fyne.Window
	mode      ObjectiveDialogMode
	objective *core.Objective // nil for create mode, existing objective for edit mode

	// Dialog
	dialog *dialog.ConfirmDialog

	// Form fields
	titleEntry       *widget.Entry
	descriptionEntry *widget.Entry
	goalSelect       *widget.Select
	methodSelect     *widget.Select
	prioritySlider   *widget.Slider
	statusSelect     *widget.Select
	contextEntry     *widget.Entry

	// Labels
	priorityLabel *widget.Label
	errorLabel    *widget.Label

	// Data
	availableGoals   []*core.Goal
	availableMethods []*core.Method
	goalIDMap        map[string]*core.Goal   // Maps display text to goal
	methodIDMap      map[string]*core.Method // Maps display text to method

	// Callback
	OnObjectiveSaved func(objective *core.Objective)
}

// NewObjectiveDialog creates a new objective dialog.
// Pass nil for objective to create a new objective, or an existing objective to edit.
func NewObjectiveDialog(app *App, parent fyne.Window, objective *core.Objective) *ObjectiveDialog {
	od := &ObjectiveDialog{
		app:         app,
		parent:      parent,
		objective:   objective,
		goalIDMap:   make(map[string]*core.Goal),
		methodIDMap: make(map[string]*core.Method),
	}

	if objective == nil {
		od.mode = ObjectiveDialogModeCreate
	} else {
		od.mode = ObjectiveDialogModeEdit
	}

	od.loadAvailableData()
	od.buildDialog()

	return od
}

// Show displays the objective dialog.
func (od *ObjectiveDialog) Show() {
	od.dialog.Show()
}

// buildDialog constructs the objective creation/edit dialog interface.
func (od *ObjectiveDialog) buildDialog() {
	od.buildFormFields()

	form := od.createForm()
	content := container.NewVBox(
		form,
		od.errorLabel,
	)

	// Dialog title and buttons
	var title string
	var confirmText string
	if od.mode == ObjectiveDialogModeCreate {
		title = "Create New Objective"
		confirmText = "Create"
	} else {
		title = "Edit Objective"
		confirmText = "Save"
	}

	od.dialog = dialog.NewCustomConfirm(
		title,
		confirmText,
		"Cancel",
		content,
		func(confirmed bool) {
			if confirmed {
				od.handleSubmit()
			} else {
				od.handleCancel()
			}
		},
		od.parent,
	)

	od.dialog.Resize(fyne.NewSize(550, 700))

	// If editing, populate fields
	if od.mode == ObjectiveDialogModeEdit {
		od.populateFields()
	}
}

// buildFormFields creates all form input fields.
func (od *ObjectiveDialog) buildFormFields() {
	// Title entry
	od.titleEntry = widget.NewEntry()
	od.titleEntry.SetPlaceHolder("Enter objective title...")
	od.titleEntry.Validator = func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("title is required")
		}
		if len(s) > 100 {
			return fmt.Errorf("title must be 100 characters or less")
		}
		return nil
	}

	// Description entry
	od.descriptionEntry = widget.NewMultiLineEntry()
	od.descriptionEntry.SetPlaceHolder("Describe what this objective aims to achieve...")
	od.descriptionEntry.Wrapping = fyne.TextWrapWord
	od.descriptionEntry.Validator = func(s string) error {
		if len(s) > 1000 {
			return fmt.Errorf("description must be 1000 characters or less")
		}
		return nil
	}

	// Goal selection (required)
	goalOptions := make([]string, 0, len(od.availableGoals))
	for _, goal := range od.availableGoals {
		displayText := fmt.Sprintf("[P%d] %s", goal.Priority, goal.Title)
		goalOptions = append(goalOptions, displayText)
		od.goalIDMap[displayText] = goal
	}
	od.goalSelect = widget.NewSelect(goalOptions, nil)
	od.goalSelect.PlaceHolder = "Select a goal..."

	// Method selection (optional)
	methodOptions := make([]string, 0, len(od.availableMethods)+1)
	methodOptions = append(methodOptions, "None") // Add "None" option
	for _, method := range od.availableMethods {
		displayText := fmt.Sprintf("[%s] %s", string(method.Domain), method.Name)
		methodOptions = append(methodOptions, displayText)
		od.methodIDMap[displayText] = method
	}
	od.methodSelect = widget.NewSelect(methodOptions, nil)
	od.methodSelect.PlaceHolder = "Select a method (optional)..."
	od.methodSelect.SetSelected("None") // Default to None

	// Priority slider
	od.prioritySlider = widget.NewSlider(1, 10)
	od.prioritySlider.SetValue(5) // Default to medium priority
	od.prioritySlider.Step = 1
	od.priorityLabel = widget.NewLabel("Priority: 5")
	od.prioritySlider.OnChanged = func(value float64) {
		od.priorityLabel.SetText(fmt.Sprintf("Priority: %d", int(value)))
	}

	// Status selection
	statusOptions := []string{
		string(core.ObjectiveStatusPending),
		string(core.ObjectiveStatusInProgress),
		string(core.ObjectiveStatusCompleted),
		string(core.ObjectiveStatusFailed),
		string(core.ObjectiveStatusPaused),
	}
	od.statusSelect = widget.NewSelect(statusOptions, nil)
	od.statusSelect.SetSelected(string(core.ObjectiveStatusPending)) // Default to pending

	// Context entry
	od.contextEntry = widget.NewMultiLineEntry()
	od.contextEntry.SetPlaceHolder("Enter context data as JSON (optional)...")
	od.contextEntry.Wrapping = fyne.TextWrapWord
	od.contextEntry.Validator = func(s string) error {
		if strings.TrimSpace(s) == "" {
			return nil // Empty is OK
		}
		var dummy map[string]interface{}
		if err := json.Unmarshal([]byte(s), &dummy); err != nil {
			return fmt.Errorf("context must be valid JSON")
		}
		return nil
	}

	// Error label
	od.errorLabel = widget.NewLabel("")
	od.errorLabel.Hide()
}

// createForm creates the form layout with all fields.
func (od *ObjectiveDialog) createForm() *fyne.Container {
	// Create form items with labels
	form := container.NewVBox(
		// Title
		widget.NewCard("Title", "", container.NewVBox(
			od.titleEntry,
			widget.NewLabel("Required. Brief, descriptive name for the objective."),
		)),

		// Goal selection
		widget.NewCard("Goal", "", container.NewVBox(
			od.goalSelect,
			widget.NewLabel("Required. Which goal does this objective serve?"),
		)),

		// Method selection
		widget.NewCard("Method", "", container.NewVBox(
			od.methodSelect,
			widget.NewLabel("Optional. Proven approach to use for this objective."),
		)),

		// Priority
		widget.NewCard("Priority", "", container.NewVBox(
			container.NewBorder(nil, nil, od.priorityLabel, nil, od.prioritySlider),
			widget.NewLabel("1=Low, 5=Medium, 10=High. Inherited from goal by default."),
		)),

		// Status
		widget.NewCard("Status", "", container.NewVBox(
			od.statusSelect,
			widget.NewLabel("Current state of the objective."),
		)),

		// Description
		widget.NewCard("Description", "", container.NewVBox(
			container.NewScroll(od.descriptionEntry),
			widget.NewLabel("Detailed explanation of what this objective accomplishes."),
		)),

		// Context
		widget.NewCard("Context", "", container.NewVBox(
			container.NewScroll(od.contextEntry),
			widget.NewLabel("Optional JSON data providing context for execution."),
		)),
	)

	return form
}

// loadAvailableData loads goals and methods from storage for selection.
func (od *ObjectiveDialog) loadAvailableData() {
	ctx := od.app.GetContext()

	// Load goals
	goalManager := od.app.GetGoalManager()
	goals, err := goalManager.ListGoals(ctx, core.GoalFilter{})
	if err != nil {
		log.Printf("Error loading goals for objective dialog: %v", err)
		od.availableGoals = []*core.Goal{}
	} else {
		// Filter out archived/completed goals for new objectives
		if od.mode == ObjectiveDialogModeCreate {
			activeGoals := make([]*core.Goal, 0)
			for _, goal := range goals {
				if goal.Status == core.GoalStatusActive {
					activeGoals = append(activeGoals, goal)
				}
			}
			od.availableGoals = activeGoals
		} else {
			od.availableGoals = goals
		}
	}

	// Load methods
	methodManager := od.app.GetMethodManager()
	methods, err := methodManager.ListMethods(ctx, core.MethodFilter{})
	if err != nil {
		log.Printf("Error loading methods for objective dialog: %v", err)
		od.availableMethods = []*core.Method{}
	} else {
		// Filter to only active methods
		activeMethods := make([]*core.Method, 0)
		for _, method := range methods {
			if method.Status == core.MethodStatusActive {
				activeMethods = append(activeMethods, method)
			}
		}
		od.availableMethods = activeMethods
	}
}

// populateFields fills the form fields when editing an existing objective.
func (od *ObjectiveDialog) populateFields() {
	if od.objective == nil {
		return
	}

	obj := od.objective

	// Set title
	od.titleEntry.SetText(obj.Title)

	// Set description
	od.descriptionEntry.SetText(obj.Description)

	// Set goal selection
	for displayText, goal := range od.goalIDMap {
		if goal.ID == obj.GoalID {
			od.goalSelect.SetSelected(displayText)
			break
		}
	}

	// Set method selection
	if obj.MethodID != "" {
		for displayText, method := range od.methodIDMap {
			if method.ID == obj.MethodID {
				od.methodSelect.SetSelected(displayText)
				break
			}
		}
	} else {
		od.methodSelect.SetSelected("None")
	}

	// Set priority
	od.prioritySlider.SetValue(float64(obj.Priority))
	od.priorityLabel.SetText(fmt.Sprintf("Priority: %d", obj.Priority))

	// Set status
	od.statusSelect.SetSelected(string(obj.Status))

	// Set context
	if len(obj.Context) > 0 {
		contextJSON, err := json.MarshalIndent(obj.Context, "", "  ")
		if err == nil {
			od.contextEntry.SetText(string(contextJSON))
		}
	}
}

// handleSubmit processes the form submission.
func (od *ObjectiveDialog) handleSubmit() {
	// Clear previous errors
	od.errorLabel.Hide()

	// Validate all fields
	if err := od.validateForm(); err != nil {
		od.showError(err.Error())
		return
	}

	// Create or update objective
	if od.mode == ObjectiveDialogModeCreate {
		od.createObjective()
	} else {
		od.updateObjective()
	}
}

// validateForm validates all form fields.
func (od *ObjectiveDialog) validateForm() error {
	// Validate title
	if err := od.titleEntry.Validator(od.titleEntry.Text); err != nil {
		return fmt.Errorf("title: %v", err)
	}

	// Validate description
	if err := od.descriptionEntry.Validator(od.descriptionEntry.Text); err != nil {
		return fmt.Errorf("description: %v", err)
	}

	// Validate goal selection
	if od.goalSelect.Selected == "" {
		return fmt.Errorf("goal selection is required")
	}

	// Validate context JSON
	if err := od.contextEntry.Validator(od.contextEntry.Text); err != nil {
		return fmt.Errorf("context: %v", err)
	}

	return nil
}

// createObjective creates a new objective.
func (od *ObjectiveDialog) createObjective() {
	ctx := od.app.GetContext()
	manager := od.app.GetObjectiveManager()

	// Get selected goal
	selectedGoal := od.goalIDMap[od.goalSelect.Selected]
	if selectedGoal == nil {
		od.showError("Invalid goal selection")
		return
	}

	// Get selected method (optional)
	var methodID string
	if od.methodSelect.Selected != "None" && od.methodSelect.Selected != "" {
		selectedMethod := od.methodIDMap[od.methodSelect.Selected]
		if selectedMethod != nil {
			methodID = selectedMethod.ID
		}
	}

	// Parse context
	context := make(map[string]interface{})
	if strings.TrimSpace(od.contextEntry.Text) != "" {
		if err := json.Unmarshal([]byte(od.contextEntry.Text), &context); err != nil {
			od.showError(fmt.Sprintf("Invalid context JSON: %v", err))
			return
		}
	}

	// Create objective
	objective, err := manager.CreateObjective(ctx, selectedGoal.ID, methodID,
		od.titleEntry.Text, od.descriptionEntry.Text, context, int(od.prioritySlider.Value))

	if err != nil {
		od.showError(fmt.Sprintf("Failed to create objective: %v", err))
		return
	}

	// Set status if different from default
	selectedStatus := core.ObjectiveStatus(od.statusSelect.Selected)
	if selectedStatus != core.ObjectiveStatusPending {
		switch selectedStatus {
		case core.ObjectiveStatusInProgress:
			_, err = manager.StartObjective(ctx, objective.ID)
		case core.ObjectiveStatusPaused:
			_, err = manager.StartObjective(ctx, objective.ID)
			if err == nil {
				_, err = manager.PauseObjective(ctx, objective.ID)
			}
		case core.ObjectiveStatusCompleted:
			_, err = manager.StartObjective(ctx, objective.ID)
			if err == nil {
				_, err = manager.CompleteObjective(ctx, objective.ID, core.ObjectiveResult{
					Success:     true,
					Message:     "Manually marked as completed",
					CompletedAt: time.Now(),
				})
			}
		case core.ObjectiveStatusFailed:
			_, err = manager.StartObjective(ctx, objective.ID)
			if err == nil {
				_, err = manager.FailObjective(ctx, objective.ID, "Manually marked as failed", 0)
			}
		}

		if err != nil {
			log.Printf("Warning: Failed to set objective status: %v", err)
			// Don't fail the creation, just log the warning
		}
	}

	od.dialog.Hide()

	// Callback
	if od.OnObjectiveSaved != nil {
		od.OnObjectiveSaved(objective)
	}
}

// updateObjective updates an existing objective.
func (od *ObjectiveDialog) updateObjective() {
	ctx := od.app.GetContext()
	manager := od.app.GetObjectiveManager()

	// Get selected goal
	selectedGoal := od.goalIDMap[od.goalSelect.Selected]
	if selectedGoal == nil {
		od.showError("Invalid goal selection")
		return
	}

	// Get selected method (optional)
	var methodID string
	if od.methodSelect.Selected != "None" && od.methodSelect.Selected != "" {
		selectedMethod := od.methodIDMap[od.methodSelect.Selected]
		if selectedMethod != nil {
			methodID = selectedMethod.ID
		}
	}

	// Parse context
	context := make(map[string]interface{})
	if strings.TrimSpace(od.contextEntry.Text) != "" {
		if err := json.Unmarshal([]byte(od.contextEntry.Text), &context); err != nil {
			od.showError(fmt.Sprintf("Invalid context JSON: %v", err))
			return
		}
	}

	// Update objective
	updates := &core.ObjectiveUpdates{
		GoalID:      &selectedGoal.ID,
		MethodID:    &methodID,
		Title:       &od.titleEntry.Text,
		Description: &od.descriptionEntry.Text,
		Priority:    func() *int { p := int(od.prioritySlider.Value); return &p }(),
		Context:     context,
	}

	objective, err := manager.UpdateObjective(ctx, od.objective.ID, *updates)
	if err != nil {
		od.showError(fmt.Sprintf("Failed to update objective: %v", err))
		return
	}

	od.dialog.Hide()

	// Callback
	if od.OnObjectiveSaved != nil {
		od.OnObjectiveSaved(objective)
	}
}

// handleCancel handles dialog cancellation.
func (od *ObjectiveDialog) handleCancel() {
	od.dialog.Hide()
}

// showError displays an error message in the dialog.
func (od *ObjectiveDialog) showError(message string) {
	od.errorLabel.SetText(fmt.Sprintf("Error: %s", message))
	od.errorLabel.Show()
}