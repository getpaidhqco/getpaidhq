package nats

import (
	"go.uber.org/fx"
	"payloop/internal/application/lib/events"
)

// Module exports NATS publisher dependency
var Module = fx.Options(
	fx.Provide(
		fx.Annotate(
			NewNatsNotificationPublisher,
			fx.As(new(events.NotificationPublisher)),
		),
	),
)

// AsNotificationPublisher is a helper function for dependency injection
func AsNotificationPublisher(target interface{}) interface{} {
	return fx.Annotate(
		target,
		fx.As(new(events.NotificationPublisher)),
	)
}