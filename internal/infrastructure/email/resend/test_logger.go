package resend

import (
	"fmt"
)

// TestLogger is a simple logger implementation for testing
type TestLogger struct{}

// NewTestLogger creates a new test logger
func NewTestLogger() *TestLogger {
	return &TestLogger{}
}

// Debug logs a debug message
func (l *TestLogger) Debug(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[DEBUG] %s %v\n", msg, keysAndValues)
}

// Info logs an info message
func (l *TestLogger) Info(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[INFO] %s %v\n", msg, keysAndValues)
}

// Warn logs a warning message
func (l *TestLogger) Warn(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[WARN] %s %v\n", msg, keysAndValues)
}

// Error logs an error message
func (l *TestLogger) Error(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[ERROR] %s %v\n", msg, keysAndValues)
}

// Fatal logs a fatal message
func (l *TestLogger) Fatal(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[FATAL] %s %v\n", msg, keysAndValues)
}

// Debugf logs a formatted debug message
func (l *TestLogger) Debugf(template string, args ...interface{}) {
	fmt.Printf("[DEBUG] %s\n", fmt.Sprintf(template, args...))
}

// Infof logs a formatted info message
func (l *TestLogger) Infof(template string, args ...interface{}) {
	fmt.Printf("[INFO] %s\n", fmt.Sprintf(template, args...))
}

// Warnf logs a formatted warning message
func (l *TestLogger) Warnf(template string, args ...interface{}) {
	fmt.Printf("[WARN] %s\n", fmt.Sprintf(template, args...))
}

// Errorf logs a formatted error message
func (l *TestLogger) Errorf(template string, args ...interface{}) {
	fmt.Printf("[ERROR] %s\n", fmt.Sprintf(template, args...))
}

// Panicf logs a formatted panic message
func (l *TestLogger) Panicf(template string, args ...interface{}) {
	fmt.Printf("[PANIC] %s\n", fmt.Sprintf(template, args...))
	panic(fmt.Sprintf(template, args...))
}

// Fatalf logs a formatted fatal message
func (l *TestLogger) Fatalf(template string, args ...interface{}) {
	fmt.Printf("[FATAL] %s\n", fmt.Sprintf(template, args...))
}

// Sync syncs the logger (no-op for test logger)
func (l *TestLogger) Sync() error {
	return nil
}