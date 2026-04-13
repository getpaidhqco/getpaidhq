package postgres

import (
	"context"
	"errors"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/lib"
)

type GatewayRepository struct {
	*PgDatabase
	logger port.Logger
}

func NewGatewayRepository(primaryDb lib.Database, logger port.Logger) port.PspRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return GatewayRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r GatewayRepository) FindById(ctx context.Context, orgId string, id string) (domain.PspConfig, error) {
	tx := r.getTransactionFromContext(ctx)

	var psp domain.PspConfig
	query := `SELECT org_id, id, name, psp_id, active, created_at, updated_at
              FROM gateways
              WHERE org_id = $1 AND id = $2`

	err := tx.QueryRow(ctx, query, orgId, id).Scan(
		&psp.OrgId,
		&psp.Id,
		&psp.Name,
		&psp.PspId,
		&psp.Active,
		&psp.CreatedAt,
		&psp.UpdatedAt,
	)
	if err != nil {
		r.logger.Errorf(`failed to find Gateway by Id %s`, err.Error())
		return domain.PspConfig{}, errors.New("not found")
	}

	return psp, nil
}

func (r GatewayRepository) Create(ctx context.Context, input domain.PspConfig) (domain.PspConfig, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO gateways (org_id, id, active, created_at, updated_at)
              VALUES ($1, $2, $3,now(), now())`

	_, err := tx.Exec(ctx, query, input.OrgId, input.Id, input.Active)
	if err != nil {
		r.logger.Error(`failed to create Gateway`, err.Error())
		return domain.PspConfig{}, err
	}

	return r.FindById(ctx, input.OrgId, input.Id)
}
