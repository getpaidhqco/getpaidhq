package service

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// activeDiscountRepo serves a fixed set of active discounts for a subscription.
type activeDiscountRepo struct {
	port.DiscountRepository
	active []domain.Discount
}

func (r *activeDiscountRepo) ActiveForSubscription(_ context.Context, _, _ string) ([]domain.Discount, error) {
	return r.active, nil
}

// findByIdCouponRepo resolves a single coupon by id.
type findByIdCouponRepo struct {
	port.CouponRepository
	byId map[string]domain.Coupon
}

func (r *findByIdCouponRepo) FindById(_ context.Context, _, id string) (domain.Coupon, error) {
	c, ok := r.byId[id]
	if !ok {
		return domain.Coupon{}, port.ErrNotFound
	}
	return c, nil
}

func TestInvoiceDiscount_RepeatingHalfOff(t *testing.T) {
	// A $100/cycle subscription with a 50%-off repeating(2) coupon active from
	// cycle 0: cycles 0 and 1 bill $50, cycle 2+ bills the full $100.
	const orgId = "org_1"
	price := domain.Price{OrgId: orgId, Id: "price_1", Category: domain.PriceCategorySubscription, Scheme: domain.Fixed, UnitPrice: 10000}
	orderRepo := &fakeOrderRepo{items: []domain.OrderItem{
		{OrgId: orgId, Id: "oi_1", OrderId: "ord_1", ProductId: "prod_1", PriceId: "price_1", Quantity: 1},
	}}
	priceRepo := &mapPriceRepo{byId: map[string]domain.Price{"price_1": price}}

	coupon, err := domain.NewCoupon(domain.NewCouponInput{
		OrgId:            orgId,
		Name:             "Half off launch",
		DiscountType:     domain.DiscountTypePercentage,
		PercentOff:       decimal.NewFromInt(50),
		Duration:         domain.DurationRepeating,
		DurationInCycles: 2,
	})
	require.NoError(t, err)

	discount, err := domain.NewDiscount(domain.NewDiscountInput{
		OrgId:          orgId,
		CouponId:       coupon.Id,
		CustomerId:     "cus_1",
		OrderId:        "ord_1",
		SubscriptionId: "sub_1",
		StartCycle:     0,
	})
	require.NoError(t, err)

	discounts := &activeDiscountRepo{active: []domain.Discount{discount}}
	coupons := &findByIdCouponRepo{byId: map[string]domain.Coupon{coupon.Id: coupon}}

	build := func(cycle int) domain.Invoice {
		svc := NewInvoiceService(newFakeInvoiceRepo(), orderRepo, priceRepo, nil, nil, silentLogger{}, discounts, coupons, nil)
		sub := domain.Subscription{
			OrgId: orgId, Id: "sub_1", OrderId: "ord_1", CustomerId: "cus_1",
			Status: domain.SubscriptionStatusActive, Currency: "USD", CyclesProcessed: cycle,
		}
		inv, berr := svc.BuildForBillingPeriod(context.Background(), sub)
		require.NoError(t, berr)
		return inv
	}

	require.EqualValues(t, 5000, build(0).Total, "cycle 0: 50%% off $100 = $50")
	require.EqualValues(t, 5000, build(1).Total, "cycle 1: still in the 2-cycle window = $50")
	require.EqualValues(t, 10000, build(2).Total, "cycle 2: discount window exhausted = $100")
}
