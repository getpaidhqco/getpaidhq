package response

import (
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payments"
	"time"
)

type Payment struct {
	OrgId          string                 `json:"org_id"`
	Id             string                 `json:"id"`
	Psp            common.Gateway         `json:"psp"`
	PspId          string                 `json:"psp_id,omitempty"`
	Reference      string                 `json:"reference,omitempty"`
	OrderId        string                 `json:"order_id,omitempty"`
	SubscriptionId string                 `json:"subscription_id,omitempty"`
	Status         payments.PaymentStatus `json:"status"`
	Recurring      bool                   `json:"recurring"`
	Currency       string                 `json:"currency"`
	Amount         int64                  `json:"amount"`
	PspFee         int64                  `json:"psp_fee,omitempty"`
	PlatformFee    int64                  `json:"platform_fee,omitempty"`
	NetAmount      int64                  `json:"net_amount,omitempty"`
	Metadata       map[string]string      `json:"metadata,omitempty"`
	CompletedAt    time.Time              `json:"completed_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

func NewPaymentFromEntity(entity entities.Payment) Payment {
	return Payment{
		OrgId:          entity.OrgId,
		Id:             entity.Id,
		Psp:            entity.Psp,
		PspId:          entity.PspId,
		Reference:      entity.Reference,
		OrderId:        entity.OrderId,
		SubscriptionId: entity.SubscriptionId,
		Status:         entity.Status,
		Recurring:      entity.Recurring,
		Currency:       entity.Currency,
		Amount:         entity.Amount,
		PspFee:         entity.PspFee,
		PlatformFee:    entity.PlatformFee,
		NetAmount:      entity.NetAmount,
		Metadata:       entity.Metadata,
		CompletedAt:    entity.CompletedAt,
		CreatedAt:      entity.CreatedAt,
		UpdatedAt:      entity.UpdatedAt,
	}
}
