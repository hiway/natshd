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
	// Ensure global level is set to Info for this test
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	var buf bytes.Buffer

	contextLogger := NewContextLogger(&buf, zerolog.InfoLevel, "test-service", "script.sh")
	contextLogger.Info().Msg("test message")

	output := buf.String()
	if output == "" {
		t.Fatalf("No output in buffer")
	}

	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse log as JSON: %v, output was: %q", err, output)
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
	logger := SetupLoggerWithWriter(&buf, "debug")

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

	if logEntry["level"] != "debug" {
		t.Errorf("Expected level 'debug' for successful request")
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

func TestLogServiceLifecycleLevels(t *testing.T) {
	tests := []struct {
		name          string
		action        string
		loggerLevel   string
		expectedLevel string
		shouldLog     bool
	}{
		// Info level actions (should log at info level)
		{"added action logs at info", "added", "info", "info", true},
		{"removed action logs at info", "removed", "info", "info", true},
		{"restarted action logs at info", "restarted", "info", "info", true},
		{"starting action logs at info", "starting", "info", "info", true},

		// Debug level actions (should log at debug level)
		{"initializing action logs at debug", "initializing", "debug", "debug", true},
		{"initialized action logs at debug", "initialized", "debug", "debug", true},
		{"script_removed action logs at debug", "script_removed", "debug", "debug", true},

		// Info logger should not see debug level actions
		{"initializing not visible at info level", "initializing", "info", "debug", false},
		{"initialized not visible at info level", "initialized", "info", "debug", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := SetupLoggerWithWriter(&buf, tt.loggerLevel)

			LogServiceLifecycle(logger, tt.action, "TestService", "test.sh")

			output := buf.String()
			hasOutput := len(strings.TrimSpace(output)) > 0

			if tt.shouldLog && !hasOutput {
				t.Errorf("Expected log output but got none for action %s", tt.action)
			}

			if !tt.shouldLog && hasOutput {
				t.Errorf("Expected no log output but got: %s", output)
			}

			if hasOutput {
				var logEntry map[string]interface{}
				err := json.Unmarshal([]byte(output), &logEntry)
				if err != nil {
					t.Fatalf("Failed to parse log as JSON: %v", err)
				}

				if logEntry["level"] != tt.expectedLevel {
					t.Errorf("Expected level '%s', got %v", tt.expectedLevel, logEntry["level"])
				}
			}
		})
	}
}

func TestLogServiceLifecycleNoDuplicateFields(t *testing.T) {
	var buf bytes.Buffer

	// Create a context logger with service and script fields
	contextLogger := NewContextLogger(&buf, zerolog.InfoLevel, "TestService", "/path/to/script.sh")
	// This should not create duplicate fields
	LogServiceLifecycle(contextLogger, "starting", "TestService", "/path/to/script.sh")

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse log as JSON: %v", err)
	}

	// Verify the log entry structure
	logEntryStr := buf.String()

	// Count occurrences of each field to ensure no duplicates
	serviceCount := strings.Count(logEntryStr, `"service":`)
	scriptCount := strings.Count(logEntryStr, `"script":`)

	if serviceCount != 1 {
		t.Errorf("Service field should appear exactly once, got %d occurrences in: %s", serviceCount, logEntryStr)
	}
	if scriptCount != 1 {
		t.Errorf("Script field should appear exactly once, got %d occurrences in: %s", scriptCount, logEntryStr)
	}

	// Verify values are correct
	if logEntry["action"] != "starting" {
		t.Errorf("Expected action 'starting', got %v", logEntry["action"])
	}
	if logEntry["service"] != "TestService" {
		t.Errorf("Expected service 'TestService', got %v", logEntry["service"])
	}
	if logEntry["script"] != "/path/to/script.sh" {
		t.Errorf("Expected script '/path/to/script.sh', got %v", logEntry["script"])
	}
	if logEntry["message"] != "Service lifecycle event" {
		t.Errorf("Expected message 'Service lifecycle event', got %v", logEntry["message"])
	}
}

func TestLogServiceLifecycleEmptyToPopulatedService(t *testing.T) {
	var buf bytes.Buffer

	// Ensure global level is set to Debug for this test
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	// Create a context logger with empty service field (simulating initial ManagedService state)
	emptyContextLogger := NewContextLogger(&buf, zerolog.DebugLevel, "", "/path/to/script.sh")

	// Log a service lifecycle event with empty context - this should only show empty service
	LogServiceLifecycle(emptyContextLogger, "initializing", "GreetingService", "/path/to/script.sh")

	// Reset buffer
	buf.Reset()

	// Now create a properly initialized logger with the service name (simulating post-initialization)
	populatedContextLogger := NewContextLogger(&buf, zerolog.DebugLevel, "GreetingService", "/path/to/script.sh")

	// This should show the populated service name (using debug level since initialized logs at debug level)
	LogServiceLifecycle(populatedContextLogger, "initialized", "GreetingService", "/path/to/script.sh")

	logOutput := buf.String()

	// We should only have the populated service field, no duplicates
	serviceCount := strings.Count(logOutput, `"service":"`)
	if serviceCount != 1 {
		t.Errorf("Service field should appear exactly once, got %d occurrences in: %s", serviceCount, logOutput)
	}

	// Should have the populated service
	if !strings.Contains(logOutput, `"service":"GreetingService"`) {
		t.Errorf("Expected populated service field, got: %s", logOutput)
	}

	// Should NOT have empty service
	if strings.Contains(logOutput, `"service":""`) {
		t.Errorf("Should not contain empty service field, got: %s", logOutput)
	}
}

// TestError is a simple error type for testing
type TestError struct {
	message string
}

func (e *TestError) Error() string {
	return e.message
}

func TestLogManagerOperation(t *testing.T) {
	tests := []struct {
		name          string
		action        string
		data          map[string]interface{}
		loggerLevel   string
		expectedLevel string
		shouldLog     bool
		expectedMsg   string
	}{
		// Info level operations
		{"starting logs at info", "starting", map[string]interface{}{"scripts_path": "/test"}, "info", "info", true, "Service manager starting"},
		{"stopping logs at info", "stopping", nil, "info", "info", true, "Service manager stopping"},
		{"discovery_completed logs at info", "discovery_completed", map[string]interface{}{"count": 3}, "info", "info", true, "Service discovery completed"},

		// Debug level operations
		{"discovering logs at debug", "discovering", map[string]interface{}{"path": "/test"}, "debug", "debug", true, "Discovering services"},
		{"file_watcher_setup logs at debug", "file_watcher_setup", map[string]interface{}{"path": "/test"}, "debug", "debug", true, "File watcher setup completed"},
		{"adding logs at debug", "adding", map[string]interface{}{"script": "test.sh"}, "debug", "debug", true, "Adding service"},

		// Info logger should not see debug operations
		{"discovering not visible at info level", "discovering", map[string]interface{}{"path": "/test"}, "info", "debug", false, "Discovering services"},
		{"adding not visible at info level", "adding", map[string]interface{}{"script": "test.sh"}, "info", "debug", false, "Adding service"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := SetupLoggerWithWriter(&buf, tt.loggerLevel)

			LogManagerOperation(logger, tt.action, tt.data)

			output := buf.String()
			hasOutput := len(strings.TrimSpace(output)) > 0

			if tt.shouldLog && !hasOutput {
				t.Errorf("Expected log output but got none for action %s", tt.action)
			}

			if !tt.shouldLog && hasOutput {
				t.Errorf("Expected no log output but got: %s", output)
			}

			if hasOutput {
				var logEntry map[string]interface{}
				err := json.Unmarshal([]byte(output), &logEntry)
				if err != nil {
					t.Fatalf("Failed to parse log as JSON: %v", err)
				}

				if logEntry["level"] != tt.expectedLevel {
					t.Errorf("Expected level '%s', got %v", tt.expectedLevel, logEntry["level"])
				}

				if logEntry["action"] != tt.action {
					t.Errorf("Expected action '%s', got %v", tt.action, logEntry["action"])
				}

				if logEntry["message"] != tt.expectedMsg {
					t.Errorf("Expected message '%s', got %v", tt.expectedMsg, logEntry["message"])
				}

				// Check data fields are present
				for key, expectedValue := range tt.data {
					actualValue := logEntry[key]
					// Handle type conversion for JSON numbers
					if expectedInt, ok := expectedValue.(int); ok {
						if actualFloat, ok := actualValue.(float64); ok {
							if int(actualFloat) != expectedInt {
								t.Errorf("Expected %s='%v', got %v", key, expectedValue, actualValue)
							}
						} else {
							t.Errorf("Expected %s='%v', got %v", key, expectedValue, actualValue)
						}
					} else if actualValue != expectedValue {
						t.Errorf("Expected %s='%v', got %v", key, expectedValue, actualValue)
					}
				}
			}
		})
	}
}

func TestLogFileWatchEvent(t *testing.T) {
	var buf bytes.Buffer
	logger := SetupLoggerWithWriter(&buf, "debug")

	LogFileWatchEvent(logger, "WRITE", "/test/script.sh")

	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse log as JSON: %v", err)
	}

	if logEntry["level"] != "debug" {
		t.Errorf("Expected level 'debug', got %v", logEntry["level"])
	}

	if logEntry["event"] != "WRITE" {
		t.Errorf("Expected event 'WRITE', got %v", logEntry["event"])
	}

	if logEntry["file"] != "/test/script.sh" {
		t.Errorf("Expected file '/test/script.sh', got %v", logEntry["file"])
	}

	if logEntry["message"] != "File system event" {
		t.Errorf("Expected message 'File system event', got %v", logEntry["message"])
	}
}
