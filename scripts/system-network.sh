#!/bin/bash
# System Network Discovery Service
# Cross-platform network configuration gathering

if [[ "$1" == "info" ]]; then
    cat <<EOF
{
    "name": "SystemService",
    "version": "1.0.0",
    "description": "Network configuration and connectivity discovery",
    "endpoints": [
        {
            "name": "GetNetwork",
            "subject": "system.network",
            "description": "Gather network interfaces, routes, and DNS configuration"
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

# Get network interfaces
get_interfaces() {
    INTERFACES=""
    
    case "$OS" in
        linux)
            # Use ip command if available, fall back to ifconfig
            if command -v ip >/dev/null 2>&1; then
                ip -j addr show 2>/dev/null | jq -c '.[]' 2>/dev/null | while read -r interface_json; do
                    name=$(echo "$interface_json" | jq -r '.ifname')
                    state=$(echo "$interface_json" | jq -r '.operstate // "unknown"')
                    mtu=$(echo "$interface_json" | jq -r '.mtu // 0')
                    mac=$(echo "$interface_json" | jq -r '.address // ""')
                    
                    # Get IP addresses
                    ipv4_addrs=$(echo "$interface_json" | jq -c '[.addr_info[] | select(.family == "inet") | .local]' 2>/dev/null)
                    ipv6_addrs=$(echo "$interface_json" | jq -c '[.addr_info[] | select(.family == "inet6") | .local]' 2>/dev/null)
                    
                    if [ -n "$INTERFACES" ]; then
                        INTERFACES="$INTERFACES,"
                    fi
                    INTERFACES="$INTERFACES{\"name\":\"$name\",\"state\":\"$state\",\"mtu\":$mtu,\"mac\":\"$mac\",\"ipv4\":$ipv4_addrs,\"ipv6\":$ipv6_addrs}"
                done
            else
                # Fall back to parsing ifconfig
                ifconfig -a 2>/dev/null | awk '
                BEGIN { RS = "\n\n"; FS = "\n" }
                /^[a-zA-Z]/ {
                    name = $1; gsub(/:.*/, "", name)
                    state = "unknown"; mtu = 0; mac = ""
                    ipv4 = "[]"; ipv6 = "[]"
                    
                    for (i = 1; i <= NF; i++) {
                        if ($i ~ /inet /) {
                            match($i, /inet ([0-9.]+)/, arr)
                            if (ipv4 == "[]") ipv4 = "[\"" arr[1] "\"]"
                            else { gsub(/\]$/, ",\"" arr[1] "\"]", ipv4) }
                        }
                        if ($i ~ /inet6 /) {
                            match($i, /inet6 ([0-9a-f:]+)/, arr)
                            if (ipv6 == "[]") ipv6 = "[\"" arr[1] "\"]"
                            else { gsub(/\]$/, ",\"" arr[1] "\"]", ipv6) }
                        }
                        if ($i ~ /ether /) {
                            match($i, /ether ([0-9a-f:]+)/, arr)
                            mac = arr[1]
                        }
                        if ($i ~ /mtu /) {
                            match($i, /mtu ([0-9]+)/, arr)
                            mtu = arr[1]
                        }
                        if ($i ~ /UP/) state = "up"
                        else if ($i ~ /DOWN/) state = "down"
                    }
                    
                    if (name != "lo") {
                        if (interfaces != "") interfaces = interfaces ","
                        interfaces = interfaces "{\"name\":\"" name "\",\"state\":\"" state "\",\"mtu\":" mtu ",\"mac\":\"" mac "\",\"ipv4\":" ipv4 ",\"ipv6\":" ipv6 "}"
                    }
                }
                END { print interfaces }'
            fi
            ;;
        freebsd|macos)
            ifconfig -a 2>/dev/null | awk '
            BEGIN { RS = "\n\n"; FS = "\n" }
            /^[a-zA-Z]/ {
                name = $1; gsub(/:.*/, "", name)
                state = "unknown"; mtu = 0; mac = ""
                ipv4 = "[]"; ipv6 = "[]"
                
                for (i = 1; i <= NF; i++) {
                    if ($i ~ /inet /) {
                        match($i, /inet ([0-9.]+)/, arr)
                        if (ipv4 == "[]") ipv4 = "[\"" arr[1] "\"]"
                        else { gsub(/\]$/, ",\"" arr[1] "\"]", ipv4) }
                    }
                    if ($i ~ /inet6 /) {
                        match($i, /inet6 ([0-9a-f:]+)/, arr)
                        if (ipv6 == "[]") ipv6 = "[\"" arr[1] "\"]"
                        else { gsub(/\]$/, ",\"" arr[1] "\"]", ipv6) }
                    }
                    if ($i ~ /ether /) {
                        match($i, /ether ([0-9a-f:]+)/, arr)
                        mac = arr[1]
                    }
                    if ($i ~ /mtu /) {
                        match($i, /mtu ([0-9]+)/, arr)
                        mtu = arr[1]
                    }
                    if ($i ~ /<.*UP.*>/) state = "up"
                    else if ($i ~ /<.*>/ && $i !~ /UP/) state = "down"
                    if ($i ~ /status: active/) state = "active"
                    if ($i ~ /status: inactive/) state = "inactive"
                }
                
                if (name != "lo0") {
                    if (interfaces != "") interfaces = interfaces ","
                    interfaces = interfaces "{\"name\":\"" name "\",\"state\":\"" state "\",\"mtu\":" mtu ",\"mac\":\"" mac "\",\"ipv4\":" ipv4 ",\"ipv6\":" ipv6 "}"
                }
            }
            END { print interfaces }'
            ;;
    esac
    
    if [ -z "$INTERFACES" ]; then
        INTERFACES="[]"
    else
        INTERFACES="[$INTERFACES]"
    fi
}

# Get routing table
get_routes() {
    ROUTES=""
    
    case "$OS" in
        linux)
            if command -v ip >/dev/null 2>&1; then
                ip route show 2>/dev/null | while read -r route; do
                    destination=$(echo "$route" | awk '{print $1}')
                    gateway=$(echo "$route" | grep -o "via [0-9.]*" | awk '{print $2}')
                    interface=$(echo "$route" | grep -o "dev [a-zA-Z0-9]*" | awk '{print $2}')
                    metric=$(echo "$route" | grep -o "metric [0-9]*" | awk '{print $2}')
                    
                    if [ -n "$ROUTES" ]; then
                        ROUTES="$ROUTES,"
                    fi
                    ROUTES="$ROUTES{\"destination\":\"$destination\",\"gateway\":\"${gateway:-}\",\"interface\":\"${interface:-}\",\"metric\":\"${metric:-}\"}"
                done
            else
                route -n 2>/dev/null | tail -n +3 | while read -r dest gateway genmask flags metric ref use iface; do
                    if [ -n "$ROUTES" ]; then
                        ROUTES="$ROUTES,"
                    fi
                    ROUTES="$ROUTES{\"destination\":\"$dest\",\"gateway\":\"$gateway\",\"interface\":\"$iface\",\"metric\":\"$metric\"}"
                done
            fi
            ;;
        freebsd)
            netstat -rn 2>/dev/null | grep "^[0-9]" | while read -r dest gateway flags refs use mtu netif; do
                if [ -n "$ROUTES" ]; then
                    ROUTES="$ROUTES,"
                fi
                ROUTES="$ROUTES{\"destination\":\"$dest\",\"gateway\":\"$gateway\",\"interface\":\"$netif\",\"flags\":\"$flags\"}"
            done
            ;;
        macos)
            netstat -rn 2>/dev/null | grep "^[0-9]" | while read -r dest gateway flags refs use mtu netif expire; do
                if [ -n "$ROUTES" ]; then
                    ROUTES="$ROUTES,"
                fi
                ROUTES="$ROUTES{\"destination\":\"$dest\",\"gateway\":\"$gateway\",\"interface\":\"$netif\",\"flags\":\"$flags\"}"
            done
            ;;
    esac
    
    if [ -z "$ROUTES" ]; then
        ROUTES="[]"
    else
        ROUTES="[$ROUTES]"
    fi
}

# Get DNS configuration
get_dns_config() {
    DNS_SERVERS="[]"
    DNS_SEARCH=""
    
    case "$OS" in
        linux)
            if command -v systemd-resolve >/dev/null 2>&1; then
                # systemd-resolved
                DNS_INFO=$(systemd-resolve --status 2>/dev/null)
                if [ -n "$DNS_INFO" ]; then
                    DNS_SERVERS=$(echo "$DNS_INFO" | grep "DNS Servers:" -A 10 | grep "^         " | awk '{print "\"" $1 "\""}' | tr '\n' ',' | sed 's/,$//' | sed 's/^/[/' | sed 's/$/]/')
                    DNS_SEARCH=$(echo "$DNS_INFO" | grep "DNS Domain:" | awk '{print $3}')
                fi
            elif [ -f /etc/resolv.conf ]; then
                # Traditional resolv.conf
                DNS_SERVERS=$(grep "^nameserver " /etc/resolv.conf | awk '{print "\"" $2 "\""}' | tr '\n' ',' | sed 's/,$//' | sed 's/^/[/' | sed 's/$/]/')
                DNS_SEARCH=$(grep "^search " /etc/resolv.conf | cut -d' ' -f2-)
            fi
            ;;
        freebsd|macos)
            if [ -f /etc/resolv.conf ]; then
                DNS_SERVERS=$(grep "^nameserver " /etc/resolv.conf | awk '{print "\"" $2 "\""}' | tr '\n' ',' | sed 's/,$//' | sed 's/^/[/' | sed 's/$/]/')
                DNS_SEARCH=$(grep "^search " /etc/resolv.conf | cut -d' ' -f2-)
            fi
            ;;
    esac
    
    [ -z "$DNS_SERVERS" ] && DNS_SERVERS="[]"
    [ -z "$DNS_SEARCH" ] && DNS_SEARCH=""
}

# Get default gateway
get_default_gateway() {
    DEFAULT_GATEWAY=""
    
    case "$OS" in
        linux)
            if command -v ip >/dev/null 2>&1; then
                DEFAULT_GATEWAY=$(ip route show default 2>/dev/null | head -1 | grep -o "via [0-9.]*" | awk '{print $2}')
            else
                DEFAULT_GATEWAY=$(route -n 2>/dev/null | grep "^0.0.0.0" | awk '{print $2}' | head -1)
            fi
            ;;
        freebsd|macos)
            DEFAULT_GATEWAY=$(netstat -rn 2>/dev/null | grep "^default" | awk '{print $2}' | head -1)
            ;;
    esac
}

# Get listening ports
get_listening_ports() {
    LISTENING_PORTS=""
    
    case "$OS" in
        linux)
            if command -v ss >/dev/null 2>&1; then
                ss -tlnp 2>/dev/null | tail -n +2 | while read -r state recv_q send_q local_addr foreign_addr process; do
                    port=$(echo "$local_addr" | grep -o ":[0-9]*$" | tr -d ':')
                    addr=$(echo "$local_addr" | sed 's/:[0-9]*$//')
                    proto="tcp"
                    
                    if [ -n "$LISTENING_PORTS" ]; then
                        LISTENING_PORTS="$LISTENING_PORTS,"
                    fi
                    LISTENING_PORTS="$LISTENING_PORTS{\"protocol\":\"$proto\",\"address\":\"$addr\",\"port\":$port,\"process\":\"$process\"}"
                done
                
                ss -ulnp 2>/dev/null | tail -n +2 | while read -r state recv_q send_q local_addr foreign_addr process; do
                    port=$(echo "$local_addr" | grep -o ":[0-9]*$" | tr -d ':')
                    addr=$(echo "$local_addr" | sed 's/:[0-9]*$//')
                    proto="udp"
                    
                    if [ -n "$LISTENING_PORTS" ]; then
                        LISTENING_PORTS="$LISTENING_PORTS,"
                    fi
                    LISTENING_PORTS="$LISTENING_PORTS{\"protocol\":\"$proto\",\"address\":\"$addr\",\"port\":$port,\"process\":\"$process\"}"
                done
            elif command -v netstat >/dev/null 2>&1; then
                netstat -tlnp 2>/dev/null | grep "LISTEN" | while read -r proto recv_q send_q local_addr foreign_addr state process; do
                    port=$(echo "$local_addr" | grep -o ":[0-9]*$" | tr -d ':')
                    addr=$(echo "$local_addr" | sed 's/:[0-9]*$//')
                    
                    if [ -n "$LISTENING_PORTS" ]; then
                        LISTENING_PORTS="$LISTENING_PORTS,"
                    fi
                    LISTENING_PORTS="$LISTENING_PORTS{\"protocol\":\"tcp\",\"address\":\"$addr\",\"port\":$port,\"process\":\"$process\"}"
                done
            fi
            ;;
        freebsd|macos)
            if command -v netstat >/dev/null 2>&1; then
                netstat -an 2>/dev/null | grep "LISTEN" | while read -r proto recv_q send_q local_addr foreign_addr state; do
                    port=$(echo "$local_addr" | grep -o "\.[0-9]*$" | tr -d '.')
                    addr=$(echo "$local_addr" | sed 's/\.[0-9]*$//')
                    proto_name="tcp"
                    
                    if [ -n "$LISTENING_PORTS" ]; then
                        LISTENING_PORTS="$LISTENING_PORTS,"
                    fi
                    LISTENING_PORTS="$LISTENING_PORTS{\"protocol\":\"$proto_name\",\"address\":\"$addr\",\"port\":$port,\"process\":\"\"}"
                done
            fi
            ;;
    esac
    
    if [ -z "$LISTENING_PORTS" ]; then
        LISTENING_PORTS="[]"
    else
        LISTENING_PORTS="[$LISTENING_PORTS]"
    fi
}

# Main execution
SUBJECT="$1"

case "$SUBJECT" in
    "system.network")
        # Gather all network information
        detect_os
        get_interfaces
        get_routes
        get_dns_config
        get_default_gateway
        get_listening_ports
        
        # Generate response
        cat <<EOF
{
    "success": true,
    "data": {
        "interfaces": $INTERFACES,
        "routes": $ROUTES,
        "dns": {
            "servers": $DNS_SERVERS,
            "search": "$DNS_SEARCH"
        },
        "default_gateway": "${DEFAULT_GATEWAY:-}",
        "listening_ports": $LISTENING_PORTS
    },
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "SystemNetworkService",
    "endpoint": "GetNetwork",
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
    "available_subjects": ["system.network"],
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "SystemNetworkService"
}
EOF
        ;;
esac
