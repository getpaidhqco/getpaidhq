package domain

import "time"

// Refund is a record of a refund issued against a Payment (via the PSP).
type Refund struct {
	OrgId       string
	Id          string
	PspRefundId string
	PaymentId   string
	Amount      int64
	Currency    string
	Reason      string
	RefundedAt  time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
