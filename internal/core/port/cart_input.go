package port

import "getpaidhq/internal/core/domain"

// AddItemInput is the command input for adding a product to a cart.
type AddItemInput struct {
	ProductId string
	PriceId   string
	Quantity  int
}

// AddProductCommand drives CartService.AddProduct. orgId / cartId are sourced
// from the authenticated context at the HTTP boundary.
type AddProductCommand struct {
	OrgId     string
	CartId    string
	ProductId string
	PriceId   string
	Quantity  int
}

// RemoveItemCommand drives CartService.RemoveItem.
type RemoveItemCommand struct {
	OrgId  string
	CartId string
	Id     string
}

// AdjustCommand drives CartService.Adjust.
type AdjustCommand struct {
	OrgId     string
	CartId    string
	ProductId string
	PriceId   string
	Quantity  int
}

// CreateCartInput drives CartService.Create.
type CreateCartInput struct {
	OrgId    string
	Cart     domain.Cart
	Metadata map[string]string
}
