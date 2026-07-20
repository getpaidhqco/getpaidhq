package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// discountRow is the postgres on-the-wire shape of a Discount.
//
// coupon_code_id, subscription_id and order_id are nullable FKs (NULL, never
// ""); customer_id is NOT NULL. status is the DiscountStatus enum held as a
// string at the boundary. redeemed_at is NOT NULL; ended_at is a nullable
// timestamp (zero domain time maps to NULL and back). metadata is a nullable
// JSONB column carried via jsonCol (no emptyIfNil, so a nil map serialises to
// JSON null).
type discountRow struct {
	OrgId          string
	Id             string
	CouponId       string
	CouponCodeId   *string
	CustomerId     string
	SubscriptionId *string
	OrderId        *string
	StartCycle     int
	Status         string
	RedeemedAt     time.Time
	EndedAt        *time.Time
	Metadata       jsonCol[map[string]string]
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

const discountColumns = `org_id, id, coupon_id, coupon_code_id, customer_id, subscription_id, order_id, start_cycle, status, redeemed_at, ended_at, metadata, created_at, updated_at`

func (r *discountRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.CouponId, &r.CouponCodeId, &r.CustomerId,
		&r.SubscriptionId, &r.OrderId, &r.StartCycle, &r.Status, &r.RedeemedAt,
		&r.EndedAt, &r.Metadata, &r.CreatedAt, &r.UpdatedAt)
}

func (r discountRow) toDomain() domain.Discount {
	return domain.Discount{
		OrgId:          r.OrgId,
		Id:             r.Id,
		CouponId:       r.CouponId,
		CouponCodeId:   strOrEmpty(r.CouponCodeId),
		CustomerId:     r.CustomerId,
		SubscriptionId: strOrEmpty(r.SubscriptionId),
		OrderId:        strOrEmpty(r.OrderId),
		StartCycle:     r.StartCycle,
		Status:         domain.DiscountStatus(r.Status),
		RedeemedAt:     r.RedeemedAt,
		EndedAt:        timeOrZero(r.EndedAt),
		Metadata:       r.Metadata.V,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func discountRowFromDomain(d domain.Discount) discountRow {
	return discountRow{
		OrgId:          d.OrgId,
		Id:             d.Id,
		CouponId:       d.CouponId,
		CouponCodeId:   nilIfEmpty(d.CouponCodeId),
		CustomerId:     d.CustomerId,
		SubscriptionId: nilIfEmpty(d.SubscriptionId),
		OrderId:        nilIfEmpty(d.OrderId),
		StartCycle:     d.StartCycle,
		Status:         string(d.Status),
		RedeemedAt:     d.RedeemedAt,
		EndedAt:        nullTime(d.EndedAt),
		Metadata:       newJSON(d.Metadata),
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}
}
