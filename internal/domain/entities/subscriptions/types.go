package subscriptions

import (
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payments"
)

type StoreSubscriptionPaymentInput struct {
	Subscription entities.Subscription `json:"subscription"`
	ChargeResult payments.ChargeResult `json:"charge_result"`
}
