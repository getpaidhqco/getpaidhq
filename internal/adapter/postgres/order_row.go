package postgres

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// orderRow is the postgres on-the-wire shape of an Order. Customer and Items
// are populated via Preload at the row level.
type orderRow struct {
	OrgId      string            `gorm:"column:org_id;primaryKey"`
	Id         string            `gorm:"column:id;primaryKey"`
	CustomerId string            `gorm:"column:customer_id"`
	Customer   customerRow       `gorm:"foreignKey:CustomerId,OrgId;references:Id,OrgId"`
	Reference  string            `gorm:"column:reference"`
	Status     domain.OrderStatus `gorm:"column:status"`
	SessionId  string            `gorm:"column:session_id"`
	CartId     string            `gorm:"column:cart_id"`
	Items      []orderItemRow    `gorm:"foreignKey:OrderId,OrgId;references:Id,OrgId"`
	Currency   string            `gorm:"column:currency"`
	Total      int64             `gorm:"column:total"`
	Metadata   map[string]string `gorm:"column:metadata;serializer:json"`
	CreatedAt  time.Time         `gorm:"column:created_at"`
	UpdatedAt  time.Time         `gorm:"column:updated_at"`
}

func (orderRow) TableName() string { return "orders" }

func (r orderRow) toDomain() domain.Order {
	return domain.Order{
		OrgId:      r.OrgId,
		Id:         r.Id,
		CustomerId: r.CustomerId,
		Customer:   r.Customer.toDomain(),
		Reference:  r.Reference,
		Status:     r.Status,
		SessionId:  r.SessionId,
		CartId:     r.CartId,
		Items:      orderItemRowsToDomain(r.Items),
		Currency:   r.Currency,
		Total:      r.Total,
		Metadata:   r.Metadata,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}

func orderRowFromDomain(o domain.Order) orderRow {
	items := make([]orderItemRow, len(o.Items))
	for i, it := range o.Items {
		items[i] = orderItemRowFromDomain(it)
	}
	return orderRow{
		OrgId:      o.OrgId,
		Id:         o.Id,
		CustomerId: o.CustomerId,
		Customer:   customerRowFromDomain(o.Customer),
		Reference:  o.Reference,
		Status:     o.Status,
		SessionId:  o.SessionId,
		CartId:     o.CartId,
		Items:      items,
		Currency:   o.Currency,
		Total:      o.Total,
		Metadata:   o.Metadata,
		CreatedAt:  o.CreatedAt,
		UpdatedAt:  o.UpdatedAt,
	}
}
