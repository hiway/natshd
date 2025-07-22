package supervisor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hiway/natshd/internal/logging"
	"github.com/hiway/natshd/internal/service"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/thejerf/suture/v4"
)

// ServiceManager manages all NATS microservices backed by shell scripts
type ServiceManager struct {
	scriptsPath   string
	natsConn      *nats.Conn
	logger        zerolog.Logger
	supervisor    *suture.Supervisor
	services      map[string]*ManagedService
	serviceTokens map[string]suture.ServiceToken
	watcher       *fsnotify.Watcher
	mutex         sync.RWMutex
}

// NewManager creates a new ServiceManager
func NewManager(scriptsPath string, natsConn *nats.Conn, logger zerolog.Logger) *ServiceManager {
	// Create a supervisor for managing services
	supervisor := suture.NewSimple("ServiceSupervisor")

	return &ServiceManager{
		scriptsPath:   scriptsPath,
		natsConn:      natsConn,
		logger:        logger.With().Str("component", "manager").Logger(),
		supervisor:    supervisor,
		services:      make(map[string]*ManagedService),
		serviceTokens: make(map[string]suture.ServiceToken),
	}
}

// Start begins the service manager, discovering services and watching for changes
func (sm *ServiceManager) Start(ctx context.Context) error {
	sm.logger.Info().
		Str("action", "starting").
		Str("scripts_path", sm.scriptsPath).
		Msg("Service manager starting")

	// Discover existing services
	if err := sm.DiscoverServices(); err != nil {
		return fmt.Errorf("failed to discover services: %w", err)
	}

	// Set up file watcher
	if err := sm.setupFileWatcher(); err != nil {
		return fmt.Errorf("failed to setup file watcher: %w", err)
	}

	// Start the supervisor
	go sm.supervisor.Serve(ctx)

	// Watch for file changes
	go sm.watchFileChanges(ctx)

	// Block until context is cancelled
	<-ctx.Done()

	// Cleanup
	sm.Stop()

	return ctx.Err()
}

// Stop gracefully stops the service manager
func (sm *ServiceManager) Stop() {
	sm.logger.Info().
		Str("action", "stopping").
		Msg("Service manager stopping")

	if sm.watcher != nil {
		sm.watcher.Close()
	}

	// Note: Suture supervisor is stopped by cancelling the context passed to Serve()
}

// DiscoverServices scans the scripts directory for valid shell scripts
func (sm *ServiceManager) DiscoverServices() error {
	sm.logger.Info().
		Str("action", "discovering").
		Str("path", sm.scriptsPath).
		Msg("Discovering services")

	// Check if scripts directory exists
	if _, err := os.Stat(sm.scriptsPath); os.IsNotExist(err) {
		sm.logger.Warn().
			Str("path", sm.scriptsPath).
			Msg("Scripts directory does not exist")
		return nil
	}

	// Walk through the scripts directory
	err := filepath.Walk(sm.scriptsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			sm.logger.Error().
				Err(err).
				Str("path", path).
				Msg("Error accessing file during discovery")
			return nil // Continue walking
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if it's a valid script
		if sm.IsValidScript(path) {
			if err := sm.AddService(path); err != nil {
				sm.logger.Error().
					Err(err).
					Str("script", path).
					Msg("Failed to add discovered service")
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk scripts directory: %w", err)
	}

	sm.logger.Info().
		Int("count", len(sm.services)).
		Msg("Service discovery completed")

	return nil
}

// AddService creates and starts a new managed service for the given script
func (sm *ServiceManager) AddService(scriptPath string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.logger.Info().
		Str("action", "adding").
		Str("script", scriptPath).
		Msg("Adding service")

	// Check if service already exists
	if _, exists := sm.services[scriptPath]; exists {
		sm.logger.Warn().
			Str("script", scriptPath).
			Msg("Service already exists")
		return nil
	}

	// Create managed service
	managedService := NewManagedService(scriptPath, sm.natsConn, sm.logger)

	// Initialize the service
	ctx := context.Background()
	if err := managedService.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize service: %w", err)
	}

	// Add to services map
	sm.services[scriptPath] = managedService

	// Add to supervisor
	token := sm.supervisor.Add(managedService)
	sm.serviceTokens[scriptPath] = token
	managedService.serviceToken = token

	logging.LogServiceLifecycle(sm.logger, "added", managedService.definition.Name, scriptPath)

	return nil
}

// RemoveService stops and removes a managed service
func (sm *ServiceManager) RemoveService(scriptPath string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.logger.Info().
		Str("action", "removing").
		Str("script", scriptPath).
		Msg("Removing service")

	// Check if service exists
	managedService, exists := sm.services[scriptPath]
	if !exists {
		sm.logger.Warn().
			Str("script", scriptPath).
			Msg("Service does not exist")
		return nil
	}

	// Remove from supervisor
	if token, exists := sm.serviceTokens[scriptPath]; exists {
		sm.supervisor.Remove(token)
		delete(sm.serviceTokens, scriptPath)
	}

	// Remove from services map
	delete(sm.services, scriptPath)

	logging.LogServiceLifecycle(sm.logger, "removed", managedService.definition.Name, scriptPath)

	return nil
}

// RestartService restarts a managed service
func (sm *ServiceManager) RestartService(scriptPath string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.logger.Info().
		Str("action", "restarting").
		Str("script", scriptPath).
		Msg("Restarting service")

	// Check if service exists
	_, exists := sm.services[scriptPath]
	if !exists {
		sm.logger.Warn().
			Str("script", scriptPath).
			Msg("Service does not exist, cannot restart")
		return nil
	}

	// Remove old service from supervisor
	if token, exists := sm.serviceTokens[scriptPath]; exists {
		sm.supervisor.Remove(token)
		delete(sm.serviceTokens, scriptPath)
	}

	// Create new managed service
	managedService := NewManagedService(scriptPath, sm.natsConn, sm.logger)

	// Initialize the service
	ctx := context.Background()
	if err := managedService.Initialize(ctx); err != nil {
		// Remove from services map if initialization failed
		delete(sm.services, scriptPath)
		return fmt.Errorf("failed to initialize restarted service: %w", err)
	}

	// Update services map
	sm.services[scriptPath] = managedService

	// Add new service to supervisor
	token := sm.supervisor.Add(managedService)
	sm.serviceTokens[scriptPath] = token
	managedService.serviceToken = token

	logging.LogServiceLifecycle(sm.logger, "restarted", managedService.definition.Name, scriptPath)

	return nil
}

// IsValidScript checks if a file is a valid executable shell script
func (sm *ServiceManager) IsValidScript(filePath string) bool {
	// Check file extension
	if !strings.HasSuffix(filePath, ".sh") {
		return false
	}

	// Check if file is executable
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	if info.Mode()&0111 == 0 {
		return false // Not executable
	}

	// Try to get service definition to validate it's a proper service script
	runner := service.NewScriptRunner(filePath)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // 5 second timeout
	defer cancel()

	_, err = runner.GetServiceDefinition(ctx)
	return err == nil
}

// setupFileWatcher creates a file system watcher for the scripts directory
func (sm *ServiceManager) setupFileWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	sm.watcher = watcher

	// Add the scripts directory to the watcher
	err = watcher.Add(sm.scriptsPath)
	if err != nil {
		return fmt.Errorf("failed to watch scripts directory: %w", err)
	}

	sm.logger.Info().
		Str("path", sm.scriptsPath).
		Msg("File watcher setup completed")

	return nil
}

// watchFileChanges monitors file system events and updates services accordingly
func (sm *ServiceManager) watchFileChanges(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-sm.watcher.Events:
			if !ok {
				return
			}

			sm.handleFileEvent(event)

		case err, ok := <-sm.watcher.Errors:
			if !ok {
				return
			}

			sm.logger.Error().
				Err(err).
				Msg("File watcher error")
		}
	}
}

// handleFileEvent processes file system events
func (sm *ServiceManager) handleFileEvent(event fsnotify.Event) {
	sm.logger.Debug().
		Str("file", event.Name).
		Str("operation", event.Op.String()).
		Msg("File event received")

	// Only process shell scripts
	if !strings.HasSuffix(event.Name, ".sh") {
		return
	}

	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		// New file created
		if sm.IsValidScript(event.Name) {
			if err := sm.AddService(event.Name); err != nil {
				sm.logger.Error().
					Err(err).
					Str("script", event.Name).
					Msg("Failed to add service for created file")
			}
		}

	case event.Op&fsnotify.Write == fsnotify.Write:
		// File modified
		if sm.IsValidScript(event.Name) {
			if err := sm.RestartService(event.Name); err != nil {
				sm.logger.Error().
					Err(err).
					Str("script", event.Name).
					Msg("Failed to restart service for modified file")
			}
		} else {
			// File is no longer valid, remove service if it exists
			if err := sm.RemoveService(event.Name); err != nil {
				sm.logger.Error().
					Err(err).
					Str("script", event.Name).
					Msg("Failed to remove service for invalid modified file")
			}
		}

	case event.Op&fsnotify.Remove == fsnotify.Remove:
		// File deleted
		if err := sm.RemoveService(event.Name); err != nil {
			sm.logger.Error().
				Err(err).
				Str("script", event.Name).
				Msg("Failed to remove service for deleted file")
		}

	case event.Op&fsnotify.Rename == fsnotify.Rename:
		// File renamed (treated as deletion)
		if err := sm.RemoveService(event.Name); err != nil {
			sm.logger.Error().
				Err(err).
				Str("script", event.Name).
				Msg("Failed to remove service for renamed file")
		}
	}
}

// String returns a string representation of the ServiceManager
func (sm *ServiceManager) String() string {
	return fmt.Sprintf("ServiceManager(%s)", sm.scriptsPath)
}
