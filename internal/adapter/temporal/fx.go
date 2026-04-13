package temporal

import (
	"go.uber.org/fx"
	"payloop/internal/adapter/temporal/activities"
)

// Module exports dependency
var Module = fx.Options(
	fx.Provide(activities.NewOrderActivities),
	fx.Provide(activities.NewOutgoingWebhookActivities),
	fx.Provide(NewTemporalEngine),
)
