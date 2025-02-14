package authz

import "payloop/internal/api/authn"

type Action string

const (
	CreateOrg Action = "CreateOrg"

	CreateCart  Action = "create_cart"
	CreateOrder Action = "CreateOrder"

	CreateProduct Action = "CreateProduct"
	ListProducts  Action = "ListProducts"

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
)

type Authz interface {
	Enforce(user authn.User, action Action, resource string) bool
}
