package repository

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/domain/entities"
	"payloop/internal/lib"
)

type PriceRepository struct {
	*lib.PgDatabase
	logger lib.Logger
}

func NewPriceRepository(database lib.Database, logger lib.Logger) PriceRepository {
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return PriceRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

// WithTrx enables repository with transaction
func (r *PriceRepository) WithTrx(trxHandle interface{}) *PriceRepository {
	if trxHandle == nil {
		r.logger.Warn("Transaction Database not found in gin context. ")
		return r
	}
	r.PgDatabase.Tx = trxHandle.(pgx.Tx)
	return r
}

func (r *PriceRepository) FindByID(ctx context.Context, acctId string, id string) (entities.Price, error) {
	var price entities.Price
	err := r.Pool.QueryRow(ctx, `SELECT acct_id,id,billing_interval,billing_interval_qty,category,scheme,currency,unit_price,trial_interval,trial_interval_qty,tax_code 
							FROM prices WHERE acct_id=@acct_id AND id=@id`,
		pgx.NamedArgs{
			"acct_id": acctId,
			"id":      id,
		}).Scan(
		&price.AccountId,
		&price.Id,
		&price.BillingInterval,
		&price.BillingIntervalQty,
		&price.Category,
		&price.Scheme,
		&price.Currency,
		&price.UnitPrice,
		&price.TrialInterval,
		&price.TrialIntervalQty,
		&price.TaxCode,
	)

	if err != nil {
		r.logger.Error(`failed to find Price`, err.Error())
		return entities.Price{}, errors.New("not found")
	}
	return price, nil
}
