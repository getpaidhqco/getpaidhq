package postgresgorm

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// orderRow is the postgres on-the-wire shape of an Order. Customer and Items
// are NOT embedded here — composition is a service-layer concern; see
// service.OrderDetails and the *Repository.FindByIds batch primitives.
type orderRow struct {
	OrgId          string             `gorm:"column:org_id;primaryKey"`
	Id             string             `gorm:"column:id;primaryKey"`
	CustomerId     string             `gorm:"column:customer_id"`
	Reference      string             `gorm:"column:reference"`
	Status         domain.OrderStatus `gorm:"column:status"`
	SessionId      string             `gorm:"column:session_id"`
	CartId         string             `gorm:"column:cart_id"`
	Currency       string             `gorm:"column:currency"`
	Total          int64              `gorm:"column:total"`
	Metadata       map[string]string  `gorm:"column:metadata;serializer:json"`
	PaymentSession any                `gorm:"column:payment_session;serializer:json"`
	CreatedAt      time.Time          `gorm:"column:created_at"`
	UpdatedAt      time.Time          `gorm:"column:updated_at"`
}

func (orderRow) TableName() string { return "orders" }

func (r orderRow) toDomain() domain.Order {
	return domain.Order{
		OrgId:          r.OrgId,
		Id:             r.Id,
		CustomerId:     r.CustomerId,
		Reference:      r.Reference,
		Status:         r.Status,
		SessionId:      r.SessionId,
		CartId:         r.CartId,
		Currency:       r.Currency,
		Total:          r.Total,
		Metadata:       r.Metadata,
		PaymentSession: r.PaymentSession,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func orderRowFromDomain(o domain.Order) orderRow {
	return orderRow{
		OrgId:          o.OrgId,
		Id:             o.Id,
		CustomerId:     o.CustomerId,
		Reference:      o.Reference,
		Status:         o.Status,
		SessionId:      o.SessionId,
		CartId:         o.CartId,
		Currency:       o.Currency,
		Total:          o.Total,
		Metadata:       o.Metadata,
		PaymentSession: o.PaymentSession,
		CreatedAt:      o.CreatedAt,
		UpdatedAt:      o.UpdatedAt,
	}
}

func orderRowsToDomain(rows []orderRow) []domain.Order {
	out := make([]domain.Order, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out
}
