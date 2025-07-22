package supervisor

import (
	"context"
	"fmt"

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

// ManagedService represents a supervised NATS microservice backed by a shell script
type ManagedService struct {
	scriptPath   string
	natsConn     *nats.Conn
	logger       zerolog.Logger
	runner       ScriptRunner
	definition   service.ServiceDefinition
	natsService  micro.Service
	initialized  bool
	serviceToken suture.ServiceToken
}

// NewManagedService creates a new managed service for the given script
func NewManagedService(scriptPath string, natsConn *nats.Conn, logger zerolog.Logger) *ManagedService {
	serviceLogger := logging.NewContextLogger(logger, "", scriptPath)

	return &ManagedService{
		scriptPath: scriptPath,
		natsConn:   natsConn,
		logger:     serviceLogger,
		runner:     service.NewScriptRunner(scriptPath),
	}
}

// Initialize loads the service definition from the script and validates it
func (ms *ManagedService) Initialize(ctx context.Context) error {
	logging.LogServiceLifecycle(ms.logger, "initializing", "", ms.scriptPath)

	// Get service definition from script
	definition, err := ms.runner.GetServiceDefinition(ctx)
	if err != nil {
		logging.LogError(ms.logger, err, "failed to get service definition")
		return fmt.Errorf("failed to get service definition: %w", err)
	}

	ms.definition = definition

	// Update logger with service name
	ms.logger = logging.NewContextLogger(ms.logger, definition.Name, ms.scriptPath)

	logging.LogServiceLifecycle(ms.logger, "initialized", definition.Name, ms.scriptPath)
	ms.initialized = true

	return nil
}

// Serve implements the suture.Service interface
func (ms *ManagedService) Serve(ctx context.Context) error {
	ms.logger.Info().
		Str("action", "starting").
		Str("service", ms.definition.Name).
		Str("script", ms.scriptPath).
		Msg("Service lifecycle event")

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
		err := service.AddEndpoint(endpoint.Name, micro.HandlerFunc(func(req micro.Request) {
			ms.HandleRequest(&NATSRequestWrapper{req: req})
		}), micro.WithEndpointSubject(endpoint.Subject))
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
		ms.HandleRequest(&natsRequestWrapper{req: req, subject: subject})
	})
}

// HandleRequest processes an incoming NATS request by executing the script
func (ms *ManagedService) HandleRequest(req Request) {
	ctx := context.Background()

	// Execute the script
	result, err := ms.runner.ExecuteRequest(ctx, req.Subject(), req.Data())

	// Log the request/response
	var responseData []byte
	if result.Success {
		responseData = result.Stdout
	}

	logging.LogRequestResponse(ms.logger, req.Subject(), req.Data(), responseData, err)

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
		req.RespondError(fmt.Errorf(errorMsg))
		return
	}

	// Send successful response
	if err := req.Respond(result.Stdout); err != nil {
		logging.LogError(ms.logger, err, "failed to send response")
	}
}

// String implements fmt.Stringer for better logging
func (ms *ManagedService) String() string {
	return fmt.Sprintf("ManagedService(%s)", ms.scriptPath)
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

// natsRequestWrapper implements Request interface for NATS micro requests
type natsRequestWrapper struct {
	req     micro.Request
	subject string
}

func (w *natsRequestWrapper) Subject() string {
	return w.subject
}

func (w *natsRequestWrapper) Data() []byte {
	return w.req.Data()
}

func (w *natsRequestWrapper) Headers() map[string][]string {
	headers := w.req.Headers()
	if headers == nil {
		return nil
	}

	result := make(map[string][]string)
	for key, values := range headers {
		result[key] = values
	}
	return result
}

func (w *natsRequestWrapper) Respond(data []byte) error {
	return w.req.Respond(data)
}

func (w *natsRequestWrapper) RespondError(err error) error {
	return w.req.Error("500", err.Error(), nil)
}
