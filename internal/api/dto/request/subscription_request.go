package request

import (
	"payloop/internal/application/dto"
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
type UpdateBillingAnchorRequest struct {
	// BillingAnchor the new billing anchor as a day between 1 and 31. If the day is not valid for the current month, it will be adjusted to the last day of the month.
	BillingAnchor int              `json:"billing_anchor" binding:"required,gte=1,lte=31"`
	ProrationMode dto.ProrationMode `json:"proration_mode" binding:"required,oneof=none credit_unused"`
}

type ResumeSubscriptionRequest struct {
	ResumeBehavior subscriptions.SubscriptionResumeBehavior `json:"resume_behavior"`
}

type ChangePlanRequest struct {
	NewVariantId   string `json:"new_variant_id" binding:"required"`
	NewPriceId     string `json:"new_price_id" binding:"required"`
	ProrationMode  string `json:"proration_mode" binding:"oneof=none immediate credit_unused"`
	EffectiveDate  string `json:"effective_date" binding:"oneof=immediate next_billing_cycle"`
	Reason         string `json:"reason"`
}
