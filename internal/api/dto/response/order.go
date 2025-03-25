package response

import (
	"payloop/internal/domain/entities"
	"time"
)

type Order struct {
	Id         string            `json:"id"`
	CustomerId string            `json:"customer_id"`
	Customer   Customer          `json:"customer"`
	Reference  string            `json:"reference"`
	Status     string            `json:"status"`
	SessionId  string            `json:"session_id"`
	CartId     string            `json:"cart_id"`
	Items      []OrderItem       `json:"items"`
	Currency   string            `json:"currency"`
	Total      int64             `json:"total"`
	Metadata   map[string]string `json:"metadata"`
	CreatedAt  time.Time         `json:"created_at"`
}

func NewOrderFromEntity(entity entities.Order) Order {
	var items []OrderItem
	for _, item := range entity.Items {
		items = append(items, NewOrderItemFromEntity(item))
	}

	return Order{
		Id:         entity.Id,
		CustomerId: entity.CustomerId,
		Customer:   NewCustomerFromEntity(entity.Customer),
		Reference:  entity.Reference,
		Items:      items,
		Status:     string(entity.Status),
		SessionId:  entity.SessionId,
		CartId:     entity.CartId,
		Currency:   entity.Currency,
		Total:      entity.Total,
		Metadata:   entity.Metadata,
		CreatedAt:  entity.CreatedAt,
	}
}
