package server

import (
	"bufio"
	"os"
	"time"

	"github.com/paulscoder/webrtc-poc/internal/logger"
)

// StreamFile streams a file line by line to the provided writer
// This is a testable version of the streamFile function from cmd/webrtc-poc/main.go
func StreamFile(writer LineWriter, filename string, delayMs int) error {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Recovered from panic in StreamFile: %v", r)
		}
	}()

	file, err := os.Open(filename)
	if err != nil {
		logger.Error("Failed to open file: %v", err)
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		// Send the line over the writer
		if err := writer.SendText(line); err != nil {
			logger.Error("Failed to send line %d: %v", lineCount, err)
			return err
		}

		logger.Debug("Sent line %d: %s", lineCount, line)

		// Delay between lines
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	if err := scanner.Err(); err != nil {
		logger.Error("Error reading file: %v", err)
		return err
	}

	logger.Info("Finished streaming file, sent %d lines", lineCount)
	return nil
}

// LineWriter is an interface for writing lines of text
// This allows us to test the StreamFile function without using WebRTC
type LineWriter interface {
	SendText(text string) error
}