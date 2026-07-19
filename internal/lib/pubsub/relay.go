package pubsub

import (
	"context"
	"time"

	"getpaidhq/internal/core/port"
)

const (
	relayPollInterval = time.Second
	relayBatchSize    = 100
	relayMaxAttempts  = 10
	relayBackoffBase  = time.Second
	relayBackoffCap   = 5 * time.Minute

	defaultPurgeInterval = 1 * time.Hour
	defaultRetention     = 24 * time.Hour
)

// Relay drains the outbox to the broker. Each row is claimed FOR UPDATE SKIP
// LOCKED and published inside its own transaction: a crash between publish
// and commit republishes that one row — at-least-once delivery.
type Relay struct {
	tx        port.TxManager
	repo      port.OutboxRepository
	publisher port.RawPublisher
	logger    port.Logger

	purgeInterval time.Duration
	retention     time.Duration

	stop chan struct{}
	done chan struct{}
}

// NewRelay builds a relay; purgeInterval and retention fall back to 10m / 24h
// when zero.
func NewRelay(tx port.TxManager, repo port.OutboxRepository, publisher port.RawPublisher, logger port.Logger, purgeInterval, retention time.Duration) *Relay {
	if purgeInterval <= 0 {
		purgeInterval = defaultPurgeInterval
	}
	if retention <= 0 {
		retention = defaultRetention
	}
	return &Relay{
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

func (r *Relay) Start() {
	go r.run()
}

// Close stops the relay and waits for the in-flight batch to finish.
func (r *Relay) Close() error {
	close(r.stop)
	<-r.done
	return nil
}

func (r *Relay) run() {
	defer close(r.done)
	poll := time.NewTicker(relayPollInterval)
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

// drain processes full batches until fewer than a batch remains.
func (r *Relay) drain() {
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
		if n < relayBatchSize {
			return
		}
	}
}

// relayBatch relays up to relayBatchSize rows, one transaction per row, so a
// failed mark rolls back only its own row — never the marks of events already
// published earlier in the batch.
func (r *Relay) relayBatch(ctx context.Context) (int, error) {
	for i := 0; i < relayBatchSize; i++ {
		claimed, err := r.relayNext(ctx)
		if err != nil {
			return i, err
		}
		if !claimed {
			return i, nil
		}
	}
	return relayBatchSize, nil
}

// relayNext claims and delivers a single row in its own transaction.
func (r *Relay) relayNext(ctx context.Context) (bool, error) {
	var claimed bool
	err := r.tx.RunInTx(ctx, func(ctx context.Context) error {
		now := time.Now().UTC()
		events, err := r.repo.ClaimPending(ctx, 1, relayMaxAttempts, now)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			return nil
		}
		claimed = true
		ev := events[0]
		if err := r.publisher.PublishPayload(ev.Topic, ev.Payload); err != nil {
			if ev.Attempts+1 >= relayMaxAttempts {
				r.logger.Errorf("[outbox] event %s (topic %s) failed %d attempts, giving up: %v", ev.EventId, ev.Topic, ev.Attempts+1, err)
			}
			return r.repo.RecordFailure(ctx, ev.Id, err.Error(), now.Add(backoff(ev.Attempts)))
		}
		return r.repo.MarkPublished(ctx, ev.Id, now)
	})
	return claimed, err
}

func backoff(attempts int) time.Duration {
	d := relayBackoffBase << attempts
	if d <= 0 || d > relayBackoffCap {
		return relayBackoffCap
	}
	return d
}
