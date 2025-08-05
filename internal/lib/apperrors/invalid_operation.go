package apperrors

import "fmt"

type InvalidOperation struct {
	Message string
	Err     error
}

// Error implements the error interface for InvalidOperation
func (e InvalidOperation) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s - %v", e.Message, e.Err)
	}
	return e.Message
}

// NewInvalidOperation creates a new InvalidOperation error
func NewInvalidOperation(message string, err error) InvalidOperation {
	return InvalidOperation{
		Message: message,
		Err:     err,
	}
}