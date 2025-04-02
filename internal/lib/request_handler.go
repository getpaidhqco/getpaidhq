package lib

import (
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/application/lib/logger"
)

// RequestHandler function
type RequestHandler struct {
	Gin *gin.Engine
}

// NewRequestHandler creates a new request handler
func NewRequestHandler(logger logger.Logger, reporter ErrorReporter) RequestHandler {
	engine := gin.Default()
	engine.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    "not_found",
			"message": "Route not found",
		})
	})
	engine.Use(sentrygin.New(sentrygin.Options{
		Repanic: true,
	}))
	return RequestHandler{Gin: engine}
}
