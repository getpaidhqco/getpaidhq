package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"log/slog"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"
)

type PriceRepository struct {
	*PgDatabase
	logger port.Logger
}

func NewPriceRepository(primaryDb lib.Database, logger port.Logger) port.PriceRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return PriceRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r PriceRepository) Create(ctx context.Context, entity domain.Price) (domain.Price, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO prices (org_id, id, label, variant_id, category, scheme, cycles, currency,
                    unit_price, min_price, suggested_price, billing_interval, billing_interval_qty,
                    trial_interval, trial_interval_qty, tax_code, metadata,
                    created_at, updated_at)
        VALUES (@org_id, @id, @label, @variant_id, @category, @scheme, @cycles, @currency,
                @unit_price, @min_price, @suggested_price, @billing_interval, @billing_interval_qty,
                @trial_interval, @trial_interval_qty, @tax_code, @metadata,
                NOW(), NOW())
       `

	metadata, err := json.Marshal(entity.Metadata)
	if err != nil {
		return domain.Price{}, err
	}

	_, err = tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":               entity.OrgId,
		"id":                   entity.Id,
		"label":                entity.Label,
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
		"metadata":             metadata,
	})

	if err != nil {
		r.logger.Error(`failed to create Price`, err.Error())
		return domain.Price{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r PriceRepository) FindById(ctx context.Context, orgId string, id string) (domain.Price, error) {
	tx := r.getTransactionFromContext(ctx)
	var price models.Price
	err := tx.QueryRow(ctx, `SELECT org_id,id,variant_id,label,billing_interval,billing_interval_qty,
       category,scheme,cycles,currency,unit_price,min_price,suggested_price,
       trial_interval,trial_interval_qty,tax_code,metadata,
       created_at,
       updated_at
							FROM prices WHERE org_id=@org_id AND id=@id`,
		pgx.NamedArgs{
			"org_id": orgId,
			"id":     id,
		}).Scan(
		&price.OrgId,
		&price.Id,
		&price.VariantId,
		&price.Label,
		&price.BillingInterval,
		&price.BillingIntervalQty,
		&price.Category,
		&price.Scheme,
		&price.Cycles,
		&price.Currency,
		&price.UnitPrice,
		&price.MinPrice,
		&price.SuggestedPrice,
		&price.TrialInterval,
		&price.TrialIntervalQty,
		&price.TaxCode,
		&price.Metadata,
		&price.CreatedAt,
		&price.UpdatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to find Price`, slog.String("err", err.Error()))
		return domain.Price{}, errors.New("not found")
	}
	return price.ToEntity(), nil
}

func (r PriceRepository) FindByVariantId(ctx context.Context, orgId string, variantId string, p domain.Pagination) ([]domain.Price, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var prices = make([]domain.Price, 0)
	var count int
	query := `SELECT org_id, id, variant_id, label, category, scheme, cycles, currency, unit_price, min_price,
                     suggested_price, billing_interval, billing_interval_qty, trial_interval, trial_interval_qty,
                     tax_code, metadata, created_at, updated_at, count(*) OVER()
              FROM prices
              WHERE org_id = @org_id AND variant_id = @variant_id
              ORDER BY
                CASE
                    WHEN @sort_dir = 'asc' THEN
                        CASE @sort_col
                            WHEN 'created_at' THEN created_at
                            ELSE NULL
                            END
                    ELSE
                        NULL
                    END
                    ASC,
                CASE WHEN @sort_dir = 'desc' THEN
                         CASE @sort_col
                             WHEN 'created_at' THEN created_at
                             ELSE NULL
                             END
                     ELSE
                         NULL
                    END
                    DESC
              LIMIT @lim OFFSET @off;`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":     orgId,
		"variant_id": variantId,
		"lim":        p.Limit,
		"off":        p.Offset,
		"sort_col":   p.SortBy,
		"sort_dir":   p.SortDirection,
	})
	if err != nil {
		r.logger.Error(`failed to find Prices by VariantId`, slog.String("err", err.Error()))
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var price models.Price
		err := rows.Scan(
			&price.OrgId,
			&price.Id,
			&price.VariantId,
			&price.Label,
			&price.Category,
			&price.Scheme,
			&price.Cycles,
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
			&count,
		)
		if err != nil {
			r.logger.Error(`failed to scan Price`, slog.String("err", err.Error()))
			return nil, 0, err
		}
		prices = append(prices, price.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, slog.String("err", rows.Err().Error()))
		return nil, 0, rows.Err()
	}

	return prices, count, nil
}

func (r PriceRepository) Update(ctx context.Context, entity domain.Price) (domain.Price, error) {
	tx := r.getTransactionFromContext(ctx)

	metadata, err := json.Marshal(entity.Metadata)
	if err != nil {
		return domain.Price{}, err
	}

	_, err = tx.Exec(ctx, `UPDATE prices
	SET label = @label, category = @category, scheme = @scheme, cycles = @cycles, currency = @currency,
	    unit_price = @unit_price, min_price = @min_price, suggested_price = @suggested_price,
	    billing_interval = @billing_interval, billing_interval_qty = @billing_interval_qty,
	    trial_interval = @trial_interval, trial_interval_qty = @trial_interval_qty,
	    tax_code = @tax_code, metadata = @metadata, updated_at = now()
	WHERE org_id = @org_id AND id = @id`,
		pgx.NamedArgs{
			"label":                entity.Label,
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
			"metadata":             metadata,
			"org_id":               entity.OrgId,
			"id":                   entity.Id,
		})

	if err != nil {
		r.logger.Error(`failed to update Price`, slog.String("err", err.Error()))
		return domain.Price{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r PriceRepository) Delete(ctx context.Context, orgId string, id string) error {
	tx := r.getTransactionFromContext(ctx)

	_, err := tx.Exec(ctx, `DELETE FROM prices WHERE org_id = $1 AND id = $2`, orgId, id)

	if err != nil {
		r.logger.Error(`failed to delete Price`, slog.String("err", err.Error()))
		return err
	}
	return nil
}
