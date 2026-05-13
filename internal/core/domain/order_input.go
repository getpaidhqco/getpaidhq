package domain

import "time"

type CartItem struct {
	ProductId string `json:"product_id" validate:"required"`
	PriceId   string `json:"price_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"required"`
}

type CreateOrderInput struct {
	OrgId           string                     `json:"org_id" validate:"required"`
	Customer        CreateOrderCommandCustomer `json:"customer" validate:"required"`
	SessionId       string                     `json:"session_id"`
	Currency        string                     `json:"currency"`
	CartItems       []CartItem                 `json:"items"`
	PspId           Gateway                    `json:"psp_id" validate:"required"`
	PaymentMethodId string                     `json:"payment_method_id"`
	Metadata        map[string]string          `json:"metadata"`
	Options         map[string]string          `json:"options"`
}

type CreateOrderResponse struct {
	Order Order               `json:"order"`
	Psp   InitPaymentResponse `json:"psp"`
}

type CompleteCheckoutSessionInput struct {
	OrgId          string                `json:"org_id" validate:"required"`
	OrderId        string                `json:"cart_id" validate:"required"`
	PaymentContext PaymentWebhookContext `json:"payment_context"`
	Metadata       map[string]string     `json:"metadata"`
}

type CreateOrderCommandCustomer struct {
	Id        string            `json:"id"`
	Email     string            `json:"email"`
	FirstName string            `json:"first_name"`
	LastName  string            `json:"last_name"`
	Phone     string            `json:"phone"`
	Metadata  map[string]string `json:"metadata"`
}

type CompleteOrderInput struct {
	OrgId           string                          `json:"org_id"`
	Id              string                          `json:"id"`
	PaymentMethodId string                          `json:"payment_method_id"`
	PaymentMethod   CompleteOrderInputPaymentMethod `json:"payment_method"`
	Payment         CompleteOrderInputPayment       `json:"payment"`
	Metadata        map[string]string               `json:"metadata"`
}

type CompleteOrderInputPayment struct {
	PspId       string            `json:"psp_id"`
	CompletedAt time.Time         `json:"completed_at"`
	Reference   string            `json:"reference"`
	Amount      int64             `json:"amount"`
	Currency    string            `json:"currency"`
	Metadata    map[string]string `json:"metadata"`
}

type CompleteOrderInputPaymentMethod struct {
	Psp            string            `json:"psp"`
	Name           string            `json:"name"`
	IsDefault      bool              `json:"is_default"`
	BillingAddress Address           `json:"billing_address"`
	Type           PaymentMethodType `json:"type" validate:"required"`
	Details        any               `json:"details"`
	Token          string            `json:"token"`
	Metadata       map[string]string `json:"metadata"`
}

type CreateOrderRow struct {
	OrgId     string                     `json:"org_id" validate:"required"`
	Customer  CreateOrderCommandCustomer `json:"customer" validate:"required"`
	SessionId string                     `json:"session_id" validate:"required"`
	Currency  string                     `json:"currency" validate:"required"`
	Metadata  map[string]string          `json:"metadata"`
}

type CartInput struct {
	Currency     string  `json:"currency" validate:"required"`
	Total        float64 `json:"total" validate:"required"`
	SubTotal     float64 `json:"sub_total" validate:"required"`
	Discount     float64 `json:"discount" validate:"required"`
	SetupFee     float64 `json:"setup_fee" validate:"required"`
	Tax          float64 `json:"tax" validate:"required"`
	TaxName      string  `json:"tax_name" validate:"required"`
	TaxRate      float64 `json:"tax_rate" validate:"required"`
	TaxInclusive bool    `json:"tax_inclusive" validate:"required"`
}
