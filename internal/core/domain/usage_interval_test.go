package domain

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// The June timeline from docs/internal/billing-model/seat-billing/README.md:
// alice, bob, carol seated since May 20; dave joins Jun 16; bob leaves Jun 21.
var (
	jun1  = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	jul1  = time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	may20 = time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)
	jun16 = time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC)
	jun21 = time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC)
)

func seatEvent(op, seat string, ts time.Time) MeterEvent {
	return MeterEvent{
		MetricCode: "seats",
		Metadata:   map[string]string{UsageOperationKey: op, "seat_id": seat},
		Timestamp:  ts,
	}
}

func juneLedger() []MeterEvent {
	return []MeterEvent{
		seatEvent(UsageOperationAdd, "alice", may20),
		seatEvent(UsageOperationAdd, "bob", may20),
		seatEvent(UsageOperationAdd, "carol", may20),
		seatEvent(UsageOperationAdd, "dave", jun16),
		seatEvent(UsageOperationRemove, "bob", jun21),
	}
}

func levelReport(v string, ts time.Time) MeterEvent {
	return MeterEvent{
		MetricCode: "seats",
		Metadata:   map[string]string{"count": v},
		Value:      decimal.RequireFromString(v),
		Timestamp:  ts,
	}
}

func TestHasOperations(t *testing.T) {
	assert.True(t, HasOperations(juneLedger()))
	assert.False(t, HasOperations([]MeterEvent{levelReport("3", may20)}))
	assert.False(t, HasOperations(nil))
}

func TestReconstructIntervals_JuneTimeline(t *testing.T) {
	got := ReconstructIntervals(juneLedger(), "seat_id")
	want := []UsageInterval{
		{Identity: "alice", From: may20},
		{Identity: "bob", From: may20, To: jun21},
		{Identity: "carol", From: may20},
		{Identity: "dave", From: jun16},
	}
	assert.Equal(t, want, got)
}

func TestReconstructIntervals_Tolerance(t *testing.T) {
	t.Run("duplicate add is idempotent", func(t *testing.T) {
		got := ReconstructIntervals([]MeterEvent{
			seatEvent(UsageOperationAdd, "a", may20),
			seatEvent(UsageOperationAdd, "a", jun16),
		}, "seat_id")
		assert.Equal(t, []UsageInterval{{Identity: "a", From: may20}}, got)
	})
	t.Run("remove without open interval is ignored", func(t *testing.T) {
		got := ReconstructIntervals([]MeterEvent{
			seatEvent(UsageOperationRemove, "ghost", jun16),
		}, "seat_id")
		assert.Empty(t, got)
	})
	t.Run("re-add after remove opens a second interval", func(t *testing.T) {
		got := ReconstructIntervals([]MeterEvent{
			seatEvent(UsageOperationAdd, "a", may20),
			seatEvent(UsageOperationRemove, "a", jun16),
			seatEvent(UsageOperationAdd, "a", jun21),
		}, "seat_id")
		want := []UsageInterval{
			{Identity: "a", From: may20, To: jun16},
			{Identity: "a", From: jun21},
		}
		assert.Equal(t, want, got)
	})
	t.Run("out-of-order events are sorted before replay", func(t *testing.T) {
		got := ReconstructIntervals([]MeterEvent{
			seatEvent(UsageOperationRemove, "a", jun21),
			seatEvent(UsageOperationAdd, "a", may20),
		}, "seat_id")
		assert.Equal(t, []UsageInterval{{Identity: "a", From: may20, To: jun21}}, got)
	})
	t.Run("events missing identity or operation are skipped", func(t *testing.T) {
		got := ReconstructIntervals([]MeterEvent{
			{Metadata: map[string]string{UsageOperationKey: UsageOperationAdd}, Timestamp: may20},
			{Metadata: map[string]string{"seat_id": "a"}, Timestamp: may20},
		}, "seat_id")
		assert.Empty(t, got)
	})
}

func TestCounts_JuneTimeline(t *testing.T) {
	intervals := ReconstructIntervals(juneLedger(), "seat_id")
	assert.Equal(t, int64(3), CountStandingAtEnd(intervals, jul1), "alice, carol, dave")
	assert.Equal(t, int64(4), CountPeakConcurrent(intervals, jun1, jul1), "all four overlap Jun 16-21")
	assert.Equal(t, int64(4), CountDistinctActive(intervals, jun1, jul1), "anyone active at any point")
}

func TestCounts_Boundaries(t *testing.T) {
	t.Run("interval closed exactly at period start does not count", func(t *testing.T) {
		iv := []UsageInterval{{Identity: "a", From: may20, To: jun1}}
		assert.Equal(t, int64(0), CountDistinctActive(iv, jun1, jul1))
		assert.Equal(t, int64(0), CountPeakConcurrent(iv, jun1, jul1))
	})
	t.Run("re-added identity counts once for distinct", func(t *testing.T) {
		iv := []UsageInterval{
			{Identity: "a", From: may20, To: jun16},
			{Identity: "a", From: jun21},
		}
		assert.Equal(t, int64(1), CountDistinctActive(iv, jun1, jul1))
	})
	t.Run("back-to-back intervals do not overlap for peak", func(t *testing.T) {
		iv := []UsageInterval{
			{Identity: "a", From: jun1, To: jun16},
			{Identity: "b", From: jun16, To: jul1},
		}
		assert.Equal(t, int64(1), CountPeakConcurrent(iv, jun1, jul1))
	})
	t.Run("interval removed mid-period does not stand at end", func(t *testing.T) {
		iv := []UsageInterval{{Identity: "a", From: jun1, To: jun21}}
		assert.Equal(t, int64(0), CountStandingAtEnd(iv, jul1))
	})
}

// The switch table from seat-billing/mapping.md on the June timeline. Midnight
// timestamps make the exact-time fractions whole-day fractions.
func TestWeightIntervals_SwitchTable(t *testing.T) {
	intervals := ReconstructIntervals(juneLedger(), "seat_id")
	cases := []struct {
		name              string
		prorateOnIncrease bool
		creditOnDecrease  bool
		want              string // rounded to 2dp
	}{
		{"B time-weighted", true, true, "3.17"},  // 1 + 1 + 20/30 + 15/30
		{"C hybrid", true, false, "3.5"},         // bob committed to period end
		{"degenerate", false, false, "4"},        // = distinct count (consistency check)
		{"credit only", false, true, "3.67"},     // 1 + 1 + 20/30 + 1
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := WeightIntervals(intervals, jun1, jul1, tc.prorateOnIncrease, tc.creditOnDecrease)
			assert.True(t, got.Round(2).Equal(decimal.RequireFromString(tc.want)),
				"got %s, want %s", got.Round(2), tc.want)
		})
	}
}

func TestWeightIntervals_Boundaries(t *testing.T) {
	t.Run("interval entirely before the period bills zero even without credit", func(t *testing.T) {
		iv := []UsageInterval{{Identity: "a", From: may20, To: time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)}}
		assert.True(t, WeightIntervals(iv, jun1, jul1, true, false).IsZero())
	})
	t.Run("zero-length period bills zero", func(t *testing.T) {
		iv := []UsageInterval{{Identity: "a", From: may20}}
		assert.True(t, WeightIntervals(iv, jun1, jun1, true, true).IsZero())
	})
}

// Level-report history: 3 seats since May 20, 4 from Jun 16, 3 from Jun 21.
func juneReports() []MeterEvent {
	return []MeterEvent{
		levelReport("3", may20),
		levelReport("4", jun16),
		levelReport("3", jun21),
	}
}

func TestLevelReports_June(t *testing.T) {
	assert.True(t, LastReportedLevel(juneReports()).Equal(decimal.NewFromInt(3)))
	assert.True(t, PeakReportedLevel(juneReports(), jun1, jul1).Equal(decimal.NewFromInt(4)))

	// Average level: 4 in force Jun 16-21 (5 days), 3 the other 25 days:
	// (3×15 + 4×5 + 3×10) / 30 = 95/30 = 3.1667
	got := WeightReportedLevels(juneReports(), jun1, jul1)
	assert.True(t, got.Round(2).Equal(decimal.RequireFromString("3.17")), "got %s", got.Round(2))
}

func TestLevelReports_StandingValueFromBeforePeriod(t *testing.T) {
	// 5 reported in May, 2 reported Jun 10 — the May value stands until Jun 10.
	reports := []MeterEvent{
		levelReport("5", time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)),
		levelReport("2", time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)),
	}
	assert.True(t, LastReportedLevel(reports).Equal(decimal.NewFromInt(2)))
	assert.True(t, PeakReportedLevel(reports, jun1, jul1).Equal(decimal.NewFromInt(5)),
		"the value standing at period start beats the in-period max")

	// (5×9 + 2×21) / 30 = 87/30 = 2.9
	got := WeightReportedLevels(reports, jun1, jul1)
	assert.True(t, got.Equal(decimal.RequireFromString("2.9")), "got %s", got)
}

func TestLevelReports_Empty(t *testing.T) {
	assert.True(t, LastReportedLevel(nil).IsZero())
	assert.True(t, PeakReportedLevel(nil, jun1, jul1).IsZero())
	assert.True(t, WeightReportedLevels(nil, jun1, jul1).IsZero())
}
