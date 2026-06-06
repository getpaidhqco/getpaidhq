package jetstream

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// --- embedded JetStream server (in-process, file storage in a temp dir) ---

func embeddedJS(t *testing.T) jetstream.JetStream {
	t.Helper()
	ns, err := natsserver.NewServer(&natsserver.Options{
		ServerName: "js_test",
		DontListen: true,
		JetStream:  true,
		StoreDir:   t.TempDir(),
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

// --- fake EventStore (only IngestBatch is exercised by the consumer) ---

type fakeStore struct {
	port.EventStore
	mu    sync.Mutex
	got   []domain.MeterEvent
	calls int
	failN int // fail the first N IngestBatch calls (transient error)
}

func (s *fakeStore) IngestBatch(_ context.Context, events []domain.MeterEvent) ([]port.IngestResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.failN > 0 {
		s.failN--
		return nil, errors.New("transient store failure")
	}
	out := make([]port.IngestResult, len(events))
	for i, e := range events {
		s.got = append(s.got, e)
		out[i] = port.IngestResult{Id: e.Id, Status: port.IngestRecorded}
	}
	return out, nil
}

func (s *fakeStore) ids() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.got))
	for i, e := range s.got {
		out[i] = e.Id
	}
	return out
}

func waitFor(t *testing.T, cond func() bool, d time.Duration) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", d)
}

func ev(id, externalId string) domain.MeterEvent {
	return domain.MeterEvent{
		OrgId: "org_1", Id: id, CustomerId: "cus_1", MetricCode: "api_calls",
		ExternalId: externalId, Value: decimal.NewFromInt(1),
		Metadata: map[string]string{"k": "v"}, Timestamp: time.Unix(1700000000, 0).UTC(),
	}
}

// --- tests ---

func TestJetStream_AcceptAndDrain(t *testing.T) {
	js := embeddedJS(t)
	store := &fakeStore{}
	cons, err := NewConsumer(context.Background(), store, js, 10, noopLogger{})
	require.NoError(t, err)
	defer cons.Close()

	ing := NewIngestor(js, noopLogger{})
	res, err := ing.Ingest(context.Background(), ev("e1", "x1"))
	require.NoError(t, err)
	assert.Equal(t, port.IngestAccepted, res.Status, "publish returns accepted")

	waitFor(t, func() bool { return len(store.ids()) == 1 }, 3*time.Second)
	assert.Equal(t, []string{"e1"}, store.ids())
}

func TestJetStream_DedupByExternalId(t *testing.T) {
	js := embeddedJS(t)
	store := &fakeStore{}
	cons, err := NewConsumer(context.Background(), store, js, 10, noopLogger{})
	require.NoError(t, err)
	defer cons.Close()

	ing := NewIngestor(js, noopLogger{})
	// Two events sharing external_id "x1" → JetStream msg-id dedup collapses them.
	_, err = ing.Ingest(context.Background(), ev("e1", "x1"))
	require.NoError(t, err)
	_, err = ing.Ingest(context.Background(), ev("e2", "x1"))
	require.NoError(t, err)

	waitFor(t, func() bool { return len(store.ids()) >= 1 }, 3*time.Second)
	time.Sleep(300 * time.Millisecond) // give a second delivery a chance, if any
	assert.Len(t, store.ids(), 1, "resend with same external_id must not double-write")
}

func TestJetStream_RedeliveryOnTransientError(t *testing.T) {
	js := embeddedJS(t)
	store := &fakeStore{failN: 1} // first batch write fails → nak → redeliver
	cons, err := NewConsumer(context.Background(), store, js, 10, noopLogger{})
	require.NoError(t, err)
	defer cons.Close()

	ing := NewIngestor(js, noopLogger{})
	_, err = ing.Ingest(context.Background(), ev("e1", "x1"))
	require.NoError(t, err)

	waitFor(t, func() bool { return len(store.ids()) == 1 }, 5*time.Second)
	store.mu.Lock()
	calls := store.calls
	store.mu.Unlock()
	assert.GreaterOrEqual(t, calls, 2, "should have retried after the transient failure")
	assert.Equal(t, []string{"e1"}, store.ids(), "exactly one net write after redelivery")
}

// noopLogger satisfies port.Logger.
type noopLogger struct{}

func (noopLogger) Debug(string, ...any)  {}
func (noopLogger) Info(string, ...any)   {}
func (noopLogger) Warn(string, ...any)   {}
func (noopLogger) Error(string, ...any)  {}
func (noopLogger) Fatal(string, ...any)  {}
func (noopLogger) Debugf(string, ...any) {}
func (noopLogger) Infof(string, ...any)  {}
func (noopLogger) Warnf(string, ...any)  {}
func (noopLogger) Errorf(string, ...any) {}
func (noopLogger) Panicf(string, ...any) {}
func (noopLogger) Fatalf(string, ...any) {}
func (noopLogger) Sync() error           { return nil }
