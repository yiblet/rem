package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.HistoryLimit != 20 {
		t.Errorf("Expected default history limit 20, got %d", config.HistoryLimit)
	}

	if config.ShowBinary != false {
		t.Errorf("Expected default show binary false, got %t", config.ShowBinary)
	}

	if config.HistoryLocation != "" {
		t.Errorf("Expected default history location empty, got %s", config.HistoryLocation)
	}
}

func TestConfigManager_LoadNonExistent(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	cm := NewConfigManagerWithPath(configPath)

	config, err := cm.Load()
	if err != nil {
		t.Fatalf("Expected no error loading non-existent config, got: %v", err)
	}

	// Should return default config
	expectedDefault := DefaultConfig()
	if config.HistoryLimit != expectedDefault.HistoryLimit {
		t.Errorf("Expected default history limit %d, got %d", expectedDefault.HistoryLimit, config.HistoryLimit)
	}
}

func TestConfigManager_SaveAndLoad(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	cm := NewConfigManagerWithPath(configPath)

	// Create test config
	testConfig := &Config{
		HistoryLimit:    100,
		ShowBinary:      true,
		HistoryLocation: "/custom/path",
	}

	// Save config
	err := cm.Save(testConfig)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load config
	loadedConfig, err := cm.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify values
	if loadedConfig.HistoryLimit != testConfig.HistoryLimit {
		t.Errorf("Expected history limit %d, got %d", testConfig.HistoryLimit, loadedConfig.HistoryLimit)
	}

	if loadedConfig.ShowBinary != testConfig.ShowBinary {
		t.Errorf("Expected show binary %t, got %t", testConfig.ShowBinary, loadedConfig.ShowBinary)
	}

	if loadedConfig.HistoryLocation != testConfig.HistoryLocation {
		t.Errorf("Expected history location %s, got %s", testConfig.HistoryLocation, loadedConfig.HistoryLocation)
	}
}

func TestConfigManager_Validation(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	cm := NewConfigManagerWithPath(configPath)

	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &Config{
				HistoryLimit: 50,
				ShowBinary:   false,
			},
			expectError: false,
		},
		{
			name: "zero history limit",
			config: &Config{
				HistoryLimit: 0,
			},
			expectError: true,
			errorMsg:    "history_limit must be greater than 0",
		},
		{
			name: "negative history limit",
			config: &Config{
				HistoryLimit: -5,
			},
			expectError: true,
			errorMsg:    "history_limit must be greater than 0",
		},
		{
			name: "excessive history limit",
			config: &Config{
				HistoryLimit: 1500,
			},
			expectError: true,
			errorMsg:    "history_limit cannot exceed 1000 items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cm.Save(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				} else if tt.errorMsg != "" && err.Error() != "invalid configuration: "+tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.name, err)
				}
			}
		})
	}
}

func TestConfigManager_Update(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	cm := NewConfigManagerWithPath(configPath)

	tests := []struct {
		name        string
		key         string
		value       string
		expectError bool
	}{
		{"valid history-limit", "history-limit", "100", false},
		{"valid show-binary true", "show-binary", "true", false},
		{"valid show-binary false", "show-binary", "false", false},
		{"valid history-location", "history-location", "/custom/path", false},
		{"invalid key", "invalid-key", "value", true},
		{"invalid history-limit", "history-limit", "not-a-number", true},
		{"invalid show-binary", "show-binary", "maybe", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cm.Update(tt.key, tt.value)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.name, err)
				}

				// Verify the value was set correctly
				retrievedValue, err := cm.Get(tt.key)
				if err != nil {
					t.Errorf("Failed to get value after update: %v", err)
				} else if retrievedValue != tt.value {
					t.Errorf("Expected retrieved value %s, got %s", tt.value, retrievedValue)
				}
			}
		})
	}
}

func TestConfigManager_Get(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	cm := NewConfigManagerWithPath(configPath)

	// Set up a config first
	config := &Config{
		HistoryLimit:    75,
		ShowBinary:      true,
		HistoryLocation: "/test/path",
	}

	err := cm.Save(config)
	if err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	tests := []struct {
		name          string
		key           string
		expectedValue string
		expectError   bool
	}{
		{"get history-limit", "history-limit", "75", false},
		{"get show-binary", "show-binary", "true", false},
		{"get history-location", "history-location", "/test/path", false},
		{"get invalid key", "invalid-key", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := cm.Get(tt.key)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.name, err)
				} else if value != tt.expectedValue {
					t.Errorf("Expected value %s, got %s", tt.expectedValue, value)
				}
			}
		})
	}
}

func TestConfigManager_List(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	cm := NewConfigManagerWithPath(configPath)

	// Use default config first (no file exists)
	values, err := cm.List()
	if err != nil {
		t.Fatalf("Failed to list default config: %v", err)
	}

	expectedKeys := []string{"history-limit", "show-binary", "history-location"}
	for _, key := range expectedKeys {
		if _, exists := values[key]; !exists {
			t.Errorf("Expected key %s to exist in list output", key)
		}
	}

	// Verify default values
	if values["history-limit"] != "20" {
		t.Errorf("Expected default history-limit 20, got %s", values["history-limit"])
	}

	if values["history-location"] != "[default]" {
		t.Errorf("Expected default history-location [default], got %s", values["history-location"])
	}
}

func TestConfigManager_GetConfigPath(t *testing.T) {
	configPath := "/test/config/path.yaml"
	cm := NewConfigManagerWithPath(configPath)

	if cm.GetConfigPath() != configPath {
		t.Errorf("Expected config path %s, got %s", configPath, cm.GetConfigPath())
	}
}

func TestNewConfigManager(t *testing.T) {
	cm, err := NewConfigManager()
	if err != nil {
		t.Fatalf("Failed to create config manager: %v", err)
	}

	// Should contain .config/rem/config.yaml in the path
	configPath := cm.GetConfigPath()
	if !filepath.IsAbs(configPath) {
		t.Errorf("Expected absolute config path, got %s", configPath)
	}

	if !strings.HasSuffix(configPath, ".config/rem/config.yaml") {
		t.Errorf("Expected config path to end with .config/rem/config.yaml, got %s", configPath)
	}
}