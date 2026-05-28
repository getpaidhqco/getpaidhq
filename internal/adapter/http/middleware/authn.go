package middleware

import (
	"context"
	"encoding/json"
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
			// CORS preflight requests carry no auth headers by design.
			// Skip authn and let the CORS layer (or the route) respond.
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			token := r.Header.Get("Authorization")
			if token == "" {
				token = r.Header.Get("x-api-key")
			}

			var (
				user          port.AuthUser
				authenticated bool
				lastErr       error
			)
			for _, authenticator := range m.authnList {
				u, err := authenticator.Authenticate(r.Context(), token)
				if err != nil {
					lastErr = err
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
				reason := "no token"
				if lastErr != nil {
					reason = lastErr.Error()
				}
				m.logger.Error(
					"Authentication failed",
					"reason", reason,
					"path", r.URL.Path,
					"method", r.Method,
					"has_authorization", r.Header.Get("Authorization") != "",
					"has_api_key", r.Header.Get("x-api-key") != "",
				)
				writeUnauthorized(w)
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

// writeUnauthorized emits the project's standard 401 envelope. The code value
// is the same one lib.AuthenticationError serializes to, so clients see one
// stable identifier whether the failure originated in this middleware or in
// the handler error serializer downstream.
//
// We assemble the envelope here (rather than importing handler.ApiErrorSerializer)
// because middleware lives upstream of the handler package and importing it
// would create a dependency cycle.
func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":    string(lib.AuthenticationError),
		"message": "unauthorized",
		"details": nil,
	})
}
