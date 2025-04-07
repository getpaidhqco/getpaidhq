package postgres

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type CartRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewCartRepository(primaryDb lib.Database, logger logger.Logger) repositories.CartRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return CartRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r CartRepository) FindById(ctx context.Context, orgId string, id string) (entities.Cart, error) {
	tx := r.getTransactionFromContext(ctx)

	var cart entities.Cart
	err := tx.QueryRow(ctx, `SELECT org_id,id,data FROM carts WHERE org_id=@org_id AND id=@id`, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	}).Scan(&cart.OrgId,
		&cart.Id,
		&cart.Data)

	if err != nil {
		r.logger.Error(`failed to find Cart`, "orgId", orgId, "id", id, "err", err.Error())
		return entities.Cart{}, err
	}
	return cart, nil
}

func (r CartRepository) Create(ctx context.Context, input entities.Cart) (entities.Cart, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO carts (org_id,id,data,metadata,created_at,updated_at) 
			  VALUES (@org_id,@id,@data,@metadata,NOW(), NOW())`

	metaJson, _ := json.Marshal(input.Metadata)

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":   input.OrgId,
		"id":       input.Id,
		"data":     input.Data,
		"metadata": metaJson,
	})

	if err != nil {
		r.logger.Error(`failed to insert Cart`, err)
		return entities.Cart{}, err
	}

	return entities.Cart{
		OrgId:  input.OrgId,
		Id:     input.Id,
		Data:   input.Data,
		Status: "",
		Total:  0,
	}, nil
}

func (r CartRepository) Update(ctx context.Context, input entities.Cart) (entities.Cart, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `UPDATE carts SET data=@data, metadata=@metadata, updated_at=NOW() 
             WHERE org_id=@org_id AND id=@id`

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id": input.OrgId,
		"id":     input.Id,
		"data":   input.Data,
	})
	if err != nil {
		r.logger.Error(`failed to update Cart`, err)
		return entities.Cart{}, err
	}

	// TODO read new values
	return input, nil
}
