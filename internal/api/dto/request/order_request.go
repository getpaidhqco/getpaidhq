package request

type CreateOrderRequest struct {
	OrgId    string                     `json:"org_id" binding:"required"` // TODO should be resolved from the API authn
	Customer CreateOrderRequestCustomer `json:"customer" binding:"required"`
	CartId   string                     `json:"cart_id" binding:"required"`
	Metadata map[string]string          `json:"metadata"`
}

type CreateOrderRequestCustomer struct {
	ID       string            `json:"id"`
	Email    string            `json:"email"`
	Name     string            `json:"name"`
	Metadata map[string]string `json:"metadata"`
}
