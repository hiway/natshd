# natshd

NATS Shell Daemon - Shell scripts as NATS microservices

`natshd` automatically discovers shell scripts in a directory and exposes them as NATS microservices. 
Drop executable shell scripts into the configured directory, and they become instantly available as networked services.

**Service Grouping**: Multiple scripts that define the same service name are automatically grouped under a single microservice, allowing you to organize related functionality across multiple script files while maintaining efficient NATS service registration.

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

**Service Grouping**: Scripts that return the same service name in their `info` response are automatically grouped together under a single NATS microservice. This allows you to split complex services across multiple files while maintaining a single service registration.

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

### Example: Service Grouping

You can create multiple script files that share the same service name:

**scripts/system-facts.sh**:

```bash
#!/bin/bash
if [[ "$1" == "info" ]]; then
    cat <<EOF
{
    "name": "SystemService",
    "version": "1.0.0",
    "description": "System information and monitoring",
    "endpoints": [
        {
            "name": "GetFacts",
            "subject": "system.facts"
        }
    ]
}
EOF
    exit 0
fi

# Handle system.facts requests
echo '{"hostname": "'$(hostname)'", "uptime": "'$(uptime -p)'"}'
```

**scripts/system-hardware.sh**:

```bash
#!/bin/bash
if [[ "$1" == "info" ]]; then
    cat <<EOF
{
    "name": "SystemService",
    "version": "1.0.0", 
    "description": "System information and monitoring",
    "endpoints": [
        {
            "name": "GetHardware",
            "subject": "system.hardware"
        }
    ]
}
EOF
    exit 0
fi

# Handle system.hardware requests  
echo '{"cpu": "'$(nproc)'", "memory": "'$(free -h | awk '/^Mem:/ {print $2}')'"}'
```

Both scripts will be grouped under a single "SystemService" microservice with endpoints for both `system.facts` and `system.hardware`.

### Make Scripts Executable

```bash
chmod +x scripts/greeting.sh 
chmod +x scripts/system-*.sh 
```

> `natshd` will automatically load/reload/remove scripts based on filesystem events.

## Using Your Services

### Discover Available Services

```bash
# List all discovered services and their endpoints
nats micro list

# Get detailed info about a specific service
nats micro info SystemService
```

**Note**: With service grouping, you'll see services organized by their declared service name rather than individual script files. Multiple scripts defining the same service name contribute their endpoints to a single service registration.

### Calling Services

```bash
# Send a request to the system facts endpoint
nats req system.facts '{}'

# Response:
# {
#   "hostname": "myserver",
#   "uptime": "up 2 days, 4 hours"
# }

# Send a request to the system hardware endpoint (from the same SystemService)
nats req system.hardware '{}'

# Response:
# {
#   "cpu": "8",
#   "memory": "16Gi"
# }

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
nats micro ping SystemService

# Monitor real-time service stats  
nats micro stats SystemService

# View service logs (if natshd is running with debug logging)
./natshd -log-level debug
```

## What's Included

The `scripts/` directory contains several example services to get you started:

- **greeting.sh** - Simple greeting service with endpoints for personalized greetings and farewells
- **uptime.sh** - System monitoring service providing uptime and load average information
- **system-facts.sh** - Comprehensive system information (OS, uptime, CPU, memory, etc.)
- **system-hardware.sh** - Hardware discovery (CPU, memory, storage, graphics, network interfaces)
- **system-kernel.sh** - Kernel version, modules, and system parameters
- **system-network.sh** - Network configuration, interfaces, routes, DNS, and listening ports
- **system-processes.sh** - Running processes, resource usage, and process statistics
- **system-storage.sh** - Storage and filesystem information, block devices, and I/O stats
- **system-users.sh** - User accounts, groups, login info, sudo/admin users, and password policy

## Features

- **Automatic Discovery**: Drop scripts into a directory, they instantly become services
- **Service Grouping**: Multiple scripts with the same service name are automatically grouped under a single microservice for efficient resource usage
- **Dynamic Registration**: Services automatically register with NATS on startup
- **Hot Reload**: Modify scripts and services update automatically  
- **Structured Logging**: JSON logging with configurable levels
- **Health Monitoring**: Built-in health checks and monitoring via NATS micro protocol
- **Resilient**: Supervised service lifecycle management with automatic restarts
- **Simple Protocol**: Scripts just need to handle `info` requests and process JSON from stdin
- **Efficient**: Service grouping reduces NATS registration overhead while maintaining full functionality

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
