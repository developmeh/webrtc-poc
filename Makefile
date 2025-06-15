.PHONY: build test clean all run

all: build

build:
	@echo "Building WebRTC proof of concept..."
	@mkdir -p bin
	@go build -o bin/webrtc-poc cmd/webrtc-poc/main.go
	@chmod +x run_demo.sh
	@echo "Build complete. Run 'bin/webrtc-poc --help' to see available commands."

test:
	@echo "Running tests..."
	@go test -v ./...

clean:
	@echo "Cleaning up..."
	@rm -rf bin
	@rm -f *.log
	@rm -f webrtc_demo.pid
	@echo "Clean complete."

run: build
	@echo "Running WebRTC demo..."
	@./run_demo.sh

test-connection: build
	@echo "Building and running WebRTC connection test..."
	@go build -o bin/test cmd/test/main.go
	@./bin/test
