// Package clickhouse is the ClickHouse backend for the usage-event store. It is one
// of two adapters behind port.EventStore (the other is internal/adapter/postgres);
// both must return identical aggregation results — see
// docs/internal/clickhouse-primer.md §7 and the parity harness in
// internal/adapter/compare.
//
// Parity model: Postgres dedups resends at WRITE time (partial unique index on
// (org_id, external_id) + ON CONFLICT DO NOTHING). ClickHouse keeps every insert and
// dedups at READ time via dedup_key = coalesce(nullif(external_id,”), id):
// dedup-immune aggregations (count→uniqExact, unique_count→uniqExact, max, latest→
// argMax) read straight through; sum first collapses to one value per dedup_key with
// argMax(value, ingested_at) before summing.
package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/shopspring/decimal"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// zeroDecimal is a Decimal(38,9) literal used to coalesce empty-set aggregates, so
// the result type always matches the value column (avoids a Decimal-vs-int mismatch).
const zeroDecimal = "toDecimal128(0, 9)"

// dedupKey is the read-time dedup expression: the external_id when set, else the
// (unique) event id. Resends share an external_id and collapse; events without one
// are all kept (each has a distinct id).
const dedupKey = "if(external_id != '', external_id, id)"

// EventStore implements port.EventStore against ClickHouse.
type EventStore struct {
	conn driver.Conn
}

// NewEventStore opens a ClickHouse connection from a clickhouse-go DSN
// (e.g. clickhouse://user:pass@host:9000/getpaidhq_usage) and pings it.
func NewEventStore(dsn string) (*EventStore, error) {
	opts, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: parse dsn: %w", err)
	}
	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("clickhouse: open: %w", err)
	}
	// Bound the initial connectivity check so a dead/slow host can't hang boot.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("clickhouse: ping: %w", err)
	}
	return &EventStore{conn: conn}, nil
}

// NewEventStoreWithConn wraps an existing connection (used by the parity harness so a
// test can inject a shared/created conn).
func NewEventStoreWithConn(conn driver.Conn) *EventStore {
	return &EventStore{conn: conn}
}

var _ port.EventStore = (*EventStore)(nil)

// Ingest inserts one event. Dedup is deferred to read time, so a resend is simply
// inserted again (and collapsed by dedup_key on read) — the status is always
// "recorded" here. async_insert lets the server batch single-event calls.
func (s *EventStore) Ingest(ctx context.Context, e domain.MeterEvent) (port.IngestResult, error) {
	meta := e.Metadata
	if meta == nil {
		meta = map[string]string{}
	}
	actx := clickhouse.Context(ctx, clickhouse.WithSettings(clickhouse.Settings{
		"async_insert":          1,
		"wait_for_async_insert": 1,
	}))
	err := s.conn.Exec(actx, `
		INSERT INTO meter_events
			(org_id, customer_id, external_customer_id, metric_code, subscription_id,
			 external_id, timestamp, value, metadata, id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.OrgId, e.CustomerId, e.ExternalCustomerId, e.MetricCode, e.SubscriptionId,
		e.ExternalId, e.Timestamp, e.Value, meta, e.Id,
	)
	if err != nil {
		return port.IngestResult{}, err
	}
	return port.IngestResult{Id: e.Id, Status: port.IngestRecorded}, nil
}

// IngestBatch inserts events with one native ClickHouse batch (the efficient path).
// Read-time dedup_key handles resends, so all are reported recorded.
func (s *EventStore) IngestBatch(ctx context.Context, events []domain.MeterEvent) ([]port.IngestResult, error) {
	results := make([]port.IngestResult, len(events))
	if len(events) == 0 {
		return results, nil
	}
	batch, err := s.conn.PrepareBatch(ctx, `
		INSERT INTO meter_events
			(org_id, customer_id, external_customer_id, metric_code, subscription_id,
			 external_id, timestamp, value, metadata, id)`)
	if err != nil {
		return nil, err
	}
	for _, e := range events {
		meta := e.Metadata
		if meta == nil {
			meta = map[string]string{}
		}
		if err := batch.Append(e.OrgId, e.CustomerId, e.ExternalCustomerId, e.MetricCode,
			e.SubscriptionId, e.ExternalId, e.Timestamp, e.Value, meta, e.Id); err != nil {
			return nil, err
		}
	}
	if err := batch.Send(); err != nil {
		return nil, err
	}
	for i, e := range events {
		results[i] = port.IngestResult{Id: e.Id, Status: port.IngestRecorded}
	}
	return results, nil
}

// where builds the shared filter: org + metric + half-open [from,to) window + match
// either customer id + optional subscription attribution. Returns the SQL fragment
// (without the WHERE keyword) and its positional args.
func where(q port.UsageQuery) (string, []any) {
	sql := "org_id = ? AND metric_code = ? AND timestamp >= ? AND timestamp < ?"
	args := []any{q.OrgId, q.MetricCode, q.From, q.To}
	// Match either customer id — but only on the ids actually provided, so an empty
	// id doesn't sweep in other customers' rows (which default to "").
	switch {
	case q.CustomerId != "" && q.ExternalCustomerId != "":
		sql += " AND (customer_id = ? OR external_customer_id = ?)"
		args = append(args, q.CustomerId, q.ExternalCustomerId)
	case q.CustomerId != "":
		sql += " AND customer_id = ?"
		args = append(args, q.CustomerId)
	case q.ExternalCustomerId != "":
		sql += " AND external_customer_id = ?"
		args = append(args, q.ExternalCustomerId)
	}
	if q.SubscriptionId != "" {
		if q.IncludeUnattributed {
			sql += " AND (subscription_id = ? OR subscription_id = '')"
		} else {
			sql += " AND subscription_id = ?"
		}
		args = append(args, q.SubscriptionId)
	}
	return sql, args
}

func (s *EventStore) Count(ctx context.Context, q port.UsageQuery) (int64, error) {
	w, args := where(q)
	var n uint64
	err := s.conn.QueryRow(ctx, "SELECT uniqExact("+dedupKey+") FROM meter_events WHERE "+w, args...).Scan(&n)
	return int64(n), err
}

func (s *EventStore) UniqueCount(ctx context.Context, q port.UsageQuery) (int64, error) {
	w, args := where(q)
	var n uint64
	// Distinct over the raw metadata field (string), matching the Postgres adapter.
	err := s.conn.QueryRow(ctx, "SELECT uniqExact(metadata[?]) FROM meter_events WHERE "+w, append([]any{q.FieldName}, args...)...).Scan(&n)
	return int64(n), err
}

func (s *EventStore) Sum(ctx context.Context, q port.UsageQuery) (decimal.Decimal, error) {
	w, args := where(q)
	// Collapse resends to one value per dedup_key (latest by ingested_at), then sum.
	sql := "SELECT COALESCE(SUM(v), " + zeroDecimal + ") FROM (" +
		"SELECT argMax(value, ingested_at) AS v FROM meter_events WHERE " + w +
		" GROUP BY " + dedupKey + ")"
	var out decimal.Decimal
	err := s.conn.QueryRow(ctx, sql, args...).Scan(&out)
	return out, err
}

func (s *EventStore) Max(ctx context.Context, q port.UsageQuery) (decimal.Decimal, error) {
	w, args := where(q)
	var out decimal.Decimal
	err := s.conn.QueryRow(ctx, "SELECT COALESCE(MAX(value), "+zeroDecimal+") FROM meter_events WHERE "+w, args...).Scan(&out)
	return out, err
}

func (s *EventStore) Latest(ctx context.Context, q port.UsageQuery) (decimal.Decimal, error) {
	w, args := where(q)
	var out decimal.Decimal
	err := s.conn.QueryRow(ctx, "SELECT argMax(value, timestamp) FROM meter_events WHERE "+w, args...).Scan(&out)
	return out, err
}

// WeightedSum (value averaged over time) needs a window query; deferred to parity
// with the Postgres adapter (spec phase 5 / primer §11).
func (s *EventStore) WeightedSum(ctx context.Context, q port.UsageQuery, initial decimal.Decimal) (decimal.Decimal, error) {
	return decimal.Zero, errors.New("weighted_sum aggregation not implemented")
}
