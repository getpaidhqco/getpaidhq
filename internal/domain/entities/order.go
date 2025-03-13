package entities

import "time"

type Order struct {
	OrgId      string            `json:"org_id"`
	Id         string            `json:"id"`
	CustomerId string            `json:"customer_id"`
	Customer   Customer          `json:"customer,omitempty"`
	Reference  string            `json:"reference"`
	Status     OrderStatus       `json:"status"`
	SessionId  string            `json:"session_id,omitempty"`
	CartId     string            `json:"cart_id,omitempty"`
	Currency   string            `json:"currency"`
	Total      int64             `json:"total"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

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
