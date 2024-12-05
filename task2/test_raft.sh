#!/bin/bash

# Function to log messages with consistent formatting
log_message() {
    echo "=== $1 ==="
}

go test ./internal/raft -v
