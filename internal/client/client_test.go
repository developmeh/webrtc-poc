package client

import (
	"errors"
	"io"
	"os"
	"testing"
	"time"
)

// MockLineReceiver is a mock implementation of the LineReceiver interface for testing
type MockLineReceiver struct {
	Lines []string
	Err   error
	Delay time.Duration
}

// ReceiveLines implements the LineReceiver interface
func (m *MockLineReceiver) ReceiveLines() (<-chan string, <-chan error) {
	lineChan := make(chan string)
	errChan := make(chan error, 1)

	go func() {
		defer close(lineChan)
		defer close(errChan)

		// Send each line with the specified delay
		for _, line := range m.Lines {
			if m.Delay > 0 {
				time.Sleep(m.Delay)
			}
			lineChan <- line
		}

		// If there's an error, send it after sending all lines
		if m.Err != nil {
			errChan <- m.Err
		}
	}()

	return lineChan, errChan
}

func TestProcessLines(t *testing.T) {
	// Test with lines sent to stdout
	t.Run("Output to stdout", func(t *testing.T) {
		testLines := []string{"Line 1", "Line 2", "Line 3"}
		receiver := &MockLineReceiver{Lines: testLines}

		lineCount, elapsed, err := ProcessLines(receiver, "")
		if err != nil {
			t.Errorf("ProcessLines returned error: %v", err)
		}

		if lineCount != len(testLines) {
			t.Errorf("Expected %d lines, got %d", len(testLines), lineCount)
		}

		if elapsed <= 0 {
			t.Errorf("Expected positive elapsed time, got %v", elapsed)
		}
	})

	// Test with lines sent to a file
	t.Run("Output to file", func(t *testing.T) {
		// Create a temporary output file
		tmpFile, err := os.CreateTemp("", "test-output-*.txt")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		tmpFile.Close()
		defer os.Remove(tmpFile.Name())

		testLines := []string{"Line 1", "Line 2", "Line 3"}
		receiver := &MockLineReceiver{Lines: testLines}

		lineCount, elapsed, err := ProcessLines(receiver, tmpFile.Name())
		if err != nil {
			t.Errorf("ProcessLines returned error: %v", err)
		}

		if lineCount != len(testLines) {
			t.Errorf("Expected %d lines, got %d", len(testLines), lineCount)
		}

		if elapsed <= 0 {
			t.Errorf("Expected positive elapsed time, got %v", elapsed)
		}

		// Read the output file and check its contents
		content, err := os.ReadFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("Failed to read output file: %v", err)
		}

		expectedContent := "Line 1\nLine 2\nLine 3\n"
		if string(content) != expectedContent {
			t.Errorf("Expected file content '%s', got '%s'", expectedContent, string(content))
		}
	})

	// Test with an error
	t.Run("Error case", func(t *testing.T) {
		testErr := errors.New("test error")
		receiver := &MockLineReceiver{Err: testErr}

		lineCount, _, err := ProcessLines(receiver, "")
		if err != testErr {
			t.Errorf("Expected error %v, got %v", testErr, err)
		}

		if lineCount != 0 {
			t.Errorf("Expected 0 lines, got %d", lineCount)
		}
	})

	// Test with EOF error (should be handled as normal completion)
	t.Run("EOF error", func(t *testing.T) {
		receiver := &MockLineReceiver{
			Lines: []string{"Line 1", "Line 2"},
			Err:   io.EOF,
		}

		lineCount, _, err := ProcessLines(receiver, "")
		if err != nil {
			t.Errorf("ProcessLines returned error: %v", err)
		}

		if lineCount != 2 {
			t.Errorf("Expected 2 lines, got %d", lineCount)
		}
	})

	// Test with invalid output file
	t.Run("Invalid output file", func(t *testing.T) {
		receiver := &MockLineReceiver{Lines: []string{"Line 1"}}

		// Use a directory as the output file, which should fail
		_, _, err := ProcessLines(receiver, "/")
		if err == nil {
			t.Error("Expected error for invalid output file, got nil")
		}
	})

	// Test with delay to verify timing
	t.Run("Respects timing", func(t *testing.T) {
		testLines := []string{"Line 1", "Line 2", "Line 3"}
		delay := 50 * time.Millisecond
		receiver := &MockLineReceiver{Lines: testLines, Delay: delay}

		start := time.Now()
		lineCount, elapsed, err := ProcessLines(receiver, "")
		actualElapsed := time.Since(start)

		if err != nil {
			t.Errorf("ProcessLines returned error: %v", err)
		}

		if lineCount != len(testLines) {
			t.Errorf("Expected %d lines, got %d", len(testLines), lineCount)
		}

		// Check that the elapsed time reported by the function is close to the actual elapsed time
		if elapsed > actualElapsed+10*time.Millisecond || elapsed < actualElapsed-10*time.Millisecond {
			t.Errorf("Reported elapsed time %v differs significantly from actual elapsed time %v", elapsed, actualElapsed)
		}

		// Check that the function took at least the expected time
		// Expected time: (number of lines) * delay
		expectedMinTime := time.Duration(len(testLines)) * delay
		if actualElapsed < expectedMinTime {
			t.Errorf("ProcessLines took %v, expected at least %v", actualElapsed, expectedMinTime)
		}
	})
}
