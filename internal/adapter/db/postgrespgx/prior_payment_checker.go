package postgrespgx

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// PriorPaymentChecker backs port.PriorPaymentChecker for the FirstTimeTransaction
// coupon restriction.
//
// The payments table has no direct customer_id column; customer linkage runs
// through orders (for one-time payments) and subscriptions (for recurring
// payments). We check for any payment whose org matches, whose status is
// succeeded, and that is linked to either an order or a subscription owned by
// the given customer.
type PriorPaymentChecker struct {
	pool *pgxpool.Pool
}

func NewPriorPaymentChecker(pool *pgxpool.Pool) port.PriorPaymentChecker {
	return &PriorPaymentChecker{pool: pool}
}

func (r *PriorPaymentChecker) HasPriorSuccessfulPayment(ctx context.Context, orgId, customerId string) (bool, error) {
	q := dbFromCtx(ctx, r.pool)
	var exists bool
	err := q.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM payments
			WHERE org_id = $1 AND status = $2
			  AND (order_id IN (SELECT id FROM orders WHERE org_id = $1 AND customer_id = $3)
			       OR subscription_id IN (SELECT id FROM subscriptions WHERE org_id = $1 AND customer_id = $3))
		)`,
		orgId, string(domain.PaymentStatusSucceeded), customerId).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
