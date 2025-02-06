package subscriptions

import (
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/prices"
)

type UpdateSubscriptionRequest struct {
	OrgId                string                      `json:"org_id"`
	Id                   string                      `json:"id"`
	Status               entities.SubscriptionStatus `json:"status"`
	DefaultPaymentMethod string                      `json:"default_payment_method"`
	Metadata             map[string]string           `json:"metadata"`
}

type UpdateSubscriptionInput struct {
	OrgId                string                      `json:"org_id"`
	Id                   string                      `json:"id"`
	Status               entities.SubscriptionStatus `json:"status"`
	DefaultPaymentMethod string                      `json:"default_payment_method"`
	Metadata             map[string]string           `json:"metadata"`
}

type CreateSubscriptionInput struct {
	OrgId string `json:"org_id"`

	PaymentMethodId string `json:"payment_method_id" binding:"required"`
	Activate        bool   `json:"activate"`

	Amount   int    `json:"amount"  binding:"required"`
	Currency string `json:"currency"  binding:"required"`

	BillingInterval    prices.BillingInterval `json:"billing_interval"  binding:"required"`
	BillingIntervalQty int                    `json:"billing_interval_qty"  binding:"required"`
	Cycles             int                    `json:"cycles"`

	TrialInterval    prices.BillingInterval `json:"trial_interval"`
	TrialIntervalQty int                    `json:"trial_interval_qty"`

	Metadata map[string]string `json:"metadata"`
}
