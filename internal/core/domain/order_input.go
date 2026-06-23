package domain

// CartItem is an item in a shopping cart or order.
type CartItem struct {
	ProductId string `json:"product_id" validate:"required"`
	PriceId   string `json:"price_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"required"`
}
