#!/bin/bash
# System Users Discovery Service
# Cross-platform user accounts and groups information

if [[ "$1" == "info" ]]; then
    cat <<EOF
{
    "name": "SystemUsersService",
    "version": "1.0.0",
    "description": "User accounts and groups discovery service",
    "endpoints": [
        {
            "name": "GetUsers",
            "subject": "system.users",
            "description": "Get user accounts, groups, and login information"
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

# Get user accounts
get_users() {
    USERS=""
    
    case "$OS" in
        linux|freebsd)
            # Parse /etc/passwd
            while IFS=: read -r username password uid gid gecos home shell; do
                # Skip system users with UID < 1000 (except root)
                if [ "$uid" -lt 1000 ] && [ "$uid" -ne 0 ]; then
                    continue
                fi
                
                # Get last login information
                last_login=""
                if command -v last >/dev/null 2>&1; then
                    last_login=$(last -1 "$username" 2>/dev/null | head -1 | awk '{print $4, $5, $6, $7}' | tr -s ' ')
                fi
                
                # Check if user is locked
                locked="false"
                if [ -f /etc/shadow ] && command -v getent >/dev/null 2>&1; then
                    shadow_entry=$(getent shadow "$username" 2>/dev/null)
                    if echo "$shadow_entry" | grep -q "^[^:]*:!"; then
                        locked="true"
                    fi
                elif echo "$password" | grep -q "!"; then
                    locked="true"
                fi
                
                # Get primary group name
                primary_group=""
                if command -v getent >/dev/null 2>&1; then
                    primary_group=$(getent group "$gid" 2>/dev/null | cut -d: -f1)
                else
                    primary_group=$(grep ":$gid:" /etc/group 2>/dev/null | cut -d: -f1)
                fi
                
                # Get supplementary groups
                supplementary_groups=""
                if command -v id >/dev/null 2>&1; then
                    supplementary_groups=$(id -Gn "$username" 2>/dev/null | tr ' ' ',')
                fi
                
                if [ -n "$USERS" ]; then
                    USERS="$USERS,"
                fi
                USERS="$USERS{\"username\":\"$username\",\"uid\":$uid,\"gid\":$gid,\"gecos\":\"$gecos\",\"home\":\"$home\",\"shell\":\"$shell\",\"primary_group\":\"$primary_group\",\"supplementary_groups\":\"$supplementary_groups\",\"locked\":$locked,\"last_login\":\"$last_login\"}"
            done < /etc/passwd
            ;;
        macos)
            # Use dscl to get user information
            if command -v dscl >/dev/null 2>&1; then
                dscl . list /Users 2>/dev/null | while read -r username; do
                    # Skip system users
                    case "$username" in
                        _*|daemon|nobody|root) 
                            if [ "$username" != "root" ]; then
                                continue
                            fi
                            ;;
                    esac
                    
                    uid=$(dscl . read "/Users/$username" UniqueID 2>/dev/null | awk '{print $2}')
                    gid=$(dscl . read "/Users/$username" PrimaryGroupID 2>/dev/null | awk '{print $2}')
                    gecos=$(dscl . read "/Users/$username" RealName 2>/dev/null | cut -d: -f2- | sed 's/^ *//')
                    home=$(dscl . read "/Users/$username" NFSHomeDirectory 2>/dev/null | awk '{print $2}')
                    shell=$(dscl . read "/Users/$username" UserShell 2>/dev/null | awk '{print $2}')
                    
                    # Skip if UID is not set (system accounts)
                    if [ -z "$uid" ] || [ "$uid" -lt 500 ] && [ "$uid" -ne 0 ]; then
                        continue
                    fi
                    
                    # Get group information
                    primary_group=$(dscl . read "/Groups" PrimaryGroupID "$gid" 2>/dev/null | head -1 | cut -d: -f1 | awk '{print $1}')
                    
                    # Get supplementary groups
                    supplementary_groups=$(id -Gn "$username" 2>/dev/null | tr ' ' ',')
                    
                    # Check if account is disabled
                    locked="false"
                    auth_authority=$(dscl . read "/Users/$username" AuthenticationAuthority 2>/dev/null)
                    if echo "$auth_authority" | grep -q "DisabledUser"; then
                        locked="true"
                    fi
                    
                    if [ -n "$USERS" ]; then
                        USERS="$USERS,"
                    fi
                    USERS="$USERS{\"username\":\"$username\",\"uid\":$uid,\"gid\":$gid,\"gecos\":\"$gecos\",\"home\":\"$home\",\"shell\":\"$shell\",\"primary_group\":\"$primary_group\",\"supplementary_groups\":\"$supplementary_groups\",\"locked\":$locked,\"last_login\":\"\"}"
                done
            fi
            ;;
    esac
    
    if [ -z "$USERS" ]; then
        USERS="[]"
    else
        USERS="[$USERS]"
    fi
}

# Get group information
get_groups() {
    GROUPS=""
    
    case "$OS" in
        linux|freebsd)
            # Parse /etc/group
            while IFS=: read -r groupname password gid members; do
                # Skip system groups with GID < 1000 (except wheel, sudo, admin)
                if [ "$gid" -lt 1000 ]; then
                    case "$groupname" in
                        wheel|sudo|admin|staff|users) ;;
                        *) continue ;;
                    esac
                fi
                
                # Count members
                member_count=0
                if [ -n "$members" ]; then
                    member_count=$(echo "$members" | tr ',' '\n' | wc -l)
                fi
                
                if [ -n "$GROUPS" ]; then
                    GROUPS="$GROUPS,"
                fi
                GROUPS="$GROUPS{\"groupname\":\"$groupname\",\"gid\":$gid,\"members\":\"$members\",\"member_count\":$member_count}"
            done < /etc/group
            ;;
        macos)
            # Use dscl to get group information
            if command -v dscl >/dev/null 2>&1; then
                dscl . list /Groups 2>/dev/null | while read -r groupname; do
                    # Skip system groups
                    case "$groupname" in
                        _*|daemon|nobody) continue ;;
                    esac
                    
                    gid=$(dscl . read "/Groups/$groupname" PrimaryGroupID 2>/dev/null | awk '{print $2}')
                    members=$(dscl . read "/Groups/$groupname" GroupMembership 2>/dev/null | cut -d: -f2- | tr ' ' ',' | sed 's/^,//')
                    
                    # Skip if GID is not set or is a system group
                    if [ -z "$gid" ] || [ "$gid" -lt 20 ]; then
                        case "$groupname" in
                            wheel|admin|staff|everyone) ;;
                            *) continue ;;
                        esac
                    fi
                    
                    # Count members
                    member_count=0
                    if [ -n "$members" ]; then
                        member_count=$(echo "$members" | tr ',' '\n' | wc -l)
                    fi
                    
                    if [ -n "$GROUPS" ]; then
                        GROUPS="$GROUPS,"
                    fi
                    GROUPS="$GROUPS{\"groupname\":\"$groupname\",\"gid\":$gid,\"members\":\"$members\",\"member_count\":$member_count}"
                done
            fi
            ;;
    esac
    
    if [ -z "$GROUPS" ]; then
        GROUPS="[]"
    else
        GROUPS="[$GROUPS]"
    fi
}

# Get currently logged in users
get_logged_in_users() {
    LOGGED_IN_USERS=""
    
    if command -v who >/dev/null 2>&1; then
        who 2>/dev/null | while read -r user tty login_time rest; do
            # Parse login time
            if [ -n "$rest" ]; then
                login_time="$login_time $rest"
            fi
            
            # Get more info with w command if available
            idle_time=""
            what=""
            if command -v w >/dev/null 2>&1; then
                w_info=$(w -h "$user" 2>/dev/null | grep "^$user " | head -1)
                if [ -n "$w_info" ]; then
                    idle_time=$(echo "$w_info" | awk '{print $4}')
                    what=$(echo "$w_info" | awk '{for(i=8;i<=NF;i++) printf "%s ", $i; print ""}' | sed 's/ *$//')
                fi
            fi
            
            if [ -n "$LOGGED_IN_USERS" ]; then
                LOGGED_IN_USERS="$LOGGED_IN_USERS,"
            fi
            LOGGED_IN_USERS="$LOGGED_IN_USERS{\"user\":\"$user\",\"tty\":\"$tty\",\"login_time\":\"$login_time\",\"idle_time\":\"$idle_time\",\"what\":\"$what\"}"
        done
    fi
    
    if [ -z "$LOGGED_IN_USERS" ]; then
        LOGGED_IN_USERS="[]"
    else
        LOGGED_IN_USERS="[$LOGGED_IN_USERS]"
    fi
}

# Get sudo/admin users
get_sudo_users() {
    SUDO_USERS=""
    
    case "$OS" in
        linux)
            # Check /etc/sudoers and sudo group
            if [ -f /etc/sudoers ]; then
                # Get users in sudo/wheel group
                if command -v getent >/dev/null 2>&1; then
                    sudo_group_users=$(getent group sudo 2>/dev/null | cut -d: -f4)
                    wheel_group_users=$(getent group wheel 2>/dev/null | cut -d: -f4)
                else
                    sudo_group_users=$(grep "^sudo:" /etc/group 2>/dev/null | cut -d: -f4)
                    wheel_group_users=$(grep "^wheel:" /etc/group 2>/dev/null | cut -d: -f4)
                fi
                
                all_sudo_users="$sudo_group_users,$wheel_group_users"
                all_sudo_users=$(echo "$all_sudo_users" | tr ',' '\n' | sort -u | grep -v "^$" | tr '\n' ',' | sed 's/,$//')
                
                if [ -n "$all_sudo_users" ]; then
                    SUDO_USERS="[\"$(echo "$all_sudo_users" | sed 's/,/","/g')\"]"
                fi
            fi
            ;;
        freebsd)
            # Check wheel group
            if command -v getent >/dev/null 2>&1; then
                wheel_users=$(getent group wheel 2>/dev/null | cut -d: -f4)
            else
                wheel_users=$(grep "^wheel:" /etc/group 2>/dev/null | cut -d: -f4)
            fi
            
            if [ -n "$wheel_users" ]; then
                SUDO_USERS="[\"$(echo "$wheel_users" | sed 's/,/","/g')\"]"
            fi
            ;;
        macos)
            # Check admin group
            if command -v dscl >/dev/null 2>&1; then
                admin_users=$(dscl . read /Groups/admin GroupMembership 2>/dev/null | cut -d: -f2- | sed 's/^ *//' | tr ' ' ',')
                if [ -n "$admin_users" ]; then
                    SUDO_USERS="[\"$(echo "$admin_users" | sed 's/,/","/g')\"]"
                fi
            fi
            ;;
    esac
    
    if [ -z "$SUDO_USERS" ]; then
        SUDO_USERS="[]"
    fi
}

# Get password policy information
get_password_policy() {
    PASSWORD_POLICY=""
    
    case "$OS" in
        linux)
            # Check /etc/login.defs and /etc/security/pwquality.conf
            max_days=""
            min_days=""
            warn_age=""
            min_length=""
            
            if [ -f /etc/login.defs ]; then
                max_days=$(grep "^PASS_MAX_DAYS" /etc/login.defs 2>/dev/null | awk '{print $2}')
                min_days=$(grep "^PASS_MIN_DAYS" /etc/login.defs 2>/dev/null | awk '{print $2}')
                warn_age=$(grep "^PASS_WARN_AGE" /etc/login.defs 2>/dev/null | awk '{print $2}')
                min_length=$(grep "^PASS_MIN_LEN" /etc/login.defs 2>/dev/null | awk '{print $2}')
            fi
            
            PASSWORD_POLICY="{\"max_days\":\"${max_days:-}\",\"min_days\":\"${min_days:-}\",\"warn_age\":\"${warn_age:-}\",\"min_length\":\"${min_length:-}\"}"
            ;;
        freebsd)
            # Check /etc/login.conf
            if [ -f /etc/login.conf ]; then
                # FreeBSD uses login.conf for password policy
                PASSWORD_POLICY="{\"source\":\"/etc/login.conf\"}"
            else
                PASSWORD_POLICY="{}"
            fi
            ;;
        macos)
            # macOS uses pwpolicy
            if command -v pwpolicy >/dev/null 2>&1; then
                PASSWORD_POLICY="{\"tool\":\"pwpolicy\"}"
            else
                PASSWORD_POLICY="{}"
            fi
            ;;
    esac
}

# Main execution
SUBJECT="$1"

case "$SUBJECT" in
    "system.users")
        # Gather all user information
        detect_os
        get_users
        get_groups
        get_logged_in_users
        get_sudo_users
        get_password_policy
        
        # Generate response
        cat <<EOF
{
    "success": true,
    "data": {
        "users": $USERS,
        "groups": $GROUPS,
        "logged_in_users": $LOGGED_IN_USERS,
        "sudo_users": $SUDO_USERS,
        "password_policy": $PASSWORD_POLICY
    },
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "SystemUsersService",
    "endpoint": "GetUsers",
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
    "available_subjects": ["system.users"],
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "SystemUsersService"
}
EOF
        ;;
esac
