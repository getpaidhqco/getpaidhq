package models

type Order struct {
	ID         uint   `json:"id"`
	CustomerID uint   `json:"customer_id"`
	Status     string `json:"status"`
	Total      int    `json:"total"`
}
