package domain

import (
	"getpaidhq/internal/lib/ids"
	"time"

	"getpaidhq/internal/lib"
)

type DiscountStatus string

const (
	DiscountStatusActive    DiscountStatus = "active"
	DiscountStatusCompleted DiscountStatus = "completed"
	DiscountStatusCancelled DiscountStatus = "cancelled"
)

// Discount is a redeemed Coupon recorded against a subscription or order.
// It holds no snapshot — the Coupon is immutable, so its terms are read live.
type Discount struct {
	OrgId        string
	Id           string
	CouponId     string
	CouponCodeId string
	CustomerId   string

	OrderId        string // always set — the order owns the discount
	SubscriptionId string // set when the discount targets a subscription's recurring invoices

	StartCycle int
	Status     DiscountStatus
	RedeemedAt time.Time
	EndedAt    time.Time

	Metadata  map[string]string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type NewDiscountInput struct {
	OrgId          string
	CouponId       string
	CouponCodeId   string
	CustomerId     string
	SubscriptionId string
	OrderId        string
	StartCycle     int
	RedeemedAt     time.Time
	Metadata       map[string]string
}

func NewDiscount(in NewDiscountInput) (Discount, error) {
	if in.OrgId == "" || in.CouponId == "" || in.CustomerId == "" || in.OrderId == "" {
		return Discount{}, lib.NewCustomError(lib.BadRequestError, "discount requires org, coupon, customer and order", nil)
	}
	if in.StartCycle < 0 {
		return Discount{}, lib.NewCustomError(lib.BadRequestError, "start_cycle must be >= 0", nil)
	}
	redeemedAt := in.RedeemedAt
	if redeemedAt.IsZero() {
		redeemedAt = time.Now().UTC()
	}
	return Discount{
		OrgId:          in.OrgId,
		Id:             ids.Generate("disc"),
		CouponId:       in.CouponId,
		CouponCodeId:   in.CouponCodeId,
		CustomerId:     in.CustomerId,
		SubscriptionId: in.SubscriptionId,
		OrderId:        in.OrderId,
		StartCycle:     in.StartCycle,
		Status:         DiscountStatusActive,
		RedeemedAt:     redeemedAt,
		Metadata:       in.Metadata,
	}, nil
}
