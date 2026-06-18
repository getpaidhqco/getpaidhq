package postgresgorm

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// paymentRow is the postgres on-the-wire shape of a Payment.
type paymentRow struct {
	OrgId          string               `gorm:"column:org_id;primaryKey"`
	Id             string               `gorm:"column:id;primaryKey"`
	Psp            domain.Gateway       `gorm:"column:psp"`
	PspId          string               `gorm:"column:psp_id"`
	Reference      string               `gorm:"column:reference"`
	OrderId        string               `gorm:"column:order_id"`
	SubscriptionId string               `gorm:"column:subscription_id"`
	InvoiceId      string               `gorm:"column:invoice_id"`
	Status         domain.PaymentStatus `gorm:"column:status"`
	Recurring      bool                 `gorm:"column:recurring"`
	Currency       string               `gorm:"column:currency"`
	Amount         int64                `gorm:"column:amount"`
	PspFee         int64                `gorm:"column:psp_fee"`
	PlatformFee    int64                `gorm:"column:platform_fee"`
	NetAmount      int64                `gorm:"column:net_amount"`
	Metadata       map[string]string    `gorm:"column:metadata;serializer:json"`
	CompletedAt    time.Time            `gorm:"column:completed_at"`
	CreatedAt      time.Time            `gorm:"column:created_at"`
	UpdatedAt      time.Time            `gorm:"column:updated_at"`
}

func (paymentRow) TableName() string { return "payments" }

func (r paymentRow) toDomain() domain.Payment {
	return domain.Payment{
		OrgId:          r.OrgId,
		Id:             r.Id,
		Psp:            r.Psp,
		PspId:          r.PspId,
		Reference:      r.Reference,
		OrderId:        r.OrderId,
		SubscriptionId: r.SubscriptionId,
		InvoiceId:      r.InvoiceId,
		Status:         r.Status,
		Recurring:      r.Recurring,
		Currency:       r.Currency,
		Amount:         r.Amount,
		PspFee:         r.PspFee,
		PlatformFee:    r.PlatformFee,
		NetAmount:      r.NetAmount,
		Metadata:       r.Metadata,
		CompletedAt:    r.CompletedAt,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func paymentRowFromDomain(p domain.Payment) paymentRow {
	return paymentRow{
		OrgId:          p.OrgId,
		Id:             p.Id,
		Psp:            p.Psp,
		PspId:          p.PspId,
		Reference:      p.Reference,
		OrderId:        p.OrderId,
		SubscriptionId: p.SubscriptionId,
		InvoiceId:      p.InvoiceId,
		Status:         p.Status,
		Recurring:      p.Recurring,
		Currency:       p.Currency,
		Amount:         p.Amount,
		PspFee:         p.PspFee,
		PlatformFee:    p.PlatformFee,
		NetAmount:      p.NetAmount,
		Metadata:       p.Metadata,
		CompletedAt:    p.CompletedAt,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

func paymentRowsToDomain(rows []paymentRow) []domain.Payment {
	out := make([]domain.Payment, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out
}
