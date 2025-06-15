package server

import (
	"os"
	"testing"
	"time"
)

// MockLineWriter is a mock implementation of the LineWriter interface for testing
type MockLineWriter struct {
	Lines []string
	Err   error
}

// SendText implements the LineWriter interface
func (m *MockLineWriter) SendText(text string) error {
	if m.Err != nil {
		return m.Err
	}
	m.Lines = append(m.Lines, text)
	return nil
}

func TestStreamFile(t *testing.T) {
	// Create a temporary test file
	tmpFile, err := os.CreateTemp("", "test-stream-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test content to the file
	testContent := []string{
		"Line 1",
		"Line 2",
		"Line 3",
		"This is a longer line with some special characters: !@#$%^&*()",
	}
	for _, line := range testContent {
		tmpFile.WriteString(line + "\n")
	}
	tmpFile.Close()

	// Test with a working writer
	t.Run("Success case", func(t *testing.T) {
		writer := &MockLineWriter{}
		err := StreamFile(writer, tmpFile.Name(), 1) // Use minimal delay for tests
		if err != nil {
			t.Errorf("StreamFile returned error: %v", err)
		}

		// Check that all lines were sent
		if len(writer.Lines) != len(testContent) {
			t.Errorf("Expected %d lines, got %d", len(testContent), len(writer.Lines))
		}

		// Check content of lines
		for i, line := range testContent {
			if i < len(writer.Lines) && writer.Lines[i] != line {
				t.Errorf("Line %d: expected '%s', got '%s'", i+1, line, writer.Lines[i])
			}
		}
	})

	// Test with a failing writer
	t.Run("Writer error", func(t *testing.T) {
		writer := &MockLineWriter{Err: os.ErrInvalid}
		err := StreamFile(writer, tmpFile.Name(), 1)
		if err == nil {
			t.Error("StreamFile should have returned an error")
		}
	})

	// Test with a non-existent file
	t.Run("File not found", func(t *testing.T) {
		writer := &MockLineWriter{}
		err := StreamFile(writer, "non-existent-file.txt", 1)
		if err == nil {
			t.Error("StreamFile should have returned an error for non-existent file")
		}
	})

	// Test with a delay
	t.Run("Respects delay", func(t *testing.T) {
		writer := &MockLineWriter{}
		delayMs := 50
		start := time.Now()
		err := StreamFile(writer, tmpFile.Name(), delayMs)
		elapsed := time.Since(start)
		if err != nil {
			t.Errorf("StreamFile returned error: %v", err)
		}

		// Check that the function took at least the expected time
		// Expected time: (number of lines - 1) * delay
		// We subtract 1 because there's no delay after the last line
		expectedMinTime := time.Duration(len(testContent)-1) * time.Duration(delayMs) * time.Millisecond
		if elapsed < expectedMinTime {
			t.Errorf("StreamFile took %v, expected at least %v", elapsed, expectedMinTime)
		}
	})
}