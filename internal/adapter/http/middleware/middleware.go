package middleware

import (
	"getpaidhq/internal/adapter/cedar"
)

// IMiddleware is the interface that all middlewares must implement.
type IMiddleware interface {
	Setup()
}

// Middlewares contains multiple middleware instances.
type Middlewares []IMiddleware

// NewMiddlewares creates the ordered list of middlewares that are applied globally.
func NewMiddlewares(
	corsMiddleware CorsMiddleware,
	dbTrxMiddleware DatabaseTrx,
	authMiddleware AuthnWrapperMiddleware,
	authzMiddleware cedar.CedarMiddleware,
) Middlewares {
	return Middlewares{
		corsMiddleware,
		dbTrxMiddleware,
		authMiddleware,
		authzMiddleware,
	}
}

// Setup sets up all middlewares in order.
func (m Middlewares) Setup() {
	for _, mw := range m {
		mw.Setup()
	}
}
