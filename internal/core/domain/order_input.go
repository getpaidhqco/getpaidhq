package domain

// CartItem is an item in a shopping cart or order.
type CartItem struct {
	ProductId string `json:"product_id" validate:"required"`
	PriceId   string `json:"price_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"required"`
}

// CreateOrderResponse is the result of a successful order creation.
type CreateOrderResponse struct {
	Order Order               `json:"order"`
	Psp   InitPaymentResponse `json:"psp"`
}

// CreateOrderCommandCustomer holds the customer details provided when creating an order.
type CreateOrderCommandCustomer struct {
	Id        string            `json:"id"`
	Email     string            `json:"email"`
	FirstName string            `json:"first_name"`
	LastName  string            `json:"last_name"`
	Phone     string            `json:"phone"`
	Metadata  map[string]string `json:"metadata"`
}

// CreateOrderRow is the database row shape for order creation.
type CreateOrderRow struct {
	OrgId     string                     `json:"org_id" validate:"required"`
	Customer  CreateOrderCommandCustomer `json:"customer" validate:"required"`
	SessionId string                     `json:"session_id" validate:"required"`
	Currency  string                     `json:"currency" validate:"required"`
	Metadata  map[string]string          `json:"metadata"`
}

// CartInput holds cart pricing details.
type CartInput struct {
	Currency     string  `json:"currency" validate:"required"`
	Total        float64 `json:"total" validate:"required"`
	SubTotal     float64 `json:"sub_total" validate:"required"`
	Discount     float64 `json:"discount" validate:"required"`
	SetupFee     float64 `json:"setup_fee" validate:"required"`
	Tax          float64 `json:"tax" validate:"required"`
	TaxName      string  `json:"tax_name" validate:"required"`
	TaxRate      float64 `json:"tax_rate" validate:"required"`
	TaxInclusive bool    `json:"tax_inclusive" validate:"required"`
}
