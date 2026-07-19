package postgrespgx

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type PspRepo struct {
	pool *pgxpool.Pool
}

func NewPspRepo(pool *pgxpool.Pool) port.PspRepository {
	return &PspRepo{pool: pool}
}

func (r *PspRepo) FindById(ctx context.Context, orgId string, id string) (domain.PspConfig, error) {
	q := dbFromCtx(ctx, r.pool)
	var row pspConfigRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+pspConfigColumns+` FROM gateways WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.PspConfig{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *PspRepo) Create(ctx context.Context, input domain.PspConfig) (domain.PspConfig, error) {
	row := pspConfigRowFromDomain(input)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO gateways (`+pspConfigColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		row.OrgId, row.Id, row.PspId, row.Name, row.Active, row.Config, row.Credentials, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.PspConfig{}, err
	}
	return r.FindById(ctx, input.OrgId, input.Id)
}
