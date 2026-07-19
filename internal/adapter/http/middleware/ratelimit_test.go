package middleware

import (
	"context"
	"encoding/json"
	"errors"
	errors2 "getpaidhq/internal/lib/errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/port"
)

// nopLogger is a do-nothing port.Logger so the middleware's startup logs don't
// spam test output. (The handler package has its own silentLogger; middleware
// tests can't import it, so we keep a local one.)
type nopLogger struct{}

func (nopLogger) Debug(string, ...any)  {}
func (nopLogger) Info(string, ...any)   {}
func (nopLogger) Warn(string, ...any)   {}
func (nopLogger) Error(string, ...any)  {}
func (nopLogger) Fatal(string, ...any)  {}
func (nopLogger) Debugf(string, ...any) {}
func (nopLogger) Infof(string, ...any)  {}
func (nopLogger) Warnf(string, ...any)  {}
func (nopLogger) Errorf(string, ...any) {}
func (nopLogger) Panicf(string, ...any) {}
func (nopLogger) Fatalf(string, ...any) {}
func (nopLogger) Sync() error           { return nil }

// stubLimiter is a programmable port.RateLimiter. It records calls so tests can
// assert keying/short-circuiting, and can be told to allow, deny (with a
// RetryAfter), or error.
type stubLimiter struct {
	mu      sync.Mutex
	allow   bool
	err     error
	retry   time.Duration
	calls   int
	lastKey string
	lastRPS int
}

func (s *stubLimiter) Allow(_ context.Context, key string, rps, _ int) (port.RateLimitResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.lastKey = key
	s.lastRPS = rps
	if s.err != nil {
		return port.RateLimitResult{}, s.err
	}
	return port.RateLimitResult{Allowed: s.allow, RetryAfter: s.retry}, nil
}

func (s *stubLimiter) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func keyByHeader(r *http.Request) string { return r.Header.Get("X-Client") }

func newCountingNext() (http.Handler, *int64) {
	var calls int64
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&calls, 1)
		w.WriteHeader(http.StatusOK)
	})
	return h, &calls
}

func serve(h http.Handler, method, clientKey string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/anything", nil)
	if clientKey != "" {
		req.Header.Set("X-Client", clientKey)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRateLimit_Disabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		limiter port.RateLimiter
		cfg     RateLimitConfig
	}{
		{
			name:    "rps <= 0 is a pass-through",
			limiter: &stubLimiter{allow: false}, // would deny if consulted
			cfg:     RateLimitConfig{RPS: 0, KeyFunc: keyByHeader},
		},
		{
			name:    "nil key func is a pass-through",
			limiter: &stubLimiter{allow: false},
			cfg:     RateLimitConfig{RPS: 100, KeyFunc: nil},
		},
		{
			name:    "nil limiter backend is a pass-through",
			limiter: nil,
			cfg:     RateLimitConfig{RPS: 100, KeyFunc: keyByHeader},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := NewRateLimitMiddleware(nopLogger{}, tt.limiter, tt.cfg)
			assert.False(t, m.Enabled())

			next, calls := newCountingNext()
			h := m.Handler()(next)

			for i := 0; i < 10; i++ {
				require.Equal(t, http.StatusOK, serve(h, http.MethodGet, "alice").Code)
			}
			assert.Equal(t, int64(10), *calls)
			if sl, ok := tt.limiter.(*stubLimiter); ok {
				assert.Zero(t, sl.callCount(), "disabled middleware must not consult the backend")
			}
		})
	}
}

func TestRateLimit_AllowsWhenBackendAllows(t *testing.T) {
	t.Parallel()
	stub := &stubLimiter{allow: true}
	m := NewRateLimitMiddleware(nopLogger{}, stub, RateLimitConfig{RPS: 5, Burst: 10, KeyFunc: keyByHeader})
	require.True(t, m.Enabled())

	next, calls := newCountingNext()
	h := m.Handler()(next)

	rec := serve(h, http.MethodGet, "alice")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, int64(1), *calls)
	assert.Equal(t, "alice", stub.lastKey, "middleware keys the limiter via KeyFunc")
	assert.Equal(t, 5, stub.lastRPS, "configured RPS is passed to the backend")
}

func TestRateLimit_BlocksWhenBackendDenies(t *testing.T) {
	t.Parallel()
	stub := &stubLimiter{allow: false, retry: 2 * time.Second}
	m := NewRateLimitMiddleware(nopLogger{}, stub, RateLimitConfig{RPS: 1, KeyFunc: keyByHeader})

	next, calls := newCountingNext()
	h := m.Handler()(next)

	rec := serve(h, http.MethodGet, "alice")
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "2", rec.Header().Get("Retry-After"), "Retry-After reflects the backend's hint")
	assert.Equal(t, int64(0), *calls, "denied request must not reach next")
}

func TestRateLimit_FailsOpenOnBackendError(t *testing.T) {
	t.Parallel()
	// A Redis blip must not become an API outage: on a backend error the
	// request is allowed through.
	stub := &stubLimiter{err: errors.New("redis down")}
	m := NewRateLimitMiddleware(nopLogger{}, stub, RateLimitConfig{RPS: 1, KeyFunc: keyByHeader})

	next, calls := newCountingNext()
	h := m.Handler()(next)

	rec := serve(h, http.MethodGet, "alice")
	require.Equal(t, http.StatusOK, rec.Code, "limiter error fails OPEN")
	assert.Equal(t, int64(1), *calls)
}

func TestRateLimit_OptionsBypass(t *testing.T) {
	t.Parallel()
	// Even a deny-everything backend must not throttle CORS preflight.
	stub := &stubLimiter{allow: false}
	m := NewRateLimitMiddleware(nopLogger{}, stub, RateLimitConfig{RPS: 1, KeyFunc: keyByHeader})

	next, calls := newCountingNext()
	h := m.Handler()(next)

	for i := 0; i < 5; i++ {
		require.Equal(t, http.StatusOK, serve(h, http.MethodOptions, "alice").Code)
	}
	assert.Equal(t, int64(5), *calls)
	assert.Zero(t, stub.callCount(), "OPTIONS must bypass the limiter entirely")
}

func TestRateLimit_TooManyRequestsEnvelope(t *testing.T) {
	t.Parallel()
	stub := &stubLimiter{allow: false, retry: 500 * time.Millisecond}
	m := NewRateLimitMiddleware(nopLogger{}, stub, RateLimitConfig{RPS: 2, KeyFunc: keyByHeader})

	next, _ := newCountingNext()
	h := m.Handler()(next)

	rec := serve(h, http.MethodGet, "alice")
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	// Sub-second RetryAfter rounds up to the 1s floor.
	assert.Equal(t, "1", rec.Header().Get("Retry-After"))

	var env struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details any    `json:"details"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env), "body=%s", rec.Body.String())
	assert.Equal(t, string(errors2.RateLimitError), env.Code)
	assert.Equal(t, "rate limit exceeded", env.Message)
	assert.Nil(t, env.Details)
}
