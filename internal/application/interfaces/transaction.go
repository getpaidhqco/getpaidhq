package interfaces

import (
	"context"
)

// TransactionService provides transaction management capabilities
type TransactionService interface {
	// WithTransaction executes the given function within a transaction and returns the result
	WithTransaction(ctx context.Context, fn func(ctx context.Context) (any, error)) (any, error)
}
