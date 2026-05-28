package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

func newWebhookSubHandlerForTest(t *testing.T, repo *fakeWhSubRepo) *WebhookSubscriptionHandler {
	t.Helper()
	svc := service.NewWebhookSubscriptionService(silentLogger{}, repo, &fakeIdempRepo{}, newPubSub())
	return NewWebhookSubscriptionHandler(svc, silentLogger{}, newRealAuthz(t))
}

func TestWebhookSubscriptionHandler_Create(t *testing.T) {
	t.Run("admin role creates the subscription", func(t *testing.T) {
		// CreateWebhookSubscription is admin-only by policy.
		repo := &fakeWhSubRepo{}
		h := newWebhookSubHandlerForTest(t, repo)

		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/webhooks", CreateWebhookSubscriptionRequest{
			Url:    "https://example.com/wh",
			Events: []string{"order.created"},
			Secret: "shh",
		})

		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		require.Len(t, repo.created, 1)
		assert.Equal(t, "https://example.com/wh", repo.created[0].URL)
		assert.Equal(t, "org_1", repo.created[0].OrgID, "OrgId pulled from auth user")
	})

	t.Run("non-admin denied by cedar", func(t *testing.T) {
		h := newWebhookSubHandlerForTest(t, &fakeWhSubRepo{})

		ts := newTestServer(fixedAuthMiddleware(ownerUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPost, "/api/webhooks", CreateWebhookSubscriptionRequest{
			Url: "https://example.com/wh", Events: []string{"order.created"},
		})

		assertErrorEnvelope(t, rec, http.StatusUnauthorized, string(lib.AuthenticationError))
	})
}

func TestWebhookSubscriptionHandler_List(t *testing.T) {
	// List is currently a placeholder returning an empty payload — the test
	// asserts the route is wired and the placeholder shape comes out as JSON.
	h := newWebhookSubHandlerForTest(t, &fakeWhSubRepo{})

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/webhooks", nil)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got ListResponse
	decodeJSON(t, rec, &got)
	assert.Equal(t, 0, got.Meta.Total)
}

// Confirm the un-set domain validation still runs: missing required Url and
// Events should fail the body validator (Fuego-level).
func TestWebhookSubscriptionHandler_Create_Validation(t *testing.T) {
	h := newWebhookSubHandlerForTest(t, &fakeWhSubRepo{})
	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/webhooks", map[string]any{})

	// Fuego rejects the body before the handler runs.
	assert.GreaterOrEqual(t, rec.Code, 400)
	assert.Less(t, rec.Code, 500)
}
