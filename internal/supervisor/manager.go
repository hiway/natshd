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
	"github.com/hiway/natshd/internal/config"
	"github.com/hiway/natshd/internal/logging"
	"github.com/hiway/natshd/internal/service"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/thejerf/suture/v4"
)

// FileEventTracker tracks file events for debouncing
type FileEventTracker struct {
	lastEventTime time.Time
	timer         *time.Timer
	mutex         sync.Mutex
}

// ServiceManager manages all NATS microservices backed by shell scripts
type ServiceManager struct {
	scriptsPath      string
	natsConn         *nats.Conn
	logger           zerolog.Logger
	supervisor       *suture.Supervisor
	services         map[string]*ManagedService     // serviceName -> ManagedService
	serviceTokens    map[string]suture.ServiceToken // serviceName -> token
	scriptToService  map[string]string              // scriptPath -> serviceName
	watcher          *fsnotify.Watcher
	mutex            sync.RWMutex
	debounceTracker  map[string]*FileEventTracker
	debounceInterval time.Duration
	config           *config.Config
	// Track file executable status for detecting permission changes
	fileExecutableStatus  map[string]bool
	permissionCheckTicker *time.Ticker
}

// NewManager creates a new ServiceManager
// NewManager creates a new ServiceManager with the provided config
func NewManager(scriptsPath string, natsConn *nats.Conn, logger zerolog.Logger, cfg config.Config) *ServiceManager {
	// Create a supervisor for managing services
	supervisor := suture.NewSimple("ServiceSupervisor")

	return &ServiceManager{
		scriptsPath:           scriptsPath,
		natsConn:              natsConn,
		logger:                logger.With().Str("component", "manager").Logger(),
		supervisor:            supervisor,
		services:              make(map[string]*ManagedService),
		serviceTokens:         make(map[string]suture.ServiceToken),
		scriptToService:       make(map[string]string),
		debounceTracker:       make(map[string]*FileEventTracker),
		debounceInterval:      500 * time.Millisecond, // 500ms debounce
		config:                &cfg,
		fileExecutableStatus:  make(map[string]bool),
		permissionCheckTicker: time.NewTicker(5 * time.Second), // Check every 5 seconds
	}
}

// Start begins the service manager, discovering services and watching for changes
func (sm *ServiceManager) Start(ctx context.Context) error {
	logging.LogManagerOperation(sm.logger, "starting", map[string]interface{}{
		"scripts_path": sm.scriptsPath,
	})

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

	// Monitor file permission changes (for Linux where fsnotify doesn't support chmod)
	go sm.watchPermissionChanges(ctx)

	// Block until context is cancelled
	<-ctx.Done()

	// Cleanup
	sm.Stop()

	return ctx.Err()
}

// Stop gracefully stops the service manager
func (sm *ServiceManager) Stop() {
	logging.LogManagerOperation(sm.logger, "stopping", nil)

	if sm.watcher != nil {
		sm.watcher.Close()
	}

	if sm.permissionCheckTicker != nil {
		sm.permissionCheckTicker.Stop()
	}

	// Note: Suture supervisor is stopped by cancelling the context passed to Serve()
}

// DiscoverServices scans the scripts directory for valid shell scripts
func (sm *ServiceManager) DiscoverServices() error {
	logging.LogManagerOperation(sm.logger, "discovering", map[string]interface{}{
		"path": sm.scriptsPath,
	})

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

	logging.LogManagerOperation(sm.logger, "discovery_completed", map[string]interface{}{
		"count": len(sm.services),
	})

	return nil
}

// AddService creates and starts a new managed service for the given script
func (sm *ServiceManager) AddService(scriptPath string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	logging.LogManagerOperation(sm.logger, "adding", map[string]interface{}{
		"script": scriptPath,
	})

	// Check if this script is already handled
	if existingServiceName, exists := sm.scriptToService[scriptPath]; exists {
		sm.logger.Warn().
			Str("script", scriptPath).
			Str("service", existingServiceName).
			Msg("Script already handled by service")
		return nil
	}

	// Get service definition from script to determine service name
	runner := service.NewScriptRunner(scriptPath)
	ctx := context.Background()
	definition, err := runner.GetServiceDefinition(ctx)
	if err != nil {
		return fmt.Errorf("failed to get service definition: %w", err)
	}

	serviceName := definition.Name

	// Check if a service with this name already exists
	if existingService, exists := sm.services[serviceName]; exists {
		// Add this script to the existing service
		existingService.AddScript(scriptPath)
		sm.scriptToService[scriptPath] = serviceName

		// Re-initialize the service to pick up the new endpoints
		if err := existingService.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to re-initialize grouped service: %w", err)
		}

		sm.logger.Info().
			Str("script", scriptPath).
			Str("service", serviceName).
			Msg("Added script to existing service group")

		logging.LogServiceLifecycle(sm.logger, "added", serviceName, scriptPath)
		return nil
	}

	// Create new managed service with config
	managedService := NewManagedService(scriptPath, sm.natsConn, sm.logger, *sm.config)
	managedService.AddScript(scriptPath)

	// Initialize the service
	if err := managedService.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize service: %w", err)
	}

	// Add to services map
	sm.services[serviceName] = managedService
	sm.scriptToService[scriptPath] = serviceName

	// Add to supervisor
	token := sm.supervisor.Add(managedService)
	sm.serviceTokens[serviceName] = token
	managedService.serviceToken = token

	logging.LogServiceLifecycle(sm.logger, "added", serviceName, scriptPath)

	return nil
}

// RemoveService stops and removes a managed service
func (sm *ServiceManager) RemoveService(scriptPath string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	logging.LogManagerOperation(sm.logger, "removing", map[string]interface{}{
		"script": scriptPath,
	})

	// Find which service this script belongs to
	serviceName, exists := sm.scriptToService[scriptPath]
	if !exists {
		sm.logger.Warn().
			Str("script", scriptPath).
			Msg("Script not tracked by any service")
		return nil
	}

	// Get the managed service
	managedService, exists := sm.services[serviceName]
	if !exists {
		sm.logger.Warn().
			Str("script", scriptPath).
			Str("service", serviceName).
			Msg("Service does not exist")
		// Clean up orphaned script tracking
		delete(sm.scriptToService, scriptPath)
		return nil
	}

	// Remove script from service
	delete(managedService.scripts, scriptPath)
	delete(sm.scriptToService, scriptPath)

	// If no scripts left in service, remove the entire service
	if len(managedService.scripts) == 0 {
		// Remove from supervisor
		if token, exists := sm.serviceTokens[serviceName]; exists {
			sm.supervisor.Remove(token)
			delete(sm.serviceTokens, serviceName)
		}

		// Remove from services map
		delete(sm.services, serviceName)

		logging.LogServiceLifecycle(sm.logger, "removed", serviceName, scriptPath)
	} else {
		// Re-initialize the service to update endpoints
		ctx := context.Background()
		if err := managedService.Initialize(ctx); err != nil {
			sm.logger.Error().
				Err(err).
				Str("service", serviceName).
				Msg("Failed to re-initialize service after script removal")
		}

		sm.logger.Info().
			Str("script", scriptPath).
			Str("service", serviceName).
			Int("remaining_scripts", len(managedService.scripts)).
			Msg("Removed script from service group")

		logging.LogServiceLifecycle(sm.logger, "script_removed", serviceName, scriptPath)
	}

	return nil
}

// RestartService restarts a managed service
func (sm *ServiceManager) RestartService(scriptPath string) error {
	return sm.RestartServiceGracefully(scriptPath)
}

// RestartServiceGracefully restarts a managed service with proper shutdown
func (sm *ServiceManager) RestartServiceGracefully(scriptPath string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	logging.LogManagerOperation(sm.logger, "restarting", map[string]interface{}{
		"script": scriptPath,
	})

	// Find which service this script belongs to
	serviceName, exists := sm.scriptToService[scriptPath]
	if !exists {
		sm.logger.Warn().
			Str("script", scriptPath).
			Msg("Script not tracked by any service, cannot restart")
		return nil
	}

	// Get the managed service
	managedService, exists := sm.services[serviceName]
	if !exists {
		sm.logger.Warn().
			Str("script", scriptPath).
			Str("service", serviceName).
			Msg("Service does not exist, cannot restart")
		return nil
	}

	// Step 1: Gracefully stop the old NATS service
	if managedService.natsService != nil {
		sm.logger.Debug().
			Str("script", scriptPath).
			Str("service", serviceName).
			Msg("Stopping old NATS service")

		if err := managedService.natsService.Stop(); err != nil {
			sm.logger.Error().
				Err(err).
				Str("script", scriptPath).
				Str("service", serviceName).
				Msg("Error stopping old NATS service")
		}

		// Give some time for the service to properly unregister
		time.Sleep(100 * time.Millisecond)
	}

	// Step 2: Remove old service from supervisor
	if token, exists := sm.serviceTokens[serviceName]; exists {
		sm.supervisor.Remove(token)
		delete(sm.serviceTokens, serviceName)
	}

	// Step 3: Re-initialize the service (it will reload all scripts including the updated one)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := managedService.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to re-initialize service after restart: %w", err)
	}

	// Step 4: Add service back to supervisor
	token := sm.supervisor.Add(managedService)
	sm.serviceTokens[serviceName] = token
	managedService.serviceToken = token

	logging.LogServiceLifecycle(sm.logger, "restarted", serviceName, scriptPath)

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

	logging.LogManagerOperation(sm.logger, "file_watcher_setup", map[string]interface{}{
		"path": sm.scriptsPath,
	})

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

// handleFileEvent processes file system events with debouncing
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
		// File modified - use debouncing to handle multiple rapid events
		sm.handleFileEventDebounced(event.Name, "write")

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

// handleFileEventDebounced handles file events with debouncing to prevent rapid restarts
func (sm *ServiceManager) handleFileEventDebounced(filePath, eventType string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Get or create tracker for this file
	tracker, exists := sm.debounceTracker[filePath]
	if !exists {
		tracker = &FileEventTracker{}
		sm.debounceTracker[filePath] = tracker
	}

	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	// Update last event time
	tracker.lastEventTime = time.Now()

	// Cancel existing timer if it exists
	if tracker.timer != nil {
		tracker.timer.Stop()
	}

	// Create new timer for debounced action
	tracker.timer = time.AfterFunc(sm.debounceInterval, func() {
		sm.executeFileEventAction(filePath, eventType)

		// Clean up tracker after execution
		sm.mutex.Lock()
		delete(sm.debounceTracker, filePath)
		sm.mutex.Unlock()
	})
}

// executeFileEventAction performs the actual file event action after debounce
func (sm *ServiceManager) executeFileEventAction(filePath, eventType string) {
	sm.logger.Debug().
		Str("file", filePath).
		Str("event", eventType).
		Msg("Executing debounced file event action")

	switch eventType {
	case "write":
		// Check if file is still valid after modification
		if sm.IsValidScript(filePath) {
			// Check if script is already tracked
			sm.mutex.RLock()
			_, exists := sm.scriptToService[filePath]
			sm.mutex.RUnlock()

			if exists {
				if err := sm.RestartServiceGracefully(filePath); err != nil {
					sm.logger.Error().
						Err(err).
						Str("script", filePath).
						Msg("Failed to restart service for modified file")
				}
			} else {
				if err := sm.AddService(filePath); err != nil {
					sm.logger.Error().
						Err(err).
						Str("script", filePath).
						Msg("Failed to add service for modified file")
				}
			}
		} else {
			// File is no longer valid, remove service if it exists
			if err := sm.RemoveService(filePath); err != nil {
				sm.logger.Error().
					Err(err).
					Str("script", filePath).
					Msg("Failed to remove service for invalid modified file")
			}
		}
	}
}

// watchPermissionChanges monitors file executable status changes to detect
// when scripts become executable (for Linux where fsnotify doesn't support chmod events)
func (sm *ServiceManager) watchPermissionChanges(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			if sm.permissionCheckTicker != nil {
				sm.permissionCheckTicker.Stop()
			}
			return
		case <-sm.permissionCheckTicker.C:
			sm.checkExecutableStatusChanges()
		}
	}
}

// checkExecutableStatusChanges scans for files that have changed executable status
func (sm *ServiceManager) checkExecutableStatusChanges() {
	// Walk through all files in scripts directory
	err := filepath.Walk(sm.scriptsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if this is a script file
		if !strings.HasSuffix(path, ".sh") {
			return nil
		}

		// Check current executable status
		isExecutable := info.Mode()&0111 != 0

		sm.mutex.Lock()
		previousStatus, existed := sm.fileExecutableStatus[path]
		sm.fileExecutableStatus[path] = isExecutable
		sm.mutex.Unlock()

		// If status changed from non-executable to executable, add the service
		if existed && !previousStatus && isExecutable {
			sm.logger.Info().
				Str("script", path).
				Msg("Script became executable - adding service")

			if err := sm.AddService(path); err != nil {
				sm.logger.Error().
					Err(err).
					Str("script", path).
					Msg("Failed to add service for newly executable script")
			}
		}
		// If status changed from executable to non-executable, remove the service
		if existed && previousStatus && !isExecutable {
			sm.logger.Info().
				Str("script", path).
				Msg("Script became non-executable - removing service")

			if err := sm.RemoveService(path); err != nil {
				sm.logger.Error().
					Err(err).
					Str("script", path).
					Msg("Failed to remove service for non-executable script")
			}
		}

		return nil
	})

	if err != nil {
		sm.logger.Error().
			Err(err).
			Msg("Failed to check executable status changes")
	}
}

// String returns a string representation of the ServiceManager
func (sm *ServiceManager) String() string {
	return fmt.Sprintf("ServiceManager(%s)", sm.scriptsPath)
}
