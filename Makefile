.PHONY: build test clean all run lint release snapshot

all: lint test build

build:
	@echo "Building WebRTC proof of concept..."
	@mkdir -p bin
	@go build -o bin/webrtc-poc cmd/webrtc-poc/main.go
	@chmod +x run_demo.sh
	@echo "Build complete. Run 'bin/webrtc-poc --help' to see available commands."

test: unit-test integration-test

unit-test:
	@echo "Running unit tests..."
	@go test -v ./internal/logger ./internal/server ./internal/client ./internal/config

integration-test:
	@echo "Running integration tests..."
	@go test -v ./internal/integration

test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

lint:
	@echo "Running linters..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, skipping lint"; \
		echo "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

clean:
	@echo "Cleaning up..."
	@rm -rf bin
	@rm -rf dist
	@rm -f *.log
	@rm -f webrtc_demo.pid
	@echo "Clean complete."

release:
	@echo "Creating a release with GoReleaser..."
	@if command -v goreleaser > /dev/null; then \
		goreleaser release --clean; \
	else \
		echo "goreleaser not found, skipping release"; \
		echo "Install with: go install github.com/goreleaser/goreleaser@latest"; \
	fi

snapshot:
	@echo "Creating a snapshot release with GoReleaser..."
	@if command -v goreleaser > /dev/null; then \
		goreleaser release --snapshot --clean; \
	else \
		echo "goreleaser not found, skipping snapshot"; \
		echo "Install with: go install github.com/goreleaser/goreleaser@latest"; \
	fi

run: build
	@echo "Running WebRTC demo..."
	@./run_demo.sh

test-connection: build
	@echo "Building and running WebRTC connection test..."
	@go build -o bin/test cmd/test/main.go
	@./bin/test
