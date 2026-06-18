package postgrespgx

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// EventStore is the hand-written pgx backend for usage events. It uses the
// operational DB pool by default (the USAGE_DATABASE_URL-unset fallback); a
// separate usage DB or the ClickHouse backend can be swapped in behind the
// port.EventStore interface. This is the pgx port of the gorm EventStore and
// reproduces its SQL behaviour exactly.
type EventStore struct {
	pool *pgxpool.Pool
}

func NewEventStore(pool *pgxpool.Pool) port.EventStore {
	return &EventStore{pool: pool}
}

// Ingest writes one meter event. ON CONFLICT DO NOTHING: a resend with the same
// (org_id, external_id) hits the partial unique index and is reported as a
// duplicate (RowsAffected == 0), matching the gorm adapter.
func (s *EventStore) Ingest(ctx context.Context, e domain.MeterEvent) (port.IngestResult, error) {
	row := meterEventRowFromDomain(e)
	q := dbFromCtx(ctx, s.pool)
	tag, err := q.Exec(ctx,
		`INSERT INTO meter_events (`+meterEventColumns+`)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		 ON CONFLICT DO NOTHING`,
		row.OrgId, row.Id, row.CustomerId, row.ExternalCustomerId, row.MetricCode,
		row.SubscriptionId, row.ExternalId, row.Metadata, row.Value, row.Timestamp, row.CreatedAt)
	if err != nil {
		return port.IngestResult{}, err
	}
	if tag.RowsAffected() == 0 {
		return port.IngestResult{Id: e.Id, Status: port.IngestDuplicate}, nil
	}
	return port.IngestResult{Id: e.Id, Status: port.IngestRecorded}, nil
}

// IngestBatch writes events in one round trip via a batched INSERT, ignoring
// rows that collide with the partial unique index (resends). Mirroring the gorm
// adapter's CreateInBatches + ON CONFLICT DO NOTHING, every input event is
// reported as recorded; results align by index with events.
func (s *EventStore) IngestBatch(ctx context.Context, events []domain.MeterEvent) ([]port.IngestResult, error) {
	results := make([]port.IngestResult, len(events))
	if len(events) == 0 {
		return results, nil
	}

	var (
		sb   strings.Builder
		args = make([]any, 0, len(events)*11)
	)
	sb.WriteString(`INSERT INTO meter_events (` + meterEventColumns + `) VALUES `)
	for i, e := range events {
		row := meterEventRowFromDomain(e)
		if i > 0 {
			sb.WriteByte(',')
		}
		base := i * 11
		sb.WriteByte('(')
		for j := 0; j < 11; j++ {
			if j > 0 {
				sb.WriteByte(',')
			}
			sb.WriteByte('$')
			sb.WriteString(strconv.Itoa(base + j + 1))
		}
		sb.WriteByte(')')
		args = append(args, row.OrgId, row.Id, row.CustomerId, row.ExternalCustomerId, row.MetricCode,
			row.SubscriptionId, row.ExternalId, row.Metadata, row.Value, row.Timestamp, row.CreatedAt)
	}
	sb.WriteString(` ON CONFLICT DO NOTHING`)

	q := dbFromCtx(ctx, s.pool)
	if _, err := q.Exec(ctx, sb.String(), args...); err != nil {
		return nil, err
	}
	for i, e := range events {
		results[i] = port.IngestResult{Id: e.Id, Status: port.IngestRecorded}
	}
	return results, nil
}

// argBuf accumulates positional query args and hands back the matching $N
// placeholder for each, so dynamic WHERE clauses are built without ever
// concatenating a user value into the SQL string.
type argBuf struct {
	args []any
}

// next records v and returns its placeholder ("$1", "$2", ...).
func (a *argBuf) next(v any) string {
	a.args = append(a.args, v)
	return "$" + strconv.Itoa(len(a.args))
}

// whereClause builds the shared WHERE for q: org + metric + half-open window +
// either customer id + optional subscription attribution + optional metadata
// filter. It returns the clause (without the leading "WHERE") and the bound
// args, mirroring the gorm adapter's scope() predicate-for-predicate. Every
// placeholder it emits is backed by an appended arg, so no $N is ever unused.
func (s *EventStore) whereClause(q port.UsageQuery, ab *argBuf) string {
	var conds []string

	conds = append(conds, "org_id = "+ab.next(q.OrgId)+" AND metric_code = "+ab.next(q.MetricCode))
	conds = append(conds, "timestamp >= "+ab.next(q.From)+" AND timestamp < "+ab.next(q.To))

	// Match either customer id — but only on the ids actually provided. Matching
	// on an empty id would sweep in every row whose NULL/blank column equals it.
	switch {
	case q.CustomerId != "" && q.ExternalCustomerId != "":
		conds = append(conds, "(customer_id = "+ab.next(q.CustomerId)+" OR external_customer_id = "+ab.next(q.ExternalCustomerId)+")")
	case q.CustomerId != "":
		conds = append(conds, "customer_id = "+ab.next(q.CustomerId))
	case q.ExternalCustomerId != "":
		conds = append(conds, "external_customer_id = "+ab.next(q.ExternalCustomerId))
	}

	if q.SubscriptionId != "" {
		if q.IncludeUnattributed {
			// Unattributed events have a NULL subscription_id (absent → NULL).
			conds = append(conds, "(subscription_id = "+ab.next(q.SubscriptionId)+" OR subscription_id IS NULL)")
		} else {
			conds = append(conds, "subscription_id = "+ab.next(q.SubscriptionId))
		}
	}

	// Filter to one slice of the meter. The default/catch-all charge
	// (FilterExclude set) bills everything not explicitly priced, including
	// events where the field is absent (NULL), so unclassified usage is captured
	// exactly once. The field name is a JSON key, not a column, so it is bound as
	// a parameter to metadata->>$n (injection-safe).
	if q.FilterField != "" {
		switch {
		case len(q.FilterExclude) > 0:
			field := ab.next(q.FilterField)
			// <> ALL($n) is the parameterized form of NOT IN (...) over a list.
			conds = append(conds, "(metadata->>"+field+" <> ALL("+ab.next(q.FilterExclude)+") OR metadata->>"+field+" IS NULL)")
		case q.FilterValue != "":
			conds = append(conds, "metadata->>"+ab.next(q.FilterField)+" = "+ab.next(q.FilterValue))
		}
	}

	return strings.Join(conds, " AND ")
}

func (s *EventStore) Count(ctx context.Context, q port.UsageQuery) (int64, error) {
	ab := &argBuf{}
	where := s.whereClause(q, ab)
	q2 := dbFromCtx(ctx, s.pool)
	var n int64
	err := q2.QueryRow(ctx, `SELECT COUNT(*) FROM meter_events WHERE `+where, ab.args...).Scan(&n)
	return n, err
}

func (s *EventStore) UniqueCount(ctx context.Context, q port.UsageQuery) (int64, error) {
	ab := &argBuf{}
	where := s.whereClause(q, ab)
	field := ab.next(q.FieldName)
	q2 := dbFromCtx(ctx, s.pool)
	var n int64
	err := q2.QueryRow(ctx,
		`SELECT COUNT(DISTINCT metadata->>`+field+`) FROM meter_events WHERE `+where, ab.args...).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return n, err
}

func (s *EventStore) Sum(ctx context.Context, q port.UsageQuery) (decimal.Decimal, error) {
	return s.scalar(ctx, q, "COALESCE(SUM(value),0)")
}

func (s *EventStore) Max(ctx context.Context, q port.UsageQuery) (decimal.Decimal, error) {
	return s.scalar(ctx, q, "COALESCE(MAX(value),0)")
}

func (s *EventStore) Latest(ctx context.Context, q port.UsageQuery) (decimal.Decimal, error) {
	ab := &argBuf{}
	where := s.whereClause(q, ab)
	q2 := dbFromCtx(ctx, s.pool)
	var out decimal.Decimal
	err := q2.QueryRow(ctx,
		`SELECT value FROM meter_events WHERE `+where+` ORDER BY timestamp DESC LIMIT 1`, ab.args...).Scan(&out)
	if errors.Is(err, pgx.ErrNoRows) {
		return decimal.Zero, nil
	}
	return out, err
}

// ListHistory returns the events matching q, ordered by timestamp ascending.
func (s *EventStore) ListHistory(ctx context.Context, q port.UsageQuery) ([]domain.MeterEvent, error) {
	ab := &argBuf{}
	where := s.whereClause(q, ab)
	q2 := dbFromCtx(ctx, s.pool)
	rows, err := q2.Query(ctx,
		`SELECT `+meterEventColumns+` FROM meter_events WHERE `+where+` ORDER BY timestamp ASC`, ab.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.MeterEvent
	for rows.Next() {
		var row meterEventRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row.toDomain())
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// AggregateGrouped aggregates q partitioned by a single metadata key, one row
// per distinct value (events missing the key collapse to the empty-string /
// NULL segment). The filter in whereClause still applies, so a grouped charge
// only splits its own slice. The group key is a JSON key bound as a parameter to
// metadata->>$n; the identical placeholder is reused in SELECT and GROUP BY.
func (s *EventStore) AggregateGrouped(ctx context.Context, q port.UsageQuery, agg domain.AggregationType, groupKey string) ([]port.GroupedUsage, error) {
	ab := &argBuf{}
	where := s.whereClause(q, ab)

	keyPlaceholder := ab.next(groupKey)
	keyExpr := "metadata->>" + keyPlaceholder

	expr, err := groupedAggExpr(agg, q.FieldName, ab)
	if err != nil {
		return nil, err
	}

	sql := `SELECT ` + keyExpr + ` AS group_value, ` + expr + ` AS quantity
	        FROM meter_events WHERE ` + where + `
	        GROUP BY ` + keyExpr

	q2 := dbFromCtx(ctx, s.pool)
	rows, err := q2.Query(ctx, sql, ab.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]port.GroupedUsage, 0)
	for rows.Next() {
		var (
			groupValue *string
			quantity   decimal.Decimal
		)
		if err := rows.Scan(&groupValue, &quantity); err != nil {
			return nil, err
		}
		out = append(out, port.GroupedUsage{Key: groupKey, Value: strOrEmpty(groupValue), Quantity: quantity})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// groupedAggExpr is the SQL aggregate for a grouped query, appending any bound
// args to ab. Latest needs DISTINCT ON / window and weighted_sum is
// unimplemented even ungrouped, so both are rejected here (matching gorm).
func groupedAggExpr(agg domain.AggregationType, fieldName string, ab *argBuf) (string, error) {
	switch agg {
	case domain.AggregationCount:
		return "COUNT(*)", nil
	case domain.AggregationSum:
		return "COALESCE(SUM(value),0)", nil
	case domain.AggregationMax:
		return "COALESCE(MAX(value),0)", nil
	case domain.AggregationUniqueCount:
		return "COUNT(DISTINCT metadata->>" + ab.next(fieldName) + ")", nil
	default:
		return "", errors.New("grouped aggregation not supported: " + string(agg))
	}
}

// scalar runs a single-row aggregate expr over q, returning decimal.Zero (not an
// error) when there are no matching rows — matching the gorm adapter.
func (s *EventStore) scalar(ctx context.Context, q port.UsageQuery, expr string) (decimal.Decimal, error) {
	ab := &argBuf{}
	where := s.whereClause(q, ab)
	q2 := dbFromCtx(ctx, s.pool)
	var out decimal.Decimal
	err := q2.QueryRow(ctx, `SELECT `+expr+` FROM meter_events WHERE `+where, ab.args...).Scan(&out)
	if errors.Is(err, pgx.ErrNoRows) {
		return decimal.Zero, nil
	}
	return out, err
}
