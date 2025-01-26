package carts

import (
	cart "github.com/mdwt/payloop-cart"
)

type CreateCartInput struct {
	AccountId string            `json:"account_id"`
	Cart      cart.Cart         `json:"carts"`
	Metadata  map[string]string `json:"metadata"`
}

type AddProductCommand struct {
	AccountId string `json:"account_id"`
	CartId    string `json:"cart_id"`
	ProductId string `json:"product_id"`
	PriceId   string `json:"price_id"`
	Quantity  int    `json:"quantity"`
}

type RemoveItemCommand struct {
	AccountId string `json:"account_id"`
	CartId    string `json:"cart_id"`
	Id        string `json:"id"`
}

type AdjustCommand struct {
	AccountId string `json:"account_id"`
	CartId    string `json:"cart_id"`
	ProductId string `json:"product_id"`
	PriceId   string `json:"price_id"`
	Quantity  int    `json:"quantity"`
}
