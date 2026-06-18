package postgrespgx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type PriceRepo struct {
	pool *pgxpool.Pool
}

func NewPriceRepo(pool *pgxpool.Pool) port.PriceRepository {
	return &PriceRepo{pool: pool}
}

func (r *PriceRepo) Create(ctx context.Context, entity domain.Price) (domain.Price, error) {
	row := priceRowFromDomain(entity)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO prices (`+priceColumns+`)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26)`,
		row.OrgId, row.Id, row.VariantId, row.Label, row.Category, row.Scheme, row.Cycles,
		row.Currency, row.UnitPrice, row.UnitCount, row.MinPrice, row.SuggestedPrice,
		row.BillingInterval, row.BillingIntervalQty, row.TrialInterval, row.TrialIntervalQty,
		row.TaxCode, row.BillableMetricId, row.Tiers, row.FilterField, row.FilterValue,
		row.ProrateOnIncrease, row.CreditOnDecrease, row.Metadata, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.Price{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *PriceRepo) FindById(ctx context.Context, orgId string, id string) (domain.Price, error) {
	q := dbFromCtx(ctx, r.pool)
	var row priceRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+priceColumns+` FROM prices WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.Price{}, translateErr(err)
	}
	return row.toDomain(), nil
}

// FindByIds batch-loads prices by ID within an org. Used by services to hydrate
// read models without N+1 (e.g. OrderItemDetails composition).
func (r *PriceRepo) FindByIds(ctx context.Context, orgId string, ids []string) ([]domain.Price, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT `+priceColumns+` FROM prices WHERE org_id = $1 AND id = ANY($2)`, orgId, ids)
	if err != nil {
		return nil, err
	}
	return r.collect(rows)
}

func (r *PriceRepo) FindByVariantId(ctx context.Context, orgId string, variantId string, p domain.Pagination) ([]domain.Price, int, error) {
	q := dbFromCtx(ctx, r.pool)
	var count int64
	if err := q.QueryRow(ctx,
		`SELECT count(*) FROM prices WHERE org_id = $1 AND variant_id = $2`, orgId, variantId).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx,
		`SELECT `+priceColumns+` FROM prices WHERE org_id = $1 AND variant_id = $2`+paginationClause(p), orgId, variantId)
	if err != nil {
		return nil, 0, err
	}
	out, err := r.collect(rows)
	if err != nil {
		return nil, 0, err
	}
	return out, int(count), nil
}

// FindByVariantIds batch-loads prices across many variants. Used by Product
// read-model composition.
func (r *PriceRepo) FindByVariantIds(ctx context.Context, orgId string, variantIds []string) ([]domain.Price, error) {
	if len(variantIds) == 0 {
		return nil, nil
	}
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT `+priceColumns+` FROM prices WHERE org_id = $1 AND variant_id = ANY($2)`, orgId, variantIds)
	if err != nil {
		return nil, err
	}
	return r.collect(rows)
}

func (r *PriceRepo) Update(ctx context.Context, entity domain.Price) (domain.Price, error) {
	row := priceRowFromDomain(entity)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE prices SET variant_id=$3, label=$4, category=$5, scheme=$6, cycles=$7, currency=$8,
		        unit_price=$9, unit_count=$10, min_price=$11, suggested_price=$12, billing_interval=$13,
		        billing_interval_qty=$14, trial_interval=$15, trial_interval_qty=$16, tax_code=$17,
		        billable_metric_id=$18, tiers=$19, filter_field=$20, filter_value=$21,
		        prorate_on_increase=$22, credit_on_decrease=$23, metadata=$24, updated_at=$25
		 WHERE org_id=$1 AND id=$2`,
		row.OrgId, row.Id, row.VariantId, row.Label, row.Category, row.Scheme, row.Cycles,
		row.Currency, row.UnitPrice, row.UnitCount, row.MinPrice, row.SuggestedPrice,
		row.BillingInterval, row.BillingIntervalQty, row.TrialInterval, row.TrialIntervalQty,
		row.TaxCode, row.BillableMetricId, row.Tiers, row.FilterField, row.FilterValue,
		row.ProrateOnIncrease, row.CreditOnDecrease, row.Metadata, row.UpdatedAt)
	if err != nil {
		return domain.Price{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *PriceRepo) Delete(ctx context.Context, orgId string, id string) error {
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx, `DELETE FROM prices WHERE org_id = $1 AND id = $2`, orgId, id)
	return err
}

// collect drains rows into domain prices, closing rows.
func (r *PriceRepo) collect(rows pgx.Rows) ([]domain.Price, error) {
	defer rows.Close()
	var out []domain.Price
	for rows.Next() {
		var row priceRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row.toDomain())
	}
	return out, rows.Err()
}
