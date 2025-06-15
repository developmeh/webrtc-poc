# WebRTC File Streaming Proof of Concept

This is a proof of concept for using WebRTC to stream a file line by line. The implementation is kept as succinct as possible while still being functional.

**[Detailed explanation of WebRTC connection process without STUN servers](WEBRTC_CONNECTION_DETAILS.md)**

## Overview

This project demonstrates how to use WebRTC data channels to stream a text file line by line from a server to a client. The server reads the file and sends each line to the client with a configurable delay. The client receives the lines and either displays them on the console or writes them to a file.

## Features

- WebRTC data channel communication
- Line-by-line file streaming with configurable delay
- Comprehensive logging
- Unified command-line interface using cobra
- Configuration via YAML file using viper
- Support for STUN servers for NAT traversal
- Works with or without a STUN server
- Automatic server and client shutdown

## Requirements

- Go 1.16 or later
- Bash (for running the demo script)

## Building

To build the project, run:

```bash
make build
```

This will create the server and client executables in the `bin` directory.

## Running the Demo

To run the demo, use:

```bash
make run
```

Or directly:

```bash
./run_demo.sh
```

The script will:
1. Build the server and client executables
2. Start the server in the background
3. Start the client in the background
4. Record the PIDs for automatic shutdown
5. Wait for the user to press Ctrl+C to stop the demo

## Command-Line Interface

The application now uses a unified CLI with cobra and viper for configuration management.

### Global Options

```
Usage:
  webrtc-poc [command]

Available Commands:
  client      Start the WebRTC file streaming client
  help        Help about any command
  server      Start the WebRTC file streaming server

Flags:
  --config string   config file (default is ./config.yaml)
  -h, --help        help for webrtc-poc
```

### Server Command

```
Usage:
  webrtc-poc server [flags]

Flags:
  --addr string    HTTP service address (default ":8080")
  --delay int      Delay between lines in milliseconds (default 1000)
  --file string    File to stream (default "sample.txt")
  -h, --help       help for server
  --stun string    STUN server address (leave empty for direct connection)
```

### Client Command

```
Usage:
  webrtc-poc client [flags]

Flags:
  -h, --help            help for client
  --output string       Output file (leave empty for stdout)
  --server string       WebRTC server URL (default "http://localhost:8080/offer")
  --stun string         STUN server address (leave empty for direct connection)
```

### Configuration File

You can also use a configuration file (YAML format) to set options. By default, the application looks for a file named `config.yaml` in the current directory. You can specify a different file using the `--config` flag.

Example configuration file:

```yaml
# Server configuration
server:
  addr: ":8080"
  file: "sample.txt"
  delay: 1000
  stun: "stun:stun.l.google.com:19302"  # Optional STUN server

# Client configuration
client:
  server: "http://localhost:8080/offer"
  output: "output.txt"  # Optional output file
  stun: "stun:stun.l.google.com:19302"  # Optional STUN server
```

## Manual Execution

If you want to run the server and client manually:

1. Start the server:
   ```bash
   bin/webrtc-poc server --file sample.txt --delay 500
   ```

2. In another terminal, start the client:
   ```bash
   bin/webrtc-poc client --output output.txt
   ```

3. Using a configuration file:
   ```bash
   bin/webrtc-poc server --config config.yaml
   bin/webrtc-poc client --config config.yaml
   ```

4. Using a STUN server for NAT traversal:
   ```bash
   bin/webrtc-poc server --stun "stun:stun.l.google.com:19302"
   bin/webrtc-poc client --stun "stun:stun.l.google.com:19302"
   ```

## Testing WebRTC Connection Establishment

To verify that WebRTC connection state monitoring works correctly, you can run the test program:

```bash
make test-connection
```

Or manually:

```bash
go build -o bin/test cmd/test/main.go
./bin/test
```

This program creates both server and client peer connections in the same process and connects them directly, bypassing the HTTP signaling mechanism. It demonstrates how to monitor WebRTC connection states and shows the expected log output when a connection is successfully established.

## Cleaning Up

To clean up build artifacts and logs:

```bash
make clean
```

## Implementation Details

- The application uses a unified CLI with cobra and viper for configuration management
- Configuration can be provided via command-line flags or a YAML configuration file
- The server uses an HTTP endpoint to exchange WebRTC signaling information
- The client connects to the server using WebRTC data channels
- The server streams the file line by line with a configurable delay
- The client receives the lines and either displays them or writes them to a file
- Both the server and client use a simple logging system for debugging
- The implementation supports two connection modes:
  1. Direct connection (default):
     - Configures WebRTC to use only local network interfaces
     - Does not use any STUN/TURN servers, ensuring complete privacy
     - All connections are established directly between peers on the local network
     - This approach provides maximum privacy but requires both peers to be on the same network
  2. STUN-assisted connection:
     - Uses a STUN server to help with NAT traversal
     - Allows connections between peers on different networks
     - Can be configured via command-line flags or the configuration file
     - Example: `--stun "stun:stun.l.google.com:19302"`

## Monitoring WebRTC Connection Status

The application logs connection state changes to help you determine if a WebRTC connection has been established. Here's how to interpret the logs:

### Connection States

WebRTC connections go through several states:

1. **New**: Initial state, connection created but no network activity yet
2. **Connecting**: ICE candidates are being exchanged and connectivity is being checked
3. **Connected**: Connection has been established successfully
4. **Disconnected**: Connection has been lost temporarily
5. **Failed**: Connection has failed and cannot be restored
6. **Closed**: Connection has been closed

### How to Check Connection Status

When running the demo, check the log files (`server.log` and `client.log`) for connection state messages:

- When a connection is successfully established, you'll see:
  ```
  [INFO] Connection state changed: connected
  [INFO] WebRTC connection established successfully!
  ```

- If a connection fails, you'll see:
  ```
  [INFO] Connection state changed: failed
  [ERROR] WebRTC connection failed
  ```

- When a data channel is opened (which happens after the connection is established):
  ```
  [INFO] Data channel opened
  ```

A successful connection typically shows these log entries in sequence:
1. Connection state changes (new → connecting → connected)
2. WebRTC connection established message
3. Data channel opened message
4. Data transfer begins (lines being sent/received)

## License

This project is open source and available under the MIT License.
