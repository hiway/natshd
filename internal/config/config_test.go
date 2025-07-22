package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name           string
		configContent  string
		expectedConfig Config
		expectError    bool
	}{
		{
			name: "valid TOML config",
			configContent: `nats_url = "nats://127.0.0.1:4222"
scripts_path = "./scripts"
log_level = "info"`,
			expectedConfig: Config{
				NatsURL:     "nats://127.0.0.1:4222",
				ScriptsPath: "./scripts",
				LogLevel:    "info",
			},
			expectError: false,
		},
		{
			name: "minimal config with defaults",
			configContent: `nats_url = "nats://localhost:4222"
scripts_path = "/tmp/scripts"`,
			expectedConfig: Config{
				NatsURL:     "nats://localhost:4222",
				ScriptsPath: "/tmp/scripts",
				LogLevel:    "info", // Should default to info
			},
			expectError: false,
		},
		{
			name: "missing required nats_url",
			configContent: `scripts_path = "./scripts"
log_level = "debug"`,
			expectedConfig: Config{},
			expectError:    true,
		},
		{
			name: "missing required scripts_path",
			configContent: `nats_url = "nats://127.0.0.1:4222"
log_level = "debug"`,
			expectedConfig: Config{},
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.toml")

			err := os.WriteFile(configPath, []byte(tt.configContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test config file: %v", err)
			}

			// Test LoadConfig
			config, err := LoadConfig(configPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if config.NatsURL != tt.expectedConfig.NatsURL {
				t.Errorf("Expected NatsURL %s, got %s", tt.expectedConfig.NatsURL, config.NatsURL)
			}

			if config.ScriptsPath != tt.expectedConfig.ScriptsPath {
				t.Errorf("Expected ScriptsPath %s, got %s", tt.expectedConfig.ScriptsPath, config.ScriptsPath)
			}

			if config.LogLevel != tt.expectedConfig.LogLevel {
				t.Errorf("Expected LogLevel %s, got %s", tt.expectedConfig.LogLevel, config.LogLevel)
			}
		})
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := LoadConfig("nonexistent.toml")
	if err == nil {
		t.Error("Expected error for nonexistent config file")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.NatsURL != "nats://127.0.0.1:4222" {
		t.Errorf("Expected default NatsURL nats://127.0.0.1:4222, got %s", config.NatsURL)
	}

	if config.ScriptsPath != "./scripts" {
		t.Errorf("Expected default ScriptsPath ./scripts, got %s", config.ScriptsPath)
	}

	if config.LogLevel != "info" {
		t.Errorf("Expected default LogLevel info, got %s", config.LogLevel)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "valid config",
			config: Config{
				NatsURL:     "nats://127.0.0.1:4222",
				ScriptsPath: "./scripts",
				LogLevel:    "info",
			},
			expectError: false,
		},
		{
			name: "empty nats_url",
			config: Config{
				NatsURL:     "",
				ScriptsPath: "./scripts",
				LogLevel:    "info",
			},
			expectError: true,
		},
		{
			name: "empty scripts_path",
			config: Config{
				NatsURL:     "nats://127.0.0.1:4222",
				ScriptsPath: "",
				LogLevel:    "info",
			},
			expectError: true,
		},
		{
			name: "invalid log level",
			config: Config{
				NatsURL:     "nats://127.0.0.1:4222",
				ScriptsPath: "./scripts",
				LogLevel:    "invalid",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}
