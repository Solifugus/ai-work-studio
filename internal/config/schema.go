// Package config defines configuration schema and validation for AI Work Studio.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Config represents the complete application configuration.
type Config struct {
	// Storage configuration
	Storage StorageConfig `toml:"storage"`

	// API configuration for LLM services
	API APIConfig `toml:"api"`

	// Budget limits for cost management
	Budget BudgetConfig `toml:"budget"`

	// Permission settings for security
	Permissions PermissionConfig `toml:"permissions"`

	// User preferences for behavior customization
	Preferences PreferenceConfig `toml:"preferences"`

	// GUI window settings
	Window WindowConfig `toml:"window"`

	// Current session state
	Session SessionConfig `toml:"session"`

	// Convenience fields for CLI/UI/Agent compatibility (not serialized)
	DataDir      string        `toml:"-"`
	BudgetLimits *BudgetConfig `toml:"-"`
	WindowPrefs  *WindowConfig `toml:"-"`
}

// SyncConvenienceFields synchronizes convenience fields with main config fields
func (c *Config) SyncConvenienceFields() {
	c.DataDir = c.Storage.DataDir
	c.BudgetLimits = &c.Budget
	c.WindowPrefs = &c.Window
}

// SyncFromConvenienceFields updates main config fields from convenience fields
func (c *Config) SyncFromConvenienceFields() {
	if c.DataDir != "" {
		c.Storage.DataDir = c.DataDir
	}
}

// EnsureDataDir creates the data directory if it doesn't exist
func (c *Config) EnsureDataDir() error {
	// Sync convenience field to main field
	c.SyncFromConvenienceFields()

	dataDir := c.Storage.DataDir
	if dataDir == "" {
		return fmt.Errorf("data directory not configured")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory %s: %w", dataDir, err)
	}

	// Verify directory is writable
	testFile := filepath.Join(dataDir, ".write-test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("data directory %s is not writable: %w", dataDir, err)
	}
	os.Remove(testFile)

	return nil
}

// Save saves the configuration to a file
func (c *Config) Save(path string) error {
	// Sync convenience fields to main fields before saving
	c.SyncFromConvenienceFields()

	// Validate configuration before saving
	if err := c.Validate(); err != nil {
		return fmt.Errorf("cannot save invalid configuration: %w", err)
	}

	// Use manager to save
	manager := &Manager{configPath: path}
	return manager.Save(c)
}

// UpdateSession updates session state and saves to file
func (c *Config) UpdateSession(path string, updates SessionUpdates) error {
	manager := &Manager{configPath: path, config: c}
	return manager.UpdateSession(updates)
}

// UpdateBudgetLimits updates budget limits and saves to file
func (c *Config) UpdateBudgetLimits(path string, updates BudgetUpdates) error {
	manager := &Manager{configPath: path, config: c}
	return manager.UpdateBudget(updates)
}

// UpdatePreferences updates user preferences and saves to file
func (c *Config) UpdatePreferences(path string, updates PreferenceUpdates) error {
	manager := &Manager{configPath: path, config: c}
	return manager.UpdatePreferences(updates)
}

// UpdateWindow updates window preferences and saves to file
func (c *Config) UpdateWindow(path string, updates WindowUpdates) error {
	manager := &Manager{configPath: path, config: c}
	return manager.UpdateWindow(updates)
}

// UpdateWindowPreferences is an alias for UpdateWindow (for UI compatibility)
func (c *Config) UpdateWindowPreferences(path string, updates WindowUpdates) error {
	return c.UpdateWindow(path, updates)
}

// Load loads configuration from a file (convenience function)
func Load(path string) (*Config, error) {
	manager := &Manager{configPath: path}
	config, err := manager.Load()
	if err != nil {
		return nil, err
	}

	// Initialize convenience fields
	config.SyncConvenienceFields()
	return config, nil
}

// StorageConfig defines data storage settings.
type StorageConfig struct {
	// DataDir is the directory for storing application data
	DataDir string `toml:"data_dir"`

	// BackupEnabled determines if automatic backups are created
	BackupEnabled bool `toml:"backup_enabled"`

	// BackupRetention is the number of days to keep backups
	BackupRetention int `toml:"backup_retention_days"`
}

// APIConfig contains settings for LLM service APIs.
type APIConfig struct {
	// Anthropic Claude API configuration
	Anthropic AnthropicConfig `toml:"anthropic"`

	// OpenAI API configuration
	OpenAI OpenAIConfig `toml:"openai"`

	// Local model configuration (e.g., llama.cpp)
	Local LocalConfig `toml:"local"`

	// DefaultProvider specifies which provider to use by default
	DefaultProvider string `toml:"default_provider"`
}

// AnthropicConfig contains Anthropic Claude API settings.
type AnthropicConfig struct {
	// APIKey for authentication (prefer environment variable)
	APIKey string `toml:"api_key"`

	// BaseURL for API endpoint (allows custom endpoints)
	BaseURL string `toml:"base_url"`

	// DefaultModel to use for requests
	DefaultModel string `toml:"default_model"`
}

// OpenAIConfig contains OpenAI API settings.
type OpenAIConfig struct {
	// APIKey for authentication (prefer environment variable)
	APIKey string `toml:"api_key"`

	// BaseURL for API endpoint (allows custom endpoints)
	BaseURL string `toml:"base_url"`

	// DefaultModel to use for requests
	DefaultModel string `toml:"default_model"`
}

// LocalConfig contains local model settings.
type LocalConfig struct {
	// Enabled determines if local models are available
	Enabled bool `toml:"enabled"`

	// ModelPath is the path to local model files
	ModelPath string `toml:"model_path"`

	// ServerURL for local model server (e.g., llama.cpp server)
	ServerURL string `toml:"server_url"`
}

// BudgetConfig defines spending limits for LLM usage.
type BudgetConfig struct {
	// DailyLimit is the maximum daily spend (in USD)
	DailyLimit float64 `toml:"daily_limit"`

	// MonthlyLimit is the maximum monthly spend (in USD)
	MonthlyLimit float64 `toml:"monthly_limit"`

	// PerRequestLimit is the maximum spend per request (in USD)
	PerRequestLimit float64 `toml:"per_request_limit"`

	// TrackingEnabled determines if usage tracking is active
	TrackingEnabled bool `toml:"tracking_enabled"`
}

// PermissionConfig defines security and access control settings.
type PermissionConfig struct {
	// AllowedDirectories lists directories the agent can access
	AllowedDirectories []string `toml:"allowed_directories"`

	// RestrictedDirectories lists directories the agent cannot access
	RestrictedDirectories []string `toml:"restricted_directories"`

	// AllowNetworkAccess determines if network operations are allowed
	AllowNetworkAccess bool `toml:"allow_network_access"`

	// AllowFileWrites determines if file modification is allowed
	AllowFileWrites bool `toml:"allow_file_writes"`

	// RequireConfirmation lists operations requiring user confirmation
	RequireConfirmation []string `toml:"require_confirmation"`
}

// PreferenceConfig contains user behavior preferences.
type PreferenceConfig struct {
	// AutoApprove determines if low-risk operations are auto-approved
	AutoApprove bool `toml:"auto_approve"`

	// VerboseOutput enables detailed logging and status information
	VerboseOutput bool `toml:"verbose_output"`

	// DefaultPriority is the default priority for new goals (1-10)
	DefaultPriority int `toml:"default_priority"`

	// InteractiveMode enables conversation-like interaction
	InteractiveMode bool `toml:"interactive_mode"`

	// ConfirmDestructive requires confirmation for destructive operations
	ConfirmDestructive bool `toml:"confirm_destructive"`
}

// WindowConfig contains GUI window settings.
type WindowConfig struct {
	// Width of the main window in pixels
	Width int `toml:"width"`

	// Height of the main window in pixels
	Height int `toml:"height"`

	// X position of the main window
	X int `toml:"x"`

	// Y position of the main window
	Y int `toml:"y"`

	// Maximized indicates if the window should be maximized
	Maximized bool `toml:"maximized"`

	// ActiveTab is the index of the currently active tab
	ActiveTab int `toml:"active_tab"`

	// Theme specifies the UI theme (light, dark, auto)
	Theme string `toml:"theme"`
}

// SessionConfig tracks current session state.
type SessionConfig struct {
	// CurrentGoalID is the goal currently being worked on
	CurrentGoalID string `toml:"current_goal_id"`

	// LastUsedDataDir is the last data directory used
	LastUsedDataDir string `toml:"last_used_data_dir"`

	// UserID identifies the current user for context tracking
	UserID string `toml:"user_id"`
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	defaultDataDir := filepath.Join(homeDir, ".ai-work-studio", "data")

	config := &Config{
		Storage: StorageConfig{
			DataDir:         defaultDataDir,
			BackupEnabled:   true,
			BackupRetention: 30,
		},
		API: APIConfig{
			Anthropic: AnthropicConfig{
				APIKey:       "", // Set via environment variable ANTHROPIC_API_KEY
				BaseURL:      "https://api.anthropic.com",
				DefaultModel: "claude-3-sonnet-20241022",
			},
			OpenAI: OpenAIConfig{
				APIKey:       "", // Set via environment variable OPENAI_API_KEY
				BaseURL:      "https://api.openai.com/v1",
				DefaultModel: "gpt-4",
			},
			Local: LocalConfig{
				Enabled:   false,
				ModelPath: filepath.Join(homeDir, ".ai-work-studio", "models"),
				ServerURL: "http://localhost:8080",
			},
			DefaultProvider: "anthropic",
		},
		Budget: BudgetConfig{
			DailyLimit:      5.00,
			MonthlyLimit:    150.00,
			PerRequestLimit: 0.50,
			TrackingEnabled: true,
		},
		Permissions: PermissionConfig{
			AllowedDirectories: []string{
				homeDir,
				"/tmp",
			},
			RestrictedDirectories: []string{
				"/etc",
				"/usr",
				"/sys",
				"/proc",
			},
			AllowNetworkAccess:  true,
			AllowFileWrites:     true,
			RequireConfirmation: []string{"delete", "move", "rename"},
		},
		Preferences: PreferenceConfig{
			AutoApprove:        false,
			VerboseOutput:      false,
			DefaultPriority:    5,
			InteractiveMode:    true,
			ConfirmDestructive: true,
		},
		Window: WindowConfig{
			Width:     1200,
			Height:    800,
			X:         100,
			Y:         100,
			Maximized: false,
			ActiveTab: 0,
			Theme:     "auto",
		},
		Session: SessionConfig{
			CurrentGoalID:   "",
			LastUsedDataDir: defaultDataDir,
			UserID:          "default-user",
		},
	}

	// Initialize convenience fields
	config.SyncConvenienceFields()
	return config
}

// Validate performs comprehensive validation of the configuration.
func (c *Config) Validate() error {
	if err := c.validateStorage(); err != nil {
		return fmt.Errorf("storage validation failed: %w", err)
	}

	if err := c.validateAPI(); err != nil {
		return fmt.Errorf("API validation failed: %w", err)
	}

	if err := c.validateBudget(); err != nil {
		return fmt.Errorf("budget validation failed: %w", err)
	}

	if err := c.validatePermissions(); err != nil {
		return fmt.Errorf("permissions validation failed: %w", err)
	}

	if err := c.validatePreferences(); err != nil {
		return fmt.Errorf("preferences validation failed: %w", err)
	}

	if err := c.validateWindow(); err != nil {
		return fmt.Errorf("window validation failed: %w", err)
	}

	if err := c.validateSession(); err != nil {
		return fmt.Errorf("session validation failed: %w", err)
	}

	return nil
}

// validateStorage validates storage configuration.
func (c *Config) validateStorage() error {
	if c.Storage.DataDir == "" {
		return fmt.Errorf("data directory cannot be empty")
	}

	if c.Storage.BackupRetention < 1 {
		return fmt.Errorf("backup retention must be at least 1 day, got %d", c.Storage.BackupRetention)
	}

	return nil
}

// validateAPI validates API configuration.
func (c *Config) validateAPI() error {
	validProviders := []string{"anthropic", "openai", "local"}
	if !contains(validProviders, c.API.DefaultProvider) {
		return fmt.Errorf("invalid default provider %q, must be one of: %v", c.API.DefaultProvider, validProviders)
	}

	// Validate Anthropic config
	if c.API.Anthropic.BaseURL == "" {
		return fmt.Errorf("Anthropic base URL cannot be empty")
	}
	if c.API.Anthropic.DefaultModel == "" {
		return fmt.Errorf("Anthropic default model cannot be empty")
	}

	// Validate OpenAI config
	if c.API.OpenAI.BaseURL == "" {
		return fmt.Errorf("OpenAI base URL cannot be empty")
	}
	if c.API.OpenAI.DefaultModel == "" {
		return fmt.Errorf("OpenAI default model cannot be empty")
	}

	// Validate local config if enabled
	if c.API.Local.Enabled {
		if c.API.Local.ModelPath == "" {
			return fmt.Errorf("local model path cannot be empty when local models are enabled")
		}
		if c.API.Local.ServerURL == "" {
			return fmt.Errorf("local server URL cannot be empty when local models are enabled")
		}
	}

	return nil
}

// validateBudget validates budget configuration.
func (c *Config) validateBudget() error {
	if c.Budget.DailyLimit < 0 {
		return fmt.Errorf("daily budget limit cannot be negative")
	}

	if c.Budget.MonthlyLimit < 0 {
		return fmt.Errorf("monthly budget limit cannot be negative")
	}

	if c.Budget.PerRequestLimit < 0 {
		return fmt.Errorf("per-request budget limit cannot be negative")
	}

	// Check that daily limit is reasonable compared to monthly
	if c.Budget.DailyLimit*30 > c.Budget.MonthlyLimit && c.Budget.MonthlyLimit > 0 {
		return fmt.Errorf("daily limit * 30 (%.2f) exceeds monthly limit (%.2f)",
			c.Budget.DailyLimit*30, c.Budget.MonthlyLimit)
	}

	return nil
}

// validatePermissions validates permission configuration.
func (c *Config) validatePermissions() error {
	// Validate directory paths
	for i, dir := range c.Permissions.AllowedDirectories {
		if !filepath.IsAbs(dir) {
			return fmt.Errorf("allowed directory %d must be absolute path: %q", i, dir)
		}
	}

	for i, dir := range c.Permissions.RestrictedDirectories {
		if !filepath.IsAbs(dir) {
			return fmt.Errorf("restricted directory %d must be absolute path: %q", i, dir)
		}
	}

	// Validate confirmation requirements
	validConfirmations := []string{"delete", "move", "rename", "modify", "create"}
	for _, req := range c.Permissions.RequireConfirmation {
		if !contains(validConfirmations, req) {
			return fmt.Errorf("invalid confirmation requirement %q, must be one of: %v", req, validConfirmations)
		}
	}

	return nil
}

// validatePreferences validates preference configuration.
func (c *Config) validatePreferences() error {
	if c.Preferences.DefaultPriority < 1 || c.Preferences.DefaultPriority > 10 {
		return fmt.Errorf("default priority must be between 1 and 10, got %d", c.Preferences.DefaultPriority)
	}

	return nil
}

// validateWindow validates window configuration.
func (c *Config) validateWindow() error {
	if c.Window.Width < 400 {
		return fmt.Errorf("window width must be at least 400 pixels, got %d", c.Window.Width)
	}

	if c.Window.Height < 300 {
		return fmt.Errorf("window height must be at least 300 pixels, got %d", c.Window.Height)
	}

	if c.Window.ActiveTab < 0 || c.Window.ActiveTab > 4 {
		return fmt.Errorf("active tab must be between 0 and 4, got %d", c.Window.ActiveTab)
	}

	validThemes := []string{"light", "dark", "auto"}
	if !contains(validThemes, c.Window.Theme) {
		return fmt.Errorf("invalid theme %q, must be one of: %v", c.Window.Theme, validThemes)
	}

	return nil
}

// validateSession validates session configuration.
func (c *Config) validateSession() error {
	if c.Session.UserID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	// Validate user ID format (alphanumeric, dashes, underscores)
	validUserID := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validUserID.MatchString(c.Session.UserID) {
		return fmt.Errorf("user ID must contain only alphanumeric characters, dashes, and underscores: %q", c.Session.UserID)
	}

	return nil
}

// ApplyEnvironmentOverrides applies environment variable overrides to the configuration.
func (c *Config) ApplyEnvironmentOverrides() {
	// API key overrides (most important for security)
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		c.API.Anthropic.APIKey = apiKey
	}
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		c.API.OpenAI.APIKey = apiKey
	}

	// Data directory override
	if dataDir := os.Getenv("AI_WORK_STUDIO_DATA_DIR"); dataDir != "" {
		c.Storage.DataDir = dataDir
	}

	// Budget overrides
	if dailyLimit := os.Getenv("AI_WORK_STUDIO_DAILY_LIMIT"); dailyLimit != "" {
		if limit, err := parseFloat(dailyLimit); err == nil && limit >= 0 {
			c.Budget.DailyLimit = limit
		}
	}
	if monthlyLimit := os.Getenv("AI_WORK_STUDIO_MONTHLY_LIMIT"); monthlyLimit != "" {
		if limit, err := parseFloat(monthlyLimit); err == nil && limit >= 0 {
			c.Budget.MonthlyLimit = limit
		}
	}

	// Provider override
	if provider := os.Getenv("AI_WORK_STUDIO_PROVIDER"); provider != "" {
		if contains([]string{"anthropic", "openai", "local"}, provider) {
			c.API.DefaultProvider = provider
		}
	}

	// Verbose output override
	if verbose := os.Getenv("AI_WORK_STUDIO_VERBOSE"); verbose != "" {
		c.Preferences.VerboseOutput = strings.ToLower(verbose) == "true"
	}

	// User ID override
	if userID := os.Getenv("AI_WORK_STUDIO_USER_ID"); userID != "" {
		c.Session.UserID = userID
	}
}

// GetConfigPath returns the default configuration file path.
func GetConfigPath() (string, error) {
	// Check for custom config path environment variable
	if configPath := os.Getenv("AI_WORK_STUDIO_CONFIG"); configPath != "" {
		return configPath, nil
	}

	// Use XDG config directory or fallback to ~/.config
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		configDir = filepath.Join(homeDir, ".config")
	}

	return filepath.Join(configDir, "ai-work-studio", "config.toml"), nil
}

// Helper functions

// contains checks if a slice contains a specific string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// parseFloat safely parses a float64 from a string.
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}