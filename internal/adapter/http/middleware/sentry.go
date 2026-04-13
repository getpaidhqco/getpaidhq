package middleware

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"

	"payloop/internal/core/port"
	"payloop/internal/lib"
)

// SentryMiddleware captures panics and reports them to Sentry.
type SentryMiddleware struct {
	handler lib.RequestHandler
	logger  port.Logger
	env     lib.Env
}

// NewSentryMiddleware creates a new SentryMiddleware.
func NewSentryMiddleware(handler lib.RequestHandler, logger port.Logger, env lib.Env) SentryMiddleware {
	return SentryMiddleware{
		handler: handler,
		logger:  logger,
		env:     env,
	}
}

// Setup registers the Sentry middleware on the gin engine.
func (m SentryMiddleware) Setup() {
	m.logger.Info("setting up sentry middleware")

	m.handler.Gin.Use(func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				m.logger.Error("recovered from panic",
					"err", r,
					"stack", string(debug.Stack()),
					"url", c.Request.URL.String())
				sentry.ConfigureScope(func(scope *sentry.Scope) {
					scope.SetTag("endpoint", c.FullPath())
					scope.SetExtra("request_method", c.Request.Method)
					scope.SetExtra("request_url", c.Request.URL.String())
					scope.SetUser(sentry.User{
						IPAddress: c.ClientIP(),
					})
				})

				sentry.CurrentHub().Recover(r)
				sentry.Flush(2 * time.Second)

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				c.Abort()
			}
		}()
		c.Next()
	})
}
