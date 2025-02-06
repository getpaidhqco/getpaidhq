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
	assert.WithinDuration(t, time.Now().UTC().AddDate(0, 1, 0), *subscription.TrialEndsAt, 10*time.Second)

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
	assert.Nil(t, subscription.NextRetry)
	assert.Equal(t, 0, subscription.CyclesProcessed)
	assert.Equal(t, 0, subscription.TotalRevenue)
	assert.Nil(t, subscription.CancelledAt)
	assert.WithinDuration(t, now, subscription.CreatedAt, 5*time.Second)
	assert.WithinDuration(t, now, subscription.UpdatedAt, 5*time.Second)
}
