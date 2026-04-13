package domain

import "time"

type Refund struct {
	OrgId       string    `json:"org_id"`
	Id          string    `json:"id"`
	PspRefundId string    `json:"psp_refund_id"`
	PaymentId   string    `json:"payment_id"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
	Reason      string    `json:"reason,omitempty"`
	RefundedAt  time.Time `json:"refunded_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
