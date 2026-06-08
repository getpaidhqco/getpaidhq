//go:build integration

// Integration tests for the ClickHouse usage-event store. These run against a REAL
// ClickHouse server managed by Testcontainers — the same testable-infra model the
// Postgres integration tests use (a fresh container per package run, never the
// developer's local stack).
//
// Run with:
//
//	go test -tags=integration ./internal/adapter/clickhouse/...
//
// What this pins: the ClickHouse adapter must produce the SAME observable
// aggregation results as the Postgres adapter (both back port.EventStore). The
// assertions deliberately mirror the Postgres event_store_integration_test number
// for number, so a divergence between the two backends fails here. It exercises
// read-time dedup (dedup_key collapses resends), the half-open [from,to) window,
// the customer-OR match (customer_id OR external_customer_id), subscription
// attribution incl. IncludeUnattributed, and the deferred weighted_sum.
package clickhouse

import (
	"context"
	_ "embed"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

//go:embed migrations/0001_meter_events.sql
var meterEventsDDL string

var (
	sharedConn driver.Conn
	sharedOnce sync.Once
	sharedErr  error
)

// testConn starts a ClickHouse container once per package run, applies the
// meter_events DDL, and returns a connection. Subsequent tests reuse it; per-test
// isolation comes from a unique org id, so rows from one test never match another's
// queries even though they share the table.
func testConn(t *testing.T) driver.Conn {
	t.Helper()

	sharedOnce.Do(func() {
		ctx := context.Background()
		req := testcontainers.ContainerRequest{
			Image:        "clickhouse/clickhouse-server:24.3-alpine",
			ExposedPorts: []string{"9000/tcp"},
			Env: map[string]string{
				"CLICKHOUSE_DB":                        "getpaidhq_usage",
				"CLICKHOUSE_USER":                      "default",
				"CLICKHOUSE_PASSWORD":                  "",
				"CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT": "1",
			},
			// ClickHouse logs "Ready for connections" to a file, not stdout, so a log
			// wait never matches here. Wait for the native port to listen, then the
			// ping-retry loop below confirms the server can actually serve queries.
			WaitingFor: wait.ForListeningPort("9000/tcp").WithStartupTimeout(90 * time.Second),
		}
		c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			sharedErr = fmt.Errorf("start clickhouse container: %w", err)
			return
		}
		// Don't tie teardown to t.Cleanup here: the container is shared across the
		// package's tests via sync.Once, so the first test's cleanup would kill it for
		// the rest. The testcontainers Ryuk reaper terminates it when the test process
		// exits (same model as the Postgres setup_test.go container).

		host, err := c.Host(ctx)
		if err != nil {
			sharedErr = fmt.Errorf("container host: %w", err)
			return
		}
		port, err := c.MappedPort(ctx, "9000/tcp")
		if err != nil {
			sharedErr = fmt.Errorf("mapped port: %w", err)
			return
		}

		conn, err := clickhouse.Open(&clickhouse.Options{
			Addr: []string{fmt.Sprintf("%s:%s", host, port.Port())},
			Auth: clickhouse.Auth{Database: "getpaidhq_usage", Username: "default"},
		})
		if err != nil {
			sharedErr = fmt.Errorf("open clickhouse: %w", err)
			return
		}

		// The server may accept TCP a beat before it can serve queries; retry the
		// ping briefly so DDL doesn't race startup.
		pctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
		for {
			if err = conn.Ping(pctx); err == nil {
				break
			}
			select {
			case <-pctx.Done():
				sharedErr = fmt.Errorf("ping clickhouse: %w", err)
				return
			case <-time.After(500 * time.Millisecond):
			}
		}

		if err := conn.Exec(ctx, meterEventsDDL); err != nil {
			sharedErr = fmt.Errorf("apply meter_events DDL: %w", err)
			return
		}
		sharedConn = conn
	})

	if sharedErr != nil {
		t.Fatalf("clickhouse test setup failed: %v", sharedErr)
	}
	return sharedConn
}

// TestClickHouseEventStore_Aggregations mirrors the Postgres adapter's integration
// test (TestEventStore_Aggregations) value-for-value, proving parity against a live
// ClickHouse: write a tricky event set, then assert every aggregation, dedup, the
// window boundary, customer matching and subscription attribution.
func TestClickHouseEventStore_Aggregations(t *testing.T) {
	conn := testConn(t)
	store := NewEventStoreWithConn(conn)
	ctx := context.Background()
	orgId := "org_" + t.Name()

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
		mk("e1", "x1", "sub_1", "eu", 10, from),                    // [from] boundary -> in
		mk("e2", "x1", "sub_1", "eu", 10, from.Add(time.Minute)),   // resend of x1 -> read-time dedup
		mk("e3", "x2", "sub_1", "us", 25, from.Add(2*time.Hour)),   // distinct, attributed
		mk("e4", "x3", "", "eu", 5, from.Add(3*time.Hour)),         // unattributed
		mk("e5", "x4", "sub_1", "ap", 100, from.Add(24*time.Hour)), // out of window
		// Foreign customer must never leak in.
		{OrgId: orgId, Id: "e6", CustomerId: "cus_2", MetricCode: "api_calls", SubscriptionId: "sub_1", ExternalId: "y1", Value: decimal.NewFromInt(999), Metadata: map[string]string{"region": "eu"}, Timestamp: from.Add(time.Hour), CreatedAt: from},
	}
	for _, e := range events {
		_, err := store.Ingest(ctx, e)
		require.NoError(t, err)
	}

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

// TestClickHouseEventStore_ExternalCustomerIdMatch proves the customer-OR match:
// usage recorded against the merchant's own external_customer_id (no internal id)
// is found when the query carries both ids, while a foreign external id is not.
func TestClickHouseEventStore_ExternalCustomerIdMatch(t *testing.T) {
	conn := testConn(t)
	store := NewEventStoreWithConn(conn)
	ctx := context.Background()
	orgId := "org_" + t.Name()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(12 * time.Hour)

	events := []domain.MeterEvent{
		{OrgId: orgId, Id: "ia", CustomerId: "cus_1", MetricCode: "api_calls", ExternalId: "i1", Value: decimal.NewFromInt(3), Timestamp: from.Add(time.Hour), CreatedAt: from},
		{OrgId: orgId, Id: "ob", ExternalCustomerId: "ext_cus_1", MetricCode: "api_calls", ExternalId: "o1", Value: decimal.NewFromInt(4), Timestamp: from.Add(2 * time.Hour), CreatedAt: from},
		{OrgId: orgId, Id: "fc", ExternalCustomerId: "someone_else", MetricCode: "api_calls", ExternalId: "f1", Value: decimal.NewFromInt(99), Timestamp: from.Add(3 * time.Hour), CreatedAt: from},
	}
	for _, e := range events {
		_, err := store.Ingest(ctx, e)
		require.NoError(t, err)
	}

	q := port.UsageQuery{OrgId: orgId, MetricCode: "api_calls", From: from, To: to, CustomerId: "cus_1", ExternalCustomerId: "ext_cus_1"}
	n, err := store.Count(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, int64(2), n, "internal + external-id match; foreign external id excluded")

	sum, err := store.Sum(ctx, q)
	require.NoError(t, err)
	assert.True(t, sum.Equal(decimal.NewFromInt(7)), "3 + 4, got %s", sum)
}

// TestClickHouseEventStore_WeightedSum_NotImplemented pins the deferred weighted_sum
// aggregation on the ClickHouse adapter, matching the Postgres adapter's behaviour.
func TestClickHouseEventStore_WeightedSum_NotImplemented(t *testing.T) {
	conn := testConn(t)
	store := NewEventStoreWithConn(conn)
	ctx := context.Background()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	q := port.UsageQuery{OrgId: "org_x", MetricCode: "api_calls", From: from, To: from.Add(time.Hour), CustomerId: "cus_1"}
	_, err := store.WeightedSum(ctx, q, decimal.Zero)
	require.Error(t, err, "weighted_sum is deferred on ClickHouse too")
}
