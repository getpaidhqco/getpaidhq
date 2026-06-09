package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// monthlyPrice / monthlyItem build a Price + OrderItem pair for testing.
// The Price was previously embedded on OrderItem.Price; after the hexagonal
// split it's a separate aggregate passed alongside the item.
func monthlyPrice(trial BillingInterval, trialQty, cycles int) Price {
	return Price{
		BillingInterval:    BillingIntervalMonth,
		BillingIntervalQty: 1,
		Category:           PriceCategorySubscription,
		Currency:           "USD",
		UnitPrice:          1000,
		TrialInterval:      trial,
		TrialIntervalQty:   trialQty,
		Cycles:             cycles,
	}
}

func monthlyItem() OrderItem {
	return OrderItem{
		OrgId:   "org_1",
		OrderId: "order_1",
	}
}

// subForItem builds a pending subscription from a single line's price; the
// subscription derives its cadence + trial from that price.
func subForItem(item OrderItem, price Price) Subscription {
	return NewSubscriptionFromLines(item.OrgId, item.OrderId, "", []Price{price})
}

func TestSetActivationDates_NoTrialNoCycles(t *testing.T) {
	now := time.Now().UTC()
	price := monthlyPrice(BillingIntervalNone, 0, 0)
	s := subForItem(monthlyItem(), price)

	got := s.SetActivationDates()

	assert.Same(t, &s, got)
	assert.WithinDuration(t, now, s.StartDate, 5*time.Second)
	assert.Equal(t, s.StartDate, s.RenewsAt)
	assert.Equal(t, s.StartDate, s.CurrentPeriodStart)
	assert.Equal(t, s.RenewsAt, s.CurrentPeriodEnd)
	assert.Equal(t, s.StartDate.Day(), s.BillingAnchor)
	assert.True(t, s.TrialEndsAt.IsZero(), "no trial -> TrialEndsAt zero")
	assert.True(t, s.EndsAt.IsZero(), "no cycles cap -> EndsAt zero")
}

func TestSetActivationDates_WithTrialSetsTrialEnd(t *testing.T) {
	price := monthlyPrice(BillingIntervalMonth, 1, 0)
	s := subForItem(monthlyItem(), price)

	s.SetActivationDates()

	assert.False(t, s.TrialEndsAt.IsZero(), "trial set -> TrialEndsAt populated")
	assert.Equal(t, s.StartDate.AddDate(0, 1, 0), s.TrialEndsAt)
}

func TestSetActivationDates_WithCyclesSetsEndsAt(t *testing.T) {
	price := monthlyPrice(BillingIntervalNone, 0, 12)
	s := subForItem(monthlyItem(), price)

	s.SetActivationDates()

	assert.False(t, s.EndsAt.IsZero(), "cycles>0 -> EndsAt populated")
	assert.Equal(t, s.StartDate.AddDate(0, 12, 0), s.EndsAt)
}

func TestSetActive_FirstCycle(t *testing.T) {
	now := time.Now().UTC()
	price := monthlyPrice(BillingIntervalNone, 0, 0)
	s := subForItem(monthlyItem(), price)

	payment := Payment{
		OrgId:       "org_1",
		Amount:      1000,
		CompletedAt: now,
	}

	got := s.SetActive(payment)

	assert.Same(t, &s, got)
	assert.Equal(t, SubscriptionStatusActive, s.Status)
	assert.Equal(t, int64(1000), s.TotalRevenue)
	assert.Equal(t, 1, s.CyclesProcessed, "first successful charge increments to 1")
	assert.Equal(t, now, s.LastCharge)
	assert.Equal(t, s.StartDate.AddDate(0, 1, 0), s.RenewsAt)
	assert.Equal(t, s.StartDate, s.CurrentPeriodStart)
	assert.Equal(t, s.RenewsAt, s.CurrentPeriodEnd)
}

func TestSetActive_RecurringChargeIncrementsCycles(t *testing.T) {
	now := time.Now().UTC()
	price := monthlyPrice(BillingIntervalNone, 0, 0)
	s := subForItem(monthlyItem(), price)
	s.CyclesProcessed = 1
	s.LastCharge = now.AddDate(0, -1, 0)

	payment := Payment{OrgId: "org_1", Amount: 1000, CompletedAt: now}

	s.SetActive(payment)

	assert.Equal(t, SubscriptionStatusActive, s.Status)
	assert.Equal(t, 2, s.CyclesProcessed, "recurring charge increments cycles")
	assert.Equal(t, int64(1000), s.TotalRevenue)
	assert.Equal(t, now, s.LastCharge)
	assert.Equal(t, s.CurrentPeriodEnd, s.RenewsAt)
	assert.Equal(t, s.CurrentPeriodStart, s.StartDate)
	assert.False(t, s.RenewsAt.IsZero(), "RenewsAt must not be the zero time")
	assert.Greater(t, s.RenewsAt.Year(), 2000, "RenewsAt must be a real date, not year 0001")
	assert.True(t, s.RenewsAt.After(s.StartDate), "recurring charge must advance RenewsAt past StartDate")
}

func TestSetActive_NoPaymentDoesNotChargeButActivates(t *testing.T) {
	price := monthlyPrice(BillingIntervalNone, 0, 0)
	s := subForItem(monthlyItem(), price)

	s.SetActive(Payment{})

	assert.Equal(t, SubscriptionStatusActive, s.Status)
	assert.Equal(t, 0, s.CyclesProcessed, "no payment -> cycles unchanged")
	assert.Equal(t, int64(0), s.TotalRevenue)
	assert.True(t, s.LastCharge.IsZero())
	assert.Equal(t, s.StartDate, s.RenewsAt)
}

func TestSetActive_ZeroAmountSkipsChargeBranch(t *testing.T) {
	price := monthlyPrice(BillingIntervalNone, 0, 0)
	s := subForItem(monthlyItem(), price)

	s.SetActive(Payment{OrgId: "org_1", Amount: 0})

	assert.Equal(t, SubscriptionStatusActive, s.Status)
	assert.Equal(t, 0, s.CyclesProcessed)
	assert.Equal(t, int64(0), s.TotalRevenue)
}

func TestUpdateBillingAnchor_NoneMode(t *testing.T) {
	now := time.Now().UTC()
	s := Subscription{
		BillingInterval:    BillingIntervalMonth,
		BillingIntervalQty: 1,
		BillingAnchor:      15,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
	}

	d := s.UpdateBillingAnchor(20, "none", 1000)

	assert.Equal(t, 20, s.BillingAnchor, "anchor updated on the subscription")
	assert.Equal(t, 15, d.OldBillingAnchor)
	assert.Equal(t, 20, d.NewBillingAnchor)
	assert.Equal(t, 0, d.CreditAmount, "none mode -> no credit")
	assert.Equal(t, 0, d.DaysCredited)
	assert.Equal(t, s.AddBillingInterval(d.NewPeriodStart), d.NewPeriodEnd)
	assert.Equal(t, d.NewPeriodStart, s.CurrentPeriodStart)
	assert.Equal(t, d.NewPeriodEnd, s.CurrentPeriodEnd)
	assert.Equal(t, d.NewPeriodEnd, s.RenewsAt)
	assert.Equal(t, min(20, daysInMonth(d.NewPeriodStart)), d.NewPeriodStart.Day())
	assert.False(t, d.NewPeriodStart.Before(now), "next billing is rolled forward past now")
}

func TestUpdateBillingAnchor_CreditUnusedMode(t *testing.T) {
	now := time.Now().UTC()
	s := Subscription{
		BillingInterval:    BillingIntervalMonth,
		BillingIntervalQty: 1,
		BillingAnchor:      15,
		CurrentPeriodStart: now.AddDate(0, 0, -5),
		CurrentPeriodEnd:   now.AddDate(0, 0, 25),
	}

	d := s.UpdateBillingAnchor(25, "credit_unused", 1000)

	assert.Equal(t, 25, s.BillingAnchor)
	assert.Equal(t, 15, d.OldBillingAnchor)
	assert.Equal(t, 25, d.NewBillingAnchor)
	assert.Greater(t, d.CreditAmount, 0, "credit_unused with days remaining -> positive credit")
	assert.Greater(t, d.DaysCredited, 0)
}

// daysInMonth returns the number of days in the month of t (UTC).
func daysInMonth(t time.Time) int {
	y, m, _ := t.Date()
	return time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
