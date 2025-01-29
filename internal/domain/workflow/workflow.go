package workflow

import "context"

type Engine interface {
	StartWorkflow(ctx context.Context, id string, payload interface{}) error
}
