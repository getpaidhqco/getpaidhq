package mocks

import (
	"github.com/stretchr/testify/mock"
	"payloop/internal/application/lib/events"
)

// MockPubSub is a mock implementation of the PubSub interface
type MockPubSub struct {
	mock.Mock
}

// Publish mocks the Publish method
func (m *MockPubSub) Publish(orgId string, topic string, message interface{}) error {
	args := m.Called(orgId, topic, message)
	return args.Error(0)
}

// Subscribe mocks the Subscribe method
func (m *MockPubSub) Subscribe(topic string, handler func(topic string, data []byte)) (events.Subscription, error) {
	args := m.Called(topic, handler)
	return args.Get(0).(events.Subscription), args.Error(1)
}

// MockSubscription is a mock implementation of the Subscription interface
type MockSubscription struct {
	mock.Mock
}

// Unsubscribe mocks the Unsubscribe method
func (m *MockSubscription) Unsubscribe() error {
	args := m.Called()
	return args.Error(0)
}

// NewMockPubSub creates a new mock pubsub
func NewMockPubSub() *MockPubSub {
	return &MockPubSub{}
}

// NewMockSubscription creates a new mock subscription
func NewMockSubscription() *MockSubscription {
	return &MockSubscription{}
}