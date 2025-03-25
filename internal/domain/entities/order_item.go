package entities

import "time"

type OrderItem struct {
	OrgId         string            `json:"org_id"`
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
	Subtotal      int64             `json:"subtotal"`
	Total         int64             `json:"total"`
	Metadata      map[string]string `json:"metadata"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}
