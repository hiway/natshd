package logging

import (
	"io"
	"os"
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

// NewContextLogger creates a logger with service and script context
func NewContextLogger(baseLogger zerolog.Logger, serviceName, scriptPath string) zerolog.Logger {
	return baseLogger.With().
		Str("service", serviceName).
		Str("script", scriptPath).
		Logger()
}

// LogRequestResponse logs NATS request/response interactions
func LogRequestResponse(logger zerolog.Logger, subject string, request, response []byte, err error) {
	event := logger.Info()

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
func LogServiceLifecycle(logger zerolog.Logger, action, serviceName, scriptPath string) {
	logger.Info().
		Str("action", action).
		Str("service", serviceName).
		Str("script", scriptPath).
		Msg("Service lifecycle event")
}

// LogFileWatchEvent logs filesystem change events
func LogFileWatchEvent(logger zerolog.Logger, event, filePath string) {
	logger.Info().
		Str("event", event).
		Str("file", filePath).
		Msg("File system event")
}

// LogError logs errors with context
func LogError(logger zerolog.Logger, err error, context string) {
	logger.Error().
		Err(err).
		Str("context", context).
		Msg("Error occurred")
}
