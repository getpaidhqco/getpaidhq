package models

import (
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payments"
)

type Payment struct {
	OrgId          string            `json:"org_id"`
	Id             string            `json:"id"`
	Psp            string            `json:"psp"`
	PspId          pgtype.Text       `json:"psp_id"`
	Reference      pgtype.Text       `json:"reference"`
	OrderId        string            `json:"order_id"`
	InvoiceId      pgtype.Text       `json:"invoice_id"`
	SubscriptionId pgtype.Text       `json:"subscription_id"`
	Recurring      bool              `json:"recurring"`
	Status         string            `json:"status"`
	Currency       string            `json:"currency"`
	Amount         int64             `json:"amount"`
	PspFee         int64             `json:"psp_fee"`
	PlatformFee    int64             `json:"platform_fee"`
	NetAmount      int64             `json:"net_amount"`
	Metadata       map[string]string `json:"metadata"`
	CompletedAt    pgtype.Date       `json:"completed_at"`
	CreatedAt      pgtype.Date       `json:"created_at"`
	UpdatedAt      pgtype.Date       `json:"updated_at"`
}

func (s *Payment) ToEntity() entities.Payment {
	return entities.Payment{
		OrgId:          s.OrgId,
		Id:             s.Id,
		Psp:            common.Gateway(s.Psp),
		PspId:          s.PspId.String,
		Reference:      s.Reference.String,
		OrderId:        s.OrderId,
		InvoiceId:      s.InvoiceId.String,
		SubscriptionId: s.SubscriptionId.String,
		Recurring:      s.Recurring,
		Status:         payments.PaymentStatus(s.Status),
		Currency:       s.Currency,
		Amount:         s.Amount,
		PspFee:         s.PspFee,
		PlatformFee:    s.PlatformFee,
		NetAmount:      s.NetAmount,
		Metadata:       s.Metadata,
		CompletedAt:    s.CompletedAt.Time,
		CreatedAt:      s.CreatedAt.Time,
		UpdatedAt:      s.UpdatedAt.Time,
	}
}
