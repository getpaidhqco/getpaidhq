package postgrespgx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type SettingRepo struct {
	pool *pgxpool.Pool
}

func NewSettingRepo(pool *pgxpool.Pool) port.SettingRepository {
	return &SettingRepo{pool: pool}
}

func (r *SettingRepo) FindById(ctx context.Context, orgId string, parentId string, id string) (domain.Setting, error) {
	q := dbFromCtx(ctx, r.pool)
	var row settingRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+settingColumns+` FROM settings WHERE org_id = $1 AND parent_id = $2 AND id = $3`,
		orgId, parentId, id)); err != nil {
		return domain.Setting{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *SettingRepo) Create(ctx context.Context, entity domain.Setting) (domain.Setting, error) {
	row := settingRowFromDomain(entity)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO settings (`+settingColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		row.OrgId, row.ParentId, row.Id, row.Type, row.Value, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.Setting{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.ParentId, entity.Id)
}

func (r *SettingRepo) List(ctx context.Context, orgId string, parentId string, p domain.Pagination) ([]domain.Setting, int, error) {
	q := dbFromCtx(ctx, r.pool)

	countSQL := `SELECT count(*) FROM settings WHERE org_id = $1`
	listSQL := `SELECT ` + settingColumns + ` FROM settings WHERE org_id = $1`
	args := []any{orgId}
	if parentId != "" {
		countSQL += ` AND parent_id = $2`
		listSQL += ` AND parent_id = $2`
		args = append(args, parentId)
	}
	listSQL += paginationClause(p)

	var count int64
	if err := q.QueryRow(ctx, countSQL, args...).Scan(&count); err != nil {
		return nil, 0, err
	}

	rows, err := q.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	out, err := r.collect(rows)
	if err != nil {
		return nil, 0, err
	}
	return out, int(count), nil
}

func (r *SettingRepo) Delete(ctx context.Context, orgId string, parentId string, id string) error {
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`DELETE FROM settings WHERE org_id = $1 AND parent_id = $2 AND id = $3`,
		orgId, parentId, id)
	return err
}

// Upsert is a real Postgres upsert keyed on the (org_id, parent_id, id) PK.
// value_type is in the DO UPDATE set so a future caller writing a different
// Type on an existing key gets correct update semantics rather than a stale
// type — matching the gorm adapter's DoUpdates of {value, value_type,
// updated_at}. created_at is deliberately not updated, so it is not in the SET
// list (a passed-but-unreferenced param would be a 42P18 error).
func (r *SettingRepo) Upsert(ctx context.Context, entity domain.Setting) (domain.Setting, error) {
	row := settingRowFromDomain(entity)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO settings (`+settingColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT (org_id, parent_id, id) DO UPDATE SET
		     value = EXCLUDED.value,
		     value_type = EXCLUDED.value_type,
		     updated_at = EXCLUDED.updated_at`,
		row.OrgId, row.ParentId, row.Id, row.Type, row.Value, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.Setting{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.ParentId, entity.Id)
}

// collect drains rows into domain settings, closing rows.
func (r *SettingRepo) collect(rows pgx.Rows) ([]domain.Setting, error) {
	defer rows.Close()
	var out []domain.Setting
	for rows.Next() {
		var row settingRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row.toDomain())
	}
	return out, rows.Err()
}
