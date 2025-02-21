package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"log/slog"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"
)

type VariantRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewVariantRepository(database lib.Database, logger logger.Logger) repositories.VariantRepository {
	pgDatabase, ok := database.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return VariantRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r VariantRepository) Create(ctx context.Context, variant entities.Variant) (entities.Variant, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO variants (org_id, id, product_id, name, description, metadata, created_at, updated_at)
			  VALUES (@org_id, @id, @product_id, @name, @description, @metadata, NOW(), NOW())`

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":      variant.OrgId,
		"id":          variant.Id,
		"product_id":  variant.ProductId,
		"name":        variant.Name,
		"description": variant.Description,
		"metadata":    variant.Metadata,
	})

	if err != nil {
		r.logger.Error(`failed to create Variant`, slog.String("err", err.Error()))
		return entities.Variant{}, err
	}
	return r.FindById(ctx, variant.OrgId, variant.Id)
}

func (r VariantRepository) FindById(ctx context.Context, orgId string, id string) (entities.Variant, error) {
	tx := r.getTransactionFromContext(ctx)

	var variant models.Variant
	query := `SELECT v.org_id, v.id, v.product_id, v.name, v.description, v.metadata, v.created_at, v.updated_at,
	                 p.org_id, p.id, p.variant_id, p.category, p.scheme, p.cycles, p.currency, p.unit_price, p.min_price, 
                     p.suggested_price, p.billing_interval, p.billing_interval_qty, p.trial_interval, p.trial_interval_qty,
                     p.tax_code, p.metadata, p.created_at, p.updated_at
              FROM variants v
              LEFT JOIN prices p ON v.org_id = p.org_id AND v.id = p.variant_id
              WHERE v.org_id = @org_id AND v.id = @id`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	})
	if err != nil {
		r.logger.Error(`failed to find Variant by ID`, slog.String("err", err.Error()))
		return entities.Variant{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var price models.Price
		err := rows.Scan(
			&variant.OrgId,
			&variant.Id,
			&variant.ProductId,
			&variant.Name,
			&variant.Description,
			&variant.Metadata,
			&variant.CreatedAt,
			&variant.UpdatedAt,
			&price.OrgId,
			&price.Id,
			&price.VariantId,
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
		)
		if err != nil {
			r.logger.Error(`failed to scan Variant and Price`, slog.String("err", err.Error()))
			return entities.Variant{}, err
		}
		variant.Prices = append(variant.Prices, price)
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, slog.String("err", rows.Err().Error()))
		return entities.Variant{}, rows.Err()
	}

	return variant.ToEntity(), nil
}
