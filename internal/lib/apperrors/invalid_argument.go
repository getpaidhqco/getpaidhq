package apperrors

import "fmt"

type InvalidArgument struct {
	Message string
	Err     error
}

// Error implements the error interface for InvalidArgument
func (e InvalidArgument) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s - %v", e.Message, e.Err)
	}
	return e.Message
}

// NewInvalidArgument creates a new InvalidArgument error
func NewInvalidArgument(message string, err error) InvalidArgument {
	return InvalidArgument{
		Message: message,
		Err:     err,
	}
}