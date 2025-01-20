package orders

import "context"

type OrderRepository interface {
	CreateOrder(ctx context.Context, input CreateOrderInput) error
}

type Customer struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	Country   string `json:"country"`
}

type CreateOrderInput struct {
	TID       string   `json:"tid"`
	Reference string   `json:"reference"`
	Currency  string   `json:"currency"`
	Total     int      `json:"total"`
	Customer  Customer `json:"customer"`
}
