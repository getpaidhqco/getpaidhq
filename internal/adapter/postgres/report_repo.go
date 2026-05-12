package postgres

import (
	"context"
	"time"

	"gorm.io/gorm"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type ReportRepo struct {
	db *gorm.DB
}

func NewReportRepo(db *gorm.DB) port.ReportRepository {
	return &ReportRepo{db: db}
}

func (r *ReportRepo) GetMRR(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]domain.RecurringRevenue, error) {
	var results []domain.RecurringRevenue
	err := r.db.WithContext(ctx).Raw(`
		SELECT period, total, count, growth_mom, type
		FROM daily_metrics
		WHERE org_id = ? AND type = 'mrr' AND period BETWEEN ? AND ?
		ORDER BY period ASC
	`, orgId, startDate, endDate).Scan(&results).Error
	return results, err
}

func (r *ReportRepo) GetARR(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]domain.RecurringRevenue, error) {
	var results []domain.RecurringRevenue
	err := r.db.WithContext(ctx).Raw(`
		SELECT period, total, count, growth_mom, type
		FROM daily_metrics
		WHERE org_id = ? AND type = 'arr' AND period BETWEEN ? AND ?
		ORDER BY period ASC
	`, orgId, startDate, endDate).Scan(&results).Error
	return results, err
}

func (r *ReportRepo) GetActiveSubscribers(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]domain.RecurringRevenue, error) {
	var results []domain.RecurringRevenue
	err := r.db.WithContext(ctx).Raw(`
		SELECT period, total, count, growth_mom, type
		FROM daily_metrics
		WHERE org_id = ? AND type = 'active_subscribers' AND period BETWEEN ? AND ?
		ORDER BY period ASC
	`, orgId, startDate, endDate).Scan(&results).Error
	return results, err
}

func (r *ReportRepo) GetRefundTotals(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]domain.RecurringRevenue, error) {
	var results []domain.RecurringRevenue
	err := r.db.WithContext(ctx).Raw(`
		SELECT period, total, count, growth_mom, type
		FROM daily_metrics
		WHERE org_id = ? AND type = 'refunds' AND period BETWEEN ? AND ?
		ORDER BY period ASC
	`, orgId, startDate, endDate).Scan(&results).Error
	return results, err
}

func (r *ReportRepo) GetCustomerChurnTotals(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]domain.RecurringRevenue, error) {
	var results []domain.RecurringRevenue
	err := r.db.WithContext(ctx).Raw(`
		SELECT period, total, count, growth_mom, type
		FROM daily_metrics
		WHERE org_id = ? AND type = 'customer_churn' AND period BETWEEN ? AND ?
		ORDER BY period ASC
	`, orgId, startDate, endDate).Scan(&results).Error
	return results, err
}

func (r *ReportRepo) GetCustomerChurnRates(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]domain.RecurringRevenue, error) {
	var results []domain.RecurringRevenue
	err := r.db.WithContext(ctx).Raw(`
		SELECT period, total, count, growth_mom, type
		FROM daily_metrics
		WHERE org_id = ? AND type = 'customer_churn_rate' AND period BETWEEN ? AND ?
		ORDER BY period ASC
	`, orgId, startDate, endDate).Scan(&results).Error
	return results, err
}

func (r *ReportRepo) UpsertSubscription(ctx context.Context, entity domain.Subscription) error {
	return r.db.WithContext(ctx).Raw(`
		INSERT INTO report_subscriptions (org_id, id, status, currency, amount, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (org_id, id) DO UPDATE SET
			status = EXCLUDED.status,
			currency = EXCLUDED.currency,
			amount = EXCLUDED.amount,
			updated_at = EXCLUDED.updated_at
	`, entity.OrgId, entity.Id, entity.Status, entity.Currency, entity.Amount, entity.CreatedAt, entity.UpdatedAt).Error
}

func (r *ReportRepo) UpsertPayment(ctx context.Context, entity domain.Payment) error {
	return r.db.WithContext(ctx).Raw(`
		INSERT INTO report_payments (org_id, id, status, currency, amount, completed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (org_id, id) DO UPDATE SET
			status = EXCLUDED.status,
			currency = EXCLUDED.currency,
			amount = EXCLUDED.amount,
			completed_at = EXCLUDED.completed_at,
			updated_at = EXCLUDED.updated_at
	`, entity.OrgId, entity.Id, entity.Status, entity.Currency, entity.Amount, entity.CompletedAt, entity.CreatedAt, entity.UpdatedAt).Error
}

func (r *ReportRepo) UpsertCustomer(ctx context.Context, entity domain.Customer) error {
	return r.db.WithContext(ctx).Raw(`
		INSERT INTO report_customers (org_id, id, email, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (org_id, id) DO UPDATE SET
			email = EXCLUDED.email,
			updated_at = EXCLUDED.updated_at
	`, entity.OrgId, entity.Id, entity.Email, entity.CreatedAt, entity.UpdatedAt).Error
}

func (r *ReportRepo) UpsertRefund(ctx context.Context, entity domain.Refund) error {
	return r.db.WithContext(ctx).Raw(`
		INSERT INTO report_refunds (org_id, id, payment_id, amount, currency, refunded_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (org_id, id) DO UPDATE SET
			amount = EXCLUDED.amount,
			currency = EXCLUDED.currency,
			refunded_at = EXCLUDED.refunded_at,
			updated_at = EXCLUDED.updated_at
	`, entity.OrgId, entity.Id, entity.PaymentId, entity.Amount, entity.Currency, entity.RefundedAt, entity.CreatedAt, entity.UpdatedAt).Error
}

func (r *ReportRepo) UpsertCustomerCohort(ctx context.Context, entity domain.CustomerCohort) error {
	return r.db.WithContext(ctx).Raw(`
		INSERT INTO report_customer_cohorts (org_id, customer_id, cohort_id, cohort_value, joined_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (org_id, customer_id, cohort_id) DO UPDATE SET
			cohort_value = EXCLUDED.cohort_value,
			updated_at = EXCLUDED.updated_at
	`, entity.OrgId, entity.CustomerId, entity.CohortId, entity.CohortValue, entity.JoinedAt, entity.CreatedAt, entity.UpdatedAt).Error
}

func (r *ReportRepo) StoreDailyMetrics(ctx context.Context, input port.ProcessDailyMetricsInput) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO daily_metrics (org_id, period, type, total, count)
		SELECT ?, ?, type, COALESCE(SUM(total), 0), COALESCE(COUNT(*), 0)
		FROM (
			SELECT 'mrr' as type, amount as total FROM report_subscriptions WHERE org_id = ? AND status = 'active'
			UNION ALL
			SELECT 'active_subscribers' as type, 1 as total FROM report_subscriptions WHERE org_id = ? AND status = 'active'
		) metrics
		GROUP BY type
		ON CONFLICT (org_id, period, type) DO UPDATE SET
			total = EXCLUDED.total,
			count = EXCLUDED.count
	`, input.OrgId, input.Date, input.OrgId, input.OrgId).Error
}

func (r *ReportRepo) ProcessDailyMetrics(ctx context.Context, d time.Time) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO daily_metrics (org_id, period, type, total, count)
		SELECT org_id, ?, 'mrr', COALESCE(SUM(amount), 0), COUNT(*)
		FROM report_subscriptions
		WHERE status = 'active'
		GROUP BY org_id
		ON CONFLICT (org_id, period, type) DO UPDATE SET
			total = EXCLUDED.total,
			count = EXCLUDED.count
	`, d).Error
}
