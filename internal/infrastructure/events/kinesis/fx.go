package kinesis

import (
	"go.uber.org/fx"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/logger"
)

// Module exports Kinesis publisher dependencies
var Module = fx.Options(
	fx.Provide(NewKinesisPublisherWithDefaultConfig),
)

// NewKinesisPublisherWithDefaultConfig creates a new Kinesis publisher with default configuration
func NewKinesisPublisherWithDefaultConfig(logger logger.Logger) (events.DurableEventPublisher, error) {
	config := DefaultConfig()
	return NewKinesisPublisher(config, logger)
}

// AsKinesisPublisher is a helper function for dependency injection
func AsKinesisPublisher(target interface{}) interface{} {
	return fx.Annotate(
		target,
		fx.As(new(events.DurableEventPublisher)),
	)
}