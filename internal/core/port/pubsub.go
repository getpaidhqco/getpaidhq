package port

import (
	"context"
	"errors"
	"fmt"

	"getpaidhq/internal/core/domain"
)

// PubSub defines the interface for publish/subscribe messaging.
type PubSub interface {
	Publish(orgId string, topic string, message any) error
	Subscribe(topic string, handler func(topic string, data []byte)) (PubSubSubscription, error)
}

// PubSubPayload is an alias for the domain PubSubPayload type.
type PubSubPayload = domain.PubSubPayload

type PubSubSubscription interface {
	Unsubscribe() error
}

// QueueClient defines the interface for message queue operations.
type QueueClient interface {
	Start(handler QueueMessageHandler)
	SendMessage(ctx context.Context, data QueueMessage) error
}

type QueueMessage struct {
	Data any              `json:"data"`
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
	if nonRetryableErr, ok := errors.AsType[*QueueHandlerError](err); ok {
		return nonRetryableErr.Retryable
	}
	return false
}
