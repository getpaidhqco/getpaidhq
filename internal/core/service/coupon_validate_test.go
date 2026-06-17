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

func seedCoupon(svc *CouponService, in port.CreateCouponInput) domain.Coupon {
	c, _ := svc.Create(context.Background(), "org_1", in)
	return c
}

func TestValidate_HappyPath(t *testing.T) {
	svc, _, ccr, _, _ := newCouponService(t)
	c := seedCoupon(svc, port.CreateCouponInput{Name: "Q", DiscountType: "percentage", PercentOff: decimal.NewFromInt(20), Duration: "forever"})
	code, _ := domain.NewCouponCode(domain.NewCouponCodeInput{OrgId: "org_1", CouponId: c.Id, Code: "SAVE"})
	ccr.byCode[code.Code] = code

	lines := []domain.DiscountableLine{{LineId: "l1", ProductId: "prd_a", Total: 1000}}
	got, err := svc.Validate(context.Background(), "org_1", "save", "cus_1", "USD", lines)
	require.NoError(t, err)
	assert.True(t, got.Valid)
	assert.EqualValues(t, 200, got.DiscountTotal)
}

func TestValidate_CodeNotFound(t *testing.T) {
	svc, _, _, _, _ := newCouponService(t)
	got, err := svc.Validate(context.Background(), "org_1", "NOPE", "cus_1", "USD", nil)
	require.NoError(t, err)
	assert.False(t, got.Valid)
	assert.Equal(t, "code_not_found", got.Reason)
}

func TestValidate_InactiveCoupon(t *testing.T) {
	svc, cr, ccr, _, _ := newCouponService(t)
	c := seedCoupon(svc, port.CreateCouponInput{Name: "Q", DiscountType: "percentage", PercentOff: decimal.NewFromInt(20), Duration: "forever"})
	c.Active = false
	cr.byId[c.Id] = c
	code, _ := domain.NewCouponCode(domain.NewCouponCodeInput{OrgId: "org_1", CouponId: c.Id, Code: "SAVE"})
	ccr.byCode[code.Code] = code

	got, _ := svc.Validate(context.Background(), "org_1", "SAVE", "cus_1", "USD", nil)
	assert.False(t, got.Valid)
	assert.Equal(t, "coupon_inactive", got.Reason)
}

func TestValidate_CodeExpired(t *testing.T) {
	svc, _, ccr, _, _ := newCouponService(t)
	c := seedCoupon(svc, port.CreateCouponInput{Name: "Q", DiscountType: "percentage", PercentOff: decimal.NewFromInt(20), Duration: "forever"})
	code, _ := domain.NewCouponCode(domain.NewCouponCodeInput{OrgId: "org_1", CouponId: c.Id, Code: "SAVE", ExpiresAt: time.Now().Add(-time.Hour)})
	ccr.byCode[code.Code] = code

	got, _ := svc.Validate(context.Background(), "org_1", "SAVE", "cus_1", "USD", nil)
	assert.False(t, got.Valid)
	assert.Equal(t, "code_expired", got.Reason)
}

func TestValidate_WrongCustomer(t *testing.T) {
	svc, _, ccr, _, _ := newCouponService(t)
	c := seedCoupon(svc, port.CreateCouponInput{Name: "Q", DiscountType: "percentage", PercentOff: decimal.NewFromInt(20), Duration: "forever"})
	code, _ := domain.NewCouponCode(domain.NewCouponCodeInput{OrgId: "org_1", CouponId: c.Id, Code: "SAVE", CustomerId: "cus_owner"})
	ccr.byCode[code.Code] = code

	got, _ := svc.Validate(context.Background(), "org_1", "SAVE", "cus_other", "USD", nil)
	assert.False(t, got.Valid)
	assert.Equal(t, "wrong_customer", got.Reason)
}

func TestValidate_BelowMinimum(t *testing.T) {
	svc, _, ccr, _, _ := newCouponService(t)
	c := seedCoupon(svc, port.CreateCouponInput{Name: "Q", DiscountType: "percentage", PercentOff: decimal.NewFromInt(20), Duration: "forever"})
	code, _ := domain.NewCouponCode(domain.NewCouponCodeInput{OrgId: "org_1", CouponId: c.Id, Code: "SAVE", Restrictions: domain.Restrictions{MinimumAmount: 5000, MinimumAmountCurrency: "USD"}})
	ccr.byCode[code.Code] = code

	lines := []domain.DiscountableLine{{LineId: "l1", ProductId: "prd_a", Total: 1000}}
	got, _ := svc.Validate(context.Background(), "org_1", "SAVE", "cus_1", "USD", lines)
	assert.False(t, got.Valid)
	assert.Equal(t, "below_minimum", got.Reason)
}

func TestValidate_NotFirstTime(t *testing.T) {
	svc, _, ccr, _, pp := newCouponService(t)
	pp.prior = true
	c := seedCoupon(svc, port.CreateCouponInput{Name: "Q", DiscountType: "percentage", PercentOff: decimal.NewFromInt(20), Duration: "forever"})
	code, _ := domain.NewCouponCode(domain.NewCouponCodeInput{OrgId: "org_1", CouponId: c.Id, Code: "SAVE", Restrictions: domain.Restrictions{FirstTimeTransaction: true}})
	ccr.byCode[code.Code] = code

	got, _ := svc.Validate(context.Background(), "org_1", "SAVE", "cus_1", "USD", nil)
	assert.False(t, got.Valid)
	assert.Equal(t, "not_first_time", got.Reason)
}
