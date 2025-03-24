package postgres

import (
	"context"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type ApiKeyRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewApiKeyRepository(database lib.Database, logger logger.Logger) repositories.ApiKeyRepository {
	pgDatabase, ok := database.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return ApiKeyRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r ApiKeyRepository) FindById(ctx context.Context, orgId string, id string) (entities.ApiKey, error) {
	tx := r.getTransactionFromContext(ctx)

	var apiKey entities.ApiKey
	err := tx.QueryRow(ctx, `SELECT org_id, id, key, created_at, updated_at FROM api_keys 
                                               WHERE org_id=$1 AND id=$2`, orgId, id).
		Scan(
			&apiKey.OrgId,
			&apiKey.Id,
			&apiKey.Key,
			&apiKey.CreatedAt,
			&apiKey.UpdatedAt,
		)

	if err != nil {
		r.logger.Error(`failed to find ApiKey`, "id", id, "err", err.Error())
		return entities.ApiKey{}, err
	}
	return apiKey, nil
}
func (r ApiKeyRepository) FindByKey(ctx context.Context, key string) (entities.ApiKey, error) {
	tx := r.getTransactionFromContext(ctx)

	var apiKey entities.ApiKey
	err := tx.QueryRow(ctx, `SELECT org_id, id, key, created_at, updated_at FROM api_keys WHERE key=$1`, key).
		Scan(
			&apiKey.OrgId,
			&apiKey.Id,
			&apiKey.Key,
			&apiKey.CreatedAt,
			&apiKey.UpdatedAt,
		)

	if err != nil {
		r.logger.Error(`failed to find ApiKey`, "err", err.Error())
		return entities.ApiKey{}, err
	}
	return apiKey, nil
}

func (r ApiKeyRepository) Create(ctx context.Context, entity entities.ApiKey) (entities.ApiKey, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO api_keys (id, key, created_at, updated_at) VALUES ($1, $2, NOW(), NOW())`

	_, err := tx.Exec(ctx, query, entity.Id, entity.Key)
	if err != nil {
		r.logger.Error(`failed to insert ApiKey`, err)
		return entities.ApiKey{}, err
	}

	return entity, nil
}

func (r ApiKeyRepository) Update(ctx context.Context, entity entities.ApiKey) (entities.ApiKey, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `UPDATE api_keys SET key=$1, updated_at=NOW() WHERE id=$2`

	_, err := tx.Exec(ctx, query, entity.Key, entity.Id)
	if err != nil {
		r.logger.Error(`failed to update ApiKey`, err)
		return entities.ApiKey{}, err
	}

	return entity, nil
}

func (r ApiKeyRepository) Delete(ctx context.Context, orgId string, id string) error {
	tx := r.getTransactionFromContext(ctx)

	query := `DELETE FROM api_keys WHERE id=$1`

	_, err := tx.Exec(ctx, query, id)
	if err != nil {
		r.logger.Error(`failed to delete ApiKey`, err)
		return err
	}

	return nil
}
