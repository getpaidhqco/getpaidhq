package postgresgorm

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// refundRow is the postgres on-the-wire shape of a Refund. Package-internal.
type refundRow struct {
	OrgId       string    `gorm:"column:org_id;primaryKey"`
	Id          string    `gorm:"column:id;primaryKey"`
	PspRefundId string    `gorm:"column:psp_refund_id"`
	PaymentId   string    `gorm:"column:payment_id"`
	Amount      int64     `gorm:"column:amount"`
	Currency    string    `gorm:"column:currency"`
	Reason      string    `gorm:"column:reason"`
	RefundedAt  time.Time `gorm:"column:refunded_at"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (refundRow) TableName() string { return "refunds" }

func (r refundRow) toDomain() domain.Refund {
	return domain.Refund{
		OrgId:       r.OrgId,
		Id:          r.Id,
		PspRefundId: r.PspRefundId,
		PaymentId:   r.PaymentId,
		Amount:      r.Amount,
		Currency:    r.Currency,
		Reason:      r.Reason,
		RefundedAt:  r.RefundedAt,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func refundRowFromDomain(r domain.Refund) refundRow {
	return refundRow{
		OrgId:       r.OrgId,
		Id:          r.Id,
		PspRefundId: r.PspRefundId,
		PaymentId:   r.PaymentId,
		Amount:      r.Amount,
		Currency:    r.Currency,
		Reason:      r.Reason,
		RefundedAt:  r.RefundedAt,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}
