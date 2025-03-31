package postgres

import (
	"context"
	"errors"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type PspRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewPspRepository(database lib.Database, logger logger.Logger) repositories.PspRepository {
	pgDatabase, ok := database.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return PspRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r PspRepository) FindById(ctx context.Context, orgId string, id string) (entities.PaymentServiceProvider, error) {
	tx := r.getTransactionFromContext(ctx)

	var psp entities.PaymentServiceProvider
	query := `SELECT org_id, id, active, created_at, updated_at
              FROM payment_service_providers
              WHERE org_id = $1 AND id = $2`

	err := tx.QueryRow(ctx, query, orgId, id).Scan(
		&psp.OrgId,
		&psp.Id,
		&psp.Active,
		&psp.CreatedAt,
		&psp.UpdatedAt,
	)
	if err != nil {
		r.logger.Error(`failed to find PaymentServiceProvider by Id`, err.Error())
		return entities.PaymentServiceProvider{}, errors.New("not found")
	}

	return psp, nil
}

func (r PspRepository) Create(ctx context.Context, input entities.PaymentServiceProvider) (entities.PaymentServiceProvider, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO payment_service_providers (org_id, id, active, created_at, updated_at)
              VALUES ($1, $2, $3,now(), now())`

	_, err := tx.Exec(ctx, query, input.OrgId, input.Id, input.Active)
	if err != nil {
		r.logger.Error(`failed to create PaymentServiceProvider`, err.Error())
		return entities.PaymentServiceProvider{}, err
	}

	return r.FindById(ctx, input.OrgId, input.Id)
}
