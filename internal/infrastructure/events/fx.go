package events

import (
	"go.uber.org/fx"
	"payloop/internal/infrastructure/events/kafka"
	"payloop/internal/infrastructure/events/nats"
)

// Module exports both NATS and Kafka publisher dependencies
var Module = fx.Options(
	nats.Module,
	kafka.Module,
)
