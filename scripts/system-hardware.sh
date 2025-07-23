#!/bin/bash
# System Hardware Discovery Service
# Cross-platform hardware information gathering

if [[ "$1" == "info" ]]; then
    cat <<EOF
{
    "name": "SystemService",
    "version": "1.0.0",
    "description": "Hardware information discovery service",
    "endpoints": [
        {
            "name": "GetHardware",
            "subject": "system.hardware",
            "description": "Gather detailed hardware information",
            "metadata": {
                "parameters": {},
                "notes": "This endpoint does not require any parameters. Send an empty JSON object: {}."
            }
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

# Get CPU information
get_cpu_info() {
    case "$OS" in
        linux)
            if [ -f /proc/cpuinfo ]; then
                CPU_MODEL=$(grep "model name" /proc/cpuinfo | head -1 | cut -d: -f2 | sed 's/^ *//')
                CPU_VENDOR=$(grep "vendor_id" /proc/cpuinfo | head -1 | cut -d: -f2 | sed 's/^ *//')
                CPU_CORES=$(grep "cpu cores" /proc/cpuinfo | head -1 | cut -d: -f2 | sed 's/^ *//')
                CPU_THREADS=$(grep -c "^processor" /proc/cpuinfo)
                CPU_CACHE_SIZE=$(grep "cache size" /proc/cpuinfo | head -1 | cut -d: -f2 | sed 's/^ *//')
                CPU_FLAGS=$(grep "^flags" /proc/cpuinfo | head -1 | cut -d: -f2 | sed 's/^ *//')
                
                # Get CPU frequency
                if [ -f /sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq ]; then
                    CPU_FREQ=$(cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq)
                    CPU_FREQ_MHZ=$((CPU_FREQ / 1000))
                elif [ -f /proc/cpuinfo ]; then
                    CPU_FREQ_MHZ=$(grep "cpu MHz" /proc/cpuinfo | head -1 | cut -d: -f2 | sed 's/^ *//' | cut -d. -f1)
                fi
            fi
            ;;
        freebsd)
            CPU_MODEL=$(sysctl -n hw.model 2>/dev/null)
            CPU_CORES=$(sysctl -n hw.ncpu 2>/dev/null)
            CPU_THREADS=$(sysctl -n hw.ncpu 2>/dev/null)
            CPU_FREQ_MHZ=$(sysctl -n dev.cpu.0.freq 2>/dev/null)
            ;;
        macos)
            CPU_MODEL=$(sysctl -n machdep.cpu.brand_string 2>/dev/null)
            CPU_VENDOR=$(sysctl -n machdep.cpu.vendor 2>/dev/null)
            CPU_CORES=$(sysctl -n hw.physicalcpu 2>/dev/null)
            CPU_THREADS=$(sysctl -n hw.logicalcpu 2>/dev/null)
            CPU_FREQ_MHZ=$(sysctl -n hw.cpufrequency_max 2>/dev/null)
            if [ -n "$CPU_FREQ_MHZ" ]; then
                CPU_FREQ_MHZ=$((CPU_FREQ_MHZ / 1000000))
            fi
            CPU_CACHE_SIZE=$(sysctl -n hw.l3cachesize 2>/dev/null)
            ;;
    esac
}

# Get memory information
get_memory_info() {
    case "$OS" in
        linux)
            if [ -f /proc/meminfo ]; then
                MEM_TOTAL=$(grep "MemTotal:" /proc/meminfo | awk '{print $2}')
                MEM_TOTAL_BYTES=$((MEM_TOTAL * 1024))
                
                # Get memory type and speed if available
                if command -v dmidecode >/dev/null 2>&1; then
                    MEM_TYPE=$(dmidecode -t memory 2>/dev/null | grep "Type:" | grep -v "Error\|Unknown" | head -1 | awk '{print $2}')
                    MEM_SPEED=$(dmidecode -t memory 2>/dev/null | grep "Speed:" | grep -v "Unknown" | head -1 | awk '{print $2 " " $3}')
                    MEM_SLOTS_TOTAL=$(dmidecode -t memory 2>/dev/null | grep "Number Of Devices:" | awk '{print $4}')
                    MEM_SLOTS_USED=$(dmidecode -t memory 2>/dev/null | grep "Size:" | grep -v "No Module Installed" | wc -l)
                fi
            fi
            ;;
        freebsd)
            MEM_TOTAL_BYTES=$(sysctl -n hw.physmem 2>/dev/null)
            MEM_PAGE_SIZE=$(sysctl -n hw.pagesize 2>/dev/null)
            ;;
        macos)
            MEM_TOTAL_BYTES=$(sysctl -n hw.memsize 2>/dev/null)
            ;;
    esac
}

# Get storage information
get_storage_info() {
    STORAGE_DEVICES=""
    
    case "$OS" in
        linux)
            # Get block devices
            if [ -d /sys/block ]; then
                for dev in /sys/block/*; do
                    device=$(basename "$dev")
                    # Skip loop, ram, and other virtual devices
                    case "$device" in
                        loop*|ram*|sr*) continue ;;
                    esac
                    
                    if [ -f "$dev/size" ]; then
                        size_sectors=$(cat "$dev/size")
                        size_bytes=$((size_sectors * 512))
                        
                        # Get device model if available
                        model=""
                        if [ -f "$dev/device/model" ]; then
                            model=$(cat "$dev/device/model" | tr -d ' ')
                        fi
                        
                        # Get device type
                        rotational="unknown"
                        if [ -f "$dev/queue/rotational" ]; then
                            rot=$(cat "$dev/queue/rotational")
                            if [ "$rot" = "0" ]; then
                                rotational="SSD"
                            else
                                rotational="HDD"
                            fi
                        fi
                        
                        if [ -n "$STORAGE_DEVICES" ]; then
                            STORAGE_DEVICES="$STORAGE_DEVICES,"
                        fi
                        STORAGE_DEVICES="$STORAGE_DEVICES{\"device\":\"/dev/$device\",\"size_bytes\":$size_bytes,\"model\":\"$model\",\"type\":\"$rotational\"}"
                    fi
                done
            fi
            ;;
        freebsd)
            # Use geom to get disk information
            if command -v geom >/dev/null 2>&1; then
                geom disk list 2>/dev/null | while read -r line; do
                    case "$line" in
                        "Geom name:"*)
                            device=$(echo "$line" | awk '{print $3}')
                            ;;
                        "Mediasize:"*)
                            size_bytes=$(echo "$line" | awk '{print $2}')
                            ;;
                        "Descr:"*)
                            model=$(echo "$line" | cut -d: -f2 | sed 's/^ *//')
                            ;;
                    esac
                done
            fi
            ;;
        macos)
            # Use diskutil to get disk information
            if command -v diskutil >/dev/null 2>&1; then
                diskutil list -plist physical 2>/dev/null | grep -A 20 "<key>AllDisks</key>" | grep "<string>" | sed 's/.*<string>\(.*\)<\/string>/\1/' | while read -r disk; do
                    info=$(diskutil info "$disk" 2>/dev/null)
                    if [ -n "$info" ]; then
                        size_bytes=$(echo "$info" | grep "Disk Size:" | awk '{print $3}' | tr -d '()')
                        model=$(echo "$info" | grep "Device / Media Name:" | cut -d: -f2 | sed 's/^ *//')
                        type=$(echo "$info" | grep "Solid State:" | awk '{print $3}')
                        if [ "$type" = "Yes" ]; then
                            disk_type="SSD"
                        else
                            disk_type="HDD"
                        fi
                        
                        if [ -n "$STORAGE_DEVICES" ]; then
                            STORAGE_DEVICES="$STORAGE_DEVICES,"
                        fi
                        STORAGE_DEVICES="$STORAGE_DEVICES{\"device\":\"/dev/$disk\",\"size_bytes\":$size_bytes,\"model\":\"$model\",\"type\":\"$disk_type\"}"
                    fi
                done
            fi
            ;;
    esac
    
    if [ -z "$STORAGE_DEVICES" ]; then
        STORAGE_DEVICES="[]"
    else
        STORAGE_DEVICES="[$STORAGE_DEVICES]"
    fi
}

# Get graphics information
get_graphics_info() {
    GPU_INFO=""
    
    case "$OS" in
        linux)
            # Try lspci first
            if command -v lspci >/dev/null 2>&1; then
                GPU_INFO=$(lspci 2>/dev/null | grep -i "vga\|3d\|display" | head -5)
            fi
            
            # Try nvidia-smi for NVIDIA cards
            if command -v nvidia-smi >/dev/null 2>&1; then
                NVIDIA_INFO=$(nvidia-smi -L 2>/dev/null)
                if [ -n "$NVIDIA_INFO" ]; then
                    GPU_INFO="$GPU_INFO\n$NVIDIA_INFO"
                fi
            fi
            ;;
        freebsd)
            if command -v pciconf >/dev/null 2>&1; then
                GPU_INFO=$(pciconf -lv 2>/dev/null | grep -A 3 -i "display\|vga")
            fi
            ;;
        macos)
            if command -v system_profiler >/dev/null 2>&1; then
                GPU_INFO=$(system_profiler SPDisplaysDataType 2>/dev/null | grep "Chipset Model:" | awk -F: '{print $2}' | sed 's/^ *//')
            fi
            ;;
    esac
}

# Get network hardware information
get_network_info() {
    NETWORK_INTERFACES=""
    
    case "$OS" in
        linux)
            # Get network interfaces
            for iface in /sys/class/net/*; do
                interface=$(basename "$iface")
                # Skip loopback
                [ "$interface" = "lo" ] && continue
                
                # Get MAC address
                mac=""
                if [ -f "$iface/address" ]; then
                    mac=$(cat "$iface/address")
                fi
                
                # Get interface state
                state="unknown"
                if [ -f "$iface/operstate" ]; then
                    state=$(cat "$iface/operstate")
                fi
                
                # Get speed if available
                speed=""
                if [ -f "$iface/speed" ]; then
                    speed=$(cat "$iface/speed" 2>/dev/null)
                fi
                
                if [ -n "$NETWORK_INTERFACES" ]; then
                    NETWORK_INTERFACES="$NETWORK_INTERFACES,"
                fi
                NETWORK_INTERFACES="$NETWORK_INTERFACES{\"interface\":\"$interface\",\"mac\":\"$mac\",\"state\":\"$state\",\"speed\":\"$speed\"}"
            done
            ;;
        freebsd|macos)
            # Use ifconfig to get interface information
            if command -v ifconfig >/dev/null 2>&1; then
                ifconfig -a 2>/dev/null | grep "^[a-z]" | while read -r line; do
                    interface=$(echo "$line" | awk '{print $1}' | tr -d ':')
                    # Skip loopback
                    [ "$interface" = "lo0" ] && continue
                    
                    # Get more details about this interface
                    iface_info=$(ifconfig "$interface" 2>/dev/null)
                    mac=$(echo "$iface_info" | grep "ether " | awk '{print $2}')
                    state=$(echo "$iface_info" | grep -o "status: [^[:space:]]*" | cut -d: -f2 | sed 's/^ *//')
                    
                    if [ -n "$NETWORK_INTERFACES" ]; then
                        NETWORK_INTERFACES="$NETWORK_INTERFACES,"
                    fi
                    NETWORK_INTERFACES="$NETWORK_INTERFACES{\"interface\":\"$interface\",\"mac\":\"$mac\",\"state\":\"$state\",\"speed\":\"\"}"
                done
            fi
            ;;
    esac
    
    if [ -z "$NETWORK_INTERFACES" ]; then
        NETWORK_INTERFACES="[]"
    else
        NETWORK_INTERFACES="[$NETWORK_INTERFACES]"
    fi
}

# Main execution
SUBJECT="$1"

case "$SUBJECT" in
    "system.hardware")
        # Gather all hardware information
        detect_os
        get_cpu_info
        get_memory_info
        get_storage_info
        get_graphics_info
        get_network_info
        
        # Generate response
        cat <<EOF
{
    "success": true,
    "data": {
        "cpu": {
            "model": "${CPU_MODEL:-unknown}",
            "vendor": "${CPU_VENDOR:-unknown}",
            "cores": ${CPU_CORES:-0},
            "threads": ${CPU_THREADS:-0},
            "frequency_mhz": ${CPU_FREQ_MHZ:-0},
            "cache_size": "${CPU_CACHE_SIZE:-unknown}",
            "flags": "${CPU_FLAGS:-unknown}"
        },
        "memory": {
            "total_bytes": ${MEM_TOTAL_BYTES:-0},
            "type": "${MEM_TYPE:-unknown}",
            "speed": "${MEM_SPEED:-unknown}",
            "slots_total": ${MEM_SLOTS_TOTAL:-0},
            "slots_used": ${MEM_SLOTS_USED:-0}
        },
        "storage": $STORAGE_DEVICES,
        "graphics": "${GPU_INFO:-unknown}",
        "network_interfaces": $NETWORK_INTERFACES
    },
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "SystemHardwareService",
    "endpoint": "GetHardware",
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
    "available_subjects": ["system.hardware"],
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "SystemHardwareService"
}
EOF
        ;;
esac
