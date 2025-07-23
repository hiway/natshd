package supervisor

import (
	"context"
	"testing"

	"github.com/hiway/natshd/internal/config"
	"github.com/hiway/natshd/internal/service"
	"github.com/rs/zerolog"
)

func TestManagedService_HostnamePrefixing(t *testing.T) {
	// Test config with explicit hostname
	testConfig := config.Config{
		Hostname: "test-server-01",
	}

	logger := zerolog.Nop()

	// Create managed service (with nil NATS connection for testing)
	managedService := NewManagedService("test.sh", nil, logger, testConfig)

	// Add a mock script runner
	serviceDefJSON := `{
		"name": "TestService",
		"version": "1.0.0",
		"endpoints": [
			{
				"name": "TestEndpoint",
				"subject": "test.endpoint"
			}
		]
	}`

	mockRunner := &MockScriptRunner{
		infoResponse: serviceDefJSON,
		executeResponse: service.ExecutionResult{
			Success:  true,
			Stdout:   []byte(`{"message": "test response"}`),
			ExitCode: 0,
		},
	}

	managedService.scripts["test.sh"] = mockRunner

	// Initialize the service
	ctx := context.Background()
	err := managedService.Initialize(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	// Verify that the service definition has hostname-prefixed subjects
	if len(managedService.definition.Endpoints) != 1 {
		t.Fatalf("Expected 1 endpoint, got %d", len(managedService.definition.Endpoints))
	}

	endpoint := managedService.definition.Endpoints[0]
	expectedSubject := "test-server-01.test.endpoint"
	if endpoint.Subject != expectedSubject {
		t.Errorf("Expected endpoint subject to be '%s', got '%s'", expectedSubject, endpoint.Subject)
	}
}

func TestManagedService_HostnamePrefixing_Auto(t *testing.T) {
	// Test config with auto hostname
	testConfig := config.Config{
		Hostname: "auto",
	}

	logger := zerolog.Nop()

	// Create managed service
	managedService := NewManagedService("test.sh", nil, logger, testConfig)

	// Add a mock script runner
	serviceDefJSON := `{
		"name": "TestService",
		"version": "1.0.0",
		"endpoints": [
			{
				"name": "TestEndpoint",
				"subject": "test.endpoint"
			}
		]
	}`

	mockRunner := &MockScriptRunner{
		infoResponse: serviceDefJSON,
		executeResponse: service.ExecutionResult{
			Success:  true,
			Stdout:   []byte(`{"message": "test response"}`),
			ExitCode: 0,
		},
	}

	managedService.scripts["test.sh"] = mockRunner

	// Initialize the service
	ctx := context.Background()
	err := managedService.Initialize(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	// Verify that the service definition has hostname-prefixed subjects
	if len(managedService.definition.Endpoints) != 1 {
		t.Fatalf("Expected 1 endpoint, got %d", len(managedService.definition.Endpoints))
	}

	endpoint := managedService.definition.Endpoints[0]

	// Should start with actual hostname when auto is used
	resolvedHostname, err := testConfig.ResolveHostname()
	if err != nil {
		t.Fatalf("Failed to resolve hostname: %v", err)
	}

	expectedSubject := resolvedHostname + ".test.endpoint"
	if endpoint.Subject != expectedSubject {
		t.Errorf("Expected endpoint subject to be '%s', got '%s'", expectedSubject, endpoint.Subject)
	}

	// Verify it's not "auto.test.endpoint"
	if endpoint.Subject == "auto.test.endpoint" {
		t.Error("Expected hostname to be resolved from 'auto', not kept as 'auto'")
	}
}

func TestServiceManager_WithHostnameConfig(t *testing.T) {
	testConfig := config.Config{
		NatsURL:     "nats://localhost:4222",
		ScriptsPath: "./test-scripts",
		LogLevel:    "info",
		Hostname:    "test-manager-host",
	}

	logger := zerolog.Nop()

	// Create manager with config
	manager := NewManager("./test-scripts", nil, logger, testConfig)

	// Verify config is stored
	if manager.config.Hostname != "test-manager-host" {
		t.Errorf("Expected manager hostname to be 'test-manager-host', got '%s'", manager.config.Hostname)
	}
}

func TestManagedService_StripHostnamePrefix(t *testing.T) {
	testConfig := config.Config{
		Hostname: "test-server",
	}

	logger := zerolog.Nop()
	managedService := NewManagedService("test.sh", nil, logger, testConfig)

	tests := []struct {
		name            string
		prefixedSubject string
		expectedResult  string
	}{
		{
			name:            "valid prefix",
			prefixedSubject: "test-server.system.facts",
			expectedResult:  "system.facts",
		},
		{
			name:            "no prefix",
			prefixedSubject: "system.facts",
			expectedResult:  "system.facts",
		},
		{
			name:            "partial match",
			prefixedSubject: "test-serve.system.facts", // Missing 'r'
			expectedResult:  "test-serve.system.facts",
		},
		{
			name:            "empty subject",
			prefixedSubject: "",
			expectedResult:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := managedService.stripHostnamePrefix(tt.prefixedSubject)
			if result != tt.expectedResult {
				t.Errorf("Expected stripHostnamePrefix('%s') to return '%s', got '%s'",
					tt.prefixedSubject, tt.expectedResult, result)
			}
		})
	}
}
