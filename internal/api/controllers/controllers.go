package controllers

import "go.uber.org/fx"

// Module exported for initializing application
var Module = fx.Options(
	fx.Provide(NewUserController),
	fx.Provide(NewOrderController),
	fx.Provide(NewOrgController),
	fx.Provide(NewCartController),
	fx.Provide(NewSessionController),
	fx.Provide(NewWebhookController),
	fx.Provide(NewSubscriptionController),
	fx.Provide(NewWebhookSubscriptionController),
	fx.Provide(NewHealthController),
	fx.Provide(NewProductController),
	fx.Provide(NewCustomerController),
)
