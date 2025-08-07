package mocks

import (
	"github.com/stretchr/testify/mock"
	"payloop/internal/application/lib/logger"
)

// MockLogger provides a reusable mock implementation of the logger interface
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Fatal(msg string, args ...any) {
	m.Called(append([]interface{}{msg}, args...)...)
}

func (m *MockLogger) Debugf(template string, args ...interface{}) {
	m.Called(append([]interface{}{template}, args...)...)
}

func (m *MockLogger) Infof(template string, args ...interface{}) {
	m.Called(append([]interface{}{template}, args...)...)
}

func (m *MockLogger) Warnf(template string, args ...interface{}) {
	m.Called(append([]interface{}{template}, args...)...)
}

func (m *MockLogger) Errorf(template string, args ...interface{}) {
	m.Called(append([]interface{}{template}, args...)...)
}

func (m *MockLogger) Panicf(template string, args ...interface{}) {
	m.Called(append([]interface{}{template}, args...)...)
}

func (m *MockLogger) Fatalf(template string, args ...interface{}) {
	m.Called(append([]interface{}{template}, args...)...)
}

func (m *MockLogger) Sync() error {
	args := m.Called()
	return args.Error(0)
}

var _ logger.Logger = (*MockLogger)(nil)

func (m *MockLogger) Info(msg string, args ...interface{}) {
	m.Called(append([]interface{}{msg}, args...)...)
}

func (m *MockLogger) Error(msg string, args ...interface{}) {
	m.Called(append([]interface{}{msg}, args...)...)
}

func (m *MockLogger) Debug(msg string, args ...interface{}) {
	m.Called(append([]interface{}{msg}, args...)...)
}

func (m *MockLogger) Warn(msg string, args ...interface{}) {
	m.Called(append([]interface{}{msg}, args...)...)
}

func (m *MockLogger) With(args ...interface{}) logger.Logger {
	returnArgs := m.Called(args...)
	return returnArgs.Get(0).(logger.Logger)
}

// NewMockLogger creates a new mock logger with common setup
func NewMockLogger() *MockLogger {
	mockLogger := &MockLogger{}

	// Set up common expectations that most tests need
	mockLogger.On("Info", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Error", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Debug", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Warn", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Fatal", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Infof", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Errorf", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Debugf", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Warnf", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Fatalf", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Panicf", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Sync").Return(nil).Maybe()
	mockLogger.On("With", mock.Anything).Return(mockLogger).Maybe()

	return mockLogger
}

// NewSilentLogger creates a mock logger that ignores all calls
func NewSilentLogger() *MockLogger {
	mockLogger := &MockLogger{}

	// Set up expectations to ignore all calls
	mockLogger.On("Info", mock.Anything, mock.Anything).Maybe().Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Maybe().Return()
	mockLogger.On("Debug", mock.Anything, mock.Anything).Maybe().Return()
	mockLogger.On("Warn", mock.Anything, mock.Anything).Maybe().Return()
	mockLogger.On("Fatal", mock.Anything, mock.Anything).Maybe().Return()
	mockLogger.On("Infof", mock.Anything, mock.Anything).Maybe().Return()
	mockLogger.On("Errorf", mock.Anything, mock.Anything).Maybe().Return()
	mockLogger.On("Debugf", mock.Anything, mock.Anything).Maybe().Return()
	mockLogger.On("Warnf", mock.Anything, mock.Anything).Maybe().Return()
	mockLogger.On("Fatalf", mock.Anything, mock.Anything).Maybe().Return()
	mockLogger.On("Panicf", mock.Anything, mock.Anything).Maybe().Return()
	mockLogger.On("Sync").Return(nil).Maybe()
	mockLogger.On("With", mock.Anything).Return(mockLogger).Maybe()

	return mockLogger
}

// TestLogger provides a simple test logger that doesn't require mock setup
type TestLogger struct{}

func NewTestLogger() *TestLogger {
	return &TestLogger{}
}

func (l *TestLogger) Debug(msg string, args ...any) {}
func (l *TestLogger) Info(msg string, args ...any)  {}
func (l *TestLogger) Warn(msg string, args ...any)  {}
func (l *TestLogger) Error(msg string, args ...any) {}
func (l *TestLogger) Fatal(msg string, args ...any) {}

func (l *TestLogger) Debugf(template string, args ...interface{}) {}
func (l *TestLogger) Infof(template string, args ...interface{})  {}
func (l *TestLogger) Warnf(template string, args ...interface{})  {}
func (l *TestLogger) Errorf(template string, args ...interface{}) {}
func (l *TestLogger) Panicf(template string, args ...interface{}) {}
func (l *TestLogger) Fatalf(template string, args ...interface{}) {}

func (l *TestLogger) Sync() error { return nil }
