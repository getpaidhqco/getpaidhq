package middleware

import (
	"net/http"

	"github.com/rs/cors"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// CorsMiddleware builds the project's CORS handler. It is a thin wrapper
// over github.com/rs/cors so the configuration lives in one place.
type CorsMiddleware struct {
	logger port.Logger
	env    lib.Env
}

func NewCorsMiddleware(logger port.Logger, env lib.Env) CorsMiddleware {
	return CorsMiddleware{logger: logger, env: env}
}

// Handler returns a net/http middleware suitable for fuego.WithGlobalMiddlewares.
func (m CorsMiddleware) Handler() func(http.Handler) http.Handler {
	m.logger.Info("Setting up cors middleware")
	return cors.New(cors.Options{
		AllowCredentials: true,
		AllowOriginFunc:  func(origin string) bool { return true },
		AllowedHeaders:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "HEAD", "PATCH", "OPTIONS", "DELETE"},
	}).Handler
}
