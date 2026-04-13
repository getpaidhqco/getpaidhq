package postgres

import (
	"context"
	"encoding/json"
	_ "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"
)

type OrderItemRepository struct {
	*PgDatabase
	logger port.Logger
}

func NewOrderItemRepository(primaryDb lib.Database, logger port.Logger) port.OrderRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return OrderItemRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r OrderItemRepository) FindOrderItemById(ctx context.Context, orgId string, id string) (domain.OrderItem, error) {
	tx := r.getTransactionFromContext(ctx)

	var orderItem models.OrderItem
	var price models.Price
	var metadata []byte

	query := `SELECT oi.org_id, oi.id, oi.order_id, oi.price_id, oi.description,
       oi.quantity, oi.metadata, oi.created_at, oi.updated_at,

       p.org_id, p.id, p.variant_id, p.category, p.scheme,
       p.label, p.currency, p.unit_price,p.cycles,
        p.billing_interval, p.billing_interval_qty,
        p.trial_interval, p.trial_interval_qty,
        p.min_price, p.suggested_price,
        p.tax_code, p.metadata, p.created_at, p.updated_at

			  FROM order_items oi
			  JOIN prices p ON oi.price_id = p.id
			  WHERE oi.org_id = $1 AND oi.id = $2`

	err := tx.QueryRow(ctx, query, orgId, id).Scan(
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
		&price.Label,
		&price.Currency,
		&price.UnitPrice,
		&price.Cycles,
		&price.BillingInterval,
		&price.BillingIntervalQty,
		&price.TrialInterval,
		&price.TrialIntervalQty,
		&price.MinPrice,
		&price.SuggestedPrice,
		&price.TaxCode,
		&price.Metadata,
		&price.CreatedAt,
		&price.UpdatedAt,
	)
	if err != nil {
		return domain.OrderItem{}, err
	}

	err = json.Unmarshal(metadata, &orderItem.Metadata)
	if err != nil {
		return domain.OrderItem{}, err
	}

	orderItem.Price = price
	return orderItem.ToEntity(), nil
}

// CreateOrderItem inserts a new order item into the database
func (r OrderItemRepository) CreateOrderItem(ctx context.Context, orderItem domain.OrderItem) (domain.OrderItem, error) {
	tx := r.getTransactionFromContext(ctx)
	query := `INSERT INTO order_items (org_id, id, order_id, product_id, variant_id,
                         price_id, description,
                         quantity, sub_total, tax_total, total, discount_total,
                         metadata, created_at, updated_at)
				  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW()) `

	metadata, err := json.Marshal(orderItem.Metadata)
	if err != nil {
		return domain.OrderItem{}, err
	}

	_, err = tx.Exec(ctx, query,
		orderItem.OrgId,
		orderItem.Id,
		orderItem.OrderId,
		orderItem.ProductId,
		pgtype.Text{String: orderItem.VariantId, Valid: orderItem.VariantId != ""},
		orderItem.PriceId,
		orderItem.Description,
		orderItem.Quantity,
		orderItem.Subtotal,
		orderItem.TaxTotal,
		orderItem.Total,
		orderItem.DiscountTotal,
		metadata,
	)
	if err != nil {
		return domain.OrderItem{}, err
	}

	return r.FindOrderItemById(ctx, orderItem.OrgId, orderItem.Id)
}

// FindOrderItemsByOrderId retrieves order items by order Id
func (r OrderItemRepository) FindOrderItemsByOrderId(ctx context.Context, orgId string, orderId string) ([]domain.OrderItem, error) {
	tx := r.getTransactionFromContext(ctx)
	var orderItems []domain.OrderItem

	query := `SELECT oi.org_id, oi.id, oi.order_id, oi.price_id, oi.description, oi.product_id, oi.variant_id,
      				 oi.quantity, oi.sub_total, oi.tax_total, oi.total, oi.discount_total,
       				 oi.metadata, oi.created_at, oi.updated_at,
			  		 p.org_id, p.id, p.trial_interval, p.trial_interval_qty, p.billing_interval, p.billing_interval_qty, p.currency, p.unit_price, p.tax_code
			  FROM order_items oi
			  JOIN prices p ON oi.org_id = p.org_id AND oi.price_id = p.id
			  WHERE oi.org_id = $1 AND oi.order_id = $2`

	rows, err := tx.Query(ctx, query, orgId, orderId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var orderItem models.OrderItem
		var price models.Price
		var metadata []byte

		err := rows.Scan(
			&orderItem.OrgId,
			&orderItem.Id,
			&orderItem.OrderId,
			&orderItem.PriceId,
			&orderItem.Description,
			&orderItem.ProductId,
			&orderItem.VariantId,
			&orderItem.Quantity,
			&orderItem.Subtotal,
			&orderItem.TaxTotal,
			&orderItem.Total,
			&orderItem.DiscountTotal,
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
		orderItems = append(orderItems, orderItem.ToEntity())
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return orderItems, nil
}

// UpdateOrderItem modifies an existing order item in the database
func (r OrderItemRepository) UpdateOrderItem(ctx context.Context, orderItem domain.OrderItem) (domain.OrderItem, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `UPDATE order_items
			  SET price_id = $1, description = $2, quantity = $3, metadata = $4, updated_at = $5
			  WHERE org_id = $6 AND id = $7
			  RETURNING org_id, id, order_id, price_id, description, quantity, metadata, created_at, updated_at`

	metadata, err := json.Marshal(orderItem.Metadata)
	if err != nil {
		return domain.OrderItem{}, err
	}

	err = tx.QueryRow(ctx, query,
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
		return domain.OrderItem{}, err
	}

	return orderItem, nil
}

// Stub implementations for OrderRepository interface methods that delegate to OrderRepository
// These are needed because NewOrderItemRepository returns port.OrderRepository

func (r OrderItemRepository) FindById(ctx context.Context, orgId string, id string) (domain.Order, error) {
	panic("use OrderRepository.FindById instead")
}

func (r OrderItemRepository) Create(ctx context.Context, entity domain.Order) (domain.Order, error) {
	panic("use OrderRepository.Create instead")
}

func (r OrderItemRepository) Update(ctx context.Context, entity domain.Order) (domain.Order, error) {
	panic("use OrderRepository.Update instead")
}

func (r OrderItemRepository) Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Order, int, error) {
	panic("use OrderRepository.Find instead")
}

// FindByOrderId is a helper used internally by OrderRepository
func (r OrderItemRepository) FindByOrderId(ctx context.Context, orgId string, orderId string) ([]domain.OrderItem, error) {
	return r.FindOrderItemsByOrderId(ctx, orgId, orderId)
}
