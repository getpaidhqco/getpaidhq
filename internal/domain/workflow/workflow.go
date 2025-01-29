package workflow

import "context"

type Engine interface {
	StartWorkflow(ctx context.Context, id string, payload interface{}) (Result, error)
}

type Result struct {
	Success bool
	Message string
	Payload interface{}
}
