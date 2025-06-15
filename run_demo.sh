#!/bin/bash

# WebRTC File Streaming Demo
# This script builds and runs the WebRTC file streaming server and client

# Exit on error
set -e

# Directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Create bin directory if it doesn't exist
mkdir -p bin

# Build the server and client
echo "Building server and client..."
go build -o bin/server cmd/server/main.go
go build -o bin/client cmd/client/main.go

# File to store PIDs
PID_FILE="webrtc_demo.pid"

# Function to clean up on exit
cleanup() {
    echo "Shutting down WebRTC demo..."
    if [ -f "$PID_FILE" ]; then
        while read -r pid; do
            if ps -p "$pid" > /dev/null; then
                echo "Killing process $pid"
                kill "$pid"
            fi
        done < "$PID_FILE"
        rm "$PID_FILE"
    fi
    echo "Shutdown complete"
}

# Register the cleanup function to be called on exit
trap cleanup EXIT

# Remove any existing PID file
if [ -f "$PID_FILE" ]; then
    rm "$PID_FILE"
fi

# Start the server in the background
echo "Starting server..."
bin/webrtc-poc server --addr ":8081" --file sample.txt --delay 500 > server.log 2>&1 &
SERVER_PID=$!
echo "Server started with PID: $SERVER_PID"
echo "$SERVER_PID" > "$PID_FILE"

# Wait for the server to start
sleep 2

# Start the client in the background
echo "Starting client..."
bin/webrtc-poc client --server "http://localhost:8081/offer" > client.log 2>&1 &
CLIENT_PID=$!
echo "Client started with PID: $CLIENT_PID"
echo "$CLIENT_PID" >> "$PID_FILE"

echo "WebRTC demo is running..."
echo "Server log: $SCRIPT_DIR/server.log"
echo "Client log: $SCRIPT_DIR/client.log"
echo "Press Ctrl+C to stop the demo"

# Wait for user to press Ctrl+C
wait
