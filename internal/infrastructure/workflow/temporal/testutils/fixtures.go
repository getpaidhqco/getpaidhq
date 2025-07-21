package testutils

import (
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities/prices"
	"time"

	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payments"
)

// CreateFastSubscription creates a subscription with fast billing cycles for testing
func CreateFastSubscription(orgId, customerId string, amount int) entities.Subscription {
	now := time.Now()
	return entities.Subscription{
		Id:              "test_sub_id",
		OrgId:           orgId,
		CustomerId:      customerId,
		Status:          entities.SubscriptionStatusActive,
		Amount:          int64(amount),
		Currency:        "USD",
		RenewsAt:        now.Add(time.Second * 5), // 5 seconds for testing
		CreatedAt:       now,
		UpdatedAt:       now,
		CyclesProcessed: 0,
		TotalRevenue:    0,
		BillingInterval: prices.BillingIntervalMonth,
		PspId:           common.Paystack,
	}
}

// CreateExpiredSubscription creates a subscription that's already expired
func CreateExpiredSubscription(orgId, customerId string) entities.Subscription {
	sub := CreateFastSubscription(orgId, customerId, 1000)
	sub.Status = entities.SubscriptionStatusExpired
	return sub
}

// CreateCancelledSubscription creates a cancelled subscription
func CreateCancelledSubscription(orgId, customerId string) entities.Subscription {
	sub := CreateFastSubscription(orgId, customerId, 1000)
	sub.Status = entities.SubscriptionStatusCancelled
	return sub
}

// CreatePausedSubscription creates a paused subscription
func CreatePausedSubscription(orgId, customerId string) entities.Subscription {
	sub := CreateFastSubscription(orgId, customerId, 1000)
	sub.Status = entities.SubscriptionStatusPaused
	return sub
}

// CreatePastDueSubscription creates a subscription that's past due
func CreatePastDueSubscription(orgId, customerId string) entities.Subscription {
	sub := CreateFastSubscription(orgId, customerId, 1000)
	sub.Status = entities.SubscriptionStatusPastDue
	sub.RenewsAt = time.Now().Add(-time.Hour) // Past due
	return sub
}

// DunningWorkflowInput represents the input for the DunningWorkflow (copied from workflows package)
type DunningWorkflowInput struct {
	OrgId                string                `json:"org_id"`
	SubscriptionId       string                `json:"subscription_id"`
	CustomerId           string                `json:"customer_id"`
	FailedAmount         int                   `json:"failed_amount"`
	Currency             string                `json:"currency"`
	InitialFailureReason string                `json:"initial_failure_reason,omitempty"`
	ParentWorkflowId     string                `json:"parent_workflow_id,omitempty"`
	PaymentResult        payments.ChargeResult `json:"payment_result"`
	Metadata             map[string]string     `json:"metadata,omitempty"`
}

// CreateDunningWorkflowInput creates test input for dunning workflow
func CreateDunningWorkflowInput(orgId, subscriptionId, customerId string) DunningWorkflowInput {
	return DunningWorkflowInput{
		OrgId:                orgId,
		SubscriptionId:       subscriptionId,
		CustomerId:           customerId,
		FailedAmount:         1000,
		Currency:             "USD",
		InitialFailureReason: "insufficient_funds",
		PaymentResult:        MockFailedChargeResult(1000),
		Metadata: map[string]string{
			"test": "true",
		},
	}
}

// CreateSubscriptionWithNextChargeDate creates a subscription with specific next charge date
func CreateSubscriptionWithNextChargeDate(orgId, customerId string, nextCharge time.Time) entities.Subscription {
	sub := CreateFastSubscription(orgId, customerId, 1000)
	sub.RenewsAt = nextCharge
	return sub
}

// CreateUpdatedSubscription creates an updated subscription after charge
func CreateUpdatedSubscription(original entities.Subscription, status entities.SubscriptionStatus) entities.Subscription {
	updated := original
	updated.Status = status
	updated.CyclesProcessed++
	updated.TotalRevenue += original.Amount
	updated.RenewsAt = time.Now().Add(time.Second * 5) // Next cycle in 5 seconds
	updated.UpdatedAt = time.Now()
	return updated
}
