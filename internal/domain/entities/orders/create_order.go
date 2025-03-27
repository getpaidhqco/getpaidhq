package orders

import (
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/payment_providers"
	"time"
)

type CartItem struct {
	ProductId string `json:"product_id" binding:"required"`
	PriceId   string `json:"price_id" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required"`
}

type CreateOrderInput struct {
	OrgId           string                     `json:"org_id" binding:"required"`
	Customer        CreateOrderCommandCustomer `json:"customer" binding:"required"`
	SessionId       string                     `json:"session_id"`
	Currency        string                     `json:"currency"`
	CartItems       []CartItem                 `json:"items"`
	PspId           common.Gateway             `json:"psp_id" binding:"required"`
	PaymentMethodId string                     `json:"payment_method_id"`
	Metadata        map[string]string          `json:"metadata"`
	Options         map[string]string          `json:"options"`
}

type CreateOrderResponse struct {
	Order entities.Order                        `json:"order"`
	Psp   payment_providers.InitPaymentResponse `json:"psp"`
}

type CompleteCheckoutSessionInput struct {
	OrgId          string                                  `json:"org_id" binding:"required"` // TODO should be resolved from the API authn
	OrderId        string                                  `json:"cart_id" binding:"required"`
	PaymentContext payment_providers.PaymentWebhookContext `json:"payment_context"`
	Metadata       map[string]string                       `json:"metadata"`
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
	Type           string            `json:"type"`
	Token          string            `json:"token"`
	ExpireAt       string            `json:"expire_at"`
	Metadata       map[string]string `json:"metadata"`
}
type Address struct {
	FirstName  string         `json:"first_name"`
	LastName   string         `json:"last_name"`
	Email      string         `json:"email"`
	Phone      string         `json:"phone"`
	Line1      string         `json:"line1"`
	Line2      string         `json:"line2"`
	City       string         `json:"city"`
	State      string         `json:"state"`
	PostalCode string         `json:"postal_code"`
	Country    common.Country `json:"country"`
}
