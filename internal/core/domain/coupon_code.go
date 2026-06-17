package domain

import (
	"strings"
	"time"

	"getpaidhq/internal/lib"
)

// Restrictions are per-code eligibility gates, stored as a JSON column.
type Restrictions struct {
	FirstTimeTransaction  bool   `json:"first_time_transaction"`
	MinimumAmount         int64  `json:"minimum_amount"`
	MinimumAmountCurrency string `json:"minimum_amount_currency"`
}

// CouponCode is the redeemable string for a Coupon (Stripe's PromotionCode).
type CouponCode struct {
	OrgId    string
	Id       string
	CouponId string
	Code     string

	Active   bool
	Metadata map[string]string

	CustomerId     string
	ExpiresAt      time.Time
	MaxRedemptions int // 0 = unlimited
	Restrictions   Restrictions

	TimesRedeemed int

	CreatedAt time.Time
	UpdatedAt time.Time
}

type NewCouponCodeInput struct {
	OrgId          string
	CouponId       string
	Code           string
	CustomerId     string
	ExpiresAt      time.Time
	MaxRedemptions int
	Restrictions   Restrictions
	Metadata       map[string]string
}

func NewCouponCode(in NewCouponCodeInput) (CouponCode, error) {
	if in.OrgId == "" || in.CouponId == "" || strings.TrimSpace(in.Code) == "" {
		return CouponCode{}, lib.NewCustomError(lib.BadRequestError, "coupon code requires org, coupon and code", nil)
	}
	if in.MaxRedemptions < 0 {
		return CouponCode{}, lib.NewCustomError(lib.BadRequestError, "max_redemptions must be >= 0", nil)
	}
	if in.Restrictions.MinimumAmount < 0 {
		return CouponCode{}, lib.NewCustomError(lib.BadRequestError, "minimum_amount must be >= 0", nil)
	}
	if in.Restrictions.MinimumAmount > 0 && len(in.Restrictions.MinimumAmountCurrency) != 3 {
		return CouponCode{}, lib.NewCustomError(lib.BadRequestError, "minimum_amount requires a 3-letter currency", nil)
	}
	return CouponCode{
		OrgId:          in.OrgId,
		Id:             lib.GenerateId("ccode"),
		CouponId:       in.CouponId,
		Code:           strings.ToUpper(strings.TrimSpace(in.Code)),
		Active:         true,
		Metadata:       in.Metadata,
		CustomerId:     in.CustomerId,
		ExpiresAt:      in.ExpiresAt,
		MaxRedemptions: in.MaxRedemptions,
		Restrictions:   in.Restrictions,
	}, nil
}
