package orders

type CreateOrderInput struct {
	AccountId string `json:"acct_id" binding:"required"`

	CustomerId string            `json:"customer_id" binding:"required"`
	Currency   string            `json:"currency" binding:"required"`
	Total      float64           `json:"total" binding:"required"`
	Metadata   map[string]string `json:"metadata"`
}
