package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// couponCodeRow is the postgres on-the-wire shape of a CouponCode.
//
// customer_id is a nullable FK (NULL, never ""). expires_at is a nullable
// timestamp (the gorm `serializer:nulltime` column). metadata and restrictions
// are nullable JSONB columns carried via jsonCol (the gorm `serializer:json`
// columns) — no emptyIfNil, matching the gorm row.
type couponCodeRow struct {
	OrgId    string
	Id       string
	CouponId string
	Code     string
	Active   bool
	Metadata jsonCol[map[string]string]

	CustomerId     *string
	ExpiresAt      *time.Time
	MaxRedemptions int
	TimesRedeemed  int
	Restrictions   jsonCol[domain.Restrictions]

	CreatedAt time.Time
	UpdatedAt time.Time
}

const couponCodeColumns = `org_id, id, coupon_id, code, active, metadata, customer_id, expires_at, max_redemptions, times_redeemed, restrictions, created_at, updated_at`

func (r *couponCodeRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.CouponId, &r.Code, &r.Active, &r.Metadata,
		&r.CustomerId, &r.ExpiresAt, &r.MaxRedemptions, &r.TimesRedeemed, &r.Restrictions,
		&r.CreatedAt, &r.UpdatedAt)
}

func (r couponCodeRow) toDomain() domain.CouponCode {
	return domain.CouponCode{
		OrgId:          r.OrgId,
		Id:             r.Id,
		CouponId:       r.CouponId,
		Code:           r.Code,
		Active:         r.Active,
		Metadata:       r.Metadata.V,
		CustomerId:     strOrEmpty(r.CustomerId),
		ExpiresAt:      timeOrZero(r.ExpiresAt),
		MaxRedemptions: r.MaxRedemptions,
		TimesRedeemed:  r.TimesRedeemed,
		Restrictions:   r.Restrictions.V,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func couponCodeRowFromDomain(c domain.CouponCode) couponCodeRow {
	return couponCodeRow{
		OrgId:          c.OrgId,
		Id:             c.Id,
		CouponId:       c.CouponId,
		Code:           c.Code,
		Active:         c.Active,
		Metadata:       newJSON(c.Metadata),
		CustomerId:     nilIfEmpty(c.CustomerId),
		ExpiresAt:      nullTime(c.ExpiresAt),
		MaxRedemptions: c.MaxRedemptions,
		TimesRedeemed:  c.TimesRedeemed,
		Restrictions:   newJSON(c.Restrictions),
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}
