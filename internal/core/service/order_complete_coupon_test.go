package service

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// ---- lightweight in-memory coupon fakes for the CompleteOrder consume path ----

// memReservationRepo serves an order's reservation and clears it on consume.
type memReservationRepo struct {
	port.CouponReservationRepository
	byOrder map[string][]domain.CouponReservation
}

func (r *memReservationRepo) FindByOrder(_ context.Context, _, orderId string) ([]domain.CouponReservation, error) {
	return r.byOrder[orderId], nil
}
func (r *memReservationRepo) DeleteByOrder(_ context.Context, _, orderId string) error {
	delete(r.byOrder, orderId)
	return nil
}

// memDiscountRepo records created discounts and serves them as order-active.
type memDiscountRepo struct {
	port.DiscountRepository
	created []domain.Discount
}

func (r *memDiscountRepo) Create(_ context.Context, d domain.Discount) (domain.Discount, error) {
	r.created = append(r.created, d)
	return d, nil
}
func (r *memDiscountRepo) ActiveForOrder(_ context.Context, _, orderId string) ([]domain.Discount, error) {
	out := make([]domain.Discount, 0, len(r.created))
	for _, d := range r.created {
		if d.OrderId == orderId {
			out = append(out, d)
		}
	}
	return out, nil
}

// memCodeRepo counts redemptions.
type memCodeRepo struct {
	port.CouponCodeRepository
	redeemed map[string]int
}

func (r *memCodeRepo) IncrementRedeemed(_ context.Context, _, id string) error {
	if r.redeemed == nil {
		r.redeemed = map[string]int{}
	}
	r.redeemed[id]++
	return nil
}

// memCouponRepo serves coupons by id (used by BuildForOrder's discount apply).
type memCouponRepo struct {
	port.CouponRepository
	byId map[string]domain.Coupon
}

func (r *memCouponRepo) FindById(_ context.Context, _, id string) (domain.Coupon, error) {
	if c, ok := r.byId[id]; ok {
		return c, nil
	}
	return domain.Coupon{}, port.ErrNotFound
}

// CompleteOrder for a pure one-time order with a held coupon reservation
// consumes the reservation into an order-owned Discount, builds ONE discounted
// invoice, settles it to paid, and clears the reservation. No orphan hold.
func TestOrderService_CompleteOrder_OneTimeCouponConsumed(t *testing.T) {
	const orgId = "org_1"

	coupon, err := domain.NewCoupon(domain.NewCouponInput{
		OrgId:        orgId,
		Name:         "Quarter off",
		DiscountType: domain.DiscountTypePercentage,
		PercentOff:   decimal.NewFromInt(25),
		Duration:     domain.DurationOnce,
	})
	require.NoError(t, err)

	res, err := domain.NewCouponReservation(domain.NewCouponReservationInput{
		OrgId:        orgId,
		CouponId:     coupon.Id,
		CouponCodeId: "cc_1",
		CustomerId:   "cust_1",
		OrderId:      "ord_1",
		ExpiresAt:    time.Now().UTC().Add(time.Hour),
	})
	require.NoError(t, err)

	resRepo := &memReservationRepo{byOrder: map[string][]domain.CouponReservation{"ord_1": {res}}}
	discRepo := &memDiscountRepo{}
	codeRepo := &memCodeRepo{}
	couponRepo := &memCouponRepo{byId: map[string]domain.Coupon{coupon.Id: coupon}}
	tx := &fakeTxManager{}

	coupons := NewCouponService(couponRepo, codeRepo, discRepo, nil, tx, silentLogger{}, resRepo)

	// Pure one-time order: a single $200 one-time line, no subscriptions.
	price := domain.Price{OrgId: orgId, Id: "price_x", Scheme: domain.Fixed, UnitPrice: 20000, Currency: domain.USD}
	priceRepo := &mapPriceRepo{byId: map[string]domain.Price{"price_x": price}}
	orderRepo := &fakeOrderRepo{order: pendingOrder(), items: []domain.OrderItem{
		{OrgId: orgId, Id: "oi_x", OrderId: "ord_1", ProductId: "prod_x", PriceId: "price_x", Quantity: 1},
	}}
	subRepo := &fakeSubRepo{} // no subscriptions
	custRepo := &fakeCustomerRepo{paymentMethod: domain.PaymentMethod{Id: "pm_1"}}
	payRepo := &fakePaymentRepo{}
	invRepo := newFakeInvoiceRepo()
	engine := &recordingEngine{}
	ps := &recordingPubSub{}

	// InvoiceService wired WITH the discount + coupon repos so the order discount
	// is applied to the combined invoice.
	invSvc := NewInvoiceService(invRepo, orderRepo, priceRepo, subRepo, nil, tx, silentLogger{}, discRepo, couponRepo, nil, nil)
	svc := NewOrderService(tx, engine, nil, priceRepo, nil, orderRepo, custRepo, subRepo, payRepo, &fakePaymentMethodRepo{}, nil, nil, ps, silentLogger{}, coupons, invSvc)

	_, err = svc.CompleteOrder(context.Background(), port.CompleteOrderInput{
		OrgId: orgId, Id: "ord_1", PaymentMethodId: "pm_1",
		Payment: port.CompleteOrderInputPayment{Amount: 15000, Currency: "USD"},
	})
	require.NoError(t, err)

	// Reservation consumed into exactly one order-owned discount; hold cleared.
	require.Len(t, discRepo.created, 1, "reservation converts to one discount")
	assert.Equal(t, "ord_1", discRepo.created[0].OrderId, "discount is order-owned")
	assert.Equal(t, "", discRepo.created[0].SubscriptionId, "no subscription on a one-time order")
	assert.Empty(t, resRepo.byOrder["ord_1"], "reservation cleared on consume")
	assert.Equal(t, 1, codeRepo.redeemed["cc_1"], "code redemption incremented")

	// One discounted, paid invoice.
	require.Len(t, invRepo.byOrder, 1, "exactly one combined invoice")
	inv := invRepo.byOrder["ord_1"]
	assert.EqualValues(t, 15000, inv.Total, "200 less 25% = 150")
	assert.EqualValues(t, 5000, inv.DiscountTotal)
	assert.Equal(t, domain.InvoiceStatusPaid, invRepo.byId[inv.Id].Status)
	require.Len(t, payRepo.created, 1)
	assert.Equal(t, inv.Id, payRepo.created[0].InvoiceId)
	assert.False(t, payRepo.created[0].Recurring, "one-time order payment is not recurring")
}

// A second BuildForOrder (re-entrant invoice build) does not create a second
// invoice — FindOrderInvoice hits the existing one.
func TestOrderService_CompleteOrder_InvoiceIdempotent(t *testing.T) {
	const orgId = "org_1"
	price := domain.Price{OrgId: orgId, Id: "price_x", Scheme: domain.Fixed, UnitPrice: 5000, Currency: domain.USD}
	priceRepo := &mapPriceRepo{byId: map[string]domain.Price{"price_x": price}}
	orderRepo := &fakeOrderRepo{order: pendingOrder(), items: []domain.OrderItem{
		{OrgId: orgId, Id: "oi_x", OrderId: "ord_1", PriceId: "price_x", Quantity: 1},
	}}
	subRepo := &fakeSubRepo{}
	custRepo := &fakeCustomerRepo{paymentMethod: domain.PaymentMethod{Id: "pm_1"}}
	payRepo := &fakePaymentRepo{}
	invRepo := newFakeInvoiceRepo()
	tx := &fakeTxManager{}
	svc := newOrderServiceWithInvoice(tx, &recordingEngine{}, orderRepo, custRepo, subRepo, &fakePaymentMethodRepo{}, payRepo, &recordingPubSub{}, priceRepo, invRepo, nil)

	_, err := svc.CompleteOrder(context.Background(), port.CompleteOrderInput{
		OrgId: orgId, Id: "ord_1", PaymentMethodId: "pm_1",
		Payment: port.CompleteOrderInputPayment{Amount: 5000, Currency: "USD"},
	})
	require.NoError(t, err)
	require.Len(t, invRepo.byOrder, 1)
	first := invRepo.byOrder["ord_1"]

	// A re-entrant build (the upfront-invoice path, or a retried completion) must
	// reuse the existing invoice rather than mint a second.
	again, err := svc.invoiceService.BuildForOrder(context.Background(), orderRepo.order)
	require.NoError(t, err)
	assert.Equal(t, first.Id, again.Id, "second build reuses the existing order invoice")
	assert.Len(t, invRepo.byOrder, 1, "no second invoice persisted")
}
