package cli

import (
	"encoding/json"
	"errors"
	"fmt"
)

// APIError is the CLI's view of the server's {code,message,details} error
// envelope, extracted from a non-OK generated response variant.
type APIError struct {
	Code    string
	Message string
	Details any
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return e.Code + ": " + e.Message
	}
	return e.Message
}

// UsageError marks errors caused by bad invocation (exit code 2).
type UsageError struct{ msg string }

func (e *UsageError) Error() string { return e.msg }

// Usagef returns a UsageError (exit code 2) formatted like fmt.Sprintf. Use
// for invalid flags, missing required arguments, or other invocation mistakes
// the user can fix by re-running with correct input.
func Usagef(format string, a ...any) error {
	return &UsageError{msg: fmt.Sprintf(format, a...)}
}

// expectOK returns the OK variant T of a generated response sum type, or
// converts a non-OK (ApiError-based) variant into an *APIError. err is the
// transport error from the client call and is returned as-is when non-nil.
func expectOK[T any](res any, err error) (T, error) {
	var zero T
	if err != nil {
		return zero, err
	}
	if v, ok := res.(T); ok {
		return v, nil
	}
	return zero, apiErrFromRes(res)
}

// apiErrFromRes turns any non-OK response variant into an error. All error
// variants marshal to the ApiError {code,message,details} body, so we
// re-encode (via the generated MarshalJSON) and read those fields back.
func apiErrFromRes(res any) error {
	if b, mErr := json.Marshal(res); mErr == nil {
		var e struct {
			Code    string          `json:"code"`
			Message string          `json:"message"`
			Details json.RawMessage `json:"details"`
		}
		if json.Unmarshal(b, &e) == nil && (e.Code != "" || e.Message != "") {
			var d any
			if len(e.Details) > 0 {
				_ = json.Unmarshal(e.Details, &d)
			}
			return &APIError{Code: e.Code, Message: e.Message, Details: d}
		}
	}
	return fmt.Errorf("unexpected API response (%T)", res)
}

// FormatError renders any command error for stderr, unwrapping the API error
// envelope when present.
func FormatError(err error) string {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		s := fmt.Sprintf("error (%s): %s", apiErr.Code, apiErr.Message)
		if apiErr.Details != nil {
			if d, jErr := json.Marshal(apiErr.Details); jErr == nil && string(d) != "null" {
				s += "\n  details: " + string(d)
			}
		}
		return s
	}
	return "error: " + err.Error()
}
