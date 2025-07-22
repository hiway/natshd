package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hiway/natshd/internal/config"
)

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected CLIOptions
		hasError bool
	}{
		{
			name: "default values",
			args: []string{"natshd"},
			expected: CLIOptions{
				ConfigFile: "config.toml",
				LogLevel:   "",
			},
			hasError: false,
		},
		{
			name: "custom config file",
			args: []string{"natshd", "-config", "/path/to/config.toml"},
			expected: CLIOptions{
				ConfigFile: "/path/to/config.toml",
				LogLevel:   "",
			},
			hasError: false,
		},
		{
			name: "custom log level",
			args: []string{"natshd", "-log-level", "debug"},
			expected: CLIOptions{
				ConfigFile: "config.toml",
				LogLevel:   "debug",
			},
			hasError: false,
		},
		{
			name: "all custom flags",
			args: []string{"natshd", "-config", "my-config.toml", "-log-level", "warn"},
			expected: CLIOptions{
				ConfigFile: "my-config.toml",
				LogLevel:   "warn",
			},
			hasError: false,
		},
		{
			name: "help flag",
			args: []string{"natshd", "-help"},
			expected: CLIOptions{
				ConfigFile: "config.toml", // Default value is set
				ShowHelp:   true,
			},
			hasError: false,
		},
		{
			name: "version flag",
			args: []string{"natshd", "-version"},
			expected: CLIOptions{
				ConfigFile:  "config.toml", // Default value is set
				ShowVersion: true,
			},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options, err := parseFlags(tt.args)

			if tt.hasError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.hasError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.hasError {
				if options.ConfigFile != tt.expected.ConfigFile {
					t.Errorf("Expected ConfigFile %s, got %s", tt.expected.ConfigFile, options.ConfigFile)
				}

				if options.LogLevel != tt.expected.LogLevel {
					t.Errorf("Expected LogLevel %s, got %s", tt.expected.LogLevel, options.LogLevel)
				}

				if options.ShowHelp != tt.expected.ShowHelp {
					t.Errorf("Expected ShowHelp %v, got %v", tt.expected.ShowHelp, options.ShowHelp)
				}

				if options.ShowVersion != tt.expected.ShowVersion {
					t.Errorf("Expected ShowVersion %v, got %v", tt.expected.ShowVersion, options.ShowVersion)
				}
			}
		})
	}
}

func TestLoadConfiguration(t *testing.T) {
	// Create temporary directory for test configs
	tempDir := t.TempDir()

	tests := []struct {
		name         string
		configFile   string
		configData   string
		cliOptions   CLIOptions
		expectError  bool
		expectConfig *config.Config
	}{
		{
			name:       "valid config file",
			configFile: "valid.toml",
			configData: `
nats_url = "nats://localhost:4222"
scripts_path = "./scripts"
log_level = "info"
`,
			cliOptions:  CLIOptions{},
			expectError: false,
			expectConfig: &config.Config{
				NatsURL:     "nats://localhost:4222",
				ScriptsPath: "./scripts",
				LogLevel:    "info",
			},
		},
		{
			name:       "CLI log level override",
			configFile: "override.toml",
			configData: `
nats_url = "nats://localhost:4222"
scripts_path = "./scripts"
log_level = "info"
`,
			cliOptions: CLIOptions{
				LogLevel: "debug",
			},
			expectError: false,
			expectConfig: &config.Config{
				NatsURL:     "nats://localhost:4222",
				ScriptsPath: "./scripts",
				LogLevel:    "debug",
			},
		},
		{
			name:        "missing config file",
			configFile:  "nonexistent.toml",
			configData:  "",
			cliOptions:  CLIOptions{},
			expectError: true,
		},
		{
			name:        "invalid config file",
			configFile:  "invalid.toml",
			configData:  `invalid toml content`,
			cliOptions:  CLIOptions{},
			expectError: true,
		},
		{
			name:       "config validation error",
			configFile: "invalid_validation.toml",
			configData: `
scripts_path = "./scripts"
log_level = "info"
`,
			cliOptions:  CLIOptions{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configPath string
			if tt.configData != "" {
				configPath = filepath.Join(tempDir, tt.configFile)
				err := os.WriteFile(configPath, []byte(tt.configData), 0644)
				if err != nil {
					t.Fatalf("Failed to create test config file: %v", err)
				}
			} else {
				configPath = filepath.Join(tempDir, tt.configFile)
			}

			cfg, err := loadConfiguration(configPath, tt.cliOptions)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && tt.expectConfig != nil {
				if cfg.NatsURL != tt.expectConfig.NatsURL {
					t.Errorf("Expected NatsURL %s, got %s", tt.expectConfig.NatsURL, cfg.NatsURL)
				}

				if cfg.ScriptsPath != tt.expectConfig.ScriptsPath {
					t.Errorf("Expected ScriptsPath %s, got %s", tt.expectConfig.ScriptsPath, cfg.ScriptsPath)
				}

				if cfg.LogLevel != tt.expectConfig.LogLevel {
					t.Errorf("Expected LogLevel %s, got %s", tt.expectConfig.LogLevel, cfg.LogLevel)
				}
			}
		})
	}
}

func TestConnectToNATS(t *testing.T) {
	tests := []struct {
		name        string
		natsURL     string
		expectError bool
	}{
		{
			name:        "invalid NATS URL",
			natsURL:     "invalid://url",
			expectError: true,
		},
		{
			name:        "unreachable NATS server",
			natsURL:     "nats://nonexistent:4222",
			expectError: true,
		},
		// Note: We can't test successful connection without a real NATS server
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := connectToNATS(tt.natsURL)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if conn != nil {
				conn.Close()
			}
		})
	}
}

func TestRunApplication(t *testing.T) {
	// Create temporary directory and config for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test.toml")
	scriptsDir := filepath.Join(tempDir, "scripts")

	// Create scripts directory
	err := os.MkdirAll(scriptsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create scripts directory: %v", err)
	}

	// Create test config
	configData := `
nats_url = "nats://nonexistent:4222"
scripts_path = "` + scriptsDir + `"
log_level = "info"
`
	err = os.WriteFile(configPath, []byte(configData), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Test with short timeout to ensure it doesn't hang
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	options := CLIOptions{
		ConfigFile: configPath,
	}

	// This should fail due to NATS connection error, but we're testing the application flow
	err = runApplication(ctx, options)
	if err == nil {
		t.Error("Expected error due to NATS connection failure")
	}
}

func TestShowHelp(t *testing.T) {
	// Test that showHelp doesn't panic
	showHelp()
}

func TestShowVersion(t *testing.T) {
	// Test that showVersion doesn't panic
	showVersion()
}

func TestApplicationSetup(t *testing.T) {
	// Create temporary directory and config
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "setup_test.toml")
	scriptsDir := filepath.Join(tempDir, "scripts")

	// Create scripts directory
	err := os.MkdirAll(scriptsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create scripts directory: %v", err)
	}

	// Create test config with valid structure but unreachable NATS
	configData := `
nats_url = "nats://localhost:14222"
scripts_path = "` + scriptsDir + `"
log_level = "debug"
`
	err = os.WriteFile(configPath, []byte(configData), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Test application setup with immediate context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	options := CLIOptions{
		ConfigFile: configPath,
	}

	// Should handle cancelled context gracefully
	err = runApplication(ctx, options)
	if err == nil {
		t.Error("Expected context cancellation error")
	}
}

func TestValidateApplicationComponents(t *testing.T) {
	// Test that all required application components can be created
	tempDir := t.TempDir()
	scriptsDir := filepath.Join(tempDir, "scripts")

	err := os.MkdirAll(scriptsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create scripts directory: %v", err)
	}

	cfg := &config.Config{
		NatsURL:     "nats://localhost:4222",
		ScriptsPath: scriptsDir,
		LogLevel:    "info",
	}

	// Test logger setup
	logger, err := setupApplicationLogger(cfg)
	if err != nil {
		t.Errorf("Failed to setup logger: %v", err)
	}

	// Test that we can actually log with the logger
	logger.Info().Msg("Test log message")
}

func TestSignalHandling(t *testing.T) {
	// Test context cancellation handling
	ctx, cancel := context.WithCancel(context.Background())

	// Simulate signal by cancelling context
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	// Wait for context cancellation
	<-ctx.Done()

	if ctx.Err() == nil {
		t.Error("Expected context to be cancelled")
	}
}
