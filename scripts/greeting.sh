#!/bin/bash
# Example NATS microservice script for natshd
# This script demonstrates how to create a simple greeting service
# Modified to test file watcher functionality

# Define the service when called with info argument
if [[ "$1" == "info" ]]; then
    cat << 'EOF'
{
    "name": "GreetingService",
    "description": "A simple greeting microservice that personalizes greetings",
    "version": "1.0.0",
    "endpoints": [
        {
            "name": "Greet",
            "subject": "greeting.greet",
            "description": "Generates a personalized greeting message"
        },
        {
            "name": "Farewell", 
            "subject": "greeting.farewell",
            "description": "Generates a farewell message"
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
    "greeting.greet")
        # Extract name from JSON request, default to "World" if not provided
        NAME=$(echo "$REQUEST" | jq -r '.name // "World"')
        GREETING=$(echo "$REQUEST" | jq -r '.greeting // "Hello"')
        
        # Generate response
        cat << EOF
{
    "success": true,
    "message": "$GREETING, $NAME! Welcome to the NATS Shell Daemon service.",
    "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "service": "GreetingService",
    "endpoint": "Greet",
    "subject": "$SUBJECT"
}
EOF
        ;;
    
    "greeting.farewell")
        # Extract name from JSON request
        NAME=$(echo "$REQUEST" | jq -r '.name // "Friend"')
        
        # Generate response
        cat << EOF
{
    "success": true,
    "message": "Goodbye, $NAME! Thank you for using our service.",
    "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "service": "GreetingService", 
    "endpoint": "Farewell",
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
    "available_subjects": ["greeting.greet", "greeting.farewell"],
    "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "service": "GreetingService"
}
EOF
        exit 1
        ;;
esac
