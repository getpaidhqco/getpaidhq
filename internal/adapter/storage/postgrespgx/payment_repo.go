package postgrespgx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type PaymentRepo struct {
	pool *pgxpool.Pool
}

func NewPaymentRepo(pool *pgxpool.Pool) port.PaymentRepository {
	return &PaymentRepo{pool: pool}
}

func (r *PaymentRepo) FindById(ctx context.Context, orgId string, id string) (domain.Payment, error) {
	q := dbFromCtx(ctx, r.pool)
	var row paymentRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+paymentColumns+` FROM payments WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.Payment{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *PaymentRepo) FindByPspId(ctx context.Context, orgId string, id string) (domain.Payment, error) {
	q := dbFromCtx(ctx, r.pool)
	var row paymentRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+paymentColumns+` FROM payments WHERE org_id = $1 AND psp_id = $2`, orgId, id)); err != nil {
		return domain.Payment{}, translateErr(err)
	}
	return row.toDomain(), nil
}

// ListByPspId is intentionally NOT org-scoped: it resolves a payment by gateway
// + PSP id across orgs (webhook lookups), mirroring the gorm adapter's WHERE.
func (r *PaymentRepo) ListByPspId(ctx context.Context, psp domain.Gateway, pspId string) ([]domain.Payment, error) {
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT `+paymentColumns+` FROM payments WHERE psp = $1 AND psp_id = $2`, string(psp), pspId)
	if err != nil {
		return nil, err
	}
	return r.collect(rows)
}

func (r *PaymentRepo) FindBySubscriptionId(ctx context.Context, orgId string, id string, p domain.Pagination) ([]domain.Payment, int, error) {
	q := dbFromCtx(ctx, r.pool)
	var count int64
	if err := q.QueryRow(ctx,
		`SELECT count(*) FROM payments WHERE org_id = $1 AND subscription_id = $2`, orgId, id).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx,
		`SELECT `+paymentColumns+` FROM payments WHERE org_id = $1 AND subscription_id = $2`+paginationClause(p), orgId, id)
	if err != nil {
		return nil, 0, err
	}
	out, err := r.collect(rows)
	if err != nil {
		return nil, 0, err
	}
	return out, int(count), nil
}

// List returns the org's payments, newest first, paginated. The gorm adapter
// ordered created_at DESC; paginationClause defaults to that same order.
func (r *PaymentRepo) List(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Payment, int, error) {
	q := dbFromCtx(ctx, r.pool)
	var count int64
	if err := q.QueryRow(ctx, `SELECT count(*) FROM payments WHERE org_id = $1`, orgId).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx, `SELECT `+paymentColumns+` FROM payments WHERE org_id = $1`+paginationClause(p), orgId)
	if err != nil {
		return nil, 0, err
	}
	out, err := r.collect(rows)
	if err != nil {
		return nil, 0, err
	}
	return out, int(count), nil
}

func (r *PaymentRepo) Create(ctx context.Context, entity domain.Payment) (domain.Payment, error) {
	row := paymentRowFromDomain(entity)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO payments (`+paymentColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
		row.OrgId, row.Id, row.Psp, row.PspId, row.Reference, row.OrderId, row.SubscriptionId,
		row.InvoiceId, row.Status, row.Recurring, row.Currency, row.Amount, row.PspFee,
		row.PlatformFee, row.NetAmount, row.Metadata, row.CompletedAt, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.Payment{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *PaymentRepo) Update(ctx context.Context, entity domain.Payment) (domain.Payment, error) {
	row := paymentRowFromDomain(entity)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE payments SET psp=$3, psp_id=$4, reference=$5, order_id=$6, subscription_id=$7,
		        invoice_id=$8, status=$9, recurring=$10, currency=$11, amount=$12, psp_fee=$13,
		        platform_fee=$14, net_amount=$15, metadata=$16, completed_at=$17, updated_at=$18
		 WHERE org_id=$1 AND id=$2`,
		row.OrgId, row.Id, row.Psp, row.PspId, row.Reference, row.OrderId, row.SubscriptionId,
		row.InvoiceId, row.Status, row.Recurring, row.Currency, row.Amount, row.PspFee,
		row.PlatformFee, row.NetAmount, row.Metadata, row.CompletedAt, row.UpdatedAt)
	if err != nil {
		return domain.Payment{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *PaymentRepo) CreateRefund(ctx context.Context, refund domain.Refund) (domain.Refund, error) {
	row := refundRowFromDomain(refund)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO refunds (`+refundColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		row.OrgId, row.Id, row.PspRefundId, row.PaymentId, row.Amount, row.Currency,
		row.Reason, row.RefundedAt, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.Refund{}, err
	}
	var created refundRow
	if err := created.scanInto(q.QueryRow(ctx,
		`SELECT `+refundColumns+` FROM refunds WHERE org_id = $1 AND id = $2`, refund.OrgId, refund.Id)); err != nil {
		return domain.Refund{}, translateErr(err)
	}
	return created.toDomain(), nil
}

// collect drains rows into domain payments, closing rows.
func (r *PaymentRepo) collect(rows pgx.Rows) ([]domain.Payment, error) {
	defer rows.Close()
	var out []domain.Payment
	for rows.Next() {
		var row paymentRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row.toDomain())
	}
	return out, rows.Err()
}
