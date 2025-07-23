package supervisor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hiway/natshd/internal/logging"
	"github.com/nats-io/nats.go"
)

func TestNewManager(t *testing.T) {
	logger := logging.SetupLogger("info")
	natsConn := (*nats.Conn)(nil) // Use nil for testing
	scriptsPath := "/path/to/scripts"

	manager := NewManager(scriptsPath, natsConn, logger)

	if manager == nil {
		t.Fatal("Expected Manager to be created")
	}

	if manager.scriptsPath != scriptsPath {
		t.Errorf("Expected scripts path %s, got %s", scriptsPath, manager.scriptsPath)
	}

	if manager.natsConn != natsConn {
		t.Error("Expected NATS connection to be set")
	}

	// Check that supervisor is created
	if manager.supervisor == nil {
		t.Error("Expected supervisor to be created")
	}

	// Check that debounce tracker is initialized
	if manager.debounceTracker == nil {
		t.Error("Expected debounce tracker to be initialized")
	}
}

func TestManager_ServiceGrouping(t *testing.T) {
	tempDir := t.TempDir()
	logger := logging.SetupLogger("info")
	natsConn := (*nats.Conn)(nil) // Use nil for testing

	manager := NewManager(tempDir, natsConn, logger)

	// Create two scripts with the same service name but different endpoints
	script1Content := `#!/bin/bash
if [[ "$1" == "info" ]]; then
  cat <<EOF
{
  "name": "SystemService",
  "version": "1.0.0",
  "description": "System management service",
  "endpoints": [
    {
      "name": "GetFacts",
      "subject": "system.facts"
    }
  ]
}
EOF
  exit 0
fi
echo "facts response"
`

	script2Content := `#!/bin/bash
if [[ "$1" == "info" ]]; then
  cat <<EOF
{
  "name": "SystemService",
  "version": "1.0.0", 
  "description": "System management service",
  "endpoints": [
    {
      "name": "GetHardware",
      "subject": "system.hardware"
    }
  ]
}
EOF
  exit 0
fi
echo "hardware response"
`

	// Write the scripts
	script1Path := filepath.Join(tempDir, "system-facts.sh")
	script2Path := filepath.Join(tempDir, "system-hardware.sh")

	err := os.WriteFile(script1Path, []byte(script1Content), 0755)
	if err != nil {
		t.Fatalf("Failed to create script1: %v", err)
	}

	err = os.WriteFile(script2Path, []byte(script2Content), 0755)
	if err != nil {
		t.Fatalf("Failed to create script2: %v", err)
	}

	// Add both services
	err = manager.AddService(script1Path)
	if err != nil {
		t.Fatalf("Failed to add script1: %v", err)
	}

	err = manager.AddService(script2Path)
	if err != nil {
		t.Fatalf("Failed to add script2: %v", err)
	}

	// Check that only one service is registered (grouped by name)
	manager.mutex.RLock()
	serviceCount := len(manager.services)
	manager.mutex.RUnlock()

	if serviceCount != 1 {
		t.Errorf("Expected 1 grouped service, got %d services", serviceCount)
	}

	// Check that the service has both endpoints
	manager.mutex.RLock()
	var service *ManagedService
	for _, svc := range manager.services {
		service = svc
		break
	}
	manager.mutex.RUnlock()

	if service == nil {
		t.Fatal("Expected to find one service")
	}

	if len(service.definition.Endpoints) != 2 {
		t.Errorf("Expected 2 endpoints in grouped service, got %d", len(service.definition.Endpoints))
	}

	// Check that subjects are correct
	expectedSubjects := map[string]bool{
		"system.facts":    false,
		"system.hardware": false,
	}

	for _, endpoint := range service.definition.Endpoints {
		if _, exists := expectedSubjects[endpoint.Subject]; exists {
			expectedSubjects[endpoint.Subject] = true
		}
	}

	for subject, found := range expectedSubjects {
		if !found {
			t.Errorf("Expected subject %s not found in grouped service", subject)
		}
	}
}

func TestManager_Start(t *testing.T) {
	// Create temporary directory for test scripts
	tempDir := t.TempDir()
	logger := logging.SetupLogger("info")
	natsConn := (*nats.Conn)(nil) // Use nil for testing

	manager := NewManager(tempDir, natsConn, logger)

	// Create a test script
	scriptPath := filepath.Join(tempDir, "test.sh")
	scriptContent := `#!/bin/bash
if [[ "$1" == "info" ]]; then
  cat <<EOF
{
  "name": "TestService",
  "version": "1.0.0",
  "description": "A test service",
  "endpoints": [
    {
      "name": "TestEndpoint",
      "subject": "test.endpoint"
    }
  ]
}
EOF
  exit 0
fi
echo "test response"
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	// Start the manager
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = manager.Start(ctx)

	// Should timeout since we're using nil NATS connection, but Start should handle it gracefully
	if err == nil {
		t.Error("Expected error due to nil NATS connection or context timeout")
	}

	// Check that services were discovered
	if len(manager.services) == 0 {
		t.Error("Expected at least one service to be discovered")
	}
}

func TestManager_RestartServiceWithGracefulShutdown(t *testing.T) {
	tempDir := t.TempDir()
	logger := logging.SetupLogger("info")
	natsConn := (*nats.Conn)(nil) // Use nil for testing

	manager := NewManager(tempDir, natsConn, logger)

	scriptPath := filepath.Join(tempDir, "test.sh")

	// Create and add a service first
	scriptContent := `#!/bin/bash
if [[ "$1" == "info" ]]; then
  cat <<EOF
{
  "name": "TestService",
  "version": "1.0.0",
  "description": "A test service",
  "endpoints": [
    {
      "name": "TestEndpoint",
      "subject": "test.endpoint"
    }
  ]
}
EOF
  exit 0
fi
echo "test response"
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	err = manager.AddService(scriptPath)
	if err != nil {
		t.Fatalf("AddService failed: %v", err)
	}

	// Get original service (services are now tracked by name, not path)
	originalService := manager.services["TestService"]

	// Restart the service with graceful shutdown
	err = manager.RestartServiceGracefully(scriptPath)
	if err != nil {
		t.Fatalf("RestartServiceGracefully failed: %v", err)
	}

	// Check that the same service instance was reused (new behavior with service grouping)
	newService := manager.services["TestService"]
	if originalService != newService {
		t.Error("Expected service instance to be reused after restart (service grouping behavior)")
	}

	// Verify service is still properly initialized
	if newService.definition.Name != "TestService" {
		t.Error("Expected service to be properly reinitialized after restart")
	}
}

func TestManager_FileEventDebouncing(t *testing.T) {
	tempDir := t.TempDir()
	logger := logging.SetupLogger("info")
	natsConn := (*nats.Conn)(nil) // Use nil for testing

	manager := NewManager(tempDir, natsConn, logger)

	scriptPath := filepath.Join(tempDir, "test.sh")
	scriptContent := `#!/bin/bash
if [[ "$1" == "info" ]]; then
  cat <<EOF
{
  "name": "TestService",
  "version": "1.0.0",
  "description": "A test service",
  "endpoints": [
    {
      "name": "TestEndpoint",
      "subject": "test.endpoint"
    }
  ]
}
EOF
  exit 0
fi
echo "test response"
`

	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	err = manager.AddService(scriptPath)
	if err != nil {
		t.Fatalf("AddService failed: %v", err)
	}

	// Test that multiple rapid events within debounce period create only one tracker
	initialTrackerCount := len(manager.debounceTracker)

	// Simulate rapid file events
	for i := 0; i < 5; i++ {
		manager.handleFileEventDebounced(scriptPath, "write")
		time.Sleep(10 * time.Millisecond) // Less than debounce period
	}

	// Check that only one tracker was created
	trackerCount := len(manager.debounceTracker)
	if trackerCount != initialTrackerCount+1 {
		t.Errorf("Expected %d trackers, got %d", initialTrackerCount+1, trackerCount)
	}

	// Wait for debounce period to complete
	time.Sleep(550 * time.Millisecond)

	// Tracker should be cleaned up after execution
	finalTrackerCount := len(manager.debounceTracker)
	if finalTrackerCount != initialTrackerCount {
		t.Errorf("Expected %d trackers after cleanup, got %d", initialTrackerCount, finalTrackerCount)
	}
}

func TestManager_DiscoverServices(t *testing.T) {
	// Create temporary directory for test scripts
	tempDir := t.TempDir()
	logger := logging.SetupLogger("info")
	natsConn := (*nats.Conn)(nil) // Use nil for testing

	manager := NewManager(tempDir, natsConn, logger)

	// Create test scripts
	scripts := []struct {
		name    string
		content string
		valid   bool
	}{
		{
			name: "valid.sh",
			content: `#!/bin/bash
if [[ "$1" == "info" ]]; then
  cat <<EOF
{
  "name": "ValidService",
  "version": "1.0.0",
  "description": "A valid service",
  "endpoints": [
    {
      "name": "ValidEndpoint",
      "subject": "valid.endpoint"
    }
  ]
}
EOF
  exit 0
fi
echo "valid response"
`,
			valid: true,
		},
		{
			name: "invalid.sh",
			content: `#!/bin/bash
echo "not a service"
`,
			valid: false,
		},
		{
			name: "not_executable.sh",
			content: `#!/bin/bash
echo "not executable"
`,
			valid: false,
		},
	}

	for _, script := range scripts {
		scriptPath := filepath.Join(tempDir, script.name)
		mode := os.FileMode(0755)
		if script.name == "not_executable.sh" {
			mode = 0644 // Not executable
		}
		err := os.WriteFile(scriptPath, []byte(script.content), mode)
		if err != nil {
			t.Fatalf("Failed to create test script %s: %v", script.name, err)
		}
	}

	// Discover services
	err := manager.DiscoverServices()
	if err != nil {
		t.Fatalf("DiscoverServices failed: %v", err)
	}

	// Should only discover valid executable scripts
	expectedServices := 1
	if len(manager.services) != expectedServices {
		t.Errorf("Expected %d services, got %d", expectedServices, len(manager.services))
	}

	// Check that the valid service was discovered (services are now tracked by name)
	if _, exists := manager.services["ValidService"]; !exists {
		t.Error("Expected valid.sh to be discovered as a service")
	}

	// Check that script-to-service mapping was created
	validScriptPath := filepath.Join(tempDir, "valid.sh")
	if serviceName, exists := manager.scriptToService[validScriptPath]; !exists || serviceName != "ValidService" {
		t.Error("Expected script-to-service mapping to be created for valid.sh")
	}
}

func TestManager_AddService(t *testing.T) {
	tempDir := t.TempDir()
	logger := logging.SetupLogger("info")
	natsConn := (*nats.Conn)(nil) // Use nil for testing

	manager := NewManager(tempDir, natsConn, logger)

	scriptPath := filepath.Join(tempDir, "test.sh")
	scriptContent := `#!/bin/bash
if [[ "$1" == "info" ]]; then
  cat <<EOF
{
  "name": "TestService",
  "version": "1.0.0",
  "description": "A test service",
  "endpoints": [
    {
      "name": "TestEndpoint",
      "subject": "test.endpoint"
    }
  ]
}
EOF
  exit 0
fi
echo "test response"
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	err = manager.AddService(scriptPath)
	if err != nil {
		t.Fatalf("AddService failed: %v", err)
	}

	// Check that service was added (services are now tracked by name, not path)
	if _, exists := manager.services["TestService"]; !exists {
		t.Error("Expected service to be added to services map")
	}

	// Check that service token was stored (tokens are now tracked by service name)
	if _, exists := manager.serviceTokens["TestService"]; !exists {
		t.Error("Expected service token to be stored")
	}

	// Check that script-to-service mapping was created
	if serviceName, exists := manager.scriptToService[scriptPath]; !exists || serviceName != "TestService" {
		t.Error("Expected script-to-service mapping to be created")
	}
}

func TestManager_RemoveService(t *testing.T) {
	tempDir := t.TempDir()
	logger := logging.SetupLogger("info")
	natsConn := (*nats.Conn)(nil) // Use nil for testing

	manager := NewManager(tempDir, natsConn, logger)

	scriptPath := filepath.Join(tempDir, "test.sh")

	// Create and add a service first
	scriptContent := `#!/bin/bash
if [[ "$1" == "info" ]]; then
  cat <<EOF
{
  "name": "TestService",
  "version": "1.0.0",
  "description": "A test service",
  "endpoints": [
    {
      "name": "TestEndpoint",
      "subject": "test.endpoint"
    }
  ]
}
EOF
  exit 0
fi
echo "test response"
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	err = manager.AddService(scriptPath)
	if err != nil {
		t.Fatalf("AddService failed: %v", err)
	}

	// Verify service was added (services are now tracked by name, not path)
	if _, exists := manager.services["TestService"]; !exists {
		t.Fatal("Service was not added")
	}

	// Remove the service
	err = manager.RemoveService(scriptPath)
	if err != nil {
		t.Fatalf("RemoveService failed: %v", err)
	}

	// Check that service was removed (since this was the only script, the service should be removed)
	if _, exists := manager.services["TestService"]; exists {
		t.Error("Expected service to be removed from services map")
	}

	// Check that script-to-service mapping was removed
	if _, exists := manager.scriptToService[scriptPath]; exists {
		t.Error("Expected script-to-service mapping to be removed")
	}
}

func TestManager_RestartService(t *testing.T) {
	tempDir := t.TempDir()
	logger := logging.SetupLogger("info")
	natsConn := (*nats.Conn)(nil) // Use nil for testing

	manager := NewManager(tempDir, natsConn, logger)

	scriptPath := filepath.Join(tempDir, "test.sh")

	// Create and add a service first
	scriptContent := `#!/bin/bash
if [[ "$1" == "info" ]]; then
  cat <<EOF
{
  "name": "TestService",
  "version": "1.0.0",
  "description": "A test service",
  "endpoints": [
    {
      "name": "TestEndpoint",
      "subject": "test.endpoint"
    }
  ]
}
EOF
  exit 0
fi
echo "test response"
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	err = manager.AddService(scriptPath)
	if err != nil {
		t.Fatalf("AddService failed: %v", err)
	}

	// Get original service (services are now tracked by name, not path)
	originalService := manager.services["TestService"]

	// Restart the service
	err = manager.RestartService(scriptPath)
	if err != nil {
		t.Fatalf("RestartService failed: %v", err)
	}

	// Check that the same service instance was reused (new behavior with service grouping)
	newService := manager.services["TestService"]
	if originalService != newService {
		t.Error("Expected service instance to be reused after restart (service grouping behavior)")
	}

	// Verify service is still properly initialized
	if newService.definition.Name != "TestService" {
		t.Error("Expected service to be properly reinitialized after restart")
	}
}

func TestManager_HandleFileEvent(t *testing.T) {
	tempDir := t.TempDir()
	logger := logging.SetupLogger("info")
	natsConn := (*nats.Conn)(nil) // Use nil for testing

	manager := NewManager(tempDir, natsConn, logger)

	scriptPath := filepath.Join(tempDir, "test.sh")
	scriptContent := `#!/bin/bash
if [[ "$1" == "info" ]]; then
  cat <<EOF
{
  "name": "TestService",
  "version": "1.0.0",
  "description": "A test service",
  "endpoints": [
    {
      "name": "TestEndpoint",
      "subject": "test.endpoint"
    }
  ]
}
EOF
  exit 0
fi
echo "test response"
`

	tests := []struct {
		name      string
		eventType string
		setup     func() error
		verify    func() bool
	}{
		{
			name:      "create event",
			eventType: "create",
			setup: func() error {
				return os.WriteFile(scriptPath, []byte(scriptContent), 0755)
			},
			verify: func() bool {
				_, exists := manager.services["TestService"]
				return exists
			},
		},
		{
			name:      "modify event",
			eventType: "write",
			setup: func() error {
				// First create the service
				err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
				if err != nil {
					return err
				}
				err = manager.AddService(scriptPath)
				if err != nil {
					return err
				}
				// Then modify it
				return os.WriteFile(scriptPath, []byte(scriptContent+"# modified"), 0755)
			},
			verify: func() bool {
				_, exists := manager.services["TestService"]
				return exists
			},
		},
		{
			name:      "remove event",
			eventType: "remove",
			setup: func() error {
				// First create and add the service
				err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
				if err != nil {
					return err
				}
				err = manager.AddService(scriptPath)
				if err != nil {
					return err
				}
				// Then remove the file
				return os.Remove(scriptPath)
			},
			verify: func() bool {
				_, exists := manager.services["TestService"]
				return !exists // Should not exist after removal
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up services map before each test
			manager.services = make(map[string]*ManagedService)
			manager.scriptToService = make(map[string]string)

			err := tt.setup()
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Simulate file event by calling the appropriate method
			switch tt.eventType {
			case "create":
				err = manager.AddService(scriptPath)
			case "write":
				err = manager.RestartService(scriptPath)
			case "remove":
				err = manager.RemoveService(scriptPath)
			}

			if err != nil {
				t.Fatalf("HandleFileEvent failed: %v", err)
			}

			if !tt.verify() {
				t.Errorf("Event %s did not have expected effect", tt.eventType)
			}
		})
	}
}

func TestManager_IsValidScript(t *testing.T) {
	tempDir := t.TempDir()
	logger := logging.SetupLogger("info")
	natsConn := (*nats.Conn)(nil) // Use nil for testing

	manager := NewManager(tempDir, natsConn, logger)

	tests := []struct {
		name        string
		filename    string
		content     string
		mode        os.FileMode
		expectValid bool
	}{
		{
			name:     "valid shell script",
			filename: "test.sh",
			content: `#!/bin/bash
if [[ "$1" == "info" ]]; then
  echo '{"name":"TestService","version":"1.0.0","description":"Test","endpoints":[{"name":"Test","subject":"test"}]}'
  exit 0
fi
echo "response"
`,
			mode:        0755,
			expectValid: true,
		},
		{
			name:        "non-shell file",
			filename:    "test.txt",
			content:     "not a shell script",
			mode:        0755,
			expectValid: false,
		},
		{
			name:     "non-executable shell script",
			filename: "test.sh",
			content: `#!/bin/bash
echo "not executable"
`,
			mode:        0644,
			expectValid: false,
		},
		{
			name:     "invalid service definition",
			filename: "invalid.sh",
			content: `#!/bin/bash
if [[ "$1" == "info" ]]; then
  echo "invalid json"
  exit 0
fi
echo "response"
`,
			mode:        0755,
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scriptPath := filepath.Join(tempDir, tt.filename)
			err := os.WriteFile(scriptPath, []byte(tt.content), tt.mode)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			isValid := manager.IsValidScript(scriptPath)
			if isValid != tt.expectValid {
				t.Errorf("Expected IsValidScript to return %v, got %v", tt.expectValid, isValid)
			}
		})
	}
}

func TestManager_String(t *testing.T) {
	logger := logging.SetupLogger("info")
	natsConn := (*nats.Conn)(nil) // Use nil for testing
	scriptsPath := "/path/to/scripts"

	manager := NewManager(scriptsPath, natsConn, logger)

	expected := "ServiceManager(/path/to/scripts)"
	if manager.String() != expected {
		t.Errorf("Expected %s, got %s", expected, manager.String())
	}
}
