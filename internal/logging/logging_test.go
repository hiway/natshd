package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestSetupLogger(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected zerolog.Level
	}{
		{"trace level", "trace", zerolog.TraceLevel},
		{"debug level", "debug", zerolog.DebugLevel},
		{"info level", "info", zerolog.InfoLevel},
		{"warn level", "warn", zerolog.WarnLevel},
		{"error level", "error", zerolog.ErrorLevel},
		{"fatal level", "fatal", zerolog.FatalLevel},
		{"panic level", "panic", zerolog.PanicLevel},
		{"invalid level defaults to info", "invalid", zerolog.InfoLevel},
		{"empty level defaults to info", "", zerolog.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := SetupLogger(tt.level)

			// Check that the global level was set correctly
			actualLevel := zerolog.GlobalLevel()
			if actualLevel != tt.expected {
				t.Errorf("Expected global level %v, got %v", tt.expected, actualLevel)
			}

			// Verify it's a valid logger by testing that it can log
			// This is sufficient since zerolog.Logger is a struct
			logger.Info().Msg("test")
		})
	}
}

func TestLoggerOutput(t *testing.T) {
	// Capture logger output
	var buf bytes.Buffer
	logger := SetupLoggerWithWriter(&buf, "info")

	logger.Info().Str("component", "test").Msg("test message")

	output := buf.String()
	if output == "" {
		t.Error("Expected log output")
	}

	// Parse the JSON to verify structure
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	if err != nil {
		t.Errorf("Failed to parse log as JSON: %v", err)
	}

	// Check required fields
	if logEntry["level"] != "info" {
		t.Errorf("Expected level 'info', got %v", logEntry["level"])
	}

	if logEntry["message"] != "test message" {
		t.Errorf("Expected message 'test message', got %v", logEntry["message"])
	}

	if logEntry["component"] != "test" {
		t.Errorf("Expected component 'test', got %v", logEntry["component"])
	}

	// Check timestamp exists
	if _, exists := logEntry["time"]; !exists {
		t.Error("Expected timestamp field in log entry")
	}
}

func TestLoggerLevels(t *testing.T) {
	tests := []struct {
		name        string
		loggerLevel string
		logLevel    string
		shouldLog   bool
	}{
		{"debug logger logs info", "debug", "info", true},
		{"debug logger logs debug", "debug", "debug", true},
		{"info logger logs info", "info", "info", true},
		{"info logger does not log debug", "info", "debug", false},
		{"warn logger logs warn", "warn", "warn", true},
		{"warn logger does not log info", "warn", "info", false},
		{"error logger logs error", "error", "error", true},
		{"error logger does not log warn", "error", "warn", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := SetupLoggerWithWriter(&buf, tt.loggerLevel)

			// Log at the specified level
			switch tt.logLevel {
			case "debug":
				logger.Debug().Msg("debug message")
			case "info":
				logger.Info().Msg("info message")
			case "warn":
				logger.Warn().Msg("warn message")
			case "error":
				logger.Error().Msg("error message")
			}

			output := buf.String()
			hasOutput := len(strings.TrimSpace(output)) > 0

			if tt.shouldLog && !hasOutput {
				t.Errorf("Expected log output but got none")
			}

			if !tt.shouldLog && hasOutput {
				t.Errorf("Expected no log output but got: %s", output)
			}
		})
	}
}

func TestNewContextLogger(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := SetupLoggerWithWriter(&buf, "info")

	contextLogger := NewContextLogger(baseLogger, "test-service", "script.sh")

	contextLogger.Info().Msg("test message")

	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse log as JSON: %v", err)
	}

	if logEntry["service"] != "test-service" {
		t.Errorf("Expected service 'test-service', got %v", logEntry["service"])
	}

	if logEntry["script"] != "script.sh" {
		t.Errorf("Expected script 'script.sh', got %v", logEntry["script"])
	}

	if logEntry["message"] != "test message" {
		t.Errorf("Expected message 'test message', got %v", logEntry["message"])
	}
}

func TestLogRequestResponse(t *testing.T) {
	var buf bytes.Buffer
	logger := SetupLoggerWithWriter(&buf, "info")

	LogRequestResponse(logger, "test.subject", []byte(`{"input":"data"}`), []byte(`{"output":"result"}`), nil)

	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse log as JSON: %v", err)
	}

	if logEntry["subject"] != "test.subject" {
		t.Errorf("Expected subject 'test.subject', got %v", logEntry["subject"])
	}

	if logEntry["request"] != `{"input":"data"}` {
		t.Errorf("Expected request payload in log")
	}

	if logEntry["response"] != `{"output":"result"}` {
		t.Errorf("Expected response payload in log")
	}

	if logEntry["level"] != "info" {
		t.Errorf("Expected level 'info' for successful request")
	}
}

func TestLogRequestResponseWithError(t *testing.T) {
	var buf bytes.Buffer
	logger := SetupLoggerWithWriter(&buf, "info")

	testError := &TestError{message: "script failed"}
	LogRequestResponse(logger, "test.subject", []byte(`{"input":"data"}`), nil, testError)

	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse log as JSON: %v", err)
	}

	if logEntry["level"] != "error" {
		t.Errorf("Expected level 'error' for failed request")
	}

	if logEntry["error"] != "script failed" {
		t.Errorf("Expected error message in log")
	}
}

func TestLogServiceLifecycle(t *testing.T) {
	var buf bytes.Buffer
	logger := SetupLoggerWithWriter(&buf, "info")

	LogServiceLifecycle(logger, "start", "TestService", "test.sh")

	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse log as JSON: %v", err)
	}

	if logEntry["action"] != "start" {
		t.Errorf("Expected action 'start', got %v", logEntry["action"])
	}

	if logEntry["service"] != "TestService" {
		t.Errorf("Expected service 'TestService', got %v", logEntry["service"])
	}

	if logEntry["script"] != "test.sh" {
		t.Errorf("Expected script 'test.sh', got %v", logEntry["script"])
	}
}

// TestError is a simple error type for testing
type TestError struct {
	message string
}

func (e *TestError) Error() string {
	return e.message
}
