package domain

import (
	"sort"
	"time"

	"github.com/shopspring/decimal"
)

// Reserved metadata keys/values for carry-over (stock) meter events. An event
// carrying an operation is an add/remove for one identity (the meter's FieldName
// key); an event without one is a level report (a numeric value under FieldName).
// See docs/internal/billing-model/stock-billing-architecture-impact.md.
const (
	UsageOperationKey    = "operation"
	UsageOperationAdd    = "add"
	UsageOperationRemove = "remove"
)

// UsageInterval is one identity's continuous span of activity, rebuilt from a
// carry-over meter's add/remove events. A zero To means still open at read time.
type UsageInterval struct {
	Identity string
	From     time.Time
	To       time.Time
}

// HasOperations reports whether any event carries the operation key — i.e. the
// history is add/remove events rather than level reports.
func HasOperations(events []MeterEvent) bool {
	for _, e := range events {
		if e.Metadata[UsageOperationKey] != "" {
			return true
		}
	}
	return false
}

// sortedByTimestamp returns a copy of events ordered by timestamp (ingest order
// is not guaranteed).
func sortedByTimestamp(events []MeterEvent) []MeterEvent {
	out := make([]MeterEvent, len(events))
	copy(out, events)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Timestamp.Before(out[j].Timestamp) })
	return out
}

// ReconstructIntervals replays add/remove events into per-identity intervals.
// Tolerance: a duplicate add for an open identity is idempotent; a remove
// without an open interval is ignored; events missing the identity or a known
// operation are skipped. Output is sorted by (Identity, From) for determinism.
func ReconstructIntervals(events []MeterEvent, fieldName string) []UsageInterval {
	open := map[string]time.Time{}
	var out []UsageInterval
	for _, e := range sortedByTimestamp(events) {
		id := e.Metadata[fieldName]
		if id == "" {
			continue
		}
		switch e.Metadata[UsageOperationKey] {
		case UsageOperationAdd:
			if _, ok := open[id]; !ok {
				open[id] = e.Timestamp
			}
		case UsageOperationRemove:
			if from, ok := open[id]; ok {
				out = append(out, UsageInterval{Identity: id, From: from, To: e.Timestamp})
				delete(open, id)
			}
		}
	}
	for id, from := range open {
		out = append(out, UsageInterval{Identity: id, From: from})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Identity != out[j].Identity {
			return out[i].Identity < out[j].Identity
		}
		return out[i].From.Before(out[j].From)
	})
	return out
}

// overlaps reports whether the interval touches the half-open period [from, to).
func (iv UsageInterval) overlaps(from, to time.Time) bool {
	return iv.From.Before(to) && (iv.To.IsZero() || iv.To.After(from))
}

// CountStandingAtEnd counts the intervals still open at the period end (the
// `latest` reading: the level standing when the period closes).
func CountStandingAtEnd(intervals []UsageInterval, to time.Time) int64 {
	var n int64
	for _, iv := range intervals {
		if iv.From.Before(to) && (iv.To.IsZero() || !iv.To.Before(to)) {
			n++
		}
	}
	return n
}

// CountPeakConcurrent is the maximum number of simultaneously open intervals
// within [from, to) (the `max` reading). Half-open semantics: an interval ending
// the instant another starts does not overlap it.
func CountPeakConcurrent(intervals []UsageInterval, from, to time.Time) int64 {
	type edge struct {
		t time.Time
		d int64
	}
	var edges []edge
	for _, iv := range intervals {
		if !iv.overlaps(from, to) {
			continue
		}
		start, end := iv.From, iv.To
		if start.Before(from) {
			start = from
		}
		if end.IsZero() || end.After(to) {
			end = to
		}
		edges = append(edges, edge{start, 1}, edge{end, -1})
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].t.Equal(edges[j].t) {
			return edges[i].d < edges[j].d
		}
		return edges[i].t.Before(edges[j].t)
	})
	var depth, peak int64
	for _, e := range edges {
		depth += e.d
		if depth > peak {
			peak = depth
		}
	}
	return peak
}

// CountDistinctActive counts the distinct identities with any interval touching
// [from, to) (the `unique_count` reading). A re-added identity counts once.
func CountDistinctActive(intervals []UsageInterval, from, to time.Time) int64 {
	seen := map[string]bool{}
	for _, iv := range intervals {
		if iv.overlaps(from, to) {
			seen[iv.Identity] = true
		}
	}
	return int64(len(seen))
}

// WeightIntervals is the time-weighted reading (`weighted_sum`): each interval
// contributes its share of the period, Σ active-time ÷ period-length, computed
// from the timestamps as sent. The price switches reshape each interval first:
//   - prorateOnIncrease: a mid-period start accrues from its start; otherwise it
//     is treated as present from the period start (billed full).
//   - creditOnDecrease: a mid-period end stops accruing at its end; otherwise it
//     is committed to the period end (no credit).
//
// Intervals that don't touch the period bill nothing regardless of switches.
func WeightIntervals(intervals []UsageInterval, from, to time.Time, prorateOnIncrease, creditOnDecrease bool) decimal.Decimal {
	period := to.Sub(from)
	if period <= 0 {
		return decimal.Zero
	}
	total := decimal.Zero
	for _, iv := range intervals {
		if !iv.overlaps(from, to) {
			continue
		}
		start, end := from, to
		if prorateOnIncrease && iv.From.After(from) {
			start = iv.From
		}
		if creditOnDecrease && !iv.To.IsZero() && iv.To.Before(to) {
			end = iv.To
		}
		if d := end.Sub(start); d > 0 {
			total = total.Add(timeFraction(d, period))
		}
	}
	return total
}

// LastReportedLevel is the `latest` reading over level reports: the value of the
// last report, zero if there are none. The caller fetches history bounded at the
// period end, so the last report is the level standing when the period closes.
func LastReportedLevel(events []MeterEvent) decimal.Decimal {
	sorted := sortedByTimestamp(events)
	if len(sorted) == 0 {
		return decimal.Zero
	}
	return sorted[len(sorted)-1].Value
}

// PeakReportedLevel is the `max` reading over level reports: the highest level in
// force during [from, to) — the last value reported before the period start, or
// any value reported inside the period.
func PeakReportedLevel(events []MeterEvent, from, to time.Time) decimal.Decimal {
	peak := decimal.Zero
	standing := decimal.Zero
	for _, e := range sortedByTimestamp(events) {
		switch {
		case e.Timestamp.After(to) || e.Timestamp.Equal(to):
			// past the period; standing/peak are settled
		case e.Timestamp.After(from):
			if e.Value.GreaterThan(peak) {
				peak = e.Value
			}
		default:
			standing = e.Value
		}
	}
	if standing.GreaterThan(peak) {
		return standing
	}
	return peak
}

// WeightReportedLevels is the `weighted_sum` reading over level reports: the
// average level across [from, to), each reported value weighted by how long it
// was in force, computed from the timestamps as sent.
func WeightReportedLevels(events []MeterEvent, from, to time.Time) decimal.Decimal {
	period := to.Sub(from)
	if period <= 0 {
		return decimal.Zero
	}
	level := decimal.Zero
	cursor := from
	total := decimal.Zero
	for _, e := range sortedByTimestamp(events) {
		switch {
		case !e.Timestamp.After(from):
			level = e.Value // the value in force when the period starts
		case e.Timestamp.Before(to):
			total = total.Add(level.Mul(timeFraction(e.Timestamp.Sub(cursor), period)))
			level = e.Value
			cursor = e.Timestamp
		}
	}
	return total.Add(level.Mul(timeFraction(to.Sub(cursor), period)))
}

// timeFraction is d ÷ period as a decimal.
func timeFraction(d, period time.Duration) decimal.Decimal {
	return decimal.NewFromInt(int64(d)).Div(decimal.NewFromInt(int64(period)))
}
