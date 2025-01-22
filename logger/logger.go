package logger

import "log"

// Logger defines the interface for logging used within the httpok framework.
// It provides methods for formatted printing and fatal errors which halt the
// program, ensuring consistent logging behavior across all components of
// httpok.
type Logger interface {
	Printf(format string, v ...interface{})
	Fatalf(format string, v ...interface{})
}

// basicRunLogger implements the RunLogger interface using Go's standard log
// package.
type StandardLogger struct{}

// Printf logs a formatted message using the standard log package's Printf
// method.
func (l *StandardLogger) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

// Fatalf logs a formatted message and then terminates the program using the
// standard log package's Fatalf method.
func (l *StandardLogger) Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}
