package kafka

import (
	"go.uber.org/fx"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/logger"
)

// Module exports Kafka publisher dependency
var Module = fx.Options(
	fx.Provide(NewKafkaPublisherWithDefaultConfig),
)

// NewKafkaPublisherWithDefaultConfig creates a new Kafka publisher with default configuration
func NewKafkaPublisherWithDefaultConfig(logger logger.Logger) (events.DurableEventPublisher, error) {
	config := DefaultConfig()
	return NewKafkaPublisher(config, logger)
}

// AsKafkaPublisher is a helper function for dependency injection
func AsKafkaPublisher(target interface{}) interface{} {
	return fx.Annotate(
		target,
		fx.As(new(events.DurableEventPublisher)),
	)
}