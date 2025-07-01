package postgres

import (
	"go.uber.org/fx"
)

// Module exports dependency
var RespositoryModules = fx.Options(
	fx.Provide(fx.Annotate(
		NewDunningRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewDocSequenceRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewUserRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewOrderRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewOrgRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewCustomerRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewSessionRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewCartRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewPriceRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewProductRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewSubscriptionItemRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewSubscriptionRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewSettingRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewPaymentRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewOrderItemRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewIdempotencyKeyRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewVariantRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewWebhookSubscriptionRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewReportRepository,
		fx.ParamTags(`name:"reportingDb"`, `name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewGatewayRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewPaymentMethodRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewApiKeyRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewCohortRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewMetadataStoreRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewInvoiceRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(fx.Annotate(
		NewDocumentRepository,
		fx.ParamTags(`name:"primaryDb"`),
	)),
)

// RepositoryWithTrx is a generic interface for repositories with transaction support
type RepositoryWithTrx[T any] interface {
	WithTrx(trxHandle interface{}) T
}
