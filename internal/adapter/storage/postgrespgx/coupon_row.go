package postgrespgx

import (
	"time"

	"github.com/shopspring/decimal"

	"getpaidhq/internal/core/domain"
)

// couponRow is the postgres on-the-wire shape of a Coupon.
//
// The discount-type-specific columns are nullable and mirror the gorm row's
// pointer fields: amount_off/currency are populated only for fixed coupons,
// percent_off only for percentage coupons, duration_in_cycles only for
// repeating coupons. percent_off is DECIMAL(5,2) and maps to a
// decimal.Decimal (which implements sql.Scanner/driver.Valuer) — held as a
// *decimal.Decimal so it can be NULL. applies_to_products is a native text[]
// column, scanned directly to/from a []string (no json serializer).
// redeem_by is a nullable timestamp (the gorm `serializer:nulltime` column).
type couponRow struct {
	OrgId    string
	Id       string
	Name     string
	Active   bool
	Metadata jsonCol[map[string]string]

	DiscountType string
	AmountOff    *int64
	Currency     *string
	PercentOff   *decimal.Decimal

	Duration         string
	DurationInCycles *int

	RedeemBy          *time.Time
	AppliesToProducts []string
	MaxRedemptions    int
	OncePerCustomer   bool

	CreatedAt time.Time
	UpdatedAt time.Time
}

const couponColumns = `org_id, id, name, active, metadata, discount_type, amount_off, currency, percent_off, duration, duration_in_cycles, redeem_by, applies_to_products, max_redemptions, once_per_customer, created_at, updated_at`

func (r *couponRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.Name, &r.Active, &r.Metadata, &r.DiscountType,
		&r.AmountOff, &r.Currency, &r.PercentOff, &r.Duration, &r.DurationInCycles,
		&r.RedeemBy, &r.AppliesToProducts, &r.MaxRedemptions, &r.OncePerCustomer,
		&r.CreatedAt, &r.UpdatedAt)
}

func (r couponRow) toDomain() domain.Coupon {
	c := domain.Coupon{
		OrgId:             r.OrgId,
		Id:                r.Id,
		Name:              r.Name,
		Active:            r.Active,
		Metadata:          r.Metadata.V,
		DiscountType:      domain.DiscountType(r.DiscountType),
		Duration:          domain.Duration(r.Duration),
		RedeemBy:          timeOrZero(r.RedeemBy),
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
		Metadata:          newJSON(c.Metadata),
		DiscountType:      string(c.DiscountType),
		Duration:          string(c.Duration),
		RedeemBy:          nullTime(c.RedeemBy),
		AppliesToProducts: c.AppliesToProducts,
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
