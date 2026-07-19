package storagetest

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

const outboxTestMaxAttempts = 10

func outboxEvent(orgId, topic string) domain.OutboxEvent {
	return domain.OutboxEvent{
		EventId:   lib.GenerateId("evt"),
		OrgId:     orgId,
		Topic:     topic,
		Payload:   []byte(`{"id":"x","org_id":"` + orgId + `","topic":"` + topic + `","data":{"n":1}}`),
		CreatedAt: now(),
	}
}

// drainOutbox empties the table so sub-scenarios don't observe each other's
// rows (ClaimPending is not org-scoped).
func drainOutbox(t *testing.T, ctx context.Context, rs RepoSet) {
	t.Helper()
	err := rs.Tx.RunInTx(ctx, func(ctx context.Context) error {
		evs, err := rs.Outbox.ClaimPending(ctx, 1000, 1<<30, now().Add(365*24*time.Hour))
		if err != nil {
			return err
		}
		for _, ev := range evs {
			if err := rs.Outbox.MarkPublished(ctx, ev.Id, now().Add(-365*24*time.Hour)); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)
	_, err = rs.Outbox.PurgePublished(ctx, now())
	require.NoError(t, err)
}

func testOutbox(t *testing.T, ctx context.Context, rs RepoSet) {
	orgId := seedOrg(t, ctx, rs)

	t.Run("CreateAndClaimInOrder", func(t *testing.T) {
		drainOutbox(t, ctx, rs)
		first := outboxEvent(orgId, "customer.created")
		second := outboxEvent(orgId, "customer.updated")
		require.NoError(t, rs.Outbox.Create(ctx, first))
		require.NoError(t, rs.Outbox.Create(ctx, second))

		err := rs.Tx.RunInTx(ctx, func(ctx context.Context) error {
			evs, err := rs.Outbox.ClaimPending(ctx, 10, outboxTestMaxAttempts, now())
			require.NoError(t, err)
			require.Len(t, evs, 2)
			assert.Equal(t, first.EventId, evs[0].EventId, "insertion order")
			assert.Equal(t, second.EventId, evs[1].EventId)
			assert.Equal(t, orgId, evs[0].OrgId)
			assert.Equal(t, "customer.created", evs[0].Topic)
			// jsonb normalizes encoding — compare semantically.
			assert.JSONEq(t, string(first.Payload), string(evs[0].Payload))
			assert.Zero(t, evs[0].Attempts)
			assert.Nil(t, evs[0].PublishedAt)
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("RolledBackTxLeavesNoRow", func(t *testing.T) {
		drainOutbox(t, ctx, rs)
		boom := errors.New("boom")
		err := rs.Tx.RunInTx(ctx, func(ctx context.Context) error {
			if err := rs.Outbox.Create(ctx, outboxEvent(orgId, "order.completed")); err != nil {
				return err
			}
			return boom
		})
		require.ErrorIs(t, err, boom)

		evs, err := rs.Outbox.ClaimPending(ctx, 10, outboxTestMaxAttempts, now())
		require.NoError(t, err)
		assert.Empty(t, evs, "rolled-back insert must leave no row")
	})

	t.Run("CommittedTxLeavesRow", func(t *testing.T) {
		drainOutbox(t, ctx, rs)
		ev := outboxEvent(orgId, "order.completed")
		require.NoError(t, rs.Tx.RunInTx(ctx, func(ctx context.Context) error {
			return rs.Outbox.Create(ctx, ev)
		}))
		evs, err := rs.Outbox.ClaimPending(ctx, 10, outboxTestMaxAttempts, now())
		require.NoError(t, err)
		require.Len(t, evs, 1)
		assert.Equal(t, ev.EventId, evs[0].EventId)
	})

	t.Run("MarkPublishedExcludesRow", func(t *testing.T) {
		drainOutbox(t, ctx, rs)
		require.NoError(t, rs.Outbox.Create(ctx, outboxEvent(orgId, "a.b")))
		evs, err := rs.Outbox.ClaimPending(ctx, 10, outboxTestMaxAttempts, now())
		require.NoError(t, err)
		require.Len(t, evs, 1)

		require.NoError(t, rs.Outbox.MarkPublished(ctx, evs[0].Id, now()))
		evs, err = rs.Outbox.ClaimPending(ctx, 10, outboxTestMaxAttempts, now())
		require.NoError(t, err)
		assert.Empty(t, evs)
	})

	t.Run("FailureBackoffAndMaxAttempts", func(t *testing.T) {
		drainOutbox(t, ctx, rs)
		require.NoError(t, rs.Outbox.Create(ctx, outboxEvent(orgId, "a.b")))
		evs, err := rs.Outbox.ClaimPending(ctx, 10, outboxTestMaxAttempts, now())
		require.NoError(t, err)
		require.Len(t, evs, 1)
		id := evs[0].Id

		next := now().Add(time.Hour)
		require.NoError(t, rs.Outbox.RecordFailure(ctx, id, "broker down", next))
		evs, err = rs.Outbox.ClaimPending(ctx, 10, outboxTestMaxAttempts, now())
		require.NoError(t, err)
		assert.Empty(t, evs, "row inside backoff window must be excluded")

		evs, err = rs.Outbox.ClaimPending(ctx, 10, outboxTestMaxAttempts, next)
		require.NoError(t, err)
		require.Len(t, evs, 1)
		assert.Equal(t, 1, evs[0].Attempts)
		assert.Equal(t, "broker down", evs[0].LastError)
		require.NotNil(t, evs[0].NextAttemptAt)

		for i := 1; i < outboxTestMaxAttempts; i++ {
			require.NoError(t, rs.Outbox.RecordFailure(ctx, id, "still down", next))
		}
		evs, err = rs.Outbox.ClaimPending(ctx, 10, outboxTestMaxAttempts, next.Add(time.Hour))
		require.NoError(t, err)
		assert.Empty(t, evs, "row at max attempts must be excluded")
	})

	t.Run("ClaimSkipsLockedRows", func(t *testing.T) {
		drainOutbox(t, ctx, rs)
		require.NoError(t, rs.Outbox.Create(ctx, outboxEvent(orgId, "a.b")))

		locked := make(chan struct{})
		release := make(chan struct{})
		errCh := make(chan error, 1)
		go func() {
			errCh <- rs.Tx.RunInTx(context.Background(), func(ctx context.Context) error {
				evs, err := rs.Outbox.ClaimPending(ctx, 10, outboxTestMaxAttempts, now())
				if err != nil {
					return err
				}
				if len(evs) != 1 {
					return errors.New("first claimer expected the row")
				}
				close(locked)
				<-release
				return nil
			})
		}()

		<-locked
		err := rs.Tx.RunInTx(ctx, func(ctx context.Context) error {
			evs, err := rs.Outbox.ClaimPending(ctx, 10, outboxTestMaxAttempts, now())
			require.NoError(t, err)
			assert.Empty(t, evs, "locked rows must be skipped, not waited on")
			return nil
		})
		require.NoError(t, err)
		close(release)
		require.NoError(t, <-errCh)
	})

	t.Run("PurgeRespectsRetention", func(t *testing.T) {
		drainOutbox(t, ctx, rs)
		oldEv := outboxEvent(orgId, "old.published")
		freshEv := outboxEvent(orgId, "fresh.published")
		pendingEv := outboxEvent(orgId, "still.pending")
		require.NoError(t, rs.Outbox.Create(ctx, oldEv))
		require.NoError(t, rs.Outbox.Create(ctx, freshEv))
		require.NoError(t, rs.Outbox.Create(ctx, pendingEv))

		evs, err := rs.Outbox.ClaimPending(ctx, 10, outboxTestMaxAttempts, now())
		require.NoError(t, err)
		require.Len(t, evs, 3)
		require.NoError(t, rs.Outbox.MarkPublished(ctx, evs[0].Id, now().Add(-48*time.Hour)))
		require.NoError(t, rs.Outbox.MarkPublished(ctx, evs[1].Id, now()))

		purged, err := rs.Outbox.PurgePublished(ctx, now().Add(-24*time.Hour))
		require.NoError(t, err)
		assert.Equal(t, int64(1), purged, "only published rows past retention are purged")

		evs, err = rs.Outbox.ClaimPending(ctx, 10, outboxTestMaxAttempts, now())
		require.NoError(t, err)
		require.Len(t, evs, 1, "pending row survives the purge")
		assert.Equal(t, pendingEv.EventId, evs[0].EventId)
	})
}
