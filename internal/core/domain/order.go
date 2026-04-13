package domain

import "time"

type Order struct {
	OrgId      string            `gorm:"column:org_id;primaryKey" json:"org_id"`
	Id         string            `gorm:"column:id;primaryKey" json:"id"`
	CustomerId string            `gorm:"column:customer_id" json:"customer_id"`
	Customer   Customer          `gorm:"foreignKey:CustomerId,OrgId;references:Id,OrgId" json:"customer,omitempty"`
	Reference  string            `gorm:"column:reference" json:"reference"`
	Status     OrderStatus       `gorm:"column:status" json:"status"`
	SessionId  string            `gorm:"column:session_id" json:"session_id,omitempty"`
	CartId     string            `gorm:"column:cart_id" json:"cart_id,omitempty"`
	Items      []OrderItem       `gorm:"foreignKey:OrderId,OrgId;references:Id,OrgId" json:"items,omitempty"`
	Currency   Currency          `gorm:"column:currency" json:"currency"`
	Total      int64             `gorm:"column:total" json:"total"`
	Metadata   map[string]string `gorm:"column:metadata;serializer:json" json:"metadata,omitempty"`
	CreatedAt  time.Time         `gorm:"column:created_at" json:"created_at"`
	UpdatedAt  time.Time         `gorm:"column:updated_at" json:"updated_at"`
}

func (Order) TableName() string { return "orders" }

// SetMetadata merges the existing metadata with the specified values.
func (o *Order) SetMetadata(meta map[string]string) *Order {
	if o.Metadata == nil {
		o.Metadata = make(map[string]string)
	}
	for key, value := range meta {
		o.Metadata[key] = value
	}
	return o
}

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusCompleted OrderStatus = "completed"
	OrderStatusExpired   OrderStatus = "expired"
	OrderStatusCancelled OrderStatus = "cancelled"
)

type OrderCompletedEvent struct {
	OrgId   string `json:"org_id"`
	OrderId string `json:"order_id"`
}
