package domain

import (
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

	SubscriptionId string // exactly one of these two is set
	OrderId        string

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
	if in.OrgId == "" || in.CouponId == "" || in.CustomerId == "" {
		return Discount{}, lib.NewCustomError(lib.BadRequestError, "discount requires org, coupon and customer", nil)
	}
	hasSub := in.SubscriptionId != ""
	hasOrder := in.OrderId != ""
	if hasSub == hasOrder { // both set or neither set
		return Discount{}, lib.NewCustomError(lib.BadRequestError, "discount needs exactly one of subscription or order", nil)
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
		Id:             lib.GenerateId("disc"),
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
