#!/bin/bash

# JSON Processor Service - Demonstrates data processing capabilities
# This service can validate, transform, and query JSON data

if [[ "$1" == "info" ]]; then
    # Return service definition
    cat <<EOF
{
    "name": "JsonProcessorService",
    "version": "1.0.0",
    "description": "JSON data processing and transformation service",
    "endpoints": [
        {
            "name": "ValidateJson",
            "subject": "json.validate",
            "description": "Validate JSON syntax and structure"
        },
        {
            "name": "TransformJson",
            "subject": "json.transform",
            "description": "Transform JSON data using jq expressions"
        },
        {
            "name": "QueryJson",
            "subject": "json.query",
            "description": "Query JSON data with JSONPath-like expressions"
        },
        {
            "name": "FormatJson",
            "subject": "json.format",
            "description": "Format and pretty-print JSON data"
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

# Extract payload if it exists
PAYLOAD=$(echo "$REQUEST_DATA" | jq -r '.payload // empty' 2>/dev/null)

case "$SUBJECT" in
    "json.validate")
        # Validate JSON syntax
        if [[ -z "$PAYLOAD" ]]; then
            echo '{"success": false, "error": "No payload provided for validation"}'
            exit 0
        fi
        
        # Try to parse the payload as JSON
        if echo "$PAYLOAD" | jq . >/dev/null 2>&1; then
            VALIDATION_RESULT="valid"
            ERROR_MESSAGE=""
        else
            VALIDATION_RESULT="invalid"
            ERROR_MESSAGE=$(echo "$PAYLOAD" | jq . 2>&1 | head -1)
        fi
        
        cat <<EOF
{
    "success": true,
    "validation": {
        "result": "$VALIDATION_RESULT",
        "error": "$ERROR_MESSAGE",
        "size": ${#PAYLOAD}
    },
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "JsonProcessorService",
    "endpoint": "ValidateJson",
    "subject": "$SUBJECT"
}
EOF
        ;;
        
    "json.transform")
        # Transform JSON using jq expression
        JQ_EXPR=$(echo "$REQUEST_DATA" | jq -r '.expression // "."')
        
        if [[ -z "$PAYLOAD" ]]; then
            echo '{"success": false, "error": "No payload provided for transformation"}'
            exit 0
        fi
        
        # Apply jq transformation
        if TRANSFORMED=$(echo "$PAYLOAD" | jq "$JQ_EXPR" 2>/dev/null); then
            cat <<EOF
{
    "success": true,
    "transformation": {
        "expression": "$JQ_EXPR",
        "original_size": ${#PAYLOAD},
        "result_size": ${#TRANSFORMED},
        "result": $TRANSFORMED
    },
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "JsonProcessorService",
    "endpoint": "TransformJson",
    "subject": "$SUBJECT"
}
EOF
        else
            cat <<EOF
{
    "success": false,
    "error": "Invalid jq expression or malformed JSON",
    "expression": "$JQ_EXPR",
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "JsonProcessorService",
    "endpoint": "TransformJson",
    "subject": "$SUBJECT"
}
EOF
        fi
        ;;
        
    "json.query")
        # Query JSON data
        QUERY=$(echo "$REQUEST_DATA" | jq -r '.query // "."')
        
        if [[ -z "$PAYLOAD" ]]; then
            echo '{"success": false, "error": "No payload provided for querying"}'
            exit 0
        fi
        
        # Execute query
        if QUERY_RESULT=$(echo "$PAYLOAD" | jq "$QUERY" 2>/dev/null); then
            cat <<EOF
{
    "success": true,
    "query": {
        "expression": "$QUERY",
        "result": $QUERY_RESULT,
        "result_type": "$(echo "$QUERY_RESULT" | jq -r 'type')"
    },
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "JsonProcessorService",
    "endpoint": "QueryJson",
    "subject": "$SUBJECT"
}
EOF
        else
            cat <<EOF
{
    "success": false,
    "error": "Query execution failed",
    "query": "$QUERY",
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "JsonProcessorService",
    "endpoint": "QueryJson",
    "subject": "$SUBJECT"
}
EOF
        fi
        ;;
        
    "json.format")
        # Format and pretty-print JSON
        if [[ -z "$PAYLOAD" ]]; then
            echo '{"success": false, "error": "No payload provided for formatting"}'
            exit 0
        fi
        
        # Format JSON
        if FORMATTED=$(echo "$PAYLOAD" | jq . 2>/dev/null); then
            # Calculate compression ratio
            ORIGINAL_SIZE=${#PAYLOAD}
            FORMATTED_SIZE=${#FORMATTED}
            COMPACT=$(echo "$PAYLOAD" | jq -c . 2>/dev/null)
            COMPACT_SIZE=${#COMPACT}
            
            cat <<EOF
{
    "success": true,
    "formatting": {
        "original_size": $ORIGINAL_SIZE,
        "formatted_size": $FORMATTED_SIZE,
        "compact_size": $COMPACT_SIZE,
        "formatted": $FORMATTED,
        "compact": $(echo "$COMPACT" | jq -R .)
    },
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "JsonProcessorService",
    "endpoint": "FormatJson",
    "subject": "$SUBJECT"
}
EOF
        else
            cat <<EOF
{
    "success": false,
    "error": "Invalid JSON syntax",
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "JsonProcessorService",
    "endpoint": "FormatJson",
    "subject": "$SUBJECT"
}
EOF
        fi
        ;;
        
    *)
        cat <<EOF
{
    "success": false,
    "error": "Unknown subject: $SUBJECT",
    "available_subjects": ["json.validate", "json.transform", "json.query", "json.format"],
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "service": "JsonProcessorService",
    "subject": "$SUBJECT"
}
EOF
        ;;
esac
