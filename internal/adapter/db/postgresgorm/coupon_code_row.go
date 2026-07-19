package postgresgorm

import (
	"time"

	"getpaidhq/internal/core/domain"
)

type couponCodeRow struct {
	OrgId    string            `gorm:"column:org_id;primaryKey"`
	Id       string            `gorm:"column:id;primaryKey"`
	CouponId string            `gorm:"column:coupon_id"`
	Code     string            `gorm:"column:code"`
	Active   bool              `gorm:"column:active"`
	Metadata map[string]string `gorm:"column:metadata;serializer:json"`

	CustomerId     *string             `gorm:"column:customer_id"`
	ExpiresAt      time.Time           `gorm:"column:expires_at;serializer:nulltime"`
	MaxRedemptions int                 `gorm:"column:max_redemptions"`
	TimesRedeemed  int                 `gorm:"column:times_redeemed"`
	Restrictions   domain.Restrictions `gorm:"column:restrictions;serializer:json"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (couponCodeRow) TableName() string { return "coupon_codes" }

func (r couponCodeRow) toDomain() domain.CouponCode {
	return domain.CouponCode{
		OrgId:          r.OrgId,
		Id:             r.Id,
		CouponId:       r.CouponId,
		Code:           r.Code,
		Active:         r.Active,
		Metadata:       r.Metadata,
		CustomerId:     strOrEmpty(r.CustomerId),
		ExpiresAt:      r.ExpiresAt,
		MaxRedemptions: r.MaxRedemptions,
		TimesRedeemed:  r.TimesRedeemed,
		Restrictions:   r.Restrictions,
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
		Metadata:       c.Metadata,
		CustomerId:     nilIfEmpty(c.CustomerId),
		ExpiresAt:      c.ExpiresAt,
		MaxRedemptions: c.MaxRedemptions,
		TimesRedeemed:  c.TimesRedeemed,
		Restrictions:   c.Restrictions,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}
