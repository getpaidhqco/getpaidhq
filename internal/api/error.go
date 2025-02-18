package api

import (
	"errors"
	"github.com/go-playground/validator/v10"
	"net/http"
	"payloop/internal/api/dto/response"
	"payloop/internal/lib"
)

// swagger:response apiError
type ApiError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details"`
}

func (e ApiError) Error() string {
	return e.Message
}

// GetHttpErrorCode maps CustomErrorType to HTTP status codes
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

// NewApiError creates a new api error
func NewApiError(code lib.CustomErrorType, message string, details interface{}) ApiError {
	return ApiError{
		Code:    string(code),
		Message: message,
		Details: details,
	}
}

func NewApiErrorFromError(err error) ApiError {
	var serr lib.CustomError
	if errs, ok := err.(validator.ValidationErrors); ok {
		msg := response.FormatValidationErrors(errs)
		return NewApiError(lib.BadRequestError, "Input validation failed", msg)
	}

	if errors.As(err, &serr) {
		if serr.Err == nil {
			return NewApiError(serr.Type, serr.Message, nil)
		}
		return NewApiError(serr.Type, serr.Message, serr.Err.Error())
	}

	return NewApiError("internal_error", err.Error(), err.Error())
}
