package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// base is a fixed reference instant used across these tests so no assertion
// ever depends on time.Now(). 2025-03-15 12:30:45 UTC.
var billingBase = time.Date(2025, time.March, 15, 12, 30, 45, 0, time.UTC)

func TestCalculateNextBillingDate_AllIntervals(t *testing.T) {
	// CurrentPeriodEnd is the base for the recurring branch. We force the
	// recurring branch by setting LastCharge + CyclesProcessed so we exercise
	// the per-interval arithmetic rather than the first-cycle shortcut.
	tests := []struct {
		name     string
		interval BillingInterval
		qty      int
		want     time.Time
	}{
		{"minute", BillingIntervalMinute, 5, billingBase.Add(5 * time.Minute)},
		{"hour", BillingIntervalHour(), 3, billingBase.Add(3 * time.Hour)},
		{"day", BillingIntervalDay, 2, billingBase.AddDate(0, 0, 2)},
		{"week", BillingIntervalWeek, 2, billingBase.AddDate(0, 0, 14)},
		{"month", BillingIntervalMonth, 1, billingBase.AddDate(0, 1, 0)},
		{"year", BillingIntervalYear, 1, billingBase.AddDate(1, 0, 0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Subscription{
				LastCharge:         billingBase,
				CyclesProcessed:    1,
				CurrentPeriodEnd:   billingBase,
				BillingInterval:    tt.interval,
				BillingIntervalQty: tt.qty,
			}
			assert.Equal(t, tt.want, s.CalculateNextBillingDate())
		})
	}
}

// BillingIntervalHour returns the "hour" interval. The domain constants do not
// declare an hour constant but CalculateNextBillingDate handles the raw "hour"
// string, so we use it directly here.
func BillingIntervalHour() BillingInterval { return BillingInterval("hour") }

func TestCalculateNextBillingDate_ZeroGuards(t *testing.T) {
	tests := []struct {
		name     string
		interval BillingInterval
		qty      int
	}{
		{"empty interval returns zero time", "", 1},
		{"qty zero returns zero time", BillingIntervalMonth, 0},
		{"qty negative returns zero time", BillingIntervalMonth, -3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Subscription{
				LastCharge:         billingBase,
				CyclesProcessed:    1,
				CurrentPeriodEnd:   billingBase,
				BillingInterval:    tt.interval,
				BillingIntervalQty: tt.qty,
			}
			assert.True(t, s.CalculateNextBillingDate().IsZero())
		})
	}
}

func TestCalculateNextBillingDate_FirstCycleVsRecurring(t *testing.T) {
	t.Run("first cycle (no charge, no cycles) returns StartDate", func(t *testing.T) {
		s := Subscription{
			StartDate:          billingBase,
			CurrentPeriodEnd:   billingBase.AddDate(0, 5, 0), // should be ignored
			BillingInterval:    BillingIntervalMonth,
			BillingIntervalQty: 1,
		}
		assert.Equal(t, billingBase, s.CalculateNextBillingDate())
	})

	t.Run("recurring when LastCharge set advances from CurrentPeriodEnd", func(t *testing.T) {
		s := Subscription{
			StartDate:          billingBase.AddDate(0, -1, 0),
			LastCharge:         billingBase,
			CyclesProcessed:    0, // LastCharge non-zero is enough to leave first-cycle branch
			CurrentPeriodEnd:   billingBase,
			BillingInterval:    BillingIntervalMonth,
			BillingIntervalQty: 1,
		}
		assert.Equal(t, billingBase.AddDate(0, 1, 0), s.CalculateNextBillingDate())
	})

	t.Run("recurring when CyclesProcessed>0 advances from CurrentPeriodEnd", func(t *testing.T) {
		s := Subscription{
			StartDate:          billingBase.AddDate(0, -1, 0),
			CyclesProcessed:    2, // CyclesProcessed non-zero is enough to leave first-cycle branch
			CurrentPeriodEnd:   billingBase,
			BillingInterval:    BillingIntervalMonth,
			BillingIntervalQty: 1,
		}
		assert.Equal(t, billingBase.AddDate(0, 1, 0), s.CalculateNextBillingDate())
	})
}

func TestAddBillingInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval BillingInterval
		qty      int
		want     time.Time
	}{
		{"minute", BillingIntervalMinute, 10, billingBase.Add(10 * time.Minute)},
		{"hour", BillingIntervalHour(), 6, billingBase.Add(6 * time.Hour)},
		{"day", BillingIntervalDay, 3, billingBase.AddDate(0, 0, 3)},
		{"week", BillingIntervalWeek, 1, billingBase.AddDate(0, 0, 7)},
		{"month", BillingIntervalMonth, 2, billingBase.AddDate(0, 2, 0)},
		{"year", BillingIntervalYear, 3, billingBase.AddDate(3, 0, 0)},
		{"unknown interval returns zero", BillingInterval("fortnight"), 1, time.Time{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Subscription{BillingInterval: tt.interval, BillingIntervalQty: tt.qty}
			assert.Equal(t, tt.want, s.AddBillingInterval(billingBase))
		})
	}
}

func TestGetNextChargeDate(t *testing.T) {
	renews := billingBase.AddDate(0, 1, 0)
	retry := billingBase.AddDate(0, 0, 3)

	tests := []struct {
		name   string
		status SubscriptionStatus
		renews time.Time
		retry  time.Time
		want   time.Time
	}{
		{
			name:   "past due returns NextRetryAt",
			status: SubscriptionStatusPastDue,
			renews: renews,
			retry:  retry,
			want:   retry,
		},
		{
			name:   "active returns RenewsAt",
			status: SubscriptionStatusActive,
			renews: renews,
			retry:  retry,
			want:   renews,
		},
		{
			name:   "other status returns RenewsAt when it is earlier",
			status: SubscriptionStatusPaused,
			renews: retry,  // earlier
			retry:  renews, // later
			want:   retry,
		},
		{
			name:   "other status returns NextRetryAt when RenewsAt is not earlier",
			status: SubscriptionStatusPaused,
			renews: renews, // later
			retry:  retry,  // earlier -> RenewsAt.Before(NextRetryAt) is false, return NextRetryAt
			want:   retry,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Subscription{Status: tt.status, RenewsAt: tt.renews, NextRetryAt: tt.retry}
			assert.Equal(t, tt.want, s.GetNextChargeDate())
		})
	}
}

func TestIsRunning(t *testing.T) {
	tests := []struct {
		status SubscriptionStatus
		want   bool
	}{
		{SubscriptionStatusActive, true},
		{SubscriptionStatusTrial, true},
		{SubscriptionStatusPastDue, true},
		{SubscriptionStatusPaused, false},
		{SubscriptionStatusCancelled, false},
		{SubscriptionStatusPending, false},
		{SubscriptionStatusExpired, false},
		{SubscriptionStatusUnpaid, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			s := Subscription{Status: tt.status}
			assert.Equal(t, tt.want, s.IsRunning())
		})
	}
}

func TestSetCancelled(t *testing.T) {
	s := &Subscription{
		Status:      SubscriptionStatusActive,
		RenewsAt:    billingBase,
		NextRetryAt: billingBase,
	}
	got := s.SetCancelled()

	assert.Same(t, s, got, "returns the same pointer for chaining")
	assert.Equal(t, SubscriptionStatusCancelled, s.Status)
	assert.True(t, s.RenewsAt.IsZero(), "RenewsAt cleared")
	assert.True(t, s.NextRetryAt.IsZero(), "NextRetryAt cleared")
	assert.False(t, s.CancelledAt.IsZero(), "CancelledAt stamped")
}

func TestSubscription_SetMetadata(t *testing.T) {
	t.Run("initialises nil map and merges", func(t *testing.T) {
		s := &Subscription{}
		got := s.SetMetadata(map[string]string{"a": "1"})
		assert.Same(t, s, got)
		assert.Equal(t, map[string]string{"a": "1"}, s.Metadata)
	})

	t.Run("merges into existing map, overwriting collisions", func(t *testing.T) {
		s := &Subscription{Metadata: map[string]string{"a": "1", "b": "2"}}
		s.SetMetadata(map[string]string{"b": "99", "c": "3"})
		assert.Equal(t, map[string]string{"a": "1", "b": "99", "c": "3"}, s.Metadata)
	})
}

func TestCalculateProrationDetails_EdgeCases(t *testing.T) {
	const day = 24 * time.Hour
	periodStart := billingBase
	periodEnd := billingBase.Add(30 * day)

	t.Run("none mode returns zero credit and copies anchors/periods", func(t *testing.T) {
		s := Subscription{CurrentPeriodStart: periodStart, CurrentPeriodEnd: periodEnd}
		newStart := billingBase.Add(5 * day)
		newEnd := billingBase.Add(35 * day)
		d := s.CalculateProrationDetails("none", billingBase.Add(15*day), 15, 20, newStart, newEnd, int64(1000))

		assert.Equal(t, 0, d.CreditAmount)
		assert.Equal(t, 0, d.DaysCredited)
		assert.Equal(t, 15, d.OldBillingAnchor)
		assert.Equal(t, 20, d.NewBillingAnchor)
		assert.Equal(t, periodStart, d.CurrentPeriodStart)
		assert.Equal(t, periodEnd, d.CurrentPeriodEnd)
		assert.Equal(t, newStart, d.NewPeriodStart)
		assert.Equal(t, newEnd, d.NewPeriodEnd)
	})

	t.Run("credit_unused with totalDays<=0 returns zero credit", func(t *testing.T) {
		// CurrentPeriodEnd == CurrentPeriodStart -> totalDays == 0
		s := Subscription{CurrentPeriodStart: periodStart, CurrentPeriodEnd: periodStart}
		d := s.CalculateProrationDetails("credit_unused", periodStart, 15, 20, periodStart, periodEnd, int64(1000))
		assert.Equal(t, 0, d.CreditAmount)
		assert.Equal(t, 0, d.DaysCredited)
	})

	t.Run("credit_unused with daysRemaining<=0 (reference at period end) returns zero credit", func(t *testing.T) {
		s := Subscription{CurrentPeriodStart: periodStart, CurrentPeriodEnd: periodEnd}
		d := s.CalculateProrationDetails("credit_unused", periodEnd, 15, 20, periodStart, periodEnd, int64(1000))
		assert.Equal(t, 0, d.CreditAmount)
		assert.Equal(t, 0, d.DaysCredited)
	})

	t.Run("credit_unused with daysRemaining<=0 (reference past period end) returns zero credit", func(t *testing.T) {
		s := Subscription{CurrentPeriodStart: periodStart, CurrentPeriodEnd: periodEnd}
		d := s.CalculateProrationDetails("credit_unused", periodEnd.Add(day), 15, 20, periodStart, periodEnd, int64(1000))
		assert.Equal(t, 0, d.CreditAmount)
		assert.Equal(t, 0, d.DaysCredited)
	})

	t.Run("credit_unused prorates by remaining days", func(t *testing.T) {
		s := Subscription{CurrentPeriodStart: periodStart, CurrentPeriodEnd: periodEnd}
		// 10 days remaining out of 30 -> 1000 * 10 / 30 = 333
		d := s.CalculateProrationDetails("credit_unused", periodEnd.Add(-10*day), 15, 20, periodStart, periodEnd, int64(1000))
		assert.Equal(t, 333, d.CreditAmount)
		assert.Equal(t, 10, d.DaysCredited)
	})

	t.Run("unknown proration mode behaves like none", func(t *testing.T) {
		s := Subscription{CurrentPeriodStart: periodStart, CurrentPeriodEnd: periodEnd}
		d := s.CalculateProrationDetails("weird_mode", billingBase.Add(15*day), 15, 20, periodStart, periodEnd, int64(1000))
		assert.Equal(t, 0, d.CreditAmount)
		assert.Equal(t, 0, d.DaysCredited)
	})
}

func TestCalculateBillingAnchor(t *testing.T) {
	// Reference clock components are carried over onto the computed anchor date.
	ref := time.Date(2024, time.January, 1, 9, 8, 7, 123, time.UTC)

	tests := []struct {
		name   string
		anchor int
		year   int
		month  int
		want   time.Time
	}{
		{
			name:   "normal anchor within month",
			anchor: 15,
			year:   2025,
			month:  3,
			want:   time.Date(2025, time.March, 15, 9, 8, 7, 123, time.UTC),
		},
		{
			name:   "anchor 31 clamps to Feb 28 (non-leap year)",
			anchor: 31,
			year:   2025,
			month:  2,
			want:   time.Date(2025, time.February, 28, 9, 8, 7, 123, time.UTC),
		},
		{
			name:   "anchor 31 clamps to Feb 29 (leap year)",
			anchor: 31,
			year:   2024,
			month:  2,
			want:   time.Date(2024, time.February, 29, 9, 8, 7, 123, time.UTC),
		},
		{
			name:   "anchor 31 in April clamps to 30",
			anchor: 31,
			year:   2025,
			month:  4,
			want:   time.Date(2025, time.April, 30, 9, 8, 7, 123, time.UTC),
		},
		{
			name:   "anchor 31 in a 31-day month stays 31",
			anchor: 31,
			year:   2025,
			month:  1,
			want:   time.Date(2025, time.January, 31, 9, 8, 7, 123, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateBillingAnchor(tt.anchor, tt.year, tt.month, ref)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCalculateNextDate(t *testing.T) {
	tests := []struct {
		name     string
		interval BillingInterval
		qty      int
		want     time.Time
	}{
		{"minute", BillingIntervalMinute, 30, billingBase.Add(30 * time.Minute)},
		{"hour", BillingIntervalHour(), 2, billingBase.Add(2 * time.Hour)},
		{"day", BillingIntervalDay, 5, billingBase.AddDate(0, 0, 5)},
		{"week", BillingIntervalWeek, 3, billingBase.AddDate(0, 0, 21)},
		{"month", BillingIntervalMonth, 6, billingBase.AddDate(0, 6, 0)},
		{"year", BillingIntervalYear, 2, billingBase.AddDate(2, 0, 0)},
		{"unknown interval returns input unchanged", BillingInterval("none"), 4, billingBase},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, calculateNextDate(tt.interval, tt.qty, billingBase))
		})
	}
}
