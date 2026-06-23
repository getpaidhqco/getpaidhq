package port

import "getpaidhq/internal/core/domain"

// AddItemInput is the input for adding a product to a cart.
type AddItemInput struct {
	ProductId string
	PriceId   string
	Quantity  int
}

// AddProductInput drives CartService.AddProduct. orgId / cartId are sourced
// from the authenticated context at the HTTP boundary.
type AddProductInput struct {
	OrgId     string
	CartId    string
	ProductId string
	PriceId   string
	Quantity  int
}

// RemoveItemInput drives CartService.RemoveItem.
type RemoveItemInput struct {
	OrgId  string
	CartId string
	Id     string
}

// AdjustItemInput drives CartService.Adjust.
type AdjustItemInput struct {
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
