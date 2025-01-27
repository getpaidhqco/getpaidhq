package entities

import cart "github.com/mdwt/payloop-cart"

type Cart struct {
	OrgId  string    `json:"org_id"`
	Id     string    `json:"id"`
	Data   cart.Cart `json:"data"`
	Status string    `json:"status"`
	Total  int       `json:"total"`
}

type CartStatus string

const (
	CartStatusPending   CartStatus = "pending"
	CartStatusCompleted CartStatus = "completed"
	CartStatusExpired   CartStatus = "expired"
)
