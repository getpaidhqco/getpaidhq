package entities

import (
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities/payments"
	"time"
)

type Payment struct {
	OrgId          string                 `json:"org_id"`
	Id             string                 `json:"id"`
	Psp            common.Gateway         `json:"psp"`
	PspId          string                 `json:"psp_id"`
	Reference      string                 `json:"reference"`
	OrderId        string                 `json:"order_id"`
	InvoiceId      string                 `json:"invoice_id"`
	SubscriptionId string                 `json:"subscription_id"`
	Status         payments.PaymentStatus `json:"status"`
	Recurring      bool                   `json:"recurring"`
	Currency       string                 `json:"currency"`
	Amount         int64                  `json:"amount"`
	PspFee         int64                  `json:"psp_fee"`
	PlatformFee    int64                  `json:"platform_fee"`
	NetAmount      int64                  `json:"net_amount"`
	Metadata       map[string]string      `json:"metadata"`
	CompletedAt    time.Time              `json:"completed_at"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// SetMetadata merges the existing metadata with the specified values.
func (p *Payment) SetMetadata(meta map[string]string) *Payment {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}
	for key, value := range meta {
		p.Metadata[key] = value
	}
	return p
}
