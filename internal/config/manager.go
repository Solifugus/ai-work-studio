// Package config provides configuration management for AI Work Studio.
// It handles loading, saving, and validating user preferences and settings using TOML format.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Manager handles configuration file operations with TOML format.
type Manager struct {
	configPath string
	config     *Config
}

// NewManager creates a new configuration manager.
func NewManager() (*Manager, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to determine config path: %w", err)
	}

	return &Manager{
		configPath: configPath,
	}, nil
}

// NewManagerWithPath creates a new configuration manager with a custom config path.
func NewManagerWithPath(configPath string) *Manager {
	return &Manager{
		configPath: configPath,
	}
}

// Load reads and validates configuration from file.
// If the file doesn't exist, returns default configuration.
// Environment variables override configuration values.
func (m *Manager) Load() (*Config, error) {
	var config *Config

	// Check if config file exists
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		// File doesn't exist, use defaults
		config = DefaultConfig()
	} else {
		// File exists, read it
		var err error
		config, err = m.readConfigFile()
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Apply environment variable overrides
	config.ApplyEnvironmentOverrides()

	// Validate the final configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	m.config = config
	return config, nil
}

// Save writes configuration to file with proper TOML formatting.
func (m *Manager) Save(config *Config) error {
	// Validate configuration before saving
	if err := config.Validate(); err != nil {
		return fmt.Errorf("cannot save invalid configuration: %w", err)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write to temporary file first (atomic write)
	tempPath := m.configPath + ".tmp"
	if err := m.writeConfigFile(config, tempPath); err != nil {
		return fmt.Errorf("failed to write temporary config file: %w", err)
	}

	// Atomic move
	if err := os.Rename(tempPath, m.configPath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to save config file: %w", err)
	}

	m.config = config
	return nil
}

// GetPath returns the configuration file path.
func (m *Manager) GetPath() string {
	return m.configPath
}

// EnsureDataDir ensures the data directory exists and is writable.
func (m *Manager) EnsureDataDir() error {
	if m.config == nil {
		return fmt.Errorf("configuration not loaded")
	}

	dataDir := m.config.Storage.DataDir

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory %q: %w", dataDir, err)
	}

	// Test write permissions by creating a temporary file
	testFile := filepath.Join(dataDir, ".write_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("data directory %q is not writable: %w", dataDir, err)
	}

	// Clean up test file
	os.Remove(testFile)

	return nil
}

// UpdateStorage updates storage configuration and saves.
func (m *Manager) UpdateStorage(updates StorageUpdates) error {
	if m.config == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Apply updates
	if updates.DataDir != nil {
		m.config.Storage.DataDir = *updates.DataDir
	}
	if updates.BackupEnabled != nil {
		m.config.Storage.BackupEnabled = *updates.BackupEnabled
	}
	if updates.BackupRetention != nil {
		if *updates.BackupRetention < 1 {
			return fmt.Errorf("backup retention must be at least 1 day")
		}
		m.config.Storage.BackupRetention = *updates.BackupRetention
	}

	return m.Save(m.config)
}

// UpdateBudget updates budget configuration and saves.
func (m *Manager) UpdateBudget(updates BudgetUpdates) error {
	if m.config == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Apply updates with validation
	if updates.DailyLimit != nil {
		if *updates.DailyLimit < 0 {
			return fmt.Errorf("daily limit cannot be negative")
		}
		m.config.Budget.DailyLimit = *updates.DailyLimit
	}
	if updates.MonthlyLimit != nil {
		if *updates.MonthlyLimit < 0 {
			return fmt.Errorf("monthly limit cannot be negative")
		}
		m.config.Budget.MonthlyLimit = *updates.MonthlyLimit
	}
	if updates.PerRequestLimit != nil {
		if *updates.PerRequestLimit < 0 {
			return fmt.Errorf("per-request limit cannot be negative")
		}
		m.config.Budget.PerRequestLimit = *updates.PerRequestLimit
	}
	if updates.TrackingEnabled != nil {
		m.config.Budget.TrackingEnabled = *updates.TrackingEnabled
	}

	return m.Save(m.config)
}

// UpdatePreferences updates user preferences and saves.
func (m *Manager) UpdatePreferences(updates PreferenceUpdates) error {
	if m.config == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Apply updates
	if updates.AutoApprove != nil {
		m.config.Preferences.AutoApprove = *updates.AutoApprove
	}
	if updates.VerboseOutput != nil {
		m.config.Preferences.VerboseOutput = *updates.VerboseOutput
	}
	if updates.DefaultPriority != nil {
		if *updates.DefaultPriority < 1 || *updates.DefaultPriority > 10 {
			return fmt.Errorf("default priority must be between 1 and 10")
		}
		m.config.Preferences.DefaultPriority = *updates.DefaultPriority
	}
	if updates.InteractiveMode != nil {
		m.config.Preferences.InteractiveMode = *updates.InteractiveMode
	}
	if updates.ConfirmDestructive != nil {
		m.config.Preferences.ConfirmDestructive = *updates.ConfirmDestructive
	}

	return m.Save(m.config)
}

// UpdateSession updates session state and saves.
func (m *Manager) UpdateSession(updates SessionUpdates) error {
	if m.config == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Apply updates
	if updates.CurrentGoalID != nil {
		m.config.Session.CurrentGoalID = *updates.CurrentGoalID
	}
	if updates.LastUsedDataDir != nil {
		m.config.Session.LastUsedDataDir = *updates.LastUsedDataDir
		// Also update the main data directory
		m.config.Storage.DataDir = *updates.LastUsedDataDir
	}
	if updates.UserID != nil {
		m.config.Session.UserID = *updates.UserID
	}

	return m.Save(m.config)
}

// UpdateWindow updates window preferences and saves.
func (m *Manager) UpdateWindow(updates WindowUpdates) error {
	if m.config == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Apply updates with validation
	if updates.Width != nil {
		if *updates.Width < 400 {
			return fmt.Errorf("window width must be at least 400 pixels")
		}
		m.config.Window.Width = *updates.Width
	}
	if updates.Height != nil {
		if *updates.Height < 300 {
			return fmt.Errorf("window height must be at least 300 pixels")
		}
		m.config.Window.Height = *updates.Height
	}
	if updates.X != nil {
		m.config.Window.X = *updates.X
	}
	if updates.Y != nil {
		m.config.Window.Y = *updates.Y
	}
	if updates.Maximized != nil {
		m.config.Window.Maximized = *updates.Maximized
	}
	if updates.ActiveTab != nil {
		if *updates.ActiveTab < 0 || *updates.ActiveTab > 4 {
			return fmt.Errorf("active tab must be between 0 and 4")
		}
		m.config.Window.ActiveTab = *updates.ActiveTab
	}
	if updates.Theme != nil {
		validThemes := []string{"light", "dark", "auto"}
		if !contains(validThemes, *updates.Theme) {
			return fmt.Errorf("invalid theme %q, must be one of: %v", *updates.Theme, validThemes)
		}
		m.config.Window.Theme = *updates.Theme
	}

	return m.Save(m.config)
}

// GetConfig returns the currently loaded configuration.
func (m *Manager) GetConfig() *Config {
	return m.config
}

// readConfigFile reads and parses the TOML configuration file.
func (m *Manager) readConfigFile() (*Config, error) {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse TOML config: %w", err)
	}

	return &config, nil
}

// writeConfigFile writes configuration to TOML file with proper formatting.
func (m *Manager) writeConfigFile(config *Config, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Create TOML encoder with custom configuration
	encoder := toml.NewEncoder(file)
	encoder.Indent = "  " // Use 2-space indentation

	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to encode TOML: %w", err)
	}

	return nil
}

// Update structures for partial updates

// StorageUpdates contains optional storage configuration updates.
type StorageUpdates struct {
	DataDir         *string
	BackupEnabled   *bool
	BackupRetention *int
}

// BudgetUpdates contains optional budget configuration updates.
type BudgetUpdates struct {
	DailyLimit      *float64
	MonthlyLimit    *float64
	PerRequestLimit *float64
	TrackingEnabled *bool
}

// PreferenceUpdates contains optional preference updates.
type PreferenceUpdates struct {
	AutoApprove        *bool
	VerboseOutput      *bool
	DefaultPriority    *int
	InteractiveMode    *bool
	ConfirmDestructive *bool
}

// SessionUpdates contains optional session updates.
type SessionUpdates struct {
	CurrentGoalID   *string
	LastUsedDataDir *string
	UserID          *string
}

// WindowUpdates contains optional window preference updates.
type WindowUpdates struct {
	Width     *int
	Height    *int
	X         *int
	Y         *int
	Maximized *bool
	ActiveTab *int
	Theme     *string
}