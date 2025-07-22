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

	// Check that the valid service was discovered
	validScriptPath := filepath.Join(tempDir, "valid.sh")
	if _, exists := manager.services[validScriptPath]; !exists {
		t.Error("Expected valid.sh to be discovered as a service")
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

	// Check that service was added
	if _, exists := manager.services[scriptPath]; !exists {
		t.Error("Expected service to be added to services map")
	}

	// Check that service token was stored
	if _, exists := manager.serviceTokens[scriptPath]; !exists {
		t.Error("Expected service token to be stored")
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

	// Verify service was added
	if _, exists := manager.services[scriptPath]; !exists {
		t.Fatal("Service was not added")
	}

	// Remove the service
	err = manager.RemoveService(scriptPath)
	if err != nil {
		t.Fatalf("RemoveService failed: %v", err)
	}

	// Check that service was removed
	if _, exists := manager.services[scriptPath]; exists {
		t.Error("Expected service to be removed from services map")
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

	// Get original service
	originalService := manager.services[scriptPath]

	// Restart the service
	err = manager.RestartService(scriptPath)
	if err != nil {
		t.Fatalf("RestartService failed: %v", err)
	}

	// Check that a new service instance was created
	newService := manager.services[scriptPath]
	if originalService == newService {
		t.Error("Expected service instance to be replaced after restart")
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
				_, exists := manager.services[scriptPath]
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
				_, exists := manager.services[scriptPath]
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
				_, exists := manager.services[scriptPath]
				return !exists // Should not exist after removal
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up services map before each test
			manager.services = make(map[string]*ManagedService)

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
