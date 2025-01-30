package temporal

import "payloop/internal/domain/workflow"

// TODO move to domain as interface once i've figured out what the interface should look like
type WorkflowPayload struct {
	engine workflow.Engine
}
