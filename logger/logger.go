package logger

import "log"

// Logger defines the interface for logging used within the httpok framework.
// It provides methods for formatted printing and fatal errors which halt the
// program, ensuring consistent logging behavior across all components of
// httpok.
type Logger interface {
	Infof(format string, v ...any)
	Errorf(format string, v ...any)
	Fatalf(format string, v ...any)
	Printf(format string, v ...any)
	Warnf(format string, v ...any)
}

// basicRunLogger implements the RunLogger interface using Go's standard log
// package.
type StandardLogger struct{}

// Infof logs a formatted message using the standard log package's Printf
// method.
func (l *StandardLogger) Infof(format string, v ...any) {
	log.Printf(format, v...)
}

// Errorf logs a formatted message using the standard log package's Printf
// method.
func (l *StandardLogger) Errorf(format string, v ...any) {
	log.Printf(format, v...)
}

// Fatalf logs a formatted message and then terminates the program using the
// standard log package's Fatalf method.
func (l *StandardLogger) Fatalf(format string, v ...any) {
	log.Fatalf(format, v...)
}

// Printf logs a formatted message using the standard log package's Printf
// method.
func (l *StandardLogger) Printf(format string, v ...any) {
	log.Printf(format, v...)
}

// Warnf logs a formatted message using the standard log package's Printf
// method.
func (l *StandardLogger) Warnf(format string, v ...any) {
	log.Printf(format, v...)
}
