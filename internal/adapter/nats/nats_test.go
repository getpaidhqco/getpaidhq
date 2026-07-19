package nats

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// runEmbeddedNats starts an in-process NATS server on a random free port
// (Port: -1) so tests are hermetic and never collide on 4222. The server is
// shut down via t.Cleanup, which combined with NatsPubSub.Close keeps the
// package goroutine-clean for goleak.
func runEmbeddedNats(t *testing.T) string {
	t.Helper()
	ns, err := natsserver.NewServer(&natsserver.Options{Host: "127.0.0.1", Port: -1})
	require.NoError(t, err)
	go ns.Start()
	if !ns.ReadyForConnections(10 * time.Second) {
		t.Fatal("embedded nats server not ready")
	}
	t.Cleanup(func() {
		ns.Shutdown()
		ns.WaitForShutdown()
	})
	return ns.ClientURL()
}

func TestNewNatsPubSub_ConnectError(t *testing.T) {
	// A refused connection is retried in the background (RetryOnFailedConnect),
	// so the error path is exercised with a malformed URL that fails to parse
	// synchronously regardless of the reconnect options.
	_, err := NewNatsPubSub("nats://%zz", lib.GetLogger())
	require.Error(t, err)
}

func TestNatsPubSub_PublishSubscribeRoundTrip(t *testing.T) {
	url := runEmbeddedNats(t)

	ps, err := NewNatsPubSub(url, lib.GetLogger())
	require.NoError(t, err)
	t.Cleanup(func() { _ = ps.Close() })

	received := make(chan []byte, 1)
	sub, err := ps.Subscribe("subscription.paused", func(_ string, data []byte) {
		received <- data
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = sub.Unsubscribe() })

	want := domain.Subscription{OrgId: "mollie", Id: "sub_2saZn2yvjfnzJ6Io2yfgEsCwtmg", Status: "paused"}
	require.NoError(t, ps.Publish(context.Background(), "mollie", "subscription.paused", want))

	select {
	case data := <-received:
		var env port.PubSubPayload
		require.NoError(t, json.Unmarshal(data, &env))
		assert.Equal(t, "mollie", env.OrgId)
		assert.Equal(t, "subscription.paused", env.Topic)

		// Data round-trips through the envelope as JSON.
		var got domain.Subscription
		b, err := json.Marshal(env.Data)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(b, &got))
		assert.Equal(t, want.Id, got.Id)
		assert.Equal(t, want.OrgId, got.OrgId)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for published message")
	}
}
