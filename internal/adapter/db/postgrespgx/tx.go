package postgrespgx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// querier is the subset of *pgxpool.Pool and pgx.Tx that the repos use. Both
// the pool and an open transaction satisfy it, so dbFromCtx can hand back
// either without the repo caring which. This is the pgx analogue of the gorm
// adapter threading a *gorm.DB.
type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// beginner is satisfied by both *pgxpool.Pool and pgx.Tx. Pool.Begin opens a
// real transaction; Tx.Begin opens a nested transaction implemented as a
// SAVEPOINT — which is exactly the nested-RunInTx semantics gorm gives us.
type beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// scanner is the common Scan surface of pgx.Row and pgx.Rows, so a row type's
// scanInto helper can be reused by both single-row QueryRow and multi-row
// CollectRows code paths.
type scanner interface {
	Scan(dest ...any) error
}

type txKey struct{}

// WithTx stashes a pgx transaction handle on ctx. Used by TxManager. Exported
// so tests can bring their own tx, mirroring the gorm adapter.
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// dbFromCtx returns the active transaction handle if one is stashed on ctx,
// otherwise the fallback pool. Mirrors the gorm adapter's dbFromCtx so every
// repo opts into the ambient transaction the same way.
func dbFromCtx(ctx context.Context, fallback querier) querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok && tx != nil {
		return tx
	}
	return fallback
}

// inTx runs fn inside a transaction so a repo that writes multiple rows
// (invoice + line items, order + items) is atomic regardless of whether a
// caller already opened a RunInTx. If a tx is already on ctx it joins it via a
// SAVEPOINT (matching gorm's nested-transaction semantics); otherwise it opens
// a fresh tx on the pool. fn receives a ctx carrying the (possibly nested) tx.
func inTx(ctx context.Context, pool *pgxpool.Pool, fn func(context.Context) error) error {
	var b beginner = pool
	if existing, ok := ctx.Value(txKey{}).(pgx.Tx); ok && existing != nil {
		b = existing
	}
	tx, err := b.Begin(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	if err := fn(WithTx(ctx, tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}

// TxManager opens a pgx transaction and threads it through ctx for the
// duration of the callback. Commits on nil return, rolls back on error or
// panic (the panic propagates after rollback). A RunInTx nested inside another
// RunInTx reuses the open tx and opens a SAVEPOINT, matching gorm.
type TxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

func (t *TxManager) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	var b beginner = t.pool
	if existing, ok := ctx.Value(txKey{}).(pgx.Tx); ok && existing != nil {
		b = existing // nested → SAVEPOINT
	}

	tx, err := b.Begin(ctx)
	if err != nil {
		return err
	}

	committed := false
	defer func() {
		if !committed {
			// On the happy path the tx is already committed and this is a
			// no-op returning ErrTxClosed, which we deliberately ignore. On
			// error or panic this is the real rollback.
			_ = tx.Rollback(ctx)
		}
	}()

	if err := fn(WithTx(ctx, tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}
