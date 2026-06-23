package port

import (
	"getpaidhq/internal/core/domain"
	"time"
)

// CreateOrderInput is the input for OrderService.Create.
type CreateOrderInput struct {
	OrgId           string
	Customer        CreateOrderInputCustomer
	SessionId       string
	Currency        string
	CartItems       []domain.CartItem
	PspId           domain.Gateway
	PaymentMethodId string
	CouponCode      string
	Metadata        map[string]string
	Options         map[string]string
}

// CreateOrderInputCustomer holds customer details provided when creating an order.
type CreateOrderInputCustomer struct {
	Id        string
	Email     string
	FirstName string
	LastName  string
	Phone     string
	Metadata  map[string]string
}

// CreateOrderResult is the result of a successful OrderService.CreateOrder call.
type CreateOrderResult struct {
	Order domain.Order
	Psp   InitPaymentResponse
}

// CompleteOrderInput is the input for OrderService.Complete.
type CompleteOrderInput struct {
	OrgId           string
	Id              string
	PaymentMethodId string
	PaymentMethod   CompleteOrderInputPaymentMethod
	Payment         CompleteOrderInputPayment
	Metadata        map[string]string
}

// CompleteOrderInputPayment holds payment details for order completion.
type CompleteOrderInputPayment struct {
	PspId       string
	CompletedAt time.Time
	Reference   string
	Amount      int64
	Currency    string
	Metadata    map[string]string
}

// CompleteOrderInputPaymentMethod holds payment method details for order completion.
type CompleteOrderInputPaymentMethod struct {
	Psp            string
	Name           string
	IsDefault      bool
	BillingAddress domain.Address
	Type           domain.PaymentMethodType
	Details        any
	Token          string
	Metadata       map[string]string
}
