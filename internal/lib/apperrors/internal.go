package apperrors

import "fmt"

type InternalError struct {
	Message string
	Err     error
}

// Error implements the error interface for InternalError
func (e InternalError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s - %v", e.Message, e.Err)
	}
	return e.Message
}

func NewInternalError(message string, err error) InternalError {
	return InternalError{
		Message: message,
		Err:     err,
	}
}
