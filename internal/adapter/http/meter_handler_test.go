package handler

import (
	"context"
	"getpaidhq/internal/lib/errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

// fakeMeterRepo embeds the port interface; unimplemented methods nil-panic if hit.
type fakeMeterRepo struct {
	port.MeterRepository
	created    domain.BillableMetric
	createN    int
	listResult []domain.BillableMetric
}

func (r *fakeMeterRepo) Create(_ context.Context, m domain.BillableMetric) (domain.BillableMetric, error) {
	r.created = m
	r.createN++
	return m, nil
}

func (r *fakeMeterRepo) Find(_ context.Context, _ string, _ domain.Pagination) ([]domain.BillableMetric, int, error) {
	return r.listResult, len(r.listResult), nil
}

func newMeterHandlerForTest(t *testing.T, repo *fakeMeterRepo) *MeterHandler {
	t.Helper()
	svc := service.NewMeterService(repo, newPubSub(), silentLogger{})
	return NewMeterHandler(svc, silentLogger{}, newRealAuthz(t))
}

func TestMeterHandler_AuthzGuards(t *testing.T) {
	repo := &fakeMeterRepo{}
	h := newMeterHandlerForTest(t, repo)
	ts := newTestServer(fixedAuthMiddleware(supportUser()))
	h.RegisterRoutes(ts.api())

	tests := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"create", http.MethodPost, "/api/meters", CreateMeterRequest{Code: "c", Name: "n", Aggregation: domain.AggregationCount}},
		{"list", http.MethodGet, "/api/meters", nil},
		{"get", http.MethodGet, "/api/meters/met_1", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doJSON(t, ts, tt.method, tt.path, tt.body)
			assertErrorEnvelope(t, rec, http.StatusForbidden, string(errors.ForbiddenError))
		})
	}
	assert.Zero(t, repo.createN, "no writes past the authz guard")
}

func TestMeterHandler_Create(t *testing.T) {
	repo := &fakeMeterRepo{}
	h := newMeterHandlerForTest(t, repo)
	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/meters", CreateMeterRequest{
		Code: "api_calls", Name: "API calls", Aggregation: domain.AggregationSum, FieldName: "calls",
	})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got MeterResponse
	decodeJSON(t, rec, &got)
	assert.Equal(t, "api_calls", got.Code)
	assert.Equal(t, domain.AggregationSum, got.Aggregation)
	assert.Equal(t, 1, repo.createN)
	assert.NotEmpty(t, repo.created.Id, "service assigns an id")
}

func TestMeterHandler_Create_ValidationRejected(t *testing.T) {
	repo := &fakeMeterRepo{}
	h := newMeterHandlerForTest(t, repo)
	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	// Missing aggregation → request validation (400), nothing stored.
	rec := doJSON(t, ts, http.MethodPost, "/api/meters", CreateMeterRequest{Code: "c", Name: "n"})
	assert.Equal(t, http.StatusBadRequest, rec.Code, "body=%s", rec.Body.String())
	assert.Zero(t, repo.createN)
}

func TestMeterHandler_Create_WithFiltersAndGroup(t *testing.T) {
	repo := &fakeMeterRepo{}
	h := newMeterHandlerForTest(t, repo)
	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodPost, "/api/meters", CreateMeterRequest{
		Code: "messages", Name: "Messages", Aggregation: domain.AggregationCount,
		Filters: []MeterFilterRequest{{Field: "type", Values: []string{"SMS", "MMS"}}},
		GroupBy: []string{"project"},
	})

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	// Persisted onto the metric.
	require.Len(t, repo.created.Filters, 1)
	assert.Equal(t, "type", repo.created.Filters[0].Field)
	assert.Equal(t, []string{"SMS", "MMS"}, repo.created.Filters[0].Values)
	assert.Equal(t, []string{"project"}, repo.created.GroupBy)
	// Echoed back in the response.
	var got MeterResponse
	decodeJSON(t, rec, &got)
	require.Len(t, got.Filters, 1)
	assert.Equal(t, "type", got.Filters[0].Field)
	assert.Equal(t, []string{"project"}, got.GroupBy)
}

func TestMeterHandler_Create_RejectsBadFilterAndGroup(t *testing.T) {
	cases := []struct {
		name string
		body CreateMeterRequest
	}{
		{"filter with no values", CreateMeterRequest{
			Code: "m", Name: "n", Aggregation: domain.AggregationCount,
			Filters: []MeterFilterRequest{{Field: "type"}},
		}},
		{"more than one group dimension", CreateMeterRequest{
			Code: "m", Name: "n", Aggregation: domain.AggregationCount,
			GroupBy: []string{"project", "cloud"},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeMeterRepo{}
			h := newMeterHandlerForTest(t, repo)
			ts := newTestServer(fixedAuthMiddleware(adminUser()))
			h.RegisterRoutes(ts.api())
			rec := doJSON(t, ts, http.MethodPost, "/api/meters", tc.body)
			assert.Equal(t, http.StatusBadRequest, rec.Code, "body=%s", rec.Body.String())
			assert.Zero(t, repo.createN, "nothing stored on a rejected meter")
		})
	}
}

func TestMeterHandler_List(t *testing.T) {
	repo := &fakeMeterRepo{listResult: []domain.BillableMetric{
		{Id: "met_1", Code: "a"}, {Id: "met_2", Code: "b"},
	}}
	h := newMeterHandlerForTest(t, repo)
	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	rec := doJSON(t, ts, http.MethodGet, "/api/meters?page=0&limit=10", nil)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var got ListResponse
	decodeJSON(t, rec, &got)
	assert.Equal(t, 2, got.Meta.Total)
}
