package request

type CartInput struct {
	Items []CartItem `json:"items"`
}

type CartItem struct {
	ProductId string `json:"product_id" binding:"required"`
	PriceId   string `json:"price_id" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required"`
}

type CreateOrderRequest struct {
	OrgId    string                     `json:"org_id" binding:"required"` // TODO should be resolved from the API authn
	Customer CreateOrderRequestCustomer `json:"customer" binding:"required"`
	CartId   string                     `json:"cart_id"`

	// Cart is required if CartId is not provided
	Cart     CartInput         `json:"cart"`
	Metadata map[string]string `json:"metadata"`
}

type CreateOrderRequestCustomer struct {
	ID       string            `json:"id"`
	Email    string            `json:"email"`
	Name     string            `json:"name"`
	Metadata map[string]string `json:"metadata"`
}
