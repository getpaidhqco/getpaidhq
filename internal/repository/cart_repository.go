package repository

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/domain/carts"
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

// WithTrx enables repository with transaction
func (r *CartRepository) WithTrx(trxHandle interface{}) *CartRepository {
	if trxHandle == nil {
		r.logger.Warn("Transaction Database not found in gin context. ")
		return r
	}
	r.PgDatabase.Tx = trxHandle.(pgx.Tx)
	return r
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

func (r *CartRepository) Create(ctx context.Context, input carts.CreateCartInput) (models.Cart, error) {
	cartId := lib.GenerateId("cart")

	query := `INSERT INTO carts (acct_id,id,data,metadata,created_at,updated_at) 
			  VALUES (@acct_id,@id,@data,@metadata,NOW(), NOW())`

	metaJson, _ := json.Marshal(input.Metadata)

	_, err := r.Tx.Exec(ctx, query, pgx.NamedArgs{
		"acct_id":  input.AccountId,
		"id":       cartId,
		"data":     input.Cart,
		"metadata": metaJson,
	})

	if err != nil {
		r.logger.Error(`failed to insert Cart`, err)
		return models.Cart{}, err
	}

	return models.Cart{
		Id:     cartId,
		Data:   input.Cart,
		Status: "",
		Total:  0,
	}, nil
}
