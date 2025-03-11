package events

import (
	"errors"
	"fmt"
)

// QueueHandlerError is a custom error type that includes a non-retryable flag and wraps another error.
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

// IsRetryable checks if the error is a QueueHandlerError and if it is non-retryable.
func IsRetryable(err error) bool {
	var nonRetryableErr *QueueHandlerError
	if errors.As(err, &nonRetryableErr) {
		return nonRetryableErr.Retryable
	}
	return false
}
