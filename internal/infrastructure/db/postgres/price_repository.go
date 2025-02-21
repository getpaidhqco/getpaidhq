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

func (r PriceRepository) Create(ctx context.Context, entity entities.Price) (entities.Price, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO prices (org_id, id, variant_id, category, scheme, cycles, currency, 
                    unit_price, min_price, suggested_price, billing_interval, billing_interval_qty, 
                    trial_interval, trial_interval_qty, tax_code, metadata,
                    created_at, updated_at)
        VALUES (@org_id, @id, @variant_id, @category, @scheme, @cycles, @currency, 
                @unit_price, @min_price, @suggested_price, @billing_interval, @billing_interval_qty, 
                @trial_interval, @trial_interval_qty, @tax_code, @metadata,
                NOW(), NOW())
       `

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":               entity.OrgId,
		"id":                   entity.Id,
		"variant_id":           entity.VariantId,
		"category":             entity.Category,
		"scheme":               entity.Scheme,
		"cycles":               entity.Cycles,
		"currency":             entity.Currency,
		"unit_price":           entity.UnitPrice,
		"min_price":            entity.MinPrice,
		"suggested_price":      entity.SuggestedPrice,
		"billing_interval":     entity.BillingInterval,
		"billing_interval_qty": entity.BillingIntervalQty,
		"trial_interval":       entity.TrialInterval,
		"trial_interval_qty":   entity.TrialIntervalQty,
		"tax_code":             entity.TaxCode,
		"metadata":             entity.Metadata,
	})

	if err != nil {
		r.logger.Error(`failed to create Price`, err.Error())
		return entities.Price{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r PriceRepository) FindById(ctx context.Context, orgId string, id string) (entities.Price, error) {
	tx := r.getTransactionFromContext(ctx)
	var price models.Price
	err := tx.QueryRow(ctx, `SELECT org_id,id,billing_interval,billing_interval_qty,
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
