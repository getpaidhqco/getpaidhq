//go:build integration

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

// TestEventStore_FilterAndGroup exercises the REAL Postgres metadata filter (WHERE
// metadata->>field …, incl. the default charge's NOT IN + absent-field branch) and the
// grouped aggregation (GROUP BY metadata->>key) together. Mirrors the messaging-API
// worked example in docs/internal/usage-filters-and-groups.md.
func TestEventStore_FilterAndGroup(t *testing.T) {
	pool := poolForTest(t)
	orgId := uniqueOrg(t, pool)
	defer cleanupOrg(t, pool, orgId)

	store := NewEventStore(pool)
	ctx := context.Background()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(12 * time.Hour)

	i := 0
	mk := func(meta map[string]string) domain.MeterEvent {
		i++
		return domain.MeterEvent{
			OrgId: orgId, Id: "ev_" + string(rune('a'+i)), CustomerId: "cus_1",
			MetricCode: "messages", ExternalId: "x_" + string(rune('a'+i)),
			Metadata: meta, Timestamp: from.Add(time.Duration(i) * time.Minute), CreatedAt: from,
		}
	}
	events := []domain.MeterEvent{
		mk(map[string]string{"type": "SMS", "project": "acme"}),
		mk(map[string]string{"type": "SMS", "project": "acme"}),
		mk(map[string]string{"type": "SMS", "project": "acme"}),
		mk(map[string]string{"type": "SMS", "project": "globex"}),
		mk(map[string]string{"type": "SMS", "project": "globex"}),
		mk(map[string]string{"type": "MMS", "project": "acme"}),
		mk(map[string]string{"type": "MMS", "project": "initech"}),
		mk(map[string]string{"type": "PUSH", "project": "acme"}), // not SMS/MMS → default charge
		mk(map[string]string{"project": "acme"}),                 // type absent → default charge
	}
	for _, e := range events {
		_, err := store.Ingest(ctx, e)
		require.NoError(t, err)
	}

	base := port.UsageQuery{OrgId: orgId, MetricCode: "messages", From: from, To: to, CustomerId: "cus_1"}

	t.Run("filter to a specific value", func(t *testing.T) {
		q := base
		q.FilterField, q.FilterValue = "type", "SMS"
		n, err := store.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(5), n, "3 acme + 2 globex SMS")
	})

	t.Run("default charge excludes priced values and folds in absent", func(t *testing.T) {
		q := base
		q.FilterField, q.FilterExclude = "type", []string{"SMS", "MMS"}
		n, err := store.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(2), n, "PUSH + the type-absent event")
	})

	t.Run("group a filtered slice by project", func(t *testing.T) {
		q := base
		q.FilterField, q.FilterValue = "type", "SMS"
		groups, err := store.AggregateGrouped(ctx, q, domain.AggregationCount, "project")
		require.NoError(t, err)

		got := map[string]int64{}
		for _, g := range groups {
			assert.Equal(t, "project", g.Key)
			got[g.Value] = g.Quantity.IntPart()
		}
		assert.Equal(t, map[string]int64{"acme": 3, "globex": 2}, got, "SMS split per project at one rate")
	})

	t.Run("grouped sum honours the filter", func(t *testing.T) {
		// All events have value 0 (count meter), so sum is 0 per group — assert the
		// segment set, proving GROUP BY + filter compose for a numeric aggregation too.
		q := base
		q.FilterField, q.FilterValue = "type", "MMS"
		groups, err := store.AggregateGrouped(ctx, q, domain.AggregationSum, "project")
		require.NoError(t, err)
		got := map[string]bool{}
		for _, g := range groups {
			got[g.Value] = true
			assert.True(t, g.Quantity.Equal(decimal.Zero))
		}
		assert.Equal(t, map[string]bool{"acme": true, "initech": true}, got)
	})
}
