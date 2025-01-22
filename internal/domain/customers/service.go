package customers

type CreateCustomerInput struct {
	AccountId string            `json:"acct_id" binding:"required"`
	Email     string            `json:"email" binding:"required"`
	Name      string            `json:"name" binding:"required"`
	Metadata  map[string]string `json:"metadata"`
}
