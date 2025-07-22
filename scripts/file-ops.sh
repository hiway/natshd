#!/bin/bash

# File Operations Service - Demonstrates file system operations
# This service can read, write, and inspect files safely

if [[ "$1" == "info" ]]; then
    # Return service definition
    cat <<EOF
{
    "name": "FileOperationsService",
    "version": "1.0.0",
    "description": "Safe file system operations service",
    "endpoints": [
        {
            "name": "ReadFile",
            "subject": "file.read",
            "description": "Read file contents with size limits"
        },
        {
            "name": "WriteFile",
            "subject": "file.write",
            "description": "Write data to file (restricted to /tmp)"
        },
        {
            "name": "FileInfo",
            "subject": "file.info",
            "description": "Get file metadata and statistics"
        },
        {
            "name": "ListDirectory",
            "subject": "file.list",
            "description": "List directory contents with filtering"
        },
        {
            "name": "HashFile",
            "subject": "file.hash",
            "description": "Calculate file checksums (MD5, SHA256)"
        }
    ]
}
EOF
    exit 0
fi

# Get the subject from the first argument
SUBJECT="$1"
echo "Processing request for subject: $SUBJECT" >&2

# Read JSON input from stdin
REQUEST_DATA=$(cat)
echo "Request data: $REQUEST_DATA" >&2

# Safety function to check if path is allowed
is_safe_path() {
    local path="$1"
    # Only allow reading from current directory and /tmp for writing
    case "$path" in
        /tmp/*|./*)
            return 0
            ;;
        *)
            if [[ "$SUBJECT" == "file.write" ]]; then
                return 1  # Writing only allowed in /tmp
            elif [[ -f "$path" && "$path" != *".."* ]]; then
                return 0  # Reading allowed for existing files without ..
            else
                return 1
            fi
            ;;
    esac
}

case "$SUBJECT" in
    "file.read")
        # Read file contents
        FILEPATH=$(echo "$REQUEST_DATA" | jq -r '.filepath // empty')
        MAX_SIZE=$(echo "$REQUEST_DATA" | jq -r '.max_size // 10240')  # Default 10KB
        
        if [[ -z "$FILEPATH" ]]; then
            echo '{"success": false, "error": "No filepath provided"}'
            exit 0
        fi
        
        if ! is_safe_path "$FILEPATH"; then
            echo '{"success": false, "error": "Access denied: unsafe path"}'
            exit 0
        fi
        
        if [[ ! -f "$FILEPATH" ]]; then
            echo '{"success": false, "error": "File not found"}'
            exit 0
        fi
        
        # Check file size
        FILE_SIZE=$(stat -c%s "$FILEPATH" 2>/dev/null || echo 0)
        if [[ $FILE_SIZE -gt $MAX_SIZE ]]; then
            cat <<EOF
{
    "success": false,
    "error": "File too large",
    "file_size": $FILE_SIZE,
    "max_size": $MAX_SIZE,
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "FileOperationsService",
    "endpoint": "ReadFile",
    "subject": "$SUBJECT"
}
EOF
            exit 0
        fi
        
        # Read file content
        CONTENT=$(cat "$FILEPATH" | base64 -w 0)
        cat <<EOF
{
    "success": true,
    "file_read": {
        "filepath": "$FILEPATH",
        "size": $FILE_SIZE,
        "content_base64": "$CONTENT",
        "encoding": "base64"
    },
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "FileOperationsService",
    "endpoint": "ReadFile",
    "subject": "$SUBJECT"
}
EOF
        ;;
        
    "file.write")
        # Write file contents
        FILEPATH=$(echo "$REQUEST_DATA" | jq -r '.filepath // empty')
        CONTENT=$(echo "$REQUEST_DATA" | jq -r '.content // empty')
        ENCODING=$(echo "$REQUEST_DATA" | jq -r '.encoding // "text"')
        
        if [[ -z "$FILEPATH" || -z "$CONTENT" ]]; then
            echo '{"success": false, "error": "Filepath and content required"}'
            exit 0
        fi
        
        if ! is_safe_path "$FILEPATH"; then
            echo '{"success": false, "error": "Access denied: can only write to /tmp"}'
            exit 0
        fi
        
        # Decode content if base64
        if [[ "$ENCODING" == "base64" ]]; then
            if ! echo "$CONTENT" | base64 -d > "$FILEPATH" 2>/dev/null; then
                echo '{"success": false, "error": "Failed to decode base64 content"}'
                exit 0
            fi
        else
            echo "$CONTENT" > "$FILEPATH"
        fi
        
        WRITTEN_SIZE=$(stat -c%s "$FILEPATH" 2>/dev/null || echo 0)
        cat <<EOF
{
    "success": true,
    "file_write": {
        "filepath": "$FILEPATH",
        "bytes_written": $WRITTEN_SIZE,
        "encoding": "$ENCODING"
    },
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "FileOperationsService",
    "endpoint": "WriteFile",
    "subject": "$SUBJECT"
}
EOF
        ;;
        
    "file.info")
        # Get file information
        FILEPATH=$(echo "$REQUEST_DATA" | jq -r '.filepath // empty')
        
        if [[ -z "$FILEPATH" ]]; then
            echo '{"success": false, "error": "No filepath provided"}'
            exit 0
        fi
        
        if [[ ! -e "$FILEPATH" ]]; then
            echo '{"success": false, "error": "File or directory not found"}'
            exit 0
        fi
        
        # Get file stats
        SIZE=$(stat -c%s "$FILEPATH" 2>/dev/null || echo 0)
        MTIME=$(stat -c%Y "$FILEPATH" 2>/dev/null || echo 0)
        PERMISSIONS=$(stat -c%a "$FILEPATH" 2>/dev/null || echo "000")
        OWNER=$(stat -c%U "$FILEPATH" 2>/dev/null || echo "unknown")
        GROUP=$(stat -c%G "$FILEPATH" 2>/dev/null || echo "unknown")
        
        if [[ -f "$FILEPATH" ]]; then
            FILE_TYPE="file"
        elif [[ -d "$FILEPATH" ]]; then
            FILE_TYPE="directory"
        elif [[ -L "$FILEPATH" ]]; then
            FILE_TYPE="symlink"
        else
            FILE_TYPE="other"
        fi
        
        cat <<EOF
{
    "success": true,
    "file_info": {
        "filepath": "$FILEPATH",
        "type": "$FILE_TYPE",
        "size": $SIZE,
        "permissions": "$PERMISSIONS",
        "owner": "$OWNER",
        "group": "$GROUP",
        "modified_time": $MTIME,
        "modified_time_iso": "$(date -u -d @$MTIME +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo 'unknown')"
    },
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "FileOperationsService",
    "endpoint": "FileInfo",
    "subject": "$SUBJECT"
}
EOF
        ;;
        
    "file.list")
        # List directory contents
        DIRPATH=$(echo "$REQUEST_DATA" | jq -r '.dirpath // "."')
        PATTERN=$(echo "$REQUEST_DATA" | jq -r '.pattern // "*"')
        
        if [[ ! -d "$DIRPATH" ]]; then
            echo '{"success": false, "error": "Directory not found"}'
            exit 0
        fi
        
        # List files matching pattern
        FILES_JSON="["
        FIRST=true
        while IFS= read -r -d '' file; do
            if [[ "$FIRST" == true ]]; then
                FIRST=false
            else
                FILES_JSON+=","
            fi
            
            BASENAME=$(basename "$file")
            SIZE=$(stat -c%s "$file" 2>/dev/null || echo 0)
            MTIME=$(stat -c%Y "$file" 2>/dev/null || echo 0)
            
            if [[ -f "$file" ]]; then
                TYPE="file"
            elif [[ -d "$file" ]]; then
                TYPE="directory"
            else
                TYPE="other"
            fi
            
            FILES_JSON+="{\"name\":\"$BASENAME\",\"type\":\"$TYPE\",\"size\":$SIZE,\"mtime\":$MTIME}"
        done < <(find "$DIRPATH" -maxdepth 1 -name "$PATTERN" -print0 2>/dev/null)
        FILES_JSON+="]"
        
        cat <<EOF
{
    "success": true,
    "directory_listing": {
        "dirpath": "$DIRPATH",
        "pattern": "$PATTERN",
        "files": $FILES_JSON
    },
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "FileOperationsService",
    "endpoint": "ListDirectory",
    "subject": "$SUBJECT"
}
EOF
        ;;
        
    "file.hash")
        # Calculate file hash
        FILEPATH=$(echo "$REQUEST_DATA" | jq -r '.filepath // empty')
        ALGORITHM=$(echo "$REQUEST_DATA" | jq -r '.algorithm // "sha256"')
        
        if [[ -z "$FILEPATH" ]]; then
            echo '{"success": false, "error": "No filepath provided"}'
            exit 0
        fi
        
        if [[ ! -f "$FILEPATH" ]]; then
            echo '{"success": false, "error": "File not found"}'
            exit 0
        fi
        
        case "$ALGORITHM" in
            "md5")
                HASH=$(md5sum "$FILEPATH" | cut -d' ' -f1)
                ;;
            "sha1")
                HASH=$(sha1sum "$FILEPATH" | cut -d' ' -f1)
                ;;
            "sha256")
                HASH=$(sha256sum "$FILEPATH" | cut -d' ' -f1)
                ;;
            *)
                echo '{"success": false, "error": "Unsupported algorithm", "supported": ["md5", "sha1", "sha256"]}'
                exit 0
                ;;
        esac
        
        SIZE=$(stat -c%s "$FILEPATH" 2>/dev/null || echo 0)
        cat <<EOF
{
    "success": true,
    "file_hash": {
        "filepath": "$FILEPATH",
        "algorithm": "$ALGORITHM",
        "hash": "$HASH",
        "file_size": $SIZE
    },
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "FileOperationsService",
    "endpoint": "HashFile",
    "subject": "$SUBJECT"
}
EOF
        ;;
        
    *)
        cat <<EOF
{
    "success": false,
    "error": "Unknown subject: $SUBJECT",
    "available_subjects": ["file.read", "file.write", "file.info", "file.list", "file.hash"],
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "FileOperationsService",
    "subject": "$SUBJECT"
}
EOF
        ;;
esac
