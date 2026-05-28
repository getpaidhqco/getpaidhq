package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/service"
)

func newReportHandlerForTest(rep *fakeReportRepo) *ReportHandler {
	svc, err := service.NewReportService(silentLogger{}, rep, noopScheduler{}, &fakeOrgRepo{})
	if err != nil {
		panic(err)
	}
	return NewReportHandler(svc, silentLogger{})
}

func TestReportHandler_GetMRR(t *testing.T) {
	t.Run("happy path returns the recorded series", func(t *testing.T) {
		rep := &fakeReportRepo{mrr: []domain.RecurringRevenue{
			{Total: 1000, Type: "mrr"},
		}}
		h := newReportHandlerForTest(rep)

		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodGet, "/api/reports/revenue/mrr?start_date=2025-01-01&end_date=2025-01-31", nil)

		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	})

	t.Run("invalid start_date format returns bad_request envelope", func(t *testing.T) {
		h := newReportHandlerForTest(&fakeReportRepo{})
		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodGet, "/api/reports/revenue/mrr?start_date=not-a-date&end_date=2025-01-31", nil)

		assertErrorEnvelope(t, rec, http.StatusBadRequest, "bad_request")
	})

	t.Run("invalid end_date format returns bad_request envelope", func(t *testing.T) {
		h := newReportHandlerForTest(&fakeReportRepo{})
		ts := newTestServer(fixedAuthMiddleware(adminUser()))
		h.RegisterRoutes(ts.api())

		rec := doJSON(t, ts, http.MethodGet, "/api/reports/revenue/mrr?start_date=2025-01-01&end_date=bogus", nil)

		assertErrorEnvelope(t, rec, http.StatusBadRequest, "bad_request")
	})
}

func TestReportHandler_OtherEndpoints(t *testing.T) {
	// Compact sanity sweep: every Get path takes the same date-range params
	// and delegates to a different repo method. Verify status 200 for each.
	rep := &fakeReportRepo{}
	h := newReportHandlerForTest(rep)

	ts := newTestServer(fixedAuthMiddleware(adminUser()))
	h.RegisterRoutes(ts.api())

	paths := []string{
		"/api/reports/revenue/arr",
		"/api/reports/active-subscribers",
		"/api/reports/refunds",
		"/api/reports/churn/totals",
		"/api/reports/churn/rates",
	}
	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			rec := doJSON(t, ts, http.MethodGet, p+"?start_date=2025-01-01&end_date=2025-01-31", nil)
			assert.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		})
	}
}
