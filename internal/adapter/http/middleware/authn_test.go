package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noopLogger satisfies lib.Logger (== port.Logger) without producing output.
type noopLogger struct{}

func (noopLogger) Debug(string, ...any)  {}
func (noopLogger) Info(string, ...any)   {}
func (noopLogger) Warn(string, ...any)   {}
func (noopLogger) Error(string, ...any)  {}
func (noopLogger) Fatal(string, ...any)  {}
func (noopLogger) Debugf(string, ...any) {}
func (noopLogger) Infof(string, ...any)  {}
func (noopLogger) Warnf(string, ...any)  {}
func (noopLogger) Errorf(string, ...any) {}
func (noopLogger) Panicf(string, ...any) {}
func (noopLogger) Fatalf(string, ...any) {}
func (noopLogger) Sync() error           { return nil }

// fakeAuthenticator is a fully controllable port.Authenticator. It records the
// token it was handed and returns a fixed user/error.
type fakeAuthenticator struct {
	user      port.AuthUser
	err       error
	calls     int
	tokenSeen string
}

func (f *fakeAuthenticator) Authenticate(_ context.Context, token string) (port.AuthUser, error) {
	f.calls++
	f.tokenSeen = token
	return f.user, f.err
}

// nextRecorder is an http.Handler that records whether it was invoked and the
// AuthUser present on the request context when it ran.
type nextRecorder struct {
	called  bool
	gotUser port.AuthUser
	userOk  bool
}

func (n *nextRecorder) ServeHTTP(_ http.ResponseWriter, r *http.Request) {
	n.called = true
	n.gotUser, n.userOk = AuthUserFrom(r.Context())
}

func newMiddleware(authns ...port.Authenticator) AuthnWrapperMiddleware {
	return NewAuthnWrapperMiddleware(authns, noopLogger{}, lib.Env{})
}

func TestAuthnWrapperMiddleware_Handler(t *testing.T) {
	validUser := port.NewAuthUser("org_1", "user_1", "a@b.com", []port.UserRole{port.RoleAdmin})

	t.Run("valid auth stores AuthUser on context and calls next", func(t *testing.T) {
		auth := &fakeAuthenticator{user: validUser}
		next := &nextRecorder{}
		h := newMiddleware(auth).Handler()(next)

		req := httptest.NewRequest(http.MethodGet, "/api/orders", nil)
		req.Header.Set("Authorization", "Bearer good-token")
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.True(t, next.called, "next should be called on success")
		assert.True(t, next.userOk, "AuthUser must be present on the context")
		assert.Equal(t, validUser, next.gotUser)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "Bearer good-token", auth.tokenSeen)
	})

	t.Run("falls back to x-api-key when Authorization header absent", func(t *testing.T) {
		auth := &fakeAuthenticator{user: validUser}
		next := &nextRecorder{}
		h := newMiddleware(auth).Handler()(next)

		req := httptest.NewRequest(http.MethodGet, "/api/orders", nil)
		req.Header.Set("x-api-key", "secret-key")
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.True(t, next.called)
		assert.Equal(t, "secret-key", auth.tokenSeen)
	})

	t.Run("auth failure returns 401 and does not call next", func(t *testing.T) {
		auth := &fakeAuthenticator{err: errors.New("bad token")}
		next := &nextRecorder{}
		h := newMiddleware(auth).Handler()(next)

		req := httptest.NewRequest(http.MethodGet, "/api/orders", nil)
		req.Header.Set("Authorization", "Bearer nope")
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.False(t, next.called, "next must not be called when unauthenticated")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		// The code must match what lib.AuthenticationError serializes to —
		// previously the middleware emitted a hardcoded "authentication_error"
		// literal that disagreed with the handler-layer envelope, so clients
		// saw two different codes for the same authn failure depending on
		// where the rejection happened.
		assert.JSONEq(t, `{"code":"auth_error","message":"unauthorized","details":null}`, rec.Body.String())
	})

	t.Run("missing token still routed through authenticator and rejected", func(t *testing.T) {
		auth := &fakeAuthenticator{err: errors.New("not allowed")}
		next := &nextRecorder{}
		h := newMiddleware(auth).Handler()(next)

		req := httptest.NewRequest(http.MethodGet, "/api/orders", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.False(t, next.called)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, "", auth.tokenSeen, "empty token forwarded to authenticator")
		assert.Equal(t, 1, auth.calls)
	})

	t.Run("OPTIONS preflight bypasses authn entirely", func(t *testing.T) {
		auth := &fakeAuthenticator{err: errors.New("should not be called")}
		next := &nextRecorder{}
		h := newMiddleware(auth).Handler()(next)

		req := httptest.NewRequest(http.MethodOptions, "/api/orders", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.True(t, next.called, "preflight must pass through to next")
		assert.Equal(t, 0, auth.calls, "authenticator must not run for OPTIONS")
		assert.False(t, next.userOk, "no AuthUser is stored for preflight")
	})

	t.Run("first successful authenticator wins; later ones not tried", func(t *testing.T) {
		first := &fakeAuthenticator{err: errors.New("first fails")}
		second := &fakeAuthenticator{user: validUser}
		third := &fakeAuthenticator{user: port.AuthUser{Id: "should-not-be-used"}}
		next := &nextRecorder{}
		h := newMiddleware(first, second, third).Handler()(next)

		req := httptest.NewRequest(http.MethodGet, "/api/orders", nil)
		req.Header.Set("Authorization", "Bearer t")
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.True(t, next.called)
		assert.Equal(t, validUser, next.gotUser)
		assert.Equal(t, 1, first.calls)
		assert.Equal(t, 1, second.calls)
		assert.Equal(t, 0, third.calls, "third authenticator must not be tried after a success")
	})
}

func TestAuthnWrapperMiddleware_OnboardingBypass(t *testing.T) {
	// A fresh user with no active org: authenticator returns a partial user and
	// ErrOnboardingRequired.
	partialUser := port.AuthUser{Id: "user_new", Email: "new@b.com"}

	tests := []struct {
		name        string
		method      string
		path        string
		wantNext    bool
		wantStatus  int
		wantUserSet bool
	}{
		{
			name:        "POST /api/organizations is allowed through with partial user",
			method:      http.MethodPost,
			path:        "/api/organizations",
			wantNext:    true,
			wantStatus:  http.StatusOK,
			wantUserSet: true,
		},
		{
			name:       "GET /api/organizations is rejected (wrong method)",
			method:     http.MethodGet,
			path:       "/api/organizations",
			wantNext:   false,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "POST to a different path is rejected",
			method:     http.MethodPost,
			path:       "/api/orders",
			wantNext:   false,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "GET to a different path is rejected",
			method:     http.MethodGet,
			path:       "/api/orders",
			wantNext:   false,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := &fakeAuthenticator{user: partialUser, err: port.ErrOnboardingRequired}
			next := &nextRecorder{}
			h := newMiddleware(auth).Handler()(next)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Authorization", "Bearer onboarding-token")
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantNext, next.called)
			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantUserSet {
				require.True(t, next.userOk, "partial AuthUser should be on context for onboarding")
				assert.Equal(t, partialUser, next.gotUser)
			}
		})
	}
}

func TestAuthUserFrom(t *testing.T) {
	t.Run("returns false when no user on context", func(t *testing.T) {
		_, ok := AuthUserFrom(context.Background())
		assert.False(t, ok)
	})

	t.Run("returns stored user", func(t *testing.T) {
		u := port.NewAuthUser("org", "id", "e@e.com", []port.UserRole{port.RoleMember})
		ctx := context.WithValue(context.Background(), AuthUserKey, u)
		got, ok := AuthUserFrom(ctx)
		require.True(t, ok)
		assert.Equal(t, u, got)
	})
}

// TestAuthn_PublicPathsBypass pins the unauthenticated allowlist: liveness
// probes and the PSP webhook receiver must reach the handler WITHOUT a token,
// while a comparable non-listed path still gets 401. The authenticator always
// errors, so any non-public path is rejected and never reaches next.
func TestAuthn_PublicPathsBypass(t *testing.T) {
	cases := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantReach  bool
	}{
		{"health is public", http.MethodGet, "/api/health", http.StatusOK, true},
		{"webhook notify is public", http.MethodPost, "/api/notify", http.StatusOK, true},
		{"health only via GET", http.MethodPost, "/api/health", http.StatusUnauthorized, false},
		{"other paths still gated", http.MethodGet, "/api/customers", http.StatusUnauthorized, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			auth := &fakeAuthenticator{err: errors.New("no token")}
			next := &nextRecorder{}
			h := newMiddleware(auth).Handler()(next)

			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			require.Equal(t, tc.wantStatus, rec.Code)
			require.Equal(t, tc.wantReach, next.called)
			if tc.wantReach {
				require.Zero(t, auth.calls, "public paths must not invoke the authenticator")
			}
		})
	}
}
