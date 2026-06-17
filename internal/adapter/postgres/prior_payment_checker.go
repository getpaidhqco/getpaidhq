package postgres

import (
	"context"

	"gorm.io/gorm"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// PriorPaymentChecker backs port.PriorPaymentChecker for the FirstTimeTransaction
// coupon restriction.
//
// The payments table has no direct customer_id column; customer linkage runs
// through orders (for one-time payments) and subscriptions (for recurring
// payments). We count payments whose org matches and that are linked to either
// an order or a subscription owned by the given customer.
type PriorPaymentChecker struct {
	db *gorm.DB
}

func NewPriorPaymentChecker(db *gorm.DB) port.PriorPaymentChecker {
	return &PriorPaymentChecker{db: db}
}

func (r *PriorPaymentChecker) HasPriorSuccessfulPayment(ctx context.Context, orgId, customerId string) (bool, error) {
	var n int64
	err := dbFromCtx(ctx, r.db).Model(&paymentRow{}).
		Scopes(OrgScope(orgId)).
		Where("status = ?", domain.PaymentStatusSucceeded).
		Where(
			`(order_id IN (SELECT id FROM orders WHERE org_id = ? AND customer_id = ?)
			  OR subscription_id IN (SELECT id FROM subscriptions WHERE org_id = ? AND customer_id = ?))`,
			orgId, customerId, orgId, customerId,
		).
		Count(&n).Error
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
