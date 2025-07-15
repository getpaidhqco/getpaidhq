package nats

import (
	"go.uber.org/fx"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
)

// Module exports NATS publisher and consumer dependencies
var Module = fx.Options(
	fx.Provide(
		fx.Annotate(
			NewNatsNotificationPublisher,
			fx.As(new(events.NotificationPublisher)),
		),
		fx.Annotate(
			NewNatsDurablePublisher,
			fx.As(new(events.DurableEventPublisher)),
		),
		fx.Annotate(
			NewConsumerService,
			fx.As(new(interfaces.ConsumerService)),
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

// AsDurablePublisher is a helper function for dependency injection
func AsDurablePublisher(target interface{}) interface{} {
	return fx.Annotate(
		target,
		fx.As(new(events.DurableEventPublisher)),
	)
}
