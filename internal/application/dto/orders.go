package dto

import (
	"payloop/internal/domain/common"
)

// CreateOrderInput represents input for creating an order
type CreateOrderInput struct {
	Customer        CreateOrderCustomer `json:"customer" jsonschema:"required,description=Customer information for the order"`
	SessionId       string              `json:"session_id,omitempty" jsonschema:"description=Session ID if order is from a checkout session"`
	Currency        string              `json:"currency,omitempty" jsonschema:"description=Currency code (ISO 4217)"`
	CartItems       []CartItem          `json:"items,omitempty" jsonschema:"description=Items to include in the order"`
	PspId           common.Gateway      `json:"psp_id" jsonschema:"required,description=Payment service provider ID"`
	PaymentMethodId string              `json:"payment_method_id,omitempty" jsonschema:"description=Payment method ID to use for payment"`
	Metadata        map[string]string   `json:"metadata,omitempty" jsonschema:"description=Additional metadata as key-value pairs"`
	Options         map[string]string   `json:"options,omitempty" jsonschema:"description=Order processing options"`
}

// CreateOrderCustomer represents customer data for order creation
type CreateOrderCustomer struct {
	Id        string            `json:"id,omitempty" jsonschema:"description=Existing customer ID (if empty, new customer will be created)"`
	Email     string            `json:"email" jsonschema:"required,description=Customer email address"`
	FirstName string            `json:"first_name,omitempty" jsonschema:"description=Customer first name"`
	LastName  string            `json:"last_name,omitempty" jsonschema:"description=Customer last name"`
	Phone     string            `json:"phone,omitempty" jsonschema:"description=Customer phone number"`
	Metadata  map[string]string `json:"metadata,omitempty" jsonschema:"description=Customer metadata"`
}

// CartItem represents an item in the shopping cart
type CartItem struct {
	ProductId string `json:"product_id" jsonschema:"required,description=Product ID"`
	PriceId   string `json:"price_id" jsonschema:"required,description=Price ID for this product"`
	Quantity  int    `json:"quantity" jsonschema:"required,minimum=1,description=Quantity of the item"`
}

// CompleteOrderInput represents input for completing an order
type CompleteOrderInput struct {
	OrderId         string            `json:"order_id" jsonschema:"required,description=Order ID to complete"`
	PaymentMethodId string            `json:"payment_method_id,omitempty" jsonschema:"description=Payment method ID to use for payment"`
	Metadata        map[string]string `json:"metadata,omitempty" jsonschema:"description=Additional metadata"`
}

// OrderListFilters represents filters for listing orders
type OrderListFilters struct {
	Page       int    `json:"page,omitempty" jsonschema:"minimum=1,description=Page number for pagination (default: 1)"`
	Limit      int    `json:"limit,omitempty" jsonschema:"minimum=1,maximum=100,description=Number of items per page (default: 20, max: 100)"`
	Status     string `json:"status,omitempty" jsonschema:"enum=pending,enum=completed,enum=failed,enum=cancelled,description=Filter by order status"`
	CustomerId string `json:"customer_id,omitempty" jsonschema:"description=Filter by customer ID"`
}