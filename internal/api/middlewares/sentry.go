package middlewares

import (
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
	"runtime/debug"
	"time"
)

// SentryMiddleware middleware for Sentry
type SentryMiddleware struct {
	handler lib.RequestHandler
	logger  logger.Logger
	env     lib.Env
}

// NewSentryMiddleware creates new Sentry middleware
func NewSentryMiddleware(handler lib.RequestHandler, logger logger.Logger, env lib.Env) SentryMiddleware {
	return SentryMiddleware{
		handler: handler,
		logger:  logger,
		env:     env,
	}
}

// Setup sets up Sentry middleware
func (m SentryMiddleware) Setup() {
	m.logger.Info("Setting up Sentry middleware")

	m.handler.Gin.Use(func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				m.logger.Error("Recovered from panic",
					"err", r,
					"stack", string(debug.Stack()),
					"url", c.Request.URL.String())
				// Add additional context to Sentry
				sentry.ConfigureScope(func(scope *sentry.Scope) {
					scope.SetTag("endpoint", c.FullPath()) // Add endpoint as a tag
					scope.SetExtra("request_method", c.Request.Method)
					scope.SetExtra("request_url", c.Request.URL.String())
					scope.SetUser(sentry.User{
						IPAddress: c.ClientIP(),
					})
				})

				// Capture the panic in Sentry
				sentry.CurrentHub().Recover(r)
				sentry.Flush(2 * time.Second) // Ensure the event is sent to Sentry

				// Respond with a 500 error
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				c.Abort()
			}
		}()
		c.Next()
	})
}
