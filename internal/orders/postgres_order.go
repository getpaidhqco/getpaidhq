package orders

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/segmentio/ksuid"
	"log/slog"
	"payloop/internal/db"
	"payloop/internal/orders/queries"
)

type PgOrderRepository struct {
	db *db.PgDatabase
}

func NewOrderRepository(pgDatabase *db.PgDatabase) *PgOrderRepository {
	return &PgOrderRepository{db: pgDatabase}
}

func (r *PgOrderRepository) CreateOrder(ctx context.Context, input CreateOrderInput) error {

	orderId := "order_" + ksuid.New().String()
	ref := input.Reference
	if ref == "" {
		ref = ksuid.New().String()
	}

	tx, _ := r.Begin(ctx)

	_, err := tx.Exec(ctx, queries.InsertOrderQuery, pgx.NamedArgs{
		"tid":       input.TID,
		"id":        orderId,
		"reference": ref,
		"currency":  input.Currency,
		"total":     input.Total,
	})
	_, err = tx.Exec(ctx, queries.InsertCustomerQuery, pgx.NamedArgs{
		"tid":       input.TID,
		"id":        ksuid.New().String(),
		"reference": ref,
		"currency":  input.Currency,
		"total":     input.Total,
	})

	if err != nil {
		slog.Error("failed to insert order", "error", err)
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		slog.Error("failed to commit transaction", "error", err)
		_ = tx.Rollback(ctx)
		return err
	}
	return nil

}
