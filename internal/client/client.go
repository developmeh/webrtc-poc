package client

import (
	"io"
	"os"
	"time"

	"github.com/paulscoder/webrtc-poc/internal/logger"
)

// LineReceiver is an interface for receiving lines of text
// This allows us to test the client functionality without using WebRTC
type LineReceiver interface {
	ReceiveLines() (<-chan string, <-chan error)
}

// ProcessLines processes lines received from a LineReceiver
// This is a testable version of the client functionality from cmd/webrtc-poc/main.go
func ProcessLines(receiver LineReceiver, output string) (int, time.Duration, error) {
	// Open the output file if specified
	var outputFile *os.File
	var err error
	if output != "" {
		outputFile, err = os.Create(output)
		if err != nil {
			logger.Error("Failed to create output file: %v", err)
			return 0, 0, err
		}
		defer outputFile.Close()
		logger.Info("Writing output to file: %s", output)
	} else {
		logger.Info("Writing output to stdout")
	}

	// Get the line and error channels from the receiver
	lineChan, errChan := receiver.ReceiveLines()

	// Process lines
	lineCount := 0
	startTime := time.Now()

	for {
		select {
		case line, ok := <-lineChan:
			if !ok {
				// Channel closed, we're done
				elapsed := time.Since(startTime)
				logger.Info("Received %d lines in %v (%.2f lines/sec)",
					lineCount, elapsed, float64(lineCount)/elapsed.Seconds())
				return lineCount, elapsed, nil
			}

			lineCount++

			// Write to output
			if outputFile != nil {
				if _, err := outputFile.WriteString(line + "\n"); err != nil {
					logger.Error("Failed to write to output file: %v", err)
					return lineCount, time.Since(startTime), err
				}
			} else {
				os.Stdout.WriteString(line + "\n")
			}

			logger.Debug("Received line %d: %s", lineCount, line)

		case err, ok := <-errChan:
			if !ok {
				// Error channel closed, but no error
				continue
			}
			if err == io.EOF {
				// EOF is expected when the stream ends
				elapsed := time.Since(startTime)
				logger.Info("Received %d lines in %v (%.2f lines/sec)",
					lineCount, elapsed, float64(lineCount)/elapsed.Seconds())
				return lineCount, elapsed, nil
			}
			// Any other error is unexpected
			logger.Error("Error receiving line: %v", err)
			return lineCount, time.Since(startTime), err
		}
	}
}