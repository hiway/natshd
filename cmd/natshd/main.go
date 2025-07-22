package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hiway/natshd/internal/config"
	"github.com/hiway/natshd/internal/logging"
	"github.com/hiway/natshd/internal/supervisor"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
)

const (
	AppName    = "natshd"
	AppVersion = "1.0.0"
)

// CLIOptions represents command-line options
type CLIOptions struct {
	ConfigFile  string
	LogLevel    string
	ShowHelp    bool
	ShowVersion bool
}

func main() {
	// Parse command line flags
	options, err := parseFlags(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Handle help and version flags
	if options.ShowHelp {
		showHelp()
		os.Exit(0)
	}

	if options.ShowVersion {
		showVersion()
		os.Exit(0)
	}

	// Set up context with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Run the application
	if err := runApplication(ctx, options); err != nil {
		fmt.Fprintf(os.Stderr, "Application error: %v\n", err)
		os.Exit(1)
	}
}

// parseFlags parses command line arguments
func parseFlags(args []string) (CLIOptions, error) {
	var options CLIOptions

	// Create a new flag set to avoid conflicts with testing
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)

	fs.StringVar(&options.ConfigFile, "config", "config.toml", "Path to configuration file")
	fs.StringVar(&options.LogLevel, "log-level", "", "Override log level (trace, debug, info, warn, error)")
	fs.BoolVar(&options.ShowHelp, "help", false, "Show help information")
	fs.BoolVar(&options.ShowVersion, "version", false, "Show version information")

	// Parse flags
	if err := fs.Parse(args[1:]); err != nil {
		return options, fmt.Errorf("failed to parse flags: %w", err)
	}

	return options, nil
}

// loadConfiguration loads and validates the configuration
func loadConfiguration(configPath string, options CLIOptions) (*config.Config, error) {
	// Load configuration from file
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override log level if provided via CLI
	if options.LogLevel != "" {
		cfg.LogLevel = options.LogLevel
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}

// connectToNATS establishes a connection to the NATS server
func connectToNATS(natsURL string) (*nats.Conn, error) {
	conn, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS server at %s: %w", natsURL, err)
	}

	return conn, nil
}

// setupApplicationLogger configures the application logger
func setupApplicationLogger(cfg *config.Config) (zerolog.Logger, error) {
	logger := logging.SetupLogger(cfg.LogLevel)
	return logger, nil
}

// runApplication runs the main application logic
func runApplication(ctx context.Context, options CLIOptions) error {
	// Load configuration
	cfg, err := loadConfiguration(options.ConfigFile, options)
	if err != nil {
		return err
	}

	// Setup logging
	logger, err := setupApplicationLogger(cfg)
	if err != nil {
		return fmt.Errorf("failed to setup logger: %w", err)
	}

	logger.Info().
		Str("app", AppName).
		Str("version", AppVersion).
		Str("config", options.ConfigFile).
		Str("nats_url", cfg.NatsURL).
		Str("scripts_path", cfg.ScriptsPath).
		Str("log_level", cfg.LogLevel).
		Msg("Starting NATS Shell Daemon")

	// Connect to NATS
	natsConn, err := connectToNATS(cfg.NatsURL)
	if err != nil {
		return err
	}
	defer natsConn.Close()

	logger.Info().
		Str("nats_url", cfg.NatsURL).
		Msg("Connected to NATS server")

	// Create service manager
	serviceManager := supervisor.NewManager(cfg.ScriptsPath, natsConn, logger)

	logger.Info().
		Str("scripts_path", cfg.ScriptsPath).
		Msg("Service manager created")

	// Start the service manager
	logger.Info().Msg("Starting service manager...")
	err = serviceManager.Start(ctx)

	// Log shutdown
	if err != nil && err != context.Canceled {
		logger.Error().Err(err).Msg("Service manager stopped with error")
		return err
	}

	logger.Info().Msg("NATS Shell Daemon stopped gracefully")
	return nil
}

// showHelp displays help information
func showHelp() {
	fmt.Printf(`%s - NATS Shell Micro Service Daemon

USAGE:
    %s [OPTIONS]

OPTIONS:
    -config <path>       Path to configuration file (default: config.toml)
    -log-level <level>   Override log level (trace, debug, info, warn, error)
    -help               Show this help message
    -version            Show version information

DESCRIPTION:
    %s is a specialized service that discovers and hosts NATS microservices 
    from shell scripts on the local filesystem. It monitors a specified 
    directory for shell scripts (*.sh) and automatically registers each 
    script as a unique NATS microservice.

CONFIGURATION:
    The configuration file is in TOML format with the following structure:

    nats_url = "nats://127.0.0.1:4222"
    scripts_path = "./scripts"
    log_level = "info"

EXAMPLES:
    # Start with default config.toml
    %s

    # Use custom configuration file
    %s -config /path/to/my-config.toml

    # Override log level to debug
    %s -log-level debug

    # Show version
    %s -version

SIGNALS:
    SIGINT, SIGTERM    Gracefully shutdown the daemon

`, AppName, AppName, AppName, AppName, AppName, AppName, AppName)
}

// showVersion displays version information
func showVersion() {
	fmt.Printf("%s version %s\n", AppName, AppVersion)
}
