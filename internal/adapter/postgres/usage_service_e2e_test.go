//go:build integration

// Aggregation-level coverage that needs the REAL Postgres SQL but not the full
// charge path: rounding of aggregated quantities (UsageService.AggregateForPeriod
// → applyRounding over a real Sum) and the deferred weighted_sum aggregation.
package postgres

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
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	defer cleanupOrg(t, db, orgId)

	store := NewEventStore(db)
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

	usage := buildUsageService(t, db)
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

// TestEventStore_WeightedSum_NotImplemented pins the current deferred state of the
// weighted_sum aggregation at both the store and the service layer. When weighted_sum
// is implemented (spec phase 5), this test must be updated to assert the computed
// value instead — it is the canary that the behaviour changed.
func TestEventStore_WeightedSum_NotImplemented(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	defer cleanupOrg(t, db, orgId)

	store := NewEventStore(db)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	q := port.UsageQuery{OrgId: orgId, MetricCode: "tokens", CustomerId: "cus_ws", From: from, To: from.Add(time.Hour)}

	_, err := store.WeightedSum(ctx, q, decimal.Zero)
	require.Error(t, err, "weighted_sum is deferred — the store must report it unimplemented")

	// And it surfaces through AggregateForPeriod for a weighted_sum meter.
	usage := buildUsageService(t, db)
	metric := domain.BillableMetric{OrgId: orgId, Code: "tokens", Aggregation: domain.AggregationWeightedSum, FieldName: "amount"}
	_, err = usage.AggregateForPeriod(ctx, metric, q)
	require.Error(t, err, "AggregateForPeriod must propagate the unimplemented weighted_sum error")
}
