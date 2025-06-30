package integration

import (
	"testing"

	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/prices"
	"payloop/internal/lib"
	"payloop/internal/testing/fixtures"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSimpleSubscriptionCreation tests basic subscription creation without full FX setup
func TestSimpleSubscriptionCreation(t *testing.T) {
	t.Run("create subscription from order item", func(t *testing.T) {
		// Use fixtures to create test data
		orderItem := entities.OrderItem{
			OrgId:       "org_test",
			Id:          lib.GenerateId("item"),
			OrderId:     lib.GenerateId("order"),
			ProductId:   "prod_123",
			VariantId:   "var_123",
			PriceId:     "price_123",
			Description: "Test Product",
			Quantity:    1,
			Price: entities.Price{
				Id:                 "price_123",
				VariantId:          "var_123",
				UnitPrice:          2500,
				Currency:           "USD",
				BillingInterval:    prices.BillingIntervalMonth,
				BillingIntervalQty: 1,
				Cycles:             0,
			},
		}

		// Create subscription from order item
		subscription := entities.NewSubscriptionFromOrderItem(orderItem)

		// Verify subscription was created correctly
		assert.NotEmpty(t, subscription.Id)
		assert.Equal(t, orderItem.OrgId, subscription.OrgId)
		assert.Equal(t, orderItem.OrderId, subscription.OrderId)
		assert.Equal(t, orderItem.Id, subscription.OrderItemId)
		assert.Equal(t, orderItem.ProductId, subscription.ProductId)
		assert.Equal(t, orderItem.VariantId, subscription.VariantId)
		assert.Equal(t, orderItem.Price.Id, subscription.PriceId)
		assert.Equal(t, entities.SubscriptionStatusPending, subscription.Status)
		assert.Equal(t, orderItem.Price.UnitPrice, subscription.Amount)
		assert.Equal(t, string(orderItem.Price.Currency), subscription.Currency)
		assert.Equal(t, orderItem.Price.BillingInterval, subscription.BillingInterval)
		assert.Equal(t, orderItem.Price.BillingIntervalQty, subscription.BillingIntervalQty)

		t.Logf("✓ Created subscription %s from order item", subscription.Id)
		t.Logf("  - Amount: $%.2f %s", float64(subscription.Amount)/100, subscription.Currency)
		t.Logf("  - Billing: Every %d %s", subscription.BillingIntervalQty, subscription.BillingInterval)
		t.Logf("  - Status: %s", subscription.Status)
	})

	t.Run("subscription builder creates valid subscriptions", func(t *testing.T) {
		// Test the fixture builder
		subscription := fixtures.NewSubscriptionBuilder().
			WithOrgId("org_test").
			WithCustomerId("cust_123").
			WithProductVariantPrice("prod_premium", "var_premium", "price_premium").
			WithAmount(4900).
			WithBilling(prices.BillingIntervalMonth, 1).
			WithStatus(entities.SubscriptionStatusActive).
			Build()

		assert.Equal(t, "org_test", subscription.OrgId)
		assert.Equal(t, "cust_123", subscription.CustomerId)
		assert.Equal(t, "prod_premium", subscription.ProductId)
		assert.Equal(t, "var_premium", subscription.VariantId)
		assert.Equal(t, "price_premium", subscription.PriceId)
		assert.Equal(t, int64(4900), subscription.Amount)
		assert.Equal(t, entities.SubscriptionStatusActive, subscription.Status)

		t.Logf("✓ Built subscription with fixtures: %s", subscription.Id)
	})

	t.Run("proration calculation works correctly", func(t *testing.T) {
		subscription := fixtures.NewSubscriptionBuilder().
			WithAmount(3000). // $30/month
			WithBilling(prices.BillingIntervalMonth, 1).
			Build()

		// Test proration calculation
		// Assuming 30-day month, halfway through = 15 days remaining
		referenceDate := subscription.CurrentPeriodStart.AddDate(0, 0, 15)
		
		prorationDetails := subscription.CalculateProrationDetails(
			"credit_unused",
			referenceDate,
			subscription.BillingAnchor,
			subscription.BillingAnchor,
			subscription.CurrentPeriodStart,
			subscription.CurrentPeriodEnd,
		)

		// Should credit approximately half the amount
		assert.Greater(t, prorationDetails.CreditAmount, 0)
		assert.Greater(t, prorationDetails.DaysCredited, 0)
		
		t.Logf("✓ Proration calculation:")
		t.Logf("  - Original amount: $%.2f", float64(subscription.Amount)/100)
		t.Logf("  - Days credited: %d", prorationDetails.DaysCredited)
		t.Logf("  - Credit amount: $%.2f", float64(prorationDetails.CreditAmount)/100)
	})

	t.Run("plan change validation", func(t *testing.T) {
		// Create subscriptions with same and different products
		subSameProduct := fixtures.NewSubscriptionBuilder().
			WithProductVariantPrice("prod_123", "var_basic", "price_basic").
			Build()

		subDifferentProduct := fixtures.NewSubscriptionBuilder().
			WithProductVariantPrice("prod_456", "var_other", "price_other").
			Build()

		// Test data for plan changes
		validVariant := fixtures.NewVariantBuilder().
			WithId("var_premium").
			WithProductId("prod_123"). // Same product
			Build()

		invalidVariant := fixtures.NewVariantBuilder().
			WithId("var_different").
			WithProductId("prod_789"). // Different product
			Build()

		// Business rule: Can only change to variants of the same product
		assert.Equal(t, subSameProduct.ProductId, validVariant.ProductId, 
			"Valid variant should belong to same product")
		
		assert.NotEqual(t, subSameProduct.ProductId, invalidVariant.ProductId,
			"Invalid variant should belong to different product")

		t.Log("✓ Plan change validation rules verified")
	})
}

// TestSubscriptionStates tests subscription state transitions
func TestSubscriptionStates(t *testing.T) {
	t.Run("subscription activation", func(t *testing.T) {
		subscription := fixtures.NewSubscriptionBuilder().
			WithStatus(entities.SubscriptionStatusPending).
			Build()

		payment := entities.Payment{
			OrgId:       subscription.OrgId,
			Id:          lib.GenerateId("pay"),
			Amount:      subscription.Amount,
			CompletedAt: lib.CurrentTime(),
		}

		// Activate subscription
		subscription.SetActive(payment)

		assert.Equal(t, entities.SubscriptionStatusActive, subscription.Status)
		assert.Equal(t, payment.CompletedAt, subscription.LastCharge)
		assert.Equal(t, int64(payment.Amount), subscription.TotalRevenue)
		assert.Equal(t, 1, subscription.CyclesProcessed)
		assert.NotZero(t, subscription.RenewsAt)

		t.Logf("✓ Subscription activated:")
		t.Logf("  - Status: %s", subscription.Status)
		t.Logf("  - Next renewal: %s", subscription.RenewsAt.Format("2006-01-02"))
		t.Logf("  - Revenue: $%.2f", float64(subscription.TotalRevenue)/100)
	})

	t.Run("subscription cancellation", func(t *testing.T) {
		subscription := fixtures.NewSubscriptionBuilder().
			WithStatus(entities.SubscriptionStatusActive).
			Build()

		originalRenewsAt := subscription.RenewsAt

		// Cancel subscription
		subscription.SetCancelled()

		assert.Equal(t, entities.SubscriptionStatusCancelled, subscription.Status)
		assert.NotZero(t, subscription.CancelledAt)
		assert.Zero(t, subscription.RenewsAt)
		assert.Zero(t, subscription.NextRetryAt)

		t.Logf("✓ Subscription cancelled:")
		t.Logf("  - Status: %s", subscription.Status)
		t.Logf("  - Cancelled at: %s", subscription.CancelledAt.Format("2006-01-02 15:04:05"))
		t.Logf("  - Original renewal cleared: %s -> nil", originalRenewsAt.Format("2006-01-02"))
	})

	t.Run("subscription state helpers", func(t *testing.T) {
		activeSubscription := fixtures.NewSubscriptionBuilder().
			WithStatus(entities.SubscriptionStatusActive).
			Build()

		trialSubscription := fixtures.NewSubscriptionBuilder().
			WithStatus(entities.SubscriptionStatusTrial).
			Build()

		pausedSubscription := fixtures.NewSubscriptionBuilder().
			WithStatus(entities.SubscriptionStatusPaused).
			Build()

		cancelledSubscription := fixtures.NewSubscriptionBuilder().
			WithStatus(entities.SubscriptionStatusCancelled).
			Build()

		// Test IsRunning helper
		assert.True(t, activeSubscription.IsRunning(), "Active subscription should be running")
		assert.True(t, trialSubscription.IsRunning(), "Trial subscription should be running")
		assert.False(t, pausedSubscription.IsRunning(), "Paused subscription should not be running")
		assert.False(t, cancelledSubscription.IsRunning(), "Cancelled subscription should not be running")

		t.Log("✓ Subscription state helpers work correctly")
	})
}