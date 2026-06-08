package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-fuego/fuego"
	"github.com/go-playground/validator/v10"

	"getpaidhq/internal/lib"
)

// PassThroughApiError is the engine ErrorHandler wired into BuildServer. It
// returns our own ApiError untouched so the serializer's `case ApiError`
// branch renders the full {code,message,details} envelope. Any other error is
// delegated to Fuego's default ErrorHandler, which normalizes fuego.* error
// types (populating Status from their ErrorWithStatus) and passes plain errors
// through. Without this, Fuego coerces ApiError (it implements ErrorWithStatus)
// into a bare HTTPError before the serializer, dropping Message and Details.
func PassThroughApiError(ctx context.Context, err error) error {
	var apiErr ApiError
	if errors.As(err, &apiErr) {
		return err
	}
	return fuego.ErrorHandler(ctx, err)
}

// ApiError is the JSON envelope every API error renders into. Code is the
// stable machine-readable identifier (one of the lib.CustomErrorType values);
// Message is human-readable; Details carries field-level errors or the
// underlying message text and may be nil.
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

// GetHttpErrorCode maps the project-level error code to an HTTP status. An
// unknown code is conservatively reported as 500 — surfacing an opaque 500 is
// safer than leaking an arbitrary 4xx that misleads the client about whether
// to retry.
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
	case string(lib.ForbiddenError):
		return http.StatusForbidden
	case string(lib.ConflictError):
		return http.StatusConflict
	case string(lib.RateLimitError):
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// NewApiError creates a new API error with the typed code, a human-readable
// message, and optional details (a string, a slice, or any JSON-serializable
// value).
func NewApiError(code lib.CustomErrorType, message string, details any) ApiError {
	return ApiError{
		Code:    string(code),
		Message: message,
		Details: details,
	}
}

// NewApiErrorFromError translates an arbitrary error into the project envelope.
// The order matters:
//
//  1. validator.ValidationErrors → BadRequestError with formatted field list.
//  2. lib.CustomError anywhere in the chain → preserve the typed code so a
//     wrapped service-layer error still renders correctly.
//  3. lib.ErrNotFound anywhere in the chain → NotFoundError. This lets
//     repositories return a wrapped ErrNotFound without every service having
//     to translate it explicitly.
//  4. fallback → BadRequestError echoing the message. Generic errors should
//     not reach this branch in production code; the catch-all exists so
//     handlers never panic on an unexpected error type.
func NewApiErrorFromError(err error) ApiError {
	if err == nil {
		return NewApiError(lib.InternalError, "unknown error", nil)
	}

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

	if errors.Is(err, lib.ErrNotFound) {
		return NewApiError(lib.NotFoundError, err.Error(), nil)
	}

	return NewApiError(lib.BadRequestError, err.Error(), err.Error())
}

// ApiErrorSerializer is wired into fuego.WithErrorSerializer so every error
// returned from a handler — including Fuego's own bind/validation errors —
// renders with the project's ApiError envelope.
func ApiErrorSerializer(w http.ResponseWriter, _ *http.Request, err error) {
	out := toApiError(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(out.GetHttpErrorCode())
	// The response writer is already past the point of recovery if Encode
	// fails (headers and status are written); a logger call here would be
	// noise. Encode failures here are typically a broken client connection.
	_ = json.NewEncoder(w).Encode(out)
}

// toApiError centralizes the err → ApiError translation. Pulled out of the
// serializer so other places (e.g. middleware writing a 401) can reuse the
// same mapping instead of hand-rolling JSON.
func toApiError(err error) ApiError {
	switch e := err.(type) {
	case ApiError:
		return e
	case fuego.HTTPError:
		return fromFuegoError(e)
	case fuego.BadRequestError:
		return fromFuegoError(fuego.HTTPError(e))
	case fuego.NotFoundError:
		return fromFuegoError(fuego.HTTPError(e))
	case fuego.UnauthorizedError:
		return fromFuegoError(fuego.HTTPError(e))
	case fuego.ForbiddenError:
		return fromFuegoError(fuego.HTTPError(e))
	case fuego.ConflictError:
		return fromFuegoError(fuego.HTTPError(e))
	default:
		return NewApiErrorFromError(err)
	}
}

func fromFuegoError(e fuego.HTTPError) ApiError {
	code := lib.BadRequestError
	switch e.StatusCode() {
	case http.StatusNotFound:
		code = lib.NotFoundError
	case http.StatusUnauthorized:
		code = lib.AuthenticationError
	case http.StatusForbidden:
		code = lib.ForbiddenError
	case http.StatusConflict:
		code = lib.ConflictError
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
