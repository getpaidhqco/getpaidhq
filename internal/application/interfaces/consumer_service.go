package interfaces

import "context"

// Consumer defines the interface for individual event consumers
type Consumer interface {
	Start(ctx context.Context) error
	Stop() error
}

// ConsumerService defines the interface for managing multiple consumers
type ConsumerService interface {
	StartAll(ctx context.Context) error
	StopAll() error
	Wait()
}

// EventConsumerManager defines the interface for managing event consumer lifecycle
type EventConsumerManager interface {
	Start(ctx context.Context) error
	Stop() error
	Wait()
}