package subscriptions

import (
	"payloop/internal/domain/entities"
)

type UpdateSubscriptionRequest struct {
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

type PauseSubscriptionInput struct {
	OrgId  string `json:"org_id"`
	Id     string `json:"id"`
	Reason string `json:"reason"`
}

type ResumeSubscriptionInput struct {
	OrgId          string                     `json:"org_id"`
	Id             string                     `json:"id"`
	ResumeBehavior SubscriptionResumeBehavior `json:"resume_behavior"`
}

type CancelSubscriptionInput struct {
	OrgId  string `json:"org_id"`
	Id     string `json:"id"`
	Reason string `json:"reason"`
}

// ChangePlanInput represents the input for changing a subscription's plan
type ChangePlanInput struct {
	OrgId          string `json:"org_id"`
	Id             string `json:"id"`
	NewVariantId   string `json:"new_variant_id" binding:"required"`
	NewPriceId     string `json:"new_price_id" binding:"required"`
	ProrationMode  string `json:"proration_mode" binding:"oneof=none immediate credit_unused"`
	EffectiveDate  string `json:"effective_date" binding:"oneof=immediate next_billing_cycle"`
	Reason         string `json:"reason"`
}

type ProcessSubscriptionChargeInput struct {
	Subscription entities.Subscription `json:"subscription"`
}
