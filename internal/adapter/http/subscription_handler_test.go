package handler

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

func newSubscriptionHandlerForTest(
	t *testing.T,
	subRepo *fakeSubRepo,
	payRepo *fakePaymentRepo,
	engine *recordingEngine,
) *SubscriptionHandler {
	t.Helper()
	narrow, err := service.NewSubscriptionService(
		&fakeSessionRepo{}, &fakeSettingRepo{}, &fakeCartRepo{},
		subRepo, &fakeCustomerRepo{}, &fakeOrderRepo{}, payRepo,
		// no gateway factory needed for the handler-level cases (they don't
		// trigger charge attempts).
		service.NewGatewayFactory(&fakePspRepo{}, &fakeSettingRepo{}, silentLogger{}, map[domain.Gateway]port.GatewayAdapter{}),
		newPubSub(), lib.NewErrorReporter(silentLogger{}), silentLogger{}, nil,
	)
	if err != nil {
		t.Fatalf("NewSubscriptionService: %v", err)
	}
	orch := service.NewSubscriptionOrchestrationService(narrow, engine, silentLogger{})
	return NewSubscriptionHandler(orch, silentLogger{}, newRealAuthz(t))
}

func TestSubscriptionHandler_Get(t *testing.T) {
	t.Run("happy path returns the subscription response", func(t *testing.T) {
		subRepo := &fakeSubRepo{byId: domain.Subscription{
			OrgId: "org_1", Id: "sub_1", Amount: 1000, Currency: "USD",
			Status: domain.SubscriptionStatusActive,
		}}
		h := newSubscriptionHandlerForTest(t, subRepo, &fakePaymentRepo{}, &recordingEngine{})

		ts := newTestServer(fixedAuthMiddleware(ownerUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodGet, "/api/subscriptions/sub_1", nil)

		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		var got SubscriptionResponse
		decodeJSON(t, rec, &got)
		assert.Equal(t, "sub_1", got.Id)
		assert.Equal(t, int64(1000), got.Amount)
		assert.EqualValues(t, "active", got.Status)
	})

	t.Run("repo error becomes the fallback envelope", func(t *testing.T) {
		subRepo := &fakeSubRepo{byIdErr: errors.New("missing")}
		h := newSubscriptionHandlerForTest(t, subRepo, &fakePaymentRepo{}, &recordingEngine{})

		ts := newTestServer(fixedAuthMiddleware(ownerUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodGet, "/api/subscriptions/sub_x", nil)

		assertErrorEnvelope(t, rec, http.StatusBadRequest, "bad_request")
	})
}

func TestSubscriptionHandler_List(t *testing.T) {
	subRepo := &fakeSubRepo{listResult: []domain.Subscription{
		{Id: "sub_1"}, {Id: "sub_2"},
	}}
	h := newSubscriptionHandlerForTest(t, subRepo, &fakePaymentRepo{}, &recordingEngine{})

	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/subscriptions?page=0&limit=20", nil)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got ListResponse
	decodeJSON(t, rec, &got)
	assert.Equal(t, 2, got.Meta.Total)
	assert.Equal(t, 20, got.Meta.Limit)
}

func TestSubscriptionHandler_Pause(t *testing.T) {
	t.Run("happy path persists the paused status and signals the engine", func(t *testing.T) {
		subRepo := &fakeSubRepo{byId: domain.Subscription{
			OrgId: "org_1", Id: "sub_1", Status: domain.SubscriptionStatusActive,
		}}
		engine := &recordingEngine{}
		h := newSubscriptionHandlerForTest(t, subRepo, &fakePaymentRepo{}, engine)

		ts := newTestServer(fixedAuthMiddleware(ownerUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPut, "/api/subscriptions/sub_1/pause", PauseSubscriptionRequest{
			Reason: "user requested",
		})

		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		require.Len(t, subRepo.updated, 1)
		assert.Equal(t, domain.SubscriptionStatusPaused, subRepo.updated[0].Status)
		// SubscriptionOrchestrationService.PauseSubscription signals the engine
		// via UpdateSubscriptionWorkflow with "subscription.paused".
		assert.Contains(t, engine.updateName, "subscription.paused")
	})

	t.Run("pausing an already-paused subscription returns bad_request envelope", func(t *testing.T) {
		subRepo := &fakeSubRepo{byId: domain.Subscription{
			Id: "sub_1", Status: domain.SubscriptionStatusPaused,
		}}
		h := newSubscriptionHandlerForTest(t, subRepo, &fakePaymentRepo{}, &recordingEngine{})

		ts := newTestServer(fixedAuthMiddleware(ownerUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodPut, "/api/subscriptions/sub_1/pause", PauseSubscriptionRequest{})

		assertErrorEnvelope(t, rec, http.StatusBadRequest, string(lib.BadRequestError))
	})
}

func TestSubscriptionHandler_Resume(t *testing.T) {
	subRepo := &fakeSubRepo{byId: domain.Subscription{
		OrgId: "org_1", Id: "sub_1", Status: domain.SubscriptionStatusPaused,
	}}
	engine := &recordingEngine{}
	h := newSubscriptionHandlerForTest(t, subRepo, &fakePaymentRepo{}, engine)

	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPut, "/api/subscriptions/sub_1/resume", ResumeSubscriptionRequest{
		// StartNewBillingPeriod doesn't need the saved RenewsAt to be in the
		// future — keeps the test deterministic.
		ResumeBehavior: domain.StartNewBillingPeriod,
	})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	require.Len(t, subRepo.updated, 1)
	// The orchestration service signals the engine on resume too.
	assert.NotEmpty(t, engine.updateName)
}

func TestSubscriptionHandler_Cancel(t *testing.T) {
	subRepo := &fakeSubRepo{byId: domain.Subscription{
		OrgId: "org_1", Id: "sub_1", Status: domain.SubscriptionStatusActive,
	}}
	engine := &recordingEngine{}
	h := newSubscriptionHandlerForTest(t, subRepo, &fakePaymentRepo{}, engine)

	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPut, "/api/subscriptions/sub_1/cancel", PauseSubscriptionRequest{Reason: "x"})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	require.Len(t, subRepo.updated, 1)
	assert.Equal(t, domain.SubscriptionStatusCancelled, subRepo.updated[0].Status)
}

func TestSubscriptionHandler_UpdateBillingAnchor_Validation(t *testing.T) {
	// billing_anchor must be 1..31 — Fuego/validator should reject 99.
	h := newSubscriptionHandlerForTest(t, &fakeSubRepo{}, &fakePaymentRepo{}, &recordingEngine{})

	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPatch, "/api/subscriptions/sub_1/billing-anchor", UpdateBillingAnchorRequest{
		BillingAnchor: 99, ProrationMode: domain.ProrationMode("none"),
	})

	// 4xx — could be 400 or 422 depending on which path validation surfaces.
	assert.GreaterOrEqual(t, rec.Code, 400)
	assert.Less(t, rec.Code, 500)
}

func TestSubscriptionHandler_Update_PassesMetadata(t *testing.T) {
	subRepo := &fakeSubRepo{byId: domain.Subscription{
		OrgId: "org_1", Id: "sub_1", Status: domain.SubscriptionStatusActive,
	}}
	h := newSubscriptionHandlerForTest(t, subRepo, &fakePaymentRepo{}, &recordingEngine{})

	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPatch, "/api/subscriptions/sub_1", domain.UpdateSubscriptionRequest{
		Metadata: map[string]string{"k": "v"},
	})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	require.Len(t, subRepo.updated, 1)
}

func TestSubscriptionHandler_ListPayments(t *testing.T) {
	payRepo := &fakePaymentRepo{bySub: []domain.Payment{
		{Id: "pay_1", Reference: "ref_1", Amount: 1000},
	}}
	h := newSubscriptionHandlerForTest(t, &fakeSubRepo{}, payRepo, &recordingEngine{})

	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/subscriptions/sub_1/payments?page=0&limit=10", nil)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got ListResponse
	decodeJSON(t, rec, &got)
	assert.Equal(t, 1, got.Meta.Total)
}

func TestSubscriptionHandler_AuthzDenied(t *testing.T) {
	// support has no permit rule for any subscription mutation → cedar
	// denies before the service runs. One table-driven test covers all five
	// mutating routes so a future authz regression on any of them fails
	// loudly here.
	cases := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"Update", http.MethodPatch, "/api/subscriptions/sub_1", domain.UpdateSubscriptionRequest{Metadata: map[string]string{"k": "v"}}},
		{"Pause", http.MethodPut, "/api/subscriptions/sub_1/pause", PauseSubscriptionRequest{}},
		{"Resume", http.MethodPut, "/api/subscriptions/sub_1/resume", ResumeSubscriptionRequest{ResumeBehavior: domain.StartNewBillingPeriod}},
		{"Cancel", http.MethodPut, "/api/subscriptions/sub_1/cancel", PauseSubscriptionRequest{Reason: "x"}},
		{"UpdateBillingAnchor", http.MethodPatch, "/api/subscriptions/sub_1/billing-anchor", UpdateBillingAnchorRequest{BillingAnchor: 15, ProrationMode: domain.ProrationModeNone}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			subRepo := &fakeSubRepo{}
			h := newSubscriptionHandlerForTest(t, subRepo, &fakePaymentRepo{}, &recordingEngine{})
			ts := newTestServer(fixedAuthMiddleware(supportUser()))
			h.RegisterRoutes(ts.api())

			rec := doJSON(t, ts, tt.method, tt.path, tt.body)

			assertErrorEnvelope(t, rec, http.StatusForbidden, string(lib.ForbiddenError))
			assert.Empty(t, subRepo.updated, "service must not run when authz denies")
		})
	}
}

func TestSubscriptionHandler_UpdateBillingAnchor_HappyPath(t *testing.T) {
	// The narrow service computes proration in-memory against the persisted
	// subscription. Seed the existing sub so the proration math succeeds.
	subRepo := &fakeSubRepo{byId: domain.Subscription{
		OrgId: "org_1", Id: "sub_1", Status: domain.SubscriptionStatusActive,
		BillingAnchor: 1, Amount: 1000,
		CurrentPeriodStart: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		CurrentPeriodEnd:   time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC),
	}}
	h := newSubscriptionHandlerForTest(t, subRepo, &fakePaymentRepo{}, &recordingEngine{})

	ts := newTestServer(fixedAuthMiddleware(ownerUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPatch, "/api/subscriptions/sub_1/billing-anchor", UpdateBillingAnchorRequest{
		BillingAnchor: 15,
		ProrationMode: domain.ProrationModeNone,
	})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got ProrationDetailsResponse
	decodeJSON(t, rec, &got)
	assert.Equal(t, 1, got.OldBillingAnchor)
	assert.Equal(t, 15, got.NewBillingAnchor)
	require.NotEmpty(t, subRepo.updated, "subscription persisted with new anchor")
}
