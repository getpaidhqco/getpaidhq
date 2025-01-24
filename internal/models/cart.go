package models

import cart "github.com/mdwt/payloop-cart"

type Cart struct {
	AccountId string    `json:"account_id"`
	Id        string    `json:"id"`
	Data      cart.Cart `json:"data"`
	Status    string    `json:"status"`
	Total     int       `json:"total"`
}

type CartStatus string

const (
	CartStatusPending   CartStatus = "pending"
	CartStatusCompleted CartStatus = "completed"
	CartStatusExpired   CartStatus = "expired"
)
