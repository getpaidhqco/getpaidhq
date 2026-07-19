package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
)

func TestOutboxPubSub_PublishWritesEnvelopeRow(t *testing.T) {
	repo := &fakeOutboxRepo{}
	ps := NewOutboxPubSub(repo, &fakePubSub{})

	require.NoError(t, ps.Publish(context.Background(), "org_1", "customer.created", domain.Customer{Id: "cus_1", OrgId: "org_1"}))

	require.Len(t, repo.events, 1)
	ev := repo.events[0]
	assert.Equal(t, "org_1", ev.OrgId)
	assert.Equal(t, "customer.created", ev.Topic)
	assert.NotEmpty(t, ev.EventId)
	assert.Nil(t, ev.PublishedAt)

	var envelope domain.PubSubPayload
	require.NoError(t, json.Unmarshal(ev.Payload, &envelope))
	assert.Equal(t, ev.EventId, envelope.Id, "envelope id and row event_id must match")
	assert.Equal(t, "org_1", envelope.OrgId)
	assert.Equal(t, "customer.created", envelope.Topic)
	assert.NotNil(t, envelope.CreatedAt, "created_at is stamped at insert time")
}

func TestOutboxPubSub_SubscribeDelegates(t *testing.T) {
	delegate := &fakePubSub{}
	ps := NewOutboxPubSub(&fakeOutboxRepo{}, delegate)

	_, err := ps.Subscribe("customer.>", func(string, []byte) {})
	require.NoError(t, err)
	assert.NotNil(t, delegate.handler, "subscription must reach the delegate broker")
}
