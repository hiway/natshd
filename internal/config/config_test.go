package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.NatsURL != "nats://127.0.0.1:4222" {
		t.Errorf("Expected default NatsURL to be 'nats://127.0.0.1:4222', got '%s'", config.NatsURL)
	}

	if config.ScriptsPath != "./scripts" {
		t.Errorf("Expected default ScriptsPath to be './scripts', got '%s'", config.ScriptsPath)
	}

	if config.LogLevel != "info" {
		t.Errorf("Expected default LogLevel to be 'info', got '%s'", config.LogLevel)
	}

	if config.Hostname != "auto" {
		t.Errorf("Expected default Hostname to be 'auto', got '%s'", config.Hostname)
	}
}

func TestResolveHostname_Auto(t *testing.T) {
	config := Config{
		Hostname: "auto",
	}

	resolved, err := config.ResolveHostname()
	if err != nil {
		t.Fatalf("Expected no error when resolving hostname 'auto', got: %v", err)
	}

	if resolved == "" {
		t.Error("Expected resolved hostname to be non-empty when using 'auto'")
	}

	if resolved == "auto" {
		t.Error("Expected resolved hostname to be actual hostname, not 'auto'")
	}

	// Should match system hostname
	expectedHostname, err := os.Hostname()
	if err != nil {
		t.Fatalf("Failed to get system hostname: %v", err)
	}

	if resolved != expectedHostname {
		t.Errorf("Expected resolved hostname '%s' to match system hostname '%s'", resolved, expectedHostname)
	}
}

func TestResolveHostname_Explicit(t *testing.T) {
	explicitHostname := "test-server-01"
	config := Config{
		Hostname: explicitHostname,
	}

	resolved, err := config.ResolveHostname()
	if err != nil {
		t.Fatalf("Expected no error when resolving explicit hostname, got: %v", err)
	}

	if resolved != explicitHostname {
		t.Errorf("Expected resolved hostname '%s' to match explicit hostname '%s'", resolved, explicitHostname)
	}
}

func TestResolveHostname_Empty(t *testing.T) {
	config := Config{
		Hostname: "",
	}

	resolved, err := config.ResolveHostname()
	if err != nil {
		t.Fatalf("Expected no error when resolving empty hostname, got: %v", err)
	}

	// Should default to system hostname
	expectedHostname, err := os.Hostname()
	if err != nil {
		t.Fatalf("Failed to get system hostname: %v", err)
	}

	if resolved != expectedHostname {
		t.Errorf("Expected resolved hostname '%s' to match system hostname '%s'", resolved, expectedHostname)
	}
}

func TestPrefixSubject(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		subject  string
		expected string
	}{
		{
			name:     "simple subject",
			hostname: "web01",
			subject:  "system.facts",
			expected: "web01.system.facts",
		},
		{
			name:     "multi-part subject",
			hostname: "db-server",
			subject:  "pkg.ensure.nginx",
			expected: "db-server.pkg.ensure.nginx",
		},
		{
			name:     "hostname with dots",
			hostname: "web01.example.com",
			subject:  "health.check",
			expected: "web01.example.com.health.check",
		},
		{
			name:     "empty subject",
			hostname: "server",
			subject:  "",
			expected: "server.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{Hostname: tt.hostname}
			result := config.PrefixSubject(tt.subject)

			if result != tt.expected {
				t.Errorf("Expected PrefixSubject('%s') to return '%s', got '%s'", tt.subject, tt.expected, result)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name           string
		configContent  string
		expectedConfig Config
		expectError    bool
	}{
		{
			name: "valid TOML config with hostname",
			configContent: `nats_url = "nats://127.0.0.1:4222"
scripts_path = "./scripts"
log_level = "info"
hostname = "test-server"`,
			expectedConfig: Config{
				NatsURL:     "nats://127.0.0.1:4222",
				ScriptsPath: "./scripts",
				LogLevel:    "info",
				Hostname:    "test-server",
			},
			expectError: false,
		},
		{
			name: "config without hostname (should default to auto)",
			configContent: `nats_url = "nats://127.0.0.1:4222"
scripts_path = "./scripts"
log_level = "info"`,
			expectedConfig: Config{
				NatsURL:     "nats://127.0.0.1:4222",
				ScriptsPath: "./scripts",
				LogLevel:    "info",
				Hostname:    "auto",
			},
			expectError: false,
		},
		{
			name: "config with hostname auto",
			configContent: `nats_url = "nats://127.0.0.1:4222"
scripts_path = "./scripts"
log_level = "info"
hostname = "auto"`,
			expectedConfig: Config{
				NatsURL:     "nats://127.0.0.1:4222",
				ScriptsPath: "./scripts",
				LogLevel:    "info",
				Hostname:    "auto",
			},
			expectError: false,
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

			if config.Hostname != tt.expectedConfig.Hostname {
				t.Errorf("Expected Hostname %s, got %s", tt.expectedConfig.Hostname, config.Hostname)
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
				Hostname:    "test-server",
			},
			expectError: false,
		},
		{
			name: "valid config with auto hostname",
			config: Config{
				NatsURL:     "nats://127.0.0.1:4222",
				ScriptsPath: "./scripts",
				LogLevel:    "info",
				Hostname:    "auto",
			},
			expectError: false,
		},
		{
			name: "empty nats_url",
			config: Config{
				NatsURL:     "",
				ScriptsPath: "./scripts",
				LogLevel:    "info",
				Hostname:    "server",
			},
			expectError: true,
		},
		{
			name: "empty scripts_path",
			config: Config{
				NatsURL:     "nats://127.0.0.1:4222",
				ScriptsPath: "",
				LogLevel:    "info",
				Hostname:    "server",
			},
			expectError: true,
		},
		{
			name: "invalid log level",
			config: Config{
				NatsURL:     "nats://127.0.0.1:4222",
				ScriptsPath: "./scripts",
				LogLevel:    "invalid",
				Hostname:    "server",
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
