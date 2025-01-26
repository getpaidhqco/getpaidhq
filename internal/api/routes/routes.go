package routes

import "go.uber.org/fx"

// Module exports dependency to container
var Module = fx.Options(
	fx.Provide(NewOrderRoutes),
	fx.Provide(NewUserRoutes),
	fx.Provide(NewAccountsRoutes),
	fx.Provide(NewCartsRoutes),
	fx.Provide(NewSessionRoutes),
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
	tenants AccountsRoutes,
	sessionRoutes SessionRoutes,
	cartRoutes CartsRoutes,
) Routes {
	return Routes{
		userRoutes,
		orderRoutes,
		tenants,
		sessionRoutes,
		cartRoutes,
	}
}

// Setup all the route
func (r Routes) Setup() {
	for _, route := range r {
		route.Setup()
	}
}
