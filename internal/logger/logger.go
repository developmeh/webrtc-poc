package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

var (
	infoLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
)

// Init initializes the loggers
func Init() {
	infoLogger = log.New(os.Stdout, "[INFO] ", log.Ldate|log.Ltime)
	errorLogger = log.New(os.Stderr, "[ERROR] ", log.Ldate|log.Ltime)
	debugLogger = log.New(os.Stdout, "[DEBUG] ", log.Ldate|log.Ltime)
}

// Info logs an info message
func Info(format string, v ...interface{}) {
	if infoLogger == nil {
		Init()
	}
	infoLogger.Output(2, fmt.Sprintf(format, v...))
}

// Error logs an error message
func Error(format string, v ...interface{}) {
	if errorLogger == nil {
		Init()
	}
	errorLogger.Output(2, fmt.Sprintf(format, v...))
}

// Debug logs a debug message
func Debug(format string, v ...interface{}) {
	if debugLogger == nil {
		Init()
	}
	debugLogger.Output(2, fmt.Sprintf(format, v...))
}

// Timer returns a function that logs the time elapsed since start
func Timer(name string) func() {
	start := time.Now()
	return func() {
		Info("%s took %v", name, time.Since(start))
	}
}