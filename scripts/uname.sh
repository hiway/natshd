#!/bin/bash
# System information microservice script for natshd
# This script provides system information via uname and related commands

# Define the service when called with info argument
if [[ "$1" == "info" ]]; then
    cat << 'EOF'
{
    "name": "SystemInfoService",
    "description": "A system information microservice that provides OS and hardware details",
    "version": "1.0.0",
    "endpoints": [
        {
            "name": "GetSystemInfo",
            "subject": "system.info",
            "description": "Returns comprehensive system information"
        },
        {
            "name": "GetKernelInfo",
            "subject": "system.kernel",
            "description": "Returns kernel version and architecture information"
        },
        {
            "name": "GetHostname",
            "subject": "system.hostname",
            "description": "Returns system hostname"
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

# Determine endpoint from subject
case "$SUBJECT" in
    "system.info")
        # Get comprehensive system information
        KERNEL_NAME=$(uname -s)
        KERNEL_RELEASE=$(uname -r)
        KERNEL_VERSION=$(uname -v)
        MACHINE=$(uname -m)
        PROCESSOR=$(uname -p)
        HARDWARE_PLATFORM=$(uname -i)
        OS=$(uname -o)
        HOSTNAME=$(uname -n)
        
        # Get additional info if available
        if command -v lsb_release &> /dev/null; then
            DISTRO=$(lsb_release -d | cut -f2)
        elif [ -f /etc/os-release ]; then
            DISTRO=$(grep PRETTY_NAME /etc/os-release | cut -d'"' -f2)
        else
            DISTRO="Unknown"
        fi
        
        # Generate response
        cat << EOF
{
    "success": true,
    "system_info": {
        "kernel_name": "$KERNEL_NAME",
        "kernel_release": "$KERNEL_RELEASE",
        "kernel_version": "$KERNEL_VERSION",
        "machine": "$MACHINE",
        "processor": "$PROCESSOR",
        "hardware_platform": "$HARDWARE_PLATFORM",
        "operating_system": "$OS",
        "hostname": "$HOSTNAME",
        "distribution": "$DISTRO"
    },
    "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "service": "SystemInfoService",
    "endpoint": "GetSystemInfo",
    "subject": "$SUBJECT"
}
EOF
        ;;
    
    "system.kernel")
        # Get kernel-specific information
        KERNEL_NAME=$(uname -s)
        KERNEL_RELEASE=$(uname -r)
        KERNEL_VERSION=$(uname -v)
        MACHINE=$(uname -m)
        
        # Generate response
        cat << EOF
{
    "success": true,
    "kernel_info": {
        "name": "$KERNEL_NAME",
        "release": "$KERNEL_RELEASE",
        "version": "$KERNEL_VERSION",
        "architecture": "$MACHINE"
    },
    "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "service": "SystemInfoService",
    "endpoint": "GetKernelInfo",
    "subject": "$SUBJECT"
}
EOF
        ;;
    
    "system.hostname")
        # Get hostname information
        HOSTNAME=$(uname -n)
        FQDN=$(hostname -f 2>/dev/null || echo "$HOSTNAME")
        
        # Generate response
        cat << EOF
{
    "success": true,
    "hostname_info": {
        "hostname": "$HOSTNAME",
        "fqdn": "$FQDN"
    },
    "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "service": "SystemInfoService",
    "endpoint": "GetHostname",
    "subject": "$SUBJECT"
}
EOF
        ;;
    
    *)
        # Unknown subject
        cat << EOF
{
    "success": false,
    "error": "Unknown subject: $SUBJECT",
    "available_subjects": ["system.info", "system.kernel", "system.hostname"],
    "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "service": "SystemInfoService"
}
EOF
        exit 1
        ;;
esac
