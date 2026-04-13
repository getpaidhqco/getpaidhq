package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/lib"
)

type SettingRepository struct {
	*PgDatabase
	logger port.Logger
}

func NewSettingRepository(primaryDb lib.Database, logger port.Logger) port.SettingRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return SettingRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r SettingRepository) FindById(ctx context.Context, orgId string, parentId string, id string) (domain.Setting, error) {
	tx := r.getTransactionFromContext(ctx)

	var setting domain.Setting
	err := tx.QueryRow(ctx,
		`SELECT org_id,parent_id,id,value_type,value FROM settings WHERE org_id=@org_id AND parent_id=@parent_id AND id=@id`,

		pgx.NamedArgs{
			"org_id":    orgId,
			"parent_id": parentId,
			"id":        id,
		}).Scan(
		&setting.OrgId,
		&setting.ParentId,
		&setting.Id,
		&setting.Type,
		&setting.Value,
	)

	if err != nil {
		return domain.Setting{}, errors.New("not found")
	}
	return setting, nil
}

func (r SettingRepository) Create(ctx context.Context, entity domain.Setting) (domain.Setting, error) {
	tx := r.getTransactionFromContext(ctx)
	var setting domain.Setting

	query := `INSERT INTO settings (org_id, parent_id, id, value, value_type, created_at, updated_at)
			  VALUES (@org_id, @parent_id, @id, @value, @value_type, NOW(), NOW())
			  ON CONFLICT (org_id, parent_id, id) DO UPDATE SET value = EXCLUDED.value, value_type = EXCLUDED.value_type, updated_at = NOW()
			  RETURNING org_id, parent_id, id, value, value_type`

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id":     entity.OrgId,
		"parent_id":  entity.ParentId,
		"id":         entity.Id,
		"value":      entity.Value,
		"value_type": entity.Type,
	}).Scan(
		&setting.OrgId,
		&setting.ParentId,
		&setting.Id,
		&setting.Value,
		&setting.Type,
	)

	if err != nil {
		r.logger.Error(`failed to insert Setting`, err.Error())
		return domain.Setting{}, err
	}

	return setting, nil
}
