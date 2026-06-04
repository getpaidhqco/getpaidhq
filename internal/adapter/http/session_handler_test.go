package handler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

func newSessionHandlerForTest(
	t *testing.T,
	sess *fakeSessionRepo,
	cart *fakeCartRepo,
) *SessionHandler {
	t.Helper()
	svc := service.NewSessionService(sess, cart, silentLogger{}, newPubSub())
	return NewSessionHandler(svc, silentLogger{}, newRealAuthz(t))
}

func TestSessionHandler_Create(t *testing.T) {
	t.Run("owner can create a session — cart + session are persisted", func(t *testing.T) {
		sess := &fakeSessionRepo{}
		cart := &fakeCartRepo{}
		h := newSessionHandlerForTest(t, sess, cart)

		ts := newTestServer(fixedAuthMiddleware(ownerUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/sessions", CreateSessionRequest{
			Currency: "USD", Country: "US",
		})

		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		var got CreateSessionResponse
		decodeJSON(t, rec, &got)
		assert.NotEmpty(t, got.Id)
		assert.NotEmpty(t, got.CartId)
		require.Len(t, cart.created, 1, "a cart was created")
		require.Len(t, sess.created, 1, "a session was created")
	})

	t.Run("support role is denied by cedar — 403 envelope", func(t *testing.T) {
		// `support` has no permit rule in policy.cedar, so even the legitimate
		// flow gets rejected by Enforce before the service runs. Authz denial
		// renders as 403 Forbidden (the caller is authenticated; the action is
		// what's not permitted), distinct from a 401 authn failure.
		sess := &fakeSessionRepo{}
		cart := &fakeCartRepo{}
		h := newSessionHandlerForTest(t, sess, cart)

		ts := newTestServer(fixedAuthMiddleware(supportUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/sessions", CreateSessionRequest{
			Currency: "USD", Country: "US",
		})

		assertErrorEnvelope(t, rec, http.StatusForbidden, string(lib.ForbiddenError))
		assert.Empty(t, sess.created, "service must not run when authz denies")
	})

	t.Run("cart create failure bubbles as the underlying error envelope", func(t *testing.T) {
		// The service returns the raw repo error (not a CustomError), so the
		// envelope falls into NewApiErrorFromError's default branch with code
		// "bad_request" and HTTP 500.
		sess := &fakeSessionRepo{}
		cart := &fakeCartRepo{createErr: errors.New("disk full")}
		h := newSessionHandlerForTest(t, sess, cart)

		ts := newTestServer(fixedAuthMiddleware(ownerUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/sessions", CreateSessionRequest{
			Currency: "USD", Country: "US",
		})

		// The bare repo error has no CustomError type and no validator
		// errors, so the serializer hits the Fuego internal error path which
		// renders as 400 with the fallback "bad_request" code.
		envelope := assertErrorEnvelope(t, rec, http.StatusBadRequest, "bad_request")
		assert.NotEmpty(t, envelope.Message)
	})
}

// Sanity check on the silent guard: a stubbed authz that always denies still
// yields the same 403 envelope regardless of cedar's verdict, useful to pin
// the handler-side enforce wrapper independently of the policy file.
func TestSessionHandler_AuthzDeniedExplicit(t *testing.T) {
	sess := &fakeSessionRepo{}
	cart := &fakeCartRepo{}
	svc := service.NewSessionService(sess, cart, silentLogger{}, newPubSub())
	h := NewSessionHandler(svc, silentLogger{}, authzStub{allow: false})

	ts := newTestServer(fixedAuthMiddleware(port.AuthUser{Id: "u", OrgId: "o", PrimaryRole: port.RoleAdmin}))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/sessions", CreateSessionRequest{Currency: "USD", Country: "US"})

	assertErrorEnvelope(t, rec, http.StatusForbidden, string(lib.ForbiddenError))
}
