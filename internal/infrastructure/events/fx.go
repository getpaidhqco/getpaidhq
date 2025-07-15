package events

import (
	"go.uber.org/fx"
	"payloop/internal/infrastructure/events/nats"
)

// Module exports publisher dependencies based on configuration
var Module = fx.Options(
	nats.Module,
	//kinesis.Module,
	//kafka.Module,
)
