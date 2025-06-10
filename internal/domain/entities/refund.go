package entities

import "time"

type RefundStatus string

const (
	RefundStatusPending   RefundStatus = "pending"
	RefundStatusCompleted RefundStatus = "completed"
	RefundStatusError     RefundStatus = "error"
)

type Refund struct {
	OrgId       string       `json:"org_id" `
	Id          string       `json:"id" `
	PspRefundId string       `json:"psp_refund_id" `
	PaymentId   string       `json:"payment_id" `
	Amount      int64        `json:"amount" `
	Currency    string       `json:"currency" `
	Reason      string       `json:"reason,omitempty" ` // Nullable field
	Status      RefundStatus `json:"status" `
	RefundedAt  time.Time    `json:"refunded_at" `
	CompletedAt *time.Time   `json:"completed_at,omitempty" ` // Nullable field
	CreatedAt   time.Time    `json:"created_at" `
	UpdatedAt   time.Time    `json:"updated_at" `
}
