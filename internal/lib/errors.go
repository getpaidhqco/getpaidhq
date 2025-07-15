package lib

import (
	"errors"
	"fmt"
)

// CustomErrorType defines a custom error type with additional information
type CustomErrorType string

const (
	BadRequestError     CustomErrorType = "bad_request"
	NotFoundError       CustomErrorType = "not_found"
	ValidationError     CustomErrorType = "validation_error"
	InternalError       CustomErrorType = "internal_error"
	AuthenticationError CustomErrorType = "auth_error"
	GatewayError        CustomErrorType = "gateway_error"
)

// CustomError struct includes additional information about the error
type CustomError struct {
	Type    CustomErrorType
	Message string
	Err     error
}

// Error implements the error interface for CustomError
func (e CustomError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s - %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// NewCustomError creates a new CustomError
func NewCustomError(errorType CustomErrorType, message string, err error) CustomError {
	return CustomError{
		Type:    errorType,
		Message: message,
		Err:     err,
	}
}

func MapDatabaseError(err error) CustomError {
	var dbErr DatabaseError
	if errors.As(err, &dbErr) {
		switch dbErr.Code {
		case NoResults:
			return CustomError{
				Type:    NotFoundError,
				Message: "Not found",
				Err:     err,
			}
		case UniqueKeyViolation: // foreign_key_violation
			return CustomError{
				Type:    BadRequestError,
				Message: "Item already exists",
				Err:     err,
			}
		case ForeignKeyViolation: // unique_violation
			return CustomError{
				Type:    BadRequestError,
				Message: "Foreign constraint violation",
				Err:     err,
			}
		}

	}
	return CustomError{
		Type:    InternalError,
		Message: "An internal error occurred",
		Err:     err,
	}
}
