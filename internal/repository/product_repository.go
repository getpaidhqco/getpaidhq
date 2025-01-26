package repository

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/domain/entities"
	"payloop/internal/lib"
)

type ProductRepository struct {
	*lib.PgDatabase
	logger lib.Logger
}

func NewProductRepository(database lib.Database, logger lib.Logger) ProductRepository {
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return ProductRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

// WithTrx enables repository with transaction
func (r *ProductRepository) WithTrx(trxHandle interface{}) *ProductRepository {
	if trxHandle == nil {
		r.logger.Warn("Transaction Database not found in gin context. ")
		return r
	}
	r.PgDatabase.Tx = trxHandle.(pgx.Tx)
	return r
}

func (r *ProductRepository) FindByID(ctx context.Context, acctId string, id string) (entities.Product, error) {
	var product entities.Product
	err := r.Pool.QueryRow(ctx, `SELECT acct_id,id,name,description,metadata 
							FROM products WHERE acct_id=@acct_id AND id=@id`,
		pgx.NamedArgs{
			"acct_id": acctId,
			"id":      id,
		}).Scan(
		&product.AccountId,
		&product.Id,
		&product.Name,
		&product.Metadata,
		&product.Description,
	)

	if err != nil {
		r.logger.Error(`failed to find Product`, err.Error())
		return entities.Product{}, errors.New("not found")
	}
	return product, nil
}
