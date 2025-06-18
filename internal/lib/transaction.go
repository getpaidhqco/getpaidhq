package lib

import (
	"context"
)

// TransactionFunc is a function that executes within a transaction
type TransactionFunc func(ctx context.Context) error

// WithTransaction executes the given function within a transaction.
// It creates a new transaction, adds it to the context, and executes the function.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
// This utility can be used anywhere in the codebase, not just in API handlers.
func WithTransaction(ctx context.Context, db Database, fn TransactionFunc) error {
	// Begin a new transaction
	txHandle, err := db.Begin(ctx)
	if err != nil {
		return err
	}

	// Create a new context with the transaction
	txCtx := context.WithValue(ctx, DBTransaction, txHandle)

	// Execute the function within the transaction
	err = fn(txCtx)

	// Handle commit/rollback based on the result
	if err != nil {
		// Rollback the transaction if there was an error
		rollbackErr := txHandle.Rollback(ctx)
		if rollbackErr != nil {
			// Log the rollback error, but return the original error
			return err
		}
		return err
	}

	// Commit the transaction if there was no error
	if commitErr := txHandle.Commit(ctx); commitErr != nil {
		return commitErr
	}

	return nil
}