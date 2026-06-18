package postgrespgx

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type SubscriptionRepo struct {
	pool *pgxpool.Pool
}

func NewSubscriptionRepo(pool *pgxpool.Pool) port.SubscriptionRepository {
	return &SubscriptionRepo{pool: pool}
}

func (r *SubscriptionRepo) FindById(ctx context.Context, orgId string, id string) (domain.Subscription, error) {
	q := dbFromCtx(ctx, r.pool)
	var row subscriptionRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+subscriptionColumns+` FROM subscriptions WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.Subscription{}, translateErr(err)
	}
	return row.toDomain(), nil
}

// FindByIdForUpdate is the row-locking variant of FindById. MUST be called
// inside a transaction (TxManager.RunInTx); dbFromCtx returns the ambient tx so
// the SELECT ... FOR UPDATE lock is held for the rest of the transaction.
// Outside a tx the lock is acquired and immediately released, defeating it.
func (r *SubscriptionRepo) FindByIdForUpdate(ctx context.Context, orgId string, id string) (domain.Subscription, error) {
	q := dbFromCtx(ctx, r.pool)
	var row subscriptionRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+subscriptionColumns+` FROM subscriptions WHERE org_id = $1 AND id = $2 FOR UPDATE`, orgId, id)); err != nil {
		return domain.Subscription{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *SubscriptionRepo) Create(ctx context.Context, entity domain.Subscription) (domain.Subscription, error) {
	row := subscriptionRowFromDomain(entity)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO subscriptions (`+subscriptionColumns+`) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
			$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31)`,
		row.OrgId, row.Id, row.PspId, row.OrderId, row.CustomerId, row.Status, row.PaymentMethodId,
		row.StartDate, row.EndDate, row.BillingInterval, row.BillingIntervalQty, row.Cycles, row.BillingAnchor,
		row.TrialInterval, row.TrialIntervalQty, row.TrialEndsAt, row.CancelAt, row.EndsAt, row.LastCharge, row.RenewsAt,
		row.CurrentPeriodStart, row.CurrentPeriodEnd, row.Retries, row.NextRetryAt, row.Currency, row.Metadata,
		row.CyclesProcessed, row.TotalRevenue, row.CancelledAt, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.Subscription{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *SubscriptionRepo) Update(ctx context.Context, entity domain.Subscription) (domain.Subscription, error) {
	row := subscriptionRowFromDomain(entity)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE subscriptions SET
			psp_id=$3, order_id=$4, customer_id=$5, status=$6, payment_method_id=$7,
			start_date=$8, end_date=$9, billing_interval=$10, billing_interval_qty=$11, cycles=$12,
			billing_anchor=$13, trial_interval=$14, trial_interval_qty=$15, trial_ends_at=$16,
			cancel_at=$17, ends_at=$18, last_charge=$19, renews_at=$20, current_period_start=$21,
			current_period_end=$22, retries=$23, next_retry=$24, currency=$25, metadata=$26,
			cycles_processed=$27, total_revenue=$28, cancelled_at=$29, updated_at=$30
		 WHERE org_id=$1 AND id=$2`,
		row.OrgId, row.Id, row.PspId, row.OrderId, row.CustomerId, row.Status, row.PaymentMethodId,
		row.StartDate, row.EndDate, row.BillingInterval, row.BillingIntervalQty, row.Cycles,
		row.BillingAnchor, row.TrialInterval, row.TrialIntervalQty, row.TrialEndsAt,
		row.CancelAt, row.EndsAt, row.LastCharge, row.RenewsAt, row.CurrentPeriodStart,
		row.CurrentPeriodEnd, row.Retries, row.NextRetryAt, row.Currency, row.Metadata,
		row.CyclesProcessed, row.TotalRevenue, row.CancelledAt, row.UpdatedAt)
	if err != nil {
		return domain.Subscription{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *SubscriptionRepo) FindByOrderId(ctx context.Context, orgId string, orderId string) ([]domain.Subscription, error) {
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT `+subscriptionColumns+` FROM subscriptions WHERE org_id = $1 AND order_id = $2`, orgId, orderId)
	if err != nil {
		return nil, err
	}
	return r.collect(rows)
}

func (r *SubscriptionRepo) FindActiveMeteredForMeter(ctx context.Context, orgId, customerId, billableMetricId string) ([]domain.Subscription, error) {
	q := dbFromCtx(ctx, r.pool)
	// A subscription is "metered for M" when it OWNS an order line carrying a
	// metered price for meter M (lines are stamped with subscription_id at order
	// creation). DISTINCT collapses subscriptions that own several metered lines
	// for the same meter.
	rows, err := q.Query(ctx,
		`SELECT DISTINCT `+subscriptionSelectQualified+`
		 FROM subscriptions
		 JOIN order_items oi ON oi.org_id = subscriptions.org_id AND oi.subscription_id = subscriptions.id
		 JOIN prices p ON p.org_id = oi.org_id AND p.id = oi.price_id
		 WHERE subscriptions.org_id = $1 AND subscriptions.customer_id = $2
		   AND p.billable_metric_id = $3
		   AND subscriptions.status IN ($4, $5, $6)
		 ORDER BY subscriptions.start_date ASC, subscriptions.created_at ASC`,
		orgId, customerId, billableMetricId,
		string(domain.SubscriptionStatusActive),
		string(domain.SubscriptionStatusTrial),
		string(domain.SubscriptionStatusPastDue))
	if err != nil {
		return nil, err
	}
	return r.collect(rows)
}

func (r *SubscriptionRepo) Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Subscription, int, error) {
	q := dbFromCtx(ctx, r.pool)
	var count int64
	if err := q.QueryRow(ctx, `SELECT count(*) FROM subscriptions WHERE org_id = $1`, orgId).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx, `SELECT `+subscriptionColumns+` FROM subscriptions WHERE org_id = $1`+paginationClause(p), orgId)
	if err != nil {
		return nil, 0, err
	}
	out, err := r.collect(rows)
	if err != nil {
		return nil, 0, err
	}
	return out, int(count), nil
}

// FindDueForBilling selects subscriptions due for a charge now. Keep the status/
// date rule below in sync with domain.Subscription.IsDueForBilling — that Go
// method is the per-subscription mirror of this SQL (used by the Hatchet
// activation spawn), and the two must agree on what "due" means.
//
// Unset date columns are NULL (nullTime maps zero time → NULL), and `col <= now`
// is already false for NULL, so unset rows are auto-excluded — no epoch guards.
func (r *SubscriptionRepo) FindDueForBilling(ctx context.Context, orgId string, now time.Time) ([]domain.Subscription, error) {
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT `+subscriptionColumns+` FROM subscriptions
		 WHERE org_id = $1 AND (
			(status = $2 AND renews_at <= $5)
			OR (status = $3 AND next_retry <= $5)
			OR (status = $4 AND trial_ends_at <= $5)
		 )`,
		orgId,
		string(domain.SubscriptionStatusActive),
		string(domain.SubscriptionStatusPastDue),
		string(domain.SubscriptionStatusTrial),
		now)
	if err != nil {
		return nil, err
	}
	return r.collect(rows)
}

func (r *SubscriptionRepo) FindUpcomingRenewals(ctx context.Context, orgId string, now time.Time, within time.Duration) ([]domain.Subscription, error) {
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT `+subscriptionColumns+` FROM subscriptions
		 WHERE org_id = $1 AND status = $2 AND renews_at > $3 AND renews_at <= $4`,
		orgId, string(domain.SubscriptionStatusActive), now, now.Add(within))
	if err != nil {
		return nil, err
	}
	return r.collect(rows)
}

// collect drains rows into domain subscriptions, closing rows.
func (r *SubscriptionRepo) collect(rows pgx.Rows) ([]domain.Subscription, error) {
	defer rows.Close()
	var out []domain.Subscription
	for rows.Next() {
		var row subscriptionRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row.toDomain())
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
