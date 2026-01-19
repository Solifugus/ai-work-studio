package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// BudgetPeriod represents different budget tracking periods.
type BudgetPeriod int

const (
	// PeriodDaily tracks daily spending
	PeriodDaily BudgetPeriod = iota
	// PeriodWeekly tracks weekly spending
	PeriodWeekly
	// PeriodMonthly tracks monthly spending
	PeriodMonthly
)

// AlertThreshold defines when to trigger budget alerts.
type AlertThreshold int

const (
	// ThresholdNone indicates no alert needed
	ThresholdNone AlertThreshold = iota
	// Threshold75 indicates 75% of budget used
	Threshold75
	// Threshold90 indicates 90% of budget used
	Threshold90
	// Threshold100 indicates 100% of budget used (exceeded)
	Threshold100
)

// BudgetConfig contains budget limits and alert configurations.
type BudgetConfig struct {
	// DailyLimit is the maximum daily spending allowed
	DailyLimit float64

	// WeeklyLimit is the maximum weekly spending allowed
	WeeklyLimit float64

	// MonthlyLimit is the maximum monthly spending allowed
	MonthlyLimit float64

	// AlertThresholds defines when to send alerts (default: 75%, 90%, 100%)
	AlertThresholds []float64

	// AlertCallback is called when thresholds are exceeded
	AlertCallback func(AlertInfo)

	// AutoStop prevents further requests when budget is exceeded
	AutoStop bool

	// GracePeriod allows small overages before hard stops
	GracePeriod float64

	// TrackingEnabled enables detailed expense tracking
	TrackingEnabled bool
}

// DefaultBudgetConfig returns sensible defaults for budget configuration.
func DefaultBudgetConfig() BudgetConfig {
	return BudgetConfig{
		DailyLimit:      5.0,   // $5 per day
		WeeklyLimit:     30.0,  // $30 per week
		MonthlyLimit:    100.0, // $100 per month
		AlertThresholds: []float64{75.0, 90.0, 100.0},
		AutoStop:        true,
		GracePeriod:     0.50, // $0.50 overage allowed
		TrackingEnabled: true,
	}
}

// BudgetManager provides comprehensive budget tracking and management.
type BudgetManager struct {
	config       BudgetConfig
	usage        *UsageTracker
	persistence  *BudgetPersistence
	alerts       *AlertManager
	mu           sync.RWMutex
	logger       *log.Logger
}

// UsageTracker tracks spending across different time periods.
type UsageTracker struct {
	Daily   map[string]float64 // date -> amount
	Weekly  map[string]float64 // week -> amount
	Monthly map[string]float64 // month -> amount

	// Detailed transaction log
	Transactions []Transaction

	// Provider and model breakdown
	ProviderSpending map[string]float64 // provider -> total spent
	ModelSpending    map[string]float64 // provider_model -> total spent
	TaskTypeSpending map[string]float64 // task_type -> total spent

	// Performance tracking for ROI analysis
	ProviderROI map[string]*ProviderROI // provider -> ROI metrics
}

// Transaction represents a single LLM request transaction.
type Transaction struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Provider    string    `json:"provider"`
	Model       string    `json:"model"`
	TaskType    string    `json:"task_type,omitempty"`
	TokensUsed  int       `json:"tokens_used"`
	Cost        float64   `json:"cost"`
	Success     bool      `json:"success"`
	Quality     float64   `json:"quality,omitempty"` // 1-10 rating
	Latency     int64     `json:"latency_ms"`        // milliseconds
	UserID      string    `json:"user_id,omitempty"`
}

// ProviderROI tracks return on investment metrics for each provider.
type ProviderROI struct {
	TotalSpent       float64 `json:"total_spent"`
	TotalRequests    int     `json:"total_requests"`
	SuccessfulReqs   int     `json:"successful_requests"`
	AverageQuality   float64 `json:"average_quality"`
	AverageLatency   float64 `json:"average_latency_ms"`
	CostPerSuccess   float64 `json:"cost_per_success"`
	QualityPerDollar float64 `json:"quality_per_dollar"`
	LastUpdated      time.Time `json:"last_updated"`
}

// AlertManager handles budget alert notifications.
type AlertManager struct {
	triggeredAlerts map[string]time.Time // threshold_period -> last alert time
	mu              sync.RWMutex
}

// BudgetPersistence handles saving/loading budget data.
type BudgetPersistence struct {
	dataPath string
}

// AlertInfo contains information about a budget alert.
type AlertInfo struct {
	Period        BudgetPeriod
	Threshold     float64
	CurrentUsage  float64
	BudgetLimit   float64
	OverageAmount float64
	Timestamp     time.Time
	Message       string
}

// NewBudgetManager creates a new budget manager with persistence.
func NewBudgetManager(dataPath string, config BudgetConfig, logger *log.Logger) (*BudgetManager, error) {
	if logger == nil {
		logger = log.New(os.Stdout, "[BudgetManager] ", log.LstdFlags)
	}

	// Ensure data directory exists
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	persistence := &BudgetPersistence{dataPath: dataPath}

	// Load existing usage data
	usage, err := persistence.LoadUsage()
	if err != nil {
		// If loading fails, start with fresh usage tracker
		logger.Printf("Warning: could not load existing usage data: %v. Starting fresh.", err)
		usage = &UsageTracker{
			Daily:            make(map[string]float64),
			Weekly:           make(map[string]float64),
			Monthly:          make(map[string]float64),
			Transactions:     make([]Transaction, 0),
			ProviderSpending: make(map[string]float64),
			ModelSpending:    make(map[string]float64),
			TaskTypeSpending: make(map[string]float64),
			ProviderROI:      make(map[string]*ProviderROI),
		}
	}

	manager := &BudgetManager{
		config:      config,
		usage:       usage,
		persistence: persistence,
		alerts: &AlertManager{
			triggeredAlerts: make(map[string]time.Time),
		},
		logger: logger,
	}

	return manager, nil
}

// RecordUsage records a new transaction and updates budget tracking.
func (bm *BudgetManager) RecordUsage(ctx context.Context, transaction Transaction) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// Set transaction timestamp if not provided
	if transaction.Timestamp.IsZero() {
		transaction.Timestamp = time.Now()
	}

	// Generate ID if not provided
	if transaction.ID == "" {
		transaction.ID = fmt.Sprintf("%d_%s_%s",
			transaction.Timestamp.Unix(),
			transaction.Provider,
			transaction.Model)
	}

	// Add to transaction log
	if bm.config.TrackingEnabled {
		bm.usage.Transactions = append(bm.usage.Transactions, transaction)
	}

	// Update time-based spending
	bm.updateTimeBasedSpending(transaction)

	// Update provider/model spending
	bm.updateProviderSpending(transaction)

	// Update ROI metrics
	bm.updateROIMetrics(transaction)

	// Check for budget alerts
	bm.checkBudgetAlerts(transaction.Timestamp)

	// Persist data
	if err := bm.persistence.SaveUsage(bm.usage); err != nil {
		bm.logger.Printf("Warning: failed to persist budget data: %v", err)
	}

	return nil
}

// updateTimeBasedSpending updates daily, weekly, and monthly spending totals.
func (bm *BudgetManager) updateTimeBasedSpending(tx Transaction) {
	date := tx.Timestamp.Format("2006-01-02")
	week := bm.getWeekKey(tx.Timestamp)
	month := tx.Timestamp.Format("2006-01")

	bm.usage.Daily[date] += tx.Cost
	bm.usage.Weekly[week] += tx.Cost
	bm.usage.Monthly[month] += tx.Cost
}

// updateProviderSpending updates provider and model spending breakdowns.
func (bm *BudgetManager) updateProviderSpending(tx Transaction) {
	bm.usage.ProviderSpending[tx.Provider] += tx.Cost

	modelKey := fmt.Sprintf("%s_%s", tx.Provider, tx.Model)
	bm.usage.ModelSpending[modelKey] += tx.Cost

	if tx.TaskType != "" {
		bm.usage.TaskTypeSpending[tx.TaskType] += tx.Cost
	}
}

// updateROIMetrics updates return on investment metrics for providers.
func (bm *BudgetManager) updateROIMetrics(tx Transaction) {
	roi, exists := bm.usage.ProviderROI[tx.Provider]
	if !exists {
		roi = &ProviderROI{
			TotalSpent:     0,
			TotalRequests:  0,
			SuccessfulReqs: 0,
		}
		bm.usage.ProviderROI[tx.Provider] = roi
	}

	roi.TotalSpent += tx.Cost
	roi.TotalRequests++

	if tx.Success {
		roi.SuccessfulReqs++
	}

	// Update average quality (if quality rating provided)
	if tx.Quality >= 1.0 && tx.Quality <= 10.0 {
		if roi.TotalRequests == 1 {
			roi.AverageQuality = tx.Quality
		} else {
			roi.AverageQuality = (roi.AverageQuality*float64(roi.TotalRequests-1) + tx.Quality) / float64(roi.TotalRequests)
		}
	}

	// Update average latency
	if roi.TotalRequests == 1 {
		roi.AverageLatency = float64(tx.Latency)
	} else {
		roi.AverageLatency = (roi.AverageLatency*float64(roi.TotalRequests-1) + float64(tx.Latency)) / float64(roi.TotalRequests)
	}

	// Calculate derived metrics
	if roi.SuccessfulReqs > 0 {
		roi.CostPerSuccess = roi.TotalSpent / float64(roi.SuccessfulReqs)
	}

	if roi.TotalSpent > 0 && roi.AverageQuality > 0 {
		roi.QualityPerDollar = roi.AverageQuality / roi.TotalSpent
	}

	roi.LastUpdated = time.Now()
}

// checkBudgetAlerts checks if any budget thresholds have been exceeded.
func (bm *BudgetManager) checkBudgetAlerts(timestamp time.Time) {
	// Check daily budget
	if bm.config.DailyLimit > 0 {
		bm.checkPeriodAlert(PeriodDaily, timestamp, bm.config.DailyLimit)
	}

	// Check weekly budget
	if bm.config.WeeklyLimit > 0 {
		bm.checkPeriodAlert(PeriodWeekly, timestamp, bm.config.WeeklyLimit)
	}

	// Check monthly budget
	if bm.config.MonthlyLimit > 0 {
		bm.checkPeriodAlert(PeriodMonthly, timestamp, bm.config.MonthlyLimit)
	}
}

// checkPeriodAlert checks if alerts should be triggered for a specific period.
func (bm *BudgetManager) checkPeriodAlert(period BudgetPeriod, timestamp time.Time, limit float64) {
	usage := bm.getCurrentUsage(period, timestamp)
	percentage := (usage / limit) * 100

	for _, threshold := range bm.config.AlertThresholds {
		if percentage >= threshold {
			alertKey := fmt.Sprintf("%.0f_%s_%s", threshold, period.String(), bm.getPeriodKey(period, timestamp))

			// Check if we've already alerted for this threshold in this period
			bm.alerts.mu.Lock()
			lastAlert, exists := bm.alerts.triggeredAlerts[alertKey]
			bm.alerts.mu.Unlock()

			if !exists || time.Since(lastAlert) > time.Hour {
				// Trigger alert
				alert := AlertInfo{
					Period:        period,
					Threshold:     threshold,
					CurrentUsage:  usage,
					BudgetLimit:   limit,
					OverageAmount: usage - limit,
					Timestamp:     timestamp,
					Message:       bm.formatAlertMessage(period, threshold, usage, limit),
				}

				// Mark alert as triggered
				bm.alerts.mu.Lock()
				bm.alerts.triggeredAlerts[alertKey] = timestamp
				bm.alerts.mu.Unlock()

				// Call alert callback if configured
				if bm.config.AlertCallback != nil {
					bm.config.AlertCallback(alert)
				}

				// Log the alert
				bm.logger.Printf("Budget Alert: %s", alert.Message)
			}
		}
	}
}

// getCurrentUsage gets the current usage for a specific period.
func (bm *BudgetManager) getCurrentUsage(period BudgetPeriod, timestamp time.Time) float64 {
	key := bm.getPeriodKey(period, timestamp)

	switch period {
	case PeriodDaily:
		return bm.usage.Daily[key]
	case PeriodWeekly:
		return bm.usage.Weekly[key]
	case PeriodMonthly:
		return bm.usage.Monthly[key]
	default:
		return 0.0
	}
}

// getPeriodKey gets the key for a specific period and timestamp.
func (bm *BudgetManager) getPeriodKey(period BudgetPeriod, timestamp time.Time) string {
	switch period {
	case PeriodDaily:
		return timestamp.Format("2006-01-02")
	case PeriodWeekly:
		return bm.getWeekKey(timestamp)
	case PeriodMonthly:
		return timestamp.Format("2006-01")
	default:
		return ""
	}
}

// getWeekKey generates a consistent week key for a timestamp.
func (bm *BudgetManager) getWeekKey(timestamp time.Time) string {
	year, week := timestamp.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

// formatAlertMessage creates a human-readable alert message.
func (bm *BudgetManager) formatAlertMessage(period BudgetPeriod, threshold, usage, limit float64) string {
	periodStr := period.String()

	if threshold >= 100 {
		return fmt.Sprintf("Budget exceeded! %s spending: $%.2f (limit: $%.2f, overage: $%.2f)",
			periodStr, usage, limit, usage-limit)
	} else {
		return fmt.Sprintf("Budget alert: %.0f%% of %s budget used ($%.2f of $%.2f)",
			threshold, periodStr, usage, limit)
	}
}

// CanAfford checks if a potential expense is within budget limits.
func (bm *BudgetManager) CanAfford(estimatedCost float64) (*AffordabilityCheck, error) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	now := time.Now()
	result := &AffordabilityCheck{
		EstimatedCost: estimatedCost,
		Timestamp:     now,
		Affordable:    true,
		Warnings:      make([]string, 0),
	}

	// Check each budget period
	periods := []struct {
		period BudgetPeriod
		limit  float64
	}{
		{PeriodDaily, bm.config.DailyLimit},
		{PeriodWeekly, bm.config.WeeklyLimit},
		{PeriodMonthly, bm.config.MonthlyLimit},
	}

	for _, p := range periods {
		if p.limit <= 0 {
			continue // No limit set for this period
		}

		currentUsage := bm.getCurrentUsage(p.period, now)
		projectedUsage := currentUsage + estimatedCost

		if projectedUsage > p.limit {
			result.Affordable = false
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("%s budget would be exceeded ($%.2f + $%.2f > $%.2f)",
					p.period.String(), currentUsage, estimatedCost, p.limit))
		} else if projectedUsage > p.limit*0.9 {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("%s budget would exceed 90%% ($%.2f + $%.2f = $%.2f of $%.2f)",
					p.period.String(), currentUsage, estimatedCost, projectedUsage, p.limit))
		}
	}

	// Apply auto-stop logic
	if !result.Affordable && bm.config.AutoStop {
		// Check if within grace period
		if estimatedCost <= bm.config.GracePeriod {
			result.Affordable = true
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Request allowed within grace period ($%.2f <= $%.2f)",
					estimatedCost, bm.config.GracePeriod))
		}
	}

	return result, nil
}

// AffordabilityCheck contains the result of a budget affordability check.
type AffordabilityCheck struct {
	EstimatedCost float64
	Affordable    bool
	Warnings      []string
	Timestamp     time.Time
}

// GetBudgetStatus returns current budget status across all periods.
func (bm *BudgetManager) GetBudgetStatus() *BudgetStatus {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	now := time.Now()
	status := &BudgetStatus{
		Timestamp: now,
		Periods:   make(map[string]*PeriodStatus),
	}

	// Get status for each period
	periods := map[string]struct {
		period BudgetPeriod
		limit  float64
	}{
		"daily":   {PeriodDaily, bm.config.DailyLimit},
		"weekly":  {PeriodWeekly, bm.config.WeeklyLimit},
		"monthly": {PeriodMonthly, bm.config.MonthlyLimit},
	}

	for name, p := range periods {
		if p.limit <= 0 {
			continue // Skip periods without limits
		}

		usage := bm.getCurrentUsage(p.period, now)
		percentage := (usage / p.limit) * 100

		status.Periods[name] = &PeriodStatus{
			Usage:      usage,
			Limit:      p.limit,
			Percentage: percentage,
			Remaining:  p.limit - usage,
		}
	}

	return status
}

// BudgetStatus contains comprehensive budget status information.
type BudgetStatus struct {
	Timestamp time.Time
	Periods   map[string]*PeriodStatus // "daily", "weekly", "monthly"
}

// PeriodStatus contains budget status for a specific time period.
type PeriodStatus struct {
	Usage      float64
	Limit      float64
	Percentage float64
	Remaining  float64
}

// GetSpendingAnalysis returns detailed spending analysis and insights.
func (bm *BudgetManager) GetSpendingAnalysis() *SpendingAnalysis {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	analysis := &SpendingAnalysis{
		Timestamp:        time.Now(),
		ProviderBreakdown: make(map[string]float64),
		ModelBreakdown:   make(map[string]float64),
		TaskTypeBreakdown: make(map[string]float64),
		ROI:              make(map[string]*ProviderROI),
		TotalSpent:       0,
		TotalRequests:    len(bm.usage.Transactions),
	}

	// Copy spending breakdowns
	for provider, amount := range bm.usage.ProviderSpending {
		analysis.ProviderBreakdown[provider] = amount
		analysis.TotalSpent += amount
	}

	for model, amount := range bm.usage.ModelSpending {
		analysis.ModelBreakdown[model] = amount
	}

	for taskType, amount := range bm.usage.TaskTypeSpending {
		analysis.TaskTypeBreakdown[taskType] = amount
	}

	// Copy ROI metrics
	for provider, roi := range bm.usage.ProviderROI {
		analysis.ROI[provider] = &ProviderROI{
			TotalSpent:       roi.TotalSpent,
			TotalRequests:    roi.TotalRequests,
			SuccessfulReqs:   roi.SuccessfulReqs,
			AverageQuality:   roi.AverageQuality,
			AverageLatency:   roi.AverageLatency,
			CostPerSuccess:   roi.CostPerSuccess,
			QualityPerDollar: roi.QualityPerDollar,
			LastUpdated:      roi.LastUpdated,
		}
	}

	// Calculate insights
	analysis.Insights = bm.generateSpendingInsights(analysis)

	return analysis
}

// SpendingAnalysis contains comprehensive spending analysis.
type SpendingAnalysis struct {
	Timestamp         time.Time
	TotalSpent        float64
	TotalRequests     int
	ProviderBreakdown map[string]float64
	ModelBreakdown    map[string]float64
	TaskTypeBreakdown map[string]float64
	ROI               map[string]*ProviderROI
	Insights          []string
}

// generateSpendingInsights creates actionable insights from spending data.
func (bm *BudgetManager) generateSpendingInsights(analysis *SpendingAnalysis) []string {
	insights := make([]string, 0)

	// Find most expensive provider
	var maxProvider string
	var maxSpent float64
	for provider, spent := range analysis.ProviderBreakdown {
		if spent > maxSpent {
			maxSpent = spent
			maxProvider = provider
		}
	}

	if maxProvider != "" {
		percentage := (maxSpent / analysis.TotalSpent) * 100
		insights = append(insights,
			fmt.Sprintf("Highest spending: %s accounts for %.1f%% of costs ($%.2f)",
				maxProvider, percentage, maxSpent))
	}

	// Find best ROI provider
	var bestROIProvider string
	var bestROI float64
	for provider, roi := range analysis.ROI {
		if roi.QualityPerDollar > bestROI {
			bestROI = roi.QualityPerDollar
			bestROIProvider = provider
		}
	}

	if bestROIProvider != "" {
		insights = append(insights,
			fmt.Sprintf("Best value: %s provides %.2f quality points per dollar",
				bestROIProvider, bestROI))
	}

	// Check for cost optimization opportunities
	if len(analysis.ProviderBreakdown) > 1 {
		insights = append(insights,
			"Consider consolidating usage to providers with better cost efficiency")
	}

	return insights
}

// String methods for enums
func (bp BudgetPeriod) String() string {
	switch bp {
	case PeriodDaily:
		return "daily"
	case PeriodWeekly:
		return "weekly"
	case PeriodMonthly:
		return "monthly"
	default:
		return "unknown"
	}
}

func (at AlertThreshold) String() string {
	switch at {
	case ThresholdNone:
		return "none"
	case Threshold75:
		return "75%"
	case Threshold90:
		return "90%"
	case Threshold100:
		return "100%"
	default:
		return "unknown"
	}
}

// Persistence methods

// SaveUsage saves usage data to disk.
func (bp *BudgetPersistence) SaveUsage(usage *UsageTracker) error {
	data, err := json.MarshalIndent(usage, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal usage data: %w", err)
	}

	filePath := filepath.Join(bp.dataPath, "budget_usage.json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write usage file: %w", err)
	}

	return nil
}

// LoadUsage loads usage data from disk.
func (bp *BudgetPersistence) LoadUsage() (*UsageTracker, error) {
	filePath := filepath.Join(bp.dataPath, "budget_usage.json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("usage file does not exist")
		}
		return nil, fmt.Errorf("failed to read usage file: %w", err)
	}

	var usage UsageTracker
	if err := json.Unmarshal(data, &usage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal usage data: %w", err)
	}

	// Initialize maps if they're nil (backwards compatibility)
	if usage.Daily == nil {
		usage.Daily = make(map[string]float64)
	}
	if usage.Weekly == nil {
		usage.Weekly = make(map[string]float64)
	}
	if usage.Monthly == nil {
		usage.Monthly = make(map[string]float64)
	}
	if usage.ProviderSpending == nil {
		usage.ProviderSpending = make(map[string]float64)
	}
	if usage.ModelSpending == nil {
		usage.ModelSpending = make(map[string]float64)
	}
	if usage.TaskTypeSpending == nil {
		usage.TaskTypeSpending = make(map[string]float64)
	}
	if usage.ProviderROI == nil {
		usage.ProviderROI = make(map[string]*ProviderROI)
	}

	return &usage, nil
}