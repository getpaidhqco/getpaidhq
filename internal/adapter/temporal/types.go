package temporal

import (
	"payloop/internal/core/port"
)

// TODO move to domain as interface once i've figured out what the interface should look like
type WorkflowPayload struct {
	engine port.Engine
}
