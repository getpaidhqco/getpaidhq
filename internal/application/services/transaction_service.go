package services

import (
	"context"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

// TransactionService provides transaction management capabilities
type TransactionService struct {
	db     lib.Database `name:"primaryDb"`
	logger logger.Logger
}

// NewTransactionService creates a new transaction service
func NewTransactionService(db lib.Database, logger logger.Logger) interfaces.TransactionService {
	return &TransactionService{
		db:     db,
		logger: logger,
	}
}

// WithTransaction executes the given function within a transaction
func (s *TransactionService) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return lib.WithTransaction(ctx, s.db, fn)
}