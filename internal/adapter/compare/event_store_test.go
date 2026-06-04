package compare

import (
	"context"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// memStore is an in-memory reference EventStore that implements the SAME observable
// semantics both real adapters must produce: read-time dedup by
// dedup_key = external_id (if set) else id, half-open [from,to) window, customer-OR
// match, and optional subscription attribution. The parity harness runs the same
// event set + queries through any two port.EventStore implementations and asserts
// they agree; running it against two memStores proves the harness and the reference
// are self-consistent, and it is the body a real Postgres-vs-ClickHouse run reuses.
type memStore struct {
	mu     sync.Mutex
	events []domain.MeterEvent // insertion order = ingest order (later wins on dedup)
}

func newMemStore() *memStore { return &memStore{} }

var _ port.EventStore = (*memStore)(nil)

func (m *memStore) Ingest(_ context.Context, e domain.MeterEvent) (port.IngestResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, e)
	return port.IngestResult{Id: e.Id}, nil
}

func dedupKeyOf(e domain.MeterEvent) string {
	if e.ExternalId != "" {
		return e.ExternalId
	}
	return e.Id
}

// scoped returns the events matching q, deduped to one per dedup_key (latest ingest
// wins). order is preserved by dedup-key first-seen for deterministic iteration.
func (m *memStore) scoped(q port.UsageQuery) []domain.MeterEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	latest := map[string]domain.MeterEvent{}
	for _, e := range m.events {
		if e.OrgId != q.OrgId || e.MetricCode != q.MetricCode {
			continue
		}
		if e.Timestamp.Before(q.From) || !e.Timestamp.Before(q.To) {
			continue
		}
		if !(e.CustomerId == q.CustomerId || e.ExternalCustomerId == q.ExternalCustomerId) {
			continue
		}
		if q.SubscriptionId != "" {
			if q.IncludeUnattributed {
				if e.SubscriptionId != q.SubscriptionId && e.SubscriptionId != "" {
					continue
				}
			} else if e.SubscriptionId != q.SubscriptionId {
				continue
			}
		}
		latest[dedupKeyOf(e)] = e // later ingest overwrites
	}
	out := make([]domain.MeterEvent, 0, len(latest))
	for _, e := range latest {
		out = append(out, e)
	}
	return out
}

func (m *memStore) Count(_ context.Context, q port.UsageQuery) (int64, error) {
	return int64(len(m.scoped(q))), nil
}

func (m *memStore) UniqueCount(_ context.Context, q port.UsageQuery) (int64, error) {
	seen := map[string]struct{}{}
	for _, e := range m.scoped(q) {
		seen[e.Metadata[q.FieldName]] = struct{}{}
	}
	return int64(len(seen)), nil
}

func (m *memStore) Sum(_ context.Context, q port.UsageQuery) (decimal.Decimal, error) {
	sum := decimal.Zero
	for _, e := range m.scoped(q) {
		sum = sum.Add(e.Value)
	}
	return sum, nil
}

func (m *memStore) Max(_ context.Context, q port.UsageQuery) (decimal.Decimal, error) {
	max := decimal.Zero
	for i, e := range m.scoped(q) {
		if i == 0 || e.Value.GreaterThan(max) {
			max = e.Value
		}
	}
	return max, nil
}

func (m *memStore) Latest(_ context.Context, q port.UsageQuery) (decimal.Decimal, error) {
	events := m.scoped(q)
	if len(events) == 0 {
		return decimal.Zero, nil
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].Timestamp.Before(events[j].Timestamp) })
	return events[len(events)-1].Value, nil
}

func (m *memStore) WeightedSum(_ context.Context, _ port.UsageQuery, _ decimal.Decimal) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

// --- parity harness ---

func dec(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}

// parityEvents is a deliberately tricky set: resends sharing an external_id (must
// collapse), an event on the [from] boundary (included), one on the [to] boundary
// (excluded), an out-of-window event, an unattributed event, and a different
// customer's event (must never leak in).
func parityEvents(from time.Time) []domain.MeterEvent {
	org := "org_1"
	return []domain.MeterEvent{
		{OrgId: org, Id: "e1", CustomerId: "cus_1", MetricCode: "api_calls", SubscriptionId: "sub_1", ExternalId: "x1", Value: dec("10"), Metadata: map[string]string{"region": "eu"}, Timestamp: from},                          // on [from] boundary -> in
		{OrgId: org, Id: "e2", CustomerId: "cus_1", MetricCode: "api_calls", SubscriptionId: "sub_1", ExternalId: "x1", Value: dec("10"), Metadata: map[string]string{"region": "eu"}, Timestamp: from.Add(time.Minute)},          // resend of x1 -> collapses
		{OrgId: org, Id: "e3", CustomerId: "cus_1", MetricCode: "api_calls", SubscriptionId: "sub_1", ExternalId: "x2", Value: dec("25"), Metadata: map[string]string{"region": "us"}, Timestamp: from.Add(2 * time.Hour)},        // distinct
		{OrgId: org, Id: "e4", CustomerId: "cus_1", MetricCode: "api_calls", SubscriptionId: "", ExternalId: "x3", Value: dec("5"), Metadata: map[string]string{"region": "eu"}, Timestamp: from.Add(3 * time.Hour)},              // unattributed
		{OrgId: org, Id: "e5", CustomerId: "cus_1", MetricCode: "api_calls", SubscriptionId: "sub_1", ExternalId: "x4", Value: dec("100"), Metadata: map[string]string{"region": "ap"}, Timestamp: from.Add(24 * time.Hour)},      // out of window (to is +12h below)
		{OrgId: org, Id: "e6", CustomerId: "cus_2", MetricCode: "api_calls", SubscriptionId: "sub_9", ExternalId: "y1", Value: dec("999"), Metadata: map[string]string{"region": "eu"}, Timestamp: from.Add(time.Hour)},           // other customer
		{OrgId: org, Id: "e7", CustomerId: "cus_1", MetricCode: "storage_gb", SubscriptionId: "sub_1", ExternalId: "z1", Value: dec("7"), Metadata: map[string]string{"region": "eu"}, Timestamp: from.Add(time.Hour)},            // other metric
	}
}

// runParity feeds the same events into both stores and asserts every aggregation
// agrees across a battery of queries. Reused by a future Postgres-vs-ClickHouse run
// (USAGE_EVENT_STORE=compare against a live ClickHouse — not runnable in CI).
func runParity(t *testing.T, a, b port.EventStore) {
	t.Helper()
	ctx := context.Background()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(12 * time.Hour)
	for _, e := range parityEvents(from) {
		if _, err := a.Ingest(ctx, e); err != nil {
			t.Fatalf("a.Ingest: %v", err)
		}
		if _, err := b.Ingest(ctx, e); err != nil {
			t.Fatalf("b.Ingest: %v", err)
		}
	}

	queries := map[string]port.UsageQuery{
		"attributed":   {OrgId: "org_1", MetricCode: "api_calls", FieldName: "region", From: from, To: to, CustomerId: "cus_1", SubscriptionId: "sub_1"},
		"with_unattr":  {OrgId: "org_1", MetricCode: "api_calls", FieldName: "region", From: from, To: to, CustomerId: "cus_1", SubscriptionId: "sub_1", IncludeUnattributed: true},
		"all_customer": {OrgId: "org_1", MetricCode: "api_calls", FieldName: "region", From: from, To: to, CustomerId: "cus_1"},
		"empty":        {OrgId: "org_1", MetricCode: "api_calls", FieldName: "region", From: from, To: to, CustomerId: "nobody"},
	}

	for name, q := range queries {
		q := q
		t.Run(name, func(t *testing.T) {
			ac, _ := a.Count(ctx, q)
			bc, _ := b.Count(ctx, q)
			if ac != bc {
				t.Errorf("Count mismatch: a=%d b=%d", ac, bc)
			}
			au, _ := a.UniqueCount(ctx, q)
			bu, _ := b.UniqueCount(ctx, q)
			if au != bu {
				t.Errorf("UniqueCount mismatch: a=%d b=%d", au, bu)
			}
			as, _ := a.Sum(ctx, q)
			bs, _ := b.Sum(ctx, q)
			if !as.Equal(bs) {
				t.Errorf("Sum mismatch: a=%s b=%s", as, bs)
			}
			am, _ := a.Max(ctx, q)
			bm, _ := b.Max(ctx, q)
			if !am.Equal(bm) {
				t.Errorf("Max mismatch: a=%s b=%s", am, bm)
			}
			al, _ := a.Latest(ctx, q)
			bl, _ := b.Latest(ctx, q)
			if !al.Equal(bl) {
				t.Errorf("Latest mismatch: a=%s b=%s", al, bl)
			}
		})
	}
}

func TestParity_ReferenceSelfConsistent(t *testing.T) {
	runParity(t, newMemStore(), newMemStore())
}

// TestParity_KnownValues pins the reference's numbers so the harness can't pass by
// two stores agreeing on a wrong answer.
func TestParity_KnownValues(t *testing.T) {
	ctx := context.Background()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(12 * time.Hour)
	s := newMemStore()
	for _, e := range parityEvents(from) {
		s.Ingest(ctx, e)
	}
	// Attributed to sub_1, in window: x1 (deduped, 10) + x2 (25). Not x3 (unattributed),
	// not x4 (out of window), not cus_2, not storage_gb.
	q := port.UsageQuery{OrgId: "org_1", MetricCode: "api_calls", FieldName: "region", From: from, To: to, CustomerId: "cus_1", SubscriptionId: "sub_1"}
	if n, _ := s.Count(ctx, q); n != 2 {
		t.Errorf("Count = %d, want 2", n)
	}
	if v, _ := s.Sum(ctx, q); !v.Equal(dec("35")) {
		t.Errorf("Sum = %s, want 35", v)
	}
	if v, _ := s.Max(ctx, q); !v.Equal(dec("25")) {
		t.Errorf("Max = %s, want 25", v)
	}
	if n, _ := s.UniqueCount(ctx, q); n != 2 { // regions eu, us
		t.Errorf("UniqueCount = %d, want 2", n)
	}
	// With unattributed: + x3 (5) -> count 3, sum 40, regions still {eu,us} = 2.
	qu := q
	qu.IncludeUnattributed = true
	if n, _ := s.Count(ctx, qu); n != 3 {
		t.Errorf("Count(unattr) = %d, want 3", n)
	}
	if v, _ := s.Sum(ctx, qu); !v.Equal(dec("40")) {
		t.Errorf("Sum(unattr) = %s, want 40", v)
	}
}

// capLogger records Warn/Error calls so the compare-wrapper test can assert a
// mismatch was surfaced.
type capLogger struct {
	mu    sync.Mutex
	warns int
	errs  int
}

func (l *capLogger) bump(warn bool) {
	l.mu.Lock()
	if warn {
		l.warns++
	} else {
		l.errs++
	}
	l.mu.Unlock()
}
func (l *capLogger) count() (int, int) { l.mu.Lock(); defer l.mu.Unlock(); return l.warns, l.errs }

func (l *capLogger) Debug(string, ...any)          {}
func (l *capLogger) Info(string, ...any)           {}
func (l *capLogger) Warn(string, ...any)           { l.bump(true) }
func (l *capLogger) Error(string, ...any)          { l.bump(false) }
func (l *capLogger) Fatal(string, ...any)          {}
func (l *capLogger) Debugf(string, ...any)         {}
func (l *capLogger) Infof(string, ...any)          {}
func (l *capLogger) Warnf(string, ...any)          {}
func (l *capLogger) Errorf(string, ...any)         {}
func (l *capLogger) Panicf(string, ...any)         {}
func (l *capLogger) Fatalf(string, ...any)         {}
func (l *capLogger) Sync() error                   { return nil }

// skewStore is a memStore whose Sum is deliberately wrong, to prove the compare
// wrapper (a) still serves the primary's correct value and (b) logs the mismatch.
type skewStore struct{ *memStore }

func (s skewStore) Sum(ctx context.Context, q port.UsageQuery) (decimal.Decimal, error) {
	v, err := s.memStore.Sum(ctx, q)
	return v.Add(dec("1")), err
}

func TestCompare_ServesPrimaryAndLogsMismatch(t *testing.T) {
	ctx := context.Background()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(12 * time.Hour)

	primary := newMemStore()
	secondary := skewStore{newMemStore()}
	log := &capLogger{}
	cs := NewEventStore(primary, secondary, log)

	for _, e := range parityEvents(from) {
		cs.Ingest(ctx, e)
	}

	q := port.UsageQuery{OrgId: "org_1", MetricCode: "api_calls", FieldName: "region", From: from, To: to, CustomerId: "cus_1", SubscriptionId: "sub_1"}
	got, err := cs.Sum(ctx, q)
	if err != nil {
		t.Fatalf("Sum: %v", err)
	}
	// Serves the PRIMARY (correct) value, not the skewed secondary.
	if !got.Equal(dec("35")) {
		t.Errorf("Sum served = %s, want primary 35", got)
	}

	// The background check logs a mismatch. Poll briefly for the goroutine.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if w, _ := log.count(); w > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	w, e := log.count()
	t.Errorf("expected a mismatch warning, got warns=%d errs=%d", w, e)
}
