package domain

import (
	"maps"
	"time"
)

// Order is the purchase aggregate. Customer and Items are populated by the
// repository when a Preload-equivalent is used; for code paths that don't
// hydrate them, only the IDs are reliable.
type Order struct {
	OrgId      string
	Id         string
	CustomerId string
	Customer   Customer
	Reference  string
	Status     OrderStatus
	SessionId  string
	CartId     string
	Items      []OrderItem
	Currency   string
	Total      int64
	Metadata   map[string]string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// SetMetadata merges the existing metadata with the specified values.
func (o *Order) SetMetadata(meta map[string]string) *Order {
	if o.Metadata == nil {
		o.Metadata = make(map[string]string)
	}
	maps.Copy(o.Metadata, meta)
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
