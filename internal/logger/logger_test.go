package logger

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

func TestInit(t *testing.T) {
	// Reset loggers before test
	infoLogger = nil
	errorLogger = nil
	debugLogger = nil

	// Call Init
	Init()

	// Check that loggers are initialized
	if infoLogger == nil {
		t.Error("infoLogger not initialized")
	}
	if errorLogger == nil {
		t.Error("errorLogger not initialized")
	}
	if debugLogger == nil {
		t.Error("debugLogger not initialized")
	}
}

func TestInfo(t *testing.T) {
	// Redirect stdout to capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset logger
	infoLogger = nil

	// Call Info
	Info("test message %d", 123)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check output
	if !strings.Contains(output, "[INFO]") {
		t.Errorf("Expected output to contain [INFO], got %s", output)
	}
	if !strings.Contains(output, "test message 123") {
		t.Errorf("Expected output to contain 'test message 123', got %s", output)
	}
}

func TestError(t *testing.T) {
	// Redirect stderr to capture output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Reset logger
	errorLogger = nil

	// Call Error
	Error("error message %d", 456)

	// Restore stderr
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check output
	if !strings.Contains(output, "[ERROR]") {
		t.Errorf("Expected output to contain [ERROR], got %s", output)
	}
	if !strings.Contains(output, "error message 456") {
		t.Errorf("Expected output to contain 'error message 456', got %s", output)
	}
}

func TestDebug(t *testing.T) {
	// Redirect stdout to capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset logger
	debugLogger = nil

	// Call Debug
	Debug("debug message %d", 789)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check output
	if !strings.Contains(output, "[DEBUG]") {
		t.Errorf("Expected output to contain [DEBUG], got %s", output)
	}
	if !strings.Contains(output, "debug message 789") {
		t.Errorf("Expected output to contain 'debug message 789', got %s", output)
	}
}

func TestTimer(t *testing.T) {
	// Redirect stdout to capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset logger
	infoLogger = nil

	// Create and call timer
	timer := Timer("test operation")
	time.Sleep(10 * time.Millisecond) // Sleep to ensure measurable time
	timer()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check output
	if !strings.Contains(output, "[INFO]") {
		t.Errorf("Expected output to contain [INFO], got %s", output)
	}
	if !strings.Contains(output, "test operation took") {
		t.Errorf("Expected output to contain 'test operation took', got %s", output)
	}
}
