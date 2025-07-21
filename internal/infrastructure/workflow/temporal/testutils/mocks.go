package testutils

import (
	"time"

	"payloop/internal/domain/entities/dunning"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/domain/entities/settings"
)

// MockSubscriptionSettings returns test subscription settings with fast timing
func MockSubscriptionSettings() settings.Subscription {
	return settings.Subscription{
		ReminderDays: 0, // No reminder delay in tests
	}
}

// MockDunningConfig returns test dunning config with fast intervals
func MockDunningConfig() dunning.DunningConfig {
	return dunning.DefaultDunningConfig()
}

// MockChargeResult creates a test charge result
func MockChargeResult(status payments.PaymentStatus, amount int) payments.ChargeResult {
	return payments.ChargeResult{
		Status:    status,
		Amount:    int64(amount),
		Reference: "test_charge_ref",
		Currency:  "USD",
	}
}

// MockSuccessfulChargeResult returns a successful charge result
func MockSuccessfulChargeResult(amount int) payments.ChargeResult {
	return MockChargeResult(payments.PaymentStatusSucceeded, amount)
}

// MockFailedChargeResult returns a failed charge result
func MockFailedChargeResult(amount int) payments.ChargeResult {
	return MockChargeResult(payments.PaymentStatusFailed, amount)
}

// MockPendingChargeResult returns a pending charge result
func MockPendingChargeResult(amount int) payments.ChargeResult {
	return MockChargeResult(payments.PaymentStatusPending, amount)
}

// MockDunningCampaign creates a test dunning campaign
func MockDunningCampaign(orgId, subscriptionId, customerId string) dunning.DunningCampaign {
	return dunning.DunningCampaign{
		Id:             "test_campaign_id",
		OrgId:          orgId,
		SubscriptionId: subscriptionId,
		CustomerId:     customerId,
		Status:         dunning.DunningStatusActive,
		FailedAmount:   1000,
		Currency:       "USD",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// ChargeAttemptCounter helps simulate progressive failures and eventual success
type ChargeAttemptCounter struct {
	attempt       int
	failuresUntil int
	successAmount int
	failureAmount int
}

// NewChargeAttemptCounter creates a new charge attempt counter
func NewChargeAttemptCounter(failuresUntil, successAmount, failureAmount int) *ChargeAttemptCounter {
	return &ChargeAttemptCounter{
		attempt:       0,
		failuresUntil: failuresUntil,
		successAmount: successAmount,
		failureAmount: failureAmount,
	}
}

// NextAttempt returns the result for the next charge attempt
func (c *ChargeAttemptCounter) NextAttempt() payments.ChargeResult {
	c.attempt++
	if c.attempt <= c.failuresUntil {
		return MockFailedChargeResult(c.failureAmount)
	}
	return MockSuccessfulChargeResult(c.successAmount)
}

// Reset resets the counter
func (c *ChargeAttemptCounter) Reset() {
	c.attempt = 0
}
