package postgrespgx

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type PaymentMethodRepo struct {
	pool *pgxpool.Pool
}

func NewPaymentMethodRepo(pool *pgxpool.Pool) port.PaymentMethodRepository {
	return &PaymentMethodRepo{pool: pool}
}

func (r *PaymentMethodRepo) FindById(ctx context.Context, orgId string, id string) (domain.PaymentMethod, error) {
	q := dbFromCtx(ctx, r.pool)
	var row paymentMethodRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+paymentMethodColumns+` FROM payment_methods WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.PaymentMethod{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *PaymentMethodRepo) Create(ctx context.Context, entity domain.PaymentMethod) (domain.PaymentMethod, error) {
	row := paymentMethodRowFromDomain(entity)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO payment_methods (`+paymentMethodColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		row.OrgId, row.Id, row.Status, row.Psp, row.Name, row.CustomerId, row.BillingAddress,
		row.Type, row.Token, row.Details, row.Metadata, row.ExpireAt, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.PaymentMethod{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *PaymentMethodRepo) Update(ctx context.Context, entity domain.PaymentMethod) (domain.PaymentMethod, error) {
	row := paymentMethodRowFromDomain(entity)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE payment_methods SET status=$3, psp=$4, name=$5, customer_id=$6, billing_address=$7,
		        type=$8, token=$9, details=$10, metadata=$11, expire_at=$12, updated_at=$13
		 WHERE org_id=$1 AND id=$2`,
		row.OrgId, row.Id, row.Status, row.Psp, row.Name, row.CustomerId, row.BillingAddress,
		row.Type, row.Token, row.Details, row.Metadata, row.ExpireAt, row.UpdatedAt)
	if err != nil {
		return domain.PaymentMethod{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *PaymentMethodRepo) FindExpiringPaymentMethods(ctx context.Context, expiry time.Time) ([]domain.PaymentMethod, error) {
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT `+paymentMethodColumns+` FROM payment_methods WHERE expire_at <= $1 AND expire_at > $2`,
		expiry, time.Time{})
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PaymentMethod
	for rows.Next() {
		var row paymentMethodRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row.toDomain())
	}
	return out, rows.Err()
}
