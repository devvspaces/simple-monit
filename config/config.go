// config/config.go
package config

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	Monitor       MonitorConfig              `yaml:"monitor"`
	Collectors    map[string]CollectorConfig `yaml:"collectors"`
	Notifications NotificationsConfig        `yaml:"notifications"`
}

// MonitorConfig contains global monitoring settings
type MonitorConfig struct {
	DefaultIntervalSeconds int `yaml:"default_interval_seconds"`
}

// CollectorConfig represents a generic collector configuration
type CollectorConfig struct {
	Enabled  bool                   `yaml:"enabled"`
	Interval int                    `yaml:"interval_seconds,omitempty"`
	Settings map[string]interface{} `yaml:"settings,omitempty"`
}

// NotificationsConfig contains all notification methods
type NotificationsConfig struct {
	Email EmailConfig `yaml:"email"`
}

// EmailConfig contains email notification settings
type EmailConfig struct {
	Enabled    bool     `yaml:"enabled"`
	From       string   `yaml:"from"`
	To         []string `yaml:"to"`
	SMTPServer string   `yaml:"smtp_server"`
	SMTPPort   int      `yaml:"smtp_port"`
	Username   string   `yaml:"username"`
	Password   string   `yaml:"password"`
}

// LoadConfig loads the configuration from the specified file path
func LoadConfig(logger *zap.Logger, path string) (*Config, error) {
	// Read configuration file
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Error("Error reading config file", zap.String("path", path), zap.Error(err))
		return nil, err
	}

	// Parse configuration
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		logger.Error("Error parsing config file", zap.String("path", path), zap.Error(err))
		return nil, err
	}

	// Validate configuration
	if err := validateConfig(logger.Named("validate"), &config); err != nil {
		logger.Error("Invalid configuration", zap.String("path", path), zap.Error(err))
		return nil, err
	}

	return &config, nil
}

// validateConfig performs basic validation on the configuration
func validateConfig(logger *zap.Logger, config *Config) error {
	// Ensure we have a valid default interval
	if config.Monitor.DefaultIntervalSeconds <= 0 {
		logger.Error("Invalid default interval", zap.Int("default_interval_seconds", config.Monitor.DefaultIntervalSeconds))
		return fmt.Errorf("monitor.default_interval_seconds must be greater than 0")
	}

	// Set default intervals for collectors if not specified
	for name, collector := range config.Collectors {
		if collector.Enabled && collector.Interval <= 0 {
			collector.Interval = config.Monitor.DefaultIntervalSeconds
			config.Collectors[name] = collector
		}
	}

	// Validate email configuration if enabled
	if config.Notifications.Email.Enabled {
		if config.Notifications.Email.From == "" {
			logger.Error("Email 'from' address is empty")
			return fmt.Errorf("email notification enabled but 'from' address is empty")
		}
		if len(config.Notifications.Email.To) == 0 {
			logger.Error("Email 'to' addresses are empty")
			return fmt.Errorf("email notification enabled but 'to' addresses are empty")
		}
		if config.Notifications.Email.SMTPServer == "" {
			logger.Error("Email SMTP server is empty")
			return fmt.Errorf("email notification enabled but 'smtp_server' is empty")
		}
		if config.Notifications.Email.SMTPPort <= 0 {
			logger.Error("Email SMTP port is invalid")
			return fmt.Errorf("email notification enabled but 'smtp_port' is invalid")
		}
	}

	return nil
}

// GetCollectorInterval returns the interval for a collector in duration
func (c *Config) GetCollectorInterval(collectorName string) time.Duration {
	collector, exists := c.Collectors[collectorName]
	if !exists || !collector.Enabled {
		return 0
	}

	interval := collector.Interval
	if interval <= 0 {
		interval = c.Monitor.DefaultIntervalSeconds
	}

	return time.Duration(interval) * time.Second
}
