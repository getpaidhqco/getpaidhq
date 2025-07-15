package nats

import (
	"github.com/stretchr/testify/assert"
	"payloop/internal/lib"
	"testing"
)

func TestNatsNotificationPublisher_Publish(t *testing.T) {
	logger := lib.GetLogger()
	publisher := NewNatsNotificationPublisher(logger)

	// Simple test message
	type TestMessage struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	}

	testMsg := TestMessage{
		ID:      "test-id",
		Message: "test message",
	}

	err := publisher.Publish("test-org", "test.topic", testMsg)
	assert.NoError(t, err)
}