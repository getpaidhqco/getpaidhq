package services

import (
	"go.uber.org/fx"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/repositories"
	"payloop/internal/domain/security"
)

// Module exports services present
var Module = fx.Options(
	fx.Provide(fx.Annotate(
		NewTransactionService,
		fx.ParamTags(`name:"primaryDb"`),
	)),
	fx.Provide(NewUserService),
	fx.Provide(NewOrderService),
	fx.Provide(NewInvoiceService),
	fx.Provide(NewOrgService),
	fx.Provide(NewSessionService),
	fx.Provide(NewCartService),
	fx.Provide(NewWebhookService),
	fx.Provide(NewSubscriptionOrchestrationService),
	fx.Provide(NewSubscriptionService),
	fx.Provide(NewWebhookSubscriptionService),
	fx.Provide(NewWorkflowService),
	fx.Provide(NewProductService),
	fx.Provide(NewCustomerService),
	fx.Provide(NewOrderWorkflowService),
	fx.Provide(NewQueueService),
	fx.Provide(NewReportService),
	fx.Provide(NewPspService),
	fx.Provide(NewMetadataService),
	fx.Provide(NewPaymentService),
	// Add the settings registry provider
	fx.Provide(
		func(vault security.TokenVault) SettingsRegistryInterface {
			return NewSettingsRegistry(vault)
		},
	),

	// Update the SettingsService provider to include the registry
	fx.Provide(
		func(repo repositories.SettingRepository, registry SettingsRegistryInterface, logger logger.Logger) interfaces.SettingsService {
			return NewSettingsService(repo, registry, logger)
		},
	),
	fx.Provide(NewDocumentService),
	fx.Provide(NewDunningService),
	fx.Provide(NewDunningOrchestrationService),
	fx.Provide(NewUsageRecordingService),
	fx.Provide(NewBillingService),
	fx.Provide(NewTierCalculationService),
	fx.Provide(NewMeterService),
	fx.Provide(NewPaymentLinkService),
	fx.Provide(NewDiscountService),
	fx.Provide(NewEventConsumerManager),
	fx.Provide(NewInvoiceOrchestrationService),
)
