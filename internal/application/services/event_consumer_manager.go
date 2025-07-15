package services

import (
	"context"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
)

// EventConsumerManager manages the lifecycle of event consumers
// Implements interfaces.EventConsumerManager
type EventConsumerManager struct {
	consumerService interfaces.ConsumerService
	logger          logger.Logger
}

// NewEventConsumerManager creates a new event consumer manager
func NewEventConsumerManager(
	consumerService interfaces.ConsumerService,
	logger logger.Logger,
) interfaces.EventConsumerManager {
	return &EventConsumerManager{
		consumerService: consumerService,
		logger:          logger,
	}
}

// Start starts all event consumers
func (m *EventConsumerManager) Start(ctx context.Context) error {
	m.logger.Info("Starting event consumer manager")
	return m.consumerService.StartAll(ctx)
}

// Stop stops all event consumers
func (m *EventConsumerManager) Stop() error {
	m.logger.Info("Stopping event consumer manager")
	return m.consumerService.StopAll()
}

// Wait waits for all consumers to stop
func (m *EventConsumerManager) Wait() {
	m.consumerService.Wait()
}