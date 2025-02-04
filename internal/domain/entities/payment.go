package entities

import (
	"payloop/internal/domain/entities/payments"
	"time"
)

type Payment struct {
	OrgId          string                 `json:"org_id"`
	Id             string                 `json:"id"`
	PspId          string                 `json:"psp_id"`
	OrderId        string                 `json:"order_id"`
	SubscriptionId string                 `json:"subscription_id"`
	Status         payments.PaymentStatus `json:"status"`
	Currency       string                 `json:"currency"`
	Amount         int                    `json:"amount"`
	PspFee         int                    `json:"psp_fee"`
	PlatformFee    int                    `json:"platform_fee"`
	NetAmount      int                    `json:"net_amount"`
	Metadata       map[string]string      `json:"metadata"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}
