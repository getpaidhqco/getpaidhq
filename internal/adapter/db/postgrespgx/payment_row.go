package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// paymentRow is the postgres on-the-wire shape of a Payment. subscription_id and
// invoice_id are nullable FK columns (NULL, never "") so an empty value never
// violates the constraint. psp_id and completed_at mirror the gorm row exactly:
// the gorm adapter held psp_id as a plain string and completed_at as a plain
// time.Time (no nulltime serializer), so we write the zero value rather than NULL
// to keep observable behaviour identical.
type paymentRow struct {
	OrgId          string
	Id             string
	Psp            string
	PspId          string
	Reference      string
	OrderId        string
	SubscriptionId *string
	InvoiceId      *string
	Status         string
	Recurring      bool
	Currency       string
	Amount         int64
	PspFee         int64
	PlatformFee    int64
	NetAmount      int64
	Metadata       jsonCol[map[string]string]
	CompletedAt    time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

const paymentColumns = `org_id, id, psp, psp_id, reference, order_id, subscription_id, invoice_id, status, recurring, currency, amount, psp_fee, platform_fee, net_amount, metadata, completed_at, created_at, updated_at`

func (r *paymentRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.Psp, &r.PspId, &r.Reference, &r.OrderId,
		&r.SubscriptionId, &r.InvoiceId, &r.Status, &r.Recurring, &r.Currency,
		&r.Amount, &r.PspFee, &r.PlatformFee, &r.NetAmount, &r.Metadata,
		&r.CompletedAt, &r.CreatedAt, &r.UpdatedAt)
}

func (r paymentRow) toDomain() domain.Payment {
	return domain.Payment{
		OrgId:          r.OrgId,
		Id:             r.Id,
		Psp:            domain.Gateway(r.Psp),
		PspId:          r.PspId,
		Reference:      r.Reference,
		OrderId:        r.OrderId,
		SubscriptionId: strOrEmpty(r.SubscriptionId),
		InvoiceId:      strOrEmpty(r.InvoiceId),
		Status:         domain.PaymentStatus(r.Status),
		Recurring:      r.Recurring,
		Currency:       r.Currency,
		Amount:         r.Amount,
		PspFee:         r.PspFee,
		PlatformFee:    r.PlatformFee,
		NetAmount:      r.NetAmount,
		Metadata:       r.Metadata.V,
		CompletedAt:    r.CompletedAt,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func paymentRowFromDomain(p domain.Payment) paymentRow {
	return paymentRow{
		OrgId:          p.OrgId,
		Id:             p.Id,
		Psp:            string(p.Psp),
		PspId:          p.PspId,
		Reference:      p.Reference,
		OrderId:        p.OrderId,
		SubscriptionId: nilIfEmpty(p.SubscriptionId),
		InvoiceId:      nilIfEmpty(p.InvoiceId),
		Status:         string(p.Status),
		Recurring:      p.Recurring,
		Currency:       p.Currency,
		Amount:         p.Amount,
		PspFee:         p.PspFee,
		PlatformFee:    p.PlatformFee,
		NetAmount:      p.NetAmount,
		Metadata:       newJSON(p.Metadata),
		CompletedAt:    p.CompletedAt,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}
