package postgrespgx

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type CartRepo struct {
	pool *pgxpool.Pool
}

func NewCartRepo(pool *pgxpool.Pool) port.CartRepository {
	return &CartRepo{pool: pool}
}

func (r *CartRepo) FindById(ctx context.Context, orgId string, id string) (domain.Cart, error) {
	q := dbFromCtx(ctx, r.pool)
	var row cartRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+cartColumns+` FROM carts WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.Cart{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *CartRepo) Create(ctx context.Context, input domain.Cart) (domain.Cart, error) {
	row := cartRowFromDomain(input)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO carts (`+cartColumns+`) VALUES ($1,$2,$3,$4,$5,$6)`,
		row.OrgId, row.Id, row.Data, row.Metadata, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.Cart{}, err
	}
	return r.FindById(ctx, input.OrgId, input.Id)
}

func (r *CartRepo) Update(ctx context.Context, input domain.Cart) (domain.Cart, error) {
	row := cartRowFromDomain(input)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE carts SET data=$3, metadata=$4, updated_at=$5 WHERE org_id=$1 AND id=$2`,
		row.OrgId, row.Id, row.Data, row.Metadata, row.UpdatedAt)
	if err != nil {
		return domain.Cart{}, err
	}
	return r.FindById(ctx, input.OrgId, input.Id)
}
