package apperrors

import "fmt"

type InternalError struct {
	Message string
	Err     error
}

// Error implements the error interface for InternalError
func (e InternalError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s - %v", e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Message)
}

func NewInternalError(message string, err error) InternalError {
	return InternalError{
		Message: message,
		Err:     err,
	}
}
