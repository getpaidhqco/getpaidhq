package middleware

import (
	"net/http"
	"strings"

	"github.com/rs/cors"

	"getpaidhq/internal/core/port"
)

// CorsMiddleware builds the project's CORS handler. allowedOrigins is the
// raw comma-separated origin list (ALLOWED_ORIGINS). An explicit "*" entry
// enables permissive CORS for development; an empty value rejects every
// cross-origin request.
type CorsMiddleware struct {
	logger         port.Logger
	allowedOrigins string
}

func NewCorsMiddleware(logger port.Logger, allowedOrigins string) CorsMiddleware {
	return CorsMiddleware{logger: logger, allowedOrigins: allowedOrigins}
}

// Handler returns a net/http middleware suitable for fuego.WithGlobalMiddlewares.
func (m CorsMiddleware) Handler() func(http.Handler) http.Handler {
	allowed := parseOrigins(m.allowedOrigins)
	openCors := contains(allowed, "*")

	opts := cors.Options{
		AllowCredentials: !openCors, // wildcard + credentials is illegal per CORS spec
		AllowedHeaders:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "HEAD", "PATCH", "OPTIONS", "DELETE"},
	}

	switch {
	case openCors:
		m.logger.Info("CORS configured with wildcard origin (ALLOWED_ORIGINS=\"*\")")
		opts.AllowOriginFunc = func(string) bool { return true }
	case len(allowed) == 0:
		m.logger.Info("CORS configured with no allowed origins (ALLOWED_ORIGINS unset)")
		opts.AllowOriginFunc = func(string) bool { return false }
	default:
		m.logger.Info("CORS allowed origins", "origins", allowed)
		opts.AllowedOrigins = allowed
	}

	return cors.New(opts).Handler
}

func parseOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func contains(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}
