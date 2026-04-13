package middleware

import (
	cors "github.com/rs/cors/wrapper/gin"

	"payloop/internal/core/port"
	"payloop/internal/lib"
)

// CorsMiddleware handles CORS configuration.
type CorsMiddleware struct {
	handler lib.RequestHandler
	logger  port.Logger
	env     lib.Env
}

// NewCorsMiddleware creates a new CorsMiddleware.
func NewCorsMiddleware(handler lib.RequestHandler, logger port.Logger, env lib.Env) CorsMiddleware {
	return CorsMiddleware{
		handler: handler,
		logger:  logger,
		env:     env,
	}
}

// Setup registers the CORS middleware on the gin engine.
func (m CorsMiddleware) Setup() {
	m.logger.Info("Setting up cors middleware")

	debug := false
	m.handler.Gin.Use(cors.New(cors.Options{
		AllowCredentials: true,
		AllowOriginFunc:  func(origin string) bool { return true },
		AllowedHeaders:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "HEAD", "PATCH", "OPTIONS", "DELETE"},
		Debug:            debug,
	}))
}
