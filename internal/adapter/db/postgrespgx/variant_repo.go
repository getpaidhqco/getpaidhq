package postgrespgx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type VariantRepo struct {
	pool *pgxpool.Pool
}

func NewVariantRepo(pool *pgxpool.Pool) port.VariantRepository {
	return &VariantRepo{pool: pool}
}

func (r *VariantRepo) Create(ctx context.Context, entity domain.Variant) (domain.Variant, error) {
	row := variantRowFromDomain(entity)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO variants (`+variantColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		row.OrgId, row.Id, row.ProductId, row.Name, row.Description, row.Metadata, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.Variant{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *VariantRepo) FindById(ctx context.Context, orgId string, id string) (domain.Variant, error) {
	q := dbFromCtx(ctx, r.pool)
	var row variantRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+variantColumns+` FROM variants WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.Variant{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *VariantRepo) FindByProductId(ctx context.Context, orgId string, productId string, p domain.Pagination) ([]domain.Variant, int, error) {
	q := dbFromCtx(ctx, r.pool)
	var count int64
	if err := q.QueryRow(ctx,
		`SELECT count(*) FROM variants WHERE org_id = $1 AND product_id = $2`, orgId, productId).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx,
		`SELECT `+variantColumns+` FROM variants WHERE org_id = $1 AND product_id = $2`+paginationClause(p), orgId, productId)
	if err != nil {
		return nil, 0, err
	}
	out, err := r.collect(rows)
	if err != nil {
		return nil, 0, err
	}
	return out, int(count), nil
}

func (r *VariantRepo) Update(ctx context.Context, entity domain.Variant) (domain.Variant, error) {
	row := variantRowFromDomain(entity)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE variants SET product_id=$3, name=$4, description=$5, metadata=$6, updated_at=$7
		 WHERE org_id=$1 AND id=$2`,
		row.OrgId, row.Id, row.ProductId, row.Name, row.Description, row.Metadata, row.UpdatedAt)
	if err != nil {
		return domain.Variant{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *VariantRepo) Delete(ctx context.Context, orgId string, id string) error {
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx, `DELETE FROM variants WHERE org_id = $1 AND id = $2`, orgId, id)
	return err
}

// collect drains rows into domain variants, closing rows.
func (r *VariantRepo) collect(rows pgx.Rows) ([]domain.Variant, error) {
	defer rows.Close()
	var out []domain.Variant
	for rows.Next() {
		var row variantRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row.toDomain())
	}
	return out, rows.Err()
}
