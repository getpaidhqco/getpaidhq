package workflows

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// PaymentSuccessInput is the workflow input for the payment-success workflow.
type PaymentSuccessInput struct {
	PaymentContext domain.PaymentWebhookContext `json:"payment_context"`
}

// BillingCycleInput is the input for the billing-cycle child workflow.
type BillingCycleInput struct {
	Subscription domain.Subscription `json:"subscription"`
}

// ReminderInput is the input for the subscription-charge-reminder workflow.
type ReminderInput struct {
	Subscription domain.Subscription `json:"subscription"`
	ReminderAt   time.Time           `json:"reminder_at"`
}

// DunningRunnerInput is the input for the dunning-runner workflow.
type DunningRunnerInput struct {
	OrgId                string              `json:"org_id"`
	CampaignId           string              `json:"campaign_id"`
	SubscriptionId       string              `json:"subscription_id"`
	CustomerId           string              `json:"customer_id"`
	FailedAmount         int64               `json:"failed_amount"`
	Currency             string              `json:"currency"`
	InitialFailureReason string              `json:"initial_failure_reason"`
	PaymentResult        domain.ChargeResult `json:"payment_result"`
	Metadata             map[string]string   `json:"metadata"`
}

// DunningAttemptInput is the input for the dunning-attempt child workflow.
type DunningAttemptInput struct {
	OrgId         string                    `json:"org_id"`
	CampaignId    string                    `json:"campaign_id"`
	AttemptNumber int                       `json:"attempt_number"`
	AttemptType   domain.DunningAttemptType `json:"attempt_type"`
}
