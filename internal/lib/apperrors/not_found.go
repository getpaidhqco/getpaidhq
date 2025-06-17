package apperrors

import "fmt"

type NotFound struct {
	Message string
	Err     error
}

// Error implements the error interface for CustomError
func (e NotFound) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s - %v", e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Message)
}

// NewNotFound creates a new NotFound error
func NewNotFound(message string, err error) NotFound {
	return NotFound{
		Message: message,
		Err:     err,
	}
}
