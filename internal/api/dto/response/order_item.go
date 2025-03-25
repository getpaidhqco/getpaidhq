package response

import (
	"payloop/internal/domain/entities"
	"time"
)

type OrderItem struct {
	Id            string            `json:"id"`
	OrderId       string            `json:"order_id"`
	ProductId     string            `json:"product_id"`
	VariantId     string            `json:"variant_id"`
	PriceId       string            `json:"price_id"`
	Price         Price             `json:"price"`
	Description   string            `json:"description"`
	Quantity      int               `json:"quantity"`
	TaxTotal      int64             `json:"tax_total"`
	DiscountTotal int64             `json:"discount_total"`
	Subtotal      int64             `json:"sub_total"`
	Total         int64             `json:"total"`
	Metadata      map[string]string `json:"metadata"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

func NewOrderItemFromEntity(entity entities.OrderItem) OrderItem {
	return OrderItem{
		Id:            entity.Id,
		OrderId:       entity.OrderId,
		PriceId:       entity.PriceId,
		ProductId:     entity.ProductId,
		VariantId:     entity.VariantId,
		Price:         NewPriceFromEntity(entity.Price),
		Description:   entity.Description,
		Quantity:      entity.Quantity,
		TaxTotal:      entity.TaxTotal,
		DiscountTotal: entity.DiscountTotal,
		Subtotal:      entity.Subtotal,
		Total:         entity.Total,
		Metadata:      entity.Metadata,
		CreatedAt:     entity.CreatedAt,
		UpdatedAt:     entity.UpdatedAt,
	}
}
