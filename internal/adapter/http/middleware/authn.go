package middleware

import (
	"context"
	"errors"
	"net/http"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// ctxKey is an unexported type so collisions with other packages' keys are
// impossible. Use the exported AuthUserKey constant to read the value.
type ctxKey int

// AuthUserKey is the context key under which a successfully authenticated
// port.AuthUser is stored by AuthnWrapperMiddleware.
const AuthUserKey ctxKey = 1

// AuthnWrapperMiddleware tries each configured authenticator in order
// against the incoming Authorization / x-api-key header. The first
// successful one wins; otherwise the request is rejected with 401.
type AuthnWrapperMiddleware struct {
	authnList []port.Authenticator
	logger    port.Logger
	env       lib.Env
}

func NewAuthnWrapperMiddleware(
	authenticators []port.Authenticator,
	logger port.Logger,
	env lib.Env,
) AuthnWrapperMiddleware {
	return AuthnWrapperMiddleware{
		authnList: authenticators,
		logger:    logger,
		env:       env,
	}
}

// Handler returns the middleware suitable for fuego.WithGlobalMiddlewares.
func (m AuthnWrapperMiddleware) Handler() func(http.Handler) http.Handler {
	m.logger.Info("Setting up authn wrapper middleware")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")
			if token == "" {
				token = r.Header.Get("x-api-key")
			}

			var (
				user          port.AuthUser
				authenticated bool
			)
			for _, authenticator := range m.authnList {
				u, err := authenticator.Authenticate(r.Context(), token)
				if err != nil {
					// Onboarding bypass: a fresh Clerk user with no active
					// org needs to create one before any other API call is
					// possible. Let POST /api/organizations through with the
					// partial AuthUser so the org-creation handler can run.
					if errors.Is(err, port.ErrOnboardingRequired) &&
						r.Method == http.MethodPost &&
						r.URL.Path == "/api/organizations" {
						user = u
						authenticated = true
						break
					}
					continue
				}
				user = u
				authenticated = true
				break
			}

			if !authenticated {
				m.logger.Error("Authentication failed", "message", "unauthorized access")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"code":"authentication_error","message":"unauthorized","details":null}`))
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), AuthUserKey, user)))
		})
	}
}

// AuthUserFrom retrieves the authenticated user previously stored on the
// request context by AuthnWrapperMiddleware. The boolean is false when no
// authn middleware was on the request path (e.g. in tests).
func AuthUserFrom(ctx context.Context) (port.AuthUser, bool) {
	u, ok := ctx.Value(AuthUserKey).(port.AuthUser)
	return u, ok
}
