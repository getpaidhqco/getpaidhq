package domain

import (
	"time"

	"github.com/shopspring/decimal"

	"getpaidhq/internal/lib"
)

type DiscountType string

const (
	DiscountTypePercentage DiscountType = "percentage"
	DiscountTypeFixed      DiscountType = "fixed"
)

type Duration string

const (
	DurationOnce      Duration = "once"
	DurationRepeating Duration = "repeating"
	DurationForever   Duration = "forever"
)

// Coupon is the immutable discount definition. Only Name, Active and Metadata
// may change after creation (see UpdateMutable / the DB immutability trigger).
type Coupon struct {
	OrgId string
	Id    string

	Name     string
	Active   bool
	Metadata map[string]string

	DiscountType DiscountType
	PercentOff   decimal.Decimal // percentage type: 0 < p <= 100
	AmountOff    int64           // fixed type: > 0, minor units
	Currency     string          // fixed type: ISO-4217

	Duration         Duration
	DurationInCycles int // repeating only, >= 1

	RedeemBy          time.Time
	AppliesToProducts []string
	MaxRedemptions    int // 0 = unlimited
	OncePerCustomer   bool

	CreatedAt time.Time
	UpdatedAt time.Time
}

type NewCouponInput struct {
	OrgId             string
	Name              string
	DiscountType      DiscountType
	PercentOff        decimal.Decimal
	AmountOff         int64
	Currency          string
	Duration          Duration
	DurationInCycles  int
	RedeemBy          time.Time
	AppliesToProducts []string
	MaxRedemptions    int
	OncePerCustomer   bool
	Metadata          map[string]string
}

func NewCoupon(in NewCouponInput) (Coupon, error) {
	c := Coupon{
		OrgId:             in.OrgId,
		Id:                lib.GenerateId("coupon"),
		Name:              in.Name,
		Active:            true,
		Metadata:          in.Metadata,
		DiscountType:      in.DiscountType,
		PercentOff:        in.PercentOff,
		AmountOff:         in.AmountOff,
		Currency:          in.Currency,
		Duration:          in.Duration,
		DurationInCycles:  in.DurationInCycles,
		RedeemBy:          in.RedeemBy,
		AppliesToProducts: in.AppliesToProducts,
		MaxRedemptions:    in.MaxRedemptions,
		OncePerCustomer:   in.OncePerCustomer,
	}
	if err := c.validate(); err != nil {
		return Coupon{}, err
	}
	return c, nil
}

func (c Coupon) validate() error {
	if c.OrgId == "" || c.Name == "" {
		return lib.NewCustomError(lib.BadRequestError, "coupon requires org and name", nil)
	}
	switch c.DiscountType {
	case DiscountTypePercentage:
		if c.AmountOff != 0 || c.Currency != "" {
			return lib.NewCustomError(lib.BadRequestError, "percentage coupon must not set amount_off/currency", nil)
		}
		if c.PercentOff.LessThanOrEqual(decimal.Zero) || c.PercentOff.GreaterThan(decimal.NewFromInt(100)) {
			return lib.NewCustomError(lib.BadRequestError, "percent_off must be in (0,100]", nil)
		}
	case DiscountTypeFixed:
		if !c.PercentOff.IsZero() {
			return lib.NewCustomError(lib.BadRequestError, "fixed coupon must not set percent_off", nil)
		}
		if c.AmountOff <= 0 {
			return lib.NewCustomError(lib.BadRequestError, "amount_off must be > 0", nil)
		}
		if len(c.Currency) != 3 {
			return lib.NewCustomError(lib.BadRequestError, "fixed coupon requires a 3-letter currency", nil)
		}
	default:
		return lib.NewCustomError(lib.BadRequestError, "discount_type must be percentage or fixed", nil)
	}
	switch c.Duration {
	case DurationRepeating:
		if c.DurationInCycles < 1 {
			return lib.NewCustomError(lib.BadRequestError, "repeating coupon requires duration_in_cycles >= 1", nil)
		}
	case DurationOnce, DurationForever:
		if c.DurationInCycles != 0 {
			return lib.NewCustomError(lib.BadRequestError, "duration_in_cycles only valid for repeating", nil)
		}
	default:
		return lib.NewCustomError(lib.BadRequestError, "duration must be once, repeating or forever", nil)
	}
	if c.MaxRedemptions < 0 {
		return lib.NewCustomError(lib.BadRequestError, "max_redemptions must be >= 0", nil)
	}
	return nil
}

func (c *Coupon) Rename(name string)              { c.Name = name }
func (c *Coupon) SetActive(active bool)           { c.Active = active }
func (c *Coupon) SetMetadata(m map[string]string) { c.Metadata = m }

func (c Coupon) appliesTo(productId string) bool {
	if len(c.AppliesToProducts) == 0 {
		return true
	}
	for _, p := range c.AppliesToProducts {
		if p == productId {
			return true
		}
	}
	return false
}
