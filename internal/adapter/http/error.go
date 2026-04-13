package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"payloop/internal/core/port"
	"payloop/internal/lib"
)

// getAuthUser safely extracts the authenticated user from the gin context.
// It returns an error if the user is not set or has an unexpected type.
func getAuthUser(c *gin.Context) (port.AuthUser, error) {
	user, exists := c.Get("user")
	if !exists {
		return port.AuthUser{}, errors.New("authentication required")
	}
	authUser, ok := user.(port.AuthUser)
	if !ok {
		return port.AuthUser{}, errors.New("invalid authentication context")
	}
	return authUser, nil
}

// ApiError represents an API error response.
// swagger:response apiError
type ApiError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details"`
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
func NewApiError(code lib.CustomErrorType, message string, details interface{}) ApiError {
	return ApiError{
		Code:    string(code),
		Message: message,
		Details: details,
	}
}

// NewApiErrorFromError creates an ApiError from a generic error.
func NewApiErrorFromError(err error) ApiError {
	var serr lib.CustomError
	if errs, ok := err.(validator.ValidationErrors); ok {
		msg := FormatValidationErrors(errs)
		return NewApiError(lib.BadRequestError, "Input validation failed", msg)
	}

	if errors.As(err, &serr) {
		if serr.Err == nil {
			return NewApiError(serr.Type, serr.Message, nil)
		}
		return NewApiError(serr.Type, serr.Message, serr.Err.Error())
	}

	return NewApiError("bad_request", err.Error(), err.Error())
}
