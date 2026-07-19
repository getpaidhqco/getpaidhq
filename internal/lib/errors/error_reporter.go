package errors

import (
	"getpaidhq/internal/lib"
	"log/slog"
)

// ErrorReporter is a utility struct for reporting errors. ReportError
// currently logs the error and is intended to be wired to a reporting
// backend later.
type ErrorReporter struct {
	logger lib.Logger
}

// NewErrorReporter creates a new instance of ErrorReporter.
func NewErrorReporter(logger lib.Logger) ErrorReporter {
	return ErrorReporter{
		logger: logger,
	}
}

func (er *ErrorReporter) ReportError(_ any, err error, data map[string]any) {
	er.logger.Error("ReportError", slog.String("err", err.Error()), slog.Any("data", data))
}
