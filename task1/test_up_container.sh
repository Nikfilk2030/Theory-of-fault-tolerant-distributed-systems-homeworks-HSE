#!/bin/bash

# Function to log messages with consistent formatting
log_message() {
    echo "=== $1 ==="
}

# Set up a cleanup function
cleanup() {
    log_message "Tearing down container"
    docker compose down
    exit
}

# Ensure cleanup happens on script exit or interruption
trap cleanup EXIT INT TERM

log_message "TEST_CASE: up container, check that simply works"

log_message "STARTING CONTAINER"
docker compose build
docker compose up -d

sleep 2

# Check the container's status
log_message "Checking container status"
if ! docker compose ps; then
    log_message "Error: The container is not running properly"
    exit 1
fi

sleep 40

log_message "LOGS"
docker compose logs

log_message "ENDING CONTAINER"
docker compose down
