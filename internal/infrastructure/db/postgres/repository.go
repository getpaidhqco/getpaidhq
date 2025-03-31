package postgres

import (
	"go.uber.org/fx"
)

// Module exports dependency
var RespositoryModules = fx.Options(
	fx.Provide(NewUserRepository),
	fx.Provide(NewOrderRepository),
	fx.Provide(NewOrgRepository),
	fx.Provide(NewCustomerRepository),
	fx.Provide(NewSessionRepository),
	fx.Provide(NewCartRepository),
	fx.Provide(NewPriceRepository),
	fx.Provide(NewProductRepository),
	fx.Provide(NewSubscriptionRepository),
	fx.Provide(NewSettingRepository),
	fx.Provide(NewPaymentRepository),
	fx.Provide(NewOrderItemRepository),
	fx.Provide(NewIdempotencyKeyRepository),
	fx.Provide(NewVariantRepository),
	fx.Provide(NewWebhookSubscriptionRepository),
	fx.Provide(NewApiKeyRepository),
	fx.Provide(NewReportRepository),
	fx.Provide(NewPaymentMethodRepository),
	fx.Provide(NewPspRepository),
)

// RepositoryWithTrx is a generic interface for repositories with transaction support
type RepositoryWithTrx[T any] interface {
	WithTrx(trxHandle interface{}) T
}
