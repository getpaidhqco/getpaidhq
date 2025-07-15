package nats

import (
	"context"
	"sync"

	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/repositories"
)

// ConsumerService manages multiple NATS consumers
// Implements interfaces.ConsumerService
type ConsumerService struct {
	consumers []interfaces.Consumer
	logger    logger.Logger
	wg        sync.WaitGroup
}

// NewConsumerService creates a new consumer service with all consumers
func NewConsumerService(
	usageRepository repositories.UsageEventRepository,
	logger logger.Logger,
) interfaces.ConsumerService {
	// Create usage event consumer
	usageConsumer := NewUsageEventConsumer(
		"gphq.usage.recorded", // Topic name
		usageRepository,
		logger,
	)

	return &ConsumerService{
		consumers: []interfaces.Consumer{usageConsumer},
		logger:    logger,
	}
}

// StartAll starts all consumers
func (s *ConsumerService) StartAll(ctx context.Context) error {
	s.logger.Info("Starting all NATS consumers")

	for _, consumer := range s.consumers {
		s.wg.Add(1)
		go func(c interfaces.Consumer) {
			defer s.wg.Done()
			if err := c.Start(ctx); err != nil {
				s.logger.Error("Consumer failed", "error", err)
			}
		}(consumer)
	}

	return nil
}

// StopAll stops all consumers gracefully
func (s *ConsumerService) StopAll() error {
	s.logger.Info("Stopping all NATS consumers")

	var errors []error
	for _, consumer := range s.consumers {
		if err := consumer.Stop(); err != nil {
			errors = append(errors, err)
			s.logger.Error("Error stopping consumer", "error", err)
		}
	}

	s.wg.Wait()

	if len(errors) > 0 {
		return errors[0] // Return first error
	}

	return nil
}

// Wait waits for all consumers to stop
func (s *ConsumerService) Wait() {
	s.wg.Wait()
}