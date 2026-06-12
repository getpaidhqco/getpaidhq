package commands

import (
	"encoding/json"
	"errors"
	"fmt"

	"getpaidhq/internal/cli/client"
)

// UsageError marks errors caused by bad invocation (exit code 2).
type UsageError struct{ msg string }

func (e *UsageError) Error() string { return e.msg }

func Usagef(format string, a ...any) error {
	return &UsageError{msg: fmt.Sprintf(format, a...)}
}

// FormatError renders any command error for stderr, unwrapping the API
// error envelope when present.
func FormatError(err error) string {
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		s := fmt.Sprintf("error (%s): %s", apiErr.Code, apiErr.Message)
		if apiErr.Details != nil {
			if d, jerr := json.Marshal(apiErr.Details); jerr == nil {
				s += "\n  details: " + string(d)
			}
		}
		return s
	}
	return "error: " + err.Error()
}
