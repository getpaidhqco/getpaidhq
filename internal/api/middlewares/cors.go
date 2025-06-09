package middlewares

import (
	cors "github.com/rs/cors/wrapper/gin"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

// CorsMiddleware middleware for cors
type CorsMiddleware struct {
	handler lib.RequestHandler
	logger  logger.Logger
	env     lib.Env
}

// NewCorsMiddleware creates new cors middleware
func NewCorsMiddleware(handler lib.RequestHandler, logger logger.Logger, env lib.Env) CorsMiddleware {
	return CorsMiddleware{
		handler: handler,
		logger:  logger,
		env:     env,
	}
}

// Setup sets up cors middleware
func (m CorsMiddleware) Setup() {
	m.logger.Info("Setting up cors middleware")

	debug := false // m.env.Env == "development"
	m.handler.Gin.Use(cors.New(cors.Options{
		AllowCredentials: true,
		AllowOriginFunc:  func(origin string) bool { return true },
		AllowedHeaders:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "HEAD", "PATCH", "OPTIONS", "DELETE"},
		Debug:            debug,
	}))
}
