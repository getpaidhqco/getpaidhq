package services

import (
	"go.uber.org/fx"
)

// Module exports services present
var Module = fx.Options(
	fx.Provide(NewUserService),
	fx.Provide(NewOrderService),
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
)
