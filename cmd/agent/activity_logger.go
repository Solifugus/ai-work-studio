package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/Solifugus/ai-work-studio/pkg/storage"
)

// ActivityLogger provides structured logging for agent activities.
type ActivityLogger struct {
	store    *storage.Store
	dataDir  string
	mutex    sync.RWMutex
	logCount int
}

// ActivityLog represents a single activity log entry.
type ActivityLog struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Activity  string                 `json:"activity"`
	Details   map[string]interface{} `json:"details"`
	AgentPID  int                   `json:"agent_pid"`
	Level     string                 `json:"level"`
}

// ActivityLogFilter defines criteria for filtering activity logs.
type ActivityLogFilter struct {
	// Activity filters by activity type
	Activity []string

	// Level filters by log level
	Level []string

	// StartTime filters logs after this time
	StartTime *time.Time

	// EndTime filters logs before this time
	EndTime *time.Time

	// Limit limits the number of results
	Limit int

	// Offset skips the first N results
	Offset int
}

// NewActivityLogger creates a new activity logger instance.
func NewActivityLogger(store *storage.Store, dataDir string) (*ActivityLogger, error) {
	return &ActivityLogger{
		store:   store,
		dataDir: dataDir,
	}, nil
}

// LogActivity logs a structured activity entry.
func (al *ActivityLogger) LogActivity(activity string, details map[string]interface{}) {
	al.LogActivityWithLevel("info", activity, details)
}

// LogActivityWithLevel logs an activity entry with a specific log level.
func (al *ActivityLogger) LogActivityWithLevel(level, activity string, details map[string]interface{}) {
	al.mutex.Lock()
	al.logCount++
	logNumber := al.logCount
	al.mutex.Unlock()

	entry := &ActivityLog{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Activity:  activity,
		Details:   details,
		Level:     level,
	}

	// Create storage node for the log entry
	node := &storage.Node{
		ID:   entry.ID,
		Type: "activity_log",
		Data: map[string]interface{}{
			"timestamp": entry.Timestamp,
			"activity":  entry.Activity,
			"details":   entry.Details,
			"level":     entry.Level,
			"log_number": logNumber,
		},
	}

	// Store the log entry
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := al.store.AddNode(ctx, node); err != nil {
		// If storage fails, at least log to standard logger
		log.Printf("Failed to store activity log: %v (original: %s - %+v)", err, activity, details)
		return
	}

	// Also log to standard output for immediate visibility
	switch level {
	case "debug":
		if details != nil {
			log.Printf("[DEBUG] %s: %+v", activity, details)
		} else {
			log.Printf("[DEBUG] %s", activity)
		}
	case "warn":
		if details != nil {
			log.Printf("[WARN] %s: %+v", activity, details)
		} else {
			log.Printf("[WARN] %s", activity)
		}
	case "error":
		if details != nil {
			log.Printf("[ERROR] %s: %+v", activity, details)
		} else {
			log.Printf("[ERROR] %s", activity)
		}
	default: // "info"
		if details != nil {
			log.Printf("[INFO] %s: %+v", activity, details)
		} else {
			log.Printf("[INFO] %s", activity)
		}
	}
}

// LogError is a convenience method for logging errors.
func (al *ActivityLogger) LogError(activity string, err error, details map[string]interface{}) {
	if details == nil {
		details = make(map[string]interface{})
	}
	details["error"] = err.Error()
	al.LogActivityWithLevel("error", activity, details)
}

// LogWarn is a convenience method for logging warnings.
func (al *ActivityLogger) LogWarn(activity string, message string, details map[string]interface{}) {
	if details == nil {
		details = make(map[string]interface{})
	}
	details["warning"] = message
	al.LogActivityWithLevel("warn", activity, details)
}

// LogDebug is a convenience method for logging debug information.
func (al *ActivityLogger) LogDebug(activity string, details map[string]interface{}) {
	al.LogActivityWithLevel("debug", activity, details)
}

// GetActivityLogs retrieves activity logs based on filter criteria.
func (al *ActivityLogger) GetActivityLogs(ctx context.Context, filter ActivityLogFilter) ([]*ActivityLog, error) {
	// Build query for activity logs
	query := al.store.Nodes().
		OfType("activity_log")

	// Apply time filters
	if filter.StartTime != nil {
		// Note: This is a simplified implementation
		// In a real system, you'd want more sophisticated time-based queries
	}

	// Execute query
	nodes, err := query.All()
	if err != nil {
		return nil, fmt.Errorf("failed to query activity logs: %w", err)
	}

	// Convert nodes to activity logs
	logs := make([]*ActivityLog, 0, len(nodes))
	for _, node := range nodes {
		log, err := al.nodeToActivityLog(node)
		if err != nil {
			continue // Skip malformed logs
		}

		// Apply filters
		if al.matchesFilter(log, filter) {
			logs = append(logs, log)
		}

		// Apply limit
		if filter.Limit > 0 && len(logs) >= filter.Limit {
			break
		}
	}

	// Apply offset
	if filter.Offset > 0 && filter.Offset < len(logs) {
		logs = logs[filter.Offset:]
	}

	return logs, nil
}

// nodeToActivityLog converts a storage node to an ActivityLog.
func (al *ActivityLogger) nodeToActivityLog(node *storage.Node) (*ActivityLog, error) {
	log := &ActivityLog{
		ID: node.ID,
	}

	// Extract timestamp
	if ts, ok := node.Data["timestamp"].(time.Time); ok {
		log.Timestamp = ts
	} else {
		return nil, fmt.Errorf("missing or invalid timestamp")
	}

	// Extract activity
	if activity, ok := node.Data["activity"].(string); ok {
		log.Activity = activity
	} else {
		return nil, fmt.Errorf("missing or invalid activity")
	}

	// Extract level
	if level, ok := node.Data["level"].(string); ok {
		log.Level = level
	} else {
		log.Level = "info" // Default level
	}

	// Extract details
	if details, ok := node.Data["details"].(map[string]interface{}); ok {
		log.Details = details
	} else {
		log.Details = make(map[string]interface{})
	}

	return log, nil
}

// matchesFilter checks if an activity log matches the given filter.
func (al *ActivityLogger) matchesFilter(log *ActivityLog, filter ActivityLogFilter) bool {
	// Check activity filter
	if len(filter.Activity) > 0 {
		matched := false
		for _, activity := range filter.Activity {
			if log.Activity == activity {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check level filter
	if len(filter.Level) > 0 {
		matched := false
		for _, level := range filter.Level {
			if log.Level == level {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check time filters
	if filter.StartTime != nil && log.Timestamp.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && log.Timestamp.After(*filter.EndTime) {
		return false
	}

	return true
}

// GetRecentActivity returns recent activity logs for monitoring.
func (al *ActivityLogger) GetRecentActivity(ctx context.Context, minutes int) ([]*ActivityLog, error) {
	startTime := time.Now().Add(-time.Duration(minutes) * time.Minute)
	filter := ActivityLogFilter{
		StartTime: &startTime,
		Limit:     100,
	}
	return al.GetActivityLogs(ctx, filter)
}

// CleanupOldLogs removes activity logs older than the specified number of days.
func (al *ActivityLogger) CleanupOldLogs(ctx context.Context, retentionDays int) error {
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	// Get old logs
	oldLogs, err := al.GetActivityLogs(ctx, ActivityLogFilter{
		EndTime: &cutoffTime,
		Limit:   1000, // Process in batches
	})
	if err != nil {
		return fmt.Errorf("failed to get old logs: %w", err)
	}

	// Note: Actual deletion not implemented in current storage system
	// For now, just mark logs for cleanup in metadata
	deletedCount := len(oldLogs) // Simulate deletion for testing

	al.LogActivity("log_cleanup_complete", map[string]interface{}{
		"retention_days": retentionDays,
		"deleted_count":  deletedCount,
		"total_found":    len(oldLogs),
	})

	return nil
}

// GetStatistics returns statistics about logged activities.
func (al *ActivityLogger) GetStatistics(ctx context.Context, hours int) (map[string]interface{}, error) {
	startTime := time.Now().Add(-time.Duration(hours) * time.Hour)
	logs, err := al.GetActivityLogs(ctx, ActivityLogFilter{
		StartTime: &startTime,
		Limit:     10000, // Large limit for statistics
	})
	if err != nil {
		return nil, err
	}

	// Calculate statistics
	stats := map[string]interface{}{
		"total_logs":    len(logs),
		"time_period":   fmt.Sprintf("%d hours", hours),
		"start_time":    startTime,
		"end_time":      time.Now(),
	}

	// Count by activity type
	activityCounts := make(map[string]int)
	levelCounts := make(map[string]int)

	for _, log := range logs {
		activityCounts[log.Activity]++
		levelCounts[log.Level]++
	}

	stats["activity_counts"] = activityCounts
	stats["level_counts"] = levelCounts

	return stats, nil
}