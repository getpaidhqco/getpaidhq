package postgresgorm

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// couponReservationRow is the postgres on-the-wire shape of a CouponReservation.
//
// coupon_code_id, customer_id, checkout_session_id and order_id are nullable FK
// columns (NULL, never ""), carried as *string and written via nilIfEmpty —
// mirroring customerRow.ExternalId.
type couponReservationRow struct {
	OrgId             string    `gorm:"column:org_id;primaryKey"`
	Id                string    `gorm:"column:id;primaryKey"`
	CouponId          string    `gorm:"column:coupon_id"`
	CouponCodeId      *string   `gorm:"column:coupon_code_id"`
	CustomerId        *string   `gorm:"column:customer_id"`
	CheckoutSessionId *string   `gorm:"column:checkout_session_id"`
	OrderId           *string   `gorm:"column:order_id"`
	ExpiresAt         time.Time `gorm:"column:expires_at"`
	CreatedAt         time.Time `gorm:"column:created_at"`
}

func (couponReservationRow) TableName() string { return "coupon_reservations" }

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
