package repository

import (
	"go.uber.org/fx"
)

// Module exports dependency
var Module = fx.Options(
	fx.Provide(NewUserRepository),
	fx.Provide(NewOrderRepository),
	fx.Provide(NewAccountRepository),
	fx.Provide(NewCustomerRepository),
	fx.Provide(NewSessionRepository),
	fx.Provide(NewCartRepository),
)

// RepositoryWithTrx is a generic interface for repositories with transaction support
type RepositoryWithTrx[T any] interface {
	WithTrx(trxHandle interface{}) T
}
