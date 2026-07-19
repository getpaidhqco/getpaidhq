package service

import (
	"context"
	"getpaidhq/internal/lib/errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

func TestCouponReserve_ValidCodeReservesOne(t *testing.T) {
	svc, _, ccr, _, _, rr := newCouponServiceWithReservations(t)
	c := seedCoupon(svc, port.CreateCouponInput{Name: "Q", DiscountType: "percentage", PercentOff: decimal.NewFromInt(20), Duration: "forever"})
	code, _ := domain.NewCouponCode(domain.NewCouponCodeInput{OrgId: "org_1", CouponId: c.Id, Code: "SAVE"})
	ccr.byCode[code.Code] = code
	ccr.byId[code.Id] = code

	res, err := svc.Reserve(context.Background(), ReserveInput{
		OrgId: "org_1", Code: "SAVE", CustomerId: "cus_1", OrderId: "ord_1", Currency: "USD",
	})
	require.NoError(t, err)
	assert.Equal(t, c.Id, res.CouponId)
	assert.Equal(t, code.Id, res.CouponCodeId)
	assert.Equal(t, "ord_1", res.OrderId)
	require.Len(t, rr.created, 1, "reservation recorded")
}

func TestCouponReserve_CapReached(t *testing.T) {
	svc, _, _, _, _, rr := newCouponServiceWithReservations(t)
	c := seedCoupon(svc, port.CreateCouponInput{Name: "Q", DiscountType: "percentage", PercentOff: decimal.NewFromInt(20), Duration: "forever", MaxRedemptions: 1})
	rr.liveByCoupon = 1 // one hold already outstanding fills the only slot

	_, err := svc.Reserve(context.Background(), ReserveInput{
		OrgId: "org_1", CouponId: c.Id, CustomerId: "cus_1", OrderId: "ord_1", Currency: "USD",
	})
	require.Error(t, err)
	var ce errors.CustomError
	require.ErrorAs(t, err, &ce)
	assert.Equal(t, errors.ConflictError, ce.Type)
	assert.Contains(t, ce.Message, "cap_reached")
	assert.Empty(t, rr.created, "no reservation created on refusal")
}

func TestCouponReserve_UnknownCode(t *testing.T) {
	svc, _, _, _, _, _ := newCouponServiceWithReservations(t)
	_, err := svc.Reserve(context.Background(), ReserveInput{
		OrgId: "org_1", Code: "NOPE", CustomerId: "cus_1", OrderId: "ord_1", Currency: "USD",
	})
	require.Error(t, err)
	var ce errors.CustomError
	require.ErrorAs(t, err, &ce)
	assert.Contains(t, ce.Message, "code_not_found")
}
