package response

import (
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payments"
	"time"
)

type Payment struct {
	Id             string                 `json:"id"`
	PspId          string                 `json:"psp_id"`
	Reference      string                 `json:"reference"`
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

func NewPaymentFromEntity(entity entities.Payment) Payment {
	return Payment{
		Id:             entity.Id,
		PspId:          entity.PspId,
		Reference:      entity.Reference,
		OrderId:        entity.OrderId,
		SubscriptionId: entity.SubscriptionId,
		Status:         entity.Status,
		Currency:       entity.Currency,
		Amount:         entity.Amount,
		PspFee:         entity.PspFee,
		PlatformFee:    entity.PlatformFee,
		NetAmount:      entity.NetAmount,
		Metadata:       entity.Metadata,
		CreatedAt:      entity.CreatedAt,
		UpdatedAt:      entity.UpdatedAt,
	}
}
