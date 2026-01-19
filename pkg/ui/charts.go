package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ChartColors provides consistent color palette for charts
var ChartColors = []color.Color{
	color.RGBA{R: 0x4A, G: 0x90, B: 0xE2, A: 255}, // Blue
	color.RGBA{R: 0x50, G: 0xE3, B: 0xA2, A: 255}, // Green
	color.RGBA{R: 0xFF, G: 0x7F, B: 0x50, A: 255}, // Orange
	color.RGBA{R: 0xE7, G: 0x5A, B: 0x7E, A: 255}, // Red
	color.RGBA{R: 0xB8, G: 0x7A, B: 0xE5, A: 255}, // Purple
	color.RGBA{R: 0xFF, G: 0xD1, B: 0x3D, A: 255}, // Yellow
}

// ProgressBar creates a simple horizontal progress bar with label
type ProgressBar struct {
	*widget.Card
	progress float64 // 0-100
	label    *widget.Label
	bar      *canvas.Rectangle
}

// NewProgressBar creates a new progress bar with title and current progress
func NewProgressBar(title string, progress float64) *ProgressBar {
	if progress > 100 {
		progress = 100
	}
	if progress < 0 {
		progress = 0
	}

	pb := &ProgressBar{
		progress: progress,
		label:    widget.NewLabel(fmt.Sprintf("%.1f%%", progress)),
	}

	// Create background rectangle
	background := canvas.NewRectangle(theme.ShadowColor())
	background.Resize(fyne.NewSize(200, 20))

	// Create progress rectangle
	pb.bar = canvas.NewRectangle(color.RGBA{R: 0x4A, G: 0x90, B: 0xE2, A: 255})
	progressWidth := float32(progress) / 100.0 * 200
	pb.bar.Resize(fyne.NewSize(progressWidth, 20))

	// Color coding based on progress
	if progress >= 90 {
		pb.bar.FillColor = color.RGBA{R: 0xE7, G: 0x5A, B: 0x7E, A: 255} // Red
	} else if progress >= 75 {
		pb.bar.FillColor = color.RGBA{R: 0xFF, G: 0x7F, B: 0x50, A: 255} // Orange
	} else {
		pb.bar.FillColor = color.RGBA{R: 0x50, G: 0xE3, B: 0xA2, A: 255} // Green
	}

	// Stack rectangles
	progressContainer := container.NewWithoutLayout(background, pb.bar)
	background.Move(fyne.NewPos(0, 0))
	pb.bar.Move(fyne.NewPos(0, 0))

	content := container.NewHBox(
		progressContainer,
		widget.NewLabel(" "),
		pb.label,
	)

	pb.Card = widget.NewCard(title, "", content)
	return pb
}

// UpdateProgress updates the progress bar value and color
func (pb *ProgressBar) UpdateProgress(progress float64) {
	if progress > 100 {
		progress = 100
	}
	if progress < 0 {
		progress = 0
	}

	pb.progress = progress
	pb.label.SetText(fmt.Sprintf("%.1f%%", progress))

	// Update bar width
	progressWidth := float32(progress) / 100.0 * 200
	pb.bar.Resize(fyne.NewSize(progressWidth, 20))

	// Update color based on progress
	if progress >= 90 {
		pb.bar.FillColor = color.RGBA{R: 0xE7, G: 0x5A, B: 0x7E, A: 255} // Red
	} else if progress >= 75 {
		pb.bar.FillColor = color.RGBA{R: 0xFF, G: 0x7F, B: 0x50, A: 255} // Orange
	} else {
		pb.bar.FillColor = color.RGBA{R: 0x50, G: 0xE3, B: 0xA2, A: 255} // Green
	}

	pb.bar.Refresh()
}

// SimpleBarChart creates a basic bar chart
type SimpleBarChart struct {
	*widget.Card
	data   map[string]float64
	maxVal float64
	bars   []*canvas.Rectangle
	labels []*widget.Label
}

// NewSimpleBarChart creates a new bar chart with the given data
func NewSimpleBarChart(title string, data map[string]float64) *SimpleBarChart {
	sbc := &SimpleBarChart{
		data: data,
	}

	// Find max value for scaling
	sbc.maxVal = 0
	for _, val := range data {
		if val > sbc.maxVal {
			sbc.maxVal = val
		}
	}

	if sbc.maxVal == 0 {
		sbc.maxVal = 1 // Avoid division by zero
	}

	// Create bars and labels
	content := container.NewVBox()
	i := 0
	for label, value := range data {
		barHeight := float32(value/sbc.maxVal) * 100 // Max 100 pixels height
		if barHeight < 5 {
			barHeight = 5 // Minimum visible height
		}

		bar := canvas.NewRectangle(ChartColors[i%len(ChartColors)])
		bar.Resize(fyne.NewSize(30, barHeight))

		labelWidget := widget.NewLabel(label)
		valueLabel := widget.NewLabel(fmt.Sprintf("%.1f", value))

		barContainer := container.NewVBox(
			container.NewCenter(bar),
			container.NewCenter(valueLabel),
			container.NewCenter(labelWidget),
		)

		content.Add(barContainer)
		sbc.bars = append(sbc.bars, bar)
		sbc.labels = append(sbc.labels, labelWidget)
		i++
	}

	if len(data) == 0 {
		content.Add(widget.NewLabel("No data available"))
	}

	scrollContent := container.NewScroll(content)
	scrollContent.SetMinSize(fyne.NewSize(300, 200))

	sbc.Card = widget.NewCard(title, "", scrollContent)
	return sbc
}

// UpdateData updates the bar chart with new data
func (sbc *SimpleBarChart) UpdateData(data map[string]float64) {
	sbc.data = data

	// Find new max value
	sbc.maxVal = 0
	for _, val := range data {
		if val > sbc.maxVal {
			sbc.maxVal = val
		}
	}

	if sbc.maxVal == 0 {
		sbc.maxVal = 1
	}

	// Update existing bars or recreate chart
	// For simplicity, we'll recreate the chart content
	content := container.NewVBox()
	i := 0
	for label, value := range data {
		barHeight := float32(value/sbc.maxVal) * 100
		if barHeight < 5 {
			barHeight = 5
		}

		bar := canvas.NewRectangle(ChartColors[i%len(ChartColors)])
		bar.Resize(fyne.NewSize(30, barHeight))

		labelWidget := widget.NewLabel(label)
		valueLabel := widget.NewLabel(fmt.Sprintf("%.1f", value))

		barContainer := container.NewVBox(
			container.NewCenter(bar),
			container.NewCenter(valueLabel),
			container.NewCenter(labelWidget),
		)

		content.Add(barContainer)
		i++
	}

	if len(data) == 0 {
		content.Add(widget.NewLabel("No data available"))
	}

	scrollContent := container.NewScroll(content)
	scrollContent.SetMinSize(fyne.NewSize(300, 200))

	sbc.Card.SetContent(scrollContent)
	sbc.Card.Refresh()
}

// SimplePieChart creates a basic pie chart representation
type SimplePieChart struct {
	*widget.Card
	data map[string]float64
}

// NewSimplePieChart creates a new pie chart (simplified as list for now)
func NewSimplePieChart(title string, data map[string]float64) *SimplePieChart {
	spc := &SimplePieChart{
		data: data,
	}

	// Calculate total for percentages
	total := 0.0
	for _, val := range data {
		total += val
	}

	content := container.NewVBox()

	if total == 0 {
		content.Add(widget.NewLabel("No data available"))
	} else {
		i := 0
		for label, value := range data {
			percentage := (value / total) * 100

			// Create colored indicator
			indicator := canvas.NewRectangle(ChartColors[i%len(ChartColors)])
			indicator.Resize(fyne.NewSize(15, 15))

			// Create label with value and percentage
			labelText := fmt.Sprintf("%s: %.1f (%.1f%%)", label, value, percentage)
			textLabel := widget.NewLabel(labelText)

			row := container.NewHBox(indicator, textLabel)
			content.Add(row)
			i++
		}
	}

	scrollContent := container.NewScroll(content)
	scrollContent.SetMinSize(fyne.NewSize(300, 150))

	spc.Card = widget.NewCard(title, "", scrollContent)
	return spc
}

// UpdateData updates the pie chart with new data
func (spc *SimplePieChart) UpdateData(data map[string]float64) {
	spc.data = data

	// Calculate total for percentages
	total := 0.0
	for _, val := range data {
		total += val
	}

	content := container.NewVBox()

	if total == 0 {
		content.Add(widget.NewLabel("No data available"))
	} else {
		i := 0
		for label, value := range data {
			percentage := (value / total) * 100

			// Create colored indicator
			indicator := canvas.NewRectangle(ChartColors[i%len(ChartColors)])
			indicator.Resize(fyne.NewSize(15, 15))

			// Create label with value and percentage
			labelText := fmt.Sprintf("%s: %.1f (%.1f%%)", label, value, percentage)
			textLabel := widget.NewLabel(labelText)

			row := container.NewHBox(indicator, textLabel)
			content.Add(row)
			i++
		}
	}

	scrollContent := container.NewScroll(content)
	scrollContent.SetMinSize(fyne.NewSize(300, 150))

	spc.Card.SetContent(scrollContent)
	spc.Card.Refresh()
}

// MetricCard creates a simple card displaying a metric with optional trend
type MetricCard struct {
	*widget.Card
	value     *widget.Label
	trend     *widget.Label
	indicator *canvas.Circle
}

// NewMetricCard creates a new metric display card
func NewMetricCard(title, value, trend string, isPositive bool) *MetricCard {
	mc := &MetricCard{
		value: widget.NewLabel(value),
		trend: widget.NewLabel(trend),
	}

	// Style the main value
	mc.value.TextStyle = fyne.TextStyle{Bold: true}

	// Create trend indicator
	var indicatorColor color.Color
	if isPositive {
		indicatorColor = color.RGBA{R: 0x50, G: 0xE3, B: 0xA2, A: 255} // Green
	} else {
		indicatorColor = color.RGBA{R: 0xE7, G: 0x5A, B: 0x7E, A: 255} // Red
	}

	mc.indicator = canvas.NewCircle(indicatorColor)
	mc.indicator.Resize(fyne.NewSize(10, 10))

	content := container.NewVBox(
		mc.value,
		container.NewHBox(mc.indicator, mc.trend),
	)

	mc.Card = widget.NewCard(title, "", content)
	return mc
}

// UpdateMetric updates the metric card with new values
func (mc *MetricCard) UpdateMetric(value, trend string, isPositive bool) {
	mc.value.SetText(value)
	mc.trend.SetText(trend)

	if isPositive {
		mc.indicator.FillColor = color.RGBA{R: 0x50, G: 0xE3, B: 0xA2, A: 255} // Green
	} else {
		mc.indicator.FillColor = color.RGBA{R: 0xE7, G: 0x5A, B: 0x7E, A: 255} // Red
	}

	mc.indicator.Refresh()
	mc.Card.Refresh()
}

// TimelineChart creates a simple timeline visualization
type TimelineChart struct {
	*widget.Card
	events []TimelineEvent
}

// TimelineEvent represents an event in the timeline
type TimelineEvent struct {
	Title       string
	Description string
	Time        string
	IsSuccess   bool
}

// NewTimelineChart creates a new timeline chart
func NewTimelineChart(title string, events []TimelineEvent) *TimelineChart {
	tc := &TimelineChart{
		events: events,
	}

	content := container.NewVBox()

	if len(events) == 0 {
		content.Add(widget.NewLabel("No recent activity"))
	} else {
		for _, event := range events {
			// Create status indicator
			var indicatorColor color.Color
			if event.IsSuccess {
				indicatorColor = color.RGBA{R: 0x50, G: 0xE3, B: 0xA2, A: 255} // Green
			} else {
				indicatorColor = color.RGBA{R: 0xE7, G: 0x5A, B: 0x7E, A: 255} // Red
			}

			indicator := canvas.NewCircle(indicatorColor)
			indicator.Resize(fyne.NewSize(8, 8))

			// Create event content
			titleLabel := widget.NewLabel(event.Title)
			titleLabel.TextStyle = fyne.TextStyle{Bold: true}

			descLabel := widget.NewLabel(event.Description)
			timeLabel := widget.NewLabel(event.Time)
			timeLabel.TextStyle = fyne.TextStyle{Italic: true}

			eventContent := container.NewVBox(titleLabel, descLabel, timeLabel)
			eventRow := container.NewHBox(indicator, eventContent)

			content.Add(eventRow)
			content.Add(widget.NewSeparator())
		}
	}

	scrollContent := container.NewScroll(content)
	scrollContent.SetMinSize(fyne.NewSize(400, 300))

	tc.Card = widget.NewCard(title, "", scrollContent)
	return tc
}

// UpdateEvents updates the timeline with new events
func (tc *TimelineChart) UpdateEvents(events []TimelineEvent) {
	tc.events = events

	content := container.NewVBox()

	if len(events) == 0 {
		content.Add(widget.NewLabel("No recent activity"))
	} else {
		for _, event := range events {
			var indicatorColor color.Color
			if event.IsSuccess {
				indicatorColor = color.RGBA{R: 0x50, G: 0xE3, B: 0xA2, A: 255}
			} else {
				indicatorColor = color.RGBA{R: 0xE7, G: 0x5A, B: 0x7E, A: 255}
			}

			indicator := canvas.NewCircle(indicatorColor)
			indicator.Resize(fyne.NewSize(8, 8))

			titleLabel := widget.NewLabel(event.Title)
			titleLabel.TextStyle = fyne.TextStyle{Bold: true}

			descLabel := widget.NewLabel(event.Description)
			timeLabel := widget.NewLabel(event.Time)
			timeLabel.TextStyle = fyne.TextStyle{Italic: true}

			eventContent := container.NewVBox(titleLabel, descLabel, timeLabel)
			eventRow := container.NewHBox(indicator, eventContent)

			content.Add(eventRow)
			content.Add(widget.NewSeparator())
		}
	}

	scrollContent := container.NewScroll(content)
	scrollContent.SetMinSize(fyne.NewSize(400, 300))

	tc.Card.SetContent(scrollContent)
	tc.Card.Refresh()
}