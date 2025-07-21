package kafka

import (
	"context"
	"go.uber.org/fx"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/repositories"
)

// Module exports Kafka publisher and consumer dependencies
var Module = fx.Options(
	fx.Provide(NewKafkaPublisherWithDefaultConfig),
	fx.Provide(NewConsumerService),
	fx.Provide(NewReportingEventConsumerWithDefaultConfig),
	fx.Invoke(StartReportingEventConsumer),
)

// NewKafkaPublisherWithDefaultConfig creates a new Kafka publisher with default configuration
func NewKafkaPublisherWithDefaultConfig(logger logger.Logger) (events.DurableEventPublisher, error) {
	config := DefaultConfig()
	return NewKafkaPublisher(config, logger)
}

// NewReportingEventConsumerWithDefaultConfig creates a new reporting event consumer with default configuration
func NewReportingEventConsumerWithDefaultConfig(
	reportRepository repositories.ReportRepository,
	logger logger.Logger,
) (*ReportingEventConsumer, error) {
	config := DefaultConfig()
	return NewReportingEventConsumer(
		config.Brokers,
		[]string{
			events.TopicBillingEvents,
			events.TopicPaymentEvents,
			events.TopicCustomerEvents,
		},
		"reporting-consumer",
		reportRepository,
		logger,
	)
}

// AsKafkaPublisher is a helper function for dependency injection
func AsKafkaPublisher(target interface{}) interface{} {
	return fx.Annotate(
		target,
		fx.As(new(events.DurableEventPublisher)),
	)
}

// StartReportingEventConsumer starts the reporting event consumer
func StartReportingEventConsumer(lc fx.Lifecycle, consumer *ReportingEventConsumer, logger logger.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting reporting event consumer")
			return consumer.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping reporting event consumer")
			return consumer.Stop()
		},
	})
}
