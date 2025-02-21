package postgres

import (
	"context"
	"encoding/json"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"
)

type OrderItemRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewOrderItemRepository(database lib.Database, logger logger.Logger) repositories.OrderItemRepository {
	pgDatabase, ok := database.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return OrderItemRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r OrderItemRepository) FindById(ctx context.Context, orgId string, id string) (entities.OrderItem, error) {
	var orderItem models.OrderItem
	var price models.Price
	var metadata []byte

	query := `SELECT oi.org_id, oi.id, oi.order_id, oi.price_id, oi.description, 
       oi.quantity, oi.metadata, oi.created_at, oi.updated_at,
			  
       p.org_id, p.id, p.variant_id, p.category, p.scheme,
       p.currency, p.unit_price, p.min_price, p.suggested_price,
       p.billing_interval, p.billing_interval_qty, p.trial_interval,
       p.trial_interval_qty, p.tax_code, p.metadata, p.created_at, p.updated_at
    
			  FROM order_items oi
			  JOIN prices p ON oi.price_id = p.id
			  WHERE oi.org_id = $1 AND oi.id = $2`

	err := r.Pool.QueryRow(ctx, query, orgId, id).Scan(
		&orderItem.OrgId,
		&orderItem.Id,
		&orderItem.OrderId,
		&orderItem.PriceId,
		&orderItem.Description,
		&orderItem.Quantity,
		&metadata,
		&orderItem.CreatedAt,
		&orderItem.UpdatedAt,

		&price.OrgId,
		&price.Id,
		&price.VariantId,
		&price.Category,
		&price.Scheme,
		&price.Currency,
		&price.UnitPrice,
		&price.MinPrice,
		&price.SuggestedPrice,
		&price.BillingInterval,
		&price.BillingIntervalQty,
		&price.TrialInterval,
		&price.TrialIntervalQty,
		&price.TaxCode,
		&price.Metadata,
		&price.CreatedAt,
		&price.UpdatedAt,
	)
	if err != nil {
		return entities.OrderItem{}, err
	}

	err = json.Unmarshal(metadata, &orderItem.Metadata)
	if err != nil {
		return entities.OrderItem{}, err
	}

	orderItem.Price = price
	return orderItem.ToEntity(), nil
}

// Create inserts a new order item into the database
func (r OrderItemRepository) Create(ctx context.Context, orderItem entities.OrderItem) (entities.OrderItem, error) {
	var p QueryRower = r.Pool
	tx := ctx.Value(lib.DBTransaction)
	if tx != nil {
		p = tx.(QueryRower)
	}
	query := `INSERT INTO order_items (org_id, id, order_id, price_id, description, quantity, metadata, created_at, updated_at)
				  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) `

	metadata, err := json.Marshal(orderItem.Metadata)
	if err != nil {
		return entities.OrderItem{}, err
	}

	_, err = p.Exec(ctx, query,
		orderItem.OrgId,
		orderItem.Id,
		orderItem.OrderId,
		orderItem.PriceId,
		orderItem.Description,
		orderItem.Quantity,
		metadata,
		orderItem.CreatedAt,
		orderItem.UpdatedAt,
	)
	if err != nil {
		return entities.OrderItem{}, err
	}

	return r.FindById(ctx, orderItem.OrgId, orderItem.Id)
}

// FindByOrderId retrieves order items by order Id
func (r OrderItemRepository) FindByOrderId(ctx context.Context, orgId string, orderId string) ([]entities.OrderItem, error) {
	var orderItems []entities.OrderItem

	query := `SELECT oi.org_id, oi.id, oi.order_id, oi.price_id, oi.description, oi.quantity, oi.metadata, oi.created_at, oi.updated_at,
			  p.org_id, p.id, p.trial_interval, p.trial_interval_qty, p.billing_interval, p.billing_interval_qty, p.currency, p.unit_price, p.tax_code
			  FROM order_items oi
			  JOIN prices p ON oi.price_id = p.id
			  WHERE oi.org_id = $1 AND oi.order_id = $2`

	rows, err := r.Pool.Query(ctx, query, orgId, orderId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var orderItem entities.OrderItem
		var price entities.Price
		var metadata []byte

		err := rows.Scan(
			&orderItem.OrgId,
			&orderItem.Id,
			&orderItem.OrderId,
			&orderItem.PriceId,
			&orderItem.Description,
			&orderItem.Quantity,
			&metadata,
			&orderItem.CreatedAt,
			&orderItem.UpdatedAt,

			&price.OrgId,
			&price.Id,
			&price.TrialInterval,
			&price.TrialIntervalQty,
			&price.BillingInterval,
			&price.BillingIntervalQty,
			&price.Currency,
			&price.UnitPrice,
			&price.TaxCode,
		)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(metadata, &orderItem.Metadata)
		if err != nil {
			return nil, err
		}

		orderItem.Price = price
		orderItems = append(orderItems, orderItem)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return orderItems, nil
}

// Update modifies an existing order item in the database
func (r OrderItemRepository) Update(ctx context.Context, orderItem entities.OrderItem) (entities.OrderItem, error) {
	query := `UPDATE order_items
			  SET price_id = $1, description = $2, quantity = $3, metadata = $4, updated_at = $5
			  WHERE org_id = $6 AND id = $7
			  RETURNING org_id, id, order_id, price_id, description, quantity, metadata, created_at, updated_at`

	metadata, err := json.Marshal(orderItem.Metadata)
	if err != nil {
		return entities.OrderItem{}, err
	}

	err = r.Pool.QueryRow(ctx, query,
		orderItem.Description,
		orderItem.Quantity,
		metadata,
		orderItem.UpdatedAt,
		orderItem.OrgId,
		orderItem.Id,
	).Scan(
		&orderItem.OrgId,
		&orderItem.Id,
		&orderItem.OrderId,
		&orderItem.PriceId,
		&orderItem.Description,
		&orderItem.Quantity,
		&orderItem.Metadata,
		&orderItem.CreatedAt,
		&orderItem.UpdatedAt,
	)
	if err != nil {
		return entities.OrderItem{}, err
	}

	return orderItem, nil
}
