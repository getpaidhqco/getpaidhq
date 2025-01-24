package carts

import (
	cart "github.com/mdwt/payloop-cart"
)

type CreateCartInput struct {
	AccountId string            `json:"account_id"`
	Cart      cart.Cart         `json:"carts"`
	Metadata  map[string]string `json:"metadata"`
}
