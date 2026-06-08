package service

import (
	"testing"

	"github.com/shopspring/decimal"

	"getpaidhq/internal/core/domain"
)

// TestApplyRounding exhaustively pins the meter rounding applied to an aggregated
// quantity before it becomes billable units. Runs in the default (non-integration)
// suite, so rounding is covered even without a database.
func TestApplyRounding(t *testing.T) {
	cases := []struct {
		name  string
		mode  string
		scale int
		in    string
		want  string
	}{
		// No rounding mode → value passes through untouched, scale ignored.
		{"none passthrough", "", 0, "2.94", "2.94"},
		{"unknown mode passthrough", "bogus", 0, "2.94", "2.94"},

		// floor: toward negative infinity at the given scale.
		{"floor scale 0", "floor", 0, "2.9", "2"},
		{"floor scale 1", "floor", 1, "2.94", "2.9"},
		{"floor exact", "floor", 0, "3", "3"},

		// ceil: toward positive infinity at the given scale.
		{"ceil scale 0", "ceil", 0, "2.1", "3"},
		{"ceil scale 1", "ceil", 1, "2.91", "3.0"},
		{"ceil exact", "ceil", 0, "3", "3"},

		// round: half away from zero at the given scale.
		{"round down scale 0", "round", 0, "2.4", "2"},
		{"round up scale 0", "round", 0, "2.5", "3"},
		{"round scale 2", "round", 2, "2.945", "2.95"},

		// Zero aggregate stays zero under every mode.
		{"zero floor", "floor", 0, "0", "0"},
		{"zero ceil", "ceil", 0, "0", "0"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			metric := domain.BillableMetric{RoundingMode: tc.mode, RoundingScale: tc.scale}
			got := applyRounding(metric, decimal.RequireFromString(tc.in))
			if !got.Equal(decimal.RequireFromString(tc.want)) {
				t.Errorf("applyRounding(%s, mode=%q scale=%d) = %s, want %s", tc.in, tc.mode, tc.scale, got, tc.want)
			}
		})
	}
}
