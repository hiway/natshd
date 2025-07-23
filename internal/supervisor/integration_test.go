package supervisor

import (
	"context"
	"testing"

	"github.com/hiway/natshd/internal/logging"
	"github.com/hiway/natshd/internal/service"
	"github.com/nats-io/nats.go"
)

func TestManagedService_IntegrationWithGreetingScript(t *testing.T) {
	logger := logging.SetupLogger("info")
	natsConn := (*nats.Conn)(nil) // Use nil for testing - we're testing parsing, not NATS integration

	// Use the actual greeting script
	scriptPath := "../../scripts/greeting.sh"
	managedService := NewManagedService(scriptPath, natsConn, logger)
	managedService.AddScript(scriptPath)

	ctx := context.Background()
	err := managedService.Initialize(ctx)

	if err != nil {
		t.Errorf("Unexpected error during initialization: %v", err)
		return
	}

	// Verify service definition
	if managedService.definition.Name != "GreetingService" {
		t.Errorf("Expected service name 'GreetingService', got '%s'", managedService.definition.Name)
	}

	if managedService.definition.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", managedService.definition.Version)
	}

	if len(managedService.definition.Endpoints) == 0 {
		t.Fatal("Expected endpoints to be defined")
	}

	// Verify endpoints with metadata
	expectedEndpoints := map[string]bool{
		"Greet":    false,
		"Farewell": false,
	}

	for _, endpoint := range managedService.definition.Endpoints {
		if _, exists := expectedEndpoints[endpoint.Name]; !exists {
			t.Errorf("Unexpected endpoint: %s", endpoint.Name)
			continue
		}
		expectedEndpoints[endpoint.Name] = true

		// Verify metadata is present
		if endpoint.Metadata == nil {
			t.Errorf("Expected metadata for endpoint %s", endpoint.Name)
			continue
		}

		// Verify description is present
		if endpoint.Description == "" {
			t.Errorf("Expected description for endpoint %s", endpoint.Name)
		}

		// Verify parameters metadata structure
		if params, ok := endpoint.Metadata["parameters"]; ok {
			paramsMap, isMap := params.(map[string]interface{})
			if !isMap {
				t.Errorf("Expected parameters to be a map for endpoint %s", endpoint.Name)
				continue
			}

			// Verify common parameter 'name' exists in both endpoints
			if nameParam, exists := paramsMap["name"]; exists {
				nameParamMap, isNameMap := nameParam.(map[string]interface{})
				if !isNameMap {
					t.Errorf("Expected name parameter to be a map for endpoint %s", endpoint.Name)
					continue
				}

				// Verify basic parameter structure
				if paramType, hasType := nameParamMap["type"]; !hasType || paramType != "string" {
					t.Errorf("Expected name parameter to have type 'string' for endpoint %s", endpoint.Name)
				}

				if description, hasDesc := nameParamMap["description"]; !hasDesc || description == "" {
					t.Errorf("Expected name parameter to have description for endpoint %s", endpoint.Name)
				}
			} else {
				t.Errorf("Expected 'name' parameter in metadata for endpoint %s", endpoint.Name)
			}

			// Verify endpoint-specific parameters
			if endpoint.Name == "Greet" {
				if greetingParam, exists := paramsMap["greeting"]; exists {
					greetingParamMap, isGreetingMap := greetingParam.(map[string]interface{})
					if !isGreetingMap {
						t.Errorf("Expected greeting parameter to be a map for Greet endpoint")
						continue
					}

					if paramType, hasType := greetingParamMap["type"]; !hasType || paramType != "string" {
						t.Errorf("Expected greeting parameter to have type 'string' for Greet endpoint")
					}
				} else {
					t.Error("Expected 'greeting' parameter in metadata for Greet endpoint")
				}
			}
		} else {
			t.Errorf("Expected 'parameters' in metadata for endpoint %s", endpoint.Name)
		}
	}

	// Verify all expected endpoints were found
	for name, found := range expectedEndpoints {
		if !found {
			t.Errorf("Expected endpoint '%s' not found", name)
		}
	}
}

func TestManagedService_IntegrationWithRealScriptRunner(t *testing.T) {
	// Use the actual greeting script
	scriptPath := "../../scripts/greeting.sh"
	// Create a real script runner to test the full integration
	runner := service.NewScriptRunner(scriptPath)

	ctx := context.Background()

	// Test getting service definition
	definition, err := runner.GetServiceDefinition(ctx)
	if err != nil {
		t.Errorf("Failed to get service definition: %v", err)
		return
	}

	// Verify the definition matches expected structure
	if definition.Name != "GreetingService" {
		t.Errorf("Expected service name 'GreetingService', got '%s'", definition.Name)
	}

	if len(definition.Endpoints) == 0 {
		t.Fatal("Expected at least one endpoint")
	}

	// Find the Greet endpoint and verify its metadata
	var greetEndpoint *service.Endpoint
	for _, endpoint := range definition.Endpoints {
		if endpoint.Name == "Greet" {
			greetEndpoint = &endpoint
			break
		}
	}

	if greetEndpoint == nil {
		t.Fatal("Expected to find 'Greet' endpoint")
	}

	if greetEndpoint.Subject != "greeting.greet" {
		t.Errorf("Expected subject 'greeting.greet', got '%s'", greetEndpoint.Subject)
	}

	if greetEndpoint.Description == "" {
		t.Error("Expected endpoint description to be non-empty")
	}

	if greetEndpoint.Metadata == nil {
		t.Fatal("Expected endpoint metadata to be present")
	}

	// Test execution
	result, err := runner.ExecuteRequest(ctx, "greeting.greet", []byte(`{"name": "TestUser", "greeting": "Hi"}`))
	if err != nil {
		t.Errorf("Failed to execute request: %v", err)
		return
	}

	if !result.Success {
		t.Errorf("Expected successful execution, got exit code %d, stderr: %s", result.ExitCode, string(result.Stderr))
		return
	}

	// Verify response contains expected content
	responseStr := string(result.Stdout)
	if responseStr == "" {
		t.Error("Expected non-empty response")
	}

	// Should contain the greeting message
	if !contains(responseStr, "Hi, TestUser!") {
		t.Errorf("Expected response to contain 'Hi, TestUser!', got: %s", responseStr)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
