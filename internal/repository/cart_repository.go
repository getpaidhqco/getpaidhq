package repository

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/domain/carts"
	"payloop/internal/domain/entities"
	"payloop/internal/lib"
)

type CartRepositoryIf interface {
	FindByID(ctx context.Context, id uint) (*entities.Cart, error)
	FindAll(ctx context.Context) ([]*entities.Cart, error)
	Create(ctx context.Context, order entities.Cart) error
	Update(ctx context.Context, order entities.Cart) error
	Delete(ctx context.Context, id uint) error
}

type CartRepository struct {
	*lib.PgDatabase
	logger lib.Logger
}

func NewCartRepository(database lib.Database, logger lib.Logger) CartRepository {
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return CartRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

// WithTrx enables repository with transaction
func (r *CartRepository) WithTrx(trxHandle interface{}) *CartRepository {
	if trxHandle == nil {
		r.logger.Warn("Transaction Database not found in gin context. ")
		return r
	}
	r.PgDatabase.Tx = trxHandle.(pgx.Tx)
	return r
}

func (r *CartRepository) FindByID(ctx context.Context, acctId string, id string) (entities.Cart, error) {
	var cart entities.Cart
	err := r.Pool.QueryRow(ctx, `SELECT org_id,id,data FROM carts WHERE org_id=@org_id AND id=@id`, pgx.NamedArgs{
		"org_id": acctId,
		"id":     id,
	}).Scan(&cart.OrgId,
		&cart.Id,
		&cart.Data)

	if err != nil {
		r.logger.Error(`failed to find Cart`, "acctId", acctId, "id", id, "err", err.Error())
		return entities.Cart{}, err
	}
	return cart, nil
}

func (r *CartRepository) Create(ctx context.Context, input carts.CreateCartInput) (entities.Cart, error) {
	cartId := lib.GenerateId("cart")

	query := `INSERT INTO carts (org_id,id,data,metadata,created_at,updated_at) 
			  VALUES (@org_id,@id,@data,@metadata,NOW(), NOW())`

	metaJson, _ := json.Marshal(input.Metadata)

	_, err := r.Pool.Exec(ctx, query, pgx.NamedArgs{
		"org_id":   input.OrgId,
		"id":       cartId,
		"data":     input.Cart,
		"metadata": metaJson,
	})

	if err != nil {
		r.logger.Error(`failed to insert Cart`, err)
		return entities.Cart{}, err
	}

	return entities.Cart{
		Id:     cartId,
		Data:   input.Cart,
		Status: "",
		Total:  0,
	}, nil
}

func (r *CartRepository) Update(ctx context.Context, input entities.Cart) (entities.Cart, error) {

	query := `UPDATE carts SET data=@data, metadata=@metadata, updated_at=NOW() 
             WHERE org_id=@org_id AND id=@id`

	_, err := r.Pool.Exec(ctx, query, pgx.NamedArgs{
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
