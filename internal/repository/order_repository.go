package repository

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/domain/customers"
	"payloop/internal/domain/orders"

	"payloop/internal/lib"

	"payloop/internal/models"
)

type OrderRepositoryIf interface {
	FindByID(ctx context.Context, id uint) (*models.Order, error)
	FindAll(ctx context.Context) ([]*models.Order, error)
	Create(ctx context.Context, order models.Order) error
	Update(ctx context.Context, order models.Order) error
	Delete(ctx context.Context, id uint) error
}

type OrderRepository struct {
	*lib.PgDatabase
	logger             lib.Logger
	customerRepository CustomerRepository
}

func NewOrderRepository(database lib.Database, customerRepository CustomerRepository, logger lib.Logger) OrderRepository {
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return OrderRepository{
		PgDatabase:         pgDatabase,
		logger:             logger,
		customerRepository: customerRepository,
	}
}

// WithTrx enables repository with transaction
func (r *OrderRepository) WithTrx(trxHandle interface{}) *OrderRepository {
	if trxHandle == nil {
		r.logger.Warn("Transaction Database not found in gin context. ")
		return r
	}
	r.PgDatabase.Tx = trxHandle.(pgx.Tx)
	return r
}

func (r *OrderRepository) FindByID(ctx context.Context, id uint) (*models.Order, error) {
	query := "SELECT id, customer_id, status, total FROM orders WHERE id=$1"
	row := r.Tx.QueryRow(ctx, query, id)

	var order models.Order
	err := row.Scan(&order.ID, &order.CustomerID, &order.Status, &order.Total)
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *OrderRepository) FindAll(ctx context.Context) ([]*models.Order, error) {
	query := "SELECT id, customer_id, status, total FROM orders"
	rows, err := r.Tx.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(&order.ID, &order.CustomerID, &order.Status, &order.Total)
		if err != nil {
			return nil, err
		}
		orders = append(orders, &order)
	}
	return orders, nil
}

func (r *OrderRepository) Create(ctx context.Context, input orders.CreateOrderRow) (models.Order, error) {

	var order models.Order

	query := `INSERT INTO orders (acct_id,id,customer_id,status,currency,total,metadata, created_at, updated_at) 
			  VALUES (@acct_id,@id,@customer_id,@status,@currency,@total,@metadata, NOW(), NOW())`

	metaJson, _ := json.Marshal(input.Metadata)

	customer, err := r.customerRepository.Create(ctx, customers.CreateCustomerInput{
		AccountId: input.AccountId,
		Email:     "test",
		Name:      "test",
		Metadata:  input.Metadata,
	})
	if err != nil {
		r.logger.Error(`failed to create customer`, err)
		return models.Order{}, err
	}

	err = r.Pool.QueryRow(ctx, query, pgx.NamedArgs{
		"acct_id":     input.AccountId,
		"id":          lib.GenerateId("order"),
		"customer_id": customer.ID,
		"status":      models.OrderStatusPending,
		"currency":    input.Currency,
		"total":       0,
		"metadata":    metaJson,
	}).Scan(&order)

	if err != nil {
		r.logger.Error(`failed to insert Order`, err)
		return models.Order{}, err
	}

	if err != nil {
		r.logger.Error(`failed to insert Order`, err)
		return models.Order{}, err
	}

	return order, nil
}
