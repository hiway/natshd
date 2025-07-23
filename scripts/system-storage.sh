#!/bin/bash
# System Storage Discovery Service
# Cross-platform storage and filesystem information gathering

if [[ "$1" == "info" ]]; then
    cat <<EOF
{
    "name": "SystemService",
    "version": "1.0.0",
    "description": "Storage and filesystem information discovery",
    "endpoints": [
        {
            "name": "GetStorage",
            "subject": "system.storage",
            "description": "Gather filesystem usage, mount points, and disk information",
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

# Get filesystem information
get_filesystems() {
    FILESYSTEMS=""
    
    case "$OS" in
        linux)
            # Use df with better formatting
            df -TPh 2>/dev/null | tail -n +2 | while read -r filesystem fstype size used avail use_percent mount; do
                # Skip special filesystems
                case "$filesystem" in
                    tmpfs|devtmpfs|sysfs|proc|devpts|securityfs|cgroup*|pstore|bpf|tracefs|debugfs|hugetlbfs|mqueue|fusectl|configfs|selinuxfs) continue ;;
                esac
                
                # Convert percentage to number
                use_num=$(echo "$use_percent" | tr -d '%')
                
                # Get additional filesystem info
                inode_info=""
                if df -i "$mount" >/dev/null 2>&1; then
                    inode_info=$(df -i "$mount" 2>/dev/null | tail -1)
                    inodes_total=$(echo "$inode_info" | awk '{print $2}')
                    inodes_used=$(echo "$inode_info" | awk '{print $3}')
                    inodes_free=$(echo "$inode_info" | awk '{print $4}')
                    inodes_use_percent=$(echo "$inode_info" | awk '{print $5}' | tr -d '%')
                fi
                
                # Get mount options
                mount_opts=""
                if mount | grep -q "^$filesystem on $mount "; then
                    mount_opts=$(mount | grep "^$filesystem on $mount " | sed 's/.*(\(.*\)).*/\1/')
                fi
                
                if [ -n "$FILESYSTEMS" ]; then
                    FILESYSTEMS="$FILESYSTEMS,"
                fi
                FILESYSTEMS="$FILESYSTEMS{\"device\":\"$filesystem\",\"fstype\":\"$fstype\",\"size\":\"$size\",\"used\":\"$used\",\"available\":\"$avail\",\"use_percent\":$use_num,\"mount_point\":\"$mount\",\"mount_options\":\"$mount_opts\",\"inodes_total\":${inodes_total:-0},\"inodes_used\":${inodes_used:-0},\"inodes_free\":${inodes_free:-0},\"inodes_use_percent\":${inodes_use_percent:-0}}"
            done
            ;;
        freebsd)
            df -h 2>/dev/null | tail -n +2 | while read -r filesystem size used avail capacity mount; do
                # Skip special filesystems
                case "$filesystem" in
                    devfs|fdescfs|linprocfs|linsysfs|tmpfs) continue ;;
                esac
                
                use_num=$(echo "$capacity" | tr -d '%')
                
                # Get filesystem type
                fstype=$(mount | grep "^$filesystem on $mount " | sed 's/.*(\([^,]*\).*/\1/' | awk '{print $1}')
                
                # Get mount options
                mount_opts=$(mount | grep "^$filesystem on $mount " | sed 's/.*(\(.*\)).*/\1/')
                
                if [ -n "$FILESYSTEMS" ]; then
                    FILESYSTEMS="$FILESYSTEMS,"
                fi
                FILESYSTEMS="$FILESYSTEMS{\"device\":\"$filesystem\",\"fstype\":\"$fstype\",\"size\":\"$size\",\"used\":\"$used\",\"available\":\"$avail\",\"use_percent\":$use_num,\"mount_point\":\"$mount\",\"mount_options\":\"$mount_opts\",\"inodes_total\":0,\"inodes_used\":0,\"inodes_free\":0,\"inodes_use_percent\":0}"
            done
            ;;
        macos)
            df -h 2>/dev/null | tail -n +2 | while read -r filesystem size used avail capacity iused ifree iuse_percent mount; do
                # Skip special filesystems
                case "$filesystem" in
                    devfs|map*) continue ;;
                esac
                
                use_num=$(echo "$capacity" | tr -d '%')
                iuse_num=$(echo "$iuse_percent" | tr -d '%')
                
                # Get filesystem type
                fstype=$(mount | grep "^$filesystem on $mount " | sed 's/.*(\([^,]*\).*/\1/' | awk '{print $1}')
                
                # Get mount options
                mount_opts=$(mount | grep "^$filesystem on $mount " | sed 's/.*(\(.*\)).*/\1/')
                
                if [ -n "$FILESYSTEMS" ]; then
                    FILESYSTEMS="$FILESYSTEMS,"
                fi
                FILESYSTEMS="$FILESYSTEMS{\"device\":\"$filesystem\",\"fstype\":\"$fstype\",\"size\":\"$size\",\"used\":\"$used\",\"available\":\"$avail\",\"use_percent\":$use_num,\"mount_point\":\"$mount\",\"mount_options\":\"$mount_opts\",\"inodes_total\":$((iused + ifree)),\"inodes_used\":$iused,\"inodes_free\":$ifree,\"inodes_use_percent\":$iuse_num}"
            done
            ;;
    esac
    
    if [ -z "$FILESYSTEMS" ]; then
        FILESYSTEMS="[]"
    else
        FILESYSTEMS="[$FILESYSTEMS]"
    fi
}

# Get block devices
get_block_devices() {
    BLOCK_DEVICES=""
    
    case "$OS" in
        linux)
            # Use lsblk if available
            if command -v lsblk >/dev/null 2>&1; then
                lsblk -J 2>/dev/null | jq -c '.blockdevices[]?' 2>/dev/null | while read -r device_json; do
                    name=$(echo "$device_json" | jq -r '.name')
                    size=$(echo "$device_json" | jq -r '.size // ""')
                    type=$(echo "$device_json" | jq -r '.type // ""')
                    fstype=$(echo "$device_json" | jq -r '.fstype // ""')
                    mountpoint=$(echo "$device_json" | jq -r '.mountpoint // ""')
                    
                    # Get additional info from /sys/block if it's a disk
                    rotational="unknown"
                    model=""
                    if [ -f "/sys/block/$name/queue/rotational" ]; then
                        rot=$(cat "/sys/block/$name/queue/rotational")
                        if [ "$rot" = "0" ]; then
                            rotational="SSD"
                        else
                            rotational="HDD"
                        fi
                    fi
                    
                    if [ -f "/sys/block/$name/device/model" ]; then
                        model=$(cat "/sys/block/$name/device/model" | tr -d ' ')
                    fi
                    
                    if [ -n "$BLOCK_DEVICES" ]; then
                        BLOCK_DEVICES="$BLOCK_DEVICES,"
                    fi
                    BLOCK_DEVICES="$BLOCK_DEVICES{\"name\":\"$name\",\"size\":\"$size\",\"type\":\"$type\",\"fstype\":\"$fstype\",\"mountpoint\":\"$mountpoint\",\"model\":\"$model\",\"rotational\":\"$rotational\"}"
                done
            else
                # Fall back to parsing /proc/partitions
                if [ -f /proc/partitions ]; then
                    tail -n +3 /proc/partitions | while read -r major minor blocks name; do
                        size_mb=$((blocks / 1024))
                        size_gb=$((size_mb / 1024))
                        
                        if [ $size_gb -gt 0 ]; then
                            size="${size_gb}G"
                        else
                            size="${size_mb}M"
                        fi
                        
                        if [ -n "$BLOCK_DEVICES" ]; then
                            BLOCK_DEVICES="$BLOCK_DEVICES,"
                        fi
                        BLOCK_DEVICES="$BLOCK_DEVICES{\"name\":\"$name\",\"size\":\"$size\",\"type\":\"disk\",\"fstype\":\"\",\"mountpoint\":\"\",\"model\":\"\",\"rotational\":\"unknown\"}"
                    done
                fi
            fi
            ;;
        freebsd)
            # Use geom to get disk info
            if command -v geom >/dev/null 2>&1; then
                geom disk list 2>/dev/null | awk '
                /^Geom name:/ { name = $3 }
                /^Mediasize:/ { 
                    size = $2
                    if (size > 1073741824) size = int(size/1073741824) "G"
                    else if (size > 1048576) size = int(size/1048576) "M"
                    else size = size "B"
                }
                /^Descr:/ { model = substr($0, index($0, ":") + 2) }
                /^$/ { 
                    if (name && size) {
                        if (devices != "") devices = devices ","
                        devices = devices "{\"name\":\"" name "\",\"size\":\"" size "\",\"type\":\"disk\",\"fstype\":\"\",\"mountpoint\":\"\",\"model\":\"" model "\",\"rotational\":\"unknown\"}"
                    }
                    name = ""; size = ""; model = ""
                }
                END { print devices }'
            fi
            ;;
        macos)
            # Use diskutil
            if command -v diskutil >/dev/null 2>&1; then
                diskutil list -plist physical 2>/dev/null | plutil -p - 2>/dev/null | grep -A 1 "AllDisks" | grep "\"" | sed 's/.*"\(.*\)".*/\1/' | while read -r disk; do
                    if [ -n "$disk" ]; then
                        info=$(diskutil info "$disk" 2>/dev/null)
                        if [ -n "$info" ]; then
                            name="$disk"
                            size=$(echo "$info" | grep "Disk Size:" | awk '{print $3}' | tr -d '()')
                            model=$(echo "$info" | grep "Device / Media Name:" | cut -d: -f2 | sed 's/^ *//')
                            solid_state=$(echo "$info" | grep "Solid State:" | awk '{print $3}')
                            
                            if [ "$solid_state" = "Yes" ]; then
                                rotational="SSD"
                            else
                                rotational="HDD"
                            fi
                            
                            if [ -n "$BLOCK_DEVICES" ]; then
                                BLOCK_DEVICES="$BLOCK_DEVICES,"
                            fi
                            BLOCK_DEVICES="$BLOCK_DEVICES{\"name\":\"$name\",\"size\":\"$size\",\"type\":\"disk\",\"fstype\":\"\",\"mountpoint\":\"\",\"model\":\"$model\",\"rotational\":\"$rotational\"}"
                        fi
                    fi
                done
            fi
            ;;
    esac
    
    if [ -z "$BLOCK_DEVICES" ]; then
        BLOCK_DEVICES="[]"
    else
        BLOCK_DEVICES="[$BLOCK_DEVICES]"
    fi
}

# Get swap information
get_swap_info() {
    SWAP_TOTAL=0
    SWAP_USED=0
    SWAP_FREE=0
    SWAP_DEVICES=""
    
    case "$OS" in
        linux)
            if [ -f /proc/swaps ]; then
                # Get swap summary from /proc/meminfo
                if [ -f /proc/meminfo ]; then
                    SWAP_TOTAL=$(grep "SwapTotal:" /proc/meminfo | awk '{print $2 * 1024}')
                    SWAP_FREE=$(grep "SwapFree:" /proc/meminfo | awk '{print $2 * 1024}')
                    SWAP_USED=$((SWAP_TOTAL - SWAP_FREE))
                fi
                
                # Get individual swap devices
                tail -n +2 /proc/swaps | while read -r filename type size used priority; do
                    size_bytes=$((size * 1024))
                    used_bytes=$((used * 1024))
                    
                    if [ -n "$SWAP_DEVICES" ]; then
                        SWAP_DEVICES="$SWAP_DEVICES,"
                    fi
                    SWAP_DEVICES="$SWAP_DEVICES{\"device\":\"$filename\",\"type\":\"$type\",\"size_bytes\":$size_bytes,\"used_bytes\":$used_bytes,\"priority\":$priority}"
                done
            fi
            ;;
        freebsd)
            # Use swapctl
            if command -v swapctl >/dev/null 2>&1; then
                swap_info=$(swapctl -s 2>/dev/null)
                if [ -n "$swap_info" ]; then
                    SWAP_TOTAL=$(echo "$swap_info" | awk '{print $2 * 1024}')
                    SWAP_USED=$(echo "$swap_info" | awk '{print $3 * 1024}')
                    SWAP_FREE=$((SWAP_TOTAL - SWAP_USED))
                fi
                
                swapctl -l 2>/dev/null | tail -n +2 | while read -r device size used; do
                    size_bytes=$((size * 1024))
                    used_bytes=$((used * 1024))
                    
                    if [ -n "$SWAP_DEVICES" ]; then
                        SWAP_DEVICES="$SWAP_DEVICES,"
                    fi
                    SWAP_DEVICES="$SWAP_DEVICES{\"device\":\"$device\",\"type\":\"partition\",\"size_bytes\":$size_bytes,\"used_bytes\":$used_bytes,\"priority\":0}"
                done
            fi
            ;;
        macos)
            # macOS handles swap differently - get VM info
            if command -v vm_stat >/dev/null 2>&1; then
                vm_info=$(vm_stat 2>/dev/null)
                if [ -n "$vm_info" ]; then
                    page_size=$(echo "$vm_info" | grep "page size" | awk '{print $8}')
                    swapouts=$(echo "$vm_info" | grep "Swapouts:" | awk '{print $2}' | tr -d '.')
                    
                    if [ -n "$page_size" ] && [ -n "$swapouts" ]; then
                        SWAP_USED=$((swapouts * page_size))
                    fi
                fi
            fi
            ;;
    esac
    
    if [ -z "$SWAP_DEVICES" ]; then
        SWAP_DEVICES="[]"
    else
        SWAP_DEVICES="[$SWAP_DEVICES]"
    fi
}

# Get I/O statistics
get_io_stats() {
    IO_STATS=""
    
    case "$OS" in
        linux)
            if [ -f /proc/diskstats ]; then
                grep -E "sd[a-z]$|nvme[0-9]+n[0-9]+$|vd[a-z]$" /proc/diskstats 2>/dev/null | while read -r major minor name reads_completed reads_merged sectors_read time_reading writes_completed writes_merged sectors_written time_writing ios_in_progress time_io weighted_time_io; do
                    read_bytes=$((sectors_read * 512))
                    write_bytes=$((sectors_written * 512))
                    
                    if [ -n "$IO_STATS" ]; then
                        IO_STATS="$IO_STATS,"
                    fi
                    IO_STATS="$IO_STATS{\"device\":\"$name\",\"reads_completed\":$reads_completed,\"writes_completed\":$writes_completed,\"bytes_read\":$read_bytes,\"bytes_written\":$write_bytes,\"time_reading_ms\":$time_reading,\"time_writing_ms\":$time_writing}"
                done
            fi
            ;;
        freebsd)
            # FreeBSD iostat
            if command -v iostat >/dev/null 2>&1; then
                iostat -x 1 1 2>/dev/null | tail -n +3 | grep -v "extended" | while read -r device r_per_s w_per_s kr_per_s kw_per_s ms_per_r ms_per_w ms_per_o queue util; do
                    if [ "$device" != "device" ]; then
                        if [ -n "$IO_STATS" ]; then
                            IO_STATS="$IO_STATS,"
                        fi
                        IO_STATS="$IO_STATS{\"device\":\"$device\",\"reads_per_sec\":$r_per_s,\"writes_per_sec\":$w_per_s,\"kb_read_per_sec\":$kr_per_s,\"kb_write_per_sec\":$kw_per_s,\"ms_per_read\":$ms_per_r,\"ms_per_write\":$ms_per_w,\"queue_depth\":$queue,\"utilization\":$util}"
                    fi
                done
            fi
            ;;
        macos)
            # macOS iostat
            if command -v iostat >/dev/null 2>&1; then
                iostat -d 1 1 2>/dev/null | tail -n +3 | while read -r device r_per_s w_per_s kr_per_s kw_per_s ms_per_r ms_per_w; do
                    if [ "$device" != "disk0" ] || [ -n "$r_per_s" ]; then
                        if [ -n "$IO_STATS" ]; then
                            IO_STATS="$IO_STATS,"
                        fi
                        IO_STATS="$IO_STATS{\"device\":\"$device\",\"reads_per_sec\":$r_per_s,\"writes_per_sec\":$w_per_s,\"kb_read_per_sec\":$kr_per_s,\"kb_write_per_sec\":$kw_per_s,\"ms_per_read\":$ms_per_r,\"ms_per_write\":$ms_per_w}"
                    fi
                done
            fi
            ;;
    esac
    
    if [ -z "$IO_STATS" ]; then
        IO_STATS="[]"
    else
        IO_STATS="[$IO_STATS]"
    fi
}

# Main execution
SUBJECT="$1"

case "$SUBJECT" in
    "system.storage")
        # Gather all storage information
        detect_os
        get_filesystems
        get_block_devices
        get_swap_info
        get_io_stats
        
        # Generate response
        cat <<EOF
{
    "success": true,
    "data": {
        "filesystems": $FILESYSTEMS,
        "block_devices": $BLOCK_DEVICES,
        "swap": {
            "total_bytes": $SWAP_TOTAL,
            "used_bytes": $SWAP_USED,
            "free_bytes": $SWAP_FREE,
            "devices": $SWAP_DEVICES
        },
        "io_stats": $IO_STATS
    },
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "SystemStorageService",
    "endpoint": "GetStorage",
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
    "available_subjects": ["system.storage"],
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "SystemStorageService"
}
EOF
        ;;
esac
