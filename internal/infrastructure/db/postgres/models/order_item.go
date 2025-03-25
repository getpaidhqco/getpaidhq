package models

import (
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
)

type OrderItem struct {
	OrgId         string            `json:"org_id"`
	Id            string            `json:"id"`
	OrderId       string            `json:"order_id"`
	ProductId     string            `json:"product_id"`
	VariantId     pgtype.Text       `json:"variant_id"`
	PriceId       string            `json:"price_id"`
	Price         Price             `json:"price"`
	Description   string            `json:"description"`
	Quantity      int               `json:"quantity"`
	TaxTotal      int64             `json:"tax_total"`
	DiscountTotal int64             `json:"discount_total"`
	Subtotal      int64             `json:"subtotal"`
	Total         int64             `json:"total"`
	Metadata      map[string]string `json:"metadata"`
	CreatedAt     pgtype.Date       `json:"created_at"`
	UpdatedAt     pgtype.Date       `json:"updated_at"`
}

func (oi *OrderItem) ToEntity() entities.OrderItem {
	return entities.OrderItem{
		OrgId:         oi.OrgId,
		Id:            oi.Id,
		OrderId:       oi.OrderId,
		ProductId:     oi.ProductId,
		VariantId:     oi.VariantId.String,
		PriceId:       oi.PriceId,
		Price:         oi.Price.ToEntity(),
		Description:   oi.Description,
		Quantity:      oi.Quantity,
		TaxTotal:      oi.TaxTotal,
		DiscountTotal: oi.DiscountTotal,
		Subtotal:      oi.Subtotal,
		Total:         oi.Total,
		Metadata:      oi.Metadata,
		CreatedAt:     oi.CreatedAt.Time,
		UpdatedAt:     oi.UpdatedAt.Time,
	}
}
