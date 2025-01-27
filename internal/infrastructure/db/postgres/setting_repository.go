package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/domain/entities"
	"payloop/internal/lib"
)

type SettingRepository struct {
	*lib.PgDatabase
	logger lib.Logger
}

func NewSettingRepository(database lib.Database, logger lib.Logger) SettingRepository {
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return SettingRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

// WithTrx enables repository with transaction
func (r *SettingRepository) WithTrx(trxHandle interface{}) *SettingRepository {
	if trxHandle == nil {
		r.logger.Warn("Transaction Database not found in gin context. ")
		return r
	}
	r.PgDatabase.Tx = trxHandle.(pgx.Tx)
	return r
}

func (r *SettingRepository) FindById(ctx context.Context, org_id string, id string) (entities.Setting, error) {
	var setting entities.Setting
	err := r.Pool.QueryRow(ctx,
		`SELECT * FROM settings WHERE org_id=@org_id AND id=@id`,
		pgx.NamedArgs{
			"org_id": org_id,
			"id":     id,
		}).Scan(
		&setting.OrgId,
		&setting.Id,
		&setting.Value,
	)

	if err != nil {
		r.logger.Error(`failed to find Setting`, err.Error())
		return entities.Setting{}, errors.New("not found")
	}
	return setting, nil
}

func (r *SettingRepository) Save(ctx context.Context, entity entities.Setting) (entities.Setting, error) {

	var setting entities.Setting

	query := `INSERT INTO settings (org_id, id, value, created_at, updated_at) 
			  VALUES (@org_id, @id, @value, NOW(), NOW())
			  ON CONFLICT DO UPDATE SET org_id, id, value, created_at, updated_at
			  
			  RETURNING org_id, id, value`

	err := r.Pool.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id":   entity.OrgId,
		"id":       entity.Id,
		"order_id": entity.Value,
	}).Scan(
		&setting.OrgId,
		&setting.Id,
		&setting.Value,
	)

	if err != nil {
		r.logger.Error(`failed to insert Setting`, err.Error())
		return entities.Setting{}, err
	}

	return setting, nil
}
