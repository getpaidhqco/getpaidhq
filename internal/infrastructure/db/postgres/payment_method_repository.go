package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"

	"payloop/internal/lib"
)

type PaymentMethodRepository struct {
	*lib.PgDatabase
	logger lib.Logger
}

func NewPaymentMethodRepository(database lib.Database, logger lib.Logger) repositories.PaymentMethodRepository {
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return PaymentMethodRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r PaymentMethodRepository) FindById(ctx context.Context, orgId string, id string) (entities.PaymentMethod, error) {
	var pm entities.PaymentMethod
	err := r.Pool.QueryRow(ctx, `SELECT org_id,id,name,customer_id,is_default,details,type FROM payment_methods WHERE org_id=@org_id AND id=@id`, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	}).Scan(&pm.OrgId,
		&pm.Id,
		&pm.Name,
		&pm.CustomerId,
		&pm.IsDefault,
		&pm.Details,
		&pm.Type,
	)

	if err != nil {
		r.logger.Error(`failed to find payment method`, "orgId", orgId, "id", id, "err", err.Error())
		return entities.PaymentMethod{}, err
	}
	return pm, nil
}

func (r PaymentMethodRepository) Create(ctx context.Context, entity entities.PaymentMethod) (entities.PaymentMethod, error) {
	var newEntity entities.PaymentMethod
	query := `INSERT INTO payment_methods (org_id, id, name, customer_id,is_default,billing_address,details,type, created_at, updated_at) 
			  VALUES (@org_id,@id, @name, @customer_id,@is_default,@billing_address,@details,@type,NOW(), NOW())
			  RETURNING (org_id,id,name,customer_id,is_default,billing_address,type,details,created_at,updated_at)`

	err := r.Pool.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id":          entity.OrgId,
		"id":              entity.Id,
		"name":            entity.Name,
		"customer_id":     entity.CustomerId,
		"is_default":      entity.IsDefault,
		"billing_address": entity.BillingAddress,
		"details":         entity.Details,
		"type":            entity.Type,
	}).Scan(&newEntity)

	if err != nil {
		r.logger.Error(`failed to insert PaymentMethod`, err)
		return entities.PaymentMethod{}, err
	}

	return newEntity, nil
}
