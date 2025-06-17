package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"
)

type SettingRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewSettingRepository(primaryDb lib.Database, logger logger.Logger) repositories.SettingRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return SettingRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r SettingRepository) FindById(ctx context.Context, orgId string, parentId string, id string) (entities.Setting, error) {
	tx := r.getTransactionFromContext(ctx)

	var setting models.Setting
	query := `SELECT org_id, parent_id, id, value_type, value, created_at, updated_at
		      FROM settings
		      WHERE org_id = $1 AND parent_id = $2 AND id = $3`

	err := tx.QueryRow(ctx, query, orgId, parentId, id).
		Scan(
			&setting.OrgId,
			&setting.ParentId,
			&setting.Id,
			&setting.ValueType,
			&setting.Value,
			&setting.CreatedAt,
			&setting.UpdatedAt,
		)
	if err != nil {
		r.logger.Error(`failed to find Setting`, err.Error())
		return entities.Setting{}, errors.New("not found")
	}

	return setting.ToEntity(), nil
}

func (r SettingRepository) FindAll(ctx context.Context, orgId string, parentId string) ([]entities.Setting, error) {
	tx := r.getTransactionFromContext(ctx)

	var settings []entities.Setting
	query := `SELECT org_id, parent_id, id, value_type, value, created_at, updated_at
		      FROM settings
		      WHERE org_id = $1 AND parent_id = $2`

	rows, err := tx.Query(ctx, query, orgId, parentId)
	if err != nil {
		r.logger.Error(`failed to find Settings`, err.Error())
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var setting models.Setting
		err := rows.Scan(
			&setting.OrgId,
			&setting.ParentId,
			&setting.Id,
			&setting.ValueType,
			&setting.Value,
			&setting.CreatedAt,
			&setting.UpdatedAt,
		)
		if err != nil {
			r.logger.Error(`failed to scan Setting`, err.Error())
			return nil, err
		}
		settings = append(settings, setting.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, rows.Err()
	}

	return settings, nil
}

func (r SettingRepository) Create(ctx context.Context, entity entities.Setting) (entities.Setting, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO settings (org_id, parent_id, id, value_type, value, created_at, updated_at)
          VALUES (@org_id, @parent_id, @id, @value_type, @value, @created_at, @updated_at)
          RETURNING org_id, parent_id, id, value_type, value, created_at, updated_at`

	var setting models.Setting

	err := tx.QueryRow(ctx, query, settingEntityToNamedArgs(entity)).
		Scan(
			&setting.OrgId,
			&setting.ParentId,
			&setting.Id,
			&setting.ValueType,
			&setting.Value,
			&setting.CreatedAt,
			&setting.UpdatedAt,
		)

	if err != nil {
		r.logger.Error(`failed to create Setting`, "err", err.Error())
		return entities.Setting{}, err
	}

	return setting.ToEntity(), nil
}

func (r SettingRepository) Update(ctx context.Context, entity entities.Setting) (entities.Setting, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `UPDATE settings
	          SET value_type = @value_type, value = @value, updated_at = @updated_at
	          WHERE org_id = @org_id AND parent_id = @parent_id AND id = @id
	          RETURNING org_id, parent_id, id, value_type, value, created_at, updated_at`

	var setting models.Setting

	err := tx.QueryRow(ctx, query, settingEntityToNamedArgs(entity)).
		Scan(
			&setting.OrgId,
			&setting.ParentId,
			&setting.Id,
			&setting.ValueType,
			&setting.Value,
			&setting.CreatedAt,
			&setting.UpdatedAt,
		)

	if err != nil {
		r.logger.Error(`failed to update Setting`, "err", err.Error())
		return entities.Setting{}, err
	}

	return setting.ToEntity(), nil
}

func (r SettingRepository) Delete(ctx context.Context, orgId string, parentId string, id string) error {
	tx := r.getTransactionFromContext(ctx)

	query := `DELETE FROM settings
	          WHERE org_id = $1 AND parent_id = $2 AND id = $3`

	_, err := tx.Exec(ctx, query, orgId, parentId, id)
	if err != nil {
		r.logger.Error(`failed to delete Setting`, "err", err.Error())
		return err
	}

	return nil
}

func settingEntityToNamedArgs(entity entities.Setting) pgx.NamedArgs {
	// Parse the value string as JSON
	var valueJson json.RawMessage
	if entity.Value != "" {
		valueJson = json.RawMessage(entity.Value)
	}

	return pgx.NamedArgs{
		"org_id":     entity.OrgId,
		"parent_id":  entity.ParentId,
		"id":         entity.Id,
		"value_type": entity.Type,
		"value":      valueJson,
		"created_at": entity.CreatedAt,
		"updated_at": entity.UpdatedAt,
	}
}
