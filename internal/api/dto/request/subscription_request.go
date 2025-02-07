package request

import (
	"payloop/internal/domain/entities/prices"
	"payloop/internal/domain/entities/subscriptions"
)

type CreateSubscriptionRequest struct {
	PaymentMethodId string `json:"payment_method_id" binding:"required"`

	Activate bool `json:"activate"`

	Amount   int    `json:"amount"  binding:"required"`
	Currency string `json:"currency"  binding:"required"`

	BillingInterval    prices.BillingInterval `json:"billing_interval"  binding:"required"`
	BillingIntervalQty int                    `json:"billing_interval_qty"  binding:"required"`
	Cycles             int                    `json:"cycles"`

	TrialInterval    prices.BillingInterval `json:"trial_interval"`
	TrialIntervalQty int                    `json:"trial_interval_qty"`

	Metadata map[string]string `json:"metadata"`
}

type ActivateSubscriptionRequest struct {
	PaymentMethodId string `json:"payment_method_id" binding:"required"`

	Amount   int    `json:"amount"  binding:"required"`
	Currency string `json:"currency"  binding:"required"`

	BillingInterval    prices.BillingInterval `json:"billing_interval"  binding:"required"`
	BillingIntervalQty int                    `json:"billing_interval_qty"  binding:"required"`
	Cycles             int                    `json:"cycles"`

	TrialInterval    prices.BillingInterval `json:"trial_interval"`
	TrialIntervalQty int                    `json:"trial_interval_qty"`

	Metadata map[string]string `json:"metadata"`
}

type PauseSubscriptionRequest struct {
	Reason string `json:"reason"`
}

type ResumeSubscriptionRequest struct {
	ResumeBehavior subscriptions.SubscriptionResumeBehavior `json:"resume_behavior"`
}
