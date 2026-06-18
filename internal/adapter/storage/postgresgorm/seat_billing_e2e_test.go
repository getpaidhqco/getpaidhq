//go:build integration

// End-to-end seat (stock) billing: the June timeline from
// docs/internal/billing-model/seat-billing/README.md ingested through the real
// RecordEvent validation, aggregated through UsageService.AggregateForPeriod for
// June AND July. July is the carry-over proof: it has no events at all, yet
// bills the standing seats. Both event styles are covered: add/remove events
// and level reports.
package postgresgorm

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

var (
	seatJun1 = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	seatJul1 = time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	seatAug1 = time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
)

func seatMetric(orgId string, agg domain.AggregationType, fieldName string) domain.BillableMetric {
	return domain.BillableMetric{
		OrgId: orgId, Code: "seats", Aggregation: agg, FieldName: fieldName,
		CarryOver: true, RoundingMode: "round", RoundingScale: 2,
	}
}

func TestSeatBilling_AddRemoveEvents(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	defer cleanupOrg(t, db, orgId)

	meters := NewMeterRepo(db)
	_, err := meters.Create(ctx, domain.BillableMetric{
		OrgId: orgId, Id: "met_seats", Code: "seats", Name: "Seats",
		Aggregation: domain.AggregationWeightedSum, FieldName: "seat_id", CarryOver: true,
	})
	require.NoError(t, err)

	usage := buildUsageService(t, db)
	record := func(extId, seat, op string, ts time.Time) {
		res := recordUsage(t, usage, port.RecordEventInput{
			OrgId: orgId, MetricCode: "seats", ExternalCustomerId: "ext_seat_e2e",
			ExternalId: extId, Timestamp: ts,
			Metadata: map[string]string{domain.UsageOperationKey: op, "seat_id": seat},
		})
		require.Equal(t, port.IngestRecorded, res.Status)
	}
	may20 := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)
	record("se1", "alice", domain.UsageOperationAdd, may20)
	record("se2", "bob", domain.UsageOperationAdd, may20)
	record("se3", "carol", domain.UsageOperationAdd, may20)
	record("se4", "dave", domain.UsageOperationAdd, time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC))
	record("se5", "bob", domain.UsageOperationRemove, time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC))

	aggregate := func(agg domain.AggregationType, from, to time.Time, prorate, credit bool) decimal.Decimal {
		got, err := usage.AggregateForPeriod(ctx, seatMetric(orgId, agg, "seat_id"), port.UsageQuery{
			OrgId: orgId, ExternalCustomerId: "ext_seat_e2e",
			From: from, To: to, ProrateOnIncrease: prorate, CreditOnDecrease: credit,
		})
		require.NoError(t, err)
		return got
	}
	eq := func(t *testing.T, got decimal.Decimal, want string) {
		t.Helper()
		assert.True(t, got.Equal(decimal.RequireFromString(want)), "got %s want %s", got, want)
	}

	t.Run("June", func(t *testing.T) {
		eq(t, aggregate(domain.AggregationLatest, seatJun1, seatJul1, false, false), "3")      // standing at end
		eq(t, aggregate(domain.AggregationMax, seatJun1, seatJul1, false, false), "4")         // peak concurrent
		eq(t, aggregate(domain.AggregationUniqueCount, seatJun1, seatJul1, false, false), "4") // distinct active
		eq(t, aggregate(domain.AggregationWeightedSum, seatJun1, seatJul1, true, true), "3.17") // B
		eq(t, aggregate(domain.AggregationWeightedSum, seatJun1, seatJul1, true, false), "3.5") // C
	})

	// July has zero events — pure carry-over: alice, carol, dave stand all month.
	t.Run("July carries the standing seats", func(t *testing.T) {
		eq(t, aggregate(domain.AggregationLatest, seatJul1, seatAug1, false, false), "3")
		eq(t, aggregate(domain.AggregationMax, seatJul1, seatAug1, false, false), "3")
		eq(t, aggregate(domain.AggregationUniqueCount, seatJul1, seatAug1, false, false), "3")
		eq(t, aggregate(domain.AggregationWeightedSum, seatJul1, seatAug1, true, true), "3")
		eq(t, aggregate(domain.AggregationWeightedSum, seatJul1, seatAug1, true, false), "3")
	})
}

func TestSeatBilling_LevelReports(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	defer cleanupOrg(t, db, orgId)

	meters := NewMeterRepo(db)
	_, err := meters.Create(ctx, domain.BillableMetric{
		OrgId: orgId, Id: "met_seat_levels", Code: "seats", Name: "Seats",
		Aggregation: domain.AggregationLatest, FieldName: "count", CarryOver: true,
	})
	require.NoError(t, err)

	usage := buildUsageService(t, db)
	report := func(extId, v string, ts time.Time) {
		res := recordUsage(t, usage, port.RecordEventInput{
			OrgId: orgId, MetricCode: "seats", ExternalCustomerId: "ext_seat_lvl",
			ExternalId: extId, Timestamp: ts,
			Metadata: map[string]string{"count": v},
		})
		require.Equal(t, port.IngestRecorded, res.Status)
	}
	// 3 seats since May 20, 4 from Jun 16, 3 from Jun 21 — the report shape of
	// the same June timeline.
	report("lv1", "3", time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC))
	report("lv2", "4", time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC))
	report("lv3", "3", time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC))

	aggregate := func(agg domain.AggregationType, from, to time.Time) decimal.Decimal {
		got, err := usage.AggregateForPeriod(ctx, seatMetric(orgId, agg, "count"), port.UsageQuery{
			OrgId: orgId, ExternalCustomerId: "ext_seat_lvl", From: from, To: to,
		})
		require.NoError(t, err)
		return got
	}
	eq := func(t *testing.T, got decimal.Decimal, want string) {
		t.Helper()
		assert.True(t, got.Equal(decimal.RequireFromString(want)), "got %s want %s", got, want)
	}

	t.Run("June", func(t *testing.T) {
		eq(t, aggregate(domain.AggregationLatest, seatJun1, seatJul1), "3")
		eq(t, aggregate(domain.AggregationMax, seatJun1, seatJul1), "4")
		// Average level: 3×15d + 4×5d + 3×10d over 30d = 3.1667 → rounded 3.17.
		eq(t, aggregate(domain.AggregationWeightedSum, seatJun1, seatJul1), "3.17")
	})

	// July: no reports — the Jun 21 value of 3 stands.
	t.Run("July carries the last reported level", func(t *testing.T) {
		eq(t, aggregate(domain.AggregationLatest, seatJul1, seatAug1), "3")
		eq(t, aggregate(domain.AggregationMax, seatJul1, seatAug1), "3")
		eq(t, aggregate(domain.AggregationWeightedSum, seatJul1, seatAug1), "3")
	})
}

// Ingest validation for carry-over meters: an event is either an add/remove
// (operation + identity) or a level report (numeric value under FieldName).
func TestSeatBilling_IngestValidation(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	defer cleanupOrg(t, db, orgId)

	meters := NewMeterRepo(db)
	_, err := meters.Create(ctx, domain.BillableMetric{
		OrgId: orgId, Id: "met_seat_val", Code: "seats", Name: "Seats",
		Aggregation: domain.AggregationWeightedSum, FieldName: "seat_id", CarryOver: true,
	})
	require.NoError(t, err)

	usage := buildUsageService(t, db)
	base := port.RecordEventInput{OrgId: orgId, MetricCode: "seats", ExternalCustomerId: "ext_seat_val"}

	t.Run("add with identity is recorded", func(t *testing.T) {
		in := base
		in.ExternalId = "iv1"
		in.Metadata = map[string]string{domain.UsageOperationKey: domain.UsageOperationAdd, "seat_id": "user_123"}
		res, err := usage.RecordEvent(ctx, in)
		require.NoError(t, err)
		assert.Equal(t, port.IngestRecorded, res.Status)
	})
	t.Run("operation without identity is rejected", func(t *testing.T) {
		in := base
		in.ExternalId = "iv2"
		in.Metadata = map[string]string{domain.UsageOperationKey: domain.UsageOperationAdd}
		_, err := usage.RecordEvent(ctx, in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "seat_id")
	})
	t.Run("unknown operation is rejected", func(t *testing.T) {
		in := base
		in.ExternalId = "iv3"
		in.Metadata = map[string]string{domain.UsageOperationKey: "assigned", "seat_id": "user_123"}
		_, err := usage.RecordEvent(ctx, in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "operation")
	})
	t.Run("level report without operation is recorded", func(t *testing.T) {
		in := base
		in.ExternalId = "iv4"
		in.Metadata = map[string]string{"seat_id": "5"} // numeric value under FieldName
		res, err := usage.RecordEvent(ctx, in)
		require.NoError(t, err)
		assert.Equal(t, port.IngestRecorded, res.Status)
	})
	t.Run("no operation and no numeric value is rejected", func(t *testing.T) {
		in := base
		in.ExternalId = "iv5"
		in.Metadata = map[string]string{"seat_id": "user_123"} // not numeric, no operation
		_, err := usage.RecordEvent(ctx, in)
		require.Error(t, err)
	})
}

// ListHistory returns the events matching the query ordered by timestamp; the
// carry-over read path calls it with a zero From to reach pre-period history.
func TestEventStore_ListHistory(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	defer cleanupOrg(t, db, orgId)

	store := NewEventStore(db)
	ingest := func(id string, ts time.Time) {
		_, err := store.Ingest(ctx, domain.MeterEvent{
			OrgId: orgId, Id: id, CustomerId: "cus_hist", MetricCode: "seats",
			Metadata:  map[string]string{domain.UsageOperationKey: domain.UsageOperationAdd, "seat_id": id},
			Timestamp: ts, CreatedAt: ts,
		})
		require.NoError(t, err)
	}
	ingest("h2", time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC))
	ingest("h1", time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)) // ingested out of order
	ingest("h3", time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC))  // after the period

	events, err := store.ListHistory(ctx, port.UsageQuery{
		OrgId: orgId, MetricCode: "seats", CustomerId: "cus_hist",
		To: seatJul1, // zero From: full history up to the period end
	})
	require.NoError(t, err)
	require.Len(t, events, 2, "pre-period event included, post-period excluded")
	assert.Equal(t, "h1", events[0].Id, "ordered by timestamp")
	assert.Equal(t, "h2", events[1].Id)
	assert.Equal(t, "h1", events[0].Metadata["seat_id"], "metadata round-trips")
}
