package postgrespgx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type ProductRepo struct {
	pool *pgxpool.Pool
}

func NewProductRepo(pool *pgxpool.Pool) port.ProductRepository {
	return &ProductRepo{pool: pool}
}

func (r *ProductRepo) FindById(ctx context.Context, orgId string, id string) (domain.Product, error) {
	q := dbFromCtx(ctx, r.pool)
	var row productRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+productColumns+` FROM products WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.Product{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *ProductRepo) Create(ctx context.Context, product domain.Product) (domain.Product, error) {
	row := productRowFromDomain(product)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO products (`+productColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		row.OrgId, row.Id, row.Name, row.Description, row.Status,
		row.ArchivedAt, row.Metadata, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.Product{}, err
	}
	return r.FindById(ctx, product.OrgId, product.Id)
}

// Find lists products, optionally filtered by status. An empty/nil statuses
// slice returns all products regardless of status; otherwise the same status
// predicate is applied to both the count and the page query so the total
// matches what's returned. Mirrors the gorm adapter's statusScope.
func (r *ProductRepo) Find(ctx context.Context, orgId string, p domain.Pagination, statuses []domain.ProductStatus) ([]domain.Product, int, error) {
	q := dbFromCtx(ctx, r.pool)

	// Build args and the optional status predicate. Status enum values are
	// passed as []string — never the defined enum type — as a single $2 array
	// arg so the count and page queries share parameter numbering.
	args := []any{orgId}
	statusClause := ""
	if len(statuses) > 0 {
		ss := make([]string, len(statuses))
		for i, s := range statuses {
			ss[i] = string(s)
		}
		args = append(args, ss)
		statusClause = ` AND status = ANY($2)`
	}

	var count int64
	if err := q.QueryRow(ctx,
		`SELECT count(*) FROM products WHERE org_id = $1`+statusClause, args...).Scan(&count); err != nil {
		return nil, 0, err
	}

	rows, err := q.Query(ctx,
		`SELECT `+productColumns+` FROM products WHERE org_id = $1`+statusClause+paginationClause(p), args...)
	if err != nil {
		return nil, 0, err
	}
	out, err := r.collect(rows)
	if err != nil {
		return nil, 0, err
	}
	return out, int(count), nil
}

func (r *ProductRepo) Update(ctx context.Context, product domain.Product) (domain.Product, error) {
	row := productRowFromDomain(product)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE products SET name=$3, description=$4, status=$5, archived_at=$6, metadata=$7, updated_at=$8
		 WHERE org_id=$1 AND id=$2`,
		row.OrgId, row.Id, row.Name, row.Description, row.Status,
		row.ArchivedAt, row.Metadata, row.UpdatedAt)
	if err != nil {
		return domain.Product{}, err
	}
	return r.FindById(ctx, product.OrgId, product.Id)
}

func (r *ProductRepo) Delete(ctx context.Context, orgId string, id string) error {
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx, `DELETE FROM products WHERE org_id = $1 AND id = $2`, orgId, id)
	// A product whose variant is referenced by order_items cannot be
	// hard-deleted — the FK is intentionally Restrict to preserve order
	// history. Surface that as a 409 with a clear message rather than a raw
	// SQLSTATE 23503 leaking out as an opaque 400.
	return asConflictOnFK(err, "Cannot delete a product that has existing orders.")
}

// collect drains rows into domain products, closing rows.
func (r *ProductRepo) collect(rows pgx.Rows) ([]domain.Product, error) {
	defer rows.Close()
	var out []domain.Product
	for rows.Next() {
		var row productRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row.toDomain())
	}
	return out, rows.Err()
}
