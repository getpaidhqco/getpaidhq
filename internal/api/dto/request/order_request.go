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
	Customer CreateOrderRequestCustomer `json:"customer" binding:"required"`
	CartId   string                     `json:"cart_id"`
	PspId    string                     `json:"psp_id"`

	// Cart is required if CartId is not provided
	Cart     CartInput         `json:"cart"`
	Metadata map[string]string `json:"metadata"`
	Options  map[string]string `json:"options"`
}

type CreateOrderRequestCustomer struct {
	ID        string            `json:"id"`
	Email     string            `json:"email"`
	FirstName string            `json:"first_name"`
	LastName  string            `json:"last_name"`
	Phone     string            `json:"phone"`
	Metadata  map[string]string `json:"metadata"`
}
