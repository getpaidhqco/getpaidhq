//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gphqjetstream "getpaidhq/internal/adapter/jetstream"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// embeddedJS starts an in-process JetStream server for the test.
func embeddedJS(t *testing.T) jetstream.JetStream {
	t.Helper()
	ns, err := natsserver.NewServer(&natsserver.Options{
		ServerName: "e2e_js", DontListen: true, JetStream: true, StoreDir: t.TempDir(),
	})
	require.NoError(t, err)
	go ns.Start()
	require.True(t, ns.ReadyForConnections(5*time.Second), "embedded nats not ready")
	nc, err := nats.Connect("", nats.InProcessServer(ns))
	require.NoError(t, err)
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	t.Cleanup(func() { nc.Close(); ns.Shutdown() })
	return js
}

// TestUsageIngest_JetStreamToPostgres_E2E exercises the full durable async path:
// Ingestor publishes to JetStream → background Consumer drains → real Postgres
// EventStore persists → aggregation reads it back. Proves the accepted event
// actually lands and is billable (the fake-store unit test can't show this).
func TestUsageIngest_JetStreamToPostgres_E2E(t *testing.T) {
	db := testDB(t)
	require.NoError(t, EnsureUsageSchema(db), "dedup index")
	orgId := uniqueOrg(t)
	defer cleanupOrg(t, db, orgId)

	store := NewEventStore(db)
	js := embeddedJS(t)

	consumer, err := gphqjetstream.NewConsumer(context.Background(), store, js, 50, noopLogger{})
	require.NoError(t, err)
	defer consumer.Close()
	ing := gphqjetstream.NewIngestor(js, noopLogger{})

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(time.Hour)
	mk := func(id, ext string, val int64) domain.MeterEvent {
		return domain.MeterEvent{
			OrgId: orgId, Id: id, CustomerId: "cus_1", MetricCode: "api_calls",
			SubscriptionId: "sub_1", ExternalId: ext, Value: decimal.NewFromInt(val),
			Metadata: map[string]string{"region": "eu"}, Timestamp: from, CreatedAt: from,
		}
	}

	ctx := context.Background()
	for _, e := range []domain.MeterEvent{mk("e1", "x1", 10), mk("e2", "x2", 25), mk("e3", "x1", 10)} {
		res, err := ing.Ingest(ctx, e)
		require.NoError(t, err)
		require.Equal(t, port.IngestAccepted, res.Status)
	}

	q := port.UsageQuery{
		OrgId: orgId, MetricCode: "api_calls", FieldName: "region",
		From: from, To: to, CustomerId: "cus_1", SubscriptionId: "sub_1",
	}

	// Wait for the consumer to drain into Postgres, then assert it's queryable.
	// x1 is sent twice (same external_id) → deduped to one row by the partial unique index.
	require.Eventually(t, func() bool {
		n, err := store.Count(ctx, q)
		return err == nil && n == 2
	}, 8*time.Second, 50*time.Millisecond, "events should persist and dedup to 2")

	sum, err := store.Sum(ctx, q)
	require.NoError(t, err)
	assert.True(t, sum.Equal(decimal.NewFromInt(35)), "sum = %s, want 35 (10 + 25, x1 deduped)", sum)
}
