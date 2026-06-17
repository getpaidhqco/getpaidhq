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

func TestRedeem_CreatesDiscountAndIncrements(t *testing.T) {
	svc, _, ccr, dr, _ := newCouponService(t)
	c := seedCoupon(svc, port.CreateCouponInput{Name: "Q", DiscountType: "percentage", PercentOff: decimal.NewFromInt(20), Duration: "repeating", DurationInCycles: 3})
	code, _ := domain.NewCouponCode(domain.NewCouponCodeInput{OrgId: "org_1", CouponId: c.Id, Code: "SAVE"})
	ccr.byCode[code.Code] = code
	ccr.byId[code.Id] = code

	d, err := svc.Redeem(context.Background(), port.RedeemCouponInput{
		OrgId: "org_1", Code: "SAVE", CustomerId: "cus_1", SubscriptionId: "sub_1", StartCycle: 4, Currency: "USD",
	})
	require.NoError(t, err)
	assert.Equal(t, c.Id, d.CouponId)
	assert.Equal(t, code.Id, d.CouponCodeId)
	assert.Equal(t, 4, d.StartCycle)
	require.Len(t, dr.created, 1)
	require.Len(t, ccr.redeemed, 1, "TimesRedeemed incremented")
}

func TestRedeem_RefusedReturnsError(t *testing.T) {
	svc, cr, ccr, _, _ := newCouponService(t)
	c := seedCoupon(svc, port.CreateCouponInput{Name: "Q", DiscountType: "percentage", PercentOff: decimal.NewFromInt(20), Duration: "forever"})
	c.Active = false
	cr.byId[c.Id] = c
	code, _ := domain.NewCouponCode(domain.NewCouponCodeInput{OrgId: "org_1", CouponId: c.Id, Code: "SAVE"})
	ccr.byCode[code.Code] = code
	ccr.byId[code.Id] = code

	_, err := svc.Redeem(context.Background(), port.RedeemCouponInput{
		OrgId: "org_1", Code: "SAVE", CustomerId: "cus_1", SubscriptionId: "sub_1",
	})
	require.Error(t, err)
}

func TestRedeem_Programmatic_NoCode(t *testing.T) {
	svc, _, _, dr, _ := newCouponService(t)
	c := seedCoupon(svc, port.CreateCouponInput{Name: "Q", DiscountType: "percentage", PercentOff: decimal.NewFromInt(20), Duration: "forever"})

	d, err := svc.Redeem(context.Background(), port.RedeemCouponInput{
		OrgId: "org_1", CouponId: c.Id, CustomerId: "cus_1", OrderId: "ord_1",
	})
	require.NoError(t, err)
	assert.Empty(t, d.CouponCodeId)
	require.Len(t, dr.created, 1)
}
