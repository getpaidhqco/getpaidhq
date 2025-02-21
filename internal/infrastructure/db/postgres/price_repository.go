package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"log/slog"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"
)

type PriceRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewPriceRepository(database lib.Database, logger logger.Logger) repositories.PriceRepository {
	pgDatabase, ok := database.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return PriceRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

// WithTrx enables repository with transaction
func (r PriceRepository) WithTrx(trxHandle interface{}) PriceRepository {
	if trxHandle == nil {
		r.logger.Warn("Transaction Database not found in gin context. ")
		return r
	}
	r.PgDatabase.Tx = trxHandle.(pgx.Tx)
	return r
}

func (r PriceRepository) FindById(ctx context.Context, orgId string, id string) (entities.Price, error) {
	var price models.Price
	err := r.Pool.QueryRow(ctx, `SELECT org_id,id,billing_interval,billing_interval_qty,
       category,scheme,cycles,currency,unit_price,
       trial_interval,trial_interval_qty,tax_code,
       updated_at,
       updated_at
							FROM prices WHERE org_id=@org_id AND id=@id`,
		pgx.NamedArgs{
			"org_id": orgId,
			"id":     id,
		}).Scan(
		&price.OrgId,
		&price.Id,
		&price.BillingInterval,
		&price.BillingIntervalQty,
		&price.Category,
		&price.Scheme,
		&price.Cycles,
		&price.Currency,
		&price.UnitPrice,
		&price.TrialInterval,
		&price.TrialIntervalQty,
		&price.TaxCode,
		&price.UpdatedAt,
		&price.CreatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to find Price`, slog.String("err", err.Error()))
		return entities.Price{}, errors.New("not found")
	}
	return price.ToEntity(), nil
}
