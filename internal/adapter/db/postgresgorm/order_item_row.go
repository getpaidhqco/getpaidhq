package postgresgorm

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// orderItemRow is the postgres on-the-wire shape of an OrderItem. Price is
// NOT embedded here — composition is a service-layer concern; see
// service.OrderItemDetails and PriceRepository.FindByIds for the read-model
// path.
type orderItemRow struct {
	OrgId     string `gorm:"column:org_id;primaryKey"`
	Id        string `gorm:"column:id;primaryKey"`
	OrderId   string `gorm:"column:order_id"`
	ProductId string `gorm:"column:product_id"`
	VariantId string `gorm:"column:variant_id"`
	PriceId   string `gorm:"column:price_id"`
	// subscription_id is a nullable FK to subscriptions; NULL when absent
	// (never "") — items have no subscription until the order completes, and
	// writing "" would violate the FK.
	SubscriptionId *string           `gorm:"column:subscription_id"`
	Description    string            `gorm:"column:description"`
	Quantity       int               `gorm:"column:quantity"`
	TaxTotal       int64             `gorm:"column:tax_total"`
	DiscountTotal  int64             `gorm:"column:discount_total"`
	Subtotal       int64             `gorm:"column:sub_total"`
	Total          int64             `gorm:"column:total"`
	Metadata       map[string]string `gorm:"column:metadata;serializer:json"`
	CreatedAt      time.Time         `gorm:"column:created_at"`
	UpdatedAt      time.Time         `gorm:"column:updated_at"`
}

func (orderItemRow) TableName() string { return "order_items" }

func (r orderItemRow) toDomain() domain.OrderItem {
	return domain.OrderItem{
		OrgId:          r.OrgId,
		Id:             r.Id,
		OrderId:        r.OrderId,
		ProductId:      r.ProductId,
		VariantId:      r.VariantId,
		PriceId:        r.PriceId,
		SubscriptionId: strOrEmpty(r.SubscriptionId),
		Description:    r.Description,
		Quantity:       r.Quantity,
		TaxTotal:       r.TaxTotal,
		DiscountTotal:  r.DiscountTotal,
		Subtotal:       r.Subtotal,
		Total:          r.Total,
		Metadata:       r.Metadata,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func orderItemRowFromDomain(i domain.OrderItem) orderItemRow {
	return orderItemRow{
		OrgId:          i.OrgId,
		Id:             i.Id,
		OrderId:        i.OrderId,
		ProductId:      i.ProductId,
		VariantId:      i.VariantId,
		PriceId:        i.PriceId,
		SubscriptionId: nilIfEmpty(i.SubscriptionId),
		Description:    i.Description,
		Quantity:       i.Quantity,
		TaxTotal:       i.TaxTotal,
		DiscountTotal:  i.DiscountTotal,
		Subtotal:       i.Subtotal,
		Total:          i.Total,
		Metadata:       i.Metadata,
		CreatedAt:      i.CreatedAt,
		UpdatedAt:      i.UpdatedAt,
	}
}

func orderItemRowsToDomain(rows []orderItemRow) []domain.OrderItem {
	out := make([]domain.OrderItem, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out
}
