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

type ProcessSubscriptionChargeInput struct {
	Subscription entities.Subscription `json:"subscription"`
}
