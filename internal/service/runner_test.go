package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewScriptRunner(t *testing.T) {
	scriptPath := "/path/to/script.sh"
	runner := NewScriptRunner(scriptPath)

	if runner.scriptPath != scriptPath {
		t.Errorf("Expected script path %s, got %s", scriptPath, runner.scriptPath)
	}
}

func TestScriptRunner_GetServiceDefinition(t *testing.T) {
	// Create a temporary script that returns valid service definition
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "test_service.sh")

	validDefinitionScript := `#!/bin/bash
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
echo "Not info call"
`

	err := os.WriteFile(scriptPath, []byte(validDefinitionScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	runner := NewScriptRunner(scriptPath)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	def, err := runner.GetServiceDefinition(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if def.Name != "TestService" {
		t.Errorf("Expected service name TestService, got %s", def.Name)
	}

	if def.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", def.Version)
	}

	if len(def.Endpoints) != 1 {
		t.Errorf("Expected 1 endpoint, got %d", len(def.Endpoints))
	}

	if len(def.Endpoints) > 0 && def.Endpoints[0].Subject != "test.endpoint" {
		t.Errorf("Expected endpoint subject test.endpoint, got %s", def.Endpoints[0].Subject)
	}
}

func TestScriptRunner_GetServiceDefinition_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "invalid_json.sh")

	invalidJSONScript := `#!/bin/bash
if [[ "$1" == "info" ]]; then
  echo "{ invalid json"
  exit 0
fi
`

	err := os.WriteFile(scriptPath, []byte(invalidJSONScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	runner := NewScriptRunner(scriptPath)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = runner.GetServiceDefinition(ctx)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestScriptRunner_GetServiceDefinition_ScriptError(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "error_script.sh")

	errorScript := `#!/bin/bash
if [[ "$1" == "info" ]]; then
  echo "Error message" >&2
  exit 1
fi
`

	err := os.WriteFile(scriptPath, []byte(errorScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	runner := NewScriptRunner(scriptPath)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = runner.GetServiceDefinition(ctx)
	if err == nil {
		t.Error("Expected error for script that exits with non-zero code")
	}

	if !strings.Contains(err.Error(), "Error message") {
		t.Errorf("Expected error message to contain stderr output, got: %v", err)
	}
}

func TestScriptRunner_GetServiceDefinition_Timeout(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "slow_script.sh")

	slowScript := `#!/bin/bash
if [[ "$1" == "info" ]]; then
  sleep 10
  echo "{}"
  exit 0
fi
`

	err := os.WriteFile(scriptPath, []byte(slowScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	runner := NewScriptRunner(scriptPath)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = runner.GetServiceDefinition(ctx)
	if err == nil {
		t.Error("Expected timeout error")
	}

	if !strings.Contains(err.Error(), "context deadline exceeded") && !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestScriptRunner_ExecuteRequest(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "echo_service.sh")

	echoScript := `#!/bin/bash
if [[ "$1" == "info" ]]; then
  cat <<EOF
{
  "name": "EchoService",
  "endpoints": [{"name": "Echo", "subject": "echo.test"}]
}
EOF
  exit 0
fi

# Echo the subject and payload
SUBJECT=$1
read PAYLOAD
echo "{\"subject\":\"${SUBJECT}\", \"payload\":${PAYLOAD}}"
`

	err := os.WriteFile(scriptPath, []byte(echoScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	runner := NewScriptRunner(scriptPath)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	subject := "echo.test"
	payload := `{"message": "hello world"}`

	result, err := runner.ExecuteRequest(ctx, subject, []byte(payload))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Success != true {
		t.Error("Expected successful execution")
	}

	if len(result.Stdout) == 0 {
		t.Error("Expected stdout output")
	}

	// Verify the output contains our input
	var output map[string]interface{}
	err = json.Unmarshal(result.Stdout, &output)
	if err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}

	if output["subject"] != subject {
		t.Errorf("Expected subject %s in output, got %v", subject, output["subject"])
	}
}

func TestScriptRunner_ExecuteRequest_ScriptError(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "error_service.sh")

	errorScript := `#!/bin/bash
if [[ "$1" == "info" ]]; then
  cat <<EOF
{
  "name": "ErrorService",
  "endpoints": [{"name": "Error", "subject": "error.test"}]
}
EOF
  exit 0
fi

echo "This is stdout output"
echo "This is stderr output" >&2
exit 1
`

	err := os.WriteFile(scriptPath, []byte(errorScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	runner := NewScriptRunner(scriptPath)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := runner.ExecuteRequest(ctx, "error.test", []byte(`{"test": true}`))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Success != false {
		t.Error("Expected failed execution")
	}

	if !strings.Contains(string(result.Stdout), "This is stdout output") {
		t.Error("Expected stdout to be captured")
	}

	if !strings.Contains(string(result.Stderr), "This is stderr output") {
		t.Error("Expected stderr to be captured")
	}

	if result.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", result.ExitCode)
	}
}

func TestScriptRunner_ExecuteRequest_Timeout(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "slow_service.sh")

	slowScript := `#!/bin/bash
if [[ "$1" == "info" ]]; then
  cat <<EOF
{
  "name": "SlowService",
  "endpoints": [{"name": "Slow", "subject": "slow.test"}]
}
EOF
  exit 0
fi

sleep 10
echo "Too slow"
`

	err := os.WriteFile(scriptPath, []byte(slowScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	runner := NewScriptRunner(scriptPath)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = runner.ExecuteRequest(ctx, "slow.test", []byte(`{}`))
	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestScriptRunner_FileNotExists(t *testing.T) {
	runner := NewScriptRunner("/nonexistent/script.sh")
	ctx := context.Background()

	_, err := runner.GetServiceDefinition(ctx)
	if err == nil {
		t.Error("Expected error for non-existent script")
	}
}

func TestExecutionResult_ToJSON(t *testing.T) {
	tests := []struct {
		name   string
		result ExecutionResult
	}{
		{
			name: "successful execution",
			result: ExecutionResult{
				Success:  true,
				Stdout:   []byte(`{"message": "success"}`),
				Stderr:   []byte(""),
				ExitCode: 0,
			},
		},
		{
			name: "failed execution",
			result: ExecutionResult{
				Success:  false,
				Stdout:   []byte("output"),
				Stderr:   []byte("error message"),
				ExitCode: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, err := tt.result.ToJSON()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			var parsed map[string]interface{}
			err = json.Unmarshal(jsonData, &parsed)
			if err != nil {
				t.Errorf("Failed to parse JSON: %v", err)
			}

			if parsed["success"] != tt.result.Success {
				t.Errorf("Expected success %v, got %v", tt.result.Success, parsed["success"])
			}

			if int(parsed["exit_code"].(float64)) != tt.result.ExitCode {
				t.Errorf("Expected exit_code %d, got %v", tt.result.ExitCode, parsed["exit_code"])
			}
		})
	}
}
