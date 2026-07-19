package domain

import (
	"getpaidhq/internal/lib/errors"
	"getpaidhq/internal/lib/ids"
	"time"
)

// CouponReservation is an ephemeral hold on a coupon code's redemption capacity
// for one checkout. Held by the order (build-now) or a checkout session
// (forward). No status — presence + ExpiresAt encode the state.
type CouponReservation struct {
	OrgId             string
	Id                string
	CouponId          string
	CouponCodeId      string // "" = programmatic / code-less
	CustomerId        string // "" until bound
	CheckoutSessionId string // holder (forward)
	OrderId           string // holder (build-now)
	ExpiresAt         time.Time
	CreatedAt         time.Time
}

type NewCouponReservationInput struct {
	OrgId             string
	CouponId          string
	CouponCodeId      string
	CustomerId        string
	CheckoutSessionId string
	OrderId           string
	ExpiresAt         time.Time
}

func NewCouponReservation(in NewCouponReservationInput) (CouponReservation, error) {
	if in.OrgId == "" || in.CouponId == "" {
		return CouponReservation{}, errors.NewCustomError(errors.ValidationError, "reservation requires org and coupon", nil)
	}
	if in.OrderId == "" && in.CheckoutSessionId == "" {
		return CouponReservation{}, errors.NewCustomError(errors.ValidationError, "reservation requires a holder (order or checkout session)", nil)
	}
	if in.ExpiresAt.IsZero() {
		return CouponReservation{}, errors.NewCustomError(errors.ValidationError, "reservation requires expires_at", nil)
	}
	now := time.Now().UTC()
	return CouponReservation{
		OrgId: in.OrgId, Id: ids.Generate("cres"),
		CouponId: in.CouponId, CouponCodeId: in.CouponCodeId, CustomerId: in.CustomerId,
		CheckoutSessionId: in.CheckoutSessionId, OrderId: in.OrderId,
		ExpiresAt: in.ExpiresAt, CreatedAt: now,
	}, nil
}

// IsLive reports whether the hold still counts at now.
func (r CouponReservation) IsLive(now time.Time) bool { return r.ExpiresAt.After(now) }
