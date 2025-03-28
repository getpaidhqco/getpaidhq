package request

type CartInput struct {
	Currency string     `json:"currency"`
	Items    []CartItem `json:"items"`
}

type CartItem struct {
	ProductId string `json:"product_id" binding:"required"`
	PriceId   string `json:"price_id" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required"`
}

type CreateOrderRequest struct {
	Customer        CreateOrderRequestCustomer `json:"customer" binding:"required"`
	PaymentMethodId string                     `json:"payment_method_id"`
	SessionId       string                     `json:"session_id"`
	PspId           string                     `json:"psp_id" binding:"required"`

	// Cart is required if SessionId is not provided
	Cart     CartInput         `json:"cart"`
	Metadata map[string]string `json:"metadata"`
	Options  map[string]string `json:"options"`
}

type CreateOrderRequestCustomer struct {
	ID        string            `json:"id"`
	Email     string            `json:"email"`
	FirstName string            `json:"first_name"`
	LastName  string            `json:"last_name"`
	Phone     string            `json:"phone"`
	Metadata  map[string]string `json:"metadata"`
}

type CompleteOrderRequest struct {
	PaymentMethodId string                          `json:"payment_method_id"`
	PaymentMethod   CompleteOrderInputPaymentMethod `json:"payment_method"`
	Payment         CompleteOrderRequestPayment     `json:"payment"`
	Metadata        map[string]string               `json:"metadata"`
}
type CompleteOrderInputPaymentMethod struct {
	Psp            string            `json:"psp"`
	Name           string            `json:"name"`
	IsDefault      bool              `json:"is_default"`
	BillingAddress Address           `json:"billing_address"`
	Type           string            `json:"type"`
	Details        interface{}       `json:"details"`
	Token          string            `json:"token"`
	Metadata       map[string]string `json:"metadata"`
}

type CompleteOrderRequestPayment struct {
	PspId       string            `json:"psp_id"`
	Reference   string            `json:"reference"`
	Amount      int64             `json:"amount"`
	CompletedAt string            `json:"completed_at"`
	Metadata    map[string]string `json:"metadata"`
	Currency    string            `json:"currency"`
}
