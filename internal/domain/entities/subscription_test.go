package entities

import (
	"encoding/json"
	"payloop/internal/domain/entities/prices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	// In pending state, dates are not set yet - they're set during activation
	assert.True(t, subscription.StartDate.IsZero())
	assert.True(t, subscription.TrialEndsAt.IsZero())

	assert.Equal(t, orderItem.Price.BillingInterval, subscription.BillingInterval)
	assert.Equal(t, orderItem.Price.BillingIntervalQty, subscription.BillingIntervalQty)

	// Legacy fields are deprecated in favor of subscription items

	assert.Equal(t, orderItem.Price.Cycles, subscription.Cycles)

	assert.True(t, subscription.CancelAt.IsZero())
	assert.True(t, subscription.EndsAt.IsZero())
	assert.True(t, subscription.LastCharge.IsZero())
	assert.True(t, subscription.RenewsAt.IsZero())
	assert.Equal(t, 0, subscription.CyclesProcessed)
	assert.Equal(t, int64(0), subscription.TotalRevenue)
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

	tests := []struct {
		name                 string
		orderItem            OrderItem
		subscriptionItems    []SubscriptionItem
		expectedTrialEndsAt  bool
		expectedTrialDays    int
	}{
		{
			name: "No trial period in OrderItem",
			orderItem: OrderItem{
				OrgId:   "org_123",
				OrderId: "order_123",
				Price: Price{
					BillingInterval:    prices.BillingIntervalMonth,
					BillingIntervalQty: 1,
					Category:           prices.PriceCategorySubscription,
					Currency:           "USD",
					UnitPrice:          1000,
					TrialInterval:      prices.BillingIntervalNone,
					TrialIntervalQty:   0,
				},
			},
			subscriptionItems:   []SubscriptionItem{},
			expectedTrialEndsAt: false,
			expectedTrialDays:   0,
		},
		{
			name: "With trial period in OrderItem",
			orderItem: OrderItem{
				OrgId:   "org_123",
				OrderId: "order_123",
				Price: Price{
					BillingInterval:    prices.BillingIntervalMonth,
					BillingIntervalQty: 1,
					Category:           prices.PriceCategorySubscription,
					Currency:           "USD",
					UnitPrice:          1000,
					TrialInterval:      prices.BillingIntervalDay,
					TrialIntervalQty:   14,
				},
			},
			subscriptionItems:   []SubscriptionItem{},
			expectedTrialEndsAt: true,
			expectedTrialDays:   14,
		},
		{
			name: "Multiple subscription items with different trial periods",
			orderItem: OrderItem{
				OrgId:   "org_123",
				OrderId: "order_123",
				Price: Price{
					BillingInterval:    prices.BillingIntervalMonth,
					BillingIntervalQty: 1,
					Category:           prices.PriceCategorySubscription,
					Currency:           "USD",
					UnitPrice:          1000,
					TrialInterval:      prices.BillingIntervalDay,
					TrialIntervalQty:   30, // This should be ignored in favor of subscription items
				},
			},
			subscriptionItems: []SubscriptionItem{
				{
					OrgId:          "org_123",
					SubscriptionId: "sub_123",
					PriceId:        "price_1",
					PriceSnapshot:  createPriceSnapshotForTest(t, prices.BillingIntervalDay, 14),
				},
				{
					OrgId:          "org_123",
					SubscriptionId: "sub_123",
					PriceId:        "price_2",
					PriceSnapshot:  createPriceSnapshotForTest(t, prices.BillingIntervalDay, 7), // Shortest trial
				},
				{
					OrgId:          "org_123",
					SubscriptionId: "sub_123",
					PriceId:        "price_3",
					PriceSnapshot:  createPriceSnapshotForTest(t, prices.BillingIntervalDay, 21),
				},
			},
			expectedTrialEndsAt: true,
			expectedTrialDays:   7, // Should use the shortest trial period (7 days)
		},
		{
			name: "One subscription item with no trial period",
			orderItem: OrderItem{
				OrgId:   "org_123",
				OrderId: "order_123",
				Price: Price{
					BillingInterval:    prices.BillingIntervalMonth,
					BillingIntervalQty: 1,
					Category:           prices.PriceCategorySubscription,
					Currency:           "USD",
					UnitPrice:          1000,
					TrialInterval:      prices.BillingIntervalDay,
					TrialIntervalQty:   14, // This should be ignored in favor of subscription items
				},
			},
			subscriptionItems: []SubscriptionItem{
				{
					OrgId:          "org_123",
					SubscriptionId: "sub_123",
					PriceId:        "price_1",
					PriceSnapshot:  createPriceSnapshotForTest(t, prices.BillingIntervalDay, 14),
				},
				{
					OrgId:          "org_123",
					SubscriptionId: "sub_123",
					PriceId:        "price_2",
					PriceSnapshot:  createPriceSnapshotForTest(t, prices.BillingIntervalNone, 0), // No trial
				},
			},
			expectedTrialEndsAt: false, // Should have no trial because one item has no trial
			expectedTrialDays:   0,
		},
		{
			name: "Different trial interval types",
			orderItem: OrderItem{
				OrgId:   "org_123",
				OrderId: "order_123",
				Price: Price{
					BillingInterval:    prices.BillingIntervalMonth,
					BillingIntervalQty: 1,
					Category:           prices.PriceCategorySubscription,
					Currency:           "USD",
					UnitPrice:          1000,
					TrialInterval:      prices.BillingIntervalDay,
					TrialIntervalQty:   30, // This should be ignored in favor of subscription items
				},
			},
			subscriptionItems: []SubscriptionItem{
				{
					OrgId:          "org_123",
					SubscriptionId: "sub_123",
					PriceId:        "price_1",
					PriceSnapshot:  createPriceSnapshotForTest(t, prices.BillingIntervalMonth, 1), // 30 days approx
				},
				{
					OrgId:          "org_123",
					SubscriptionId: "sub_123",
					PriceId:        "price_2",
					PriceSnapshot:  createPriceSnapshotForTest(t, prices.BillingIntervalWeek, 1), // 7 days
				},
			},
			expectedTrialEndsAt: true,
			expectedTrialDays:   7, // Should use the shortest trial period (7 days)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subscription := NewSubscriptionFromOrderItem(tt.orderItem)
			subscription.Items = tt.subscriptionItems
			subscription.SetActivationDates()

			assert.WithinDuration(t, now, subscription.StartDate, 10*time.Second)
			assert.WithinDuration(t, now, subscription.CurrentPeriodStart, 10*time.Second)
			assert.Equal(t, now.Day(), subscription.BillingAnchor)

			if tt.expectedTrialEndsAt {
				assert.False(t, subscription.TrialEndsAt.IsZero(), "Expected trial end date to be set")
				expectedTrialEnd := now.AddDate(0, 0, tt.expectedTrialDays)
				assert.WithinDuration(t, expectedTrialEnd, subscription.TrialEndsAt, 10*time.Second)
			} else {
				assert.True(t, subscription.TrialEndsAt.IsZero(), "Expected trial end date to be zero")
			}
		})
	}
}

// Helper function to create a price snapshot for testing
func createPriceSnapshotForTest(t *testing.T, trialInterval prices.BillingInterval, trialIntervalQty int) json.RawMessage {
	price := Price{
		BillingInterval:    prices.BillingIntervalMonth,
		BillingIntervalQty: 1,
		Category:           prices.PriceCategorySubscription,
		Currency:           "USD",
		UnitPrice:          1000,
		TrialInterval:      trialInterval,
		TrialIntervalQty:   trialIntervalQty,
	}

	snapshot, err := json.Marshal(price)
	require.NoError(t, err, "Failed to marshal price")
	return snapshot
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

			// For tests that depend on the actual number of days in the period, we need to check approximately
			if tt.name == "Credit Unused - All Days Remaining" {
				totalDays := int(tt.periodEnd.Sub(tt.periodStart).Hours() / 24)
				assert.Equal(t, totalDays, details.DaysCredited)
				assert.Equal(t, tt.amount, int64(details.CreditAmount))
			} else if tt.name == "Credit Unused - Half Period" {
				// For half period, we expect approximately half the amount and half the days
				totalDays := int(tt.periodEnd.Sub(tt.periodStart).Hours() / 24)
				expectedDays := totalDays / 2
				expectedAmount := int(float64(tt.amount) * float64(expectedDays) / float64(totalDays))

				// Allow for a small margin of error (±1 day, ±50 in amount)
				assert.InDelta(t, expectedDays, details.DaysCredited, 1)
				assert.InDelta(t, expectedAmount, details.CreditAmount, 50)
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

// TestNewSubscriptionFromOrderItem_TraditionalSubscription tests creation from traditional subscription order items
func TestNewSubscriptionFromOrderItem_TraditionalSubscription(t *testing.T) {
	tests := []struct {
		name string
		orderItem OrderItem
		expectedSettings func(t *testing.T, sub Subscription)
	}{
		{
			name: "Basic Monthly Subscription",
			orderItem: OrderItem{
				OrgId:       "org_123",
				Id:          "item_123",
				OrderId:     "order_123",
				ProductId:   "prod_123",
				VariantId:   "var_123",
				Description: "Pro Plan",
				Price: Price{
					Id:                 "price_123",
					BillingInterval:    prices.BillingIntervalMonth,
					BillingIntervalQty: 1,
					Category:           prices.PriceCategorySubscription,
					Currency:           "USD",
					UnitPrice:          2999, // $29.99
					TrialInterval:      prices.BillingIntervalNone,
					TrialIntervalQty:   0,
					Cycles:             0, // Unlimited
				},
			},
			expectedSettings: func(t *testing.T, sub Subscription) {
				// Basic subscription properties
				assert.Equal(t, "org_123", sub.OrgId)
				assert.Equal(t, "order_123", sub.OrderId)
				assert.Equal(t, "item_123", sub.OrderItemId)
				assert.Equal(t, SubscriptionStatusPending, sub.Status)

				// Billing configuration
				assert.Equal(t, prices.BillingIntervalMonth, sub.BillingInterval)
				assert.Equal(t, 1, sub.BillingIntervalQty)
				assert.Equal(t, 0, sub.Cycles) // Unlimited

				// Trial settings (should be empty for no trial)
				assert.True(t, sub.TrialEndsAt.IsZero())

				// Initial state
				assert.False(t, sub.DunningActive)
				assert.Equal(t, 0, sub.CyclesProcessed)
				assert.Equal(t, int64(0), sub.TotalRevenue)

				// Subscription items
				require.Len(t, sub.Items, 1)
				item := sub.Items[0]
				assert.Equal(t, "org_123", item.OrgId)
				assert.Equal(t, sub.Id, item.SubscriptionId)
				assert.Equal(t, "price_123", item.PriceId)
				assert.Equal(t, "prod_123", item.ProductId)
				assert.Equal(t, "var_123", item.VariantId)
				assert.Equal(t, "Pro Plan", item.Description)
				assert.Equal(t, SubscriptionItemStatusActive, item.Status)
				assert.Equal(t, 1, item.Quantity)
				assert.Equal(t, int64(2999), item.Amount)
				assert.Equal(t, "USD", item.Currency)

				// Usage flags should be false for traditional subscription
				assert.False(t, item.HasUsage)
				assert.Equal(t, UsageType(""), item.UsageType)
				assert.Equal(t, UnitType(""), item.UnitType)
				assert.Equal(t, AggregationType(""), item.AggregationType)
				assert.Equal(t, float64(0), item.PercentageRate)
				assert.Equal(t, int64(0), item.FixedFee)
				assert.Equal(t, int64(0), item.UnitPrice)
			},
		},
		{
			name: "Annual Subscription with Trial",
			orderItem: OrderItem{
				OrgId:       "org_456",
				Id:          "item_456",
				OrderId:     "order_456",
				ProductId:   "prod_456",
				Description: "Enterprise Plan",
				Price: Price{
					Id:                 "price_456",
					BillingInterval:    prices.BillingIntervalYear,
					BillingIntervalQty: 1,
					Category:           prices.PriceCategorySubscription,
					Currency:           "USD",
					UnitPrice:          29999, // $299.99
					TrialInterval:      prices.BillingIntervalDay,
					TrialIntervalQty:   14, // 14-day trial
					Cycles:             0,
				},
			},
			expectedSettings: func(t *testing.T, sub Subscription) {
				// Billing configuration
				assert.Equal(t, prices.BillingIntervalYear, sub.BillingInterval)
				assert.Equal(t, 1, sub.BillingIntervalQty)

				// Trial settings should be empty in pending state
				// (trial dates are set during activation)
				assert.True(t, sub.TrialEndsAt.IsZero())

				// Subscription item
				require.Len(t, sub.Items, 1)
				item := sub.Items[0]
				assert.Equal(t, int64(29999), item.Amount)
				assert.False(t, item.HasUsage)
			},
		},
		{
			name: "Limited Cycle Subscription",
			orderItem: OrderItem{
				OrgId:       "org_789",
				Id:          "item_789",
				OrderId:     "order_789",
				ProductId:   "prod_789",
				Description: "6-Month Plan",
				Price: Price{
					Id:                 "price_789",
					BillingInterval:    prices.BillingIntervalMonth,
					BillingIntervalQty: 1,
					Category:           prices.PriceCategorySubscription,
					Currency:           "EUR",
					UnitPrice:          1999, // €19.99
					TrialInterval:      prices.BillingIntervalNone,
					TrialIntervalQty:   0,
					Cycles:             6, // Limited to 6 cycles
				},
			},
			expectedSettings: func(t *testing.T, sub Subscription) {
				// Cycle limitation
				assert.Equal(t, 6, sub.Cycles)

				// Currency
				item := sub.Items[0]
				assert.Equal(t, "EUR", item.Currency)
				assert.Equal(t, int64(1999), item.Amount)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := NewSubscriptionFromOrderItem(tt.orderItem)

			// Common assertions for all traditional subscriptions
			assert.NotEmpty(t, sub.Id)
			assert.True(t, strings.HasPrefix(sub.Id, "sub_"))
			assert.WithinDuration(t, time.Now().UTC(), sub.CreatedAt, 5*time.Second)
			assert.WithinDuration(t, time.Now().UTC(), sub.UpdatedAt, 5*time.Second)

			// Run test-specific assertions
			tt.expectedSettings(t, sub)
		})
	}
}

// TestNewSubscriptionFromOrderItem_UsageBasedBilling tests creation from usage-based billing order items
func TestNewSubscriptionFromOrderItem_UsageBasedBilling(t *testing.T) {
	tests := []struct {
		name string
		orderItem OrderItem
		expectedSettings func(t *testing.T, sub Subscription)
	}{
		{
			name: "API Calls - Pure Usage (Sum/Count)",
			orderItem: OrderItem{
				OrgId:       "org_api",
				Id:          "item_api",
				OrderId:     "order_api",
				ProductId:   "prod_api",
				Description: "API Calls",
				Price: Price{
					Id:                 "price_api",
					BillingInterval:    prices.BillingIntervalMonth,
					BillingIntervalQty: 1,
					Category:           prices.PriceCategoryUsage,
					Currency:           "USD",
					UnitPrice:          10, // $0.10 per 1000 calls
					UsageType:          prices.UsageTypeMetered,
					UnitType:           prices.UnitTypeCount,
					AggregationType:    prices.AggregationTypeSum,
				},
			},
			expectedSettings: func(t *testing.T, sub Subscription) {
				// Subscription item should have usage configuration
				require.Len(t, sub.Items, 1)
				item := sub.Items[0]

				// Usage flags and configuration
				assert.True(t, item.HasUsage)
				assert.Equal(t, UsageTypeMetered, item.UsageType)
				assert.Equal(t, UnitTypeCount, item.UnitType)
				assert.Equal(t, AggregationTypeSum, item.AggregationType)

				// Pricing should be unit-based
				assert.Equal(t, int64(10), item.UnitPrice)
				assert.Equal(t, int64(0), item.Amount) // No fixed amount for pure usage
				assert.Equal(t, float64(0), item.PercentageRate)
				assert.Equal(t, int64(0), item.FixedFee)
			},
		},
		{
			name: "Storage - Average GB Hours",
			orderItem: OrderItem{
				OrgId:       "org_storage",
				Id:          "item_storage",
				OrderId:     "order_storage",
				ProductId:   "prod_storage",
				Description: "Cloud Storage",
				Price: Price{
					Id:                 "price_storage",
					BillingInterval:    prices.BillingIntervalMonth,
					BillingIntervalQty: 1,
					Category:           prices.PriceCategoryUsage,
					Currency:           "USD",
					UnitPrice:          5, // $0.05 per GB-hour
					UsageType:          prices.UsageTypeMetered,
					UnitType:           prices.UnitTypeGbHours,
					AggregationType:    prices.AggregationTypeAverage,
				},
			},
			expectedSettings: func(t *testing.T, sub Subscription) {
				item := sub.Items[0]
				assert.True(t, item.HasUsage)
				assert.Equal(t, UsageTypeMetered, item.UsageType)
				assert.Equal(t, UnitTypeGBHours, item.UnitType)
				assert.Equal(t, AggregationTypeAverage, item.AggregationType)
				assert.Equal(t, int64(5), item.UnitPrice)
			},
		},
		{
			name: "Active Seats - Max Billing",
			orderItem: OrderItem{
				OrgId:       "org_seats",
				Id:          "item_seats",
				OrderId:     "order_seats",
				ProductId:   "prod_seats",
				Description: "User Seats",
				Price: Price{
					Id:                 "price_seats",
					BillingInterval:    prices.BillingIntervalMonth,
					BillingIntervalQty: 1,
					Category:           prices.PriceCategoryUsage,
					Currency:           "USD",
					UnitPrice:          2000, // $20 per seat
					UsageType:          prices.UsageTypeMetered,
					UnitType:           prices.UnitTypeSeats,
					AggregationType:    prices.AggregationTypeMax,
				},
			},
			expectedSettings: func(t *testing.T, sub Subscription) {
				item := sub.Items[0]
				assert.True(t, item.HasUsage)
				assert.Equal(t, UsageTypeMetered, item.UsageType)
				assert.Equal(t, UnitTypeSeats, item.UnitType)
				assert.Equal(t, AggregationTypeMax, item.AggregationType)
				assert.Equal(t, int64(2000), item.UnitPrice)
			},
		},
		{
			name: "Payment Processing - Percentage + Fixed Fee",
			orderItem: OrderItem{
				OrgId:       "org_payments",
				Id:          "item_payments",
				OrderId:     "order_payments",
				ProductId:   "prod_payments",
				Description: "Payment Processing",
				Price: Price{
					Id:                 "price_payments",
					BillingInterval:    prices.BillingIntervalMonth,
					BillingIntervalQty: 1,
					Category:           prices.PriceCategoryUsage,
					Currency:           "USD",
					PercentageRate:     2.9,  // 2.9%
					FixedFee:           30,   // $0.30 per transaction
					UsageType:          prices.UsageTypeMetered,
					UnitType:           prices.UnitTypeTransactions,
					AggregationType:    prices.AggregationTypeSum,
				},
			},
			expectedSettings: func(t *testing.T, sub Subscription) {
				item := sub.Items[0]
				assert.True(t, item.HasUsage)
				assert.Equal(t, UsageTypeMetered, item.UsageType)
				assert.Equal(t, UnitTypeTransactions, item.UnitType)
				assert.Equal(t, AggregationTypeSum, item.AggregationType)

				// Transaction fees use percentage + fixed fee model
				assert.Equal(t, float64(2.9), item.PercentageRate)
				assert.Equal(t, int64(30), item.FixedFee)
				assert.Equal(t, int64(0), item.UnitPrice) // Not used for percentage-based
				assert.Equal(t, int64(0), item.Amount)    // No fixed amount
			},
		},
		{
			name: "Bandwidth - Sum of GB",
			orderItem: OrderItem{
				OrgId:       "org_bandwidth",
				Id:          "item_bandwidth",
				OrderId:     "order_bandwidth",
				ProductId:   "prod_bandwidth",
				Description: "Data Transfer",
				Price: Price{
					Id:                 "price_bandwidth",
					BillingInterval:    prices.BillingIntervalMonth,
					BillingIntervalQty: 1,
					Category:           prices.PriceCategoryUsage,
					Currency:           "USD",
					UnitPrice:          10, // $0.10 per GB
					UsageType:          prices.UsageTypeMetered,
					UnitType:           prices.UnitTypeStorage,
					AggregationType:    prices.AggregationTypeSum,
				},
			},
			expectedSettings: func(t *testing.T, sub Subscription) {
				item := sub.Items[0]
				assert.True(t, item.HasUsage)
				assert.Equal(t, UsageTypeMetered, item.UsageType)
				assert.Equal(t, UnitType("storage"), item.UnitType)
				assert.Equal(t, AggregationTypeSum, item.AggregationType)
				assert.Equal(t, int64(10), item.UnitPrice)
			},
		},
		{
			name: "Compute Time - Last Value During Period",
			orderItem: OrderItem{
				OrgId:       "org_compute",
				Id:          "item_compute",
				OrderId:     "order_compute",
				ProductId:   "prod_compute",
				Description: "Compute Minutes",
				Price: Price{
					Id:                 "price_compute",
					BillingInterval:    prices.BillingIntervalMonth,
					BillingIntervalQty: 1,
					Category:           prices.PriceCategoryUsage,
					Currency:           "USD",
					UnitPrice:          5, // $0.05 per minute
					UsageType:          prices.UsageTypeMetered,
					UnitType:           prices.UnitTypeCustom,
					AggregationType:    prices.AggregationTypeLastDuringPeriod,
				},
			},
			expectedSettings: func(t *testing.T, sub Subscription) {
				item := sub.Items[0]
				assert.True(t, item.HasUsage)
				assert.Equal(t, UsageTypeMetered, item.UsageType)
				assert.Equal(t, UnitType("custom"), item.UnitType)
				assert.Equal(t, AggregationTypeLastDuringPeriod, item.AggregationType)
				assert.Equal(t, int64(5), item.UnitPrice)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := NewSubscriptionFromOrderItem(tt.orderItem)

			// Common assertions for all usage-based subscriptions
			assert.NotEmpty(t, sub.Id)
			assert.True(t, strings.HasPrefix(sub.Id, "sub_"))
			assert.Equal(t, tt.orderItem.OrgId, sub.OrgId)
			assert.Equal(t, tt.orderItem.OrderId, sub.OrderId)
			assert.Equal(t, tt.orderItem.Id, sub.OrderItemId)
			assert.Equal(t, SubscriptionStatusPending, sub.Status)

			// Billing configuration
			assert.Equal(t, tt.orderItem.Price.BillingInterval, sub.BillingInterval)
			assert.Equal(t, tt.orderItem.Price.BillingIntervalQty, sub.BillingIntervalQty)

			// Run test-specific assertions
			tt.expectedSettings(t, sub)
		})
	}
}

// TestNewSubscriptionFromOrderItem_HybridBilling tests creation from hybrid (fixed + usage) order items
func TestNewSubscriptionFromOrderItem_HybridBilling(t *testing.T) {
	tests := []struct {
		name string
		orderItem OrderItem
		expectedSettings func(t *testing.T, sub Subscription)
	}{
		{
			name: "Base Plan + Overage API Calls",
			orderItem: OrderItem{
				OrgId:       "org_hybrid",
				Id:          "item_hybrid",
				OrderId:     "order_hybrid",
				ProductId:   "prod_hybrid",
				Description: "Pro Plan with API Overages",
				Price: Price{
					Id:                 "price_hybrid",
					BillingInterval:    prices.BillingIntervalMonth,
					BillingIntervalQty: 1,
					Category:           prices.PriceCategoryHybrid,
					Currency:           "USD",
					UnitPrice:          4900, // $49 base fee
					UsageType:          prices.UsageTypeMetered,
					UnitType:           prices.UnitTypeCount,
					AggregationType:    prices.AggregationTypeSum,
					OverageUnitPrice:   1, // $0.001 per API call over limit
				},
			},
			expectedSettings: func(t *testing.T, sub Subscription) {
				item := sub.Items[0]

				// Should have both fixed amount and usage configuration
				assert.Equal(t, int64(4900), item.Amount) // Fixed base fee
				assert.True(t, item.HasUsage)
				assert.Equal(t, UsageTypeMetered, item.UsageType)
				assert.Equal(t, UnitTypeCount, item.UnitType)
				assert.Equal(t, AggregationTypeSum, item.AggregationType)

				// Overage pricing
				assert.Equal(t, int64(1), item.UnitPrice) // Overage rate
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := NewSubscriptionFromOrderItem(tt.orderItem)

			// Common assertions
			assert.Equal(t, tt.orderItem.OrgId, sub.OrgId)
			assert.Equal(t, SubscriptionStatusPending, sub.Status)

			// Run test-specific assertions
			tt.expectedSettings(t, sub)
		})
	}
}

// TestNewSubscriptionFromOrderItem_EdgeCases tests edge cases and validation
func TestNewSubscriptionFromOrderItem_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		orderItem OrderItem
		expectedSettings func(t *testing.T, sub Subscription)
	}{
		{
			name: "Zero Amount Subscription",
			orderItem: OrderItem{
				OrgId:       "org_free",
				Id:          "item_free",
				OrderId:     "order_free",
				ProductId:   "prod_free",
				Description: "Free Plan",
				Price: Price{
					Id:                 "price_free",
					BillingInterval:    prices.BillingIntervalMonth,
					BillingIntervalQty: 1,
					Category:           prices.PriceCategorySubscription,
					Currency:           "USD",
					UnitPrice:          0, // Free
				},
			},
			expectedSettings: func(t *testing.T, sub Subscription) {
				item := sub.Items[0]
				assert.Equal(t, int64(0), item.Amount)
				assert.False(t, item.HasUsage)
				assert.Equal(t, "USD", item.Currency)
			},
		},
		{
			name: "Very High Frequency Billing",
			orderItem: OrderItem{
				OrgId:       "org_frequent",
				Id:          "item_frequent",
				OrderId:     "order_frequent",
				ProductId:   "prod_frequent",
				Description: "Hourly Plan",
				Price: Price{
					Id:                 "price_frequent",
					BillingInterval:    prices.BillingIntervalHour,
					BillingIntervalQty: 1,
					Category:           prices.PriceCategorySubscription,
					Currency:           "USD",
					UnitPrice:          100, // $1.00 per hour
				},
			},
			expectedSettings: func(t *testing.T, sub Subscription) {
				assert.Equal(t, prices.BillingIntervalHour, sub.BillingInterval)
				assert.Equal(t, 1, sub.BillingIntervalQty)
				assert.Equal(t, int64(100), sub.Items[0].Amount)
			},
		},
		{
			name: "Multi-Year Billing with Large Interval",
			orderItem: OrderItem{
				OrgId:       "org_longtime",
				Id:          "item_longtime",
				OrderId:     "order_longtime",
				ProductId:   "prod_longtime",
				Description: "5-Year Plan",
				Price: Price{
					Id:                 "price_longtime",
					BillingInterval:    prices.BillingIntervalYear,
					BillingIntervalQty: 5, // Every 5 years
					Category:           prices.PriceCategorySubscription,
					Currency:           "USD",
					UnitPrice:          99999, // $999.99
				},
			},
			expectedSettings: func(t *testing.T, sub Subscription) {
				assert.Equal(t, prices.BillingIntervalYear, sub.BillingInterval)
				assert.Equal(t, 5, sub.BillingIntervalQty)
				assert.Equal(t, int64(99999), sub.Items[0].Amount)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := NewSubscriptionFromOrderItem(tt.orderItem)

			// Common assertions
			assert.Equal(t, tt.orderItem.OrgId, sub.OrgId)
			assert.Equal(t, SubscriptionStatusPending, sub.Status)
			require.Len(t, sub.Items, 1)

			// Run test-specific assertions
			tt.expectedSettings(t, sub)
		})
	}
}
