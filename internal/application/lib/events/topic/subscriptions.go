package topic

import (
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payments"
)

type SubscriptionPaymentChargeSuccessEvent struct {
	OrgId          string            `json:"org_id"`
	SubscriptionId string            `json:"subscription_id"`
	OrderId        string            `json:"order_id"`
	PaymentId      string            `json:"payment_id"`
	Metadata       map[string]string `json:"metadata"`
	Payment        entities.Payment  `json:"payment"`
}

type SubscriptionPaymentChargeFailureEvent struct {
	Subscription entities.Subscription `json:"subscription"`
	ChargeResult payments.ChargeResult `json:"charge_result"`
}

func NewSubscriptionPaymentChargeSuccessEvent(sub entities.Subscription, payment entities.Payment) SubscriptionPaymentChargeSuccessEvent {
	return SubscriptionPaymentChargeSuccessEvent{
		OrgId:          sub.OrgId,
		SubscriptionId: sub.Id,
		OrderId:        sub.OrderId,
		PaymentId:      payment.Id,
		Metadata:       sub.Metadata,
		Payment:        payment,
	}
}

func GetSubscriptionTopic(status entities.SubscriptionStatus) string {
	switch status {
	case entities.SubscriptionStatusActive:
		return TopicSubscriptionActivated
	case entities.SubscriptionStatusPaused:
		return TopicSubscriptionPaused
	case entities.SubscriptionStatusCancelled:
		return TopicSubscriptionCancelled
	case entities.SubscriptionStatusExpired:
		return SubscriptionStatusExpired
	case entities.SubscriptionStatusPastDue:
		return SubscriptionStatusExpired

	default:
		return ""
	}
}
