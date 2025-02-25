package carts

import "payloop/internal/infrastructure/cart"

type CreateCartInput struct {
	OrgId    string            `json:"org_id"`
	Cart     cart.Cart         `json:"carts"`
	Metadata map[string]string `json:"metadata"`
}

type AddProductCommand struct {
	OrgId     string `json:"org_id"`
	CartId    string `json:"cart_id"`
	ProductId string `json:"product_id"`
	PriceId   string `json:"price_id"`
	Quantity  int    `json:"quantity"`
}

type RemoveItemCommand struct {
	OrgId  string `json:"org_id"`
	CartId string `json:"cart_id"`
	Id     string `json:"id"`
}

type AdjustCommand struct {
	OrgId     string `json:"org_id"`
	CartId    string `json:"cart_id"`
	ProductId string `json:"product_id"`
	PriceId   string `json:"price_id"`
	Quantity  int    `json:"quantity"`
}
