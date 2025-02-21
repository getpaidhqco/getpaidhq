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
	logger logger.Logger
}

func NewProductRepository(database lib.Database, logger logger.Logger) repositories.ProductRepository {
	pgDatabase, ok := database.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return ProductRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
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
	err := tx.QueryRow(ctx, `SELECT org_id,id,name,description,metadata,created_at,updated_at
							FROM products WHERE org_id=@org_id AND id=@id`,
		pgx.NamedArgs{
			"org_id": orgId,
			"id":     id,
		}).Scan(
		&product.OrgId,
		&product.Id,
		&product.Name,
		&product.Description,
		&product.Metadata,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to find Product`, err.Error())
		return entities.Product{}, errors.New("not found")
	}
	return product.ToEntity(), nil
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
		var product entities.Product
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
		products = append(products, product)
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return products, count, nil
}

func (r ProductRepository) CreatePrice(ctx context.Context, entity entities.Price) (entities.Price, error) {
	tx := r.getTransactionFromContext(ctx)

	var price models.Price

	r.logger.Debug("BillingInterval value: ", entity.BillingInterval)
	query := `INSERT INTO prices (org_id, id, variant_id, category, scheme, cycles, currency, 
                    unit_price, min_price, suggested_price, billing_interval, billing_interval_qty, 
                    trial_interval, trial_interval_qty, tax_code, metadata,
                    created_at, updated_at)
        VALUES (@org_id, @id, @variant_id, @category, @scheme, @cycles, @currency, 
                @unit_price, @min_price, @suggested_price, @billing_interval, @billing_interval_qty, 
                @trial_interval, @trial_interval_qty, @tax_code, @metadata,
                NOW(), NOW())
		RETURNING org_id, id, variant_id, category, scheme, cycles, currency, 
                    unit_price, min_price, suggested_price, billing_interval, billing_interval_qty, 
                    trial_interval, trial_interval_qty, tax_code, metadata,
                    created_at, updated_at
       `

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
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
	}).Scan(

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
		r.logger.Error(`failed to create Price`, err.Error())
		return entities.Price{}, err
	}
	return price.ToEntity(), nil
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
