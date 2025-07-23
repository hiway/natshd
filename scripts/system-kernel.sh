#!/bin/bash
# System Kernel Discovery Service
# Cross-platform kernel information gathering

if [[ "$1" == "info" ]]; then
    cat <<EOF
{
    "name": "SystemService",
    "version": "1.0.0",
    "description": "Kernel version, modules, and parameters discovery",
    "endpoints": [
        {
            "name": "GetKernel",
            "subject": "system.kernel",
            "description": "Get kernel version, loaded modules, and system parameters",
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

# Get kernel version information
get_kernel_info() {
    KERNEL_NAME=$(uname -s)
    KERNEL_RELEASE=$(uname -r)
    KERNEL_VERSION=$(uname -v)
    KERNEL_MACHINE=$(uname -m)
    KERNEL_PROCESSOR=$(uname -p 2>/dev/null || echo "unknown")
    
    case "$OS" in
        linux)
            # Get additional Linux kernel info
            if [ -f /proc/version ]; then
                KERNEL_VERSION_FULL=$(cat /proc/version)
                KERNEL_COMPILER=$(echo "$KERNEL_VERSION_FULL" | grep -o "gcc version [^)]*" | head -1)
            fi
            
            # Get kernel command line
            if [ -f /proc/cmdline ]; then
                KERNEL_CMDLINE=$(cat /proc/cmdline)
            fi
            
            # Get boot time
            if [ -f /proc/stat ]; then
                BOOT_TIME=$(grep "^btime " /proc/stat | awk '{print $2}')
                if [ -n "$BOOT_TIME" ]; then
                    BOOT_TIME_READABLE=$(date -d "@$BOOT_TIME" "+%Y-%m-%d %H:%M:%S" 2>/dev/null)
                fi
            fi
            ;;
        freebsd)
            # Get FreeBSD kernel info
            KERNEL_IDENT=$(sysctl -n kern.ident 2>/dev/null)
            KERNEL_HOSTNAME=$(sysctl -n kern.hostname 2>/dev/null)
            BOOT_TIME=$(sysctl -n kern.boottime 2>/dev/null | sed 's/.*sec = \([0-9]*\).*/\1/')
            if [ -n "$BOOT_TIME" ]; then
                BOOT_TIME_READABLE=$(date -r "$BOOT_TIME" "+%Y-%m-%d %H:%M:%S" 2>/dev/null)
            fi
            ;;
        macos)
            # Get macOS kernel info
            KERNEL_IDENT=$(sysctl -n kern.version 2>/dev/null | head -1)
            KERNEL_HOSTNAME=$(sysctl -n kern.hostname 2>/dev/null)
            BOOT_TIME=$(sysctl -n kern.boottime 2>/dev/null | sed 's/.*sec = \([0-9]*\).*/\1/')
            if [ -n "$BOOT_TIME" ]; then
                BOOT_TIME_READABLE=$(date -r "$BOOT_TIME" "+%Y-%m-%d %H:%M:%S" 2>/dev/null)
            fi
            ;;
    esac
}

# Get loaded kernel modules
get_kernel_modules() {
    KERNEL_MODULES=""
    
    case "$OS" in
        linux)
            if [ -f /proc/modules ]; then
                while read -r name size used_by_count used_by state load_addr; do
                    # Clean up used_by field
                    used_by_clean=$(echo "$used_by" | tr -d '-' | tr ',' ' ')
                    
                    if [ -n "$KERNEL_MODULES" ]; then
                        KERNEL_MODULES="$KERNEL_MODULES,"
                    fi
                    KERNEL_MODULES="$KERNEL_MODULES{\"name\":\"$name\",\"size\":$size,\"used_by_count\":$used_by_count,\"used_by\":\"$used_by_clean\",\"state\":\"$state\"}"
                done < /proc/modules
            elif command -v lsmod >/dev/null 2>&1; then
                lsmod | tail -n +2 | while read -r name size used_by; do
                    used_by_count=$(echo "$used_by" | tr ',' '\n' | wc -l)
                    if [ "$used_by" = "-" ]; then
                        used_by_count=0
                        used_by=""
                    fi
                    
                    if [ -n "$KERNEL_MODULES" ]; then
                        KERNEL_MODULES="$KERNEL_MODULES,"
                    fi
                    KERNEL_MODULES="$KERNEL_MODULES{\"name\":\"$name\",\"size\":$size,\"used_by_count\":$used_by_count,\"used_by\":\"$used_by\",\"state\":\"Live\"}"
                done
            fi
            ;;
        freebsd)
            if command -v kldstat >/dev/null 2>&1; then
                kldstat | tail -n +2 | while read -r id refs address size name; do
                    if [ -n "$KERNEL_MODULES" ]; then
                        KERNEL_MODULES="$KERNEL_MODULES,"
                    fi
                    KERNEL_MODULES="$KERNEL_MODULES{\"name\":\"$name\",\"size\":\"$size\",\"refs\":$refs,\"address\":\"$address\",\"id\":$id}"
                done
            fi
            ;;
        macos)
            if command -v kextstat >/dev/null 2>&1; then
                kextstat | tail -n +2 | while read -r index refs address size wired architecture name version; do
                    if [ -n "$KERNEL_MODULES" ]; then
                        KERNEL_MODULES="$KERNEL_MODULES,"
                    fi
                    KERNEL_MODULES="$KERNEL_MODULES{\"name\":\"$name\",\"version\":\"$version\",\"size\":\"$size\",\"refs\":$refs,\"address\":\"$address\",\"architecture\":\"$architecture\"}"
                done
            fi
            ;;
    esac
    
    if [ -z "$KERNEL_MODULES" ]; then
        KERNEL_MODULES="[]"
    else
        KERNEL_MODULES="[$KERNEL_MODULES]"
    fi
}

# Get kernel parameters
get_kernel_parameters() {
    KERNEL_PARAMETERS=""
    
    case "$OS" in
        linux)
            # Get sysctl parameters
            if command -v sysctl >/dev/null 2>&1; then
                # Get some important kernel parameters
                important_params="kernel.hostname kernel.ostype kernel.osrelease kernel.version vm.swappiness vm.dirty_ratio fs.file-max net.core.somaxconn"
                
                for param in $important_params; do
                    value=$(sysctl -n "$param" 2>/dev/null)
                    if [ -n "$value" ]; then
                        if [ -n "$KERNEL_PARAMETERS" ]; then
                            KERNEL_PARAMETERS="$KERNEL_PARAMETERS,"
                        fi
                        KERNEL_PARAMETERS="$KERNEL_PARAMETERS{\"parameter\":\"$param\",\"value\":\"$value\"}"
                    fi
                done
            fi
            ;;
        freebsd)
            # Get sysctl parameters
            if command -v sysctl >/dev/null 2>&1; then
                # Get some important kernel parameters
                important_params="kern.hostname kern.ostype kern.osrelease kern.version vm.swap_total kern.maxfiles kern.maxproc"
                
                for param in $important_params; do
                    value=$(sysctl -n "$param" 2>/dev/null)
                    if [ -n "$value" ]; then
                        if [ -n "$KERNEL_PARAMETERS" ]; then
                            KERNEL_PARAMETERS="$KERNEL_PARAMETERS,"
                        fi
                        KERNEL_PARAMETERS="$KERNEL_PARAMETERS{\"parameter\":\"$param\",\"value\":\"$value\"}"
                    fi
                done
            fi
            ;;
        macos)
            # Get sysctl parameters
            if command -v sysctl >/dev/null 2>&1; then
                # Get some important kernel parameters
                important_params="kern.hostname kern.ostype kern.osrelease kern.version kern.maxfiles kern.maxproc hw.memsize hw.ncpu"
                
                for param in $important_params; do
                    value=$(sysctl -n "$param" 2>/dev/null)
                    if [ -n "$value" ]; then
                        if [ -n "$KERNEL_PARAMETERS" ]; then
                            KERNEL_PARAMETERS="$KERNEL_PARAMETERS,"
                        fi
                        KERNEL_PARAMETERS="$KERNEL_PARAMETERS{\"parameter\":\"$param\",\"value\":\"$value\"}"
                    fi
                done
            fi
            ;;
    esac
    
    if [ -z "$KERNEL_PARAMETERS" ]; then
        KERNEL_PARAMETERS="[]"
    else
        KERNEL_PARAMETERS="[$KERNEL_PARAMETERS]"
    fi
}

# Get security features
get_security_features() {
    SECURITY_FEATURES=""
    
    case "$OS" in
        linux)
            # Check various security features
            selinux_status="disabled"
            apparmor_status="disabled"
            
            # Check SELinux
            if command -v getenforce >/dev/null 2>&1; then
                selinux_status=$(getenforce 2>/dev/null | tr '[:upper:]' '[:lower:]')
            elif [ -f /selinux/enforce ]; then
                enforce=$(cat /selinux/enforce 2>/dev/null)
                case "$enforce" in
                    1) selinux_status="enforcing" ;;
                    0) selinux_status="permissive" ;;
                esac
            fi
            
            # Check AppArmor
            if command -v aa-status >/dev/null 2>&1; then
                if aa-status --enabled 2>/dev/null; then
                    apparmor_status="enabled"
                fi
            elif [ -f /sys/kernel/security/apparmor/profiles ]; then
                apparmor_status="enabled"
            fi
            
            # Check if ASLR is enabled
            aslr_status="unknown"
            if [ -f /proc/sys/kernel/randomize_va_space ]; then
                aslr_val=$(cat /proc/sys/kernel/randomize_va_space)
                case "$aslr_val" in
                    0) aslr_status="disabled" ;;
                    1) aslr_status="conservative" ;;
                    2) aslr_status="full" ;;
                esac
            fi
            
            SECURITY_FEATURES="{\"selinux\":\"$selinux_status\",\"apparmor\":\"$apparmor_status\",\"aslr\":\"$aslr_status\"}"
            ;;
        freebsd)
            # Check FreeBSD security features
            jail_enable="unknown"
            if sysctl -n security.jail.jailed >/dev/null 2>&1; then
                jail_val=$(sysctl -n security.jail.jailed 2>/dev/null)
                if [ "$jail_val" = "1" ]; then
                    jail_enable="yes"
                else
                    jail_enable="no"
                fi
            fi
            
            SECURITY_FEATURES="{\"jail\":\"$jail_enable\"}"
            ;;
        macos)
            # Check macOS security features
            sip_status="unknown"
            if command -v csrutil >/dev/null 2>&1; then
                sip_output=$(csrutil status 2>/dev/null)
                if echo "$sip_output" | grep -q "enabled"; then
                    sip_status="enabled"
                elif echo "$sip_output" | grep -q "disabled"; then
                    sip_status="disabled"
                fi
            fi
            
            SECURITY_FEATURES="{\"system_integrity_protection\":\"$sip_status\"}"
            ;;
    esac
}

# Main execution
SUBJECT="$1"

case "$SUBJECT" in
    "system.kernel")
        # Gather all kernel information
        detect_os
        get_kernel_info
        get_kernel_modules
        get_kernel_parameters
        get_security_features
        
        # Generate response
        cat <<EOF
{
    "success": true,
    "data": {
        "kernel": {
            "name": "${KERNEL_NAME:-unknown}",
            "release": "${KERNEL_RELEASE:-unknown}",
            "version": "${KERNEL_VERSION:-unknown}",
            "machine": "${KERNEL_MACHINE:-unknown}",
            "processor": "${KERNEL_PROCESSOR:-unknown}",
            "version_full": "${KERNEL_VERSION_FULL:-}",
            "compiler": "${KERNEL_COMPILER:-}",
            "cmdline": "${KERNEL_CMDLINE:-}",
            "ident": "${KERNEL_IDENT:-}",
            "hostname": "${KERNEL_HOSTNAME:-}",
            "boot_time": "${BOOT_TIME:-}",
            "boot_time_readable": "${BOOT_TIME_READABLE:-}"
        },
        "modules": $KERNEL_MODULES,
        "parameters": $KERNEL_PARAMETERS,
        "security": $SECURITY_FEATURES
    },
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "SystemKernelService",
    "endpoint": "GetKernel",
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
    "available_subjects": ["system.kernel"],
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "SystemKernelService"
}
EOF
        ;;
esac
