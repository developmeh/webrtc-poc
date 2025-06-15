# Testing Documentation for WebRTC-POC

This document provides information about the tests implemented for the WebRTC-POC project and how to run them.

## Test Structure

The project includes both unit tests and integration tests:

### Unit Tests

Unit tests are located in the following packages:

1. **Logger Tests** (`internal/logger/logger_test.go`):
   - Tests the initialization of loggers
   - Tests logging functions (Info, Error, Debug)
   - Tests the timer functionality

2. **Server Tests** (`internal/server/server_test.go`):
   - Tests the StreamFile function that streams a file line by line
   - Tests handling of various error conditions
   - Tests respecting the delay between lines

3. **Client Tests** (`internal/client/client_test.go`):
   - Tests the ProcessLines function that processes lines received from a LineReceiver
   - Tests output to stdout and to a file
   - Tests handling of various error conditions
   - Tests timing and performance metrics

4. **Configuration Tests** (`internal/config/config_test.go`):
   - Tests loading configuration from a file
   - Tests loading default configuration
   - Tests saving configuration to a file
   - Tests handling of invalid configuration

### Integration Tests

Integration tests are located in the `internal/integration` package:

1. **End-to-End File Transfer Test** (`internal/integration/integration_test.go`):
   - Tests the end-to-end file transfer functionality
   - Creates a server and client in the same process
   - Transfers a file from server to client
   - Verifies that the file is transferred correctly

## Running Tests

You can run the tests using the following make targets:

### Running All Tests

```bash
make test
```

This will run both unit tests and integration tests.

### Running Unit Tests Only

```bash
make unit-test
```

This will run only the unit tests.

### Running Integration Tests Only

```bash
make integration-test
```

This will run only the integration tests.

### Running Tests with Coverage

```bash
make test-coverage
```

This will run all tests with coverage and generate a coverage report at `coverage.html`.

## Test Requirements

- Go 1.16 or later
- The tests are designed to be self-contained and do not require any external services
- Integration tests use local WebRTC connections without STUN/TURN servers

## Adding New Tests

When adding new functionality to the project, please also add corresponding tests:

1. For new packages, create a `<package>_test.go` file in the same directory
2. Use table-driven tests where appropriate
3. Mock external dependencies to ensure tests are isolated and repeatable
4. For integration tests, consider adding them to the existing integration test file or create a new one in the `internal/integration` package

## Continuous Integration

The tests are automatically run as part of the CI/CD pipeline. The pipeline will fail if any tests fail.