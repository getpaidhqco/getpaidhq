package api

import (
	"errors"
	"github.com/go-playground/validator/v10"
	"net/http"
	"payloop/internal/api/dto/response"
	"payloop/internal/lib"
	"payloop/internal/lib/apperrors"
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
	// Check for validation errors
	if errs, ok := err.(validator.ValidationErrors); ok {
		msg := response.FormatValidationErrors(errs)
		return NewApiError(lib.BadRequestError, "Input validation failed", msg)
	}

	// Check for domain apperrors
	var notFound apperrors.NotFound
	var invalidOp apperrors.InvalidOperation
	var invalidArg apperrors.InvalidArgument
	var internalErr apperrors.InternalError
	
	switch {
	case errors.As(err, &notFound):
		return NewApiError(lib.NotFoundError, notFound.Message, nil)
	case errors.As(err, &invalidOp):
		return NewApiError(lib.BadRequestError, invalidOp.Message, nil)
	case errors.As(err, &invalidArg):
		return NewApiError(lib.BadRequestError, invalidArg.Message, nil)
	case errors.As(err, &internalErr):
		return NewApiError(lib.InternalError, internalErr.Message, internalErr.Err.Error())
	}

	// Check for legacy CustomError
	var serr lib.CustomError
	if errors.As(err, &serr) {
		if serr.Err == nil {
			return NewApiError(serr.Type, serr.Message, nil)
		}
		return NewApiError(serr.Type, serr.Message, serr.Err.Error())
	}

	// Default to bad request with the error message
	return NewApiError(lib.BadRequestError, err.Error(), err.Error())
}
