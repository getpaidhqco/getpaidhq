package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"
)

type ProductRepository struct {
	*PgDatabase
	logger    logger.Logger
	priceRepo repositories.PriceRepository
}

func NewProductRepository(
	primaryDb lib.Database,
	logger logger.Logger,
	priceRepo repositories.PriceRepository,
) repositories.ProductRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return ProductRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
		priceRepo:  priceRepo,
	}
}

func (r ProductRepository) Create(ctx context.Context, product entities.Product) (entities.Product, error) {
	tx := r.getTransactionFromContext(ctx)

	_, err := tx.Exec(ctx, `INSERT INTO products (org_id, id, name, description, metadata, created_at, updated_at)
								VALUES ($1, $2, $3, $4, $5, now(), now())`,
		product.OrgId, product.Id, product.Name, product.Description, product.Metadata)

	if err != nil {
		r.logger.Error(`failed to create Product`, err.Error())
		return entities.Product{}, err
	}
	return r.FindById(ctx, product.OrgId, product.Id)
}

func (r ProductRepository) FindById(ctx context.Context, orgId string, id string) (entities.Product, error) {
	tx := r.getTransactionFromContext(ctx)

	var product models.Product
	query := `SELECT p.org_id, p.id, p.name, p.description, p.metadata, p.created_at, p.updated_at,
	                 v.org_id, v.id, v.product_id, v.name, v.description, v.metadata, v.created_at, v.updated_at,
	                 pr.org_id, pr.id, pr.label, pr.variant_id, pr.category, pr.scheme, pr.cycles, pr.currency, pr.unit_price, pr.min_price, 
                     pr.suggested_price, pr.billing_interval, pr.billing_interval_qty, pr.trial_interval, pr.trial_interval_qty,
                     pr.tax_code, 
                     pr.meter_id, pr.has_usage, pr.percentage_rate, pr.fixed_fee, pr.overage_unit_price, pr.included_usage, pr.usage_limit,
                     pr.metadata, pr.created_at, pr.updated_at
              FROM products p
              LEFT JOIN variants v ON p.org_id = v.org_id AND p.id = v.product_id
              LEFT JOIN prices pr ON v.org_id = pr.org_id AND v.id = pr.variant_id
              WHERE p.org_id = @org_id AND p.id = @id`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	})
	if err != nil {
		r.logger.Error(`failed to find Product by Id`, err.Error())
		return entities.Product{}, errors.New("not found")
	}
	defer rows.Close()

	var variantsMap = make(map[string]*models.Variant)
	for rows.Next() {
		var variant models.Variant
		var price models.Price
		err := rows.Scan(
			&product.OrgId,
			&product.Id,
			&product.Name,
			&product.Description,
			&product.Metadata,
			&product.CreatedAt,
			&product.UpdatedAt,
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
			&price.Label,
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
			&price.MeterId,
			&price.HasUsage,
			&price.PercentageRate,
			&price.FixedFee,
			&price.OverageUnitPrice,
			&price.IncludedUsage,
			&price.UsageLimit,
			&price.Metadata,
			&price.CreatedAt,
			&price.UpdatedAt,
		)
		if err != nil {
			r.logger.Error(`failed to scan Product, Variant and Price`, err.Error())
			return entities.Product{}, err
		}
		if variant.Id.Valid {
			if v, ok := variantsMap[variant.Id.String]; ok {
				if price.OrgId.Valid {
					v.Prices = append(v.Prices, price)
				}
			} else {
				if price.OrgId.Valid {
					variant.Prices = append(variant.Prices, price)
				}
				variantsMap[variant.Id.String] = &variant
			}
		}
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return entities.Product{}, rows.Err()
	}

	// Create a map to store price tiers for each price
	priceTiers := make(map[string][]entities.PriceTier)

	// Collect all prices from all variants
	for _, variant := range variantsMap {
		for _, price := range variant.Prices {
			if price.OrgId.Valid && price.Id.Valid {
				// Load price tiers for this price
				tiers, err := r.priceRepo.GetPriceTiers(ctx, price.OrgId.String, price.Id.String)
				if err != nil {
					r.logger.Error(`failed to load price tiers`, err.Error())
					// Continue even if we can't load tiers for a specific price
					continue
				}
				// Store the tiers in the map using the price ID as the key
				priceTiers[price.Id.String] = tiers
			}
		}
		product.Variants = append(product.Variants, *variant)
	}

	// Convert the product to an entity
	productEntity := product.ToEntity()

	// Add price tiers to each price in the product entity
	for i, variant := range productEntity.Variants {
		for j, price := range variant.Prices {
			if tiers, ok := priceTiers[price.Id]; ok {
				productEntity.Variants[i].Prices[j].Tiers = tiers
			}
		}
	}

	return productEntity, nil
}

func (r ProductRepository) Find(ctx context.Context, orgId string, p request.Pagination) ([]entities.Product, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var products = make([]entities.Product, 0)
	var count int
	query := `SELECT org_id, id, name, description, metadata, created_at, updated_at, count(*) OVER()
			  FROM products 
			  WHERE org_id = @org_id
			ORDER BY
				-- Simplified to NULL if not sorting in ascending order.
				CASE
					WHEN @sort_dir = 'asc' THEN
						CASE @sort_col
							-- Check for each possible value of sort_col.
							WHEN 'created_at' THEN created_at
							--- etc.
							ELSE NULL
							END
					ELSE
						NULL
					END
					ASC ,

				-- Same as before, but for sort_dir = 'desc'
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
		"org_id":   orgId,
		"lim":      p.Limit,
		"off":      p.Offset,
		"sort_col": p.SortBy,
		"sort_dir": p.SortDirection,
	})
	if err != nil {
		r.logger.Error(`failed to find Products`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var product models.Product
		err := rows.Scan(
			&product.OrgId,
			&product.Id,
			&product.Name,
			&product.Description,
			&product.Metadata,
			&product.CreatedAt,
			&product.UpdatedAt,
			&count,
		)
		if err != nil {
			r.logger.Error(`failed to scan Product`, err.Error())
			return nil, 0, err
		}
		products = append(products, product.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return products, count, nil
}

func (r ProductRepository) CreateVariant(ctx context.Context, entity entities.Variant) (entities.Variant, error) {
	tx := r.getTransactionFromContext(ctx)

	var variant models.Variant

	query := `INSERT INTO variants (org_id, id, product_id,name,
                      description,metadata,
                    created_at, updated_at)
        VALUES (@org_id, @id,@product_id,@name,@description,@metadata,
                NOW(), NOW())
       `

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":      entity.OrgId,
		"id":          entity.Id,
		"product_id":  entity.ProductId,
		"name":        entity.Name,
		"description": entity.Description,
		"metadata":    entity.Metadata,
	})

	if err != nil {
		r.logger.Error(`failed to create Variant`, err.Error())
		return entities.Variant{}, err
	}
	return variant.ToEntity(), nil
}

func (r ProductRepository) Update(ctx context.Context, product entities.Product) (entities.Product, error) {
	tx := r.getTransactionFromContext(ctx)

	_, err := tx.Exec(ctx, `UPDATE products 
							SET name = $1, description = $2, metadata = $3, updated_at = now()
							WHERE org_id = $4 AND id = $5`,
		product.Name, product.Description, product.Metadata, product.OrgId, product.Id)

	if err != nil {
		r.logger.Error(`failed to update Product`, err.Error())
		return entities.Product{}, err
	}
	return r.FindById(ctx, product.OrgId, product.Id)
}

func (r ProductRepository) Delete(ctx context.Context, orgId string, id string) error {
	tx := r.getTransactionFromContext(ctx)

	_, err := tx.Exec(ctx, `DELETE FROM products WHERE org_id = $1 AND id = $2`, orgId, id)

	if err != nil {
		r.logger.Error(`failed to delete Product`, err.Error())
		return err
	}
	return nil
}
