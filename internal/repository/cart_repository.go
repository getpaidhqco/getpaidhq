package repository

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/domain/cart"
	"payloop/internal/lib"

	"payloop/internal/models"
)

type CartRepositoryIf interface {
	FindByID(ctx context.Context, id uint) (*models.Cart, error)
	FindAll(ctx context.Context) ([]*models.Cart, error)
	Create(ctx context.Context, order models.Cart) error
	Update(ctx context.Context, order models.Cart) error
	Delete(ctx context.Context, id uint) error
}

type CartRepository struct {
	*lib.PgDatabase
	logger lib.Logger
}

func NewCartRepository(database lib.Database, logger lib.Logger) CartRepository {
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return CartRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r *CartRepository) FindByID(ctx context.Context, acctId string, id string) (models.Cart, error) {
	var cart models.Cart
	err := r.Pool.QueryRow(ctx, `SELECT * FROM carts WHERE acct_id=@acct_id AND id=@id`, pgx.NamedArgs{
		"acctId": acctId,
		"id":     id,
	}).Scan(&cart)

	if err != nil {
		r.logger.Error(`failed to find Cart`, err)
		return models.Cart{}, err
	}
	return cart, nil
}

func (r *CartRepository) Create(ctx context.Context, input cart.CreateCartInput) (models.Cart, error) {

	var order models.Cart

	query := `INSERT INTO carts (acct_id,id,data,metadata,created_at,updated_at) 
			  VALUES (@acct_id,@id,@data,@metadata,@metadata, NOW(), NOW())`

	metaJson, _ := json.Marshal(input.Metadata)

	err := r.Pool.QueryRow(ctx, query, pgx.NamedArgs{
		"acctId":   input.AccountId,
		"id":       lib.GenerateId("cart"),
		"data":     input.Cart,
		"metadata": metaJson,
	}).Scan(&order)

	if err != nil {
		r.logger.Error(`failed to insert Cart`, err)
		return models.Cart{}, err
	}

	if err != nil {
		r.logger.Error(`failed to insert Cart`, err)
		return models.Cart{}, err
	}

	return order, nil
}
