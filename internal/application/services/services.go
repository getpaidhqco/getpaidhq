package services

import (
	"go.uber.org/fx"
	"payloop/internal/application/interfaces"
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
	fx.Provide(NewSettingsService),
	fx.Provide(fx.Annotate(
		NewDocumentService,
		fx.As(new(interfaces.DocumentService)),
	)),
	fx.Provide(NewDunningService),
	fx.Provide(NewDunningOrchestrationService),
	fx.Provide(fx.Annotate(
		NewUsageRecordingService,
		fx.As(new(interfaces.UsageRecordingService)),
	)),
)
