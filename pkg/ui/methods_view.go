package ui

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/yourusername/ai-work-studio/pkg/core"
)

// MethodsView provides a comprehensive interface for viewing and managing methods
type MethodsView struct {
	app    *App
	window fyne.Window

	// UI Components
	container     *container.Split
	searchEntry   *widget.Entry
	domainSelect  *widget.Select
	statusSelect  *widget.Select
	methodsList   *widget.List
	detailsView   *container.AppTabs
	refreshBtn    *widget.Button

	// Data
	methods        []*core.Method
	filteredMethods []*core.Method
	selectedMethod *core.Method
}

// NewMethodsView creates a new methods view
func NewMethodsView(app *App, window fyne.Window) *MethodsView {
	mv := &MethodsView{
		app:    app,
		window: window,
	}

	mv.createUI()
	mv.loadMethods()

	return mv
}

// createUI initializes the user interface components
func (mv *MethodsView) createUI() {
	// Create search and filter controls
	mv.searchEntry = widget.NewEntry()
	mv.searchEntry.SetPlaceHolder("Search methods by name or description...")
	mv.searchEntry.OnChanged = mv.onSearchChanged

	// Domain filter
	mv.domainSelect = widget.NewSelect(
		[]string{"All Domains", "General", "Domain Specific", "User Specific"},
		mv.onDomainChanged,
	)
	mv.domainSelect.SetSelected("All Domains")

	// Status filter
	mv.statusSelect = widget.NewSelect(
		[]string{"All Status", "Active", "Deprecated", "Superseded"},
		mv.onStatusChanged,
	)
	mv.statusSelect.SetSelected("All Status")

	// Refresh button
	mv.refreshBtn = widget.NewButton("Refresh", mv.onRefresh)

	// Create controls container
	controlsContainer := container.NewHBox(
		mv.searchEntry,
		mv.domainSelect,
		mv.statusSelect,
		mv.refreshBtn,
	)

	// Create methods list
	mv.methodsList = widget.NewList(
		mv.methodsListLength,
		mv.createMethodsListItem,
		mv.updateMethodsListItem,
	)
	mv.methodsList.OnSelected = mv.onMethodSelected

	// Create master panel
	masterPanel := container.NewBorder(
		controlsContainer, // top
		nil,               // bottom
		nil,               // left
		nil,               // right
		mv.methodsList,    // center
	)

	// Create details view
	mv.detailsView = container.NewAppTabs()
	mv.createDetailsPanel()

	// Create split layout
	mv.container = container.NewHSplit(masterPanel, mv.detailsView)
	mv.container.SetOffset(0.4) // 40% for list, 60% for details
}

// createDetailsPanel creates the method details panel
func (mv *MethodsView) createDetailsPanel() {
	// Overview Tab
	overviewTab := container.NewTabItem("Overview", mv.createOverviewPanel())
	mv.detailsView.Append(overviewTab)

	// Approach Tab
	approachTab := container.NewTabItem("Approach", mv.createApproachPanel())
	mv.detailsView.Append(approachTab)

	// History Tab
	historyTab := container.NewTabItem("History", mv.createHistoryPanel())
	mv.detailsView.Append(historyTab)

	// Metrics Tab
	metricsTab := container.NewTabItem("Metrics", mv.createMetricsPanel())
	mv.detailsView.Append(metricsTab)
}

// createOverviewPanel creates the overview details panel
func (mv *MethodsView) createOverviewPanel() fyne.CanvasObject {
	content := container.NewVBox(
		widget.NewLabel("Select a method to view details"),
	)
	return container.NewScroll(content)
}

// createApproachPanel creates the approach details panel
func (mv *MethodsView) createApproachPanel() fyne.CanvasObject {
	content := container.NewVBox(
		widget.NewLabel("Select a method to view approach"),
	)
	return container.NewScroll(content)
}

// createHistoryPanel creates the history panel
func (mv *MethodsView) createHistoryPanel() fyne.CanvasObject {
	content := container.NewVBox(
		widget.NewLabel("Select a method to view evolution history"),
	)
	return container.NewScroll(content)
}

// createMetricsPanel creates the metrics panel
func (mv *MethodsView) createMetricsPanel() fyne.CanvasObject {
	content := container.NewVBox(
		widget.NewLabel("Select a method to view metrics"),
	)
	return container.NewScroll(content)
}

// methodsListLength returns the number of filtered methods
func (mv *MethodsView) methodsListLength() int {
	return len(mv.filteredMethods)
}

// createMethodsListItem creates a new list item widget
func (mv *MethodsView) createMethodsListItem() fyne.CanvasObject {
	nameLabel := widget.NewLabel("")
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	domainLabel := widget.NewLabel("")
	successLabel := widget.NewLabel("")
	lastUsedLabel := widget.NewLabel("")

	// Create compact layout for list item
	return container.NewVBox(
		nameLabel,
		container.NewHBox(
			widget.NewLabel("Domain:"), domainLabel,
			widget.NewLabel("Success:"), successLabel,
			widget.NewLabel("Last Used:"), lastUsedLabel,
		),
		widget.NewSeparator(),
	)
}

// updateMethodsListItem updates a list item with method data
func (mv *MethodsView) updateMethodsListItem(id widget.ListItemID, item fyne.CanvasObject) {
	if id >= len(mv.filteredMethods) {
		return
	}

	method := mv.filteredMethods[id]
	vboxContainer := item.(*fyne.Container)

	// Update name label
	nameLabel := vboxContainer.Objects[0].(*widget.Label)
	nameLabel.SetText(method.Name)

	// Update details container
	detailsContainer := vboxContainer.Objects[1].(*fyne.Container)

	// Domain
	domainLabel := detailsContainer.Objects[1].(*widget.Label)
	domainLabel.SetText(string(method.Domain))

	// Success rate
	successLabel := detailsContainer.Objects[3].(*widget.Label)
	successLabel.SetText(fmt.Sprintf("%.1f%%", method.Metrics.SuccessRate()))

	// Last used
	lastUsedLabel := detailsContainer.Objects[5].(*widget.Label)
	if method.Metrics.LastUsed.IsZero() {
		lastUsedLabel.SetText("Never")
	} else {
		lastUsedLabel.SetText(method.Metrics.LastUsed.Format("Jan 2, 2006"))
	}
}

// loadMethods loads methods from the method manager
func (mv *MethodsView) loadMethods() {
	ctx := mv.app.GetContext()
	methodManager := mv.app.GetMethodManager()

	methods, err := methodManager.ListMethods(ctx, core.MethodFilter{})
	if err != nil {
		log.Printf("Error loading methods: %v", err)
		mv.app.ShowError("Error", "Failed to load methods: "+err.Error())
		return
	}

	mv.methods = methods
	mv.applyFilters()
}

// applyFilters applies current search and filter criteria
func (mv *MethodsView) applyFilters() {
	// Guard against nil components during initialization
	if mv.searchEntry == nil || mv.domainSelect == nil || mv.statusSelect == nil || mv.methodsList == nil {
		return
	}

	searchText := strings.ToLower(mv.searchEntry.Text)
	selectedDomain := mv.domainSelect.Selected
	selectedStatus := mv.statusSelect.Selected

	mv.filteredMethods = make([]*core.Method, 0)

	for _, method := range mv.methods {
		// Apply search filter
		if searchText != "" {
			nameMatch := strings.Contains(strings.ToLower(method.Name), searchText)
			descMatch := strings.Contains(strings.ToLower(method.Description), searchText)
			if !nameMatch && !descMatch {
				continue
			}
		}

		// Apply domain filter
		if selectedDomain != "All Domains" {
			expectedDomain := mv.domainSelectToEnum(selectedDomain)
			if method.Domain != expectedDomain {
				continue
			}
		}

		// Apply status filter
		if selectedStatus != "All Status" {
			expectedStatus := mv.statusSelectToEnum(selectedStatus)
			if method.Status != expectedStatus {
				continue
			}
		}

		mv.filteredMethods = append(mv.filteredMethods, method)
	}

	// Sort by success rate descending, then by name
	sort.Slice(mv.filteredMethods, func(i, j int) bool {
		iRate := mv.filteredMethods[i].Metrics.SuccessRate()
		jRate := mv.filteredMethods[j].Metrics.SuccessRate()

		if iRate != jRate {
			return iRate > jRate
		}
		return mv.filteredMethods[i].Name < mv.filteredMethods[j].Name
	})

	mv.methodsList.Refresh()
}

// domainSelectToEnum converts domain selection to enum
func (mv *MethodsView) domainSelectToEnum(selection string) core.MethodDomain {
	switch selection {
	case "General":
		return core.MethodDomainGeneral
	case "Domain Specific":
		return core.MethodDomainSpecific
	case "User Specific":
		return core.MethodDomainUser
	default:
		return core.MethodDomainGeneral
	}
}

// statusSelectToEnum converts status selection to enum
func (mv *MethodsView) statusSelectToEnum(selection string) core.MethodStatus {
	switch selection {
	case "Active":
		return core.MethodStatusActive
	case "Deprecated":
		return core.MethodStatusDeprecated
	case "Superseded":
		return core.MethodStatusSuperseded
	default:
		return core.MethodStatusActive
	}
}

// Event handlers

// onSearchChanged handles search text changes
func (mv *MethodsView) onSearchChanged(text string) {
	mv.applyFilters()
}

// onDomainChanged handles domain filter changes
func (mv *MethodsView) onDomainChanged(selection string) {
	mv.applyFilters()
}

// onStatusChanged handles status filter changes
func (mv *MethodsView) onStatusChanged(selection string) {
	mv.applyFilters()
}

// onRefresh handles the refresh button
func (mv *MethodsView) onRefresh() {
	mv.loadMethods()
}

// onMethodSelected handles method selection in the list
func (mv *MethodsView) onMethodSelected(id widget.ListItemID) {
	if id >= len(mv.filteredMethods) {
		return
	}

	mv.selectedMethod = mv.filteredMethods[id]
	mv.updateDetailsView()
}

// updateDetailsView updates the details panel with selected method
func (mv *MethodsView) updateDetailsView() {
	if mv.selectedMethod == nil {
		return
	}

	// Update Overview Tab
	mv.updateOverviewTab()

	// Update Approach Tab
	mv.updateApproachTab()

	// Update History Tab
	mv.updateHistoryTab()

	// Update Metrics Tab
	mv.updateMetricsTab()
}

// updateOverviewTab updates the overview tab with method details
func (mv *MethodsView) updateOverviewTab() {
	method := mv.selectedMethod

	nameLabel := widget.NewLabel(method.Name)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	content := container.NewVBox(
		nameLabel,
		widget.NewSeparator(),
		widget.NewLabel("Description:"),
		widget.NewLabel(method.Description),
		widget.NewSeparator(),
		container.NewHBox(
			widget.NewLabel("Domain:"), widget.NewLabel(string(method.Domain)),
		),
		container.NewHBox(
			widget.NewLabel("Version:"), widget.NewLabel(method.Version),
		),
		container.NewHBox(
			widget.NewLabel("Status:"), widget.NewLabel(string(method.Status)),
		),
		container.NewHBox(
			widget.NewLabel("Created:"), widget.NewLabel(method.CreatedAt.Format("Jan 2, 2006 15:04")),
		),
		widget.NewSeparator(),
		widget.NewLabel("Success Metrics:"),
		container.NewHBox(
			widget.NewLabel("Executions:"), widget.NewLabel(fmt.Sprintf("%d", method.Metrics.ExecutionCount)),
		),
		container.NewHBox(
			widget.NewLabel("Success Rate:"), widget.NewLabel(fmt.Sprintf("%.1f%%", method.Metrics.SuccessRate())),
		),
		container.NewHBox(
			widget.NewLabel("Average Rating:"), widget.NewLabel(fmt.Sprintf("%.1f/10", method.Metrics.AverageRating)),
		),
		container.NewHBox(
			widget.NewLabel("Last Used:"), widget.NewLabel(mv.formatLastUsed(method.Metrics.LastUsed)),
		),
	)

	scrollContent := container.NewScroll(content)
	mv.detailsView.Items[0].Content = scrollContent
	mv.detailsView.Refresh()
}

// updateApproachTab updates the approach tab with method steps
func (mv *MethodsView) updateApproachTab() {
	method := mv.selectedMethod

	content := container.NewVBox()

	if len(method.Approach) == 0 {
		content.Add(widget.NewLabel("No approach steps defined"))
	} else {
		content.Add(widget.NewLabel("Method Approach:"))
		content.Add(widget.NewSeparator())

		for i, step := range method.Approach {
			stepLabel := widget.NewLabel(fmt.Sprintf("Step %d:", i+1))
			stepLabel.TextStyle = fyne.TextStyle{Bold: true}
			content.Add(stepLabel)

			content.Add(widget.NewLabel(step.Description))

			if len(step.Tools) > 0 {
				content.Add(widget.NewLabel("Tools: " + strings.Join(step.Tools, ", ")))
			}

			if len(step.Heuristics) > 0 {
				content.Add(widget.NewLabel("Heuristics:"))
				for _, heuristic := range step.Heuristics {
					content.Add(widget.NewLabel("  • " + heuristic))
				}
			}

			if i < len(method.Approach)-1 {
				content.Add(widget.NewSeparator())
			}
		}
	}

	scrollContent := container.NewScroll(content)
	mv.detailsView.Items[1].Content = scrollContent
	mv.detailsView.Refresh()
}

// updateHistoryTab updates the history tab with evolution information
func (mv *MethodsView) updateHistoryTab() {
	// For now, show basic version info
	// In the future, this could show full evolution chain from GetMethodEvolution
	method := mv.selectedMethod

	content := container.NewVBox(
		widget.NewLabel("Version History:"),
		widget.NewSeparator(),
		container.NewHBox(
			widget.NewLabel("Current Version:"), widget.NewLabel(method.Version),
		),
		container.NewHBox(
			widget.NewLabel("Created:"), widget.NewLabel(method.CreatedAt.Format("Jan 2, 2006 15:04")),
		),
		widget.NewSeparator(),
		widget.NewLabel("Evolution tracking will be available in a future version."),
	)

	scrollContent := container.NewScroll(content)
	mv.detailsView.Items[2].Content = scrollContent
	mv.detailsView.Refresh()
}

// updateMetricsTab updates the metrics tab with visual charts
func (mv *MethodsView) updateMetricsTab() {
	method := mv.selectedMethod

	// Create success rate progress bar
	successBar := NewProgressBar("Success Rate", method.Metrics.SuccessRate())

	// Create rating progress bar (convert 1-10 to percentage)
	ratingPercentage := (method.Metrics.AverageRating / 10.0) * 100
	ratingBar := NewProgressBar("Average Rating", ratingPercentage)

	// Create execution count metric
	execCard := NewMetricCard(
		"Total Executions",
		fmt.Sprintf("%d", method.Metrics.ExecutionCount),
		fmt.Sprintf("%d successful", method.Metrics.SuccessCount),
		method.Metrics.SuccessRate() > 75,
	)

	// Create last used metric
	lastUsedCard := NewMetricCard(
		"Last Used",
		mv.formatLastUsed(method.Metrics.LastUsed),
		mv.getTimeSinceLastUsed(method.Metrics.LastUsed),
		!method.Metrics.LastUsed.IsZero(),
	)

	content := container.NewVBox(
		successBar.Card,
		ratingBar.Card,
		container.NewHBox(execCard.Card, lastUsedCard.Card),
	)

	mv.detailsView.Items[3].Content = content
	mv.detailsView.Refresh()
}

// Helper methods

// formatLastUsed formats the last used time
func (mv *MethodsView) formatLastUsed(lastUsed time.Time) string {
	if lastUsed.IsZero() {
		return "Never"
	}
	return lastUsed.Format("Jan 2, 2006 15:04")
}

// getTimeSinceLastUsed returns a human-readable time since last use
func (mv *MethodsView) getTimeSinceLastUsed(lastUsed time.Time) string {
	if lastUsed.IsZero() {
		return "Never used"
	}

	since := time.Since(lastUsed)

	if since.Hours() < 24 {
		return fmt.Sprintf("%.0f hours ago", since.Hours())
	} else if since.Hours() < 24*7 {
		return fmt.Sprintf("%.0f days ago", since.Hours()/24)
	} else if since.Hours() < 24*30 {
		return fmt.Sprintf("%.0f weeks ago", since.Hours()/(24*7))
	} else {
		return fmt.Sprintf("%.0f months ago", since.Hours()/(24*30))
	}
}

// GetContainer returns the main container for this view
func (mv *MethodsView) GetContainer() fyne.CanvasObject {
	return mv.container
}

// Refresh refreshes the methods data
func (mv *MethodsView) Refresh() {
	mv.loadMethods()
}