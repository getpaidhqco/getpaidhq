package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/lib"
)

// noopMwLogger satisfies the package's logger interface without producing output.
type noopMwLogger struct{}

func (noopMwLogger) Debug(string, ...any)  {}
func (noopMwLogger) Info(string, ...any)   {}
func (noopMwLogger) Warn(string, ...any)   {}
func (noopMwLogger) Error(string, ...any)  {}
func (noopMwLogger) Fatal(string, ...any)  {}
func (noopMwLogger) Debugf(string, ...any) {}
func (noopMwLogger) Infof(string, ...any)  {}
func (noopMwLogger) Warnf(string, ...any)  {}
func (noopMwLogger) Errorf(string, ...any) {}
func (noopMwLogger) Panicf(string, ...any) {}
func (noopMwLogger) Fatalf(string, ...any) {}
func (noopMwLogger) Sync() error           { return nil }

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func TestParseOrigins(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty returns nil", "", nil},
		{"single origin", "https://app.example.com", []string{"https://app.example.com"}},
		{"comma-separated", "https://a.com,https://b.com", []string{"https://a.com", "https://b.com"}},
		{"whitespace trimmed", "  https://a.com  , https://b.com  ", []string{"https://a.com", "https://b.com"}},
		{"empty entries dropped", "https://a.com,,,https://b.com", []string{"https://a.com", "https://b.com"}},
		{"only commas/whitespace", " , ,  ", []string{}},
		{"wildcard", "*", []string{"*"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseOrigins(c.in)
			if c.want == nil {
				assert.Nil(t, got)
				return
			}
			assert.Equal(t, c.want, got)
		})
	}
}

func TestContains(t *testing.T) {
	assert.True(t, contains([]string{"a", "*", "b"}, "*"))
	assert.True(t, contains([]string{"a"}, "a"))
	assert.False(t, contains([]string{"a", "b"}, "c"))
	assert.False(t, contains(nil, "x"))
}

func TestCorsMiddleware_Wildcard_AllowsAnyOrigin(t *testing.T) {
	m := NewCorsMiddleware(noopMwLogger{}, lib.Env{AllowedOrigins: "*"})
	h := m.Handler()(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://random.example.org")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	// rs/cors echoes the wildcard or the request origin.
	allow := rec.Header().Get("Access-Control-Allow-Origin")
	assert.NotEmpty(t, allow, "wildcard config sets Access-Control-Allow-Origin")
	// Credentials must be off when wildcard is in play.
	assert.NotEqual(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCorsMiddleware_Allowlist_AllowsListedDeniesOther(t *testing.T) {
	m := NewCorsMiddleware(noopMwLogger{}, lib.Env{AllowedOrigins: "https://app.example.com"})
	h := m.Handler()(okHandler())

	// Listed origin: gets the echo.
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, "https://app.example.com", rec.Header().Get("Access-Control-Allow-Origin"))

	// Non-listed origin: no allow header.
	req2 := httptest.NewRequest(http.MethodOptions, "/", nil)
	req2.Header.Set("Origin", "https://evil.example.org")
	req2.Header.Set("Access-Control-Request-Method", "POST")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	assert.Empty(t, rec2.Header().Get("Access-Control-Allow-Origin"), "non-listed origin must not be echoed back")
}

func TestCorsMiddleware_Empty_BlocksAllCrossOrigin(t *testing.T) {
	m := NewCorsMiddleware(noopMwLogger{}, lib.Env{AllowedOrigins: ""})
	h := m.Handler()(okHandler())

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://anywhere.example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"), "no allowed origins means no cross-origin access")
}
