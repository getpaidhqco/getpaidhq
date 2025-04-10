package entities

import "time"

type Refund struct {
	OrgId       string    `json:"org_id" `
	Id          string    `json:"id" `
	PspRefundId string    `json:"psp_refund_id" `
	PaymentId   string    `json:"payment_id" `
	Amount      int64     `json:"amount" `
	Currency    string    `json:"currency" `
	Reason      string    `json:"reason,omitempty" ` // Nullable field
	RefundedAt  time.Time `json:"refunde_at" `
	CreatedAt   time.Time `json:"created_at" `
	UpdatedAt   time.Time `json:"updated_at" `
}
