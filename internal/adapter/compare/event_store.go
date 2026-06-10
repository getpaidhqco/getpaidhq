// Package compare wraps two port.EventStore backends (primary + secondary) so the
// two can be checked against each other on real data without affecting billing.
// Every read serves the PRIMARY result; the SECONDARY runs in the background, and any
// mismatch or error is logged with both timings. Ingest writes to both (primary
// synchronously — its result is returned; secondary best-effort, errors logged).
//
// Wired when USAGE_EVENT_STORE=compare. The intended use is primary=Postgres,
// secondary=ClickHouse: serve the proven backend, measure and verify the new one
// before committing. See docs/internal/clickhouse-primer.md §7.
package compare

import (
	"context"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// maxConcurrentChecks bounds in-flight background comparisons. Each read spawns one
// goroutine; under sustained read load an unbounded spawn would pile up secondary
// queries. When the bound is reached, further checks are skipped (and counted) rather
// than queued — compare is a diagnostic mode, not a hot path.
const maxConcurrentChecks = 8

// EventStore is the comparing wrapper.
type EventStore struct {
	primary   port.EventStore
	secondary port.EventStore
	logger    port.Logger
	// timeout bounds each background secondary read so a slow/hung secondary never
	// leaks goroutines. The check is detached from the caller's context.
	timeout time.Duration
	// sem caps concurrent background checks (non-blocking acquire; skip when full).
	sem chan struct{}
}

func NewEventStore(primary, secondary port.EventStore, logger port.Logger) *EventStore {
	return &EventStore{
		primary:   primary,
		secondary: secondary,
		logger:    logger,
		timeout:   30 * time.Second,
		sem:       make(chan struct{}, maxConcurrentChecks),
	}
}

// acquire takes a slot without blocking; false means the check should be skipped
// because too many comparisons are already in flight.
func (s *EventStore) acquire() bool {
	select {
	case s.sem <- struct{}{}:
		return true
	default:
		s.logger.Debug("compare: check skipped (max concurrent checks reached)")
		return false
	}
}

func (s *EventStore) release() { <-s.sem }

var _ port.EventStore = (*EventStore)(nil)

// Ingest writes to both. The primary's result is authoritative; a secondary error is
// logged, not returned (the secondary must never block or fail ingestion).
func (s *EventStore) Ingest(ctx context.Context, e domain.MeterEvent) (port.IngestResult, error) {
	res, err := s.primary.Ingest(ctx, e)
	if err != nil {
		return res, err
	}
	if _, serr := s.secondary.Ingest(ctx, e); serr != nil {
		s.logger.Error("compare: secondary ingest failed", "event_id", e.Id, "err", serr.Error())
	}
	return res, nil
}

// IngestBatch writes to both; the primary's results are authoritative.
func (s *EventStore) IngestBatch(ctx context.Context, events []domain.MeterEvent) ([]port.IngestResult, error) {
	res, err := s.primary.IngestBatch(ctx, events)
	if err != nil {
		return res, err
	}
	if _, serr := s.secondary.IngestBatch(ctx, events); serr != nil {
		s.logger.Error("compare: secondary batch ingest failed", "count", len(events), "err", serr.Error())
	}
	return res, nil
}

func (s *EventStore) Count(ctx context.Context, q port.UsageQuery) (int64, error) {
	prim, dur := timeInt(func() (int64, error) { return s.primary.Count(ctx, q) })
	s.checkInt(q, "count", prim.v, prim.err, dur, func(c context.Context) (int64, error) { return s.secondary.Count(c, q) })
	return prim.v, prim.err
}

func (s *EventStore) UniqueCount(ctx context.Context, q port.UsageQuery) (int64, error) {
	prim, dur := timeInt(func() (int64, error) { return s.primary.UniqueCount(ctx, q) })
	s.checkInt(q, "unique_count", prim.v, prim.err, dur, func(c context.Context) (int64, error) { return s.secondary.UniqueCount(c, q) })
	return prim.v, prim.err
}

func (s *EventStore) Sum(ctx context.Context, q port.UsageQuery) (decimal.Decimal, error) {
	prim, dur := timeDec(func() (decimal.Decimal, error) { return s.primary.Sum(ctx, q) })
	s.checkDec(q, "sum", prim.v, prim.err, dur, func(c context.Context) (decimal.Decimal, error) { return s.secondary.Sum(c, q) })
	return prim.v, prim.err
}

func (s *EventStore) Max(ctx context.Context, q port.UsageQuery) (decimal.Decimal, error) {
	prim, dur := timeDec(func() (decimal.Decimal, error) { return s.primary.Max(ctx, q) })
	s.checkDec(q, "max", prim.v, prim.err, dur, func(c context.Context) (decimal.Decimal, error) { return s.secondary.Max(c, q) })
	return prim.v, prim.err
}

func (s *EventStore) Latest(ctx context.Context, q port.UsageQuery) (decimal.Decimal, error) {
	prim, dur := timeDec(func() (decimal.Decimal, error) { return s.primary.Latest(ctx, q) })
	s.checkDec(q, "latest", prim.v, prim.err, dur, func(c context.Context) (decimal.Decimal, error) { return s.secondary.Latest(c, q) })
	return prim.v, prim.err
}

// ListHistory serves the primary only — a row-level fetch has no scalar result to
// compare in the background.
func (s *EventStore) ListHistory(ctx context.Context, q port.UsageQuery) ([]domain.MeterEvent, error) {
	return s.primary.ListHistory(ctx, q)
}

func (s *EventStore) AggregateGrouped(ctx context.Context, q port.UsageQuery, agg domain.AggregationType, groupKey string) ([]port.GroupedUsage, error) {
	start := time.Now()
	primV, primErr := s.primary.AggregateGrouped(ctx, q, agg, groupKey)
	dur := time.Since(start)
	s.checkGrouped(q, "aggregate_grouped", primV, primErr, dur, func(c context.Context) ([]port.GroupedUsage, error) {
		return s.secondary.AggregateGrouped(c, q, agg, groupKey)
	})
	return primV, primErr
}

// checkGrouped is the grouped-aggregation analogue of checkDec: it compares the two
// backends' segment sets order-independently and logs any mismatch in the background.
func (s *EventStore) checkGrouped(q port.UsageQuery, op string, primV []port.GroupedUsage, primErr error, primDur time.Duration, sec func(context.Context) ([]port.GroupedUsage, error)) {
	if !s.acquire() {
		return
	}
	go func() {
		defer s.release()
		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
		defer cancel()
		start := time.Now()
		secV, secErr := sec(ctx)
		secDur := time.Since(start)
		if primErr != nil || secErr != nil {
			s.logErrs(op, q, primErr, secErr)
			return
		}
		if !groupedEqual(primV, secV) {
			s.logger.Warn("compare: mismatch",
				"op", op, "metric", q.MetricCode, "org", q.OrgId,
				"primary", groupedString(primV), "secondary", groupedString(secV),
				"primary_ms", primDur.Milliseconds(), "secondary_ms", secDur.Milliseconds())
			return
		}
		s.logger.Debug("compare: match grouped", "op", op, "metric", q.MetricCode,
			"primary_ms", primDur.Milliseconds(), "secondary_ms", secDur.Milliseconds())
	}()
}

// groupedEqual reports whether two grouped results have the same value→quantity set,
// independent of row order.
func groupedEqual(a, b []port.GroupedUsage) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[string]decimal.Decimal, len(a))
	for _, g := range a {
		m[g.Value] = g.Quantity
	}
	for _, g := range b {
		v, ok := m[g.Value]
		if !ok || !v.Equal(g.Quantity) {
			return false
		}
	}
	return true
}

func groupedString(g []port.GroupedUsage) string {
	parts := make([]string, len(g))
	for i, x := range g {
		parts[i] = x.Value + "=" + x.Quantity.String()
	}
	return strings.Join(parts, ",")
}

// --- background comparison plumbing ---

type intResult struct {
	v   int64
	err error
}

type decResult struct {
	v   decimal.Decimal
	err error
}

func timeInt(f func() (int64, error)) (intResult, time.Duration) {
	start := time.Now()
	v, err := f()
	return intResult{v, err}, time.Since(start)
}

func timeDec(f func() (decimal.Decimal, error)) (decResult, time.Duration) {
	start := time.Now()
	v, err := f()
	return decResult{v, err}, time.Since(start)
}

func (s *EventStore) checkInt(q port.UsageQuery, op string, primV int64, primErr error, primDur time.Duration, sec func(context.Context) (int64, error)) {
	if !s.acquire() {
		return
	}
	go func() {
		defer s.release()
		// Detached from the caller's context (the request may finish first) and
		// bounded by timeout so the goroutine always exits.
		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
		defer cancel()
		start := time.Now()
		secV, secErr := sec(ctx)
		secDur := time.Since(start)
		if primErr != nil || secErr != nil {
			s.logErrs(op, q, primErr, secErr)
			return
		}
		if primV != secV {
			s.logger.Warn("compare: mismatch",
				"op", op, "metric", q.MetricCode, "org", q.OrgId,
				"primary", primV, "secondary", secV,
				"primary_ms", primDur.Milliseconds(), "secondary_ms", secDur.Milliseconds())
			return
		}
		s.logger.Debug("compare: match", "op", op, "metric", q.MetricCode,
			"primary_ms", primDur.Milliseconds(), "secondary_ms", secDur.Milliseconds())
	}()
}

func (s *EventStore) checkDec(q port.UsageQuery, op string, primV decimal.Decimal, primErr error, primDur time.Duration, sec func(context.Context) (decimal.Decimal, error)) {
	if !s.acquire() {
		return
	}
	go func() {
		defer s.release()
		// Detached from the caller's context (the request may finish first) and
		// bounded by timeout so the goroutine always exits.
		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
		defer cancel()
		start := time.Now()
		secV, secErr := sec(ctx)
		secDur := time.Since(start)
		if primErr != nil || secErr != nil {
			s.logErrs(op, q, primErr, secErr)
			return
		}
		if !primV.Equal(secV) {
			s.logger.Warn("compare: mismatch",
				"op", op, "metric", q.MetricCode, "org", q.OrgId,
				"primary", primV.String(), "secondary", secV.String(),
				"primary_ms", primDur.Milliseconds(), "secondary_ms", secDur.Milliseconds())
			return
		}
		s.logger.Debug("compare: match", "op", op, "metric", q.MetricCode,
			"primary_ms", primDur.Milliseconds(), "secondary_ms", secDur.Milliseconds())
	}()
}

func (s *EventStore) logErrs(op string, q port.UsageQuery, primErr, secErr error) {
	var p, sec string
	if primErr != nil {
		p = primErr.Error()
	}
	if secErr != nil {
		sec = secErr.Error()
	}
	s.logger.Error("compare: read error", "op", op, "metric", q.MetricCode,
		"primary_err", p, "secondary_err", sec)
}
