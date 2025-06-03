package entities

import (
	"payloop/internal/domain/entities/prices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSubscriptionFromOrderItem_FreeTrial(t *testing.T) {
	now := time.Now().UTC()
	trialQty := 1
	trialInterval := prices.BillingIntervalMonth

	orderItem := OrderItem{
		OrgId:   "org_123",
		OrderId: "order_123",
		Price: Price{
			BillingInterval:    prices.BillingIntervalMonth,
			BillingIntervalQty: 1,
			Category:           prices.PriceCategorySubscription,
			Currency:           "USD",
			UnitPrice:          1000,
			TrialInterval:      trialInterval,
			TrialIntervalQty:   trialQty,
		},
	}

	subscription := NewSubscriptionFromOrderItem(orderItem)

	assert.Equal(t, orderItem.OrgId, subscription.OrgId)
	assert.Equal(t, orderItem.OrderId, subscription.OrderId)

	assert.Equal(t, SubscriptionStatusPending, subscription.Status)
	assert.WithinDuration(t, time.Now().UTC().AddDate(0, 1, 0), subscription.StartDate, 10*time.Second)
	assert.NotNil(t, subscription.TrialEndsAt)
	assert.WithinDuration(t, time.Now().UTC().AddDate(0, 1, 0), subscription.TrialEndsAt, 10*time.Second)

	assert.Equal(t, orderItem.Price.BillingInterval, subscription.BillingInterval)
	assert.Equal(t, orderItem.Price.BillingIntervalQty, subscription.BillingIntervalQty)

	assert.Equal(t, orderItem.Price.Currency, subscription.Currency)
	assert.Equal(t, orderItem.Price.UnitPrice, subscription.Amount)

	assert.Equal(t, 0, subscription.Cycles)
	assert.Equal(t, now.Day(), subscription.BillingAnchor)

	assert.Nil(t, subscription.CancelAt)
	assert.Nil(t, subscription.EndsAt)
	assert.Nil(t, subscription.LastCharge)
	assert.Nil(t, subscription.RenewsAt)
	assert.Equal(t, 0, subscription.Retries)
	assert.Nil(t, subscription.NextRetryAt)
	assert.Equal(t, 0, subscription.CyclesProcessed)
	assert.Equal(t, 0, subscription.TotalRevenue)
	assert.Nil(t, subscription.CancelledAt)
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
				BillingInterval:    prices.BillingIntervalMonth,
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
				BillingInterval:    prices.BillingIntervalMonth,
				BillingIntervalQty: 1,
			},
			expectedNextDate: now.AddDate(0, 1, 0),
		},
		{
			name: "With LastCharge",
			subscription: Subscription{
				StartDate:          now.AddDate(0, -1, 0),
				LastCharge:         now,
				BillingInterval:    prices.BillingIntervalMonth,
				BillingIntervalQty: 1,
			},
			expectedNextDate: now.AddDate(0, 1, 0),
		},
		{
			name: "With LastCharge and CyclesProcessed",
			subscription: Subscription{
				StartDate:          now.AddDate(0, -2, 0),
				LastCharge:         now,
				BillingInterval:    prices.BillingIntervalMonth,
				BillingIntervalQty: 1,
				CyclesProcessed:    1,
			},
			expectedNextDate: now.AddDate(0, 1, 0),
		},
		{
			name: "Weekly Billing Interval",
			subscription: Subscription{
				StartDate:          now.AddDate(0, 0, -7),
				LastCharge:         now,
				BillingInterval:    prices.BillingIntervalWeek,
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
	trialQty := 0
	trialInterval := prices.BillingIntervalNone

	orderItem := OrderItem{
		OrgId:   "org_123",
		OrderId: "order_123",
		Price: Price{
			BillingInterval:    prices.BillingIntervalMonth,
			BillingIntervalQty: 1,
			Category:           prices.PriceCategorySubscription,
			Currency:           "USD",
			UnitPrice:          1000,
			TrialInterval:      trialInterval,
			TrialIntervalQty:   trialQty,
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
	// Create a subscription with known values
	subscription := Subscription{
		Amount:             1000,
		BillingInterval:    "month",
		BillingIntervalQty: 1,
		BillingAnchor:      15,
		CurrentPeriodStart: time.Now().UTC(),
		CurrentPeriodEnd:   time.Now().UTC().AddDate(0, 1, 0),
	}

	// Test with no proration
	prorationDetails := subscription.UpdateBillingAnchor(20, "none")

	// Verify the billing anchor was updated
	assert.Equal(t, 20, subscription.BillingAnchor)

	// Verify the proration details
	assert.Equal(t, 0, prorationDetails.CreditAmount)
	assert.Equal(t, 0, prorationDetails.DaysCredited)
	assert.Equal(t, 15, prorationDetails.OldBillingAnchor)
	assert.Equal(t, 20, prorationDetails.NewBillingAnchor)
	assert.False(t, prorationDetails.NewPeriodStart.IsZero())
	assert.False(t, prorationDetails.NewPeriodEnd.IsZero())

	// Test with credit_unused proration
	subscription.BillingAnchor = 15 // Reset
	prorationDetails = subscription.UpdateBillingAnchor(25, "credit_unused")

	// Verify the billing anchor was updated
	assert.Equal(t, 25, subscription.BillingAnchor)

	// Verify the proration details have credit amount
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
	periodEnd := now.AddDate(0, 1, 0) // 1 month period
	referenceDate := now.AddDate(0, 0, 15) // 15 days into the period
	amount := int64(1000) // $10.00
	oldBillingAnchor := 15
	newBillingAnchor := 20
	newPeriodStart := now.AddDate(0, 0, 5) // 5 days from now
	newPeriodEnd := newPeriodStart.AddDate(0, 1, 0) // 1 month after new period start

	tests := []struct {
		name             string
		prorationMode    string
		amount           int64
		periodStart      time.Time
		periodEnd        time.Time
		referenceDate    time.Time
		oldBillingAnchor int
		newBillingAnchor int
		newPeriodStart   time.Time
		newPeriodEnd     time.Time
		expectedDetails  ProrationDetails
	}{
		{
			name:             "No Proration",
			prorationMode:    "none",
			amount:           amount,
			periodStart:      periodStart,
			periodEnd:        periodEnd,
			referenceDate:    referenceDate,
			oldBillingAnchor: oldBillingAnchor,
			newBillingAnchor: newBillingAnchor,
			newPeriodStart:   newPeriodStart,
			newPeriodEnd:     newPeriodEnd,
			expectedDetails: ProrationDetails{
				CreditAmount:       0,
				DaysCredited:       0,
				CurrentPeriodStart: periodStart,
				CurrentPeriodEnd:   periodEnd,
				OldBillingAnchor:   oldBillingAnchor,
				NewBillingAnchor:   newBillingAnchor,
				NewPeriodStart:     newPeriodStart,
				NewPeriodEnd:       newPeriodEnd,
			},
		},
		{
			name:             "Credit Unused - Half Period",
			prorationMode:    "credit_unused",
			amount:           amount,
			periodStart:      periodStart,
			periodEnd:        periodEnd,
			referenceDate:    referenceDate,
			oldBillingAnchor: oldBillingAnchor,
			newBillingAnchor: newBillingAnchor,
			newPeriodStart:   newPeriodStart,
			newPeriodEnd:     newPeriodEnd,
			expectedDetails: ProrationDetails{
				CreditAmount:       500, // Half of the amount
				DaysCredited:       15,  // Half of the period
				CurrentPeriodStart: periodStart,
				CurrentPeriodEnd:   periodEnd,
				OldBillingAnchor:   oldBillingAnchor,
				NewBillingAnchor:   newBillingAnchor,
				NewPeriodStart:     newPeriodStart,
				NewPeriodEnd:       newPeriodEnd,
			},
		},
		{
			name:             "Credit Unused - No Days Remaining",
			prorationMode:    "credit_unused",
			amount:           amount,
			periodStart:      periodStart,
			periodEnd:        periodEnd,
			referenceDate:    periodEnd.Add(time.Hour), // After period end
			oldBillingAnchor: oldBillingAnchor,
			newBillingAnchor: newBillingAnchor,
			newPeriodStart:   newPeriodStart,
			newPeriodEnd:     newPeriodEnd,
			expectedDetails: ProrationDetails{
				CreditAmount:       0,
				DaysCredited:       0,
				CurrentPeriodStart: periodStart,
				CurrentPeriodEnd:   periodEnd,
				OldBillingAnchor:   oldBillingAnchor,
				NewBillingAnchor:   newBillingAnchor,
				NewPeriodStart:     newPeriodStart,
				NewPeriodEnd:       newPeriodEnd,
			},
		},
		{
			name:             "Credit Unused - All Days Remaining",
			prorationMode:    "credit_unused",
			amount:           amount,
			periodStart:      periodStart,
			periodEnd:        periodEnd,
			referenceDate:    periodStart, // At period start
			oldBillingAnchor: oldBillingAnchor,
			newBillingAnchor: newBillingAnchor,
			newPeriodStart:   newPeriodStart,
			newPeriodEnd:     newPeriodEnd,
			expectedDetails: ProrationDetails{
				CreditAmount:       1000, // Full amount
				DaysCredited:       30,   // Full period (approximately)
				CurrentPeriodStart: periodStart,
				CurrentPeriodEnd:   periodEnd,
				OldBillingAnchor:   oldBillingAnchor,
				NewBillingAnchor:   newBillingAnchor,
				NewPeriodStart:     newPeriodStart,
				NewPeriodEnd:       newPeriodEnd,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a subscription with the test case values
			subscription := Subscription{
				Amount:             tt.amount,
				CurrentPeriodStart: tt.periodStart,
				CurrentPeriodEnd:   tt.periodEnd,
			}

			details := subscription.CalculateProrationDetails(
				tt.prorationMode,
				tt.referenceDate,
				tt.oldBillingAnchor,
				tt.newBillingAnchor,
				tt.newPeriodStart,
				tt.newPeriodEnd,
			)

			// For the "All Days Remaining" test, we need to check approximately due to month length variations
			if tt.name == "Credit Unused - All Days Remaining" {
				totalDays := int(tt.periodEnd.Sub(tt.periodStart).Hours() / 24)
				assert.Equal(t, totalDays, details.DaysCredited)
				assert.Equal(t, tt.amount, int64(details.CreditAmount))
			} else {
				assert.Equal(t, tt.expectedDetails.CreditAmount, details.CreditAmount)
				assert.Equal(t, tt.expectedDetails.DaysCredited, details.DaysCredited)
			}

			assert.Equal(t, tt.expectedDetails.CurrentPeriodStart, details.CurrentPeriodStart)
			assert.Equal(t, tt.expectedDetails.CurrentPeriodEnd, details.CurrentPeriodEnd)
			assert.Equal(t, tt.expectedDetails.OldBillingAnchor, details.OldBillingAnchor)
			assert.Equal(t, tt.expectedDetails.NewBillingAnchor, details.NewBillingAnchor)
			assert.Equal(t, tt.expectedDetails.NewPeriodStart, details.NewPeriodStart)
			assert.Equal(t, tt.expectedDetails.NewPeriodEnd, details.NewPeriodEnd)
		})
	}
}
