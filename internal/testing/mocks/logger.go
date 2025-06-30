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
	//TODO implement me
	panic("implement me")
}

func (m *MockLogger) Debugf(template string, args ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (m *MockLogger) Infof(template string, args ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (m *MockLogger) Warnf(template string, args ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (m *MockLogger) Errorf(template string, args ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (m *MockLogger) Panicf(template string, args ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (m *MockLogger) Fatalf(template string, args ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (m *MockLogger) Sync() error {
	//TODO implement me
	panic("implement me")
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
	mockLogger.On("With", mock.Anything).Return(mockLogger).Maybe()

	return mockLogger
}
