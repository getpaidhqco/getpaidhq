package postgrespgx

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type CouponCodeRepo struct {
	pool *pgxpool.Pool
}

func NewCouponCodeRepo(pool *pgxpool.Pool) port.CouponCodeRepository {
	return &CouponCodeRepo{pool: pool}
}

func (r *CouponCodeRepo) Create(ctx context.Context, code domain.CouponCode) (domain.CouponCode, error) {
	row := couponCodeRowFromDomain(code)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO coupon_codes (`+couponCodeColumns+`)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		row.OrgId, row.Id, row.CouponId, row.Code, row.Active, row.Metadata,
		row.CustomerId, row.ExpiresAt, row.MaxRedemptions, row.TimesRedeemed, row.Restrictions,
		row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.CouponCode{}, err
	}
	return r.findById(ctx, code.OrgId, code.Id)
}

func (r *CouponCodeRepo) UpdateMutable(ctx context.Context, orgId, id string, active bool, metadata map[string]string) (domain.CouponCode, error) {
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE coupon_codes SET active=$3, metadata=$4, updated_at=now() WHERE org_id=$1 AND id=$2`,
		orgId, id, active, newJSON(metadata))
	if err != nil {
		return domain.CouponCode{}, err
	}
	return r.findById(ctx, orgId, id)
}

func (r *CouponCodeRepo) IncrementRedeemed(ctx context.Context, orgId, id string) error {
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE coupon_codes SET times_redeemed = times_redeemed + 1 WHERE org_id = $1 AND id = $2`,
		orgId, id)
	return err
}

// FindByCode resolves a code case-insensitively.
func (r *CouponCodeRepo) FindByCode(ctx context.Context, orgId, code string) (domain.CouponCode, error) {
	q := dbFromCtx(ctx, r.pool)
	var row couponCodeRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+couponCodeColumns+` FROM coupon_codes WHERE org_id = $1 AND upper(code) = upper($2)`,
		orgId, code)); err != nil {
		return domain.CouponCode{}, translateErr(err)
	}
	return row.toDomain(), nil
}

// FindByCodeForUpdate resolves a code case-insensitively and row-locks it.
func (r *CouponCodeRepo) FindByCodeForUpdate(ctx context.Context, orgId, code string) (domain.CouponCode, error) {
	q := dbFromCtx(ctx, r.pool)
	var row couponCodeRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+couponCodeColumns+` FROM coupon_codes WHERE org_id = $1 AND upper(code) = upper($2) FOR UPDATE`,
		orgId, code)); err != nil {
		return domain.CouponCode{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *CouponCodeRepo) FindByCouponId(ctx context.Context, orgId, couponId string) ([]domain.CouponCode, error) {
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT `+couponCodeColumns+` FROM coupon_codes WHERE org_id = $1 AND coupon_id = $2 ORDER BY created_at DESC`,
		orgId, couponId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.CouponCode, 0)
	for rows.Next() {
		var row couponCodeRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row.toDomain())
	}
	return out, rows.Err()
}

func (r *CouponCodeRepo) findById(ctx context.Context, orgId, id string) (domain.CouponCode, error) {
	q := dbFromCtx(ctx, r.pool)
	var row couponCodeRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+couponCodeColumns+` FROM coupon_codes WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.CouponCode{}, translateErr(err)
	}
	return row.toDomain(), nil
}
