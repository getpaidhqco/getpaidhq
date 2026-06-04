package postgres

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// orderItemRow is the postgres on-the-wire shape of an OrderItem. The
// embedded priceRow is preloaded via the gorm foreignKey relationship below;
// callers that don't need the price can ignore it (gorm omits the join if
// Preload isn't called).
type orderItemRow struct {
	OrgId         string            `gorm:"column:org_id;primaryKey"`
	Id            string            `gorm:"column:id;primaryKey"`
	OrderId       string            `gorm:"column:order_id"`
	ProductId     string            `gorm:"column:product_id"`
	VariantId     string            `gorm:"column:variant_id"`
	PriceId       string            `gorm:"column:price_id"`
	Price         priceRow          `gorm:"foreignKey:PriceId,OrgId;references:Id,OrgId"`
	Description   string            `gorm:"column:description"`
	Quantity      int               `gorm:"column:quantity"`
	TaxTotal      int64             `gorm:"column:tax_total"`
	DiscountTotal int64             `gorm:"column:discount_total"`
	Subtotal      int64             `gorm:"column:sub_total"`
	Total         int64             `gorm:"column:total"`
	Metadata      map[string]string `gorm:"column:metadata;serializer:json"`
	CreatedAt     time.Time         `gorm:"column:created_at"`
	UpdatedAt     time.Time         `gorm:"column:updated_at"`
}

func (orderItemRow) TableName() string { return "order_items" }

func (r orderItemRow) toDomain() domain.OrderItem {
	return domain.OrderItem{
		OrgId:         r.OrgId,
		Id:            r.Id,
		OrderId:       r.OrderId,
		ProductId:     r.ProductId,
		VariantId:     r.VariantId,
		PriceId:       r.PriceId,
		Price:         r.Price.toDomain(),
		Description:   r.Description,
		Quantity:      r.Quantity,
		TaxTotal:      r.TaxTotal,
		DiscountTotal: r.DiscountTotal,
		Subtotal:      r.Subtotal,
		Total:         r.Total,
		Metadata:      r.Metadata,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

func orderItemRowFromDomain(i domain.OrderItem) orderItemRow {
	return orderItemRow{
		OrgId:         i.OrgId,
		Id:            i.Id,
		OrderId:       i.OrderId,
		ProductId:     i.ProductId,
		VariantId:     i.VariantId,
		PriceId:       i.PriceId,
		Price:         priceRowFromDomain(i.Price),
		Description:   i.Description,
		Quantity:      i.Quantity,
		TaxTotal:      i.TaxTotal,
		DiscountTotal: i.DiscountTotal,
		Subtotal:      i.Subtotal,
		Total:         i.Total,
		Metadata:      i.Metadata,
		CreatedAt:     i.CreatedAt,
		UpdatedAt:     i.UpdatedAt,
	}
}

func orderItemRowsToDomain(rows []orderItemRow) []domain.OrderItem {
	out := make([]domain.OrderItem, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out
}
