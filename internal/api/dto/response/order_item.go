package response

import (
	"payloop/internal/domain/entities"
	"time"
)

type OrderItem struct {
	Id          string            `json:"id"`
	OrderId     string            `json:"order_id"`
	PriceId     string            `json:"price_id"`
	Price       Price             `json:"price"`
	Description string            `json:"description"`
	Quantity    int               `json:"quantity"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

func NewOrderItemFromEntity(entity entities.OrderItem) OrderItem {
	return OrderItem{
		Id:          entity.Id,
		OrderId:     entity.OrderId,
		PriceId:     entity.PriceId,
		Price:       NewPriceFromEntity(entity.Price),
		Description: entity.Description,
		Quantity:    entity.Quantity,
		Metadata:    entity.Metadata,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}
}
