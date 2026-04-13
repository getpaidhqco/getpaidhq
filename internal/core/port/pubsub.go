package port

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// PubSub defines the interface for publish/subscribe messaging.
type PubSub interface {
	Publish(orgId string, topic string, message interface{}) error
	Subscribe(topic string, handler func(topic string, data []byte)) (PubSubSubscription, error)
}

type PubSubPayload struct {
	Id        string      `json:"id"`
	OrgId     string      `json:"org_id"`
	Topic     string      `json:"topic"`
	Data      interface{} `json:"data"`
	CreatedAt time.Time   `json:"created_at"`
}

type PubSubSubscription interface {
	Unsubscribe() error
}

// QueueClient defines the interface for message queue operations.
type QueueClient interface {
	Start(handler QueueMessageHandler)
	SendMessage(ctx context.Context, data QueueMessage) error
}

type QueueMessage struct {
	Data interface{}      `json:"data"`
	Type QueueMessageType `json:"type"`
}

type QueueMessageType string

const (
	QueueIncomingWebhook     QueueMessageType = "incoming_webhook"
	QueueReportingDataChange QueueMessageType = "reporting_data_change"
)

type QueueMessageHandler func(msg QueueMessage) error

// QueueHandlerError is a custom error type that includes a retryable flag and wraps another error.
type QueueHandlerError struct {
	Message   string
	Retryable bool
	Err       error
}

// Error implements the error interface for QueueHandlerError.
func (e *QueueHandlerError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the wrapped error.
func (e *QueueHandlerError) Unwrap() error {
	return e.Err
}

// NewQueueHandlerError creates a new QueueHandlerError.
func NewQueueHandlerError(message string, retryable bool, err error) error {
	return &QueueHandlerError{
		Message:   message,
		Retryable: retryable,
		Err:       err,
	}
}

// IsRetryable checks if the error is a QueueHandlerError and if it is retryable.
func IsRetryable(err error) bool {
	var nonRetryableErr *QueueHandlerError
	if errors.As(err, &nonRetryableErr) {
		return nonRetryableErr.Retryable
	}
	return false
}
