package handler

import (
	"context"
	"encoding/json"
	"errors"
	errors2 "getpaidhq/internal/lib/errors"
	"net/http"

	"github.com/go-fuego/fuego"
	"github.com/go-playground/validator/v10"
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
	case string(errors2.BadRequestError):
		return http.StatusBadRequest
	case string(errors2.NotFoundError):
		return http.StatusNotFound
	case string(errors2.ValidationError):
		return http.StatusUnprocessableEntity
	case string(errors2.InternalError):
		return http.StatusInternalServerError
	case string(errors2.AuthenticationError):
		return http.StatusUnauthorized
	case string(errors2.ForbiddenError):
		return http.StatusForbidden
	case string(errors2.ConflictError):
		return http.StatusConflict
	case string(errors2.RateLimitError):
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// NewApiError creates a new API error with the typed code, a human-readable
// message, and optional details (a string, a slice, or any JSON-serializable
// value).
func NewApiError(code errors2.CustomErrorType, message string, details any) ApiError {
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
		return NewApiError(errors2.InternalError, "unknown error", nil)
	}

	var vErrs validator.ValidationErrors
	if errors.As(err, &vErrs) {
		return NewApiError(errors2.BadRequestError, "Input validation failed", FormatValidationErrors(vErrs))
	}

	if serr, ok := errors.AsType[errors2.CustomError](err); ok {
		if serr.Err == nil {
			return NewApiError(serr.Type, serr.Message, nil)
		}
		return NewApiError(serr.Type, serr.Message, serr.Err.Error())
	}

	if errors.Is(err, errors2.ErrNotFound) {
		return NewApiError(errors2.NotFoundError, err.Error(), nil)
	}

	return NewApiError(errors2.BadRequestError, err.Error(), err.Error())
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
	code := errors2.BadRequestError
	switch e.StatusCode() {
	case http.StatusNotFound:
		code = errors2.NotFoundError
	case http.StatusUnauthorized:
		code = errors2.AuthenticationError
	case http.StatusForbidden:
		code = errors2.ForbiddenError
	case http.StatusConflict:
		code = errors2.ConflictError
	case http.StatusUnprocessableEntity:
		code = errors2.ValidationError
	case http.StatusInternalServerError:
		code = errors2.InternalError
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
