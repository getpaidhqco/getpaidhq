package postgrespgx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type DiscountRepo struct {
	pool *pgxpool.Pool
}

func NewDiscountRepo(pool *pgxpool.Pool) port.DiscountRepository {
	return &DiscountRepo{pool: pool}
}

func (r *DiscountRepo) Create(ctx context.Context, discount domain.Discount) (domain.Discount, error) {
	row := discountRowFromDomain(discount)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO discounts (`+discountColumns+`)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		row.OrgId, row.Id, row.CouponId, row.CouponCodeId, row.CustomerId,
		row.SubscriptionId, row.OrderId, row.StartCycle, row.Status, row.RedeemedAt,
		row.EndedAt, row.Metadata, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.Discount{}, err
	}
	return r.FindById(ctx, discount.OrgId, discount.Id)
}

func (r *DiscountRepo) Update(ctx context.Context, discount domain.Discount) (domain.Discount, error) {
	row := discountRowFromDomain(discount)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE discounts SET coupon_id=$3, coupon_code_id=$4, customer_id=$5, subscription_id=$6,
		        order_id=$7, start_cycle=$8, status=$9, redeemed_at=$10, ended_at=$11,
		        metadata=$12, created_at=$13, updated_at=$14
		 WHERE org_id=$1 AND id=$2`,
		row.OrgId, row.Id, row.CouponId, row.CouponCodeId, row.CustomerId,
		row.SubscriptionId, row.OrderId, row.StartCycle, row.Status, row.RedeemedAt,
		row.EndedAt, row.Metadata, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.Discount{}, err
	}
	return r.FindById(ctx, discount.OrgId, discount.Id)
}

func (r *DiscountRepo) FindById(ctx context.Context, orgId, id string) (domain.Discount, error) {
	q := dbFromCtx(ctx, r.pool)
	var row discountRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+discountColumns+` FROM discounts WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.Discount{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *DiscountRepo) ActiveForSubscription(ctx context.Context, orgId, subscriptionId string) ([]domain.Discount, error) {
	return r.activeBy(ctx, orgId, "subscription_id", subscriptionId)
}

// ActiveForOrder returns only order-level discounts (subscription_id IS NULL),
// so a subscription-targeted discount never leaks into a one-time order invoice.
func (r *DiscountRepo) ActiveForOrder(ctx context.Context, orgId, orderId string) ([]domain.Discount, error) {
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT `+discountColumns+` FROM discounts
		 WHERE org_id = $1 AND status = $2 AND order_id = $3 AND subscription_id IS NULL`,
		orgId, string(domain.DiscountStatusActive), orderId)
	if err != nil {
		return nil, err
	}
	return r.collect(rows)
}

// activeBy lists active discounts scoped to org and a single equality on the
// named column. col is an internal allowlisted identifier (never user input),
// so concatenating it into the WHERE is safe.
func (r *DiscountRepo) activeBy(ctx context.Context, orgId, col, arg string) ([]domain.Discount, error) {
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT `+discountColumns+` FROM discounts WHERE org_id = $1 AND status = $2 AND `+col+` = $3`,
		orgId, string(domain.DiscountStatusActive), arg)
	if err != nil {
		return nil, err
	}
	return r.collect(rows)
}

func (r *DiscountRepo) CountByCoupon(ctx context.Context, orgId, couponId string) (int, error) {
	q := dbFromCtx(ctx, r.pool)
	var n int64
	err := q.QueryRow(ctx,
		`SELECT count(*) FROM discounts WHERE org_id = $1 AND coupon_id = $2`, orgId, couponId).Scan(&n)
	return int(n), err
}

func (r *DiscountRepo) CountByCouponAndCustomer(ctx context.Context, orgId, couponId, customerId string) (int, error) {
	q := dbFromCtx(ctx, r.pool)
	var n int64
	err := q.QueryRow(ctx,
		`SELECT count(*) FROM discounts WHERE org_id = $1 AND coupon_id = $2 AND customer_id = $3`,
		orgId, couponId, customerId).Scan(&n)
	return int(n), err
}

// collect drains rows into domain discounts, closing rows.
func (r *DiscountRepo) collect(rows pgx.Rows) ([]domain.Discount, error) {
	defer rows.Close()
	out := make([]domain.Discount, 0)
	for rows.Next() {
		var row discountRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row.toDomain())
	}
	return out, rows.Err()
}
