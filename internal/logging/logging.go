package logging

import (
	"bytes"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// SetupLogger configures and returns a structured JSON logger with the specified level
func SetupLogger(level string) zerolog.Logger {
	return SetupLoggerWithWriter(os.Stdout, level)
}

// SetupLoggerWithWriter configures a logger with a custom writer (useful for testing)
func SetupLoggerWithWriter(writer io.Writer, level string) zerolog.Logger {
	// Parse and set the log level
	var logLevel zerolog.Level
	var err error

	if level == "" {
		logLevel = zerolog.InfoLevel
	} else {
		logLevel, err = zerolog.ParseLevel(level)
		if err != nil {
			// Default to info level if parsing fails
			logLevel = zerolog.InfoLevel
		}
	}

	zerolog.SetGlobalLevel(logLevel)

	// Configure zerolog for production JSON output
	zerolog.TimeFieldFormat = time.RFC3339

	return zerolog.New(writer).
		With().
		Timestamp().
		Logger()
}

// NewContextLogger creates a new logger with service and script context
func NewContextLogger(writer io.Writer, level zerolog.Level, serviceName, scriptPath string) zerolog.Logger {
	freshLogger := zerolog.New(writer).Level(level)
	contextLogger := freshLogger.With().
		Timestamp().
		Str("service", serviceName).
		Str("script", scriptPath).
		Logger()
	return contextLogger
}

// LogRequestResponse logs NATS request/response interactions
func LogRequestResponse(logger zerolog.Logger, subject string, request, response []byte, err error) {
		event := logger.Debug()
		if err != nil {
			event = logger.Error().Err(err)
		}

	event = event.
		Str("subject", subject).
		Str("request", string(request))

	if response != nil {
		event = event.Str("response", string(response))
	}

	event.Msg("NATS request processed")
}

// LogServiceLifecycle logs service start, stop, and restart events
// This function avoids field duplication by only adding fields that aren't already in the logger context
// Uses different log levels based on action importance:
// - Info: added, removed, restarted, starting (key operational events)
// - Debug: initializing, initialized, script_removed (internal state transitions)
func LogServiceLifecycle(logger zerolog.Logger, action, serviceName, scriptPath string) {
	// Test if logger already has context by creating a test event
	var testBuf bytes.Buffer
	testLogger := logger.Output(&testBuf)
	testLogger.Info().Msg("")
	testOutput := testBuf.String()

	// Check if service and script fields already exist in the context
	hasServiceContext := strings.Contains(testOutput, `"service":`)
	hasScriptContext := strings.Contains(testOutput, `"script":`)

	// Determine log level based on action
	var event *zerolog.Event
	switch action {
	case "initializing", "initialized", "script_removed":
		// Debug level for internal state transitions
		event = logger.Debug().Str("action", action)
	case "added", "removed", "restarted", "starting":
		// Info level for key operational events
		event = logger.Info().Str("action", action)
	default:
		// Default to info level for unknown actions
		event = logger.Info().Str("action", action)
	}

	// Only add fields that aren't already in the context
	if !hasServiceContext && serviceName != "" {
		event = event.Str("service", serviceName)
	}
	if !hasScriptContext && scriptPath != "" {
		event = event.Str("script", scriptPath)
	}

	event.Msg("Service lifecycle event")
}

// LogFileWatchEvent logs filesystem change events at debug level
func LogFileWatchEvent(logger zerolog.Logger, event, filePath string) {
	logger.Debug().
		Str("event", event).
		Str("file", filePath).
		Msg("File system event")
}

// LogManagerOperation logs service manager operations at appropriate levels
// - Debug: discovering, file watcher setup, adding individual scripts
// - Info: discovery completion count, manager start/stop
func LogManagerOperation(logger zerolog.Logger, action string, data map[string]interface{}) {
	var event *zerolog.Event

	switch action {
	case "starting", "stopping", "discovery_completed":
		// Info level for key operational milestones
		event = logger.Info()
	case "discovering", "file_watcher_setup", "adding", "removing", "restarting":
		// Debug level for internal operations
		event = logger.Debug()
	default:
		// Default to debug level for unknown actions
		event = logger.Debug()
	}

	event = event.Str("action", action)

	// Add any additional data fields
	for key, value := range data {
		switch v := value.(type) {
		case string:
			event = event.Str(key, v)
		case int:
			event = event.Int(key, v)
		case bool:
			event = event.Bool(key, v)
		default:
			event = event.Interface(key, v)
		}
	}

	// Choose message based on action type
	switch action {
	case "starting":
		event.Msg("Service manager starting")
	case "stopping":
		event.Msg("Service manager stopping")
	case "discovering":
		event.Msg("Discovering services")
	case "discovery_completed":
		event.Msg("Service discovery completed")
	case "file_watcher_setup":
		event.Msg("File watcher setup completed")
	case "adding":
		event.Msg("Adding service")
	case "removing":
		event.Msg("Removing service")
	case "restarting":
		event.Msg("Restarting service")
	default:
		event.Msg("Manager operation")
	}
}

// LogError logs errors with context
func LogError(logger zerolog.Logger, err error, context string) {
	logger.Error().
		Err(err).
		Str("context", context).
		Msg("Error occurred")
}
