package postgres

import (
	"context"
	"errors"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type GatewayRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewGatewayRepository(primaryDb lib.Database, logger logger.Logger) repositories.PspRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return GatewayRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r GatewayRepository) FindById(ctx context.Context, orgId string, id string) (entities.Gateway, error) {
	tx := r.getTransactionFromContext(ctx)

	var psp entities.Gateway
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
		return entities.Gateway{}, errors.New("not found")
	}

	return psp, nil
}

func (r GatewayRepository) Create(ctx context.Context, input entities.Gateway) (entities.Gateway, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO gateways (org_id, id, active, created_at, updated_at)
              VALUES ($1, $2, $3,now(), now())`

	_, err := tx.Exec(ctx, query, input.OrgId, input.Id, input.Active)
	if err != nil {
		r.logger.Error(`failed to create Gateway`, err.Error())
		return entities.Gateway{}, err
	}

	return r.FindById(ctx, input.OrgId, input.Id)
}

func (r GatewayRepository) Update(ctx context.Context, input entities.Gateway) (entities.Gateway, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `UPDATE gateways 
              SET name = $1, active = $2, updated_at = now()
              WHERE org_id = $3 AND id = $4`

	_, err := tx.Exec(ctx, query, input.Name, input.Active, input.OrgId, input.Id)
	if err != nil {
		r.logger.Error(`failed to update Gateway`, err.Error())
		return entities.Gateway{}, err
	}

	return r.FindById(ctx, input.OrgId, input.Id)
}

func (r GatewayRepository) Delete(ctx context.Context, orgId string, id string) error {
	tx := r.getTransactionFromContext(ctx)

	query := `DELETE FROM gateways WHERE org_id = $1 AND id = $2`

	_, err := tx.Exec(ctx, query, orgId, id)
	if err != nil {
		r.logger.Error(`failed to delete Gateway`, err.Error())
		return err
	}

	return nil
}

func (r GatewayRepository) FindAll(ctx context.Context, orgId string) ([]entities.Gateway, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT org_id, id, name, psp_id, active, created_at, updated_at
              FROM gateways
              WHERE org_id = $1`

	rows, err := tx.Query(ctx, query, orgId)
	if err != nil {
		r.logger.Error(`failed to find all Gateways`, err.Error())
		return nil, err
	}
	defer rows.Close()

	var gateways []entities.Gateway
	for rows.Next() {
		var gateway entities.Gateway
		err := rows.Scan(
			&gateway.OrgId,
			&gateway.Id,
			&gateway.Name,
			&gateway.PspId,
			&gateway.Active,
			&gateway.CreatedAt,
			&gateway.UpdatedAt,
		)
		if err != nil {
			r.logger.Error(`failed to scan Gateway`, err.Error())
			return nil, err
		}
		gateways = append(gateways, gateway)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error(`error iterating Gateway rows`, err.Error())
		return nil, err
	}

	return gateways, nil
}
