package postgres

import (
	"time"

	"github.com/lib/pq"
	"github.com/shopspring/decimal"

	"getpaidhq/internal/core/domain"
)

type couponRow struct {
	OrgId    string            `gorm:"column:org_id;primaryKey"`
	Id       string            `gorm:"column:id;primaryKey"`
	Name     string            `gorm:"column:name"`
	Active   bool              `gorm:"column:active"`
	Metadata map[string]string `gorm:"column:metadata;serializer:json"`

	DiscountType domain.DiscountType `gorm:"column:discount_type"`
	AmountOff    *int64              `gorm:"column:amount_off"`
	Currency     *string             `gorm:"column:currency"`
	PercentOff   *decimal.Decimal    `gorm:"column:percent_off;type:numeric"`

	Duration         domain.Duration `gorm:"column:duration"`
	DurationInCycles *int            `gorm:"column:duration_in_cycles"`

	RedeemBy          time.Time      `gorm:"column:redeem_by;serializer:nulltime"`
	AppliesToProducts pq.StringArray `gorm:"column:applies_to_products;type:text[]"`
	MaxRedemptions    int            `gorm:"column:max_redemptions"`
	OncePerCustomer   bool           `gorm:"column:once_per_customer"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (couponRow) TableName() string { return "coupons" }

func (r couponRow) toDomain() domain.Coupon {
	c := domain.Coupon{
		OrgId:             r.OrgId,
		Id:                r.Id,
		Name:              r.Name,
		Active:            r.Active,
		Metadata:          r.Metadata,
		DiscountType:      r.DiscountType,
		Duration:          r.Duration,
		RedeemBy:          r.RedeemBy,
		AppliesToProducts: []string(r.AppliesToProducts),
		MaxRedemptions:    r.MaxRedemptions,
		OncePerCustomer:   r.OncePerCustomer,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
	if r.AmountOff != nil {
		c.AmountOff = *r.AmountOff
	}
	if r.Currency != nil {
		c.Currency = *r.Currency
	}
	if r.PercentOff != nil {
		c.PercentOff = *r.PercentOff
	}
	if r.DurationInCycles != nil {
		c.DurationInCycles = *r.DurationInCycles
	}
	return c
}

func couponRowFromDomain(c domain.Coupon) couponRow {
	r := couponRow{
		OrgId:             c.OrgId,
		Id:                c.Id,
		Name:              c.Name,
		Active:            c.Active,
		Metadata:          c.Metadata,
		DiscountType:      c.DiscountType,
		Duration:          c.Duration,
		RedeemBy:          c.RedeemBy,
		AppliesToProducts: pq.StringArray(c.AppliesToProducts),
		MaxRedemptions:    c.MaxRedemptions,
		OncePerCustomer:   c.OncePerCustomer,
		CreatedAt:         c.CreatedAt,
		UpdatedAt:         c.UpdatedAt,
	}
	if c.DiscountType == domain.DiscountTypeFixed {
		amt := c.AmountOff
		cur := c.Currency
		r.AmountOff = &amt
		r.Currency = &cur
	} else {
		pct := c.PercentOff
		r.PercentOff = &pct
	}
	if c.Duration == domain.DurationRepeating {
		cyc := c.DurationInCycles
		r.DurationInCycles = &cyc
	}
	return r
}
