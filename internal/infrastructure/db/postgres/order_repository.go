package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
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
	var order entities.Order
	var customer entities.Customer

	query := `SELECT orders.org_id, orders.id, orders.customer_id, orders.reference, orders.status, orders.session_id, orders.cart_id, orders.currency, orders.total, orders.metadata, orders.created_at, orders.updated_at,
                 c.org_id, c.id, c.email, c.name, c.created_at, c.updated_at
			  FROM orders
			  JOIN customers c ON orders.org_id=c.org_id AND orders.customer_id = c.id
			  WHERE orders.org_id = $1 AND orders.id = $2`

	err := r.Pool.QueryRow(ctx, query, orgId, id).Scan(
		&order.OrgId,
		&order.Id,
		&order.CustomerId,
		&order.Reference,
		&order.Status,
		&order.SessionId,
		&order.CartId,
		&order.Currency,
		&order.Total,
		&order.Metadata,
		&order.CreatedAt,
		&order.UpdatedAt,

		&customer.OrgId,
		&customer.Id,
		&customer.Email,
		&customer.Name,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			fmt.Println(pgErr.Message) // => syntax error at end of input
			fmt.Println(pgErr.Code)    // => 42601
			r.logger.Error("failed to find Order", "err", pgErr.Message, "code", pgErr.Code)
		}
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Error("Order not found")
		}
		return entities.Order{}, err
	}

	return order, nil
}

func (r OrderRepository) Create(ctx context.Context, entity entities.Order) (entities.Order, error) {
	p := r.Pool
	tx := ctx.Value(lib.DBTransaction).(lib.Committer)
	if tx != nil {
		p = tx.GetClient().(*pgxpool.Pool)
	}

	var order entities.Order

	query := `INSERT INTO orders (org_id,id,customer_id,cart_id,reference,status,session_id,currency,total,metadata, created_at, updated_at) 
			  VALUES (@org_id,@id,@customer_id,@cart_id,@reference,@status,@session_id, @currency,@total,@metadata, NOW(), NOW())
			  RETURNING org_id,id,customer_id,reference,status,session_id,cart_id,currency,total,metadata,created_at, updated_at`

	metaJson, _ := json.Marshal(entity.Metadata)

	err := p.QueryRow(ctx, query, pgx.NamedArgs{
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
	}).Scan(
		&order.OrgId,
		&order.Id,
		&order.CustomerId,
		&order.Reference,
		&order.Status,
		&order.SessionId,
		&order.CartId,
		&order.Currency,
		&order.Total,
		&order.Metadata,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to insert Order`, err.Error())
		return entities.Order{}, err
	}

	return order, nil
}

// Update updates an existing order in the database and joins with the customer
func (r OrderRepository) Update(ctx context.Context, entity entities.Order) (entities.Order, error) {

	query := `UPDATE orders
				SET customer_id = @customer_id, cart_id = @cart_id, reference = @reference, status = @status, session_id = @session_id, currency = @currency, total = @total, metadata = @metadata, updated_at = NOW()
				WHERE org_id = @org_id AND id = @id`

	metaJson, _ := json.Marshal(entity.Metadata)

	_, err := r.Pool.Exec(ctx, query, pgx.NamedArgs{
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
	})

	if err != nil {
		r.logger.Error(`failed to update Order`, err.Error())
		return entities.Order{}, err
	}

	return r.FindById(ctx, entity.OrgId, entity.Id)
}
