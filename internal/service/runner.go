package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// ScriptRunner handles execution of shell scripts for service operations
type ScriptRunner struct {
	scriptPath string
}

// ExecutionResult represents the result of executing a script
type ExecutionResult struct {
	Success  bool   `json:"success"`
	Stdout   []byte `json:"stdout,omitempty"`
	Stderr   []byte `json:"stderr,omitempty"`
	ExitCode int    `json:"exit_code"`
}

// NewScriptRunner creates a new script runner for the given script path
func NewScriptRunner(scriptPath string) *ScriptRunner {
	return &ScriptRunner{
		scriptPath: scriptPath,
	}
}

// GetServiceDefinition executes the script with "info" argument to get service definition
func (sr *ScriptRunner) GetServiceDefinition(ctx context.Context) (ServiceDefinition, error) {
	cmd := exec.CommandContext(ctx, sr.scriptPath, "info")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Check if context was cancelled (timeout)
		if ctx.Err() != nil {
			return ServiceDefinition{}, fmt.Errorf("script execution timeout: %w", ctx.Err())
		}

		stderrOutput := stderr.String()
		if stderrOutput != "" {
			return ServiceDefinition{}, fmt.Errorf("script execution failed: %w (stderr: %s)", err, stderrOutput)
		}
		return ServiceDefinition{}, fmt.Errorf("script execution failed: %w", err)
	}

	var def ServiceDefinition
	if err := json.Unmarshal(stdout.Bytes(), &def); err != nil {
		return ServiceDefinition{}, fmt.Errorf("failed to parse service definition JSON: %w", err)
	}

	if err := def.Validate(); err != nil {
		return ServiceDefinition{}, fmt.Errorf("invalid service definition: %w", err)
	}

	return def, nil
}

// ExecuteRequest executes the script with the given subject and payload
func (sr *ScriptRunner) ExecuteRequest(ctx context.Context, subject string, payload []byte) (ExecutionResult, error) {
	cmd := exec.CommandContext(ctx, sr.scriptPath, subject)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = bytes.NewReader(payload)

	err := cmd.Run()

	result := ExecutionResult{
		Success:  err == nil,
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		ExitCode: 0,
	}

	if err != nil {
		// Check if context was cancelled (timeout)
		if ctx.Err() != nil {
			return result, fmt.Errorf("script execution timeout: %w", ctx.Err())
		}

		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			// Non-exit error (e.g., file not found)
			return result, fmt.Errorf("script execution failed: %w", err)
		}
	}

	return result, nil
}

// ToJSON converts the execution result to JSON format
func (er ExecutionResult) ToJSON() ([]byte, error) {
	// Create a simplified structure for JSON output
	jsonResult := map[string]interface{}{
		"success":   er.Success,
		"exit_code": er.ExitCode,
	}

	if len(er.Stdout) > 0 {
		jsonResult["stdout"] = string(er.Stdout)
	}

	if len(er.Stderr) > 0 {
		jsonResult["stderr"] = string(er.Stderr)
	}

	return json.Marshal(jsonResult)
}
