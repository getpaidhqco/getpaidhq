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
	"getpaidhq/internal/lib"
)

func seedReservation(t *testing.T, rr *fakeCouponReservationRepo, c domain.Coupon, codeId, orderId string) domain.CouponReservation {
	t.Helper()
	r, err := domain.NewCouponReservation(domain.NewCouponReservationInput{
		OrgId: "org_1", CouponId: c.Id, CouponCodeId: codeId, CustomerId: "cus_1",
		OrderId: orderId, ExpiresAt: time.Now().UTC().Add(time.Hour),
	})
	require.NoError(t, err)
	if rr.byOrder == nil {
		rr.byOrder = map[string][]domain.CouponReservation{}
	}
	rr.byOrder[orderId] = append(rr.byOrder[orderId], r)
	return r
}

func TestCouponConsume_CreatesDiscountIncrementsAndDeletes(t *testing.T) {
	svc, _, ccr, dr, _, rr := newCouponServiceWithReservations(t)
	c := seedCoupon(svc, port.CreateCouponInput{Name: "Q", DiscountType: "percentage", PercentOff: decimal.NewFromInt(50), Duration: "repeating", DurationInCycles: 2})
	code, _ := domain.NewCouponCode(domain.NewCouponCodeInput{OrgId: "org_1", CouponId: c.Id, Code: "SAVE"})
	ccr.byCode[code.Code] = code
	ccr.byId[code.Id] = code
	seedReservation(t, rr, c, code.Id, "ord_1")

	d, err := svc.Consume(context.Background(), ConsumeInput{
		OrgId: "org_1", OrderId: "ord_1", SubscriptionId: "sub_1", StartCycle: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, "sub_1", d.SubscriptionId)
	assert.Equal(t, "ord_1", d.OrderId)
	assert.Equal(t, 0, d.StartCycle)
	require.Len(t, dr.created, 1, "one discount created")
	assert.Equal(t, "sub_1", dr.created[0].SubscriptionId)
	assert.Equal(t, "ord_1", dr.created[0].OrderId)
	assert.Equal(t, 0, dr.created[0].StartCycle)
	require.Len(t, ccr.redeemed, 1, "code times_redeemed incremented")
	assert.Contains(t, rr.deletedOrders, "ord_1", "reservation deleted")
}

func TestCouponConsume_NoReservationIsNoOp(t *testing.T) {
	svc, _, _, dr, _, rr := newCouponServiceWithReservations(t)
	d, err := svc.Consume(context.Background(), ConsumeInput{
		OrgId: "org_1", OrderId: "ord_none", SubscriptionId: "sub_1",
	})
	require.NoError(t, err)
	assert.Empty(t, dr.created)
	assert.Empty(t, rr.deletedOrders)
	assert.Equal(t, domain.Discount{}, d)
}

func TestCouponConsume_ConflictClearsReservation(t *testing.T) {
	svc, _, ccr, dr, _, rr := newCouponServiceWithReservations(t)
	c := seedCoupon(svc, port.CreateCouponInput{Name: "Q", DiscountType: "percentage", PercentOff: decimal.NewFromInt(50), Duration: "once"})
	code, _ := domain.NewCouponCode(domain.NewCouponCodeInput{OrgId: "org_1", CouponId: c.Id, Code: "SAVE"})
	ccr.byId[code.Id] = code
	seedReservation(t, rr, c, code.Id, "ord_1")
	// Already consumed: the (org,coupon,subscription) unique index rejects the insert.
	dr.createErr = lib.NewCustomError(lib.ConflictError, "discount already exists", nil)

	_, err := svc.Consume(context.Background(), ConsumeInput{
		OrgId: "org_1", OrderId: "ord_1", SubscriptionId: "sub_1",
	})
	require.NoError(t, err, "conflict is treated as already-consumed, not an error")
	assert.Contains(t, rr.deletedOrders, "ord_1", "stale reservation cleared")
	assert.Empty(t, ccr.redeemed, "no double increment on conflict")
}

func TestCouponRelease_Idempotent(t *testing.T) {
	svc, _, _, _, _, rr := newCouponServiceWithReservations(t)
	require.NoError(t, svc.Release(context.Background(), "org_1", "ord_1"))
	require.NoError(t, svc.Release(context.Background(), "org_1", "ord_1"))
	assert.Equal(t, []string{"ord_1", "ord_1"}, rr.deletedOrders)
}
