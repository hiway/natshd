---
description: "Instructions for debugging natshd scripts on remote FreeBSD systems"
applyTo: "**/**"
---
# Debugging Remote FreeBSD Scripts for natshd

This document provides clear instructions for LLMs to effectively debug, deploy, and test natshd scripts on remote FreeBSD systems.

## Overview

The natshd daemon automatically reloads scripts from the `/usr/local/share/natshd/scripts/` directory without requiring a restart. This makes development and testing very efficient.

## Development Workflow

### 1. Script Deployment

Scripts must be deployed to the remote FreeBSD system for testing:

```bash
# Copy script to temporary location (regular user access)
scp scripts/script-name.sh dev:/tmp/script-name.sh

# Move to final location and set permissions (requires sudo)
ssh dev "sudo cp /tmp/script-name.sh /usr/local/share/natshd/scripts/script-name.sh && sudo chmod +x /usr/local/share/natshd/scripts/script-name.sh"
```

**Important:** natshd automatically detects and reloads new/updated scripts. No daemon restart is required.

### 2. Testing Scripts

#### Direct Script Testing

Test scripts directly to debug their output:

```bash
# Test script info endpoint
ssh dev "sudo -u natshd /usr/local/share/natshd/scripts/script-name.sh info"

# Test script functionality with subject
ssh dev "echo '{}' | sudo -u natshd /usr/local/share/natshd/scripts/script-name.sh subject.name"
```

#### NATS Endpoint Testing

Test through the NATS system (end-to-end testing):

```bash
# Test endpoint (replace xostd with appropriate prefix)
nats req xostd.subject.name '{}'

# Test with specific payload
nats req xostd.subject.name '{"key": "value"}'
```

## FreeBSD-Specific Debugging

### Common FreeBSD Commands for System Information

```bash
# System information
ssh dev "uname -a"                    # Kernel and system info
ssh dev "sysctl kern.version"         # Detailed kernel version
ssh dev "sysctl kern.boottime"        # Boot time (for uptime calculations)
ssh dev "sysctl hw.ncpu"              # CPU count
ssh dev "sysctl hw.physmem"           # Physical memory

# Process and service management
ssh dev "ps aux | grep natshd"        # Check natshd processes
ssh dev "service natshd status"       # Service status
ssh dev "sockstat -l | grep nats"     # Check NATS listening ports

# File system and permissions
ssh dev "ls -la /usr/local/share/natshd/scripts/"  # Script directory
ssh dev "ls -la /usr/local/etc/natshd/"            # Config directory
```

### FreeBSD vs Linux Differences

Key differences to be aware of when debugging:

| Aspect | FreeBSD | Linux |
|--------|---------|-------|
| Boot time | `sysctl kern.boottime` | `/proc/stat` (btime) |
| Load average | `sysctl vm.loadavg` | `/proc/loadavg` |
| Memory info | `sysctl hw.physmem` | `/proc/meminfo` |
| CPU count | `sysctl hw.ncpu` | `/proc/cpuinfo` or `nproc` |
| Process list | `ps aux` | `ps aux` (same) |
| Date from timestamp | `date -r TIMESTAMP` | `date -d @TIMESTAMP` |

## Common Debugging Patterns

### 1. Boot Time Issues

FreeBSD boot time format: `{ sec = 1751810055, usec = 907914 } Sun Jul  6 13:54:15 2025`

**Correct extraction:**

```bash
ssh dev "sysctl -n kern.boottime | grep -o 'sec = [0-9]*' | head -1 | awk '{print \$3}'"
```

**Verify the timestamp:**

```bash
ssh dev "date -r 1751810055"  # Should show correct boot date
```

### 2. JSON Output Validation

Always validate JSON output structure:

```bash
# Test and validate JSON
ssh dev "echo '{}' | sudo -u natshd /usr/local/share/natshd/scripts/script-name.sh subject.name | jq ."
```

### 3. Script Permissions and User Context

Scripts run as the `natshd` user, which may have limited permissions:

```bash
# Check what natshd user can access
ssh dev "sudo -u natshd id"
ssh dev "sudo -u natshd whoami"
ssh dev "sudo -u natshd ls -la /usr/local/share/natshd/scripts/"
```

## Service Discovery and Status

### Check Available Services

```bash
# List all microservices
nats micro ls

# Get detailed service info
nats micro info SERVICE_ID
```

### Service Registration Process

1. Scripts must have an `info` endpoint that returns service metadata
2. natshd automatically registers scripts as microservices
3. Endpoints are prefixed (e.g., `xostd.` on FreeBSD)

## Error Patterns and Solutions

### Common Error: "Permission denied"

**Problem:** Script deployment fails due to permissions

**Solution:**

```bash
# Always use two-step deployment
scp script.sh dev:/tmp/script.sh
ssh dev "sudo cp /tmp/script.sh /usr/local/share/natshd/scripts/ && sudo chmod +x /usr/local/share/natshd/scripts/script.sh"
```

### Common Error: "Command not found"

**Problem:** FreeBSD command differences

**Investigation:**

```bash
ssh dev "which COMMAND"           # Check if command exists
ssh dev "man COMMAND"             # Check command syntax
ssh dev "COMMAND --help"          # Get help if available
```

### Common Error: Incorrect timestamp parsing

**Problem:** Date/time parsing differs between systems

**FreeBSD date command patterns:**

```bash
ssh dev "date -r TIMESTAMP"       # Convert Unix timestamp to date
ssh dev "date +%s"                # Get current Unix timestamp
ssh dev "date -u +%Y-%m-%dT%H:%M:%SZ"  # ISO 8601 UTC format
```

## Testing Workflow Example

1. **Develop locally** - Edit script in workspace
2. **Deploy** - Copy to remote FreeBSD system
3. **Test directly** - Run script manually to check output
4. **Test via NATS** - Use `nats req` to test end-to-end
5. **Debug issues** - Use FreeBSD-specific commands to investigate
6. **Iterate** - Repeat until working correctly

## Quick Reference Commands

```bash
# Deploy script
scp scripts/example.sh dev:/tmp/ && ssh dev "sudo cp /tmp/example.sh /usr/local/share/natshd/scripts/ && sudo chmod +x /usr/local/share/natshd/scripts/example.sh"

# Test script directly
ssh dev "echo '{}' | sudo -u natshd /usr/local/share/natshd/scripts/example.sh subject.name"

# Test via NATS
nats req xostd.subject.name '{}'

# Check service status
nats micro ls

# Debug FreeBSD specifics
ssh dev "sysctl kern.boottime"
ssh dev "date -r \$(sysctl -n kern.boottime | grep -o 'sec = [0-9]*' | head -1 | awk '{print \$3}')"
```

Remember: natshd automatically reloads scripts, so no service restart is needed after deployment!
