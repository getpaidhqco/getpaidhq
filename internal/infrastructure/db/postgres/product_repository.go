package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type ProductRepository struct {
	*lib.PgDatabase
	logger lib.Logger
}

func NewProductRepository(database lib.Database, logger lib.Logger) repositories.ProductRepository {
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return ProductRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
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
