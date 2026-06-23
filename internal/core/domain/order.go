package domain

import (
	"maps"
	"time"
)

// Order is the purchase aggregate. Cross-aggregate references are by ID only;
// Customer and Items are loaded via service.OrderDetails composition.
type Order struct {
	OrgId      string
	Id         string
	CustomerId string
	Reference  string
	Status     OrderStatus
	SessionId  string
	CartId     string
	Currency   string
	Total      int64
	Metadata   map[string]string
	// PaymentSession is the PSP session payload (arbitrary shape); nil when no
	// session has been initialized yet. Stored as a nullable JSONB column.
	PaymentSession any
	CreatedAt      time.Time
	UpdatedAt      time.Time
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
