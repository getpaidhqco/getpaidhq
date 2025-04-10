package models

import (
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
	"time"
)

type Refund struct {
	OrgId       string      `json:"org_id"`
	Id          string      `json:"id"`
	PspRefundId pgtype.Text `json:"psp_refund_id"`
	PaymentId   string      `json:"payment_id"`
	Amount      int64       `json:"amount"`
	Currency    string      `json:"currency"`
	Reason      pgtype.Text `json:"reason,omitempty"` // Nullable field
	RefundedAt  time.Time   `json:"refunded_at"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

func (r *Refund) ToEntity() entities.Refund {
	return entities.Refund{
		OrgId:       r.OrgId,
		Id:          r.Id,
		PspRefundId: r.PspRefundId.String,
		PaymentId:   r.PaymentId,
		Amount:      r.Amount,
		Currency:    r.Currency,
		Reason:      r.Reason.String,
		RefundedAt:  r.RefundedAt,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}
