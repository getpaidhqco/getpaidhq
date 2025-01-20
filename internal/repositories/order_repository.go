package repositories

import (
	"context"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/db"
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
	db *db.PgDatabase
}

func NewOrderRepository(db *db.PgDatabase) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) FindByID(ctx context.Context, id uint) (*models.Order, error) {
	query := "SELECT id, customer_id, status, total FROM orders WHERE id=$1"
	row := r.db.QueryRow(ctx, query, id)

	var order models.Order
	err := row.Scan(&order.ID, &order.CustomerID, &order.Status, &order.Total)
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *OrderRepository) FindAll(ctx context.Context) ([]*models.Order, error) {
	query := "SELECT id, customer_id, status, total FROM orders"
	rows, err := r.db.Query(ctx, query)
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

func (r *OrderRepository) Create(ctx context.Context, order models.Order) error {
	query := "INSERT INTO orders (customer_id, status, total) VALUES ($1, $2, $3)"
	_, err := r.db.Exec(ctx, query, order.CustomerID, order.Status, order.Total)
	return err
}

func (r *OrderRepository) Update(ctx context.Context, order models.Order) error {
	query := "UPDATE orders SET customer_id=$1, status=$2, total=$3 WHERE id=$4"
	_, err := r.db.Exec(ctx, query, order.CustomerID, order.Status, order.Total, order.ID)
	return err
}

func (r *OrderRepository) Delete(ctx context.Context, id uint) error {
	query := "DELETE FROM orders WHERE id=$1"
	_, err := r.db.Exec(ctx, query, id)
	return err
}
