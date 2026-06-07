//go:build integration

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

// TestEventStore_Aggregations exercises the REAL Postgres SQL (the unit-level parity
// harness only compares in-memory references). It pins write-time dedup (the unique
// index AutoMigrate creates from the row's gorm tag), the half-open [from,to) window,
// the customer match, and subscription attribution incl. IncludeUnattributed.
func TestEventStore_Aggregations(t *testing.T) {
	db := testDB(t)
	orgId := uniqueOrg(t)
	defer cleanupOrg(t, db, orgId)

	store := NewEventStore(db)
	ctx := context.Background()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(12 * time.Hour)

	mk := func(id, ext, sub, region string, val int64, ts time.Time) domain.MeterEvent {
		return domain.MeterEvent{
			OrgId: orgId, Id: id, CustomerId: "cus_1", MetricCode: "api_calls",
			SubscriptionId: sub, ExternalId: ext, Value: decimal.NewFromInt(val),
			Metadata: map[string]string{"region": region}, Timestamp: ts, CreatedAt: ts,
		}
	}
	events := []domain.MeterEvent{
		mk("e1", "x1", "sub_1", "eu", 10, from),                    // on [from] boundary -> in
		mk("e3", "x2", "sub_1", "us", 25, from.Add(2*time.Hour)),   // distinct, attributed
		mk("e4", "x3", "", "eu", 5, from.Add(3*time.Hour)),         // unattributed
		mk("e5", "x4", "sub_1", "ap", 100, from.Add(24*time.Hour)), // out of window
	}
	for _, e := range events {
		_, err := store.Ingest(ctx, e)
		require.NoError(t, err)
	}
	// Resend of x1 (same external_id) must be deduped by the partial unique index.
	res, err := store.Ingest(ctx, mk("e2", "x1", "sub_1", "eu", 10, from.Add(time.Minute)))
	require.NoError(t, err)
	assert.Equal(t, port.IngestDuplicate, res.Status, "resend with seen external_id must be reported as duplicate")
	// Foreign customer must never leak in.
	_, err = store.Ingest(ctx, domain.MeterEvent{OrgId: orgId, Id: "e6", CustomerId: "cus_2", MetricCode: "api_calls", ExternalId: "y1", Value: decimal.NewFromInt(999), Metadata: map[string]string{"region": "eu"}, Timestamp: from.Add(time.Hour)})
	require.NoError(t, err)

	attributed := port.UsageQuery{OrgId: orgId, MetricCode: "api_calls", FieldName: "region", From: from, To: to, CustomerId: "cus_1", SubscriptionId: "sub_1"}
	withUnattr := attributed
	withUnattr.IncludeUnattributed = true

	t.Run("attributed only", func(t *testing.T) {
		n, err := store.Count(ctx, attributed)
		require.NoError(t, err)
		assert.Equal(t, int64(2), n, "x1 (deduped) + x2; not unattributed/out-of-window/foreign")

		sum, err := store.Sum(ctx, attributed)
		require.NoError(t, err)
		assert.True(t, sum.Equal(decimal.NewFromInt(35)), "10+25, got %s", sum)

		max, err := store.Max(ctx, attributed)
		require.NoError(t, err)
		assert.True(t, max.Equal(decimal.NewFromInt(25)), "got %s", max)

		uc, err := store.UniqueCount(ctx, attributed)
		require.NoError(t, err)
		assert.Equal(t, int64(2), uc, "regions eu, us")

		latest, err := store.Latest(ctx, attributed)
		require.NoError(t, err)
		assert.True(t, latest.Equal(decimal.NewFromInt(25)), "latest in-window attributed is x2=25, got %s", latest)
	})

	t.Run("with unattributed", func(t *testing.T) {
		n, err := store.Count(ctx, withUnattr)
		require.NoError(t, err)
		assert.Equal(t, int64(3), n, "+ x3 unattributed")

		sum, err := store.Sum(ctx, withUnattr)
		require.NoError(t, err)
		assert.True(t, sum.Equal(decimal.NewFromInt(40)), "10+25+5, got %s", sum)

		latest, err := store.Latest(ctx, withUnattr)
		require.NoError(t, err)
		assert.True(t, latest.Equal(decimal.NewFromInt(5)), "latest now x3=5 at +3h, got %s", latest)
	})
}
