package ui

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/Solifugus/ai-work-studio/pkg/core"
)

// GoalsView represents the goals management interface with hierarchical display,
// CRUD operations, search/filter capabilities, and status indicators.
type GoalsView struct {
	app    *App
	parent fyne.Window

	// Main container
	container *fyne.Container

	// UI Components
	toolbar     *container.AppTabs
	searchEntry *widget.Entry
	filterSelect *widget.Select
	sortSelect   *widget.Select
	goalsTree   *widget.Tree
	statusLabel *widget.Label

	// Data
	goals     []*core.Goal
	goalNodes map[string]*GoalTreeNode // Maps goal IDs to tree nodes
	rootGoals []string                 // IDs of top-level goals (no parents)

	// State
	searchFilter   string
	statusFilter   core.GoalStatus
	sortMode      string
	selectedGoalID string
}

// GoalTreeNode represents a goal in the tree structure.
type GoalTreeNode struct {
	Goal     *core.Goal
	Children []string // Goal IDs of child goals
	Parent   string   // Goal ID of parent goal
}

// NewGoalsView creates a new goals management view.
func NewGoalsView(app *App, parent fyne.Window) *GoalsView {
	gv := &GoalsView{
		app:       app,
		parent:    parent,
		goalNodes: make(map[string]*GoalTreeNode),
		sortMode:  "priority", // Default sort by priority
	}

	gv.buildUI()
	gv.refreshData()

	return gv
}

// buildUI constructs the user interface components.
func (gv *GoalsView) buildUI() {
	// Create toolbar with search and filter controls
	gv.buildToolbar()

	// Create the goals tree
	gv.buildGoalsTree()

	// Create status bar
	gv.statusLabel = widget.NewLabel("Ready")
	gv.statusLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Arrange components in border layout
	content := container.NewVBox(gv.toolbar, gv.goalsTree)
	gv.container = container.NewBorder(
		nil,            // top
		gv.statusLabel, // bottom
		nil,            // left
		nil,            // right
		container.NewScroll(content), // center
	)
}

// buildToolbar creates the search and filter controls.
func (gv *GoalsView) buildToolbar() {
	// Search entry
	gv.searchEntry = widget.NewEntry()
	gv.searchEntry.SetPlaceHolder("Search goals...")
	gv.searchEntry.OnChanged = func(text string) {
		gv.searchFilter = text
		gv.applyFiltersAndSort()
	}

	// Status filter
	statusOptions := []string{"All", "Active", "Paused", "Completed", "Archived"}
	gv.filterSelect = widget.NewSelect(statusOptions, func(selected string) {
		switch selected {
		case "Active":
			gv.statusFilter = core.GoalStatusActive
		case "Paused":
			gv.statusFilter = core.GoalStatusPaused
		case "Completed":
			gv.statusFilter = core.GoalStatusCompleted
		case "Archived":
			gv.statusFilter = core.GoalStatusArchived
		default:
			gv.statusFilter = "" // "All" - no filter
		}
		gv.applyFiltersAndSort()
	})
	gv.filterSelect.SetSelected("All")

	// Sort options
	sortOptions := []string{"Priority", "Title", "Created", "Status"}
	gv.sortSelect = widget.NewSelect(sortOptions, func(selected string) {
		gv.sortMode = strings.ToLower(selected)
		gv.applyFiltersAndSort()
	})
	gv.sortSelect.SetSelected("Priority")

	// Action buttons
	newButton := widget.NewButtonWithIcon("New Goal", theme.ContentAddIcon(), func() {
		gv.showCreateGoalDialog()
	})

	editButton := widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() {
		if gv.selectedGoalID != "" {
			gv.showEditGoalDialog(gv.selectedGoalID)
		}
	})

	deleteButton := widget.NewButtonWithIcon("Archive", theme.ContentRemoveIcon(), func() {
		if gv.selectedGoalID != "" {
			gv.archiveGoal(gv.selectedGoalID)
		}
	})

	refreshButton := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		gv.refreshData()
	})

	// Arrange toolbar components
	searchContainer := container.NewBorder(nil, nil,
		widget.NewLabel("Search:"), nil, gv.searchEntry)
	filterContainer := container.NewBorder(nil, nil,
		widget.NewLabel("Status:"), nil, gv.filterSelect)
	sortContainer := container.NewBorder(nil, nil,
		widget.NewLabel("Sort:"), nil, gv.sortSelect)

	controlsRow := container.NewHBox(
		searchContainer,
		widget.NewSeparator(),
		filterContainer,
		widget.NewSeparator(),
		sortContainer,
	)

	buttonsRow := container.NewHBox(
		newButton,
		editButton,
		deleteButton,
		widget.NewSeparator(),
		refreshButton,
	)

	gv.toolbar = container.NewAppTabs(
		container.NewTabItem("Controls", controlsRow),
		container.NewTabItem("Actions", buttonsRow),
	)
}

// buildGoalsTree creates the hierarchical tree widget.
func (gv *GoalsView) buildGoalsTree() {
	gv.goalsTree = widget.NewTree(
		// Child UIDs function - returns child goal IDs for a given goal ID
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			if uid == "" {
				// Root level - return top-level goals
				return gv.rootGoals
			}

			// Return children for this goal
			if node, exists := gv.goalNodes[uid]; exists {
				return node.Children
			}
			return []string{}
		},

		// Is branch function - determines if a node can have children
		func(uid widget.TreeNodeID) bool {
			if uid == "" {
				return true // Root can have children
			}

			// Check if this goal has children
			if node, exists := gv.goalNodes[uid]; exists {
				return len(node.Children) > 0
			}
			return false
		},

		// Create node function - creates the display widget for each goal
		func(branch bool) fyne.CanvasObject {
			return gv.createGoalNodeWidget()
		},

		// Update node function - updates the display for a specific goal
		func(uid widget.TreeNodeID, branch bool, node fyne.CanvasObject) {
			gv.updateGoalNodeWidget(uid, node)
		},
	)

	// Handle selection changes
	gv.goalsTree.OnSelected = func(uid widget.TreeNodeID) {
		gv.selectedGoalID = uid
		gv.updateStatusBar()
	}

	// Handle double-click to edit
	// Note: Fyne doesn't have a built-in double-click handler for Tree
	// This could be implemented with custom gesture handling if needed
}

// createGoalNodeWidget creates the basic widget structure for displaying a goal in the tree.
func (gv *GoalsView) createGoalNodeWidget() fyne.CanvasObject {
	statusIcon := widget.NewIcon(theme.InfoIcon())
	priorityLabel := widget.NewLabel("P5")
	titleLabel := widget.NewLabel("Goal Title")

	priorityLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Set minimum size for consistent layout
	statusIcon.Resize(fyne.NewSize(16, 16))
	priorityLabel.Resize(fyne.NewSize(24, 20))

	return container.NewHBox(
		statusIcon,
		priorityLabel,
		titleLabel,
	)
}

// updateGoalNodeWidget updates the display for a specific goal node.
func (gv *GoalsView) updateGoalNodeWidget(uid widget.TreeNodeID, nodeWidget fyne.CanvasObject) {
	if uid == "" {
		return
	}

	goalNode, exists := gv.goalNodes[uid]
	if !exists {
		return
	}

	goal := goalNode.Goal
	containerObj := nodeWidget.(*fyne.Container)

	if len(containerObj.Objects) >= 3 {
		// Update status icon
		statusIcon := containerObj.Objects[0].(*widget.Icon)
		statusIcon.SetResource(gv.getStatusIcon(goal.Status))

		// Update priority label with color coding
		priorityLabel := containerObj.Objects[1].(*widget.Label)
		priorityLabel.SetText(fmt.Sprintf("P%d", goal.Priority))

		// Color code priority: 1-3=red(high), 4-6=yellow(medium), 7-10=green(low)
		if goal.Priority <= 3 {
			priorityLabel.Importance = widget.HighImportance
		} else if goal.Priority <= 6 {
			priorityLabel.Importance = widget.MediumImportance
		} else {
			priorityLabel.Importance = widget.LowImportance
		}

		// Update title
		titleLabel := containerObj.Objects[2].(*widget.Label)
		titleLabel.SetText(goal.Title)

		// Dim completed/archived goals
		if goal.Status == core.GoalStatusCompleted || goal.Status == core.GoalStatusArchived {
			titleLabel.TextStyle = fyne.TextStyle{Italic: true}
		} else {
			titleLabel.TextStyle = fyne.TextStyle{}
		}
	}
}

// getStatusIcon returns the appropriate icon for a goal status.
func (gv *GoalsView) getStatusIcon(status core.GoalStatus) fyne.Resource {
	switch status {
	case core.GoalStatusActive:
		return theme.MediaPlayIcon()
	case core.GoalStatusPaused:
		return theme.MediaPauseIcon()
	case core.GoalStatusCompleted:
		return theme.ConfirmIcon()
	case core.GoalStatusArchived:
		return theme.ContentClearIcon()
	default:
		return theme.InfoIcon()
	}
}

// refreshData loads goals from storage and rebuilds the tree structure.
func (gv *GoalsView) refreshData() {
	gv.updateStatusBar("Loading goals...")

	// Load all goals
	ctx := gv.app.GetContext()
	goals, err := gv.app.GetGoalManager().ListGoals(ctx, core.GoalFilter{})
	if err != nil {
		log.Printf("Failed to load goals: %v", err)
		gv.updateStatusBar("Error loading goals")
		return
	}

	gv.goals = goals
	gv.rebuildTreeStructure()
	gv.applyFiltersAndSort()

	count := len(gv.goals)
	gv.updateStatusBar(fmt.Sprintf("Loaded %d goal(s)", count))
}

// rebuildTreeStructure analyzes goal relationships and builds the tree structure.
func (gv *GoalsView) rebuildTreeStructure() {
	gv.goalNodes = make(map[string]*GoalTreeNode)
	gv.rootGoals = []string{}

	// First pass: create nodes for all goals
	for _, goal := range gv.goals {
		gv.goalNodes[goal.ID] = &GoalTreeNode{
			Goal:     goal,
			Children: []string{},
			Parent:   "", // Will be set in second pass
		}
	}

	// Second pass: build relationships by querying parent/child relationships
	ctx := gv.app.GetContext()
	for goalID := range gv.goalNodes {
		// Get parent goals (goals this goal serves)
		parentGoals, err := gv.app.GetGoalManager().GetParentGoals(ctx, goalID)
		if err != nil {
			log.Printf("Failed to get parent goals for %s: %v", goalID, err)
			continue
		}

		if len(parentGoals) > 0 {
			// This goal has parents - add it as a child to the first parent
			parentID := parentGoals[0].ID
			if parentNode, exists := gv.goalNodes[parentID]; exists {
				parentNode.Children = append(parentNode.Children, goalID)
				gv.goalNodes[goalID].Parent = parentID
			}
		} else {
			// This is a root goal (no parents)
			gv.rootGoals = append(gv.rootGoals, goalID)
		}
	}
}

// applyFiltersAndSort applies current filters and sorting to the tree display.
func (gv *GoalsView) applyFiltersAndSort() {
	// Return early if goals are not loaded yet
	if gv.goals == nil {
		return
	}

	// Filter goals based on search and status
	filteredGoals := []*core.Goal{}

	for _, goal := range gv.goals {
		// Apply status filter
		if gv.statusFilter != "" && goal.Status != gv.statusFilter {
			continue
		}

		// Apply search filter
		if gv.searchFilter != "" {
			searchLower := strings.ToLower(gv.searchFilter)
			if !strings.Contains(strings.ToLower(goal.Title), searchLower) &&
			   !strings.Contains(strings.ToLower(goal.Description), searchLower) {
				continue
			}
		}

		filteredGoals = append(filteredGoals, goal)
	}

	// Update goal nodes with filtered goals
	gv.updateFilteredGoalNodes(filteredGoals)

	// Apply sorting to root goals
	gv.sortRootGoals()

	// Refresh tree display
	gv.goalsTree.Refresh()
}

// updateFilteredGoalNodes updates the tree structure with filtered goals.
func (gv *GoalsView) updateFilteredGoalNodes(filteredGoals []*core.Goal) {
	// Create a set of filtered goal IDs for quick lookup
	filteredIDs := make(map[string]bool)
	for _, goal := range filteredGoals {
		filteredIDs[goal.ID] = true
	}

	// Update root goals list to only include filtered goals
	filteredRoots := []string{}
	for _, rootID := range gv.rootGoals {
		if filteredIDs[rootID] {
			filteredRoots = append(filteredRoots, rootID)
		}
	}
	gv.rootGoals = filteredRoots

	// Update children lists to only include filtered goals
	for goalID, node := range gv.goalNodes {
		if !filteredIDs[goalID] {
			continue
		}

		filteredChildren := []string{}
		for _, childID := range node.Children {
			if filteredIDs[childID] {
				filteredChildren = append(filteredChildren, childID)
			}
		}
		node.Children = filteredChildren
	}
}

// sortRootGoals sorts the root goals based on the current sort mode.
func (gv *GoalsView) sortRootGoals() {
	sort.Slice(gv.rootGoals, func(i, j int) bool {
		goalA := gv.goalNodes[gv.rootGoals[i]].Goal
		goalB := gv.goalNodes[gv.rootGoals[j]].Goal

		switch gv.sortMode {
		case "priority":
			return goalA.Priority < goalB.Priority // Lower number = higher priority
		case "title":
			return goalA.Title < goalB.Title
		case "created":
			return goalA.CreatedAt.Before(goalB.CreatedAt)
		case "status":
			return string(goalA.Status) < string(goalB.Status)
		default:
			return goalA.Priority < goalB.Priority
		}
	})

	// Also sort children for each goal
	for _, node := range gv.goalNodes {
		gv.sortGoalChildren(node)
	}
}

// sortGoalChildren sorts the children of a specific goal node.
func (gv *GoalsView) sortGoalChildren(node *GoalTreeNode) {
	sort.Slice(node.Children, func(i, j int) bool {
		goalA := gv.goalNodes[node.Children[i]].Goal
		goalB := gv.goalNodes[node.Children[j]].Goal

		switch gv.sortMode {
		case "priority":
			return goalA.Priority < goalB.Priority
		case "title":
			return goalA.Title < goalB.Title
		case "created":
			return goalA.CreatedAt.Before(goalB.CreatedAt)
		case "status":
			return string(goalA.Status) < string(goalB.Status)
		default:
			return goalA.Priority < goalB.Priority
		}
	})
}

// updateStatusBar updates the status bar with the given message.
func (gv *GoalsView) updateStatusBar(message ...string) {
	if len(message) > 0 {
		gv.statusLabel.SetText(message[0])
	} else if gv.selectedGoalID != "" {
		if node, exists := gv.goalNodes[gv.selectedGoalID]; exists {
			goal := node.Goal
			status := fmt.Sprintf("Selected: %s | Status: %s | Priority: %d | Created: %s",
				goal.Title, goal.Status, goal.Priority, goal.CreatedAt.Format("2006-01-02"))
			gv.statusLabel.SetText(status)
		}
	} else {
		gv.statusLabel.SetText("Ready")
	}
}

// Dialog methods (these will call methods in goal_dialog.go)

// showCreateGoalDialog shows the dialog for creating a new goal.
func (gv *GoalsView) showCreateGoalDialog() {
	dialog := NewGoalDialog(gv.app, gv.parent, GoalDialogModeCreate, nil)
	dialog.OnGoalSaved = func(goal *core.Goal) {
		gv.refreshData() // Refresh the view after creating a goal
	}
	dialog.Show()
}

// showEditGoalDialog shows the dialog for editing an existing goal.
func (gv *GoalsView) showEditGoalDialog(goalID string) {
	ctx := gv.app.GetContext()
	goal, err := gv.app.GetGoalManager().GetGoal(ctx, goalID)
	if err != nil {
		log.Printf("Failed to get goal for editing: %v", err)
		return
	}

	dialog := NewGoalDialog(gv.app, gv.parent, GoalDialogModeEdit, goal)
	dialog.OnGoalSaved = func(updatedGoal *core.Goal) {
		gv.refreshData() // Refresh the view after updating a goal
	}
	dialog.Show()
}

// archiveGoal archives the specified goal.
func (gv *GoalsView) archiveGoal(goalID string) {
	ctx := gv.app.GetContext()

	// Update goal status to archived
	archiveStatus := core.GoalStatusArchived
	updates := core.GoalUpdates{
		Status: &archiveStatus,
	}

	_, err := gv.app.GetGoalManager().UpdateGoal(ctx, goalID, updates)
	if err != nil {
		log.Printf("Failed to archive goal: %v", err)
		gv.updateStatusBar("Error archiving goal")
		return
	}

	gv.refreshData() // Refresh to show updated status
	gv.updateStatusBar("Goal archived")
}

// GetContainer returns the main container widget for this view.
func (gv *GoalsView) GetContainer() *fyne.Container {
	return gv.container
}