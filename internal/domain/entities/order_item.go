package entities

import "time"

type OrderItem struct {
	OrgId       string            `json:"org_id"`
	Id          string            `json:"id"`
	OrderId     string            `json:"order_id"`
	PriceId     string            `json:"price_id"`
	Price       Price             `json:"price"`
	Description string            `json:"description"`
	Quantity    int               `json:"quantity"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}
