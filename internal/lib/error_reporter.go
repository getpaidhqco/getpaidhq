package lib

import (
	"context"
	"github.com/getsentry/sentry-go"
	"log/slog"
	"payloop/internal/application/lib/logger"
)

// ErrorReporter is a utility struct for reporting errors to Sentry.
type ErrorReporter struct {
	logger logger.Logger
}

// NewErrorReporter creates a new instance of ErrorReporter.
func NewErrorReporter(logger logger.Logger) *ErrorReporter {
	err := sentry.Init(sentry.ClientOptions{
		Dsn: "https://e7437ace0055b5fb41c205559cea6dbd@o529990.ingest.us.sentry.io/4509072453664772", // Replace with your Sentry DSN
	})
	if err != nil {
		logger.Fatal("sentry.Init: ", slog.String("err", err.Error()))
	}

	return &ErrorReporter{
		logger: logger,
	}
}

// ReportError reports an error to Sentry.
func (er *ErrorReporter) ReportError(ctx context.Context, err error) {
	sentry.WithScope(func(scope *sentry.Scope) {
		if hub := sentry.GetHubFromContext(ctx); hub != nil {
			hub.CaptureException(err)
		} else {
			sentry.CaptureException(err)
		}
	})
	er.logger.Error("Error reported to Sentry", slog.String("err", err.Error()))
}
