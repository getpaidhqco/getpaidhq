package port

type AuthzAction string

const (
	ActionCreateOrg Action = "CreateOrg"

	ActionCreateCart             Action = "create_cart"
	ActionCreateOrder            Action = "CreateOrder"
	ActionListOrderSubscriptions Action = "ListOrderSubscriptions"

	ActionCreateProduct Action = "CreateProduct"
	ActionListProducts  Action = "ListProducts"
	ActionGetProduct    Action = "GetProduct"
	ActionUpdateProduct Action = "UpdateProduct"
	ActionDeleteProduct Action = "DeleteProduct"

	ActionCreateVariant Action = "CreateVariant"
	ActionGetVariant    Action = "GetVariant"
	ActionListVariants  Action = "ListVariants"
	ActionUpdateVariant Action = "UpdateVariant"
	ActionDeleteVariant Action = "DeleteVariant"

	ActionCreatePrice Action = "CreatePrice"
	ActionGetPrice    Action = "GetPrice"
	ActionListPrices  Action = "ListPrices"
	ActionUpdatePrice Action = "UpdatePrice"
	ActionDeletePrice Action = "DeletePrice"

	ActionCreatePaymentServiceProvider Action = "CreatePaymentServiceProvider"
	ActionGetPaymentServiceProvider    Action = "GetPaymentServiceProvider"
	ActionUpdatePaymentServiceProvider Action = "UpdatePaymentServiceProvider"
	ActionDeletePaymentServiceProvider Action = "DeletePaymentServiceProvider"

	ActionCreateCustomer Action = "CreateCustomer"

	ActionCreatePaymentMethod Action = "CreatePaymentMethod"
	ActionUpdatePaymentMethod Action = "UpdatePaymentMethod"
	ActionGetPaymentMethod    Action = "GetPaymentMethod"

	ActionAddProductToCart   Action = "AddProductToCart"
	ActionRemoveItemFromCart Action = "RemoveItemFromCart"
	ActionProcessWebhook    Action = "ProcessWebhook"
	ActionCreateSession     Action = "CreateSession"
	ActionUpdateSubscription Action = "UpdateSubscription"
	ActionPauseSubscription  Action = "PauseSubscription"
	ActionResumeSubscription Action = "ResumeSubscription"
	ActionHealthcheck        Action = "Healthcheck"

	ActionCreateWebhookSubscription Action = "CreateWebhookSubscription"
	ActionListWebhookSubscriptions  Action = "ListWebhookSubscriptions"
)

type Action = AuthzAction

// Authz enforces authorization policies.
type Authz interface {
	Enforce(user AuthUser, action Action, resource string) bool
}
