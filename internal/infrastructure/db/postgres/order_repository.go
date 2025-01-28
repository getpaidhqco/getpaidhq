package postgres

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type OrderRepository struct {
	*lib.PgDatabase
	logger lib.Logger
}

func NewOrderRepository(database lib.Database, logger lib.Logger) repositories.OrderRepository {
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return OrderRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

// WithTrx enables repository with transaction
func (r OrderRepository) WithTrx(trxHandle interface{}) OrderRepository {
	if trxHandle == nil {
		r.logger.Warn("Transaction Database not found in gin context. ")
		return r
	}
	r.PgDatabase.Tx = trxHandle.(pgx.Tx)
	return r
}

func (r OrderRepository) FindById(ctx context.Context, orgId string, id string) (entities.Order, error) {
	query := "SELECT id, customer_id, status, total FROM orders WHERE id=$1"
	row := r.Tx.QueryRow(ctx, query, id)

	var order entities.Order
	err := row.Scan(&order.Id, &order.CustomerId, &order.Status, &order.Total)
	if err != nil {
		return entities.Order{}, err
	}
	return order, nil
}

func (r OrderRepository) Create(ctx context.Context, entity entities.Order) (entities.Order, error) {

	var order entities.Order

	query := `INSERT INTO orders (org_id,id,customer_id,cart_id,reference,status,session_id,currency,total,metadata, created_at, updated_at) 
			  VALUES (@org_id,@id,@customer_id,@cart_id,@reference,@status,@session_id, @currency,@total,@metadata, NOW(), NOW())
			  RETURNING (org_id,id,customer_id,reference,status,session_id,cart_id,currency,total,metadata,created_at, updated_at)`

	metaJson, _ := json.Marshal(entity.Metadata)

	err := r.Pool.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id":      entity.OrgId,
		"id":          entity.Id,
		"customer_id": entity.CustomerId,
		"cart_id":     entity.CartId,
		"reference":   entity.Reference,
		"session_id":  entity.SessionId,
		"status":      entity.Status,
		"currency":    entity.Currency,
		"total":       entity.Total,
		"metadata":    metaJson,
	}).Scan(&order)

	if err != nil {
		r.logger.Error(`failed to insert Order`, err.Error())
		return entities.Order{}, err
	}

	return order, nil
}
