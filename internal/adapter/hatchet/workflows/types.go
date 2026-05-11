package workflows

import (
	"payloop/internal/core/domain"
	"time"
)

// PaymentSuccessInput is the workflow input for the payment-success DAG.
type PaymentSuccessInput struct {
	PaymentContext domain.PaymentWebhookContext `json:"payment_context"`
}

// PaymentRefundedInput is the workflow input for the payment-refunded workflow.
type PaymentRefundedInput struct {
	PaymentContext domain.PaymentWebhookContext `json:"payment_context"`
}

// BillingCycleInput is the input for the billing-cycle DAG.
type BillingCycleInput struct {
	Subscription domain.Subscription `json:"subscription"`
}

// ReminderInput is the input for the subscription-charge-reminder durable task.
type ReminderInput struct {
	Subscription domain.Subscription `json:"subscription"`
	ReminderAt   time.Time           `json:"reminder_at"`
}

// DunningRunnerInput is the input for the dunning-runner durable task. Only
// the OrgId + CampaignId are strictly required; everything else is carried for
// observability + debugging on the Hatchet UI.
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

// DunningAttemptInput is the input for the dunning-attempt DAG run.
type DunningAttemptInput struct {
	OrgId         string                    `json:"org_id"`
	CampaignId    string                    `json:"campaign_id"`
	AttemptNumber int                       `json:"attempt_number"`
	AttemptType   domain.DunningAttemptType `json:"attempt_type"`
}
