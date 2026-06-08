package handler

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

type usageHTTPMeterRepo struct {
	port.MeterRepository
	metric domain.BillableMetric
}

// FindByCode returns the metric only for its own code; any other code is unknown,
// so a batch can mix recorded and rejected events.
func (r *usageHTTPMeterRepo) FindByCode(_ context.Context, _, code string) (domain.BillableMetric, error) {
	if code != r.metric.Code {
		return domain.BillableMetric{}, port.ErrNotFound
	}
	return r.metric, nil
}

type usageHTTPCustomerRepo struct {
	port.CustomerRepository
	known bool
}

func (r *usageHTTPCustomerRepo) FindById(_ context.Context, _, _ string) (domain.Customer, error) {
	if !r.known {
		return domain.Customer{}, port.ErrNotFound
	}
	return domain.Customer{OrgId: "org_1", Id: "cus_1"}, nil
}

type usageHTTPEventStore struct {
	port.EventStore
	ingested int
}

func (s *usageHTTPEventStore) Ingest(_ context.Context, e domain.MeterEvent) (port.IngestResult, error) {
	s.ingested++
	return port.IngestResult{Id: e.Id, Status: port.IngestRecorded}, nil
}

type usageHandlerDeps struct {
	customerKnown bool
	es            *usageHTTPEventStore
	subs          *fakeSubRepo
	orders        *fakeOrderRepo
	prices        *fakePriceRepo
}

func newUsageHandler(t *testing.T, d usageHandlerDeps) *UsageHandler {
	t.Helper()
	meters := &usageHTTPMeterRepo{metric: domain.BillableMetric{OrgId: "org_1", Id: "met_1", Code: "api_calls", Aggregation: domain.AggregationCount}}
	customers := &usageHTTPCustomerRepo{known: d.customerKnown}
	if d.subs == nil {
		d.subs = &fakeSubRepo{}
	}
	if d.orders == nil {
		d.orders = &fakeOrderRepo{}
	}
	if d.prices == nil {
		d.prices = &fakePriceRepo{}
	}
	svc := service.NewUsageService(meters, customers, d.subs, d.orders, d.prices, d.es, d.es, newPubSub(), silentLogger{})
	return NewUsageHandler(svc, silentLogger{}, newRealAuthz(t))
}

func TestUsageHandler_Ingest_Batch(t *testing.T) {
	es := &usageHTTPEventStore{}
	h := newUsageHandler(t, usageHandlerDeps{customerKnown: true, es: es})
	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/usage/ingest", IngestEventsRequest{
		Events: []RecordEventRequest{
			{MetricCode: "api_calls", CustomerId: "cus_1"},
			{MetricCode: "bogus", CustomerId: "cus_1"}, // unknown metric → rejected, others unaffected
		},
	})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got IngestEventsResponse
	decodeJSON(t, rec, &got)
	require.Len(t, got.Results, 2)
	assert.Equal(t, 0, got.Results[0].Index)
	assert.Equal(t, "recorded", got.Results[0].Status)
	assert.Equal(t, 1, got.Results[1].Index)
	assert.Equal(t, "rejected", got.Results[1].Status)
	assert.NotEmpty(t, got.Results[1].Error)
	assert.Equal(t, 1, es.ingested, "only the valid event is written")
}

func TestUsageHandler_Ingest_RejectsEmptyAndOversize(t *testing.T) {
	h := newUsageHandler(t, usageHandlerDeps{customerKnown: true, es: &usageHTTPEventStore{}})
	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	t.Run("empty batch", func(t *testing.T) {
		rec := doJSON(t, ts, http.MethodPost, "/api/usage/ingest", IngestEventsRequest{Events: []RecordEventRequest{}})
		assert.Equal(t, http.StatusBadRequest, rec.Code, "body=%s", rec.Body.String())
	})

	t.Run("over the 500 cap", func(t *testing.T) {
		big := make([]RecordEventRequest, 501)
		for i := range big {
			big[i] = RecordEventRequest{MetricCode: "api_calls", CustomerId: "cus_1"}
		}
		rec := doJSON(t, ts, http.MethodPost, "/api/usage/ingest", IngestEventsRequest{Events: big})
		assert.Equal(t, http.StatusBadRequest, rec.Code, "body=%s", rec.Body.String())
	})
}

func TestUsageHandler_AuthzGuards(t *testing.T) {
	es := &usageHTTPEventStore{}
	h := newUsageHandler(t, usageHandlerDeps{customerKnown: true, es: es})
	ts := newTestServer(fixedAuthMiddleware(supportUser())) // no permits
	h.RegisterRoutes(ts.api())

	ingest := doJSON(t, ts, http.MethodPost, "/api/usage/ingest", IngestEventsRequest{
		Events: []RecordEventRequest{{MetricCode: "api_calls", CustomerId: "cus_1"}},
	})
	read := doJSON(t, ts, http.MethodGet, "/api/subscriptions/sub_1/usage", nil)

	assert.Equal(t, http.StatusForbidden, ingest.Code)
	assert.Equal(t, http.StatusForbidden, read.Code)
	assert.Zero(t, es.ingested, "authz must reject before the service runs")
}

func TestUsageHandler_SubscriptionUsage_NotFound(t *testing.T) {
	subs := &fakeSubRepo{byIdErr: port.ErrNotFound}
	h := newUsageHandler(t, usageHandlerDeps{customerKnown: true, es: &usageHTTPEventStore{}, subs: subs})
	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/subscriptions/ghost/usage", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code, "body=%s", rec.Body.String())
}

// A non-metered subscription returns an empty meters list — exercises the read
// endpoint end-to-end without the aggregation collaborators.
func TestUsageHandler_SubscriptionUsage_NonMetered(t *testing.T) {
	now := time.Now().UTC()
	subs := &fakeSubRepo{byId: domain.Subscription{
		OrgId: "org_1", Id: "sub_1",
		CurrentPeriodStart: now, CurrentPeriodEnd: now.AddDate(0, 1, 0),
	}}
	// No order lines stamped with this sub → no metered usage (empty meters).
	prices := &fakePriceRepo{byId: domain.Price{OrgId: "org_1", Id: "price_1"}}
	deps := usageHandlerDeps{customerKnown: true, es: &usageHTTPEventStore{}, subs: subs, prices: prices}

	t.Run("admin reads", func(t *testing.T) {
		h := newUsageHandler(t, deps)
		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodGet, "/api/subscriptions/sub_1/usage", nil)
		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		var got SubscriptionUsageResponse
		decodeJSON(t, rec, &got)
		assert.Equal(t, "sub_1", got.SubscriptionId)
		assert.Empty(t, got.Meters)
	})

	t.Run("member can read but cannot ingest", func(t *testing.T) {
		h := newUsageHandler(t, deps)
		ts := newTestServer(fixedAuthMiddleware(memberUser()))
		h.RegisterRoutes(ts.api())

		read := doJSON(t, ts, http.MethodGet, "/api/subscriptions/sub_1/usage", nil)
		assert.Equal(t, http.StatusOK, read.Code, "member may read usage; body=%s", read.Body.String())

		ingest := doJSON(t, ts, http.MethodPost, "/api/usage/ingest", IngestEventsRequest{
			Events: []RecordEventRequest{{MetricCode: "api_calls", CustomerId: "cus_1"}},
		})
		assert.Equal(t, http.StatusForbidden, ingest.Code, "member may not ingest")
	})
}
