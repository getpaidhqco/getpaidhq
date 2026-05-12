package handler

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"

	"getpaidhq/internal/lib"
)

// ApiError represents an API error response.
// swagger:response apiError
type ApiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details"`
}

func (e ApiError) Error() string {
	return e.Message
}

// GetHttpErrorCode maps CustomErrorType to HTTP status codes.
func (e ApiError) GetHttpErrorCode() int {
	switch e.Code {
	case string(lib.BadRequestError):
		return http.StatusBadRequest
	case string(lib.NotFoundError):
		return http.StatusNotFound
	case string(lib.ValidationError):
		return http.StatusUnprocessableEntity
	case string(lib.InternalError):
		return http.StatusInternalServerError
	case string(lib.AuthenticationError):
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// NewApiError creates a new API error.
func NewApiError(code lib.CustomErrorType, message string, details any) ApiError {
	return ApiError{
		Code:    string(code),
		Message: message,
		Details: details,
	}
}

// NewApiErrorFromError creates an ApiError from a generic error.
func NewApiErrorFromError(err error) ApiError {
	if errs, ok := err.(validator.ValidationErrors); ok {
		msg := FormatValidationErrors(errs)
		return NewApiError(lib.BadRequestError, "Input validation failed", msg)
	}

	if serr, ok := errors.AsType[lib.CustomError](err); ok {
		if serr.Err == nil {
			return NewApiError(serr.Type, serr.Message, nil)
		}
		return NewApiError(serr.Type, serr.Message, serr.Err.Error())
	}

	return NewApiError("bad_request", err.Error(), err.Error())
}
