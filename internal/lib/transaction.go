package lib

import (
	"context"
)

// TransactionFunc is a function that executes within a transaction and returns a result
type TransactionFunc[T any] func(ctx context.Context) (T, error)

// WithTransaction executes the given function within a transaction.
// It creates a new transaction, adds it to the context, and executes the function.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
// This utility can be used anywhere in the codebase, not just in API handlers.
// WithTransaction executes the given function within a transaction and returns the result
func WithTransaction[T any](ctx context.Context, db Database, fn TransactionFunc[T]) (T, error) {
	var zero T

	// Begin a new transaction
	txHandle, err := db.Begin(ctx)
	if err != nil {
		return zero, err
	}

	// Create a new context with the transaction
	txCtx := context.WithValue(ctx, DBTransaction, txHandle)

	// Execute the function within the transaction
	result, err := fn(txCtx)

	// Handle commit/rollback based on the result
	if err != nil {
		// Rollback the transaction if there was an error
		rollbackErr := txHandle.Rollback(ctx)
		if rollbackErr != nil {
			// Log the rollback error, but return the original error
			return zero, err
		}
		return zero, err
	}

	// Commit the transaction if there was no error
	if commitErr := txHandle.Commit(ctx); commitErr != nil {
		return zero, commitErr
	}

	return result, nil
}
