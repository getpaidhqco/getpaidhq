package authz

import "payloop/internal/api/authn"

type Action string

const (
	CreateOrg Action = "CreateOrg"

	CreateCart             Action = "create_cart"
	CreateOrder            Action = "CreateOrder"
	ListOrderSubscriptions Action = "ListOrderSubscriptions"

	CreateProduct Action = "CreateProduct"
	ListProducts  Action = "ListProducts"
	GetProduct    Action = "GetProduct"
	UpdateProduct Action = "UpdateProduct"
	DeleteProduct Action = "DeleteProduct"

	CreateVariant Action = "CreateVariant"
	GetVariant    Action = "GetVariant"
	ListVariants  Action = "ListVariants"
	UpdateVariant Action = "UpdateVariant"
	DeleteVariant Action = "DeleteVariant"

	CreatePrice Action = "CreatePrice"
	GetPrice    Action = "GetPrice"
	ListPrices  Action = "ListPrices"
	UpdatePrice Action = "UpdatePrice"
	DeletePrice Action = "DeletePrice"

	CreatePaymentServiceProvider Action = "CreatePaymentServiceProvider"
	GetPaymentServiceProvider    Action = "GetPaymentServiceProvider"
	UpdatePaymentServiceProvider Action = "UpdatePaymentServiceProvider"
	DeletePaymentServiceProvider Action = "DeletePaymentServiceProvider"

	CreateCustomer Action = "CreateCustomer"

	CreatePaymentMethod Action = "CreatePaymentMethod"
	UpdatePaymentMethod Action = "UpdatePaymentMethod"
	GetPaymentMethod    Action = "GetPaymentMethod"

	AddProductToCart   Action = "AddProductToCart"
	RemoveItemFromCart Action = "RemoveItemFromCart"
	ProcessWebhook     Action = "ProcessWebhook"
	CreateSession      Action = "CreateSession"
	UpdateSubscription Action = "UpdateSubscription"
	PauseSubscription  Action = "PauseSubscription"
	ResumeSubscription Action = "ResumeSubscription"
	Healthcheck        Action = "Healthcheck"

	// webhook subscriptions
	CreateWebhookSubscription Action = "CreateWebhookSubscription"
	ListWebhookSubscriptions  Action = "ListWebhookSubscriptions"

	// Invoice actions
	CreateInvoice   Action = "CreateInvoice"
	UpdateInvoice   Action = "UpdateInvoice"
	DownloadInvoice Action = "DownloadInvoice"
	GetInvoice      Action = "GetInvoice"
	ListInvoices    Action = "ListInvoices"

	// Payment actions
	RefundPayment   Action = "RefundPayment"
)

type Authz interface {
	Enforce(user authn.User, action Action, resource string) bool
}
