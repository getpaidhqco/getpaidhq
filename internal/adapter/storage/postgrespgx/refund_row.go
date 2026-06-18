package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// refundRow is the postgres on-the-wire shape of a Refund. psp_refund_id and
// reason are nullable columns (NULL, never ""). The gorm row held them as plain
// strings; here they map through nilIfEmpty/strOrEmpty so an empty value lands
// as NULL on write and reads back as "".
type refundRow struct {
	OrgId       string
	Id          string
	PspRefundId *string
	PaymentId   string
	Amount      int64
	Currency    string
	Reason      *string
	RefundedAt  time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

const refundColumns = `org_id, id, psp_refund_id, payment_id, amount, currency, reason, refunded_at, created_at, updated_at`

func (r *refundRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.PspRefundId, &r.PaymentId, &r.Amount,
		&r.Currency, &r.Reason, &r.RefundedAt, &r.CreatedAt, &r.UpdatedAt)
}

func (r refundRow) toDomain() domain.Refund {
	return domain.Refund{
		OrgId:       r.OrgId,
		Id:          r.Id,
		PspRefundId: strOrEmpty(r.PspRefundId),
		PaymentId:   r.PaymentId,
		Amount:      r.Amount,
		Currency:    r.Currency,
		Reason:      strOrEmpty(r.Reason),
		RefundedAt:  r.RefundedAt,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func refundRowFromDomain(r domain.Refund) refundRow {
	return refundRow{
		OrgId:       r.OrgId,
		Id:          r.Id,
		PspRefundId: nilIfEmpty(r.PspRefundId),
		PaymentId:   r.PaymentId,
		Amount:      r.Amount,
		Currency:    r.Currency,
		Reason:      nilIfEmpty(r.Reason),
		RefundedAt:  r.RefundedAt,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}
