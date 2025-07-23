package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config represents the application configuration
type Config struct {
	NatsURL     string `toml:"nats_url"`
	ScriptsPath string `toml:"scripts_path"`
	LogLevel    string `toml:"log_level"`
	Hostname    string `toml:"hostname"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() Config {
	return Config{
		NatsURL:     "nats://127.0.0.1:4222",
		ScriptsPath: "./scripts",
		LogLevel:    "info",
		Hostname:    "auto",
	}
}

// ResolveHostname returns the actual hostname to use
// If hostname is "auto" or empty, it returns the system hostname
// Otherwise it returns the configured hostname
func (c Config) ResolveHostname() (string, error) {
	if c.Hostname == "auto" || c.Hostname == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return "", fmt.Errorf("failed to get system hostname: %w", err)
		}
		return hostname, nil
	}
	return c.Hostname, nil
}

// PrefixSubject prefixes a NATS subject with the resolved hostname
func (c Config) PrefixSubject(subject string) string {
	hostname, err := c.ResolveHostname()
	if err != nil {
		// Fallback to "unknown" if hostname resolution fails
		hostname = "unknown"
	}
	return hostname + "." + subject
}

// LoadConfig loads configuration from a TOML file
func LoadConfig(path string) (Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return Config{}, fmt.Errorf("config file not found: %s", path)
	}

	// Start with an empty config to detect missing required fields
	var config Config
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return Config{}, fmt.Errorf("failed to decode config file: %w", err)
	}

	// Apply defaults for optional fields
	if config.LogLevel == "" {
		config.LogLevel = "info"
	}

	if config.Hostname == "" {
		config.Hostname = "auto"
	}

	if err := config.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Validate checks if the configuration is valid
func (c Config) Validate() error {
	if c.NatsURL == "" {
		return fmt.Errorf("nats_url is required")
	}

	if c.ScriptsPath == "" {
		return fmt.Errorf("scripts_path is required")
	}

	validLogLevels := map[string]bool{
		"trace": true,
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
		"panic": true,
	}

	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s, must be one of: trace, debug, info, warn, error, fatal, panic", c.LogLevel)
	}

	return nil
}
