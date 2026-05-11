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
