#!/usr/bin/env bash
# System uptime microservice script for natshd
# This script provides system uptime information

# Cross-platform OS detection
detect_os() {
    case "$(uname -s)" in
        Linux) OS="linux" ;;
        FreeBSD) OS="freebsd" ;;
        Darwin) OS="macos" ;;
        *) OS="unknown" ;;
    esac
}

# Define the service when called with info argument
if [[ "$1" == "info" ]]; then
    cat << 'EOF'
{
    "name": "UptimeService",
    "description": "A system monitoring microservice that provides uptime information",
    "version": "1.0.0",
    "endpoints": [
        {
            "name": "GetUptime",
            "subject": "system.uptime",
            "description": "Returns system uptime information"
        },
        {
            "name": "GetLoadAverage",
            "subject": "system.load",
            "description": "Returns system load average"
        }
    ]
}
EOF
    exit 0
fi

# Read JSON request from stdin
REQUEST=$(cat)

# Extract the subject from the first argument
SUBJECT="$1"

# Log the incoming request for debugging
echo "Processing request for subject: $SUBJECT" >&2
echo "Request data: $REQUEST" >&2

# Detect operating system
detect_os

# Determine endpoint from subject
case "$SUBJECT" in
    "system.uptime")
        # Get uptime information
        UPTIME_RAW=$(uptime)
        
        # Get uptime seconds - cross platform
        case "$OS" in
            linux)
                UPTIME_SECONDS=$(awk '{print int($1)}' /proc/uptime)
                ;;
            freebsd|macos)
                # Use sysctl to get boot time and calculate uptime
                BOOT_TIME=$(sysctl -n kern.boottime | grep -o "sec = [0-9]*" | head -1 | awk '{print $3}')
                CURRENT_TIME=$(date +%s)
                UPTIME_SECONDS=$((CURRENT_TIME - BOOT_TIME))
                ;;
            *)
                # Fallback: try to parse uptime output (less reliable)
                UPTIME_SECONDS=0
                ;;
        esac
        
        # Calculate days, hours, minutes
        DAYS=$((UPTIME_SECONDS / 86400))
        HOURS=$(((UPTIME_SECONDS % 86400) / 3600))
        MINUTES=$(((UPTIME_SECONDS % 3600) / 60))
        
        # Generate response
        cat << EOF
{
    "success": true,
    "uptime": {
        "raw": "$UPTIME_RAW",
        "seconds": $UPTIME_SECONDS,
        "formatted": "${DAYS}d ${HOURS}h ${MINUTES}m",
        "days": $DAYS,
        "hours": $HOURS,
        "minutes": $MINUTES
    },
    "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "service": "UptimeService",
    "endpoint": "GetUptime",
    "subject": "$SUBJECT",
    "os": "$OS"
}
EOF
        ;;
    
    "system.load")
        # Get load average information - cross platform
        case "$OS" in
            linux)
                # Parse from uptime command
                LOAD_RAW=$(uptime | grep -oE 'load average: [0-9]+\.[0-9]+, [0-9]+\.[0-9]+, [0-9]+\.[0-9]+')
                LOAD_1MIN=$(echo "$LOAD_RAW" | awk '{print $3}' | sed 's/,//')
                LOAD_5MIN=$(echo "$LOAD_RAW" | awk '{print $4}' | sed 's/,//')
                LOAD_15MIN=$(echo "$LOAD_RAW" | awk '{print $5}')
                ;;
            freebsd|macos)
                # Use sysctl for more reliable load average
                LOAD_OUTPUT=$(sysctl -n vm.loadavg)
                LOAD_RAW="load averages: $(echo "$LOAD_OUTPUT" | awk '{print $2", "$3", "$4}')"
                LOAD_1MIN=$(echo "$LOAD_OUTPUT" | awk '{print $2}')
                LOAD_5MIN=$(echo "$LOAD_OUTPUT" | awk '{print $3}')
                LOAD_15MIN=$(echo "$LOAD_OUTPUT" | awk '{print $4}')
                ;;
            *)
                LOAD_RAW=""
                LOAD_1MIN=""
                LOAD_5MIN=""
                LOAD_15MIN=""
                ;;
        esac
        
        # Get CPU count for context - cross platform
        case "$OS" in
            linux)
                CPU_COUNT=$(nproc)
                ;;
            freebsd|macos)
                CPU_COUNT=$(sysctl -n hw.ncpu)
                ;;
            *)
                CPU_COUNT=0
                ;;
        esac
        
        # Generate response
        cat << EOF
{
    "success": true,
    "load_average": {
        "raw": "$LOAD_RAW",
        "1min": "$LOAD_1MIN",
        "5min": "$LOAD_5MIN",
        "15min": "$LOAD_15MIN",
        "cpu_count": $CPU_COUNT
    },
    "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "service": "UptimeService",
    "endpoint": "GetLoadAverage",
    "subject": "$SUBJECT",
    "os": "$OS"
}
EOF
        ;;
    
    *)
        # Unknown subject
        cat << EOF
{
    "success": false,
    "error": "Unknown subject: $SUBJECT",
    "available_subjects": ["system.uptime", "system.load"],
    "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "service": "UptimeService"
}
EOF
        exit 1
        ;;
esac
