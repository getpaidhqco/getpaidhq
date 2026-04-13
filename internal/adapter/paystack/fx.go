package paystack

import "go.uber.org/fx"

// Module exports dependency
var Module = fx.Options(
	fx.Provide(NewPaystackFactory),
	fx.Provide(NewWebhookParser),
)
