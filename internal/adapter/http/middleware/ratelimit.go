package middleware

import (
	"encoding/json"
	"getpaidhq/internal/lib/errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"getpaidhq/internal/core/port"
)

// RateLimitConfig configures the HTTP rate-limit middleware. The actual
// allow/deny accounting is delegated to a port.RateLimiter (in-memory or
// Redis-backed); this struct only carries the per-request budget and how to
// derive the client key.
type RateLimitConfig struct {
	// RPS is the sustained requests-per-second allowed per client key. A
	// value <= 0 disables the middleware (it becomes a pass-through).
	RPS int

	// Burst is the token-bucket capacity — the largest momentary spike a
	// client may make before being throttled to RPS. When <= 0 the limiter
	// defaults it to RPS.
	Burst int

	// KeyFunc derives the rate-limit key from a request — typically the
	// securely-resolved client IP. Required for the limiter to be active.
	KeyFunc func(*http.Request) string
}

// RateLimitMiddleware applies a per-client rate limit, delegating the decision
// to a port.RateLimiter. Over-limit requests receive a 429 with the project's
// standard error envelope and a Retry-After header. If the limiter backend
// errors (e.g. Redis is unreachable) the request is allowed through — failing
// OPEN, so a limiter outage degrades to "no limiting" rather than taking the
// whole API down.
type RateLimitMiddleware struct {
	logger  port.Logger
	limiter port.RateLimiter
	rps     int
	burst   int
	keyFunc func(*http.Request) string
}

// NewRateLimitMiddleware builds the middleware around a limiter backend. It is
// safe to construct even when disabled (RPS<=0, nil KeyFunc, or nil limiter):
// Handler then returns a transparent pass-through and Enabled reports false.
func NewRateLimitMiddleware(logger port.Logger, limiter port.RateLimiter, cfg RateLimitConfig) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		logger:  logger,
		limiter: limiter,
		rps:     cfg.RPS,
		burst:   cfg.Burst,
		keyFunc: cfg.KeyFunc,
	}
}

// Enabled reports whether the limiter will actually throttle. It is disabled
// when RPS is non-positive, no KeyFunc is set, or no backend limiter is wired.
func (m *RateLimitMiddleware) Enabled() bool {
	return m.rps > 0 && m.keyFunc != nil && m.limiter != nil
}

// Handler returns a net/http middleware suitable for
// fuego.WithGlobalMiddlewares. When disabled it returns an identity wrapper so
// there is zero per-request overhead.
func (m *RateLimitMiddleware) Handler() func(http.Handler) http.Handler {
	if !m.Enabled() {
		m.logger.Info("Rate limiting disabled (RATE_LIMIT_RPS<=0 or no backend)")
		return func(next http.Handler) http.Handler { return next }
	}
	m.logger.Info("Rate limiting enabled", "rps", m.rps, "burst", m.burst)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// CORS preflight carries no auth and should never be throttled
			// (mirrors the authn middleware's OPTIONS bypass); a throttled
			// preflight would break the real request that follows.
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			res, err := m.limiter.Allow(r.Context(), m.keyFunc(r), m.rps, m.burst)
			if err != nil {
				// Fail OPEN: a limiter-backend failure must not become an
				// API outage. Allow the request and log for visibility.
				m.logger.Warn("Rate limiter backend error; allowing request", "error", err.Error())
				next.ServeHTTP(w, r)
				return
			}
			if !res.Allowed {
				m.writeTooManyRequests(w, res.RetryAfter)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// writeTooManyRequests emits the project's standard 429 envelope plus a
// Retry-After header. As with the authn middleware's writeUnauthorized, we
// assemble it here rather than import the handler serializer (middleware lives
// upstream of handler, so importing it would create a dependency cycle). The
// code matches lib.RateLimitError, which the handler serializer also maps to
// 429, so clients see one stable identifier wherever the limit is enforced.
func (m *RateLimitMiddleware) writeTooManyRequests(w http.ResponseWriter, retryAfter time.Duration) {
	// Retry-After is whole seconds, rounded up, with a 1s floor so clients
	// always back off a sensible minimum.
	seconds := 1
	if s := int(math.Ceil(retryAfter.Seconds())); s > seconds {
		seconds = s
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":    string(errors.RateLimitError),
		"message": "rate limit exceeded",
		"details": nil,
	})
}
