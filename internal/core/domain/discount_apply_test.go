package domain

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func appliedForever(c Coupon, redeemedAt time.Time) AppliedDiscount {
	return AppliedDiscount{
		Coupon:   c,
		Discount: Discount{StartCycle: 0, RedeemedAt: redeemedAt, Status: DiscountStatusActive},
	}
}

func TestApplyDiscounts_Percentage(t *testing.T) {
	c := Coupon{DiscountType: DiscountTypePercentage, PercentOff: decimal.NewFromInt(25), Duration: DurationForever}
	lines := []DiscountableLine{{LineId: "l1", ProductId: "prd_a", Total: 1000}}
	got := ApplyDiscounts(lines, []AppliedDiscount{appliedForever(c, time.Unix(1, 0))}, 0, "USD")
	assert.EqualValues(t, 250, got["l1"])
}

func TestApplyDiscounts_FixedClampsToBaseNoCarry(t *testing.T) {
	c := Coupon{DiscountType: DiscountTypeFixed, AmountOff: 5000, Currency: "USD", Duration: DurationOnce}
	lines := []DiscountableLine{{LineId: "l1", ProductId: "prd_a", Total: 1000}}
	got := ApplyDiscounts(lines, []AppliedDiscount{appliedForever(c, time.Unix(1, 0))}, 0, "USD")
	assert.EqualValues(t, 1000, got["l1"], "leftover is lost, never below zero")
}

func TestApplyDiscounts_ProductTargetingOnlyMatchingLines(t *testing.T) {
	c := Coupon{DiscountType: DiscountTypePercentage, PercentOff: decimal.NewFromInt(50), Duration: DurationForever, AppliesToProducts: []string{"prd_a"}}
	lines := []DiscountableLine{
		{LineId: "l1", ProductId: "prd_a", Total: 1000},
		{LineId: "l2", ProductId: "prd_b", Total: 1000},
	}
	got := ApplyDiscounts(lines, []AppliedDiscount{appliedForever(c, time.Unix(1, 0))}, 0, "USD")
	assert.EqualValues(t, 500, got["l1"])
	assert.EqualValues(t, 0, got["l2"])
}

func TestApplyDiscounts_StackingOrderedByRedeemedAt(t *testing.T) {
	pct := Coupon{DiscountType: DiscountTypePercentage, PercentOff: decimal.NewFromInt(50), Duration: DurationForever}
	fixed := Coupon{DiscountType: DiscountTypeFixed, AmountOff: 100, Currency: "USD", Duration: DurationForever}
	lines := []DiscountableLine{{LineId: "l1", ProductId: "prd_a", Total: 1000}}
	// fixed redeemed first (t=1), then pct (t=2): 1000-100=900, then 50% => 450; total 550.
	applied := []AppliedDiscount{
		{Coupon: pct, Discount: Discount{RedeemedAt: time.Unix(2, 0)}},
		{Coupon: fixed, Discount: Discount{RedeemedAt: time.Unix(1, 0)}},
	}
	got := ApplyDiscounts(lines, applied, 0, "USD")
	assert.EqualValues(t, 550, got["l1"])
}

func TestApplyDiscounts_StackingNeverBelowZero(t *testing.T) {
	a := Coupon{DiscountType: DiscountTypeFixed, AmountOff: 800, Currency: "USD", Duration: DurationForever}
	b := Coupon{DiscountType: DiscountTypeFixed, AmountOff: 800, Currency: "USD", Duration: DurationForever}
	lines := []DiscountableLine{{LineId: "l1", ProductId: "prd_a", Total: 1000}}
	applied := []AppliedDiscount{
		{Coupon: a, Discount: Discount{RedeemedAt: time.Unix(1, 0)}},
		{Coupon: b, Discount: Discount{RedeemedAt: time.Unix(2, 0)}},
	}
	got := ApplyDiscounts(lines, applied, 0, "USD")
	assert.EqualValues(t, 1000, got["l1"])
}

func TestApplyDiscounts_OutOfWindowSkipped(t *testing.T) {
	c := Coupon{DiscountType: DiscountTypePercentage, PercentOff: decimal.NewFromInt(50), Duration: DurationOnce}
	lines := []DiscountableLine{{LineId: "l1", ProductId: "prd_a", Total: 1000}}
	a := AppliedDiscount{Coupon: c, Discount: Discount{StartCycle: 0, RedeemedAt: time.Unix(1, 0)}}
	assert.EqualValues(t, 500, ApplyDiscounts(lines, []AppliedDiscount{a}, 0, "USD")["l1"])
	assert.EqualValues(t, 0, ApplyDiscounts(lines, []AppliedDiscount{a}, 1, "USD")["l1"], "once: only cycle 0")
}

func TestApplyDiscounts_RepeatingWindow(t *testing.T) {
	c := Coupon{DiscountType: DiscountTypePercentage, PercentOff: decimal.NewFromInt(10), Duration: DurationRepeating, DurationInCycles: 3}
	a := AppliedDiscount{Coupon: c, Discount: Discount{StartCycle: 2, RedeemedAt: time.Unix(1, 0)}}
	lines := []DiscountableLine{{LineId: "l1", ProductId: "prd_a", Total: 1000}}
	assert.EqualValues(t, 0, ApplyDiscounts(lines, []AppliedDiscount{a}, 1, "USD")["l1"])
	assert.EqualValues(t, 100, ApplyDiscounts(lines, []AppliedDiscount{a}, 2, "USD")["l1"])
	assert.EqualValues(t, 100, ApplyDiscounts(lines, []AppliedDiscount{a}, 4, "USD")["l1"])
	assert.EqualValues(t, 0, ApplyDiscounts(lines, []AppliedDiscount{a}, 5, "USD")["l1"])
}

func TestApplyDiscounts_FixedCurrencyMismatchSkipped(t *testing.T) {
	c := Coupon{DiscountType: DiscountTypeFixed, AmountOff: 100, Currency: "EUR", Duration: DurationForever}
	lines := []DiscountableLine{{LineId: "l1", ProductId: "prd_a", Total: 1000}}
	got := ApplyDiscounts(lines, []AppliedDiscount{appliedForever(c, time.Unix(1, 0))}, 0, "USD")
	assert.EqualValues(t, 0, got["l1"])
}

func TestApplyDiscounts_ProportionalAllocationSumsExact(t *testing.T) {
	c := Coupon{DiscountType: DiscountTypeFixed, AmountOff: 100, Currency: "USD", Duration: DurationForever}
	lines := []DiscountableLine{
		{LineId: "l1", ProductId: "prd_a", Total: 333},
		{LineId: "l2", ProductId: "prd_a", Total: 333},
		{LineId: "l3", ProductId: "prd_a", Total: 334},
	}
	got := ApplyDiscounts(lines, []AppliedDiscount{appliedForever(c, time.Unix(1, 0))}, 0, "USD")
	assert.EqualValues(t, 100, got["l1"]+got["l2"]+got["l3"], "allocations sum to the raw discount exactly")
}
