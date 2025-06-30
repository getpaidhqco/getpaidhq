package mocks

import (
	"payloop/internal/lib/pubsub"

	"github.com/stretchr/testify/mock"
)

// MockPubSub provides a reusable mock for pubsub interface
type MockPubSub struct {
	mock.Mock
}

var _ pubsub.PubSub = (*MockPubSub)(nil)

func (m *MockPubSub) Publish(orgId string, topic string, data interface{}) error {
	args := m.Called(orgId, topic, data)
	return args.Error(0)
}

func (m *MockPubSub) Subscribe(topic string, handler pubsub.MessageHandler) error {
	args := m.Called(topic, handler)
	return args.Error(0)
}

func (m *MockPubSub) Close() error {
	args := m.Called()
	return args.Error(0)
}

// NewMockPubSub creates a new mock pubsub with common setup
func NewMockPubSub() *MockPubSub {
	mockPubSub := &MockPubSub{}
	
	// Set up common expectations that most tests need
	mockPubSub.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	
	return mockPubSub
}

// NewSilentPubSub creates a mock pubsub that ignores all calls
func NewSilentPubSub() *MockPubSub {
	mockPubSub := &MockPubSub{}
	
	// Set up expectations to ignore all calls
	mockPubSub.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	mockPubSub.On("Subscribe", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockPubSub.On("Close").Return(nil).Maybe()
	
	return mockPubSub
}