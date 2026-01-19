package llm

import (
	"context"
	"log"
	"testing"
	"time"
)

// testLogger creates a logger that discards output during tests.
func testLogger() *log.Logger {
	return log.New(&testLogWriter{}, "test: ", log.LstdFlags)
}

// testLogWriter discards log output during tests.
type testLogWriter struct{}

func (w *testLogWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func TestNewBudgetManager(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultBudgetConfig()
	logger := testLogger()

	bm, err := NewBudgetManager(tempDir, config, logger)
	if err != nil {
		t.Fatalf("Failed to create budget manager: %v", err)
	}

	if bm == nil {
		t.Fatal("Budget manager should not be nil")
	}

	if bm.config.DailyLimit != config.DailyLimit {
		t.Error("Should use provided config")
	}

	if bm.usage == nil {
		t.Error("Should initialize usage tracker")
	}

	if bm.persistence == nil {
		t.Error("Should initialize persistence")
	}
}

func TestDefaultBudgetConfig(t *testing.T) {
	config := DefaultBudgetConfig()

	if config.DailyLimit <= 0 {
		t.Error("Should have positive daily limit")
	}

	if config.WeeklyLimit <= 0 {
		t.Error("Should have positive weekly limit")
	}

	if config.MonthlyLimit <= 0 {
		t.Error("Should have positive monthly limit")
	}

	if len(config.AlertThresholds) == 0 {
		t.Error("Should have alert thresholds")
	}

	// Check that thresholds are reasonable
	for _, threshold := range config.AlertThresholds {
		if threshold <= 0 || threshold > 100 {
			t.Errorf("Alert threshold should be 0-100, got %f", threshold)
		}
	}

	if config.GracePeriod < 0 {
		t.Error("Grace period should be non-negative")
	}
}

func TestRecordUsage(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultBudgetConfig()
	bm, err := NewBudgetManager(tempDir, config, testLogger())
	if err != nil {
		t.Fatalf("Failed to create budget manager: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	transaction := Transaction{
		Provider:   "anthropic",
		Model:      "claude-3-haiku",
		TaskType:   "analysis",
		TokensUsed: 1000,
		Cost:       0.05,
		Success:    true,
		Quality:    8.5,
		Latency:    2000, // 2 seconds
		Timestamp:  now,
	}

	err = bm.RecordUsage(ctx, transaction)
	if err != nil {
		t.Fatalf("Failed to record usage: %v", err)
	}

	// Check that daily usage was updated
	dateKey := now.Format("2006-01-02")
	if bm.usage.Daily[dateKey] != 0.05 {
		t.Errorf("Expected daily usage 0.05, got %f", bm.usage.Daily[dateKey])
	}

	// Check that weekly usage was updated
	weekKey := bm.getWeekKey(now)
	if bm.usage.Weekly[weekKey] != 0.05 {
		t.Errorf("Expected weekly usage 0.05, got %f", bm.usage.Weekly[weekKey])
	}

	// Check that monthly usage was updated
	monthKey := now.Format("2006-01")
	if bm.usage.Monthly[monthKey] != 0.05 {
		t.Errorf("Expected monthly usage 0.05, got %f", bm.usage.Monthly[monthKey])
	}

	// Check provider spending
	if bm.usage.ProviderSpending["anthropic"] != 0.05 {
		t.Errorf("Expected provider spending 0.05, got %f", bm.usage.ProviderSpending["anthropic"])
	}

	// Check model spending
	modelKey := "anthropic_claude-3-haiku"
	if bm.usage.ModelSpending[modelKey] != 0.05 {
		t.Errorf("Expected model spending 0.05, got %f", bm.usage.ModelSpending[modelKey])
	}

	// Check task type spending
	if bm.usage.TaskTypeSpending["analysis"] != 0.05 {
		t.Errorf("Expected task type spending 0.05, got %f", bm.usage.TaskTypeSpending["analysis"])
	}

	// Check ROI metrics
	roi, exists := bm.usage.ProviderROI["anthropic"]
	if !exists {
		t.Fatal("Should have ROI metrics for anthropic")
	}

	if roi.TotalSpent != 0.05 {
		t.Errorf("Expected total spent 0.05, got %f", roi.TotalSpent)
	}

	if roi.TotalRequests != 1 {
		t.Errorf("Expected 1 request, got %d", roi.TotalRequests)
	}

	if roi.SuccessfulReqs != 1 {
		t.Errorf("Expected 1 successful request, got %d", roi.SuccessfulReqs)
	}

	if roi.AverageQuality != 8.5 {
		t.Errorf("Expected average quality 8.5, got %f", roi.AverageQuality)
	}

	if roi.AverageLatency != 2000 {
		t.Errorf("Expected average latency 2000ms, got %f", roi.AverageLatency)
	}

	// Check transaction log
	if len(bm.usage.Transactions) != 1 {
		t.Errorf("Expected 1 transaction in log, got %d", len(bm.usage.Transactions))
	}

	if bm.usage.Transactions[0].ID == "" {
		t.Error("Transaction should have an ID")
	}
}

func TestMultipleUsageRecords(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultBudgetConfig()
	bm, err := NewBudgetManager(tempDir, config, testLogger())
	if err != nil {
		t.Fatalf("Failed to create budget manager: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	transactions := []Transaction{
		{
			Provider:  "anthropic",
			Model:     "claude-3-haiku",
			TaskType:  "analysis",
			Cost:      0.05,
			Success:   true,
			Quality:   8.5,
			Timestamp: now,
		},
		{
			Provider:  "anthropic",
			Model:     "claude-3-haiku",
			TaskType:  "analysis",
			Cost:      0.06,
			Success:   true,
			Quality:   7.5,
			Timestamp: now.Add(time.Minute),
		},
		{
			Provider:  "anthropic",
			Model:     "claude-3-haiku",
			TaskType:  "analysis",
			Cost:      0.04,
			Success:   false,
			Quality:   6.0,
			Timestamp: now.Add(2 * time.Minute),
		},
	}

	for _, tx := range transactions {
		err = bm.RecordUsage(ctx, tx)
		if err != nil {
			t.Fatalf("Failed to record usage: %v", err)
		}
	}

	// Check cumulative costs
	expectedTotal := 0.05 + 0.06 + 0.04
	dateKey := now.Format("2006-01-02")
	if bm.usage.Daily[dateKey] != expectedTotal {
		t.Errorf("Expected daily usage %f, got %f", expectedTotal, bm.usage.Daily[dateKey])
	}

	// Check ROI metrics calculations
	roi := bm.usage.ProviderROI["anthropic"]
	if roi.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", roi.TotalRequests)
	}

	if roi.SuccessfulReqs != 2 {
		t.Errorf("Expected 2 successful requests, got %d", roi.SuccessfulReqs)
	}

	// Check average quality calculation (should include all ratings)
	expectedAvgQuality := (8.5 + 7.5 + 6.0) / 3.0
	if roi.AverageQuality < expectedAvgQuality-0.01 || roi.AverageQuality > expectedAvgQuality+0.01 {
		t.Errorf("Expected average quality ~%.2f, got %.2f", expectedAvgQuality, roi.AverageQuality)
	}

	// Check cost per success
	expectedCostPerSuccess := expectedTotal / 2.0 // 2 successful requests
	if roi.CostPerSuccess < expectedCostPerSuccess-0.01 || roi.CostPerSuccess > expectedCostPerSuccess+0.01 {
		t.Errorf("Expected cost per success ~%.3f, got %.3f", expectedCostPerSuccess, roi.CostPerSuccess)
	}
}

func TestCanAfford(t *testing.T) {
	tempDir := t.TempDir()
	config := BudgetConfig{
		DailyLimit:   1.0,
		WeeklyLimit:  5.0,
		MonthlyLimit: 20.0,
		AutoStop:     true,
		GracePeriod:  0.10,
	}
	bm, err := NewBudgetManager(tempDir, config, testLogger())
	if err != nil {
		t.Fatalf("Failed to create budget manager: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	// Test when budget is available
	check, err := bm.CanAfford(0.50)
	if err != nil {
		t.Fatalf("CanAfford failed: %v", err)
	}

	if !check.Affordable {
		t.Error("Should be affordable with empty budget")
	}

	if len(check.Warnings) != 0 {
		t.Errorf("Should have no warnings with plenty of budget, got %d", len(check.Warnings))
	}

	// Record some usage
	tx := Transaction{
		Provider:  "anthropic",
		Model:     "claude-3-haiku",
		Cost:      0.80, // 80% of daily budget
		Success:   true,
		Timestamp: now,
	}
	err = bm.RecordUsage(ctx, tx)
	if err != nil {
		t.Fatalf("Failed to record usage: %v", err)
	}

	// Test when close to budget limit
	check, err = bm.CanAfford(0.15)
	if err != nil {
		t.Fatalf("CanAfford failed: %v", err)
	}

	if !check.Affordable {
		t.Error("Should still be affordable within daily limit")
	}

	if len(check.Warnings) == 0 {
		t.Error("Should have warnings when close to budget limit")
	}

	// Test when exceeding budget
	check, err = bm.CanAfford(0.30) // Would total 1.10, exceeding 1.0 daily limit
	if err != nil {
		t.Fatalf("CanAfford failed: %v", err)
	}

	if check.Affordable {
		t.Error("Should not be affordable when exceeding daily limit")
	}

	// Test with grace period
	check, err = bm.CanAfford(0.05) // Within grace period of 0.10
	if err != nil {
		t.Fatalf("CanAfford failed: %v", err)
	}

	if !check.Affordable {
		t.Error("Should be affordable within grace period")
	}
}

func TestBudgetAlerts(t *testing.T) {
	tempDir := t.TempDir()

	alertsTriggered := make([]AlertInfo, 0)
	config := BudgetConfig{
		DailyLimit:      1.0,
		WeeklyLimit:     5.0,
		MonthlyLimit:    20.0,
		AlertThresholds: []float64{50.0, 75.0, 100.0},
		AlertCallback: func(alert AlertInfo) {
			alertsTriggered = append(alertsTriggered, alert)
		},
		TrackingEnabled: true,
	}

	bm, err := NewBudgetManager(tempDir, config, testLogger())
	if err != nil {
		t.Fatalf("Failed to create budget manager: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	// Record usage that triggers 50% threshold
	tx1 := Transaction{
		Provider:  "anthropic",
		Model:     "claude-3-haiku",
		Cost:      0.60, // 60% of daily budget
		Success:   true,
		Timestamp: now,
	}
	err = bm.RecordUsage(ctx, tx1)
	if err != nil {
		t.Fatalf("Failed to record usage: %v", err)
	}

	// Should trigger 50% alert
	if len(alertsTriggered) != 1 {
		t.Errorf("Expected 1 alert for 50%% threshold, got %d", len(alertsTriggered))
	} else {
		alert := alertsTriggered[0]
		if alert.Threshold != 50.0 {
			t.Errorf("Expected 50%% threshold alert, got %.0f%%", alert.Threshold)
		}
		if alert.Period != PeriodDaily {
			t.Errorf("Expected daily period alert, got %s", alert.Period.String())
		}
	}

	// Record more usage to trigger 75% threshold
	tx2 := Transaction{
		Provider:  "anthropic",
		Model:     "claude-3-haiku",
		Cost:      0.20, // Total now 80% of daily budget
		Success:   true,
		Timestamp: now.Add(time.Minute),
	}
	err = bm.RecordUsage(ctx, tx2)
	if err != nil {
		t.Fatalf("Failed to record usage: %v", err)
	}

	// Should trigger 75% alert
	if len(alertsTriggered) != 2 {
		t.Errorf("Expected 2 alerts total, got %d", len(alertsTriggered))
	}

	// Record usage to exceed budget (100% threshold)
	tx3 := Transaction{
		Provider:  "anthropic",
		Model:     "claude-3-haiku",
		Cost:      0.30, // Total now 110% of daily budget
		Success:   true,
		Timestamp: now.Add(2 * time.Minute),
	}
	err = bm.RecordUsage(ctx, tx3)
	if err != nil {
		t.Fatalf("Failed to record usage: %v", err)
	}

	// Should trigger 100% alert
	if len(alertsTriggered) != 3 {
		t.Errorf("Expected 3 alerts total, got %d", len(alertsTriggered))
	} else {
		alert := alertsTriggered[2]
		if alert.Threshold != 100.0 {
			t.Errorf("Expected 100%% threshold alert, got %.0f%%", alert.Threshold)
		}
		if alert.OverageAmount <= 0 {
			t.Error("Should have positive overage amount for 100% alert")
		}
	}
}

func TestGetBudgetStatus(t *testing.T) {
	tempDir := t.TempDir()
	config := BudgetConfig{
		DailyLimit:   1.0,
		WeeklyLimit:  5.0,
		MonthlyLimit: 20.0,
	}
	bm, err := NewBudgetManager(tempDir, config, testLogger())
	if err != nil {
		t.Fatalf("Failed to create budget manager: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	// Record some usage
	tx := Transaction{
		Provider:  "anthropic",
		Model:     "claude-3-haiku",
		Cost:      0.30, // 30% of daily budget
		Success:   true,
		Timestamp: now,
	}
	err = bm.RecordUsage(ctx, tx)
	if err != nil {
		t.Fatalf("Failed to record usage: %v", err)
	}

	status := bm.GetBudgetStatus()
	if status == nil {
		t.Fatal("Budget status should not be nil")
	}

	// Check daily status
	dailyStatus, exists := status.Periods["daily"]
	if !exists {
		t.Fatal("Should have daily status")
	}

	if dailyStatus.Usage != 0.30 {
		t.Errorf("Expected daily usage 0.30, got %f", dailyStatus.Usage)
	}

	if dailyStatus.Limit != 1.0 {
		t.Errorf("Expected daily limit 1.0, got %f", dailyStatus.Limit)
	}

	expectedPercentage := 30.0
	if dailyStatus.Percentage < expectedPercentage-1 || dailyStatus.Percentage > expectedPercentage+1 {
		t.Errorf("Expected daily percentage ~%.0f%%, got %.1f%%", expectedPercentage, dailyStatus.Percentage)
	}

	expectedRemaining := 0.70
	if dailyStatus.Remaining < expectedRemaining-0.01 || dailyStatus.Remaining > expectedRemaining+0.01 {
		t.Errorf("Expected daily remaining ~%.2f, got %.2f", expectedRemaining, dailyStatus.Remaining)
	}
}

func TestGetSpendingAnalysis(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultBudgetConfig()
	bm, err := NewBudgetManager(tempDir, config, testLogger())
	if err != nil {
		t.Fatalf("Failed to create budget manager: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	// Record usage from multiple providers
	transactions := []Transaction{
		{
			Provider:  "anthropic",
			Model:     "claude-3-haiku",
			TaskType:  "analysis",
			Cost:      0.05,
			Success:   true,
			Quality:   8.0,
			Timestamp: now,
		},
		{
			Provider:  "anthropic",
			Model:     "claude-3-sonnet",
			TaskType:  "creative",
			Cost:      0.15,
			Success:   true,
			Quality:   9.0,
			Timestamp: now,
		},
		{
			Provider:  "openai",
			Model:     "gpt-3.5-turbo",
			TaskType:  "analysis",
			Cost:      0.03,
			Success:   true,
			Quality:   7.5,
			Timestamp: now,
		},
	}

	for _, tx := range transactions {
		err = bm.RecordUsage(ctx, tx)
		if err != nil {
			t.Fatalf("Failed to record usage: %v", err)
		}
	}

	analysis := bm.GetSpendingAnalysis()
	if analysis == nil {
		t.Fatal("Spending analysis should not be nil")
	}

	expectedTotal := 0.05 + 0.15 + 0.03
	if analysis.TotalSpent != expectedTotal {
		t.Errorf("Expected total spent %f, got %f", expectedTotal, analysis.TotalSpent)
	}

	if analysis.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", analysis.TotalRequests)
	}

	// Check provider breakdown
	if analysis.ProviderBreakdown["anthropic"] != 0.20 {
		t.Errorf("Expected anthropic spending 0.20, got %f", analysis.ProviderBreakdown["anthropic"])
	}

	if analysis.ProviderBreakdown["openai"] != 0.03 {
		t.Errorf("Expected openai spending 0.03, got %f", analysis.ProviderBreakdown["openai"])
	}

	// Check model breakdown
	if analysis.ModelBreakdown["anthropic_claude-3-haiku"] != 0.05 {
		t.Errorf("Expected claude-3-haiku spending 0.05, got %f", analysis.ModelBreakdown["anthropic_claude-3-haiku"])
	}

	// Check task type breakdown
	expectedAnalysisCost := 0.05 + 0.03 // Both anthropic and openai analysis tasks
	if analysis.TaskTypeBreakdown["analysis"] != expectedAnalysisCost {
		t.Errorf("Expected analysis spending %f, got %f", expectedAnalysisCost, analysis.TaskTypeBreakdown["analysis"])
	}

	// Check ROI data
	anthropicROI, exists := analysis.ROI["anthropic"]
	if !exists {
		t.Fatal("Should have ROI data for anthropic")
	}

	if anthropicROI.TotalSpent != 0.20 {
		t.Errorf("Expected anthropic total spent 0.20, got %f", anthropicROI.TotalSpent)
	}

	if anthropicROI.QualityPerDollar <= 0 {
		t.Error("Should have positive quality per dollar for anthropic")
	}

	// Check insights
	if len(analysis.Insights) == 0 {
		t.Error("Should have some insights")
	}
}

func TestBudgetPersistence(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultBudgetConfig()

	// Create first budget manager and record some usage
	bm1, err := NewBudgetManager(tempDir, config, testLogger())
	if err != nil {
		t.Fatalf("Failed to create budget manager: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	tx := Transaction{
		Provider:  "anthropic",
		Model:     "claude-3-haiku",
		TaskType:  "analysis",
		Cost:      0.05,
		Success:   true,
		Quality:   8.5,
		Timestamp: now,
	}

	err = bm1.RecordUsage(ctx, tx)
	if err != nil {
		t.Fatalf("Failed to record usage: %v", err)
	}

	// Create second budget manager from same directory
	bm2, err := NewBudgetManager(tempDir, config, testLogger())
	if err != nil {
		t.Fatalf("Failed to create second budget manager: %v", err)
	}

	// Should have loaded the previous data
	dateKey := now.Format("2006-01-02")
	if bm2.usage.Daily[dateKey] != 0.05 {
		t.Errorf("Expected loaded daily usage 0.05, got %f", bm2.usage.Daily[dateKey])
	}

	if bm2.usage.ProviderSpending["anthropic"] != 0.05 {
		t.Errorf("Expected loaded provider spending 0.05, got %f", bm2.usage.ProviderSpending["anthropic"])
	}

	if len(bm2.usage.Transactions) != 1 {
		t.Errorf("Expected 1 loaded transaction, got %d", len(bm2.usage.Transactions))
	}

	// Check that ROI data was loaded
	roi, exists := bm2.usage.ProviderROI["anthropic"]
	if !exists {
		t.Fatal("Should have loaded ROI data for anthropic")
	}

	if roi.AverageQuality != 8.5 {
		t.Errorf("Expected loaded average quality 8.5, got %f", roi.AverageQuality)
	}
}

func TestWeekKeyGeneration(t *testing.T) {
	tempDir := t.TempDir()
	bm, err := NewBudgetManager(tempDir, DefaultBudgetConfig(), testLogger())
	if err != nil {
		t.Fatalf("Failed to create budget manager: %v", err)
	}

	// Test that week keys are consistent
	date1 := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC) // Monday
	date2 := time.Date(2024, 1, 16, 12, 0, 0, 0, time.UTC) // Tuesday
	date3 := time.Date(2024, 1, 22, 12, 0, 0, 0, time.UTC) // Next Monday

	week1 := bm.getWeekKey(date1)
	week2 := bm.getWeekKey(date2)
	week3 := bm.getWeekKey(date3)

	if week1 != week2 {
		t.Errorf("Monday and Tuesday should be in same week: %s vs %s", week1, week2)
	}

	if week1 == week3 {
		t.Errorf("Different weeks should have different keys: %s vs %s", week1, week3)
	}

	// Week keys should be in expected format
	if week1 == "" {
		t.Error("Week key should not be empty")
	}
}

func TestBudgetPeriodEnum(t *testing.T) {
	periods := []BudgetPeriod{PeriodDaily, PeriodWeekly, PeriodMonthly}
	for _, period := range periods {
		if period.String() == "" || period.String() == "unknown" {
			t.Errorf("BudgetPeriod %d should have valid string representation", int(period))
		}
	}
}

func TestTransactionIDGeneration(t *testing.T) {
	tempDir := t.TempDir()
	bm, err := NewBudgetManager(tempDir, DefaultBudgetConfig(), testLogger())
	if err != nil {
		t.Fatalf("Failed to create budget manager: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	// Test transaction without ID (should be generated)
	tx1 := Transaction{
		Provider:  "anthropic",
		Model:     "claude-3-haiku",
		Cost:      0.05,
		Success:   true,
		Timestamp: now,
	}

	err = bm.RecordUsage(ctx, tx1)
	if err != nil {
		t.Fatalf("Failed to record usage: %v", err)
	}

	if len(bm.usage.Transactions) != 1 {
		t.Fatalf("Expected 1 transaction, got %d", len(bm.usage.Transactions))
	}

	if bm.usage.Transactions[0].ID == "" {
		t.Error("Transaction ID should be generated")
	}

	// Test transaction with explicit ID
	tx2 := Transaction{
		ID:        "custom_id_123",
		Provider:  "openai",
		Model:     "gpt-3.5-turbo",
		Cost:      0.03,
		Success:   true,
		Timestamp: now,
	}

	err = bm.RecordUsage(ctx, tx2)
	if err != nil {
		t.Fatalf("Failed to record usage: %v", err)
	}

	if len(bm.usage.Transactions) != 2 {
		t.Fatalf("Expected 2 transactions, got %d", len(bm.usage.Transactions))
	}

	// Find the second transaction
	var foundTx *Transaction
	for _, tx := range bm.usage.Transactions {
		if tx.Provider == "openai" {
			foundTx = &tx
			break
		}
	}

	if foundTx == nil {
		t.Fatal("Should find openai transaction")
	}

	if foundTx.ID != "custom_id_123" {
		t.Errorf("Expected custom ID 'custom_id_123', got '%s'", foundTx.ID)
	}
}

// Benchmark tests

func BenchmarkRecordUsage(b *testing.B) {
	tempDir := b.TempDir()
	bm, err := NewBudgetManager(tempDir, DefaultBudgetConfig(), testLogger())
	if err != nil {
		b.Fatalf("Failed to create budget manager: %v", err)
	}

	ctx := context.Background()
	transaction := Transaction{
		Provider:   "anthropic",
		Model:      "claude-3-haiku",
		TaskType:   "analysis",
		TokensUsed: 1000,
		Cost:       0.05,
		Success:    true,
		Quality:    8.0,
		Latency:    2000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transaction.Timestamp = time.Now()
		err = bm.RecordUsage(ctx, transaction)
		if err != nil {
			b.Fatalf("Failed to record usage: %v", err)
		}
	}
}

func BenchmarkCanAfford(b *testing.B) {
	tempDir := b.TempDir()
	bm, err := NewBudgetManager(tempDir, DefaultBudgetConfig(), testLogger())
	if err != nil {
		b.Fatalf("Failed to create budget manager: %v", err)
	}

	// Record some baseline usage
	ctx := context.Background()
	tx := Transaction{
		Provider:  "anthropic",
		Model:     "claude-3-haiku",
		Cost:      1.0,
		Success:   true,
		Timestamp: time.Now(),
	}
	err = bm.RecordUsage(ctx, tx)
	if err != nil {
		b.Fatalf("Failed to record baseline usage: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := bm.CanAfford(0.50)
		if err != nil {
			b.Fatalf("CanAfford failed: %v", err)
		}
	}
}

func BenchmarkGetSpendingAnalysis(b *testing.B) {
	tempDir := b.TempDir()
	bm, err := NewBudgetManager(tempDir, DefaultBudgetConfig(), testLogger())
	if err != nil {
		b.Fatalf("Failed to create budget manager: %v", err)
	}

	// Record some transactions
	ctx := context.Background()
	providers := []string{"anthropic", "openai"}
	models := []string{"claude-3-haiku", "gpt-3.5-turbo"}

	for i := 0; i < 100; i++ {
		tx := Transaction{
			Provider:  providers[i%len(providers)],
			Model:     models[i%len(models)],
			TaskType:  "analysis",
			Cost:      0.01,
			Success:   true,
			Timestamp: time.Now(),
		}
		err = bm.RecordUsage(ctx, tx)
		if err != nil {
			b.Fatalf("Failed to record usage: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bm.GetSpendingAnalysis()
	}
}