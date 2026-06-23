package postgresgorm

import (
	"context"

	"gorm.io/gorm"

	"getpaidhq/internal/core/port"
)

type txKey struct{}

// WithTx stashes a gorm transaction handle on ctx. Used by TxManager.
// Exported so tests and other adapters can bring their own tx.
func WithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// dbFromCtx returns the active transaction handle if one is stashed on
// ctx, otherwise the fallback db. The returned handle already has ctx
// attached — callers should NOT chain a second .WithContext(ctx).
func dbFromCtx(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return fallback.WithContext(ctx)
}

// TxManager opens a gorm transaction and threads it through ctx for the
// duration of the callback. Commits on nil return, rolls back on error
// or panic (panics are re-raised by gorm.Transaction).
type TxManager struct {
	db *gorm.DB
}

func NewTxManager(db *gorm.DB) *TxManager {
	return &TxManager{db: db}
}

// Compile-time check that TxManager satisfies the port.
var _ port.TxManager = (*TxManager)(nil)

func (t *TxManager) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	return dbFromCtx(ctx, t.db).Transaction(func(tx *gorm.DB) error {
		return fn(WithTx(ctx, tx))
	})
}
