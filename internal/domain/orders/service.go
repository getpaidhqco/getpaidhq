package orders

type CreateOrderRow struct {
	AccountId string            `json:"acct_id" binding:"required"` // TODO should be resolved from the API authn
	Customer  CustomerInput     `json:"customer" binding:"required"`
	SessionId string            `json:"session_id" binding:"required"`
	Currency  string            `json:"currency" binding:"required"`
	Metadata  map[string]string `json:"metadata"`
}
type CreateOrderInput struct {
	AccountId string            `json:"acct_id" binding:"required"` // TODO should be resolved from the API authn
	Customer  CustomerInput     `json:"customer" binding:"required"`
	SessionId string            `json:"session_id" binding:"required"`
	Metadata  map[string]string `json:"metadata"`
}

type CustomerInput struct {
	ID       string            `json:"id"`
	Email    string            `json:"email"`
	Name     string            `json:"name"`
	Metadata map[string]string `json:"metadata"`
}

type CartInput struct {
	Currency     string  `json:"currency" binding:"required"`
	Total        float64 `json:"total" binding:"required"`
	SubTotal     float64 `json:"sub_total" binding:"required"`
	Discount     float64 `json:"discount" binding:"required"`
	SetupFee     float64 `json:"setup_fee" binding:"required"`
	Tax          float64 `json:"tax" binding:"required"`
	TaxName      string  `json:"tax_name" binding:"required"`
	TaxRate      float64 `json:"tax_rate" binding:"required"`
	TaxInclusive bool    `json:"tax_inclusive" binding:"required"`
}
