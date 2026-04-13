package domain

import "time"

type Refund struct {
	OrgId       string    `gorm:"column:org_id;primaryKey" json:"org_id"`
	Id          string    `gorm:"column:id;primaryKey" json:"id"`
	PspRefundId string    `gorm:"column:psp_refund_id" json:"psp_refund_id"`
	PaymentId   string    `gorm:"column:payment_id" json:"payment_id"`
	Amount      int64     `gorm:"column:amount" json:"amount"`
	Currency    string    `gorm:"column:currency" json:"currency"`
	Reason      string    `gorm:"column:reason" json:"reason,omitempty"`
	RefundedAt  time.Time `gorm:"column:refunded_at" json:"refunded_at"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Refund) TableName() string { return "refunds" }
