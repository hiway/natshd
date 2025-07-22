#!/bin/bash
# System Facts Discovery Service
# Cross-platform system information gathering

if [[ "$1" == "info" ]]; then
    cat <<EOF
{
    "name": "SystemFactsService",
    "version": "1.0.0",
    "description": "Comprehensive system information discovery",
    "endpoints": [
        {
            "name": "GetFacts",
            "subject": "system.facts",
            "description": "Gather comprehensive system information"
        }
    ]
}
EOF
    exit 0
fi

# Cross-platform OS detection
detect_os() {
    case "$(uname -s)" in
        Linux)
            OS="linux"
            if [ -f /etc/os-release ]; then
                . /etc/os-release
                OS_FAMILY="$ID"
                OS_NAME="$NAME"
                OS_VERSION="$VERSION_ID"
                OS_VERSION_FULL="$VERSION"
            elif [ -f /etc/redhat-release ]; then
                OS_FAMILY="rhel"
                OS_NAME=$(cat /etc/redhat-release)
            elif [ -f /etc/debian_version ]; then
                OS_FAMILY="debian"
                OS_NAME="Debian"
                OS_VERSION=$(cat /etc/debian_version)
            fi
            KERNEL_NAME="$(uname -s)"
            KERNEL_RELEASE="$(uname -r)"
            KERNEL_VERSION="$(uname -v)"
            ;;
        FreeBSD)
            OS="freebsd"
            OS_FAMILY="freebsd"
            OS_NAME="FreeBSD"
            OS_VERSION="$(uname -r)"
            KERNEL_NAME="$(uname -s)"
            KERNEL_RELEASE="$(uname -r)"
            KERNEL_VERSION="$(uname -v)"
            ;;
        Darwin)
            OS="macos"
            OS_FAMILY="darwin"
            OS_NAME="macOS"
            OS_VERSION="$(sw_vers -productVersion)"
            OS_BUILD="$(sw_vers -buildVersion)"
            KERNEL_NAME="$(uname -s)"
            KERNEL_RELEASE="$(uname -r)"
            KERNEL_VERSION="$(uname -v)"
            ;;
        *)
            OS="unknown"
            OS_FAMILY="unknown"
            ;;
    esac
    
    ARCHITECTURE="$(uname -m)"
    HOSTNAME="$(hostname)"
    FQDN="$(hostname -f 2>/dev/null || hostname)"
}

# Get uptime information
get_uptime() {
    case "$OS" in
        linux|freebsd)
            if [ -f /proc/uptime ]; then
                UPTIME_SECONDS=$(awk '{print int($1)}' /proc/uptime)
            else
                # FreeBSD and others
                UPTIME_SECONDS=$(sysctl -n kern.boottime | awk '{print $4}' | tr -d ',')
                if [ -n "$UPTIME_SECONDS" ]; then
                    CURRENT_TIME=$(date +%s)
                    UPTIME_SECONDS=$((CURRENT_TIME - UPTIME_SECONDS))
                fi
            fi
            ;;
        macos)
            BOOT_TIME=$(sysctl -n kern.boottime | sed 's/.*sec = \([0-9]*\).*/\1/')
            CURRENT_TIME=$(date +%s)
            UPTIME_SECONDS=$((CURRENT_TIME - BOOT_TIME))
            ;;
    esac
    
    if [ -n "$UPTIME_SECONDS" ]; then
        UPTIME_DAYS=$((UPTIME_SECONDS / 86400))
        UPTIME_HOURS=$(((UPTIME_SECONDS % 86400) / 3600))
        UPTIME_MINUTES=$(((UPTIME_SECONDS % 3600) / 60))
    fi
}

# Get CPU information
get_cpu_info() {
    case "$OS" in
        linux)
            if [ -f /proc/cpuinfo ]; then
                CPU_MODEL=$(grep "model name" /proc/cpuinfo | head -1 | cut -d: -f2 | sed 's/^ *//')
                CPU_CORES=$(grep -c "^processor" /proc/cpuinfo)
                CPU_THREADS=$(grep -c "^processor" /proc/cpuinfo)
                
                # Get CPU architecture details
                if grep -q "^flags.*lm" /proc/cpuinfo; then
                    CPU_ARCH="x86_64"
                else
                    CPU_ARCH="$(uname -m)"
                fi
            fi
            ;;
        freebsd)
            CPU_MODEL=$(sysctl -n hw.model 2>/dev/null)
            CPU_CORES=$(sysctl -n hw.ncpu 2>/dev/null)
            CPU_THREADS=$(sysctl -n hw.ncpu 2>/dev/null)
            CPU_ARCH="$(uname -m)"
            ;;
        macos)
            CPU_MODEL=$(sysctl -n machdep.cpu.brand_string 2>/dev/null)
            CPU_CORES=$(sysctl -n hw.physicalcpu 2>/dev/null)
            CPU_THREADS=$(sysctl -n hw.logicalcpu 2>/dev/null)
            CPU_ARCH="$(uname -m)"
            ;;
    esac
}

# Get memory information
get_memory_info() {
    case "$OS" in
        linux)
            if [ -f /proc/meminfo ]; then
                MEM_TOTAL=$(grep "MemTotal:" /proc/meminfo | awk '{print $2}')
                MEM_FREE=$(grep "MemFree:" /proc/meminfo | awk '{print $2}')
                MEM_AVAILABLE=$(grep "MemAvailable:" /proc/meminfo | awk '{print $2}')
                MEM_BUFFERS=$(grep "Buffers:" /proc/meminfo | awk '{print $2}')
                MEM_CACHED=$(grep "^Cached:" /proc/meminfo | awk '{print $2}')
                SWAP_TOTAL=$(grep "SwapTotal:" /proc/meminfo | awk '{print $2}')
                SWAP_FREE=$(grep "SwapFree:" /proc/meminfo | awk '{print $2}')
                
                # Convert from KB to bytes
                MEM_TOTAL_BYTES=$((MEM_TOTAL * 1024))
                MEM_FREE_BYTES=$((MEM_FREE * 1024))
                MEM_AVAILABLE_BYTES=$((MEM_AVAILABLE * 1024))
                SWAP_TOTAL_BYTES=$((SWAP_TOTAL * 1024))
                SWAP_FREE_BYTES=$((SWAP_FREE * 1024))
            fi
            ;;
        freebsd)
            MEM_TOTAL_BYTES=$(sysctl -n hw.physmem 2>/dev/null)
            MEM_FREE_PAGES=$(sysctl -n vm.stats.vm.v_free_count 2>/dev/null)
            PAGE_SIZE=$(sysctl -n vm.stats.vm.v_page_size 2>/dev/null)
            if [ -n "$MEM_FREE_PAGES" ] && [ -n "$PAGE_SIZE" ]; then
                MEM_FREE_BYTES=$((MEM_FREE_PAGES * PAGE_SIZE))
            fi
            SWAP_TOTAL_BYTES=$(swapctl -s | awk '{print $2*1024}' 2>/dev/null)
            ;;
        macos)
            MEM_TOTAL_BYTES=$(sysctl -n hw.memsize 2>/dev/null)
            # Get memory pressure from vm_stat
            VM_STAT=$(vm_stat 2>/dev/null)
            if [ -n "$VM_STAT" ]; then
                PAGE_SIZE=$(vm_stat | grep "page size" | awk '{print $8}')
                FREE_PAGES=$(echo "$VM_STAT" | grep "Pages free" | awk '{print $3}' | tr -d '.')
                if [ -n "$FREE_PAGES" ] && [ -n "$PAGE_SIZE" ]; then
                    MEM_FREE_BYTES=$((FREE_PAGES * PAGE_SIZE))
                fi
            fi
            ;;
    esac
}

# Get timezone information
get_timezone_info() {
    if command -v timedatectl >/dev/null 2>&1; then
        TIMEZONE=$(timedatectl | grep "Time zone" | awk '{print $3}')
    elif [ -f /etc/timezone ]; then
        TIMEZONE=$(cat /etc/timezone)
    elif [ -L /etc/localtime ]; then
        TIMEZONE=$(readlink /etc/localtime | sed 's|.*/zoneinfo/||')
    else
        TIMEZONE=$(date +%Z)
    fi
    
    CURRENT_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    LOCAL_TIME=$(date +"%Y-%m-%dT%H:%M:%S%z")
}

# Main execution
SUBJECT="$1"
REQUEST=$(cat)

case "$SUBJECT" in
    "system.facts")
        # Gather all system information
        detect_os
        get_uptime
        get_cpu_info
        get_memory_info
        get_timezone_info
        
        # Generate response
        cat <<EOF
{
    "success": true,
    "data": {
        "system": {
            "hostname": "${HOSTNAME:-unknown}",
            "fqdn": "${FQDN:-unknown}",
            "domain": "${FQDN#*.}",
            "timezone": "${TIMEZONE:-unknown}",
            "current_time": "${CURRENT_TIME}",
            "local_time": "${LOCAL_TIME}",
            "uptime": {
                "seconds": ${UPTIME_SECONDS:-0},
                "days": ${UPTIME_DAYS:-0},
                "hours": ${UPTIME_HOURS:-0},
                "minutes": ${UPTIME_MINUTES:-0},
                "formatted": "${UPTIME_DAYS:-0}d ${UPTIME_HOURS:-0}h ${UPTIME_MINUTES:-0}m"
            }
        },
        "os": {
            "family": "${OS_FAMILY:-unknown}",
            "name": "${OS_NAME:-unknown}",
            "version": "${OS_VERSION:-unknown}",
            "version_full": "${OS_VERSION_FULL:-unknown}",
            "build": "${OS_BUILD:-}",
            "architecture": "${ARCHITECTURE:-unknown}"
        },
        "kernel": {
            "name": "${KERNEL_NAME:-unknown}",
            "release": "${KERNEL_RELEASE:-unknown}",
            "version": "${KERNEL_VERSION:-unknown}"
        },
        "cpu": {
            "model": "${CPU_MODEL:-unknown}",
            "architecture": "${CPU_ARCH:-unknown}",
            "cores": ${CPU_CORES:-0},
            "threads": ${CPU_THREADS:-0}
        },
        "memory": {
            "total_bytes": ${MEM_TOTAL_BYTES:-0},
            "free_bytes": ${MEM_FREE_BYTES:-0},
            "available_bytes": ${MEM_AVAILABLE_BYTES:-0},
            "buffers_bytes": ${MEM_BUFFERS_BYTES:-0},
            "cached_bytes": ${MEM_CACHED_BYTES:-0},
            "swap_total_bytes": ${SWAP_TOTAL_BYTES:-0},
            "swap_free_bytes": ${SWAP_FREE_BYTES:-0}
        }
    },
    "timestamp": "${CURRENT_TIME}",
    "service": "SystemFactsService",
    "endpoint": "GetFacts",
    "subject": "$SUBJECT",
    "os": "$OS",
    "os_version": "$OS_VERSION"
}
EOF
        ;;
    *)
        cat <<EOF
{
    "success": false,
    "error": "Unknown subject: $SUBJECT",
    "available_subjects": ["system.facts"],
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "SystemFactsService"
}
EOF
        ;;
esac
