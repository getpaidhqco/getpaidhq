package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/lib"
)

// passthroughTx runs the callback without a real transaction.
type passthroughTx struct{}

func (passthroughTx) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// fakeOutboxRepo is an in-memory port.OutboxRepository sufficient for relay
// tests: ClaimPending applies the same pending predicate as the real adapters.
type fakeOutboxRepo struct {
	events []domain.OutboxEvent
}

func (f *fakeOutboxRepo) Create(_ context.Context, ev domain.OutboxEvent) error {
	ev.Id = int64(len(f.events) + 1)
	f.events = append(f.events, ev)
	return nil
}

func (f *fakeOutboxRepo) ClaimPending(_ context.Context, limit, maxAttempts int, now time.Time) ([]domain.OutboxEvent, error) {
	var out []domain.OutboxEvent
	for _, ev := range f.events {
		if len(out) == limit {
			break
		}
		due := ev.NextAttemptAt == nil || !ev.NextAttemptAt.After(now)
		if ev.PublishedAt == nil && ev.Attempts < maxAttempts && due {
			out = append(out, ev)
		}
	}
	return out, nil
}

func (f *fakeOutboxRepo) MarkPublished(_ context.Context, id int64, at time.Time) error {
	ev := f.byId(id)
	ev.PublishedAt = &at
	return nil
}

func (f *fakeOutboxRepo) RecordFailure(_ context.Context, id int64, lastError string, nextAttemptAt time.Time) error {
	ev := f.byId(id)
	ev.Attempts++
	ev.LastError = lastError
	ev.NextAttemptAt = &nextAttemptAt
	return nil
}

func (f *fakeOutboxRepo) PurgePublished(_ context.Context, olderThan time.Time) (int64, error) {
	var kept []domain.OutboxEvent
	var purged int64
	for _, ev := range f.events {
		if ev.PublishedAt != nil && ev.PublishedAt.Before(olderThan) {
			purged++
			continue
		}
		kept = append(kept, ev)
	}
	f.events = kept
	return purged, nil
}

func (f *fakeOutboxRepo) byId(id int64) *domain.OutboxEvent {
	for i := range f.events {
		if f.events[i].Id == id {
			return &f.events[i]
		}
	}
	panic("unknown outbox id")
}

// fakeRawPublisher fails topics listed in failTopics and records the rest.
type fakeRawPublisher struct {
	failTopics map[string]bool
	published  []string // topics in publish order
}

func (f *fakeRawPublisher) PublishPayload(topic string, _ []byte) error {
	if f.failTopics[topic] {
		return errors.New("broker down")
	}
	f.published = append(f.published, topic)
	return nil
}

func newTestRelay(repo *fakeOutboxRepo, pub *fakeRawPublisher) *OutboxRelay {
	return NewOutboxRelay(passthroughTx{}, repo, pub, silentLogger{}, 0, 0)
}

func pendingEvent(id int64, topic string) domain.OutboxEvent {
	return domain.OutboxEvent{
		Id: id, EventId: lib.GenerateId("evt"), OrgId: "org_1", Topic: topic,
		Payload: []byte(`{"topic":"` + topic + `"}`), CreatedAt: time.Now().UTC(),
	}
}

func TestOutboxRelay_SuccessMarksPublished(t *testing.T) {
	repo := &fakeOutboxRepo{events: []domain.OutboxEvent{pendingEvent(1, "customer.created")}}
	pub := &fakeRawPublisher{}

	n, err := newTestRelay(repo, pub).relayBatch(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []string{"customer.created"}, pub.published)
	require.NotNil(t, repo.events[0].PublishedAt)
	assert.Zero(t, repo.events[0].Attempts)
}

func TestOutboxRelay_FailureBumpsAttemptsAndBackoff(t *testing.T) {
	repo := &fakeOutboxRepo{events: []domain.OutboxEvent{pendingEvent(1, "customer.created")}}
	pub := &fakeRawPublisher{failTopics: map[string]bool{"customer.created": true}}
	relay := newTestRelay(repo, pub)

	n, err := relay.relayBatch(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	ev := repo.events[0]
	assert.Nil(t, ev.PublishedAt)
	assert.Equal(t, 1, ev.Attempts)
	assert.Equal(t, "broker down", ev.LastError)
	require.NotNil(t, ev.NextAttemptAt)
	assert.True(t, ev.NextAttemptAt.After(time.Now().UTC()), "backoff deadline must be in the future")

	// The row is backing off, so the next batch does not re-claim it.
	n, err = relay.relayBatch(context.Background())
	require.NoError(t, err)
	assert.Zero(t, n)
	assert.Equal(t, 1, repo.events[0].Attempts)
}

func TestOutboxRelay_FailingRowDoesNotBlockLaterRows(t *testing.T) {
	repo := &fakeOutboxRepo{events: []domain.OutboxEvent{
		pendingEvent(1, "bad.topic"),
		pendingEvent(2, "customer.created"),
	}}
	pub := &fakeRawPublisher{failTopics: map[string]bool{"bad.topic": true}}

	n, err := newTestRelay(repo, pub).relayBatch(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, []string{"customer.created"}, pub.published)
	assert.Nil(t, repo.events[0].PublishedAt)
	assert.NotNil(t, repo.events[1].PublishedAt)
}

func TestOutboxRelay_MaxAttemptsExcluded(t *testing.T) {
	ev := pendingEvent(1, "customer.created")
	ev.Attempts = outboxMaxAttempts
	repo := &fakeOutboxRepo{events: []domain.OutboxEvent{ev}}
	pub := &fakeRawPublisher{}

	n, err := newTestRelay(repo, pub).relayBatch(context.Background())
	require.NoError(t, err)
	assert.Zero(t, n)
	assert.Empty(t, pub.published)
}

func TestOutboxRelay_PurgeDeletesOnlyOldPublishedRows(t *testing.T) {
	old := time.Now().UTC().Add(-48 * time.Hour)
	fresh := time.Now().UTC()
	oldPublished := pendingEvent(1, "a")
	oldPublished.PublishedAt = &old
	freshPublished := pendingEvent(2, "b")
	freshPublished.PublishedAt = &fresh
	pending := pendingEvent(3, "c")
	repo := &fakeOutboxRepo{events: []domain.OutboxEvent{oldPublished, freshPublished, pending}}

	purged, err := repo.PurgePublished(context.Background(), time.Now().UTC().Add(-24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(1), purged)
	assert.Len(t, repo.events, 2)
}

func TestOutboxBackoff(t *testing.T) {
	assert.Equal(t, time.Second, outboxBackoff(0))
	assert.Equal(t, 2*time.Second, outboxBackoff(1))
	assert.Equal(t, 64*time.Second, outboxBackoff(6))
	assert.Equal(t, outboxBackoffCap, outboxBackoff(20), "large attempt counts cap out")
}
