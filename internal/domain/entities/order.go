package entities

import "time"

type Order struct {
	AccountId  string            `json:"account_id"`
	Id         string            `json:"id"`
	CustomerId string            `json:"customer_id"`
	Status     OrderStatus       `json:"status"`
	SessionId  string            `json:"session_id"`
	CartId     string            `json:"cart_id"`
	Currency   string            `json:"currency"`
	Total      int               `json:"total"`
	Metadata   map[string]string `json:"metadata"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusCompleted OrderStatus = "completed"
	OrderStatusExpired   OrderStatus = "expired"
	OrderStatusCancelled OrderStatus = "cancelled"
)
