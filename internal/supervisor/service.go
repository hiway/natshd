package supervisor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/hiway/natshd/internal/config"
	"github.com/hiway/natshd/internal/logging"
	"github.com/hiway/natshd/internal/service"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
	"github.com/rs/zerolog"
	"github.com/thejerf/suture/v4"
)

// ScriptRunner interface for executing scripts (allows for mocking)
type ScriptRunner interface {
	GetServiceDefinition(ctx context.Context) (service.ServiceDefinition, error)
	ExecuteRequest(ctx context.Context, subject string, payload []byte) (service.ExecutionResult, error)
}

// ManagedService represents a supervised NATS microservice backed by shell script(s)
type ManagedService struct {
	scripts      map[string]ScriptRunner // scriptPath -> runner mapping
	natsConn     *nats.Conn
	logger       zerolog.Logger
	definition   service.ServiceDefinition
	natsService  micro.Service
	initialized  bool
	serviceToken suture.ServiceToken
	config       config.Config
}

// NewManagedService creates a new managed service with the provided config
func NewManagedService(scriptPath string, natsConn *nats.Conn, logger zerolog.Logger, cfg config.Config) *ManagedService {
	serviceLogger := logging.NewContextLogger(os.Stderr, logger.GetLevel(), "", scriptPath)
	return &ManagedService{
		scripts:  make(map[string]ScriptRunner),
		natsConn: natsConn,
		logger:   serviceLogger,
		config:   cfg,
	}
}

// AddScript adds a script to this managed service (for grouping scripts by service name)
func (ms *ManagedService) AddScript(scriptPath string) {
	ms.scripts[scriptPath] = service.NewScriptRunner(scriptPath)
}

// Initialize loads the service definition from the scripts and validates it
func (ms *ManagedService) Initialize(ctx context.Context) error {
	if len(ms.scripts) == 0 {
		return fmt.Errorf("no scripts added to service")
	}

	// Get first script path for logging purposes
	var firstScriptPath string
	for path := range ms.scripts {
		firstScriptPath = path
		break
	}

	logging.LogServiceLifecycle(ms.logger, "initializing", "", firstScriptPath)

	// Get service definition from the first script to establish the service name
	var firstRunner ScriptRunner
	for _, runner := range ms.scripts {
		firstRunner = runner
		break
	}

	definition, err := firstRunner.GetServiceDefinition(ctx)
	if err != nil {
		logging.LogError(ms.logger, err, "failed to get service definition")
		return fmt.Errorf("failed to get service definition: %w", err)
	}

	// Start with the first script's definition
	ms.definition = definition

	// Collect all unique endpoints from all scripts with the same service name
	allEndpoints := make(map[string]service.Endpoint) // subject -> endpoint
	for scriptPath, runner := range ms.scripts {
		scriptDef, err := runner.GetServiceDefinition(ctx)
		if err != nil {
			logging.LogError(ms.logger, err, "failed to get service definition from script "+scriptPath)
			continue // Skip this script but continue with others
		}

		// Verify service name matches
		if scriptDef.Name != definition.Name {
			ms.logger.Warn().
				Str("script", scriptPath).
				Str("expected_name", definition.Name).
				Str("actual_name", scriptDef.Name).
				Msg("Script service name mismatch, skipping")
			continue
		}

		// Add endpoints from this script
		for _, endpoint := range scriptDef.Endpoints {
			// Apply hostname prefixing to the subject
			originalSubject := endpoint.Subject
			endpoint.Subject = ms.config.PrefixSubject(originalSubject)

			if existing, exists := allEndpoints[endpoint.Subject]; exists {
				ms.logger.Warn().
					Str("subject", endpoint.Subject).
					Str("original_subject", originalSubject).
					Str("existing_name", existing.Name).
					Str("new_name", endpoint.Name).
					Msg("Duplicate endpoint subject found, keeping first")
				continue
			}
			allEndpoints[endpoint.Subject] = endpoint
		}
	}

	// Convert map back to slice
	endpoints := make([]service.Endpoint, 0, len(allEndpoints))
	for _, endpoint := range allEndpoints {
		endpoints = append(endpoints, endpoint)
	}
	ms.definition.Endpoints = endpoints

	// Update logger with service name only (script path is already in context)
	ms.logger = logging.NewContextLogger(os.Stderr, ms.logger.GetLevel(), definition.Name, firstScriptPath)

	logging.LogServiceLifecycle(ms.logger, "initialized", definition.Name, firstScriptPath)
	ms.initialized = true

	return nil
}

// Serve implements the suture.Service interface
func (ms *ManagedService) Serve(ctx context.Context) error {
	// Get first script path for logging
	var firstScriptPath string
	for path := range ms.scripts {
		firstScriptPath = path
		break
	}

	logging.LogServiceLifecycle(ms.logger, "starting", ms.definition.Name, firstScriptPath)

	// Check if NATS connection is available
	if ms.natsConn == nil {
		return fmt.Errorf("NATS connection is nil")
	}

	// Create NATS microservice
	config := micro.Config{
		Name:        ms.definition.Name,
		Version:     ms.definition.Version,
		Description: ms.definition.Description,
	}

	// Add service to NATS
	service, err := micro.AddService(ms.natsConn, config)
	if err != nil {
		return fmt.Errorf("failed to add NATS microservice: %w", err)
	}

	// Add endpoints
	for _, endpoint := range ms.definition.Endpoints {
		endpoint := endpoint // capture loop variable

		// Prepare endpoint options
		opts := []micro.EndpointOpt{
			micro.WithEndpointSubject(endpoint.Subject),
		}

		// Convert metadata to NATS format if present
		if endpoint.Metadata != nil {
			natsMetadata := make(map[string]string)

			// Convert complex metadata to JSON strings
			for key, value := range endpoint.Metadata {
				if jsonBytes, err := json.Marshal(value); err == nil {
					natsMetadata[key] = string(jsonBytes)
				}
			}

			// Add description as a simple metadata field if present
			if endpoint.Description != "" {
				natsMetadata["description"] = endpoint.Description
			}

			if len(natsMetadata) > 0 {
				opts = append(opts, micro.WithEndpointMetadata(natsMetadata))
			}
		} else if endpoint.Description != "" {
			// If no complex metadata but description exists, add it
			natsMetadata := map[string]string{
				"description": endpoint.Description,
			}
			opts = append(opts, micro.WithEndpointMetadata(natsMetadata))
		}

		err := service.AddEndpoint(endpoint.Name, micro.HandlerFunc(func(req micro.Request) {
			ms.HandleRequest(&NATSRequestWrapper{req: req})
		}), opts...)
		if err != nil {
			return fmt.Errorf("failed to add endpoint %s: %w", endpoint.Name, err)
		}
	}

	// Store service for cleanup
	ms.natsService = service

	// Wait for context cancellation
	<-ctx.Done()

	// Cleanup
	if ms.natsService != nil {
		if err := ms.natsService.Stop(); err != nil {
			ms.logger.Error().Err(err).Msg("Error stopping NATS service")
		}
	}

	return ctx.Err()
}

// createHandler creates a NATS micro handler for the given subject
func (ms *ManagedService) createHandler(subject string) micro.Handler {
	return micro.HandlerFunc(func(req micro.Request) {
		ms.HandleRequest(&NATSRequestWrapper{req: req})
	})
}

// HandleRequest processes an incoming NATS request by executing the script
func (ms *ManagedService) HandleRequest(req Request) {
	ctx := context.Background()

	// Find the script that handles this subject
	var runner ScriptRunner
	requestSubject := req.Subject()

	for _, scriptRunner := range ms.scripts {
		// Get the service definition for this script
		def, err := scriptRunner.GetServiceDefinition(ctx)
		if err != nil {
			continue // Skip scripts that can't provide definition
		}

		// Check if this script handles the requested subject
		// We need to compare against the hostname-prefixed subjects
		for _, endpoint := range def.Endpoints {
			prefixedSubject := ms.config.PrefixSubject(endpoint.Subject)
			if prefixedSubject == requestSubject {
				runner = scriptRunner
				break
			}
		}

		if runner != nil {
			break
		}
	}

	if runner == nil {
		req.RespondError(fmt.Errorf("no script found for subject: %s", requestSubject))
		return
	}

	// Execute the script with the original (unprefixed) subject
	// We need to pass the original subject to the script, not the hostname-prefixed one
	originalSubject := ms.stripHostnamePrefix(requestSubject)
	result, err := runner.ExecuteRequest(ctx, originalSubject, req.Data())

	// Log the request/response
	var responseData []byte
	if result.Success {
		responseData = result.Stdout
	}

	logging.LogRequestResponse(ms.logger, requestSubject, req.Data(), responseData, err)

	// Send response
	if err != nil {
		// Script execution failed
		req.RespondError(fmt.Errorf("script execution failed: %w", err))
		return
	}

	if !result.Success {
		// Script returned non-zero exit code
		errorMsg := fmt.Sprintf("script failed with exit code %d", result.ExitCode)
		if len(result.Stderr) > 0 {
			errorMsg += fmt.Sprintf(": %s", string(result.Stderr))
		}
		req.RespondError(fmt.Errorf("%s", errorMsg))
		return
	}

	// Send successful response
	if err := req.Respond(result.Stdout); err != nil {
		logging.LogError(ms.logger, err, "failed to send response")
	}
}

// stripHostnamePrefix removes the hostname prefix from a subject
// Returns the original subject without the hostname prefix
func (ms *ManagedService) stripHostnamePrefix(subject string) string {
	hostname, err := ms.config.ResolveHostname()
	if err != nil {
		// If we can't resolve hostname, return the subject as-is
		return subject
	}

	prefix := hostname + "."
	if len(subject) > len(prefix) && subject[:len(prefix)] == prefix {
		return subject[len(prefix):]
	}

	// If no prefix found, return as-is
	return subject
}

// String implements fmt.Stringer for better logging
func (ms *ManagedService) String() string {
	// Get first script path for string representation
	for path := range ms.scripts {
		return fmt.Sprintf("ManagedService(%s)", path)
	}
	return fmt.Sprintf("ManagedService(%s)", ms.definition.Name)
}

// NATSRequestWrapper wraps a NATS micro.Request to implement our Request interface
type NATSRequestWrapper struct {
	req micro.Request
}

func (w *NATSRequestWrapper) Subject() string {
	return w.req.Subject()
}

func (w *NATSRequestWrapper) Data() []byte {
	return w.req.Data()
}

func (w *NATSRequestWrapper) Headers() map[string][]string {
	if w.req.Headers() == nil {
		return nil
	}

	// Convert nats.Header to map[string][]string
	headers := make(map[string][]string)
	for k, v := range w.req.Headers() {
		headers[k] = v
	}
	return headers
}

func (w *NATSRequestWrapper) Respond(data []byte) error {
	return w.req.Respond(data)
}

func (w *NATSRequestWrapper) RespondError(err error) error {
	return w.req.Error("500", err.Error(), nil)
}

// Request interface abstracts NATS requests for easier testing
type Request interface {
	Subject() string
	Data() []byte
	Headers() map[string][]string
	Respond(data []byte) error
	RespondError(err error) error
}
