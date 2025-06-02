package middlewares

import (
	"go.uber.org/fx"
	"payloop/internal/infrastructure/authz/cedar"
)

// Module Middleware exported
var Module = fx.Options(
	fx.Provide(NewSentryMiddleware),
	fx.Provide(fx.Annotate(
		NewAuthnWrapperMiddleware,
		fx.ParamTags(`group:"authenticators"`),
	)),
	fx.Provide(NewCorsMiddleware),
	fx.Provide(fx.Annotate(
		NewDatabaseTrx,
		fx.ParamTags(`name:"primaryDb"`),
	)),

	fx.Provide(NewMiddlewares),
)

// IMiddleware middleware interface
type IMiddleware interface {
	Setup()
}

// Middlewares contains multiple middleware
type Middlewares []IMiddleware

// NewMiddlewares creates new middlewares
// Register the middleware that should be applied directly (globally)
func NewMiddlewares(
	corsMiddleware CorsMiddleware,
	dbTrxMiddleware DatabaseTrx,
	authMiddleware AuthnWrapperMiddleware,
	authzMiddleware cedar.CedarMiddleware,
	sentryMiddleware SentryMiddleware,
) Middlewares {
	return Middlewares{
		corsMiddleware,
		dbTrxMiddleware,
		authMiddleware,
		authzMiddleware,
		sentryMiddleware,
	}
}

// Setup sets up middlewares
func (m Middlewares) Setup() {
	for _, middleware := range m {
		middleware.Setup()
	}
}
