package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"log/slog"
	"payloop/internal/api/dto/request"
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

func NewVariantRepository(primaryDb lib.Database, logger logger.Logger) repositories.VariantRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
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
		r.logger.Error(`failed to find Variant by Id`, slog.String("err", err.Error()))
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
		if price.OrgId.Valid {
			variant.Prices = append(variant.Prices, price)
		}
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, slog.String("err", rows.Err().Error()))
		return entities.Variant{}, rows.Err()
	}

	return variant.ToEntity(), nil
}

func (r VariantRepository) FindByProductId(ctx context.Context, orgId string, productId string, p request.Pagination) ([]entities.Variant, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var variants = make([]entities.Variant, 0)
	var count int
	query := `SELECT v.org_id, v.id, v.product_id, v.name, v.description, v.metadata, v.created_at, v.updated_at, count(*) OVER()
			  FROM variants v
			  WHERE v.org_id = @org_id AND v.product_id = @product_id
			  ORDER BY
				CASE
					WHEN @sort_dir = 'asc' THEN
						CASE @sort_col
							WHEN 'created_at' THEN v.created_at
							ELSE NULL
							END
					ELSE
						NULL
					END
					ASC,
				CASE WHEN @sort_dir = 'desc' THEN
						 CASE @sort_col
							 WHEN 'created_at' THEN v.created_at
							 ELSE NULL
							 END
					 ELSE
						 NULL
					END
					DESC
			  LIMIT @lim OFFSET @off;`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":     orgId,
		"product_id": productId,
		"lim":        p.Limit,
		"off":        p.Offset,
		"sort_col":   p.SortBy,
		"sort_dir":   p.SortDirection,
	})
	if err != nil {
		r.logger.Error(`failed to find Variants by ProductId`, slog.String("err", err.Error()))
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var variant models.Variant
		err := rows.Scan(
			&variant.OrgId,
			&variant.Id,
			&variant.ProductId,
			&variant.Name,
			&variant.Description,
			&variant.Metadata,
			&variant.CreatedAt,
			&variant.UpdatedAt,
			&count,
		)
		if err != nil {
			r.logger.Error(`failed to scan Variant`, slog.String("err", err.Error()))
			return nil, 0, err
		}
		variants = append(variants, variant.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, slog.String("err", rows.Err().Error()))
		return nil, 0, rows.Err()
	}

	return variants, count, nil
}

func (r VariantRepository) Update(ctx context.Context, entity entities.Variant) (entities.Variant, error) {
	tx := r.getTransactionFromContext(ctx)

	_, err := tx.Exec(ctx, `UPDATE variants 
							SET name = $1, description = $2, metadata = $3, updated_at = now()
							WHERE org_id = $4 AND id = $5`,
		entity.Name, entity.Description, entity.Metadata, entity.OrgId, entity.Id)

	if err != nil {
		r.logger.Error(`failed to update Variant`, slog.String("err", err.Error()))
		return entities.Variant{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r VariantRepository) Delete(ctx context.Context, orgId string, id string) error {
	tx := r.getTransactionFromContext(ctx)

	_, err := tx.Exec(ctx, `DELETE FROM variants WHERE org_id = $1 AND id = $2`, orgId, id)

	if err != nil {
		r.logger.Error(`failed to delete Variant`, slog.String("err", err.Error()))
		return err
	}
	return nil
}
