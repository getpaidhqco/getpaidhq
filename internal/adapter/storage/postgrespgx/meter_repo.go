package postgrespgx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type MeterRepo struct {
	pool *pgxpool.Pool
}

func NewMeterRepo(pool *pgxpool.Pool) port.MeterRepository {
	return &MeterRepo{pool: pool}
}

func (r *MeterRepo) FindByCode(ctx context.Context, orgId, code string) (domain.BillableMetric, error) {
	q := dbFromCtx(ctx, r.pool)
	var row billableMetricRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+billableMetricColumns+` FROM billable_metrics WHERE org_id = $1 AND code = $2`, orgId, code)); err != nil {
		return domain.BillableMetric{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *MeterRepo) FindById(ctx context.Context, orgId, id string) (domain.BillableMetric, error) {
	q := dbFromCtx(ctx, r.pool)
	var row billableMetricRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+billableMetricColumns+` FROM billable_metrics WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.BillableMetric{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *MeterRepo) Create(ctx context.Context, m domain.BillableMetric) (domain.BillableMetric, error) {
	m.Metadata = emptyIfNil(m.Metadata)
	row := billableMetricRowFromDomain(m)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO billable_metrics (`+billableMetricColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		row.OrgId, row.Id, row.Code, row.Name, row.Aggregation, row.FieldName, row.CarryOver,
		row.RoundingMode, row.RoundingScale, row.Filters, row.GroupBy, row.Metadata,
		row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.BillableMetric{}, err
	}
	return r.FindByCode(ctx, m.OrgId, m.Code)
}

func (r *MeterRepo) Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.BillableMetric, int, error) {
	q := dbFromCtx(ctx, r.pool)
	var count int64
	if err := q.QueryRow(ctx, `SELECT count(*) FROM billable_metrics WHERE org_id = $1`, orgId).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx,
		`SELECT `+billableMetricColumns+` FROM billable_metrics WHERE org_id = $1`+paginationClause(p), orgId)
	if err != nil {
		return nil, 0, err
	}
	out, err := r.collect(rows)
	if err != nil {
		return nil, 0, err
	}
	return out, int(count), nil
}

// collect drains rows into domain metrics, closing rows.
func (r *MeterRepo) collect(rows pgx.Rows) ([]domain.BillableMetric, error) {
	defer rows.Close()
	var out []billableMetricRow
	for rows.Next() {
		var row billableMetricRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return billableMetricRowsToDomain(out), nil
}
