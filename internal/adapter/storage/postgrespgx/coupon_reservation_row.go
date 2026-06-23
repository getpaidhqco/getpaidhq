package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// couponReservationRow is the postgres on-the-wire shape of a CouponReservation.
//
// coupon_code_id, customer_id, checkout_session_id and order_id are nullable FK
// columns (NULL, never ""), carried as *string via nilIfEmpty/strOrEmpty —
// mirroring couponCodeRow.CustomerId.
type couponReservationRow struct {
	OrgId             string
	Id                string
	CouponId          string
	CouponCodeId      *string
	CustomerId        *string
	CheckoutSessionId *string
	OrderId           *string
	ExpiresAt         time.Time
	CreatedAt         time.Time
}

const couponReservationColumns = `org_id, id, coupon_id, coupon_code_id, customer_id, checkout_session_id, order_id, expires_at, created_at`

func (r *couponReservationRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.CouponId, &r.CouponCodeId, &r.CustomerId,
		&r.CheckoutSessionId, &r.OrderId, &r.ExpiresAt, &r.CreatedAt)
}

func (r couponReservationRow) toDomain() domain.CouponReservation {
	return domain.CouponReservation{
		OrgId:             r.OrgId,
		Id:                r.Id,
		CouponId:          r.CouponId,
		CouponCodeId:      strOrEmpty(r.CouponCodeId),
		CustomerId:        strOrEmpty(r.CustomerId),
		CheckoutSessionId: strOrEmpty(r.CheckoutSessionId),
		OrderId:           strOrEmpty(r.OrderId),
		ExpiresAt:         r.ExpiresAt,
		CreatedAt:         r.CreatedAt,
	}
}

func couponReservationRowFromDomain(c domain.CouponReservation) couponReservationRow {
	return couponReservationRow{
		OrgId:             c.OrgId,
		Id:                c.Id,
		CouponId:          c.CouponId,
		CouponCodeId:      nilIfEmpty(c.CouponCodeId),
		CustomerId:        nilIfEmpty(c.CustomerId),
		CheckoutSessionId: nilIfEmpty(c.CheckoutSessionId),
		OrderId:           nilIfEmpty(c.OrderId),
		ExpiresAt:         c.ExpiresAt,
		CreatedAt:         c.CreatedAt,
	}
}
