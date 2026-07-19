package errors

import (
	"errors"
	"fmt"
)

// CustomErrorType is a stable identifier for an error class. It is exposed to
// API consumers as the `code` field of the JSON error envelope, so values
// MUST be considered part of the public contract — renaming one is a breaking
// change for clients that switch on it.
type CustomErrorType string

const (
	BadRequestError     CustomErrorType = "bad_request"
	NotFoundError       CustomErrorType = "not_found"
	ValidationError     CustomErrorType = "validation_error"
	InternalError       CustomErrorType = "internal_error"
	AuthenticationError CustomErrorType = "auth_error"
	// ForbiddenError is returned when the caller is authenticated but the
	// authorization policy denies the requested action. It maps to HTTP 403,
	// distinguishing authz failures (Cedar denial) from authn failures
	// (missing or invalid token), which is what AuthenticationError covers.
	ForbiddenError CustomErrorType = "forbidden"
	// ConflictError is returned when a request collides with the current
	// resource state (duplicate key, concurrent update, etc). Maps to HTTP 409.
	ConflictError CustomErrorType = "conflict"
	// RateLimitError is returned when the caller has exceeded the allowed
	// request rate. Maps to HTTP 429. Clients should back off and retry,
	// honoring the Retry-After header when present.
	RateLimitError CustomErrorType = "rate_limit_exceeded"
)

// Common sentinel errors. Callers wrap these with fmt.Errorf("...: %w", ...)
// and inspect with errors.Is, so adding context never breaks identity checks.
var (
	// ErrNotFound is the canonical "resource missing" error. Repository
	// implementations return this (or wrap it) so service-layer code can
	// branch on errors.Is(err, lib.ErrNotFound) without importing GORM or
	// any specific storage driver.
	ErrNotFound = errors.New("not found")
)

// CustomError carries a typed error code, a human-readable message safe for
// API responses, and an optional wrapped underlying error. The wrapped error
// is exposed via Unwrap, so errors.Is and errors.As walk through it.
type CustomError struct {
	Type    CustomErrorType
	Message string
	Err     error
}

// Error renders the error for logs. It includes the underlying error if one
// is present so a log line tells the full chain even when the caller did not
// log the wrapped error itself.
func (e CustomError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap exposes the wrapped error so errors.Is and errors.As work through a
// CustomError just as they would through fmt.Errorf("...: %w", ...).
func (e CustomError) Unwrap() error {
	return e.Err
}

// NewCustomError builds a CustomError. Pass err==nil when there is no
// underlying cause (e.g. a validation rule rejected the input outright).
func NewCustomError(errorType CustomErrorType, message string, err error) CustomError {
	return CustomError{
		Type:    errorType,
		Message: message,
		Err:     err,
	}
}
