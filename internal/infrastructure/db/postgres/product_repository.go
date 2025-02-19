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
	"payloop/internal/lib"
)

type ProductRepository struct {
	*lib.PgDatabase
	logger logger.Logger
}

func NewProductRepository(database lib.Database, logger logger.Logger) repositories.ProductRepository {
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return ProductRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r ProductRepository) Create(ctx context.Context, product entities.Product) (entities.Product, error) {
	err := r.Pool.QueryRow(ctx, `INSERT INTO products (org_id, id, name, description, metadata)
								VALUES ($1, $2, $3, $4, $5)
								RETURNING org_id, id, name, description, metadata`,
		product.OrgId, product.Id, product.Name, product.Description, product.Metadata).Scan(
		&product.OrgId,
		&product.Id,
		&product.Name,
		&product.Description,
		&product.Metadata,
	)

	if err != nil {
		r.logger.Error(`failed to create Product`, err.Error())
		return entities.Product{}, err
	}
	return product, nil
}

func (r ProductRepository) FindById(ctx context.Context, orgId string, id string) (entities.Product, error) {
	var product entities.Product
	err := r.Pool.QueryRow(ctx, `SELECT org_id,id,name,description,metadata 
							FROM products WHERE org_id=@org_id AND id=@id`,
		pgx.NamedArgs{
			"org_id": orgId,
			"id":     id,
		}).Scan(
		&product.OrgId,
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

func (r ProductRepository) Find(ctx context.Context, orgId string, p request.Pagination) ([]entities.Product, int, error) {
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
	rows, err := r.Pool.Query(ctx, query, pgx.NamedArgs{
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
