package postgrespgx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type InvoiceRepo struct {
	pool *pgxpool.Pool
}

func NewInvoiceRepo(pool *pgxpool.Pool) port.InvoiceRepository {
	return &InvoiceRepo{pool: pool}
}

// Create persists the invoice and all its line items atomically. The gorm
// adapter relies on gorm cascading the association in one Create; here we open
// a transaction (joining the caller's ambient tx via SAVEPOINT when one is
// present, else a fresh tx) and insert the invoice followed by each line item.
// On success we reselect the full invoice with its line items hydrated, exactly
// as gorm's FindById-after-Create does.
func (r *InvoiceRepo) Create(ctx context.Context, entity domain.Invoice) (domain.Invoice, error) {
	entity.Metadata = emptyIfNil(entity.Metadata)
	row := invoiceRowFromDomain(entity)

	err := inTx(ctx, r.pool, func(ctx context.Context) error {
		q := dbFromCtx(ctx, r.pool)
		if _, err := q.Exec(ctx,
			`INSERT INTO invoices (`+invoiceColumns+`)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
			row.OrgId, row.Id, row.Number, row.SubscriptionId, row.CustomerId, row.OrderId,
			row.Status, row.Currency, row.Subtotal, row.DiscountTotal, row.Total,
			row.Cycle, row.PeriodStart, row.PeriodEnd, row.Metadata, row.CreatedAt, row.UpdatedAt); err != nil {
			return err
		}
		for _, li := range entity.LineItems {
			lr := invoiceLineItemRowFromDomain(li)
			if _, err := q.Exec(ctx,
				`INSERT INTO invoice_line_items (`+invoiceLineItemColumns+`)
				 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
				lr.OrgId, lr.Id, lr.InvoiceId, lr.PriceId, lr.Kind, lr.Description,
				lr.Quantity, lr.UnitAmount, lr.Total, lr.DiscountTotal, lr.Metadata,
				lr.CreatedAt, lr.UpdatedAt); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return domain.Invoice{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *InvoiceRepo) NextInvoiceNumber(ctx context.Context, orgId string) (int64, error) {
	q := dbFromCtx(ctx, r.pool)
	var next int64
	err := q.QueryRow(ctx, `
		INSERT INTO invoice_counters (org_id, value, created_at, updated_at)
		VALUES ($1, 1, NOW(), NOW())
		ON CONFLICT (org_id) DO UPDATE
		SET value = invoice_counters.value + 1, updated_at = NOW()
		RETURNING value`, orgId).Scan(&next)
	if err != nil {
		return 0, err
	}
	return next, nil
}

func (r *InvoiceRepo) SetInvoiceCounter(ctx context.Context, orgId string, value int64) error {
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx, `
		INSERT INTO invoice_counters (org_id, value, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (org_id) DO UPDATE
		SET value = EXCLUDED.value, updated_at = NOW()`, orgId, value)
	return err
}

func (r *InvoiceRepo) FindById(ctx context.Context, orgId string, id string) (domain.Invoice, error) {
	q := dbFromCtx(ctx, r.pool)
	var row invoiceRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+invoiceColumns+` FROM invoices WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.Invoice{}, translateErr(err)
	}
	inv := row.toDomain()
	lineItems, err := r.lineItems(ctx, orgId, id)
	if err != nil {
		return domain.Invoice{}, err
	}
	inv.LineItems = lineItems
	return inv, nil
}

func (r *InvoiceRepo) FindBySubscriptionCycle(ctx context.Context, orgId, subscriptionId string, cycle int) (domain.Invoice, error) {
	q := dbFromCtx(ctx, r.pool)
	var row invoiceRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+invoiceColumns+` FROM invoices WHERE org_id = $1 AND subscription_id = $2 AND cycle = $3`,
		orgId, subscriptionId, cycle)); err != nil {
		return domain.Invoice{}, translateErr(err)
	}
	inv := row.toDomain()
	lineItems, err := r.lineItems(ctx, orgId, inv.Id)
	if err != nil {
		return domain.Invoice{}, err
	}
	inv.LineItems = lineItems
	return inv, nil
}

func (r *InvoiceRepo) List(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Invoice, int, error) {
	q := dbFromCtx(ctx, r.pool)
	var count int64
	if err := q.QueryRow(ctx, `SELECT count(*) FROM invoices WHERE org_id = $1`, orgId).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx,
		`SELECT `+invoiceColumns+` FROM invoices WHERE org_id = $1`+paginationClause(p), orgId)
	if err != nil {
		return nil, 0, err
	}
	out, err := r.collect(ctx, orgId, rows)
	if err != nil {
		return nil, 0, err
	}
	return out, int(count), nil
}

func (r *InvoiceRepo) FindBySubscriptionId(ctx context.Context, orgId, subscriptionId string, p domain.Pagination) ([]domain.Invoice, int, error) {
	q := dbFromCtx(ctx, r.pool)
	var count int64
	if err := q.QueryRow(ctx,
		`SELECT count(*) FROM invoices WHERE org_id = $1 AND subscription_id = $2`,
		orgId, subscriptionId).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx,
		`SELECT `+invoiceColumns+` FROM invoices WHERE org_id = $1 AND subscription_id = $2`+paginationClause(p),
		orgId, subscriptionId)
	if err != nil {
		return nil, 0, err
	}
	out, err := r.collect(ctx, orgId, rows)
	if err != nil {
		return nil, 0, err
	}
	return out, int(count), nil
}

func (r *InvoiceRepo) Update(ctx context.Context, entity domain.Invoice) (domain.Invoice, error) {
	entity.Metadata = emptyIfNil(entity.Metadata)
	row := invoiceRowFromDomain(entity)
	q := dbFromCtx(ctx, r.pool)
	// Update the invoice row only (status / total transitions); line items are
	// written at Create time and not mutated here. created_at is intentionally
	// not in the SET list — every $N below is referenced.
	_, err := q.Exec(ctx,
		`UPDATE invoices SET number=$3, subscription_id=$4, customer_id=$5, order_id=$6, status=$7,
		        currency=$8, subtotal=$9, discount_total=$10, total=$11, cycle=$12,
		        period_start=$13, period_end=$14, metadata=$15, updated_at=$16
		 WHERE org_id=$1 AND id=$2`,
		row.OrgId, row.Id, row.Number, row.SubscriptionId, row.CustomerId, row.OrderId, row.Status,
		row.Currency, row.Subtotal, row.DiscountTotal, row.Total, row.Cycle,
		row.PeriodStart, row.PeriodEnd, row.Metadata, row.UpdatedAt)
	if err != nil {
		return domain.Invoice{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

// lineItems hydrates an invoice's line items, mirroring the gorm Preload. The
// gorm adapter imposes no explicit order; we order by created_at then id so the
// result is deterministic (stable insert order) without changing the set.
func (r *InvoiceRepo) lineItems(ctx context.Context, orgId, invoiceId string) ([]domain.InvoiceLineItem, error) {
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT `+invoiceLineItemColumns+` FROM invoice_line_items
		 WHERE org_id = $1 AND invoice_id = $2 ORDER BY created_at, id`, orgId, invoiceId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []invoiceLineItemRow
	for rows.Next() {
		var li invoiceLineItemRow
		if err := li.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, li)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return invoiceLineItemRowsToDomain(out), nil
}

// collect drains invoice rows, hydrating each invoice's line items, closing
// rows. Line items are fetched after the parent rows are fully read so the
// pooled connection is free for the follow-up queries.
func (r *InvoiceRepo) collect(ctx context.Context, orgId string, rows pgx.Rows) ([]domain.Invoice, error) {
	defer rows.Close()
	var invoices []domain.Invoice
	for rows.Next() {
		var row invoiceRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		invoices = append(invoices, row.toDomain())
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	for i := range invoices {
		lineItems, err := r.lineItems(ctx, orgId, invoices[i].Id)
		if err != nil {
			return nil, err
		}
		invoices[i].LineItems = lineItems
	}
	return invoices, nil
}
