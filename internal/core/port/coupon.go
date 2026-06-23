package port

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"getpaidhq/internal/core/domain"
)

type CouponRepository interface {
	Create(ctx context.Context, coupon domain.Coupon) (domain.Coupon, error)
	// UpdateMutable persists ONLY name, active and metadata — terms are immutable.
	UpdateMutable(ctx context.Context, orgId, id, name string, active bool, metadata map[string]string) (domain.Coupon, error)
	FindById(ctx context.Context, orgId, id string) (domain.Coupon, error)
	FindByIdForUpdate(ctx context.Context, orgId, id string) (domain.Coupon, error) // SELECT ... FOR UPDATE
	Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Coupon, int, error)
	DeleteIfUnreferenced(ctx context.Context, orgId, id string) error
}

type CouponCodeRepository interface {
	Create(ctx context.Context, code domain.CouponCode) (domain.CouponCode, error)
	UpdateMutable(ctx context.Context, orgId, id string, active bool, metadata map[string]string) (domain.CouponCode, error)
	IncrementRedeemed(ctx context.Context, orgId, id string) error
	FindByCode(ctx context.Context, orgId, code string) (domain.CouponCode, error)          // case-insensitive
	FindByCodeForUpdate(ctx context.Context, orgId, code string) (domain.CouponCode, error) // SELECT ... FOR UPDATE, case-insensitive
	FindByCouponId(ctx context.Context, orgId, couponId string) ([]domain.CouponCode, error)
}

type DiscountRepository interface {
	Create(ctx context.Context, discount domain.Discount) (domain.Discount, error)
	Update(ctx context.Context, discount domain.Discount) (domain.Discount, error)
	FindById(ctx context.Context, orgId, id string) (domain.Discount, error)
	ActiveForSubscription(ctx context.Context, orgId, subscriptionId string) ([]domain.Discount, error)
	ActiveForOrder(ctx context.Context, orgId, orderId string) ([]domain.Discount, error)
	CountByCoupon(ctx context.Context, orgId, couponId string) (int, error)
	CountByCouponAndCustomer(ctx context.Context, orgId, couponId, customerId string) (int, error)
}

// CouponReservationRepository persists ephemeral capacity holds (build-now: order-held).
type CouponReservationRepository interface {
	Create(ctx context.Context, r domain.CouponReservation) (domain.CouponReservation, error)
	FindByOrder(ctx context.Context, orgId, orderId string) ([]domain.CouponReservation, error)
	DeleteByOrder(ctx context.Context, orgId, orderId string) error
	CountLiveByCoupon(ctx context.Context, orgId, couponId string, now time.Time) (int, error)
	CountLiveByCode(ctx context.Context, orgId, couponCodeId string, now time.Time) (int, error)
	ExistsLiveForCustomer(ctx context.Context, orgId, couponId, customerId string, now time.Time) (bool, error)
	DeleteExpired(ctx context.Context, now time.Time) (int, error)
}

// PriorPaymentChecker backs the FirstTimeTransaction restriction.
type PriorPaymentChecker interface {
	HasPriorSuccessfulPayment(ctx context.Context, orgId, customerId string) (bool, error)
}

// ----- service input DTOs (also used as Fuego request bodies) -----

type CreateCouponInput struct {
	Name              string            `json:"name" validate:"required"`
	DiscountType      string            `json:"discount_type" validate:"required,oneof=percentage fixed"`
	PercentOff        decimal.Decimal   `json:"percent_off"`
	AmountOff         int64             `json:"amount_off"`
	Currency          string            `json:"currency"`
	Duration          string            `json:"duration" validate:"required,oneof=once repeating forever"`
	DurationInCycles  int               `json:"duration_in_cycles"`
	RedeemBy          time.Time         `json:"redeem_by"`
	AppliesToProducts []string          `json:"applies_to_products"`
	MaxRedemptions    int               `json:"max_redemptions"`
	OncePerCustomer   bool              `json:"once_per_customer"`
	Metadata          map[string]string `json:"metadata"`
}

type UpdateCouponInput struct {
	Name     string            `json:"name" validate:"required"`
	Active   bool              `json:"active"`
	Metadata map[string]string `json:"metadata"`
}

type CreateCouponCodeInput struct {
	Code           string              `json:"code" validate:"required"`
	CustomerId     string              `json:"customer_id"`
	ExpiresAt      time.Time           `json:"expires_at"`
	MaxRedemptions int                 `json:"max_redemptions"`
	Restrictions   domain.Restrictions `json:"restrictions"`
	Metadata       map[string]string   `json:"metadata"`
}

type UpdateCouponCodeInput struct {
	Active   bool              `json:"active"`
	Metadata map[string]string `json:"metadata"`
}

type RedeemCouponInput struct {
	OrgId          string
	Code           string // empty => programmatic; use CouponId
	CouponId       string
	CustomerId     string
	SubscriptionId string
	OrderId        string
	StartCycle     int
	Amount         int64 // for the minimum-amount re-check (0 = skip)
	Currency       string
}
