package lib

import (
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"net/http"
)

// RequestHandler function
type RequestHandler struct {
	Gin *gin.Engine
}

// NewRequestHandler creates a new request handler
func NewRequestHandler(logger Logger, reporter ErrorReporter) RequestHandler {
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
	engine.Use(func(c *gin.Context) {
		logger.Debugf("-------- %s %s", c.Request.Method, c.Request.URL.Path)
		c.Next()
	})
	// Register custom validations
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		err := v.RegisterValidation("iso4217", ValidateCurrency)
		if err != nil {
			logger.Errorf("Failed to register custom validator: %v", err)
		}
	}

	return RequestHandler{Gin: engine}
}
