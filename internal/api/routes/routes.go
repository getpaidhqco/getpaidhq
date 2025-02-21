package routes

import "go.uber.org/fx"

// Module exports dependency to container
var Module = fx.Options(
	fx.Provide(NewOrderRoutes),
	fx.Provide(NewUserRoutes),
	fx.Provide(NewOrgsRoutes),
	fx.Provide(NewCartsRoutes),
	fx.Provide(NewWebhookRoutes),
	fx.Provide(NewSessionRoutes),
	fx.Provide(NewSubscriptionRoutes),
	fx.Provide(NewHealthRoutes),
	fx.Provide(NewWebhookSubscriptionRoutes),
	fx.Provide(NewProductRoutes),
	fx.Provide(NewCustomerRoutes),
	fx.Provide(NewRoutes),
)

// Routes contains multiple routes
type Routes []Route

// Route interface
type Route interface {
	Setup()
}

// NewRoutes sets up routes
func NewRoutes(
	userRoutes UserRoutes,
	orderRoutes OrderRoutes,
	tenants OrgsRoutes,
	sessionRoutes SessionRoutes,
	cartRoutes CartsRoutes,
	webhooks WebhookRoutes,
	subscriptions SubscriptionRoutes,
	health HealthRoutes,
	whsRoutes WebhookSubscriptionRoutes,
	productRoutes ProductRoutes,
	customerRoutes CustomerRoutes,
) Routes {
	return Routes{
		userRoutes,
		orderRoutes,
		tenants,
		sessionRoutes,
		cartRoutes,
		webhooks,
		subscriptions,
		health,
		whsRoutes,
		productRoutes,
		customerRoutes,
	}
}

// Setup all the route
func (r Routes) Setup() {
	for _, route := range r {
		route.Setup()
	}
}
