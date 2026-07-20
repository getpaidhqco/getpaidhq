//go:build integration

// Aggregation-level coverage that needs the REAL Postgres SQL but not the full
// charge path: rounding of aggregated quantities (UsageService.AggregateForPeriod
// → applyRounding over a real Sum) and the deferred weighted_sum aggregation.
package postgrespgx

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// TestUsageService_AggregateForPeriod_Rounding proves the meter's RoundingMode /
// RoundingScale is applied to the aggregated quantity. Two fractional events
// (1.5 + 1.4 = 2.9 raw) are summed, then rounded per the meter config. The raw sum
// comes from real SQL; the rounding from applyRounding.
func TestUsageService_AggregateForPeriod_Rounding(t *testing.T) {
	pool := poolForTest(t)
	ctx := context.Background()
	orgId := uniqueOrg(t, pool)
	defer cleanupOrg(t, pool, orgId)

	store := NewEventStore(pool)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(12 * time.Hour)

	// Two in-window events summing to 2.9.
	for i, v := range []string{"1.5", "1.4"} {
		_, err := store.Ingest(ctx, domain.MeterEvent{
			OrgId: orgId, Id: "r" + v, CustomerId: "cus_round", MetricCode: "tokens",
			ExternalId: "ext" + v,
			Value:      decimal.RequireFromString(v),
			Timestamp:  from.Add(time.Duration(i+1) * time.Hour),
			CreatedAt:  from,
		})
		require.NoError(t, err)
	}

	usage := buildUsageService(t, pool)
	q := port.UsageQuery{OrgId: orgId, MetricCode: "tokens", CustomerId: "cus_round", From: from, To: to}

	cases := []struct {
		name  string
		mode  string
		scale int
		want  string
	}{
		{"none", "", 0, "2.9"},
		{"floor scale 0", "floor", 0, "2"},
		{"ceil scale 0", "ceil", 0, "3"},
		{"round scale 0", "round", 0, "3"},
		{"round scale 1", "round", 1, "2.9"},
		{"floor scale 1", "floor", 1, "2.9"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			metric := domain.BillableMetric{
				OrgId: orgId, Code: "tokens", Aggregation: domain.AggregationSum,
				FieldName: "amount", RoundingMode: tc.mode, RoundingScale: tc.scale,
			}
			got, err := usage.AggregateForPeriod(ctx, metric, q)
			require.NoError(t, err)
			assert.True(t, got.Equal(decimal.RequireFromString(tc.want)),
				"%s: got %s, want %s", tc.name, got, tc.want)
		})
	}
}

// TestEventStore_WeightedSum_FlowForbidden pins that weighted_sum only exists on
// carry-over meters: a flow meter resets each period, so a time-averaged level
// would underbill every quiet period. Forbidden at meter creation; the read path
// guards defensively.
func TestEventStore_WeightedSum_FlowForbidden(t *testing.T) {
	pool := poolForTest(t)
	ctx := context.Background()
	orgId := uniqueOrg(t, pool)
	defer cleanupOrg(t, pool, orgId)

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	q := port.UsageQuery{OrgId: orgId, MetricCode: "tokens", CustomerId: "cus_ws", From: from, To: from.Add(time.Hour)}

	usage := buildUsageService(t, pool)
	metric := domain.BillableMetric{OrgId: orgId, Code: "tokens", Aggregation: domain.AggregationWeightedSum, FieldName: "amount"}
	_, err := usage.AggregateForPeriod(ctx, metric, q)
	require.Error(t, err, "weighted_sum without carry_over must be rejected")
}
