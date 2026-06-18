package postgrespgx

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type SessionRepo struct {
	pool *pgxpool.Pool
}

func NewSessionRepo(pool *pgxpool.Pool) port.SessionRepository {
	return &SessionRepo{pool: pool}
}

func (r *SessionRepo) FindById(ctx context.Context, orgId string, id string) (domain.Session, error) {
	q := dbFromCtx(ctx, r.pool)
	var row sessionRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+sessionColumns+` FROM sessions WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.Session{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *SessionRepo) Create(ctx context.Context, input domain.Session) (domain.Session, error) {
	row := sessionRowFromDomain(input)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO sessions (`+sessionColumns+`) VALUES ($1,$2,$3,$4,$5)`,
		row.OrgId, row.Id, row.CartId, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.Session{}, err
	}
	return r.FindById(ctx, input.OrgId, input.Id)
}
