package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// fakeReportRepo serves configurable analytics results and records the daily
// metrics processing call.
type fakeReportRepo struct {
	port.ReportRepository
	mrr            []domain.RecurringRevenue
	activeSubs     []domain.RecurringRevenue
	refunds        []domain.RecurringRevenue
	churnTotals    []domain.RecurringRevenue
	churnRates     []domain.RecurringRevenue
	err            error
	processedDates []time.Time
	processErr     error
}

func (r *fakeReportRepo) GetMRR(_ context.Context, _ string, _, _ time.Time) ([]domain.RecurringRevenue, error) {
	return r.mrr, r.err
}
func (r *fakeReportRepo) GetActiveSubscribers(_ context.Context, _ string, _, _ time.Time) ([]domain.RecurringRevenue, error) {
	return r.activeSubs, r.err
}
func (r *fakeReportRepo) GetRefundTotals(_ context.Context, _ string, _, _ time.Time) ([]domain.RecurringRevenue, error) {
	return r.refunds, r.err
}
func (r *fakeReportRepo) GetCustomerChurnTotals(_ context.Context, _ string, _, _ time.Time) ([]domain.RecurringRevenue, error) {
	return r.churnTotals, r.err
}
func (r *fakeReportRepo) GetCustomerChurnRates(_ context.Context, _ string, _, _ time.Time) ([]domain.RecurringRevenue, error) {
	return r.churnRates, r.err
}
func (r *fakeReportRepo) ProcessDailyMetrics(_ context.Context, d time.Time) error {
	if r.processErr != nil {
		return r.processErr
	}
	r.processedDates = append(r.processedDates, d)
	return nil
}

func newReportService(repo port.ReportRepository) *ReportService {
	svc, err := NewReportService(silentLogger{}, repo, noopScheduler{}, nil)
	if err != nil {
		panic(err)
	}
	return svc
}

func TestReportService_Analytics(t *testing.T) {
	ctx := context.Background()
	from, to := time.Now().AddDate(0, -1, 0), time.Now()

	t.Run("MRR and ARR both proxy GetMRR", func(t *testing.T) {
		repo := &fakeReportRepo{mrr: []domain.RecurringRevenue{{}, {}}}
		svc := newReportService(repo)

		mrr, err := svc.GetMonthlyRecurringRevenue(ctx, "org_1", from, to)
		require.NoError(t, err)
		assert.Len(t, mrr, 2)

		arr, err := svc.GetAnnualRecurringRevenue(ctx, "org_1", from, to)
		require.NoError(t, err)
		assert.Len(t, arr, 2)
	})

	t.Run("active subscribers", func(t *testing.T) {
		repo := &fakeReportRepo{activeSubs: []domain.RecurringRevenue{{}}}
		svc := newReportService(repo)
		got, err := svc.GetActiveSubscribers(ctx, "org_1", from, to)
		require.NoError(t, err)
		assert.Len(t, got, 1)
	})

	t.Run("refund totals", func(t *testing.T) {
		repo := &fakeReportRepo{refunds: []domain.RecurringRevenue{{}, {}, {}}}
		svc := newReportService(repo)
		got, err := svc.GetRefundTotals(ctx, "org_1", from, to)
		require.NoError(t, err)
		assert.Len(t, got, 3)
	})

	t.Run("churn totals and rates", func(t *testing.T) {
		repo := &fakeReportRepo{churnTotals: []domain.RecurringRevenue{{}}, churnRates: []domain.RecurringRevenue{{}, {}}}
		svc := newReportService(repo)

		totals, err := svc.GetCustomerChurnTotals(ctx, "org_1", from, to)
		require.NoError(t, err)
		assert.Len(t, totals, 1)

		rates, err := svc.GetCustomerChurnRates(ctx, "org_1", from, to)
		require.NoError(t, err)
		assert.Len(t, rates, 2)
	})

	t.Run("repo error is surfaced", func(t *testing.T) {
		repo := &fakeReportRepo{err: errors.New("query failed")}
		svc := newReportService(repo)

		_, err := svc.GetMonthlyRecurringRevenue(ctx, "org_1", from, to)
		require.Error(t, err)
	})
}

func TestReportService_StoreDailyMetrics(t *testing.T) {
	t.Run("processes yesterday's metrics", func(t *testing.T) {
		repo := &fakeReportRepo{}
		svc := newReportService(repo)

		svc.StoreDailyMetrics()

		require.Len(t, repo.processedDates, 1)
		assert.WithinDuration(t, time.Now().AddDate(0, 0, -1), repo.processedDates[0], time.Minute)
	})

	t.Run("processing failure is swallowed (cron task)", func(t *testing.T) {
		repo := &fakeReportRepo{processErr: errors.New("db down")}
		svc := newReportService(repo)

		// Should not panic; the task logs and returns.
		svc.StoreDailyMetrics()
		assert.Empty(t, repo.processedDates)
	})
}
