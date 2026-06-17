package service

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

func newCouponService(t *testing.T) (*CouponService, *fakeCouponRepo, *fakeCouponCodeRepo, *fakeDiscountRepo, *fakePriorPayments) {
	t.Helper()
	cr := &fakeCouponRepo{byId: map[string]domain.Coupon{}}
	ccr := &fakeCouponCodeRepo{byCode: map[string]domain.CouponCode{}, byId: map[string]domain.CouponCode{}}
	dr := &fakeDiscountRepo{}
	pp := &fakePriorPayments{}
	svc := NewCouponService(cr, ccr, dr, pp, noopTx{}, silentLogger{})
	return svc, cr, ccr, dr, pp
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
}

func (r *fakeDiscountRepo) Create(_ context.Context, d domain.Discount) (domain.Discount, error) {
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

type noopTx struct{}

func (noopTx) RunInTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }
