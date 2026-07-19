package postgrespgx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

const orgColumns = `id, name, country, timezone, status, metadata, created_at, updated_at`

type OrgRepo struct {
	pool *pgxpool.Pool
}

func NewOrgRepo(pool *pgxpool.Pool) port.OrgRepository {
	return &OrgRepo{pool: pool}
}

func scanOrg(row pgx.Row) (orgRow, error) {
	var o orgRow
	err := row.Scan(&o.Id, &o.Name, &o.Country, &o.Timezone, &o.Status, &o.Metadata, &o.CreatedAt, &o.UpdatedAt)
	return o, err
}

func (r *OrgRepo) Create(ctx context.Context, entity domain.Org) (domain.Org, error) {
	row := orgRowFromDomain(entity)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO orgs (`+orgColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		row.Id, row.Name, row.Country, row.Timezone, row.Status, row.Metadata, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.Org{}, err
	}
	return r.findById(ctx, entity.Id)
}

func (r *OrgRepo) findById(ctx context.Context, id string) (domain.Org, error) {
	q := dbFromCtx(ctx, r.pool)
	row, err := scanOrg(q.QueryRow(ctx, `SELECT `+orgColumns+` FROM orgs WHERE id = $1`, id))
	if err != nil {
		return domain.Org{}, translateErr(err)
	}
	return row.toDomain(), nil
}

// ListIds returns all org ids — see the port doc; the billing sweep gates on
// subscription status per-org, so this intentionally returns every org.
func (r *OrgRepo) ListIds(ctx context.Context) ([]string, error) {
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx, `SELECT id FROM orgs`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	return ids, nil
}
