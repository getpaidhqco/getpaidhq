package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
	"time"
)

type PaymentMethodRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewPaymentMethodRepository(database lib.Database, logger logger.Logger) repositories.PaymentMethodRepository {
	pgDatabase, ok := database.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return PaymentMethodRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

// Implementing the missing methods for PaymentMethodRepository

func (r PaymentMethodRepository) FindById(ctx context.Context, orgId string, id string) (entities.PaymentMethod, error) {
	tx := r.getTransactionFromContext(ctx)

	var pm entities.PaymentMethod
	err := tx.QueryRow(ctx, `SELECT org_id, id, token, psp, name, customer_id, is_default, details, type FROM payment_methods WHERE org_id=@org_id AND id=@id`, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	}).Scan(&pm.OrgId,
		&pm.Id,
		&pm.Token,
		&pm.Psp,
		&pm.Name,
		&pm.CustomerId,
		&pm.IsDefault,
		&pm.Details,
		&pm.Type,
	)

	if err != nil {
		r.logger.Error(`failed to find payment method`, "orgId", orgId, "id", id, "err", err.Error())
		return entities.PaymentMethod{}, mapError(err)
	}
	return pm, nil
}

func (r PaymentMethodRepository) Create(ctx context.Context, entity entities.PaymentMethod) (entities.PaymentMethod, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO payment_methods (org_id, id, token, psp, name, customer_id, is_default, details, type, created_at, updated_at)
			  VALUES (@org_id, @id, @token, @psp, @name, @customer_id, @is_default, @details, @type, now(), now())
			  ON CONFLICT (org_id, customer_id, token) DO UPDATE SET
				  token = EXCLUDED.token,
				  psp = EXCLUDED.psp,
				  name = EXCLUDED.name,
				  customer_id = EXCLUDED.customer_id,
				  is_default = EXCLUDED.is_default,
				  details = EXCLUDED.details,
				  type = EXCLUDED.type,
				  updated_at = now() 
			  RETURNING org_id, id, token, psp, name, customer_id, is_default, details, type, created_at, updated_at`

	var newEntity entities.PaymentMethod
	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id":      entity.OrgId,
		"id":          entity.Id,
		"name":        entity.Name,
		"psp":         entity.Psp,
		"token":       entity.Token,
		"customer_id": entity.CustomerId,
		"is_default":  entity.IsDefault,
		"details":     entity.Details,
		"type":        entity.Type,
	}).Scan(
		&newEntity.OrgId,
		&newEntity.Id,
		&newEntity.Token,
		&newEntity.Psp,
		&newEntity.Name,
		&newEntity.CustomerId,
		&newEntity.IsDefault,
		&newEntity.Details,
		&newEntity.Type,
		&newEntity.CreatedAt,
		&newEntity.UpdatedAt,
	)

	if err != nil {
		return entities.PaymentMethod{}, mapError(err)
	}

	return newEntity, nil
}

func (r PaymentMethodRepository) Update(ctx context.Context, entity entities.PaymentMethod) (entities.PaymentMethod, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `UPDATE payment_methods SET token=@Token, 
                     psp=@Psp, 
                     name=@Name,
                     customer_id=@CustomerId, 
                     is_default=@IsDefault, 
                     details=@Details,
                     type=@Type,
                     updated_at=now()
              WHERE org_id=@OrgId AND id=@Id
              RETURNING org_id, id, token, psp, name, customer_id, is_default, details, type, created_at, updated_at`

	var updatedEntity entities.PaymentMethod
	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"OrgId":      entity.OrgId,
		"Id":         entity.Id,
		"Token":      entity.Token,
		"Psp":        entity.Psp,
		"Name":       entity.Name,
		"CustomerId": entity.CustomerId,
		"IsDefault":  entity.IsDefault,
		"Details":    entity.Details,
		"Type":       entity.Type,
	}).Scan(
		&updatedEntity.OrgId,
		&updatedEntity.Id,
		&updatedEntity.Token,
		&updatedEntity.Psp,
		&updatedEntity.Name,
		&updatedEntity.CustomerId,
		&updatedEntity.IsDefault,
		&updatedEntity.Details,
		&updatedEntity.Type,
		&updatedEntity.CreatedAt,
		&updatedEntity.UpdatedAt,
	)

	if err != nil {
		return entities.PaymentMethod{}, mapError(err)
	}

	return updatedEntity, nil
}

// FindExpiringPaymentMethods returns a list of payment methods that are expiring before the given expiry time
// It's not Organization specific and should not be called from anywhere other than the dunning service
func (r PaymentMethodRepository) FindExpiringPaymentMethods(ctx context.Context, expiry time.Time) ([]entities.PaymentMethod, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT org_id, id, token, psp, name, customer_id, is_default, details, type, created_at, updated_at
			  FROM payment_methods
			  WHERE expire_at <= @expiry`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"expiry": expiry,
	})
	if err != nil {
		r.logger.Error(`failed to find expiring payment methods`, "expiry", expiry, "err", err.Error())
		return nil, mapError(err)
	}
	defer rows.Close()

	var paymentMethods []entities.PaymentMethod
	for rows.Next() {
		var pm entities.PaymentMethod
		err := rows.Scan(
			&pm.OrgId,
			&pm.Id,
			&pm.Token,
			&pm.Psp,
			&pm.Name,
			&pm.CustomerId,
			&pm.IsDefault,
			&pm.Details,
			&pm.Type,
			&pm.CreatedAt,
			&pm.UpdatedAt,
		)
		if err != nil {
			r.logger.Error(`failed to scan payment method`, "err", err.Error())
			return nil, mapError(err)
		}
		paymentMethods = append(paymentMethods, pm)
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, "err", rows.Err().Error())
		return nil, mapError(rows.Err())
	}

	return paymentMethods, nil
}
