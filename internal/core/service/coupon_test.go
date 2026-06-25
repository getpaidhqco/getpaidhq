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

func newCouponService(t *testing.T) (*CouponService, *fakeCouponRepo, *fakeCouponCodeRepo, *fakeDiscountRepo, *fakePriorPayments) {
	t.Helper()
	svc, cr, ccr, dr, pp, _ := newCouponServiceWithReservations(t)
	return svc, cr, ccr, dr, pp
}

// newCouponServiceWithReservations is newCouponService plus the reservation fake
// handle, for the Reserve/Consume tests.
func newCouponServiceWithReservations(t *testing.T) (*CouponService, *fakeCouponRepo, *fakeCouponCodeRepo, *fakeDiscountRepo, *fakePriorPayments, *fakeCouponReservationRepo) {
	t.Helper()
	cr := &fakeCouponRepo{byId: map[string]domain.Coupon{}}
	ccr := &fakeCouponCodeRepo{byCode: map[string]domain.CouponCode{}, byId: map[string]domain.CouponCode{}}
	dr := &fakeDiscountRepo{}
	pp := &fakePriorPayments{}
	rr := &fakeCouponReservationRepo{}
	svc := NewCouponService(cr, ccr, dr, pp, noopTx{}, silentLogger{}, rr)
	return svc, cr, ccr, dr, pp, rr
}

func TestCouponService_Create(t *testing.T) {
	svc, cr, _, _, _ := newCouponService(t)
	got, err := svc.Create(context.Background(), "org_1", port.CreateCouponInput{
		Name: "Q", DiscountType: "percentage", PercentOff: decimal.NewFromInt(25), Duration: "forever",
	})
	require.NoError(t, err)
	assert.True(t, got.Active)
	require.Len(t, cr.created, 1)
}

func TestCouponService_Create_InvalidRejected(t *testing.T) {
	svc, _, _, _, _ := newCouponService(t)
	_, err := svc.Create(context.Background(), "org_1", port.CreateCouponInput{
		Name: "Bad", DiscountType: "percentage", PercentOff: decimal.NewFromInt(25),
		AmountOff: 500, Duration: "forever",
	})
	require.Error(t, err)
}

func TestCouponService_Update_OnlyMutable(t *testing.T) {
	svc, cr, _, _, _ := newCouponService(t)
	c, _ := svc.Create(context.Background(), "org_1", port.CreateCouponInput{Name: "Q", DiscountType: "percentage", PercentOff: decimal.NewFromInt(10), Duration: "forever"})
	_, err := svc.Update(context.Background(), "org_1", c.Id, port.UpdateCouponInput{Name: "Renamed", Active: false})
	require.NoError(t, err)
	assert.Equal(t, "Renamed", cr.byId[c.Id].Name)
	assert.False(t, cr.byId[c.Id].Active)
}

func TestCouponService_CreateCode(t *testing.T) {
	svc, _, ccr, _, _ := newCouponService(t)
	c, _ := svc.Create(context.Background(), "org_1", port.CreateCouponInput{Name: "Q", DiscountType: "percentage", PercentOff: decimal.NewFromInt(10), Duration: "forever"})
	got, err := svc.CreateCode(context.Background(), "org_1", c.Id, port.CreateCouponCodeInput{Code: "save10"})
	require.NoError(t, err)
	assert.Equal(t, "SAVE10", got.Code)
	require.Len(t, ccr.created, 1)
}

// ----- fakes -----

type fakeCouponRepo struct {
	port.CouponRepository
	byId    map[string]domain.Coupon
	created []domain.Coupon
}

func (r *fakeCouponRepo) Create(_ context.Context, c domain.Coupon) (domain.Coupon, error) {
	r.created = append(r.created, c)
	r.byId[c.Id] = c
	return c, nil
}
func (r *fakeCouponRepo) FindById(_ context.Context, _, id string) (domain.Coupon, error) {
	c, ok := r.byId[id]
	if !ok {
		return domain.Coupon{}, port.ErrNotFound
	}
	return c, nil
}
func (r *fakeCouponRepo) UpdateMutable(_ context.Context, orgId, id, name string, active bool, md map[string]string) (domain.Coupon, error) {
	c := r.byId[id]
	c.Name, c.Active, c.Metadata = name, active, md
	r.byId[id] = c
	return c, nil
}
func (r *fakeCouponRepo) FindByIdForUpdate(ctx context.Context, orgId, id string) (domain.Coupon, error) {
	return r.FindById(ctx, orgId, id)
}

type fakeCouponCodeRepo struct {
	port.CouponCodeRepository
	byCode   map[string]domain.CouponCode
	byId     map[string]domain.CouponCode
	created  []domain.CouponCode
	redeemed []string
}

func (r *fakeCouponCodeRepo) Create(_ context.Context, c domain.CouponCode) (domain.CouponCode, error) {
	r.created = append(r.created, c)
	r.byCode[c.Code] = c
	r.byId[c.Id] = c
	return c, nil
}
func (r *fakeCouponCodeRepo) FindByCode(_ context.Context, _, code string) (domain.CouponCode, error) {
	c, ok := r.byCode[code]
	if !ok {
		return domain.CouponCode{}, port.ErrNotFound
	}
	return c, nil
}
func (r *fakeCouponCodeRepo) FindByCodeForUpdate(ctx context.Context, orgId, code string) (domain.CouponCode, error) {
	return r.FindByCode(ctx, orgId, code)
}
func (r *fakeCouponCodeRepo) IncrementRedeemed(_ context.Context, _, id string) error {
	r.redeemed = append(r.redeemed, id)
	c := r.byId[id]
	c.TimesRedeemed++
	r.byId[id] = c
	r.byCode[c.Code] = c
	return nil
}

type fakeDiscountRepo struct {
	port.DiscountRepository
	created         []domain.Discount
	countByCoupon   int
	countByCustomer int
	createErr       error // when set, Create returns this instead of recording
}

func (r *fakeDiscountRepo) Create(_ context.Context, d domain.Discount) (domain.Discount, error) {
	if r.createErr != nil {
		return domain.Discount{}, r.createErr
	}
	r.created = append(r.created, d)
	return d, nil
}
func (r *fakeDiscountRepo) CountByCoupon(_ context.Context, _, _ string) (int, error) {
	return r.countByCoupon, nil
}
func (r *fakeDiscountRepo) CountByCouponAndCustomer(_ context.Context, _, _, _ string) (int, error) {
	return r.countByCustomer, nil
}

type fakePriorPayments struct{ prior bool }

func (p *fakePriorPayments) HasPriorSuccessfulPayment(_ context.Context, _, _ string) (bool, error) {
	return p.prior, nil
}

type fakeCouponReservationRepo struct {
	port.CouponReservationRepository
	created       []domain.CouponReservation
	deletedOrders []string
	byOrder       map[string][]domain.CouponReservation
	liveByCoupon  int
	liveByCode    int
	existsForCust bool
}

func (r *fakeCouponReservationRepo) Create(_ context.Context, res domain.CouponReservation) (domain.CouponReservation, error) {
	r.created = append(r.created, res)
	if r.byOrder == nil {
		r.byOrder = map[string][]domain.CouponReservation{}
	}
	if res.OrderId != "" {
		r.byOrder[res.OrderId] = append(r.byOrder[res.OrderId], res)
	}
	return res, nil
}
func (r *fakeCouponReservationRepo) FindByOrder(_ context.Context, _, orderId string) ([]domain.CouponReservation, error) {
	return r.byOrder[orderId], nil
}
func (r *fakeCouponReservationRepo) DeleteByOrder(_ context.Context, _, orderId string) error {
	r.deletedOrders = append(r.deletedOrders, orderId)
	if r.byOrder != nil {
		delete(r.byOrder, orderId)
	}
	return nil
}
func (r *fakeCouponReservationRepo) CountLiveByCoupon(_ context.Context, _, _ string, _ time.Time) (int, error) {
	return r.liveByCoupon, nil
}
func (r *fakeCouponReservationRepo) CountLiveByCode(_ context.Context, _, _ string, _ time.Time) (int, error) {
	return r.liveByCode, nil
}
func (r *fakeCouponReservationRepo) ExistsLiveForCustomer(_ context.Context, _, _, _ string, _ time.Time) (bool, error) {
	return r.existsForCust, nil
}

type noopTx struct{}

func (noopTx) RunInTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

// noopUsage is a MeteredUsageReader that reports zero usage — for invoice tests
// that don't exercise metered billing but must wire a real (non-nil) reader.
type noopUsage struct{}

func (noopUsage) MeteredUsageForSubscription(ctx context.Context, sub domain.Subscription, price domain.Price, from, to time.Time) (MeteredUsage, error) {
	return MeteredUsage{}, nil
}

// noopCoupons is an OrderCoupons that reserves/consumes nothing — for order
// tests that don't exercise coupons but must wire a real (non-nil) coupon dep.
type noopCoupons struct{}

func (noopCoupons) Reserve(ctx context.Context, in ReserveInput) (domain.CouponReservation, error) {
	return domain.CouponReservation{}, nil
}

func (noopCoupons) Consume(ctx context.Context, in ConsumeInput) (domain.Discount, error) {
	return domain.Discount{}, nil
}

// noopInvoicing is an OrderInvoicing that builds no invoice (BuildForOrder
// returns port.ErrNotFound, mirroring an order with nothing to invoice) — for
// order tests that don't exercise invoicing but must wire a real (non-nil) dep.
// This preserves the old "nil invoiceService → no invoice" behaviour.
type noopInvoicing struct{}

func (noopInvoicing) BuildForOrder(ctx context.Context, order domain.Order) (domain.Invoice, error) {
	return domain.Invoice{}, port.ErrNotFound
}

func (noopInvoicing) MarkOpen(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error) {
	return domain.Invoice{}, nil
}

func (noopInvoicing) SettleOrderInvoice(ctx context.Context, orgId, invoiceId string) error {
	return nil
}

// noopBillingInvoicing is a BillingInvoicing that has no current-cycle invoice
// (FindCurrentCycle returns port.ErrNotFound) — for subscription/dunning tests
// that don't exercise per-cycle invoicing but must wire a real (non-nil) dep.
// Preserves the old "nil invoiceService → no-op" behaviour.
type noopBillingInvoicing struct{}

func (noopBillingInvoicing) BuildForBillingPeriod(ctx context.Context, sub domain.Subscription) (domain.Invoice, error) {
	return domain.Invoice{}, nil
}

func (noopBillingInvoicing) FindCurrentCycle(ctx context.Context, orgId, subscriptionId string, cycle int) (domain.Invoice, error) {
	return domain.Invoice{}, port.ErrNotFound
}

func (noopBillingInvoicing) MarkOpen(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error) {
	return domain.Invoice{}, nil
}

func (noopBillingInvoicing) MarkSettled(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error) {
	return domain.Invoice{}, nil
}

func (noopBillingInvoicing) MarkUncollectible(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error) {
	return domain.Invoice{}, nil
}

func (noopBillingInvoicing) Void(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error) {
	return domain.Invoice{}, nil
}
