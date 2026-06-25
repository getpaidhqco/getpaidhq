package service

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

func TestInvoiceService_BuildForOrder_Mixed(t *testing.T) {
	// One order: a $100/mo recurring plan (qty 1) + a $50 one-time line.
	const orgId = "org_1"
	plan := domain.Price{
		OrgId: orgId, Id: "price_plan", Category: domain.PriceCategorySubscription,
		Scheme: domain.Fixed, UnitPrice: 10000, BillingInterval: domain.BillingIntervalMonth, BillingIntervalQty: 1,
	}
	oneTime := domain.Price{OrgId: orgId, Id: "price_setup", Scheme: domain.Fixed, UnitPrice: 5000}

	orderRepo := &fakeOrderRepo{items: []domain.OrderItem{
		{OrgId: orgId, Id: "oi_plan", OrderId: "ord_1", ProductId: "prod_a", PriceId: "price_plan", Quantity: 1},
		{OrgId: orgId, Id: "oi_setup", OrderId: "ord_1", ProductId: "prod_b", PriceId: "price_setup", Quantity: 1},
	}}
	priceRepo := &mapPriceRepo{byId: map[string]domain.Price{"price_plan": plan, "price_setup": oneTime}}

	sub := domain.Subscription{OrgId: orgId, Id: "sub_1", OrderId: "ord_1", CustomerId: "cus_1", Status: domain.SubscriptionStatusActive, Currency: "USD"}
	subRepo := &fakeSubRepo{byOrderId: []domain.Subscription{sub}}

	svc := NewInvoiceService(newFakeInvoiceRepo(), orderRepo, priceRepo, subRepo, nil, nil, silentLogger{}, nil, nil, nil, nil)

	order := domain.Order{OrgId: orgId, Id: "ord_1", CustomerId: "cus_1", Currency: "USD"}
	inv, err := svc.BuildForOrder(context.Background(), order)
	require.NoError(t, err)

	require.Len(t, inv.LineItems, 2, "want one recurring base line + one one-time base line")
	require.EqualValues(t, 15000, inv.Total, "100/mo + 50 one-time = 15000c")
	require.Equal(t, 0, inv.Cycle)
	require.Equal(t, sub.Id, inv.SubscriptionId, "single-sub order: invoice is that sub's cycle-0 invoice")
	require.Equal(t, "ord_1", inv.OrderId)
	require.EqualValues(t, 1, inv.Number)
}

func TestInvoiceService_BuildForOrder_PureOneTimeWithDiscount(t *testing.T) {
	// Pure one-time order ($200) with a 25%-off order-level discount → $150,
	// and no subscription linkage.
	const orgId = "org_1"
	price := domain.Price{OrgId: orgId, Id: "price_x", Scheme: domain.Fixed, UnitPrice: 20000}
	orderRepo := &fakeOrderRepo{items: []domain.OrderItem{
		{OrgId: orgId, Id: "oi_x", OrderId: "ord_2", ProductId: "prod_x", PriceId: "price_x", Quantity: 1},
	}}
	priceRepo := &mapPriceRepo{byId: map[string]domain.Price{"price_x": price}}

	coupon, err := domain.NewCoupon(domain.NewCouponInput{
		OrgId:        orgId,
		Name:         "Quarter off",
		DiscountType: domain.DiscountTypePercentage,
		PercentOff:   decimal.NewFromInt(25),
		Duration:     domain.DurationOnce,
	})
	require.NoError(t, err)
	discount, err := domain.NewDiscount(domain.NewDiscountInput{
		OrgId:      orgId,
		CouponId:   coupon.Id,
		CustomerId: "cus_1",
		OrderId:    "ord_2",
		StartCycle: 0,
	})
	require.NoError(t, err)

	discounts := &orderDiscountRepo{active: []domain.Discount{discount}}
	coupons := &findByIdCouponRepo{byId: map[string]domain.Coupon{coupon.Id: coupon}}
	subRepo := &fakeSubRepo{} // no subscriptions on the order

	svc := NewInvoiceService(newFakeInvoiceRepo(), orderRepo, priceRepo, subRepo, nil, nil, silentLogger{}, discounts, coupons, nil, nil)

	order := domain.Order{OrgId: orgId, Id: "ord_2", CustomerId: "cus_1", Currency: "USD"}
	inv, err := svc.BuildForOrder(context.Background(), order)
	require.NoError(t, err)

	require.Len(t, inv.LineItems, 1)
	require.EqualValues(t, 15000, inv.Total, "200 less 25% = 150")
	require.EqualValues(t, 5000, inv.DiscountTotal)
	require.Equal(t, "", inv.SubscriptionId, "pure one-time order has no subscription linkage")
}

func TestInvoiceService_BuildForOrder_ReservationDiscount(t *testing.T) {
	// Pre-payment (upfront) path: the order has a LIVE coupon reservation but NO
	// committed Discount yet (ActiveForOrder is empty). The built invoice must
	// still carry the discounted total, resolved from the reservation's coupon.
	// $200 line, 25%-off coupon → Total 15000, DiscountTotal 5000.
	const orgId = "org_1"
	price := domain.Price{OrgId: orgId, Id: "price_x", Scheme: domain.Fixed, UnitPrice: 20000}
	orderRepo := &fakeOrderRepo{items: []domain.OrderItem{
		{OrgId: orgId, Id: "oi_x", OrderId: "ord_r", ProductId: "prod_x", PriceId: "price_x", Quantity: 1},
	}}
	priceRepo := &mapPriceRepo{byId: map[string]domain.Price{"price_x": price}}

	coupon, err := domain.NewCoupon(domain.NewCouponInput{
		OrgId:        orgId,
		Name:         "Quarter off",
		DiscountType: domain.DiscountTypePercentage,
		PercentOff:   decimal.NewFromInt(25),
		Duration:     domain.DurationOnce,
	})
	require.NoError(t, err)

	// No committed discount (empty ActiveForOrder), but a live reservation exists.
	discounts := &orderDiscountRepo{active: nil}
	coupons := &findByIdCouponRepo{byId: map[string]domain.Coupon{coupon.Id: coupon}}
	reservations := &fakeCouponReservationRepo{byOrder: map[string][]domain.CouponReservation{
		"ord_r": {{OrgId: orgId, Id: "cres_1", CouponId: coupon.Id, CustomerId: "cus_1", OrderId: "ord_r"}},
	}}
	subRepo := &fakeSubRepo{} // no subscriptions on the order

	svc := NewInvoiceService(newFakeInvoiceRepo(), orderRepo, priceRepo, subRepo, nil, nil, silentLogger{}, discounts, coupons, reservations, nil)

	order := domain.Order{OrgId: orgId, Id: "ord_r", CustomerId: "cus_1", Currency: "USD"}
	inv, err := svc.BuildForOrder(context.Background(), order)
	require.NoError(t, err)

	require.Len(t, inv.LineItems, 1)
	require.EqualValues(t, 15000, inv.Total, "200 less 25% = 150, from the reserved coupon")
	require.EqualValues(t, 5000, inv.DiscountTotal)
}

func TestInvoiceService_BuildForOrder_Idempotent(t *testing.T) {
	const orgId = "org_1"
	price := domain.Price{OrgId: orgId, Id: "price_x", Scheme: domain.Fixed, UnitPrice: 5000}
	orderRepo := &fakeOrderRepo{items: []domain.OrderItem{
		{OrgId: orgId, Id: "oi_x", OrderId: "ord_3", PriceId: "price_x", Quantity: 1},
	}}
	priceRepo := &mapPriceRepo{byId: map[string]domain.Price{"price_x": price}}
	repo := newFakeInvoiceRepo()
	svc := NewInvoiceService(repo, orderRepo, priceRepo, &fakeSubRepo{}, nil, nil, silentLogger{}, nil, nil, nil, nil)

	order := domain.Order{OrgId: orgId, Id: "ord_3", CustomerId: "cus_1", Currency: "USD"}
	first, err := svc.BuildForOrder(context.Background(), order)
	require.NoError(t, err)

	second, err := svc.BuildForOrder(context.Background(), order)
	require.NoError(t, err)
	require.Equal(t, first.Id, second.Id, "second build reuses the existing order invoice")
	require.EqualValues(t, 1, second.Number, "no second invoice number consumed")
	require.Len(t, repo.byId, 1, "only one invoice persisted")
}

func TestInvoiceService_BuildForOrder_NoItems(t *testing.T) {
	const orgId = "org_1"
	orderRepo := &fakeOrderRepo{items: nil}
	svc := NewInvoiceService(newFakeInvoiceRepo(), orderRepo, &mapPriceRepo{}, &fakeSubRepo{}, nil, nil, silentLogger{}, nil, nil, nil, nil)

	order := domain.Order{OrgId: orgId, Id: "ord_empty", CustomerId: "cus_1", Currency: "USD"}
	_, err := svc.BuildForOrder(context.Background(), order)
	require.ErrorIs(t, err, port.ErrNotFound)
}

// orderDiscountRepo serves a fixed set of order-owned active discounts.
type orderDiscountRepo struct {
	port.DiscountRepository
	active []domain.Discount
}

func (r *orderDiscountRepo) ActiveForOrder(_ context.Context, _, _ string) ([]domain.Discount, error) {
	return r.active, nil
}
