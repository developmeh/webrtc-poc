.PHONY: build test clean all run

all: build

build:
	@echo "Building WebRTC proof of concept..."
	@mkdir -p bin
	@go build -o bin/server cmd/server/main.go
	@go build -o bin/client cmd/client/main.go
	@chmod +x run_demo.sh
	@echo "Build complete. Run './run_demo.sh' to start the demo."

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
