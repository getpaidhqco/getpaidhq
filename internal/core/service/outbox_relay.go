package service

import (
	"context"
	"time"

	"getpaidhq/internal/core/port"
)

// Relay tuning. Poll/batch/attempt values are deliberately constants; the
// purge pass is configurable (see NewOutboxRelay).
const (
	outboxPollInterval = time.Second
	outboxBatchSize    = 100
	outboxMaxAttempts  = 10

	// Backoff for a failed row: base doubled per attempt, capped.
	outboxBackoffBase = time.Second
	outboxBackoffCap  = 5 * time.Minute

	defaultOutboxPurgeInterval = 10 * time.Minute
	defaultOutboxRetention     = 24 * time.Hour
)

// OutboxRelay drains the outbox to the broker: every poll it claims a batch
// of pending rows (FOR UPDATE SKIP LOCKED, so concurrent server instances are
// safe), publishes each stored envelope verbatim via the raw publisher, and
// marks the outcome. Publishing inside the lock-holding transaction is
// deliberate — a crash after publish but before commit republishes the row,
// giving at-least-once delivery. A failing row backs off exponentially and,
// at max attempts, is left for inspection without blocking later rows.
type OutboxRelay struct {
	tx        port.TxManager
	repo      port.OutboxRepository
	publisher port.RawPublisher
	logger    port.Logger

	purgeInterval time.Duration
	retention     time.Duration

	stop chan struct{}
	done chan struct{}
}

// NewOutboxRelay builds a relay. purgeInterval and retention fall back to
// 10m / 24h when zero (unset env).
func NewOutboxRelay(tx port.TxManager, repo port.OutboxRepository, publisher port.RawPublisher, logger port.Logger, purgeInterval, retention time.Duration) *OutboxRelay {
	if purgeInterval <= 0 {
		purgeInterval = defaultOutboxPurgeInterval
	}
	if retention <= 0 {
		retention = defaultOutboxRetention
	}
	return &OutboxRelay{
		tx:            tx,
		repo:          repo,
		publisher:     publisher,
		logger:        logger,
		purgeInterval: purgeInterval,
		retention:     retention,
		stop:          make(chan struct{}),
		done:          make(chan struct{}),
	}
}

// Start launches the relay loop. Call Close to stop it.
func (r *OutboxRelay) Start() {
	go r.run()
}

// Close stops the relay and waits for the in-flight batch to finish. Safe to
// call once; implements io.Closer for the app shutdown list.
func (r *OutboxRelay) Close() error {
	close(r.stop)
	<-r.done
	return nil
}

func (r *OutboxRelay) run() {
	defer close(r.done)
	poll := time.NewTicker(outboxPollInterval)
	defer poll.Stop()
	purge := time.NewTicker(r.purgeInterval)
	defer purge.Stop()

	for {
		select {
		case <-r.stop:
			return
		case <-poll.C:
			r.drain()
		case <-purge.C:
			if n, err := r.repo.PurgePublished(context.Background(), time.Now().UTC().Add(-r.retention)); err != nil {
				r.logger.Warnf("[outbox] purge failed: %v", err)
			} else if n > 0 {
				r.logger.Debugf("[outbox] purged %d published events", n)
			}
		}
	}
}

// drain processes full batches until the outbox has fewer pending rows than a
// batch, so a backlog clears faster than one batch per poll tick.
func (r *OutboxRelay) drain() {
	for {
		select {
		case <-r.stop:
			return
		default:
		}
		n, err := r.relayBatch(context.Background())
		if err != nil {
			r.logger.Warnf("[outbox] relay batch failed: %v", err)
			return
		}
		if n < outboxBatchSize {
			return
		}
	}
}

// relayBatch claims and delivers one batch inside a single transaction,
// returning how many rows were claimed.
func (r *OutboxRelay) relayBatch(ctx context.Context) (int, error) {
	var claimed int
	err := r.tx.RunInTx(ctx, func(ctx context.Context) error {
		now := time.Now().UTC()
		events, err := r.repo.ClaimPending(ctx, outboxBatchSize, outboxMaxAttempts, now)
		if err != nil {
			return err
		}
		claimed = len(events)
		for _, ev := range events {
			if err := r.publisher.PublishPayload(ev.Topic, ev.Payload); err != nil {
				backoff := outboxBackoff(ev.Attempts)
				if recErr := r.repo.RecordFailure(ctx, ev.Id, err.Error(), now.Add(backoff)); recErr != nil {
					return recErr
				}
				if ev.Attempts+1 >= outboxMaxAttempts {
					r.logger.Errorf("[outbox] event %s (topic %s) failed %d attempts, giving up: %v", ev.EventId, ev.Topic, ev.Attempts+1, err)
				}
				continue
			}
			if err := r.repo.MarkPublished(ctx, ev.Id, now); err != nil {
				return err
			}
		}
		return nil
	})
	return claimed, err
}

// outboxBackoff returns the delay before the next attempt after `attempts`
// prior failures: base * 2^attempts, capped.
func outboxBackoff(attempts int) time.Duration {
	d := outboxBackoffBase << attempts
	if d <= 0 || d > outboxBackoffCap {
		return outboxBackoffCap
	}
	return d
}
