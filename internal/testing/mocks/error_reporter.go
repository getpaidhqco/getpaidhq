package mocks

import (
	"github.com/stretchr/testify/mock"
	"payloop/internal/application/lib/logger"
)

// MockErrorReporter is a mock implementation of an error reporter for testing
type MockErrorReporter struct {
	mock.Mock
	Logger logger.Logger
}

// NewMockErrorReporter creates a new mock error reporter
func NewMockErrorReporter(logger logger.Logger) *MockErrorReporter {
	return &MockErrorReporter{
		Logger: logger,
	}
}

// ReportError mocks the ReportError method
func (m *MockErrorReporter) ReportError(ctx interface{}, err error, data map[string]interface{}) {
	m.Called(ctx, err, data)
	// Log the error but don't send to Sentry
	m.Logger.Error("Mock error reporter: ", err.Error())
}
