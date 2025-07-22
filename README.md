# natshd

NATS Shell Daemon - Shell scripts as NATS microservices

`natshd` automatically discovers shell scripts in a directory and exposes them as NATS microservices. 
Drop executable shell scripts into the configured directory, and they become instantly available as networked services.

## Quick Start

### Prerequisites

- Go 1.23 or later
- NATS Server running (default: `nats://127.0.0.1:4222`)

### Installation

```bash
git clone https://github.com/hiway/natshd.git
cd natshd
make build
```

### Configuration

Copy the sample configuration and customize as needed:

```bash
cp config.toml.sample config.toml
```

The default configuration:

```toml
nats_url = "nats://127.0.0.1:4222"
scripts_path = "./scripts"
log_level = "info"
```

### Running natshd

```bash
# Start the daemon
./natshd

# Or run with custom config
./natshd -config /path/to/config.toml

# Enable debug logging
./natshd -log-level debug
```

## Writing Service Scripts

Each shell script becomes a microservice by implementing two simple requirements:

1. **Service Discovery**: Respond to `info` argument with service metadata
2. **Request Handling**: Process requests from stdin and respond via stdout

### Example: Simple Greeting Service

```bash
#!/bin/bash

# Service definition (required)
if [[ "$1" == "info" ]]; then
    cat <<EOF
{
    "name": "GreetingService",
    "version": "1.0.0", 
    "description": "A simple greeting service",
    "endpoints": [
        {
            "name": "Greet",
            "subject": "greeting.hello",
            "description": "Generate personalized greetings"
        }
    ]
}
EOF
    exit 0
fi

# Request handling
SUBJECT="$1"
REQUEST=$(cat)

# Extract name from JSON request
NAME=$(echo "$REQUEST" | jq -r '.name // "World"')

# Generate response
cat <<EOF
{
    "success": true,
    "message": "Hello, $NAME!",
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
```

### Make It Executable

```bash
chmod +x scripts/greeting.sh
```

## Using Your Services

### Discover Available Services

```bash
# List all discovered services and their endpoints
nats micro list

# Get detailed info about a specific service
nats micro info GreetingService
```

### Calling Services

```bash
# Send a request to the greeting service
nats req greeting.hello '{"name": "Alice"}'

# Response:
# {
#   "success": true,
#   "message": "Hello, Alice!",
#   "timestamp": "2024-07-23T10:30:00Z"
# }
```

### Service Health and Monitoring

```bash
# Check service health
nats micro ping GreetingService

# Monitor real-time service stats  
nats micro stats GreetingService

# View service logs (if natshd is running with debug logging)
./natshd -log-level debug
```

## What's Included

The `scripts/` directory contains several example services to get you started:

- **greeting.sh** - Simple greeting service with multiple endpoints
- **uptime.sh** - System monitoring service (uptime, load average)
- **json-processor.sh** - JSON validation, transformation, and formatting
- **file-ops.sh** - File operations and utilities
- **uname.sh** - System information service

## Features

- **Automatic Discovery**: Drop scripts into a directory, they instantly become services
- **Dynamic Registration**: Services automatically register with NATS on startup
- **Hot Reload**: Modify scripts and services update automatically
- **Structured Logging**: JSON logging with configurable levels
- **Health Monitoring**: Built-in health checks and monitoring via NATS micro protocol
- **Resilient**: Supervised service lifecycle management
- **Simple Protocol**: Scripts just need to handle `info` requests and process JSON from stdin

## Development

```bash
# Run tests
make test

# Run with debug logging
make debug

# Build binary
make build

# Run development version
make run
```

## License

See [LICENSE](LICENSE) file for details.
