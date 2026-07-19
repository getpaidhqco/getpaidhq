package postgresgorm

import (
	"time"

	"getpaidhq/internal/core/domain"
)

type discountRow struct {
	OrgId          string                `gorm:"column:org_id;primaryKey"`
	Id             string                `gorm:"column:id;primaryKey"`
	CouponId       string                `gorm:"column:coupon_id"`
	CouponCodeId   *string               `gorm:"column:coupon_code_id"`
	CustomerId     string                `gorm:"column:customer_id"`
	SubscriptionId *string               `gorm:"column:subscription_id"`
	OrderId        *string               `gorm:"column:order_id"`
	StartCycle     int                   `gorm:"column:start_cycle"`
	Status         domain.DiscountStatus `gorm:"column:status"`
	RedeemedAt     time.Time             `gorm:"column:redeemed_at"`
	EndedAt        time.Time             `gorm:"column:ended_at;serializer:nulltime"`
	Metadata       map[string]string     `gorm:"column:metadata;serializer:json"`
	CreatedAt      time.Time             `gorm:"column:created_at"`
	UpdatedAt      time.Time             `gorm:"column:updated_at"`
}

func (discountRow) TableName() string { return "discounts" }

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
		Status:         r.Status,
		RedeemedAt:     r.RedeemedAt,
		EndedAt:        r.EndedAt,
		Metadata:       r.Metadata,
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
		Status:         d.Status,
		RedeemedAt:     d.RedeemedAt,
		EndedAt:        d.EndedAt,
		Metadata:       d.Metadata,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}
}
