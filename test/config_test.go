package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Solifugus/ai-work-studio/internal/config"
)

// TestConfigManager tests the complete configuration management workflow.
func TestConfigManager(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "ai-work-studio-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.toml")

	t.Run("NewManager", func(t *testing.T) {
		manager := config.NewManagerWithPath(configPath)
		if manager == nil {
			t.Fatal("NewManagerWithPath returned nil")
		}

		if manager.GetPath() != configPath {
			t.Errorf("Expected config path %q, got %q", configPath, manager.GetPath())
		}
	})

	t.Run("LoadDefaultConfig", func(t *testing.T) {
		manager := config.NewManagerWithPath(configPath)

		// Load config when file doesn't exist (should return defaults)
		cfg, err := manager.Load()
		if err != nil {
			t.Fatalf("Failed to load default config: %v", err)
		}

		// Verify default values
		if cfg.API.DefaultProvider != "anthropic" {
			t.Errorf("Expected default provider 'anthropic', got %q", cfg.API.DefaultProvider)
		}

		if cfg.Budget.DailyLimit != 5.00 {
			t.Errorf("Expected daily limit 5.00, got %f", cfg.Budget.DailyLimit)
		}

		if cfg.Preferences.DefaultPriority != 5 {
			t.Errorf("Expected default priority 5, got %d", cfg.Preferences.DefaultPriority)
		}

		if cfg.Window.Width != 1200 {
			t.Errorf("Expected window width 1200, got %d", cfg.Window.Width)
		}
	})

	t.Run("SaveAndLoadConfig", func(t *testing.T) {
		manager := config.NewManagerWithPath(configPath)

		// Load default config
		cfg, err := manager.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Modify config
		cfg.API.DefaultProvider = "openai"
		cfg.Budget.DailyLimit = 3.00
		cfg.Preferences.VerboseOutput = true

		// Save config
		if err := manager.Save(cfg); err != nil {
			t.Fatalf("Failed to save config: %v", err)
		}

		// Verify file was created
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Fatal("Config file was not created")
		}

		// Load again and verify changes
		newManager := config.NewManagerWithPath(configPath)
		loadedCfg, err := newManager.Load()
		if err != nil {
			t.Fatalf("Failed to reload config: %v", err)
		}

		if loadedCfg.API.DefaultProvider != "openai" {
			t.Errorf("Expected provider 'openai', got %q", loadedCfg.API.DefaultProvider)
		}

		if loadedCfg.Budget.DailyLimit != 3.00 {
			t.Errorf("Expected daily limit 3.00, got %f", loadedCfg.Budget.DailyLimit)
		}

		if !loadedCfg.Preferences.VerboseOutput {
			t.Error("Expected verbose output to be true")
		}
	})

	t.Run("EnvironmentOverrides", func(t *testing.T) {
		// Set environment variables
		os.Setenv("ANTHROPIC_API_KEY", "test-key-anthropic")
		os.Setenv("OPENAI_API_KEY", "test-key-openai")
		os.Setenv("AI_WORK_STUDIO_PROVIDER", "local")
		os.Setenv("AI_WORK_STUDIO_VERBOSE", "true")
		os.Setenv("AI_WORK_STUDIO_DAILY_LIMIT", "4.50")
		defer func() {
			os.Unsetenv("ANTHROPIC_API_KEY")
			os.Unsetenv("OPENAI_API_KEY")
			os.Unsetenv("AI_WORK_STUDIO_PROVIDER")
			os.Unsetenv("AI_WORK_STUDIO_VERBOSE")
			os.Unsetenv("AI_WORK_STUDIO_DAILY_LIMIT")
		}()

		manager := config.NewManagerWithPath(configPath)
		cfg, err := manager.Load()
		if err != nil {
			t.Fatalf("Failed to load config with env overrides: %v", err)
		}

		// Verify environment overrides
		if cfg.API.Anthropic.APIKey != "test-key-anthropic" {
			t.Errorf("Expected Anthropic API key 'test-key-anthropic', got %q", cfg.API.Anthropic.APIKey)
		}

		if cfg.API.OpenAI.APIKey != "test-key-openai" {
			t.Errorf("Expected OpenAI API key 'test-key-openai', got %q", cfg.API.OpenAI.APIKey)
		}

		if cfg.API.DefaultProvider != "local" {
			t.Errorf("Expected provider 'local', got %q", cfg.API.DefaultProvider)
		}

		if !cfg.Preferences.VerboseOutput {
			t.Error("Expected verbose output to be true from environment")
		}

		if cfg.Budget.DailyLimit != 4.50 {
			t.Errorf("Expected daily limit 4.50, got %f", cfg.Budget.DailyLimit)
		}
	})

	t.Run("UpdateMethods", func(t *testing.T) {
		manager := config.NewManagerWithPath(configPath)
		_, err := manager.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Test budget updates
		budgetUpdates := config.BudgetUpdates{
			DailyLimit:   floatToPtr(4.00),
			MonthlyLimit: floatToPtr(200.00),
		}
		if err := manager.UpdateBudget(budgetUpdates); err != nil {
			t.Fatalf("Failed to update budget: %v", err)
		}

		// Test preference updates
		preferenceUpdates := config.PreferenceUpdates{
			AutoApprove:     boolToPtr(true),
			DefaultPriority: intToPtr(7),
		}
		if err := manager.UpdatePreferences(preferenceUpdates); err != nil {
			t.Fatalf("Failed to update preferences: %v", err)
		}

		// Test session updates
		sessionUpdates := config.SessionUpdates{
			CurrentGoalID: stringToPtr("test-goal-123"),
			UserID:        stringToPtr("test-user"),
		}
		if err := manager.UpdateSession(sessionUpdates); err != nil {
			t.Fatalf("Failed to update session: %v", err)
		}

		// Verify updates were saved
		newCfg := manager.GetConfig()
		if newCfg.Budget.DailyLimit != 4.00 {
			t.Errorf("Expected daily limit 4.00, got %f", newCfg.Budget.DailyLimit)
		}

		if !newCfg.Preferences.AutoApprove {
			t.Error("Expected auto approve to be true")
		}

		if newCfg.Preferences.DefaultPriority != 7 {
			t.Errorf("Expected priority 7, got %d", newCfg.Preferences.DefaultPriority)
		}

		if newCfg.Session.CurrentGoalID != "test-goal-123" {
			t.Errorf("Expected goal ID 'test-goal-123', got %q", newCfg.Session.CurrentGoalID)
		}
	})

	t.Run("EnsureDataDir", func(t *testing.T) {
		manager := config.NewManagerWithPath(configPath)
		cfg, err := manager.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Set data dir to temp location
		testDataDir := filepath.Join(tempDir, "test-data")
		cfg.Storage.DataDir = testDataDir
		if err := manager.Save(cfg); err != nil {
			t.Fatalf("Failed to save config with new data dir: %v", err)
		}

		// Test directory creation
		if err := manager.EnsureDataDir(); err != nil {
			t.Fatalf("Failed to ensure data dir: %v", err)
		}

		// Verify directory exists
		if _, err := os.Stat(testDataDir); os.IsNotExist(err) {
			t.Error("Data directory was not created")
		}
	})
}

// TestConfigValidation tests configuration validation rules.
func TestConfigValidation(t *testing.T) {
	t.Run("ValidDefaultConfig", func(t *testing.T) {
		cfg := config.DefaultConfig()
		if err := cfg.Validate(); err != nil {
			t.Errorf("Default config should be valid: %v", err)
		}
	})

	t.Run("InvalidBudgetLimits", func(t *testing.T) {
		cfg := config.DefaultConfig()

		// Test negative daily limit
		cfg.Budget.DailyLimit = -1.0
		if err := cfg.Validate(); err == nil {
			t.Error("Expected validation error for negative daily limit")
		}

		// Reset and test invalid monthly vs daily
		cfg = config.DefaultConfig()
		cfg.Budget.DailyLimit = 100.0
		cfg.Budget.MonthlyLimit = 50.0
		if err := cfg.Validate(); err == nil {
			t.Error("Expected validation error when daily*30 > monthly")
		}
	})

	t.Run("InvalidPriority", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Preferences.DefaultPriority = 11
		if err := cfg.Validate(); err == nil {
			t.Error("Expected validation error for priority > 10")
		}

		cfg.Preferences.DefaultPriority = 0
		if err := cfg.Validate(); err == nil {
			t.Error("Expected validation error for priority < 1")
		}
	})

	t.Run("InvalidWindowSettings", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Window.Width = 100
		if err := cfg.Validate(); err == nil {
			t.Error("Expected validation error for width < 400")
		}

		cfg = config.DefaultConfig()
		cfg.Window.Height = 100
		if err := cfg.Validate(); err == nil {
			t.Error("Expected validation error for height < 300")
		}
	})

	t.Run("InvalidProvider", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.API.DefaultProvider = "invalid"
		if err := cfg.Validate(); err == nil {
			t.Error("Expected validation error for invalid provider")
		}
	})

	t.Run("EmptyRequiredFields", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Storage.DataDir = ""
		if err := cfg.Validate(); err == nil {
			t.Error("Expected validation error for empty data dir")
		}

		cfg = config.DefaultConfig()
		cfg.Session.UserID = ""
		if err := cfg.Validate(); err == nil {
			t.Error("Expected validation error for empty user ID")
		}
	})
}

// TestConfigPath tests configuration path detection.
func TestConfigPath(t *testing.T) {
	t.Run("DefaultPath", func(t *testing.T) {
		// Temporarily unset custom config env var
		originalConfig := os.Getenv("AI_WORK_STUDIO_CONFIG")
		os.Unsetenv("AI_WORK_STUDIO_CONFIG")
		defer func() {
			if originalConfig != "" {
				os.Setenv("AI_WORK_STUDIO_CONFIG", originalConfig)
			}
		}()

		path, err := config.GetConfigPath()
		if err != nil {
			t.Fatalf("Failed to get config path: %v", err)
		}

		expected := filepath.Join(os.Getenv("HOME"), ".config", "ai-work-studio", "config.toml")
		if path != expected {
			t.Errorf("Expected path %q, got %q", expected, path)
		}
	})

	t.Run("CustomPath", func(t *testing.T) {
		customPath := "/custom/config/path.toml"
		os.Setenv("AI_WORK_STUDIO_CONFIG", customPath)
		defer os.Unsetenv("AI_WORK_STUDIO_CONFIG")

		path, err := config.GetConfigPath()
		if err != nil {
			t.Fatalf("Failed to get custom config path: %v", err)
		}

		if path != customPath {
			t.Errorf("Expected custom path %q, got %q", customPath, path)
		}
	})
}

// Helper functions for pointer conversions
func floatToPtr(f float64) *float64 {
	return &f
}

func stringToPtr(s string) *string {
	return &s
}

func boolToPtr(b bool) *bool {
	return &b
}

func intToPtr(i int) *int {
	return &i
}