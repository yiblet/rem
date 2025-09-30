package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the rem configuration
type Config struct {
	HistoryLimit    int    `yaml:"history_limit"`
	HistoryLocation string `yaml:"history_location,omitempty"`
	ShowBinary      bool   `yaml:"show_binary"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		HistoryLimit: 20,
		ShowBinary:   false,
	}
}

// ConfigManager manages configuration persistence
type ConfigManager struct {
	configPath string
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() (*ConfigManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "rem")
	configPath := filepath.Join(configDir, "config.yaml")

	return &ConfigManager{
		configPath: configPath,
	}, nil
}

// NewConfigManagerWithPath creates a config manager with custom config path
func NewConfigManagerWithPath(configPath string) *ConfigManager {
	return &ConfigManager{
		configPath: configPath,
	}
}

// Load reads the configuration from file, or returns default if file doesn't exist
func (cm *ConfigManager) Load() (*Config, error) {
	// If config file doesn't exist, return default config
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate and set defaults for missing fields
	if err := cm.validateAndSetDefaults(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// Save writes the configuration to file
func (cm *ConfigManager) Save(config *Config) error {
	// Validate configuration before saving
	if err := cm.validateAndSetDefaults(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Ensure config directory exists
	configDir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// validateAndSetDefaults validates configuration and sets defaults for missing fields
func (cm *ConfigManager) validateAndSetDefaults(config *Config) error {
	if config.HistoryLimit <= 0 {
		return fmt.Errorf("history_limit must be greater than 0")
	}

	if config.HistoryLimit > 1000 {
		return fmt.Errorf("history_limit cannot exceed 1000 items")
	}

	return nil
}

// GetConfigPath returns the path to the config file
func (cm *ConfigManager) GetConfigPath() string {
	return cm.configPath
}

// Update modifies a specific configuration value
func (cm *ConfigManager) Update(key, value string) error {
	config, err := cm.Load()
	if err != nil {
		return err
	}

	switch key {
	case "history-limit":
		var historyLimit int
		if _, err := fmt.Sscanf(value, "%d", &historyLimit); err != nil {
			return fmt.Errorf("invalid integer value for history-limit: %s", value)
		}
		config.HistoryLimit = historyLimit
	case "show-binary":
		switch value {
		case "true":
			config.ShowBinary = true
		case "false":
			config.ShowBinary = false
		default:
			return fmt.Errorf("invalid boolean value for show-binary: %s (must be 'true' or 'false')", value)
		}
	case "history-location":
		config.HistoryLocation = value
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	return cm.Save(config)
}

// Get returns the value for a specific configuration key
func (cm *ConfigManager) Get(key string) (string, error) {
	config, err := cm.Load()
	if err != nil {
		return "", err
	}

	switch key {
	case "history-limit":
		return fmt.Sprintf("%d", config.HistoryLimit), nil
	case "show-binary":
		return fmt.Sprintf("%t", config.ShowBinary), nil
	case "history-location":
		if config.HistoryLocation == "" {
			return "[default]", nil
		}
		return config.HistoryLocation, nil
	default:
		return "", fmt.Errorf("unknown configuration key: %s", key)
	}
}

// List returns all configuration keys and values
func (cm *ConfigManager) List() (map[string]string, error) {
	config, err := cm.Load()
	if err != nil {
		return nil, err
	}

	result := map[string]string{
		"history-limit":    fmt.Sprintf("%d", config.HistoryLimit),
		"show-binary":      fmt.Sprintf("%t", config.ShowBinary),
		"history-location": config.HistoryLocation,
	}

	if result["history-location"] == "" {
		result["history-location"] = "[default]"
	}

	return result, nil
}
