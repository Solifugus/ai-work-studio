// Package utils provides utility functions and services for the AI Work Studio.
package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogLevel represents the severity level of a log entry.
type LogLevel string

const (
	// LogLevelDebug for detailed debugging information
	LogLevelDebug LogLevel = "DEBUG"
	// LogLevelInfo for general information messages
	LogLevelInfo LogLevel = "INFO"
	// LogLevelWarning for warning messages that don't stop execution
	LogLevelWarning LogLevel = "WARNING"
	// LogLevelError for error messages
	LogLevelError LogLevel = "ERROR"
)

// LogDestination represents where logs should be written.
type LogDestination string

const (
	// LogDestinationFile writes logs to a file
	LogDestinationFile LogDestination = "file"
	// LogDestinationConsole writes logs to stdout/stderr
	LogDestinationConsole LogDestination = "console"
	// LogDestinationBoth writes logs to both file and console
	LogDestinationBoth LogDestination = "both"
)

// LogConfig holds configuration for the logging system.
type LogConfig struct {
	// Level defines the minimum log level to output
	Level LogLevel `json:"level"`
	// Destination defines where logs should be written
	Destination LogDestination `json:"destination"`
	// FilePath is the path to the log file (when destination includes file)
	FilePath string `json:"file_path"`
	// FileMaxSize is the maximum size in MB before rotation
	FileMaxSize int `json:"file_max_size"`
	// FileMaxBackups is the maximum number of old log files to retain
	FileMaxBackups int `json:"file_max_backups"`
	// FileMaxAge is the maximum age in days to retain old log files
	FileMaxAge int `json:"file_max_age"`
	// Component identifies the component creating logs
	Component string `json:"component"`
}

// LogContext holds contextual information for structured logging.
type LogContext struct {
	GoalID      string `json:"goal_id,omitempty"`
	ObjectiveID string `json:"objective_id,omitempty"`
	MethodID    string `json:"method_id,omitempty"`
	UserID      string `json:"user_id,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	Component   string `json:"component,omitempty"`
}

// Logger provides structured logging interface.
type Logger interface {
	// Debug logs debug-level messages
	Debug(ctx LogContext, message string, fields ...map[string]interface{})
	// Info logs info-level messages
	Info(ctx LogContext, message string, fields ...map[string]interface{})
	// Warning logs warning-level messages
	Warning(ctx LogContext, message string, fields ...map[string]interface{})
	// Error logs error-level messages
	Error(ctx LogContext, message string, fields ...map[string]interface{})
	// WithContext returns a new logger instance with updated context
	WithContext(ctx LogContext) Logger
	// Close closes any resources used by the logger
	Close() error
}

// ConcreteLogger implements the Logger interface using logrus.
type ConcreteLogger struct {
	logger   *logrus.Logger
	config   LogConfig
	context  LogContext
	fileHook io.Closer
}

// NewLogger creates a new logger instance with the given configuration.
func NewLogger(config LogConfig) (*ConcreteLogger, error) {
	logger := logrus.New()

	// Set log level
	level, err := parseLogLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %s: %w", config.Level, err)
	}
	logger.SetLevel(level)

	// Set JSON format for structured logging
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	// Configure output destinations
	if err := configureOutput(logger, config); err != nil {
		return nil, fmt.Errorf("failed to configure output: %w", err)
	}

	return &ConcreteLogger{
		logger: logger,
		config: config,
		context: LogContext{
			Component: config.Component,
		},
	}, nil
}

// Debug logs debug-level messages.
func (l *ConcreteLogger) Debug(ctx LogContext, message string, fields ...map[string]interface{}) {
	l.logWithContext(logrus.DebugLevel, ctx, message, fields...)
}

// Info logs info-level messages.
func (l *ConcreteLogger) Info(ctx LogContext, message string, fields ...map[string]interface{}) {
	l.logWithContext(logrus.InfoLevel, ctx, message, fields...)
}

// Warning logs warning-level messages.
func (l *ConcreteLogger) Warning(ctx LogContext, message string, fields ...map[string]interface{}) {
	l.logWithContext(logrus.WarnLevel, ctx, message, fields...)
}

// Error logs error-level messages.
func (l *ConcreteLogger) Error(ctx LogContext, message string, fields ...map[string]interface{}) {
	l.logWithContext(logrus.ErrorLevel, ctx, message, fields...)
}

// WithContext returns a new logger instance with updated context.
func (l *ConcreteLogger) WithContext(ctx LogContext) Logger {
	newLogger := &ConcreteLogger{
		logger:   l.logger,
		config:   l.config,
		fileHook: l.fileHook,
		context:  mergeContext(l.context, ctx),
	}
	return newLogger
}

// Close closes any resources used by the logger.
func (l *ConcreteLogger) Close() error {
	if l.fileHook != nil {
		return l.fileHook.Close()
	}
	return nil
}

// logWithContext logs a message with the provided context and additional fields.
func (l *ConcreteLogger) logWithContext(level logrus.Level, ctx LogContext, message string, fields ...map[string]interface{}) {
	entry := l.logger.WithFields(logrus.Fields{
		"component": l.mergedContext(ctx).Component,
	})

	// Add context fields
	contextFields := l.contextToFields(l.mergedContext(ctx))
	for key, value := range contextFields {
		if value != "" {
			entry = entry.WithField(key, value)
		}
	}

	// Add additional fields
	for _, fieldMap := range fields {
		for key, value := range fieldMap {
			entry = entry.WithField(key, value)
		}
	}

	entry.Log(level, message)
}

// mergedContext merges the logger's base context with the provided context.
func (l *ConcreteLogger) mergedContext(ctx LogContext) LogContext {
	return mergeContext(l.context, ctx)
}

// contextToFields converts LogContext to logrus fields.
func (l *ConcreteLogger) contextToFields(ctx LogContext) logrus.Fields {
	fields := logrus.Fields{}

	if ctx.GoalID != "" {
		fields["goal_id"] = ctx.GoalID
	}
	if ctx.ObjectiveID != "" {
		fields["objective_id"] = ctx.ObjectiveID
	}
	if ctx.MethodID != "" {
		fields["method_id"] = ctx.MethodID
	}
	if ctx.UserID != "" {
		fields["user_id"] = ctx.UserID
	}
	if ctx.SessionID != "" {
		fields["session_id"] = ctx.SessionID
	}

	return fields
}

// parseLogLevel converts LogLevel string to logrus Level.
func parseLogLevel(level LogLevel) (logrus.Level, error) {
	switch strings.ToUpper(string(level)) {
	case "DEBUG":
		return logrus.DebugLevel, nil
	case "INFO":
		return logrus.InfoLevel, nil
	case "WARNING", "WARN":
		return logrus.WarnLevel, nil
	case "ERROR":
		return logrus.ErrorLevel, nil
	default:
		return logrus.InfoLevel, fmt.Errorf("unknown log level: %s", level)
	}
}

// configureOutput sets up log output destinations based on configuration.
func configureOutput(logger *logrus.Logger, config LogConfig) error {
	switch config.Destination {
	case LogDestinationConsole:
		logger.SetOutput(os.Stdout)
	case LogDestinationFile:
		if config.FilePath == "" {
			return fmt.Errorf("file_path is required when destination is file")
		}

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(config.FilePath), 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		// Set up log rotation
		rotatingFile := &lumberjack.Logger{
			Filename:   config.FilePath,
			MaxSize:    config.FileMaxSize,
			MaxBackups: config.FileMaxBackups,
			MaxAge:     config.FileMaxAge,
			Compress:   true,
		}

		logger.SetOutput(rotatingFile)
	case LogDestinationBoth:
		if config.FilePath == "" {
			return fmt.Errorf("file_path is required when destination includes file")
		}

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(config.FilePath), 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		// Set up log rotation
		rotatingFile := &lumberjack.Logger{
			Filename:   config.FilePath,
			MaxSize:    config.FileMaxSize,
			MaxBackups: config.FileMaxBackups,
			MaxAge:     config.FileMaxAge,
			Compress:   true,
		}

		// Create multi-writer for both console and file
		multiWriter := io.MultiWriter(os.Stdout, rotatingFile)
		logger.SetOutput(multiWriter)
	default:
		return fmt.Errorf("unknown log destination: %s", config.Destination)
	}

	return nil
}

// mergeContext merges two LogContext structs, with the second taking precedence.
func mergeContext(base, override LogContext) LogContext {
	result := base

	if override.GoalID != "" {
		result.GoalID = override.GoalID
	}
	if override.ObjectiveID != "" {
		result.ObjectiveID = override.ObjectiveID
	}
	if override.MethodID != "" {
		result.MethodID = override.MethodID
	}
	if override.UserID != "" {
		result.UserID = override.UserID
	}
	if override.SessionID != "" {
		result.SessionID = override.SessionID
	}
	if override.Component != "" {
		result.Component = override.Component
	}

	return result
}

// DefaultLogConfig returns a default logging configuration.
func DefaultLogConfig(component string) LogConfig {
	return LogConfig{
		Level:          LogLevelInfo,
		Destination:    LogDestinationBoth,
		FilePath:       fmt.Sprintf("./logs/%s.log", component),
		FileMaxSize:    100, // 100MB
		FileMaxBackups: 5,
		FileMaxAge:     30, // 30 days
		Component:      component,
	}
}

// Loggers holds configured loggers for different components.
type LoggerManager struct {
	loggers map[string]Logger
}

// NewLoggerManager creates a new logger manager.
func NewLoggerManager() *LoggerManager {
	return &LoggerManager{
		loggers: make(map[string]Logger),
	}
}

// GetLogger returns a logger for the specified component, creating it if necessary.
func (lm *LoggerManager) GetLogger(component string, config ...LogConfig) (Logger, error) {
	if logger, exists := lm.loggers[component]; exists {
		return logger, nil
	}

	var logConfig LogConfig
	if len(config) > 0 {
		logConfig = config[0]
	} else {
		logConfig = DefaultLogConfig(component)
	}

	logger, err := NewLogger(logConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger for component %s: %w", component, err)
	}

	lm.loggers[component] = logger
	return logger, nil
}

// Close closes all managed loggers.
func (lm *LoggerManager) Close() error {
	var errors []string

	for component, logger := range lm.loggers {
		if err := logger.Close(); err != nil {
			errors = append(errors, fmt.Sprintf("failed to close logger for %s: %v", component, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing loggers: %s", strings.Join(errors, "; "))
	}

	return nil
}