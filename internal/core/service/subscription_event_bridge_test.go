package service

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// fakeEngine records UpdateSubscriptionWorkflow calls and short-circuits the
// rest of the port.Engine surface.
type fakeEngine struct {
	mu      sync.Mutex
	updates []updateCall
}

type updateCall struct {
	name string
	sub  domain.Subscription
}

func (f *fakeEngine) StartWorkflow(ctx context.Context, id port.WorkflowType, payload any) (port.WorkflowResult, error) {
	return port.WorkflowResult{}, nil
}
func (f *fakeEngine) StartSubscriptionWorkflow(ctx context.Context, sub domain.Subscription) error {
	return nil
}
func (f *fakeEngine) UpdateSubscriptionWorkflow(ctx context.Context, name string, sub domain.Subscription) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updates = append(f.updates, updateCall{name: name, sub: sub})
	return nil
}
func (f *fakeEngine) CancelSubscriptionWorkflow(ctx context.Context, sub domain.Subscription) error {
	return nil
}
func (f *fakeEngine) SignalSubscriptionWorkflow(ctx context.Context, signal string, sub domain.Subscription, payload any) error {
	return nil
}

// silentLogger drops everything — Logger isn't exported as an interface here,
// so reuse port.Logger directly via a no-op embed.
type silentLogger struct{}

func (silentLogger) Debug(string, ...any)              {}
func (silentLogger) Info(string, ...any)               {}
func (silentLogger) Warn(string, ...any)               {}
func (silentLogger) Error(string, ...any)              {}
func (silentLogger) Sync() error                       { return nil }
func (silentLogger) Fatalf(string, ...any)             {}
func (silentLogger) Fatal(string, ...any)              {}
func (silentLogger) Infof(string, ...any)              {}
func (silentLogger) Debugf(string, ...any)             {}
func (silentLogger) Errorf(string, ...any)             {}
func (silentLogger) Panicf(string, ...any)             {}
func (silentLogger) Warnf(string, ...any)              {}

// fakePubSub captures the topic->handler registration so the test can inject
// raw bytes directly.
type fakePubSub struct {
	handler func(topic string, data []byte)
}

func (f *fakePubSub) Publish(orgId, topic string, message any) error { return nil }
func (f *fakePubSub) Subscribe(topic string, handler func(string, []byte)) (port.PubSubSubscription, error) {
	f.handler = handler
	return fakeSub{}, nil
}

type fakeSub struct{}

func (fakeSub) Unsubscribe() error { return nil }

// envelope mimics the layout SubscriptionService publishes.
func envelope(t *testing.T, topic string, sub domain.Subscription) []byte {
	t.Helper()
	b, err := json.Marshal(port.PubSubPayload{
		Id:    "evt_1",
		OrgId: sub.OrgId,
		Topic: topic,
		Data:  sub,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func TestSubscriptionEventBridge_ForwardsPaused(t *testing.T) {
	engine := &fakeEngine{}
	ps := &fakePubSub{}
	_ = NewSubscriptionEventBridge(engine, ps, silentLogger{})

	if ps.handler == nil {
		t.Fatal("bridge did not register a pubsub handler")
	}

	sub := domain.Subscription{OrgId: "org_1", Id: "sub_2"}
	ps.handler(port.TopicSubscriptionPaused, envelope(t, port.TopicSubscriptionPaused, sub))

	if len(engine.updates) != 1 {
		t.Fatalf("expected 1 forwarded update, got %d", len(engine.updates))
	}
	if engine.updates[0].name != port.TopicSubscriptionPaused {
		t.Errorf("update name: got %q", engine.updates[0].name)
	}
	if engine.updates[0].sub.Id != "sub_2" {
		t.Errorf("subscription id lost in marshal round-trip: got %q", engine.updates[0].sub.Id)
	}
}

func TestSubscriptionEventBridge_IgnoresUnknownTopic(t *testing.T) {
	engine := &fakeEngine{}
	ps := &fakePubSub{}
	_ = NewSubscriptionEventBridge(engine, ps, silentLogger{})

	sub := domain.Subscription{OrgId: "org_1", Id: "sub_2"}
	ps.handler(port.TopicSubscriptionResumed, envelope(t, port.TopicSubscriptionResumed, sub))

	if len(engine.updates) != 0 {
		t.Errorf("expected no forwarded update for unhandled topic, got %d", len(engine.updates))
	}
}

func TestSubscriptionEventBridge_DroppedOnBadEnvelope(t *testing.T) {
	engine := &fakeEngine{}
	ps := &fakePubSub{}
	_ = NewSubscriptionEventBridge(engine, ps, silentLogger{})

	ps.handler(port.TopicSubscriptionPaused, []byte("not json"))

	if len(engine.updates) != 0 {
		t.Errorf("expected no forwarded update on bad envelope, got %d", len(engine.updates))
	}
}
