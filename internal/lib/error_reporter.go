package lib

import (
	"context"
	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"log/slog"
	"payloop/internal/application/lib/logger"
)

// ErrorReporter is a utility struct for reporting errors to Sentry.
type ErrorReporter struct {
	logger logger.Logger
}

// NewErrorReporter creates a new instance of ErrorReporter.
func NewErrorReporter(logger logger.Logger) ErrorReporter {
	err := sentry.Init(sentry.ClientOptions{
		Dsn: "https://48ff010c02c013dfadac4dd94e9db1d5@o529990.ingest.us.sentry.io/4509072497180673", // Replace with your Sentry DSN
	})
	if err != nil {
		logger.Fatal("sentry.Init: ", slog.String("err", err.Error()))
	}

	return ErrorReporter{
		logger: logger,
	}
}

func (er *ErrorReporter) ReportError(ctx interface{}, err error, data map[string]interface{}) {
	switch c := ctx.(type) {
	case *gin.Context:
		if hub := sentrygin.GetHubFromContext(c); hub != nil {
			hub.WithScope(func(scope *sentry.Scope) {
				scope.SetContext("extra", data)
				hub.CaptureException(err)
			})
		}
	case context.Context:
		er.logger.Error("Unsupported context type", slog.String("err", "Unsupported context type"))
		sentry.WithScope(func(scope *sentry.Scope) {
			scope.SetContext("extra", data)
			if hub := sentry.GetHubFromContext(c); hub != nil {
				hub.CaptureException(err)
			} else {
				sentry.CaptureException(err)
			}
		})

	default:
		er.logger.Error("Unsupported context type", slog.String("err", "Unsupported context type"))
	}
	er.logger.Debug("Error reported to Sentry", slog.String("err", err.Error()))
}
