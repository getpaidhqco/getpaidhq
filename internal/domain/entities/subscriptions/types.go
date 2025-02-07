package subscriptions

import (
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payments"
)

type StoreSubscriptionPaymentInput struct {
	Subscription entities.Subscription `json:"subscription"`
	ChargeResult payments.ChargeResult `json:"charge_result"`
}

type Status string

const (
	StatusTrial     Status = "trial"
	StatusActive    Status = "active"
	StatusPastDue   Status = "past_due"
	StatusPaused    Status = "paused"
	StatusUnpaid    Status = "unpaid"
	StatusCancelled Status = "cancelled"
	StatusPending   Status = "pending"
	StatusExpired   Status = "expired"
)
