package postgrespgx

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type CouponReservationRepo struct {
	pool *pgxpool.Pool
}

func NewCouponReservationRepo(pool *pgxpool.Pool) port.CouponReservationRepository {
	return &CouponReservationRepo{pool: pool}
}

func (r *CouponReservationRepo) Create(ctx context.Context, res domain.CouponReservation) (domain.CouponReservation, error) {
	row := couponReservationRowFromDomain(res)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO coupon_reservations (`+couponReservationColumns+`)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		row.OrgId, row.Id, row.CouponId, row.CouponCodeId, row.CustomerId,
		row.CheckoutSessionId, row.OrderId, row.ExpiresAt, row.CreatedAt)
	if err != nil {
		return domain.CouponReservation{}, err
	}
	return r.findById(ctx, res.OrgId, res.Id)
}

func (r *CouponReservationRepo) FindByOrder(ctx context.Context, orgId, orderId string) ([]domain.CouponReservation, error) {
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT `+couponReservationColumns+` FROM coupon_reservations WHERE org_id = $1 AND order_id = $2`,
		orgId, orderId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.CouponReservation, 0)
	for rows.Next() {
		var row couponReservationRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row.toDomain())
	}
	return out, rows.Err()
}

func (r *CouponReservationRepo) DeleteByOrder(ctx context.Context, orgId, orderId string) error {
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`DELETE FROM coupon_reservations WHERE org_id = $1 AND order_id = $2`, orgId, orderId)
	return err
}

func (r *CouponReservationRepo) CountLiveByCoupon(ctx context.Context, orgId, couponId string, now time.Time) (int, error) {
	q := dbFromCtx(ctx, r.pool)
	var n int
	err := q.QueryRow(ctx,
		`SELECT count(*) FROM coupon_reservations WHERE org_id = $1 AND coupon_id = $2 AND expires_at > $3`,
		orgId, couponId, now).Scan(&n)
	return n, err
}

func (r *CouponReservationRepo) CountLiveByCode(ctx context.Context, orgId, couponCodeId string, now time.Time) (int, error) {
	q := dbFromCtx(ctx, r.pool)
	var n int
	err := q.QueryRow(ctx,
		`SELECT count(*) FROM coupon_reservations WHERE org_id = $1 AND coupon_code_id = $2 AND expires_at > $3`,
		orgId, couponCodeId, now).Scan(&n)
	return n, err
}

func (r *CouponReservationRepo) ExistsLiveForCustomer(ctx context.Context, orgId, couponId, customerId string, now time.Time) (bool, error) {
	q := dbFromCtx(ctx, r.pool)
	var exists bool
	err := q.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM coupon_reservations WHERE org_id = $1 AND coupon_id = $2 AND customer_id = $3 AND expires_at > $4)`,
		orgId, couponId, customerId, now).Scan(&exists)
	return exists, err
}

func (r *CouponReservationRepo) DeleteExpired(ctx context.Context, now time.Time) (int, error) {
	q := dbFromCtx(ctx, r.pool)
	tag, err := q.Exec(ctx,
		`DELETE FROM coupon_reservations WHERE expires_at <= $1`, now)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (r *CouponReservationRepo) findById(ctx context.Context, orgId, id string) (domain.CouponReservation, error) {
	q := dbFromCtx(ctx, r.pool)
	var row couponReservationRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+couponReservationColumns+` FROM coupon_reservations WHERE org_id = $1 AND id = $2`,
		orgId, id)); err != nil {
		return domain.CouponReservation{}, translateErr(err)
	}
	return row.toDomain(), nil
}
