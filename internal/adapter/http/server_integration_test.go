package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"getpaidhq/internal/lib/errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/adapter/http/middleware"
	"getpaidhq/internal/core/port"
)

// fixedAuthenticator returns a hard-coded AuthUser for every token. Mirrors
// the upstream Authenticator surface so the real middleware can use it.
type fixedAuthenticator struct {
	user port.AuthUser
	err  error
}

func (f fixedAuthenticator) Authenticate(_ context.Context, _ string) (port.AuthUser, error) {
	return f.user, f.err
}

// TestAuthnMiddleware_RejectsUnauthenticated mirrors what BuildServer does in
// production: stack the Authn middleware on a mux that includes a single
// handler. A request with no Authorization header is rejected before the
// handler ever runs.
func TestAuthnMiddleware_RejectsUnauthenticated(t *testing.T) {
	authn := fixedAuthenticator{err: errors.NewCustomError(errors.AuthenticationError, "no token", nil)}
	wrap := middleware.NewAuthnWrapperMiddleware([]port.Authenticator{authn}, silentLogger{}).Handler()

	ts := newTestServer(wrap)
	NewHealthHandler(silentLogger{}).RegisterRoutes(ts.api())

	// Use a gated path, NOT /api/health (which is intentionally public). authn
	// is a global middleware that runs before routing, so an unauthenticated
	// request to any non-public path is rejected with 401 before it routes.
	req := httptest.NewRequest(http.MethodGet, "/api/customers", nil)
	rec := httptest.NewRecorder()
	asHandler(ts.srv, ts.mws).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthnMiddleware_AcceptsValidToken(t *testing.T) {
	authn := fixedAuthenticator{user: ownerUser()}
	wrap := middleware.NewAuthnWrapperMiddleware([]port.Authenticator{authn}, silentLogger{}).Handler()

	ts := newTestServer(wrap)
	NewHealthHandler(silentLogger{}).RegisterRoutes(ts.api())

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Authorization", "Bearer good")
	rec := httptest.NewRecorder()
	asHandler(ts.srv, ts.mws).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
}

// TestOnboardingBypass exercises the contract documented in middleware/authn:
// POST /api/organizations bypasses the normal auth-required gate when the
// authenticator returns ErrOnboardingRequired. The org handler should then
// receive a partial AuthUser on the context.
func TestOnboardingBypass(t *testing.T) {
	partial := port.AuthUser{Id: "user_new", Email: "new@example.com"}
	authn := fixedAuthenticator{user: partial, err: port.ErrOnboardingRequired}
	wrap := middleware.NewAuthnWrapperMiddleware([]port.Authenticator{authn}, silentLogger{}).Handler()

	orgRepo := &fakeOrgRepo{}
	h := newOrgHandlerForTest(orgRepo, &fakeCustomerRepo{}, &fakeApiKeyRepo{})

	ts := newTestServer(wrap)
	h.RegisterRoutes(ts.api())

	raw, err := json.Marshal(CreateOrgRequest{Name: "Acme", Country: "US", Timezone: "UTC"})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/organizations", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer onboarding-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	asHandler(ts.srv, ts.mws).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	require.Len(t, orgRepo.created, 1, "the partial AuthUser must still let org creation proceed")
}

// TestOnboardingBypass_NotAppliedElsewhere verifies that the bypass is
// scoped to POST /api/organizations only — any other path still gets 401.
func TestOnboardingBypass_NotAppliedElsewhere(t *testing.T) {
	partial := port.AuthUser{Id: "user_new"}
	authn := fixedAuthenticator{user: partial, err: port.ErrOnboardingRequired}
	wrap := middleware.NewAuthnWrapperMiddleware([]port.Authenticator{authn}, silentLogger{}).Handler()

	ts := newTestServer(wrap)
	NewHealthHandler(silentLogger{}).RegisterRoutes(ts.api())

	// A gated path (not the public /api/health): the onboarding bypass must
	// NOT apply here, so the partial/ErrOnboardingRequired user is rejected.
	req := httptest.NewRequest(http.MethodGet, "/api/customers", nil)
	req.Header.Set("Authorization", "Bearer onboarding-token")
	rec := httptest.NewRecorder()
	asHandler(ts.srv, ts.mws).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
