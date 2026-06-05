package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

type usageHTTPMeterRepo struct {
	port.MeterRepository
	metric domain.BillableMetric
	ok     bool
}

func (r *usageHTTPMeterRepo) FindByCode(_ context.Context, _, _ string) (domain.BillableMetric, error) {
	if !r.ok {
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
	return port.IngestResult{Id: e.Id}, nil
}

func newUsageHandlerForTest(meterOk, customerKnown bool, es *usageHTTPEventStore) *UsageHandler {
	meters := &usageHTTPMeterRepo{ok: meterOk, metric: domain.BillableMetric{OrgId: "org_1", Id: "met_1", Code: "api_calls", Aggregation: domain.AggregationCount}}
	customers := &usageHTTPCustomerRepo{known: customerKnown}
	svc := service.NewUsageService(meters, customers, nil, es, newPubSub(), silentLogger{})
	return NewUsageHandler(svc, silentLogger{})
}

func TestUsageHandler_RecordEvent(t *testing.T) {
	es := &usageHTTPEventStore{}
	h := newUsageHandlerForTest(true, true, es)
	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/usage/events", RecordEventRequest{
		MetricCode: "api_calls", CustomerId: "cus_1",
	})
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got RecordEventResponse
	decodeJSON(t, rec, &got)
	assert.Equal(t, "recorded", got.Status)
	assert.Equal(t, 1, es.ingested)
}

func TestUsageHandler_RecordEvent_MissingMetricCode(t *testing.T) {
	es := &usageHTTPEventStore{}
	h := newUsageHandlerForTest(true, true, es)
	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	// metric_code is validate:"required" → request validation rejects (400).
	rec := doJSON(t, ts, http.MethodPost, "/api/usage/events", RecordEventRequest{CustomerId: "cus_1"})
	assert.Equal(t, http.StatusBadRequest, rec.Code, "body=%s", rec.Body.String())
	assert.Zero(t, es.ingested)
}

func TestUsageHandler_RecordEvent_UnknownCustomer(t *testing.T) {
	es := &usageHTTPEventStore{}
	h := newUsageHandlerForTest(true, false, es) // customer not known
	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/usage/events", RecordEventRequest{MetricCode: "api_calls", CustomerId: "ghost"})
	assert.Equal(t, http.StatusNotFound, rec.Code, "body=%s", rec.Body.String())
	assert.Zero(t, es.ingested)
}
