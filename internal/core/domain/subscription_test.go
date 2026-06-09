package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSubscriptionForCadence(t *testing.T) {
	now := time.Now().UTC()

	subscription := NewSubscriptionForCadence("org_123", "order_123", "cus_1", BillingIntervalMonth, 1, 12, "USD")

	assert.Equal(t, "org_123", subscription.OrgId)
	assert.Equal(t, "order_123", subscription.OrderId)
	assert.Equal(t, "cus_1", subscription.CustomerId)
	assert.Equal(t, SubscriptionStatusPending, subscription.Status)
	assert.Equal(t, BillingIntervalMonth, subscription.BillingInterval)
	assert.Equal(t, 1, subscription.BillingIntervalQty)
	assert.Equal(t, 12, subscription.Cycles)
	assert.Equal(t, "USD", subscription.Currency)
	assert.NotEmpty(t, subscription.Id)
	assert.Equal(t, 0, subscription.Retries)
	assert.Equal(t, 0, subscription.CyclesProcessed)
	assert.Equal(t, int64(0), subscription.TotalRevenue)
	assert.True(t, subscription.CancelAt.IsZero())
	assert.True(t, subscription.EndsAt.IsZero())
	assert.WithinDuration(t, now, subscription.CreatedAt, 5*time.Second)
	assert.WithinDuration(t, now, subscription.UpdatedAt, 5*time.Second)
}

func TestNextBillingDate(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name             string
		subscription     Subscription
		expectedNextDate time.Time
	}{
		{
			name: "No LastCharge, No CyclesProcessed",
			subscription: Subscription{
				StartDate:          now,
				BillingInterval:    BillingIntervalMonth,
				BillingIntervalQty: 1,
			},
			expectedNextDate: now,
		},
		{
			name: "Started now, With LastCharge, 1 CycleProcessed",
			subscription: Subscription{
				StartDate:          now,
				LastCharge:         now,
				CyclesProcessed:    1,
				CurrentPeriodEnd:   now,
				BillingInterval:    BillingIntervalMonth,
				BillingIntervalQty: 1,
			},
			expectedNextDate: now.AddDate(0, 1, 0),
		},
		{
			name: "With LastCharge",
			subscription: Subscription{
				StartDate:          now.AddDate(0, -1, 0),
				LastCharge:         now,
				CyclesProcessed:    1,
				CurrentPeriodEnd:   now,
				BillingInterval:    BillingIntervalMonth,
				BillingIntervalQty: 1,
			},
			expectedNextDate: now.AddDate(0, 1, 0),
		},
		{
			name: "With LastCharge and CyclesProcessed",
			subscription: Subscription{
				StartDate:          now.AddDate(0, -2, 0),
				LastCharge:         now,
				CyclesProcessed:    1,
				CurrentPeriodEnd:   now,
				BillingInterval:    BillingIntervalMonth,
				BillingIntervalQty: 1,
			},
			expectedNextDate: now.AddDate(0, 1, 0),
		},
		{
			name: "Weekly Billing Interval",
			subscription: Subscription{
				StartDate:          now.AddDate(0, 0, -7),
				LastCharge:         now,
				CyclesProcessed:    1,
				CurrentPeriodEnd:   now,
				BillingInterval:    BillingIntervalWeek,
				BillingIntervalQty: 1,
			},
			expectedNextDate: now.AddDate(0, 0, 7),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextDate := tt.subscription.CalculateNextBillingDate()
			assert.WithinDuration(t, tt.expectedNextDate, nextDate, time.Second)
		})
	}
}

func TestSetActivationDates(t *testing.T) {
	now := time.Now().UTC()

	orderItem := OrderItem{
		OrgId:   "org_123",
		OrderId: "order_123",
	}
	price := Price{
		BillingInterval:    BillingIntervalMonth,
		BillingIntervalQty: 1,
		Category:           PriceCategorySubscription,
		Currency:           "USD",
		UnitPrice:          1000,
		TrialInterval:      BillingIntervalNone,
		TrialIntervalQty:   0,
	}

	subscription := NewSubscriptionForCadence(orderItem.OrgId, orderItem.OrderId, "", price.BillingInterval, price.BillingIntervalQty, price.Cycles, string(price.Currency))
	subscription.SetActivationDates(price)

	assert.WithinDuration(t, now, subscription.StartDate, 10*time.Second)
	assert.WithinDuration(t, now, subscription.CurrentPeriodStart, 10*time.Second)
	assert.WithinDuration(t, now, subscription.RenewsAt, 10*time.Second)
	assert.Equal(t, now.Day(), subscription.BillingAnchor)
}

func TestUpdateBillingAnchor(t *testing.T) {
	subscription := Subscription{
		BillingInterval:    "month",
		BillingIntervalQty: 1,
		BillingAnchor:      15,
		CurrentPeriodStart: time.Now().UTC(),
		CurrentPeriodEnd:   time.Now().UTC().AddDate(0, 1, 0),
	}

	prorationDetails := subscription.UpdateBillingAnchor(20, "none", 1000)

	assert.Equal(t, 20, subscription.BillingAnchor)
	assert.Equal(t, 0, prorationDetails.CreditAmount)
	assert.Equal(t, 0, prorationDetails.DaysCredited)
	assert.Equal(t, 15, prorationDetails.OldBillingAnchor)
	assert.Equal(t, 20, prorationDetails.NewBillingAnchor)
	assert.False(t, prorationDetails.NewPeriodStart.IsZero())
	assert.False(t, prorationDetails.NewPeriodEnd.IsZero())

	subscription.BillingAnchor = 15
	prorationDetails = subscription.UpdateBillingAnchor(25, "credit_unused", 1000)

	assert.Equal(t, 25, subscription.BillingAnchor)
	assert.Greater(t, prorationDetails.CreditAmount, 0)
	assert.Greater(t, prorationDetails.DaysCredited, 0)
	assert.Equal(t, 15, prorationDetails.OldBillingAnchor)
	assert.Equal(t, 25, prorationDetails.NewBillingAnchor)
	assert.False(t, prorationDetails.NewPeriodStart.IsZero())
	assert.False(t, prorationDetails.NewPeriodEnd.IsZero())
}

func TestCalculateProrationDetails(t *testing.T) {
	// Frozen, exact 30-day window so the proration math is deterministic.
	// Using time.Duration arithmetic (not AddDate) keeps the period exactly
	// 30 days regardless of which calendar month the test runs in, and UTC
	// avoids DST shifts. The previous version seeded from time.Now()+AddDate,
	// which gave a 28-31 day period and made the test clock-dependent.
	const day = 24 * time.Hour
	base := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	periodStart := base
	periodEnd := base.Add(30 * day)
	referenceDate := base.Add(15 * day)
	amount := int64(1000)

	tests := []struct {
		name           string
		prorationMode  string
		referenceDate  time.Time
		expectedCredit int
		expectedDays   int
	}{
		{
			name:           "No Proration",
			prorationMode:  "none",
			referenceDate:  referenceDate,
			expectedCredit: 0,
			expectedDays:   0,
		},
		{
			name:           "Credit Unused - Half Period",
			prorationMode:  "credit_unused",
			referenceDate:  referenceDate,
			expectedCredit: 500,
			expectedDays:   15,
		},
		{
			name:           "Credit Unused - No Days Remaining",
			prorationMode:  "credit_unused",
			referenceDate:  periodEnd.Add(time.Hour),
			expectedCredit: 0,
			expectedDays:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subscription := Subscription{
				CurrentPeriodStart: periodStart,
				CurrentPeriodEnd:   periodEnd,
			}

			details := subscription.CalculateProrationDetails(
				tt.prorationMode, tt.referenceDate, 15, 20,
				base.Add(5*day), base.Add(35*day), amount,
			)

			assert.Equal(t, tt.expectedCredit, details.CreditAmount)
			assert.Equal(t, tt.expectedDays, details.DaysCredited)
		})
	}
}

func TestSubscription_IsDueForBilling(t *testing.T) {
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name string
		sub  Subscription
		want bool
	}{
		{
			name: "active and due (renews in past)",
			sub:  Subscription{Status: SubscriptionStatusActive, RenewsAt: past},
			want: true,
		},
		{
			name: "active and due (renews exactly now)",
			sub:  Subscription{Status: SubscriptionStatusActive, RenewsAt: now},
			want: true,
		},
		{
			name: "active but renews in future (not due)",
			sub:  Subscription{Status: SubscriptionStatusActive, RenewsAt: future},
			want: false,
		},
		{
			name: "active with zero renews (not due)",
			sub:  Subscription{Status: SubscriptionStatusActive},
			want: false,
		},
		{
			name: "past_due with retry in past (due)",
			sub:  Subscription{Status: SubscriptionStatusPastDue, NextRetryAt: past},
			want: true,
		},
		{
			name: "past_due with zero retry (not due)",
			sub:  Subscription{Status: SubscriptionStatusPastDue},
			want: false,
		},
		{
			name: "trial ended (due)",
			sub:  Subscription{Status: SubscriptionStatusTrial, TrialEndsAt: past},
			want: true,
		},
		{
			name: "trial not ended (not due)",
			sub:  Subscription{Status: SubscriptionStatusTrial, TrialEndsAt: future},
			want: false,
		},
		{
			name: "trial with zero trial-ends (not due)",
			sub:  Subscription{Status: SubscriptionStatusTrial},
			want: false,
		},
		{
			name: "cancelled with past renews (not due)",
			sub:  Subscription{Status: SubscriptionStatusCancelled, RenewsAt: past},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.sub.IsDueForBilling(now))
		})
	}
}
