package postgres

import (
	"context"
	"sync"
	"time"

	"gorm.io/gorm"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// ReportRepo is a deliberate tombstone.
//
// The previous implementation read and wrote a reporting schema that does
// not match schemas/reporting/schema.prisma:
//
//   - Upserts targeted report_subscriptions / report_payments /
//     report_customers / report_refunds / report_customer_cohorts. The real
//     tables are subscriptions / payments / customers / refunds /
//     customer_cohorts (no report_ prefix).
//   - Get* queries selected `period, total, count, growth_mom, type` from
//     daily_metrics as if it were polymorphic. The real daily_metrics is
//     keyed (org_id, date) with one column per metric (mrr, arr,
//     customer_count, churn_count, churn_total, refund_count, refund_total,
//     successful_payments, failed_payments, …).
//
// The reporting layer is no longer wired in NewApp; the HTTP routes have
// been removed and the daily cron / NATS event bridge are gone. This file
// is kept as a marker for whoever revives reports: rewrite each method
// against the actual Prisma schema, restore a service + handler, and
// re-register the routes in internal/config/server.go.
//
// Every method is a no-op that logs "report repo not implemented" once per
// method per process — if anything reaches these, we want to see it.
type ReportRepo struct {
	db     *gorm.DB
	logger port.Logger

	mu    sync.Mutex
	warns map[string]*sync.Once
}

// NewReportRepo builds the stub. The *gorm.DB is retained so a future
// implementation does not have to thread it back through app wiring.
func NewReportRepo(db *gorm.DB, logger port.Logger) *ReportRepo {
	return &ReportRepo{
		db:     db,
		logger: logger,
		warns:  map[string]*sync.Once{},
	}
}

func (r *ReportRepo) warnOnce(method string) {
	r.mu.Lock()
	once, ok := r.warns[method]
	if !ok {
		once = &sync.Once{}
		r.warns[method] = once
	}
	r.mu.Unlock()
	once.Do(func() {
		r.logger.Warn("report repo not implemented", "method", method)
	})
}

func (r *ReportRepo) GetMRR(_ context.Context, _ string, _, _ time.Time) ([]domain.RecurringRevenue, error) {
	r.warnOnce("GetMRR")
	return nil, nil
}

func (r *ReportRepo) GetARR(_ context.Context, _ string, _, _ time.Time) ([]domain.RecurringRevenue, error) {
	r.warnOnce("GetARR")
	return nil, nil
}

func (r *ReportRepo) GetActiveSubscribers(_ context.Context, _ string, _, _ time.Time) ([]domain.RecurringRevenue, error) {
	r.warnOnce("GetActiveSubscribers")
	return nil, nil
}

func (r *ReportRepo) GetRefundTotals(_ context.Context, _ string, _, _ time.Time) ([]domain.RecurringRevenue, error) {
	r.warnOnce("GetRefundTotals")
	return nil, nil
}

func (r *ReportRepo) GetCustomerChurnTotals(_ context.Context, _ string, _, _ time.Time) ([]domain.RecurringRevenue, error) {
	r.warnOnce("GetCustomerChurnTotals")
	return nil, nil
}

func (r *ReportRepo) GetCustomerChurnRates(_ context.Context, _ string, _, _ time.Time) ([]domain.RecurringRevenue, error) {
	r.warnOnce("GetCustomerChurnRates")
	return nil, nil
}
