# natshd

NATS Shell Daemon - Shell scripts as NATS microservices

`natshd` automatically discovers shell scripts in a directory and exposes them as NATS microservices. 
Drop executable shell scripts into the configured directory, and they become instantly available as networked services.

**Why?**: I use nats-cli when prototyping [NATS](https://nats.io) services, `natshd` makes it easy to prototype nats micro services using shell scripts and keep them available beyond a terminal session. Some services that I run on my shelf-hosted server need no more than a shell script, and natshd is good enough for the task.

**Status**: As of this writing, the idea and the implemented project are not even a day old - I may update this for adding authentication, env vars to not require `jq` to parse json and a handful other quality-of-life enhancements. This is not time-tested, nor production-ready, as with any code that you may find online - it is your responsibility to verify it does what you think it does.

> Note: `natshd` is developed using LLM Assisted Coding - if you're allergic, this is your cue.

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
hostname = "auto"
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

## Hostname Targeting

`natshd` automatically prefixes all NATS subjects with the system hostname, enabling you to target specific nodes or groups of nodes in a multi-host deployment.

### Configuration Options

- `hostname = "auto"` (default) - Automatically uses the system hostname
- `hostname = "web01"` - Use an explicit hostname
- `hostname = "production-server"` - Use any custom identifier

### How It Works

With hostname prefixing, your service subjects become:

```bash
# Original subject: system.facts
# Becomes: hostname.system.facts

# Examples:
# web01.system.facts
# db-server.system.facts  
# production-api.greeting.hello
```

### Multi-Host Usage

```bash
# Target a specific host
nats req web01.system.facts '{}'

# Target all hosts with wildcard
nats req "*.system.facts" '{}'

# Target hosts matching a pattern
nats req "web*.system.facts" '{}'
nats req "production-*.system.facts" '{}'
```

This enables powerful deployment patterns:

- **Development clusters**: Each developer's machine has its own hostname prefix
- **Environment separation**: `staging-web01`, `production-web01`, etc.
- **Service discovery**: Find all instances of a service across your infrastructure
- **Rolling deployments**: Target specific subsets of nodes during updates

## Writing Service Scripts


Each shell script becomes a microservice by implementing two simple requirements:

1. **Service Discovery**: Respond to `info` argument with service metadata
2. **Request Handling**: Process requests from stdin and respond via stdout

### Example: Simple Greeting Service

```bash
#!/usr/bin/env bash

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

**Service Grouping**: Scripts that return the same service name in their `info` response are automatically grouped together under a single NATS microservice. This allows you to split complex services across multiple files while maintaining a single service registration.

**scripts/system-facts.sh**:

```bash
#!/usr/bin/env bash
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
#!/usr/bin/env bash
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

### Example: Metadata

You can include a `metadata` field in each endpoint definition to describe parameters, types, and other details. This metadata will be visible in `nats micro info` output and is passed through to the NATS microservice registry.

```bash
#!/usr/bin/env bash

# Service definition with endpoint metadata
if [[ "$1" == "info" ]]; then
    cat <<EOF
{
    "name": "GreetingService",
    "version": "1.0.0",
    "description": "A greeting service with endpoint metadata",
    "endpoints": [
        {
            "name": "Greet",
            "subject": "greeting.greet",
            "description": "Generates a personalized greeting message",
            "metadata": {
                "parameters": {
                    "name": {
                        "type": "string",
                        "description": "The name of the person to greet",
                        "default": "World"
                    },
                    "greeting": {
                        "type": "string",
                        "description": "The greeting message to use",
                        "default": "Hello"
                    }
                }
            }
        }
    ]
}
EOF
    exit 0
fi

# Request handling
SUBJECT="$1"
REQUEST=$(cat)

# Extract parameters from JSON request
NAME=$(echo "$REQUEST" | jq -r '.name // "World"')
GREETING=$(echo "$REQUEST" | jq -r '.greeting // "Hello"')

# Generate response
cat <<EOF
{
    "success": true,
    "message": "$GREETING, $NAME!",
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "GreetingService",
    "endpoint": "Greet",
    "subject": "$SUBJECT"
}
EOF
```

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
# Send a request to the system facts endpoint (with hostname prefix)
nats req $(hostname).system.facts '{}'

# Response:
# {
#   "hostname": "myserver",
#   "uptime": "up 2 days, 4 hours"
# }

# Send a request to the system hardware endpoint (from the same SystemService)
nats req $(hostname).system.hardware '{}'

# Response:
# {
#   "cpu": "8",
#   "memory": "16Gi"
# }

# Send a request to the greeting service
nats req $(hostname).greeting.hello '{"name": "Alice"}'

# Response:
# {
#   "success": true,
#   "message": "Hello, Alice!",
#   "timestamp": "2024-07-23T10:30:00Z"
# }

# Target a specific host by name
nats req web01.system.facts '{}'

# Target all hosts with a wildcard
nats req "*.system.facts" '{}'
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
