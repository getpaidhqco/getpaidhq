package postgrespgx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

type CouponRepo struct {
	pool *pgxpool.Pool
}

func NewCouponRepo(pool *pgxpool.Pool) port.CouponRepository {
	return &CouponRepo{pool: pool}
}

func (r *CouponRepo) Create(ctx context.Context, coupon domain.Coupon) (domain.Coupon, error) {
	row := couponRowFromDomain(coupon)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO coupons (`+couponColumns+`)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		row.OrgId, row.Id, row.Name, row.Active, row.Metadata, row.DiscountType,
		row.AmountOff, row.Currency, row.PercentOff, row.Duration, row.DurationInCycles,
		row.RedeemBy, row.AppliesToProducts, row.MaxRedemptions, row.OncePerCustomer,
		row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.Coupon{}, err
	}
	return r.FindById(ctx, coupon.OrgId, coupon.Id)
}

// UpdateMutable persists ONLY name, active and metadata — terms are immutable.
func (r *CouponRepo) UpdateMutable(ctx context.Context, orgId, id, name string, active bool, metadata map[string]string) (domain.Coupon, error) {
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE coupons SET name=$3, active=$4, metadata=$5, updated_at=now() WHERE org_id=$1 AND id=$2`,
		orgId, id, name, active, newJSON(metadata))
	if err != nil {
		return domain.Coupon{}, err
	}
	return r.FindById(ctx, orgId, id)
}

func (r *CouponRepo) FindById(ctx context.Context, orgId, id string) (domain.Coupon, error) {
	q := dbFromCtx(ctx, r.pool)
	var row couponRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+couponColumns+` FROM coupons WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.Coupon{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *CouponRepo) Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Coupon, int, error) {
	q := dbFromCtx(ctx, r.pool)
	var count int64
	if err := q.QueryRow(ctx, `SELECT count(*) FROM coupons WHERE org_id = $1`, orgId).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx, `SELECT `+couponColumns+` FROM coupons WHERE org_id = $1`+paginationClause(p), orgId)
	if err != nil {
		return nil, 0, err
	}
	out, err := r.collect(rows)
	if err != nil {
		return nil, 0, err
	}
	return out, int(count), nil
}

func (r *CouponRepo) DeleteIfUnreferenced(ctx context.Context, orgId, id string) error {
	q := dbFromCtx(ctx, r.pool)
	var count int64
	if err := q.QueryRow(ctx,
		`SELECT count(*) FROM discounts WHERE org_id = $1 AND coupon_id = $2`, orgId, id).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return lib.NewCustomError(lib.BadRequestError, "coupon has discounts and cannot be deleted", nil)
	}
	_, err := q.Exec(ctx, `DELETE FROM coupons WHERE org_id = $1 AND id = $2`, orgId, id)
	return err
}

// collect drains rows into domain coupons, closing rows.
func (r *CouponRepo) collect(rows pgx.Rows) ([]domain.Coupon, error) {
	defer rows.Close()
	out := []domain.Coupon{}
	for rows.Next() {
		var row couponRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row.toDomain())
	}
	return out, rows.Err()
}
