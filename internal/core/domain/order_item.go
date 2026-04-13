package domain

import "time"

type OrderItem struct {
	OrgId         string            `gorm:"column:org_id;primaryKey" json:"org_id"`
	Id            string            `gorm:"column:id;primaryKey" json:"id"`
	OrderId       string            `gorm:"column:order_id" json:"order_id"`
	ProductId     string            `gorm:"column:product_id" json:"product_id"`
	VariantId     string            `gorm:"column:variant_id" json:"variant_id"`
	PriceId       string            `gorm:"column:price_id" json:"price_id"`
	Price         Price             `gorm:"foreignKey:PriceId,OrgId;references:Id,OrgId" json:"price"`
	Description   string            `gorm:"column:description" json:"description"`
	Quantity      int               `gorm:"column:quantity" json:"quantity"`
	TaxTotal      int64             `gorm:"column:tax_total" json:"tax_total"`
	DiscountTotal int64             `gorm:"column:discount_total" json:"discount_total"`
	Subtotal      int64             `gorm:"column:sub_total" json:"subtotal"`
	Total         int64             `gorm:"column:total" json:"total"`
	Metadata      map[string]string `gorm:"column:metadata;serializer:json" json:"metadata"`
	CreatedAt     time.Time         `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time         `gorm:"column:updated_at" json:"updated_at"`
}

func (OrderItem) TableName() string { return "order_items" }
