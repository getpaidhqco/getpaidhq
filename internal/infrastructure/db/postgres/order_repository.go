package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"
)

type OrderRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewOrderRepository(database lib.Database, logger logger.Logger) repositories.OrderRepository {
	pgDatabase, ok := database.(*PgDatabase)
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
	tx := r.getTransactionFromContext(ctx)

	var order entities.Order
	var customer models.Customer

	query := `SELECT orders.org_id, orders.id, orders.customer_id, orders.reference,
       				orders.status, orders.session_id, orders.cart_id, orders.currency, orders.total, 
       				orders.metadata, orders.created_at, orders.updated_at,
       				
                 	c.org_id, c.id, c.email, c.first_name, c.last_name, c.created_at, c.updated_at
			  FROM orders
			      
			  JOIN customers c ON orders.org_id=c.org_id AND orders.customer_id = c.id
			  
			  WHERE orders.org_id = $1 AND orders.id = $2`

	err := tx.QueryRow(ctx, query, orgId, id).Scan(
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
		&customer.FirstName,
		&customer.LastName,
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

	order.Customer = customer.ToEntity()

	// Get all the order items
	orderItemsRepo := NewOrderItemRepository(r.PgDatabase, r.logger)
	orderItems, err := orderItemsRepo.FindByOrderId(ctx, orgId, id)
	if err != nil {
		r.logger.Error("failed to find Order Items", err.Error())
		return entities.Order{}, err
	}
	order.Items = orderItems

	return order, nil
}

func (r OrderRepository) Create(ctx context.Context, entity entities.Order) (entities.Order, error) {
	tx := r.getTransactionFromContext(ctx)

	var order entities.Order

	query := `INSERT INTO orders (org_id,id,customer_id,cart_id,reference,status,session_id,currency,total,metadata, created_at, updated_at) 
			  VALUES (@org_id,@id,@customer_id,@cart_id,@reference,@status,@session_id, @currency,@total,@metadata, NOW(), NOW())
			  RETURNING org_id,id,customer_id,reference,status,session_id,cart_id,currency,total,metadata,created_at, updated_at`

	metaJson, _ := json.Marshal(entity.Metadata)

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
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
	tx := r.getTransactionFromContext(ctx)

	query := `UPDATE orders
				SET customer_id = @customer_id, cart_id = @cart_id, reference = @reference, status = @status, session_id = @session_id, currency = @currency, total = @total, metadata = @metadata, updated_at = NOW()
				WHERE org_id = @org_id AND id = @id`

	metaJson, _ := json.Marshal(entity.Metadata)

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
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

// Find retrieves orders based on the given pagination and sorting parameters.
func (r OrderRepository) Find(ctx context.Context, orgId string, p request.Pagination) ([]entities.Order, int, error) {
	tx := r.getTransactionFromContext(ctx)
	r.logger.Debugf("sort_dir[%s] sort_col[%s]", p.SortDirection, p.SortBy)

	var orders = make([]entities.Order, 0)
	var count int
	query := `SELECT o.org_id, o.id, o.customer_id, o.reference, 
       o.status, o.session_id, o.cart_id, o.currency, o.total, 
       o.metadata, o.created_at, o.updated_at,
       c.org_id, c.id, c.email, c.first_name, c.last_name, c.created_at, c.updated_at,
       count(*) OVER()
   FROM orders o
   JOIN customers c ON o.org_id=c.org_id AND o.customer_id = c.id
   WHERE o.org_id = @org_id
   ORDER BY
   CASE
        WHEN @sort_dir = 'asc' THEN
            CASE @sort_col
                WHEN 'created_at' THEN o.created_at
                ELSE NULL
                END
        ELSE
            NULL
        END
        ASC,
    CASE
        WHEN @sort_dir = 'desc' THEN
            CASE @sort_col
                WHEN 'created_at' THEN o.created_at
                ELSE NULL
                END
        ELSE
            NULL
        END
        DESC
	LIMIT @lim OFFSET @off;`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":   orgId,
		"lim":      p.Limit,
		"off":      p.Offset,
		"sort_col": p.SortBy,
		"sort_dir": p.SortDirection,
	})
	if err != nil {
		r.logger.Error(`failed to find Orders`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var order entities.Order
		var customer models.Customer

		err := rows.Scan(
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
			&customer.FirstName,
			&customer.LastName,
			&customer.CreatedAt,
			&customer.UpdatedAt,

			&count,
		)
		if err != nil {
			r.logger.Error(`failed to scan Order`, err.Error())
			return nil, 0, err
		}
		order.Customer = customer.ToEntity()
		orders = append(orders, order)
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return orders, count, nil
}
