package response

import (
	"payloop/internal/domain/entities"
	"time"
)

// Refund represents a refund response
type Refund struct {
	OrgId       string    `json:"org_id"`
	Id          string    `json:"id"`
	PspRefundId string    `json:"psp_refund_id,omitempty"`
	PaymentId   string    `json:"payment_id"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
	Reason      string    `json:"reason,omitempty"`
	RefundedAt  time.Time `json:"refunded_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewRefundFromEntity creates a new Refund response from a Refund entity
func NewRefundFromEntity(entity entities.Refund) Refund {
	return Refund{
		OrgId:       entity.OrgId,
		Id:          entity.Id,
		PspRefundId: entity.PspRefundId,
		PaymentId:   entity.PaymentId,
		Amount:      entity.Amount,
		Currency:    entity.Currency,
		Reason:      entity.Reason,
		RefundedAt:  entity.RefundedAt,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}
}