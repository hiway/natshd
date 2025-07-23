#!/bin/bash
# System Processes Discovery Service
# Cross-platform running processes and resource usage information

if [[ "$1" == "info" ]]; then
    cat <<EOF
{
    "name": "SystemService",
    "version": "1.0.0",
    "description": "Running processes and resource usage discovery",
    "endpoints": [
        {
            "name": "GetProcesses",
            "subject": "system.processes",
            "description": "Get running processes and their resource usage"
        }
    ]
}
EOF
    exit 0
fi

# Cross-platform OS detection
detect_os() {
    case "$(uname -s)" in
        Linux) OS="linux" ;;
        FreeBSD) OS="freebsd" ;;
        Darwin) OS="macos" ;;
        *) OS="unknown" ;;
    esac
}

# Get running processes
get_processes() {
    PROCESSES=""
    
    case "$OS" in
        linux)
            # Use ps with detailed format
            ps axo pid,ppid,user,pcpu,pmem,vsz,rss,tty,stat,start,time,comm --no-headers 2>/dev/null | while read -r pid ppid user pcpu pmem vsz rss tty stat start time comm; do
                # Convert VSZ and RSS from KB to bytes
                vsz_bytes=$((vsz * 1024))
                rss_bytes=$((rss * 1024))
                
                # Clean up command name
                comm_clean=$(echo "$comm" | sed 's/[][]//g')
                
                if [ -n "$PROCESSES" ]; then
                    PROCESSES="$PROCESSES,"
                fi
                PROCESSES="$PROCESSES{\"pid\":$pid,\"ppid\":$ppid,\"user\":\"$user\",\"cpu_percent\":$pcpu,\"memory_percent\":$pmem,\"vsz_bytes\":$vsz_bytes,\"rss_bytes\":$rss_bytes,\"tty\":\"$tty\",\"state\":\"$stat\",\"start_time\":\"$start\",\"cpu_time\":\"$time\",\"command\":\"$comm_clean\"}"
            done
            ;;
        freebsd)
            # FreeBSD ps format
            ps axo pid,ppid,user,pcpu,pmem,vsz,rss,tty,state,start,time,comm 2>/dev/null | tail -n +2 | while read -r pid ppid user pcpu pmem vsz rss tty state start time comm; do
                vsz_bytes=$((vsz * 1024))
                rss_bytes=$((rss * 1024))
                
                if [ -n "$PROCESSES" ]; then
                    PROCESSES="$PROCESSES,"
                fi
                PROCESSES="$PROCESSES{\"pid\":$pid,\"ppid\":$ppid,\"user\":\"$user\",\"cpu_percent\":$pcpu,\"memory_percent\":$pmem,\"vsz_bytes\":$vsz_bytes,\"rss_bytes\":$rss_bytes,\"tty\":\"$tty\",\"state\":\"$state\",\"start_time\":\"$start\",\"cpu_time\":\"$time\",\"command\":\"$comm\"}"
            done
            ;;
        macos)
            # macOS ps format
            ps axo pid,ppid,user,pcpu,pmem,vsz,rss,tty,stat,lstart,time,comm 2>/dev/null | tail -n +2 | while IFS= read -r line; do
                # Parse the line more carefully due to lstart having spaces
                pid=$(echo "$line" | awk '{print $1}')
                ppid=$(echo "$line" | awk '{print $2}')
                user=$(echo "$line" | awk '{print $3}')
                pcpu=$(echo "$line" | awk '{print $4}')
                pmem=$(echo "$line" | awk '{print $5}')
                vsz=$(echo "$line" | awk '{print $6}')
                rss=$(echo "$line" | awk '{print $7}')
                tty=$(echo "$line" | awk '{print $8}')
                stat=$(echo "$line" | awk '{print $9}')
                # lstart spans multiple fields, time is second to last, comm is last
                time=$(echo "$line" | awk '{print $(NF-1)}')
                comm=$(echo "$line" | awk '{print $NF}')
                
                vsz_bytes=$((vsz * 1024))
                rss_bytes=$((rss * 1024))
                
                if [ -n "$PROCESSES" ]; then
                    PROCESSES="$PROCESSES,"
                fi
                PROCESSES="$PROCESSES{\"pid\":$pid,\"ppid\":$ppid,\"user\":\"$user\",\"cpu_percent\":$pcpu,\"memory_percent\":$pmem,\"vsz_bytes\":$vsz_bytes,\"rss_bytes\":$rss_bytes,\"tty\":\"$tty\",\"state\":\"$stat\",\"start_time\":\"\",\"cpu_time\":\"$time\",\"command\":\"$comm\"}"
            done
            ;;
    esac
    
    if [ -z "$PROCESSES" ]; then
        PROCESSES="[]"
    else
        PROCESSES="[$PROCESSES]"
    fi
}

# Get system load averages
get_load_averages() {
    case "$OS" in
        linux)
            if [ -f /proc/loadavg ]; then
                load_info=$(cat /proc/loadavg)
                LOAD_1MIN=$(echo "$load_info" | awk '{print $1}')
                LOAD_5MIN=$(echo "$load_info" | awk '{print $2}')
                LOAD_15MIN=$(echo "$load_info" | awk '{print $3}')
                RUNNING_PROCS=$(echo "$load_info" | awk '{print $4}' | cut -d'/' -f1)
                TOTAL_PROCS=$(echo "$load_info" | awk '{print $4}' | cut -d'/' -f2)
            fi
            ;;
        freebsd|macos)
            if command -v uptime >/dev/null 2>&1; then
                uptime_output=$(uptime)
                LOAD_1MIN=$(echo "$uptime_output" | awk '{print $10}' | tr -d ',')
                LOAD_5MIN=$(echo "$uptime_output" | awk '{print $11}' | tr -d ',')
                LOAD_15MIN=$(echo "$uptime_output" | awk '{print $12}')
                
                # Get process counts
                TOTAL_PROCS=$(ps ax | wc -l | tr -d ' ')
                RUNNING_PROCS=$(ps axo state | grep -c "R")
            fi
            ;;
    esac
}

# Get top processes by CPU usage
get_top_cpu_processes() {
    TOP_CPU_PROCESSES=""
    
    case "$OS" in
        linux)
            ps axo pid,user,pcpu,comm --sort=-pcpu --no-headers 2>/dev/null | head -10 | while read -r pid user pcpu comm; do
                if [ -n "$TOP_CPU_PROCESSES" ]; then
                    TOP_CPU_PROCESSES="$TOP_CPU_PROCESSES,"
                fi
                TOP_CPU_PROCESSES="$TOP_CPU_PROCESSES{\"pid\":$pid,\"user\":\"$user\",\"cpu_percent\":$pcpu,\"command\":\"$comm\"}"
            done
            ;;
        freebsd|macos)
            ps axo pid,user,pcpu,comm 2>/dev/null | sort -k3 -nr | head -10 | tail -n +2 | while read -r pid user pcpu comm; do
                if [ -n "$TOP_CPU_PROCESSES" ]; then
                    TOP_CPU_PROCESSES="$TOP_CPU_PROCESSES,"
                fi
                TOP_CPU_PROCESSES="$TOP_CPU_PROCESSES{\"pid\":$pid,\"user\":\"$user\",\"cpu_percent\":$pcpu,\"command\":\"$comm\"}"
            done
            ;;
    esac
    
    if [ -z "$TOP_CPU_PROCESSES" ]; then
        TOP_CPU_PROCESSES="[]"
    else
        TOP_CPU_PROCESSES="[$TOP_CPU_PROCESSES]"
    fi
}

# Get top processes by memory usage
get_top_memory_processes() {
    TOP_MEMORY_PROCESSES=""
    
    case "$OS" in
        linux)
            ps axo pid,user,pmem,rss,comm --sort=-pmem --no-headers 2>/dev/null | head -10 | while read -r pid user pmem rss comm; do
                rss_bytes=$((rss * 1024))
                if [ -n "$TOP_MEMORY_PROCESSES" ]; then
                    TOP_MEMORY_PROCESSES="$TOP_MEMORY_PROCESSES,"
                fi
                TOP_MEMORY_PROCESSES="$TOP_MEMORY_PROCESSES{\"pid\":$pid,\"user\":\"$user\",\"memory_percent\":$pmem,\"rss_bytes\":$rss_bytes,\"command\":\"$comm\"}"
            done
            ;;
        freebsd|macos)
            ps axo pid,user,pmem,rss,comm 2>/dev/null | sort -k3 -nr | head -10 | tail -n +2 | while read -r pid user pmem rss comm; do
                rss_bytes=$((rss * 1024))
                if [ -n "$TOP_MEMORY_PROCESSES" ]; then
                    TOP_MEMORY_PROCESSES="$TOP_MEMORY_PROCESSES,"
                fi
                TOP_MEMORY_PROCESSES="$TOP_MEMORY_PROCESSES{\"pid\":$pid,\"user\":\"$user\",\"memory_percent\":$pmem,\"rss_bytes\":$rss_bytes,\"command\":\"$comm\"}"
            done
            ;;
    esac
    
    if [ -z "$TOP_MEMORY_PROCESSES" ]; then
        TOP_MEMORY_PROCESSES="[]"
    else
        TOP_MEMORY_PROCESSES="[$TOP_MEMORY_PROCESSES]"
    fi
}

# Get process statistics
get_process_stats() {
    case "$OS" in
        linux)
            ZOMBIE_PROCS=$(ps axo state --no-headers | grep -c "Z")
            SLEEPING_PROCS=$(ps axo state --no-headers | grep -c "S")
            RUNNING_PROCS=$(ps axo state --no-headers | grep -c "R")
            STOPPED_PROCS=$(ps axo state --no-headers | grep -c "T")
            ;;
        freebsd|macos)
            ZOMBIE_PROCS=$(ps axo state | grep -c "Z")
            SLEEPING_PROCS=$(ps axo state | grep -c "S")
            RUNNING_PROCS=$(ps axo state | grep -c "R")
            STOPPED_PROCS=$(ps axo state | grep -c "T")
            ;;
    esac
}

# Get CPU information for context
get_cpu_count() {
    case "$OS" in
        linux)
            CPU_COUNT=$(nproc 2>/dev/null || grep -c "^processor" /proc/cpuinfo)
            ;;
        freebsd)
            CPU_COUNT=$(sysctl -n hw.ncpu 2>/dev/null)
            ;;
        macos)
            CPU_COUNT=$(sysctl -n hw.logicalcpu 2>/dev/null)
            ;;
    esac
}

# Main execution
SUBJECT="$1"

case "$SUBJECT" in
    "system.processes")
        # Gather all process information
        detect_os
        get_processes
        get_load_averages
        get_top_cpu_processes
        get_top_memory_processes
        get_process_stats
        get_cpu_count
        
        # Generate response
        cat <<EOF
{
    "success": true,
    "data": {
        "summary": {
            "total_processes": ${TOTAL_PROCS:-0},
            "running_processes": ${RUNNING_PROCS:-0},
            "sleeping_processes": ${SLEEPING_PROCS:-0},
            "stopped_processes": ${STOPPED_PROCS:-0},
            "zombie_processes": ${ZOMBIE_PROCS:-0},
            "cpu_count": ${CPU_COUNT:-0}
        },
        "load_averages": {
            "1min": ${LOAD_1MIN:-0},
            "5min": ${LOAD_5MIN:-0},
            "15min": ${LOAD_15MIN:-0}
        },
        "top_cpu_processes": $TOP_CPU_PROCESSES,
        "top_memory_processes": $TOP_MEMORY_PROCESSES,
        "all_processes": $PROCESSES
    },
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "SystemProcessesService",
    "endpoint": "GetProcesses", 
    "subject": "$SUBJECT",
    "os": "$OS"
}
EOF
        ;;
    *)
        cat <<EOF
{
    "success": false,
    "error": "Unknown subject: $SUBJECT",
    "available_subjects": ["system.processes"],
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "SystemProcessesService"
}
EOF
        ;;
esac
