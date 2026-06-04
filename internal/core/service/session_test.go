package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type fakeSessionRepo struct {
	port.SessionRepository
	createErr error
	created   []domain.Session
}

func (r *fakeSessionRepo) Create(_ context.Context, s domain.Session) (domain.Session, error) {
	if r.createErr != nil {
		return domain.Session{}, r.createErr
	}
	r.created = append(r.created, s)
	return s, nil
}

func TestSessionService_CreateSession(t *testing.T) {
	t.Run("creates a cart then a session bound to it, and publishes", func(t *testing.T) {
		cart := &fakeCartRepo{}
		sess := &fakeSessionRepo{}
		ps := &recordingPubSub{}
		svc := NewSessionService(sess, cart, silentLogger{}, ps)

		got, err := svc.CreateSession(context.Background(), port.CreateSessionInput{OrgId: "org_1", Currency: "USD"})

		require.NoError(t, err)
		require.Len(t, cart.created, 1)
		require.Len(t, sess.created, 1)
		assert.Equal(t, cart.created[0].Id, got.CartId, "session references the new cart")
		assert.Equal(t, "USD", cart.created[0].Data.Currency)
		assert.True(t, ps.hasTopic(port.TopicSessionCreated))
	})

	t.Run("cart create failure aborts before session and event", func(t *testing.T) {
		cart := &fakeCartRepo{createErr: errors.New("db down")}
		sess := &fakeSessionRepo{}
		ps := &recordingPubSub{}
		svc := NewSessionService(sess, cart, silentLogger{}, ps)

		_, err := svc.CreateSession(context.Background(), port.CreateSessionInput{OrgId: "org_1", Currency: "USD"})

		require.Error(t, err)
		assert.Empty(t, sess.created)
		assert.False(t, ps.hasTopic(port.TopicSessionCreated))
	})
}
