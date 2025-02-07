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
