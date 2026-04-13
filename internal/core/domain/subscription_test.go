package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSubscriptionFromOrderItem_FreeTrial(t *testing.T) {
	now := time.Now().UTC()
	trialInterval := BillingIntervalMonth

	orderItem := OrderItem{
		OrgId:   "org_123",
		OrderId: "order_123",
		Price: Price{
			BillingInterval:    BillingIntervalMonth,
			BillingIntervalQty: 1,
			Category:           PriceCategorySubscription,
			Currency:           "USD",
			UnitPrice:          1000,
			TrialInterval:      trialInterval,
			TrialIntervalQty:   1,
		},
	}

	subscription := NewSubscriptionFromOrderItem(orderItem)

	assert.Equal(t, orderItem.OrgId, subscription.OrgId)
	assert.Equal(t, orderItem.OrderId, subscription.OrderId)
	assert.Equal(t, SubscriptionStatusPending, subscription.Status)
	assert.Equal(t, orderItem.Price.BillingInterval, subscription.BillingInterval)
	assert.Equal(t, orderItem.Price.BillingIntervalQty, subscription.BillingIntervalQty)
	assert.Equal(t, string(orderItem.Price.Currency), subscription.Currency)
	assert.Equal(t, orderItem.Price.UnitPrice, subscription.Amount)
	assert.Equal(t, 0, subscription.Cycles)
	assert.Equal(t, 0, subscription.Retries)
	assert.Equal(t, 0, subscription.CyclesProcessed)
	assert.Equal(t, int64(0), subscription.TotalRevenue)
	assert.True(t, subscription.CancelAt.IsZero())
	assert.True(t, subscription.EndsAt.IsZero())
	assert.True(t, subscription.LastCharge.IsZero())
	assert.True(t, subscription.NextRetryAt.IsZero())
	assert.True(t, subscription.CancelledAt.IsZero())
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
		Price: Price{
			BillingInterval:    BillingIntervalMonth,
			BillingIntervalQty: 1,
			Category:           PriceCategorySubscription,
			Currency:           "USD",
			UnitPrice:          1000,
			TrialInterval:      BillingIntervalNone,
			TrialIntervalQty:   0,
		},
	}

	subscription := NewSubscriptionFromOrderItem(orderItem)
	subscription.SetActivationDates()

	assert.WithinDuration(t, now, subscription.StartDate, 10*time.Second)
	assert.WithinDuration(t, now, subscription.CurrentPeriodStart, 10*time.Second)
	assert.WithinDuration(t, now, subscription.RenewsAt, 10*time.Second)
	assert.Equal(t, now.Day(), subscription.BillingAnchor)
}

func TestUpdateBillingAnchor(t *testing.T) {
	subscription := Subscription{
		Amount:             1000,
		BillingInterval:    "month",
		BillingIntervalQty: 1,
		BillingAnchor:      15,
		CurrentPeriodStart: time.Now().UTC(),
		CurrentPeriodEnd:   time.Now().UTC().AddDate(0, 1, 0),
	}

	prorationDetails := subscription.UpdateBillingAnchor(20, "none")

	assert.Equal(t, 20, subscription.BillingAnchor)
	assert.Equal(t, 0, prorationDetails.CreditAmount)
	assert.Equal(t, 0, prorationDetails.DaysCredited)
	assert.Equal(t, 15, prorationDetails.OldBillingAnchor)
	assert.Equal(t, 20, prorationDetails.NewBillingAnchor)
	assert.False(t, prorationDetails.NewPeriodStart.IsZero())
	assert.False(t, prorationDetails.NewPeriodEnd.IsZero())

	subscription.BillingAnchor = 15
	prorationDetails = subscription.UpdateBillingAnchor(25, "credit_unused")

	assert.Equal(t, 25, subscription.BillingAnchor)
	assert.Greater(t, prorationDetails.CreditAmount, 0)
	assert.Greater(t, prorationDetails.DaysCredited, 0)
	assert.Equal(t, 15, prorationDetails.OldBillingAnchor)
	assert.Equal(t, 25, prorationDetails.NewBillingAnchor)
	assert.False(t, prorationDetails.NewPeriodStart.IsZero())
	assert.False(t, prorationDetails.NewPeriodEnd.IsZero())
}

func TestCalculateProrationDetails(t *testing.T) {
	now := time.Now().UTC()
	periodStart := now
	periodEnd := now.AddDate(0, 1, 0)
	referenceDate := now.AddDate(0, 0, 15)
	amount := int64(1000)

	tests := []struct {
		name            string
		prorationMode   string
		referenceDate   time.Time
		expectedCredit  int
		expectedDays    int
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
				Amount:             amount,
				CurrentPeriodStart: periodStart,
				CurrentPeriodEnd:   periodEnd,
			}

			details := subscription.CalculateProrationDetails(
				tt.prorationMode, tt.referenceDate, 15, 20,
				now.AddDate(0, 0, 5), now.AddDate(0, 0, 5).AddDate(0, 1, 0),
			)

			assert.Equal(t, tt.expectedCredit, details.CreditAmount)
			assert.Equal(t, tt.expectedDays, details.DaysCredited)
		})
	}
}
