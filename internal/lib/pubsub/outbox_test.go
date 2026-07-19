package pubsub

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
)

func TestOutbox_PublishWritesEnvelopeRow(t *testing.T) {
	repo := &fakeOutboxRepo{}
	ps := NewOutbox(repo, &fakeDelegate{})

	require.NoError(t, ps.Publish(context.Background(), "org_1", "customer.created", domain.Customer{Id: "cus_1", OrgId: "org_1"}))

	require.Len(t, repo.events, 1)
	ev := repo.events[0]
	assert.Equal(t, "org_1", ev.OrgId)
	assert.Equal(t, "customer.created", ev.Topic)
	assert.NotEmpty(t, ev.EventId)
	assert.Nil(t, ev.PublishedAt)

	var envelope domain.PubSubPayload
	require.NoError(t, json.Unmarshal(ev.Payload, &envelope))
	assert.Equal(t, ev.EventId, envelope.Id)
	assert.Equal(t, "org_1", envelope.OrgId)
	assert.Equal(t, "customer.created", envelope.Topic)
	assert.NotNil(t, envelope.CreatedAt)
}

func TestOutbox_SubscribeDelegates(t *testing.T) {
	delegate := &fakeDelegate{}
	ps := NewOutbox(&fakeOutboxRepo{}, delegate)

	_, err := ps.Subscribe("customer.>", func(string, []byte) {})
	require.NoError(t, err)
	assert.Equal(t, []string{"customer.>"}, delegate.subscribed)
}
