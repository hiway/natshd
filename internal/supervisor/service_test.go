package supervisor

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/hiway/natshd/internal/logging"
	"github.com/hiway/natshd/internal/service"
	"github.com/nats-io/nats.go"
)

func TestNewManagedService(t *testing.T) {
	logger := logging.SetupLogger("info")
	scriptPath := "/path/to/script.sh"
	natsConn := (*nats.Conn)(nil) // Use nil for testing

	managedService := NewManagedService(scriptPath, natsConn, logger)

	if managedService == nil {
		t.Fatal("Expected ManagedService to be created")
	}

	if managedService.scripts == nil {
		t.Error("Expected scripts map to be initialized")
	}

	if len(managedService.scripts) != 0 {
		t.Errorf("Expected empty scripts map, got %d entries", len(managedService.scripts))
	}

	// Test adding a script
	managedService.AddScript(scriptPath)
	if len(managedService.scripts) != 1 {
		t.Errorf("Expected 1 script after adding, got %d", len(managedService.scripts))
	}

	if _, exists := managedService.scripts[scriptPath]; !exists {
		t.Error("Expected script to be added to scripts map")
	}

	// Logger should be configured properly (just test that it's not nil)
	managedService.logger.Info().Msg("test")
}

func TestManagedService_Initialize(t *testing.T) {
	tests := []struct {
		name           string
		scriptResponse string
		expectError    bool
	}{
		{
			name: "valid service definition",
			scriptResponse: `{
				"name": "TestService",
				"version": "1.0.0",
				"description": "A test service",
				"endpoints": [
					{
						"name": "TestEndpoint",
						"subject": "test.endpoint"
					}
				]
			}`,
			expectError: false,
		},
		{
			name:           "invalid JSON response",
			scriptResponse: `{"name": "TestService"`,
			expectError:    true,
		},
		{
			name: "missing required fields",
			scriptResponse: `{
				"version": "1.0.0",
				"description": "A test service"
			}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logging.SetupLogger("info")
			natsConn := (*nats.Conn)(nil) // Use nil for testing
			managedService := NewManagedService("test.sh", natsConn, logger)

			// Add the script to the service for new service grouping structure
			managedService.AddScript("test.sh")

			// Mock the script runner to return the test response
			// TODO: Update for new service grouping structure - replace the runner
			mockRunner := &MockScriptRunner{
				infoResponse: tt.scriptResponse,
			}
			// Replace the script runner with our mock
			managedService.scripts["test.sh"] = mockRunner

			ctx := context.Background()
			err := managedService.Initialize(ctx)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && managedService.definition.Name == "" {
				t.Error("Expected service definition to be set")
			}
		})
	}
}

func TestManagedService_InitializeWithMetadata(t *testing.T) {
	logger := logging.SetupLogger("info")
	natsConn := (*nats.Conn)(nil) // Use nil for testing
	managedService := NewManagedService("test.sh", natsConn, logger)

	// Add the script to the service
	managedService.AddScript("test.sh")

	// Service definition with endpoint metadata
	scriptResponse := `{
		"name": "GreetingService",
		"version": "1.0.0",
		"description": "A greeting service with metadata",
		"endpoints": [
			{
				"name": "Greet",
				"subject": "greeting.greet",
				"description": "Generates personalized greetings",
				"metadata": {
					"parameters": {
						"name": {
							"type": "string",
							"description": "Name of person to greet",
							"default": "World"
						},
						"greeting": {
							"type": "string", 
							"description": "Greeting message",
							"default": "Hello"
						}
					}
				}
			}
		]
	}`

	mockRunner := &MockScriptRunner{
		infoResponse: scriptResponse,
	}
	managedService.scripts["test.sh"] = mockRunner

	ctx := context.Background()
	err := managedService.Initialize(ctx)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify metadata was parsed correctly
	if len(managedService.definition.Endpoints) != 1 {
		t.Fatalf("Expected 1 endpoint, got %d", len(managedService.definition.Endpoints))
	}

	endpoint := managedService.definition.Endpoints[0]
	if endpoint.Metadata == nil {
		t.Error("Expected endpoint metadata to be parsed")
	}

	if endpoint.Description != "Generates personalized greetings" {
		t.Errorf("Expected description 'Generates personalized greetings', got '%s'", endpoint.Description)
	}

	// Check metadata structure
	if params, ok := endpoint.Metadata["parameters"]; ok {
		paramsMap := params.(map[string]interface{})
		if len(paramsMap) != 2 {
			t.Errorf("Expected 2 parameters in metadata, got %d", len(paramsMap))
		}

		if _, exists := paramsMap["name"]; !exists {
			t.Error("Expected 'name' parameter in metadata")
		}

		if _, exists := paramsMap["greeting"]; !exists {
			t.Error("Expected 'greeting' parameter in metadata")
		}
	} else {
		t.Error("Expected 'parameters' key in metadata")
	}
}

func TestManagedService_Serve(t *testing.T) {
	logger := logging.SetupLogger("info")
	natsConn := (*nats.Conn)(nil) // Use nil for testing
	managedService := NewManagedService("test.sh", natsConn, logger)

	// Mock successful initialization
	managedService.definition = service.ServiceDefinition{
		Name:        "TestService",
		Version:     "1.0.0",
		Description: "A test service",
		Endpoints: []service.Endpoint{
			{Name: "TestEndpoint", Subject: "test.endpoint"},
		},
	}
	// TODO: Update for new service grouping structure
	// managedService.runner = &MockScriptRunner{}
	managedService.initialized = true

	// Test serving in background with a context that will be cancelled immediately
	// Since we're using a nil NATS connection, the micro.AddService will fail
	// but we want to test the lifecycle management, not the NATS integration
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Run Serve - it should fail because natsConn is nil, but that's expected
	err := managedService.Serve(ctx)

	// Should get an error about NATS connection being nil, not a timeout
	if err == nil {
		t.Error("Expected error due to nil NATS connection")
	}
}

// func TestManagedService_HandleRequest(t *testing.T) {
// 	logger := logging.SetupLogger("info")
// 	natsConn := (*nats.Conn)(nil) // Use nil for testing
// 	managedService := NewManagedService("test.sh", natsConn, logger)

// 	// Set up mock script runner with expected response
// 	// TODO: Update for new service grouping structure
// 	mockRunner := &MockScriptRunner{
// 		executeResponse: service.ExecutionResult{
// 			Success:  true,
// 			Stdout:   []byte(`{"result": "success"}`),
// 			Stderr:   []byte{},
// 			ExitCode: 0,
// 		},
// 	}
// 	// TODO: Update for new service grouping structure
// 	// managedService.runner = mockRunner

// 	// Create a mock request
// 	request := &MockRequest{
// 		subject: "test.endpoint",
// 		data:    []byte(`{"input": "test"}`),
// 	}

// 	// Handle the request
// 	managedService.HandleRequest(request)

// 	// Verify script was executed with correct parameters
// 	if mockRunner.lastSubject != "test.endpoint" {
// 		t.Errorf("Expected subject test.endpoint, got %s", mockRunner.lastSubject)
// 	}

// 	if string(mockRunner.lastPayload) != `{"input": "test"}` {
// 		t.Errorf("Expected payload %s, got %s", `{"input": "test"}`, string(mockRunner.lastPayload))
// 	}

// 	// Verify response was sent
// 	if !request.responded {
// 		t.Error("Expected response to be sent")
// 	}
// }

func TestManagedService_HandleRequestWithError(t *testing.T) {
	logger := logging.SetupLogger("info")
	natsConn := (*nats.Conn)(nil) // Use nil for testing
	managedService := NewManagedService("test.sh", natsConn, logger)

	// Set up mock script runner with error response
	// TODO: Update for new service grouping structure
	/*
		mockRunner := &MockScriptRunner{
			executeResponse: service.ExecutionResult{
				Success:  false,
				Stdout:   []byte{},
				Stderr:   []byte("script error"),
				ExitCode: 1,
			},
		}
		// TODO: Update for new service grouping structure
		// managedService.runner = mockRunner
	*/

	// Create a mock request
	request := &MockRequest{
		subject: "test.endpoint",
		data:    []byte(`{"input": "test"}`),
	}

	// Handle the request
	managedService.HandleRequest(request)

	// Verify response was sent with error
	if !request.responded {
		t.Error("Expected response to be sent")
	}

	if request.responseError == nil {
		t.Error("Expected error response")
	}
}

func TestManagedService_String(t *testing.T) {
	logger := logging.SetupLogger("info")
	natsConn := (*nats.Conn)(nil) // Use nil for testing
	managedService := NewManagedService("/path/to/test.sh", natsConn, logger)

	// Add the script to the service (new service grouping structure)
	managedService.AddScript("/path/to/test.sh")

	expected := "ManagedService(/path/to/test.sh)"
	if managedService.String() != expected {
		t.Errorf("Expected %s, got %s", expected, managedService.String())
	}
}

// Mock implementations for testing

type MockScriptRunner struct {
	infoResponse    string
	executeResponse service.ExecutionResult
	executeError    error
	lastSubject     string
	lastPayload     []byte
}

func (m *MockScriptRunner) GetServiceDefinition(ctx context.Context) (service.ServiceDefinition, error) {
	if m.infoResponse == "" {
		return service.ServiceDefinition{}, nil
	}

	var def service.ServiceDefinition
	err := json.Unmarshal([]byte(m.infoResponse), &def)
	if err != nil {
		return service.ServiceDefinition{}, err
	}

	return def, def.Validate()
}

func (m *MockScriptRunner) ExecuteRequest(ctx context.Context, subject string, payload []byte) (service.ExecutionResult, error) {
	m.lastSubject = subject
	m.lastPayload = payload
	return m.executeResponse, m.executeError
}

type MockRequest struct {
	subject       string
	data          []byte
	responded     bool
	responseData  []byte
	responseError error
}

func (m *MockRequest) Subject() string {
	return m.subject
}

func (m *MockRequest) Data() []byte {
	return m.data
}

func (m *MockRequest) Headers() map[string][]string {
	return nil
}

func (m *MockRequest) Respond(data []byte) error {
	m.responded = true
	m.responseData = data
	return nil
}

func (m *MockRequest) RespondError(err error) error {
	m.responded = true
	m.responseError = err
	return nil
}
