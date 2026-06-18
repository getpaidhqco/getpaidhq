package postgrespgx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type ApiKeyRepo struct {
	pool *pgxpool.Pool
}

func NewApiKeyRepo(pool *pgxpool.Pool) port.ApiKeyRepository {
	return &ApiKeyRepo{pool: pool}
}

func (r *ApiKeyRepo) FindById(ctx context.Context, orgId string, id string) (domain.ApiKey, error) {
	q := dbFromCtx(ctx, r.pool)
	var row apiKeyRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+apiKeyColumns+` FROM api_keys WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.ApiKey{}, translateErr(err)
	}
	return row.toDomain(), nil
}

// FindByKey looks up an API key by its HMAC hash. The caller (apikey authn
// middleware) is responsible for hashing the raw key with the configured pepper
// before calling — see lib.HashApiKey. The lookup is intentionally NOT
// org-scoped: it hits the global unique index on key_hash, so a single row
// across all orgs is matched and existence-vs-absence does not leak a timing
// difference from a row scan. The argument carries the already-hashed value.
func (r *ApiKeyRepo) FindByKey(ctx context.Context, key string) (domain.ApiKey, error) {
	q := dbFromCtx(ctx, r.pool)
	var row apiKeyRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+apiKeyColumns+` FROM api_keys WHERE key_hash = $1`, key)); err != nil {
		return domain.ApiKey{}, translateErr(err)
	}
	return row.toDomain(), nil
}

// List returns the org's API keys with stable pagination. Callers MUST NOT
// surface KeyHash to end-users.
func (r *ApiKeyRepo) List(ctx context.Context, orgId string, pagination domain.Pagination) ([]domain.ApiKey, int, error) {
	q := dbFromCtx(ctx, r.pool)
	var count int64
	if err := q.QueryRow(ctx, `SELECT count(*) FROM api_keys WHERE org_id = $1`, orgId).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx, `SELECT `+apiKeyColumns+` FROM api_keys WHERE org_id = $1`+paginationClause(pagination), orgId)
	if err != nil {
		return nil, 0, err
	}
	out, err := r.collect(rows)
	if err != nil {
		return nil, 0, err
	}
	return out, int(count), nil
}

func (r *ApiKeyRepo) Create(ctx context.Context, entity domain.ApiKey) (domain.ApiKey, error) {
	row := apiKeyRowFromDomain(entity)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO api_keys (`+apiKeyColumns+`) VALUES ($1,$2,$3,$4,$5,$6)`,
		row.OrgId, row.Id, row.Name, row.KeyHash, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.ApiKey{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *ApiKeyRepo) Update(ctx context.Context, entity domain.ApiKey) (domain.ApiKey, error) {
	row := apiKeyRowFromDomain(entity)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE api_keys SET name=$3, key_hash=$4, updated_at=$5 WHERE org_id=$1 AND id=$2`,
		row.OrgId, row.Id, row.Name, row.KeyHash, row.UpdatedAt)
	if err != nil {
		return domain.ApiKey{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *ApiKeyRepo) Delete(ctx context.Context, orgId string, id string) error {
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx, `DELETE FROM api_keys WHERE org_id = $1 AND id = $2`, orgId, id)
	return err
}

// collect drains rows into domain api keys, closing rows.
func (r *ApiKeyRepo) collect(rows pgx.Rows) ([]domain.ApiKey, error) {
	defer rows.Close()
	var out []domain.ApiKey
	for rows.Next() {
		var row apiKeyRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row.toDomain())
	}
	return out, rows.Err()
}
