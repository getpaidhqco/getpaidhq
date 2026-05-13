package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-fuego/fuego"
	"github.com/go-playground/validator/v10"

	"getpaidhq/internal/lib"
)

// ApiError represents an API error response.
type ApiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details"`
}

func (e ApiError) Error() string {
	return e.Message
}

// StatusCode lets Fuego dispatch the right HTTP status when an ApiError
// is returned from a handler. Implements fuego.ErrorWithStatus.
func (e ApiError) StatusCode() int {
	return e.GetHttpErrorCode()
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
	var vErrs validator.ValidationErrors
	if errors.As(err, &vErrs) {
		return NewApiError(lib.BadRequestError, "Input validation failed", FormatValidationErrors(vErrs))
	}

	if serr, ok := errors.AsType[lib.CustomError](err); ok {
		if serr.Err == nil {
			return NewApiError(serr.Type, serr.Message, nil)
		}
		return NewApiError(serr.Type, serr.Message, serr.Err.Error())
	}

	return NewApiError("bad_request", err.Error(), err.Error())
}

// ApiErrorSerializer is wired into fuego.WithErrorSerializer so every error
// returned from a handler — including Fuego's own bind/validation errors —
// renders with the project's ApiError envelope.
func ApiErrorSerializer(w http.ResponseWriter, _ *http.Request, err error) {
	var out ApiError
	switch e := err.(type) {
	case ApiError:
		out = e
	case fuego.HTTPError:
		out = fromFuegoError(e)
	case fuego.BadRequestError:
		out = fromFuegoError(fuego.HTTPError(e))
	case fuego.NotFoundError:
		out = fromFuegoError(fuego.HTTPError(e))
	case fuego.UnauthorizedError:
		out = fromFuegoError(fuego.HTTPError(e))
	case fuego.ForbiddenError:
		out = fromFuegoError(fuego.HTTPError(e))
	case fuego.ConflictError:
		out = fromFuegoError(fuego.HTTPError(e))
	default:
		out = NewApiErrorFromError(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(out.GetHttpErrorCode())
	_ = json.NewEncoder(w).Encode(out)
}

func fromFuegoError(e fuego.HTTPError) ApiError {
	code := lib.BadRequestError
	switch e.StatusCode() {
	case http.StatusNotFound:
		code = lib.NotFoundError
	case http.StatusUnauthorized:
		code = lib.AuthenticationError
	case http.StatusUnprocessableEntity:
		code = lib.ValidationError
	case http.StatusInternalServerError:
		code = lib.InternalError
	}
	msg := e.Title
	if msg == "" {
		msg = e.Error()
	}
	var details any
	if len(e.Errors) > 0 {
		details = e.Errors
	} else if e.Detail != "" {
		details = e.Detail
	}
	return NewApiError(code, msg, details)
}
