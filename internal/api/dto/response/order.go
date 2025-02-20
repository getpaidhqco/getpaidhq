package response

import (
	"payloop/internal/domain/entities"
	"time"
)

type Order struct {
	Id         string            `json:"id"`
	CustomerId string            `json:"customer_id"`
	Reference  string            `json:"reference"`
	Status     string            `json:"status"`
	SessionId  string            `json:"session_id"`
	CartId     string            `json:"cart_id"`
	Currency   string            `json:"currency"`
	Total      int64             `json:"total"`
	Metadata   map[string]string `json:"metadata"`
	CreatedAt  time.Time         `json:"created_at"`
}

func NewOrderFromEntity(entity entities.Order) Order {
	return Order{
		Id:         entity.Id,
		CustomerId: entity.CustomerId,
		Reference:  entity.Reference,
		Status:     string(entity.Status),
		SessionId:  entity.SessionId,
		CartId:     entity.CartId,
		Currency:   entity.Currency,
		Total:      entity.Total,
		Metadata:   entity.Metadata,
		CreatedAt:  entity.CreatedAt,
	}
}
