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

	defaultPurgeInterval = 10 * time.Minute
	defaultRetention     = 24 * time.Hour
)

// Relay drains the outbox to the broker. Rows are claimed FOR UPDATE SKIP
// LOCKED and published inside the claiming transaction: a crash between
// publish and commit republishes the row — at-least-once delivery.
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

func (r *Relay) relayBatch(ctx context.Context) (int, error) {
	var claimed int
	err := r.tx.RunInTx(ctx, func(ctx context.Context) error {
		now := time.Now().UTC()
		events, err := r.repo.ClaimPending(ctx, relayBatchSize, relayMaxAttempts, now)
		if err != nil {
			return err
		}
		claimed = len(events)
		for _, ev := range events {
			if err := r.publisher.PublishPayload(ev.Topic, ev.Payload); err != nil {
				if recErr := r.repo.RecordFailure(ctx, ev.Id, err.Error(), now.Add(backoff(ev.Attempts))); recErr != nil {
					return recErr
				}
				if ev.Attempts+1 >= relayMaxAttempts {
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

func backoff(attempts int) time.Duration {
	d := relayBackoffBase << attempts
	if d <= 0 || d > relayBackoffCap {
		return relayBackoffCap
	}
	return d
}
