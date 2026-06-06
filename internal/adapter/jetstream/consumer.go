package jetstream

import (
	"context"
	"encoding/json"
	"runtime/debug"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

const (
	ackWait       = 30 * time.Second // redeliver if a write doesn't ack in time
	maxDeliver    = 5                // attempts before a message is dead-lettered
	fetchMaxWait  = time.Second      // flush interval: how long Fetch waits to fill a batch
	idleBackoff   = time.Second      // pause after a fetch error before retrying
	closeGraceTTL = 10 * time.Second
)

// Consumer drains the usage-event work queue into the EventStore in batches. One
// durable consumer; out-of-order delivery is safe (aggregations key on event time).
type Consumer struct {
	store     port.EventStore
	consumer  jetstream.Consumer
	batchSize int
	logger    port.Logger
	cancel    context.CancelFunc
	done      chan struct{}
}

// NewConsumer ensures the stream + durable consumer exist and starts the drain loop.
func NewConsumer(ctx context.Context, store port.EventStore, js jetstream.JetStream, batchSize int, logger port.Logger) (*Consumer, error) {
	if batchSize < 1 {
		batchSize = 1
	}
	if err := EnsureStream(ctx, js); err != nil {
		return nil, err
	}
	cons, err := js.CreateOrUpdateConsumer(ctx, StreamName, jetstream.ConsumerConfig{
		Durable:       ConsumerName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       ackWait,
		MaxDeliver:    maxDeliver,
		FilterSubject: SubjectIngest,
	})
	if err != nil {
		return nil, err
	}

	runCtx, cancel := context.WithCancel(context.Background())
	c := &Consumer{store: store, consumer: cons, batchSize: batchSize, logger: logger, cancel: cancel, done: make(chan struct{})}
	go c.run(runCtx)
	logger.Infof("[jetstream] usage consumer started (batch=%d)", batchSize)
	return c, nil
}

func (c *Consumer) run(ctx context.Context) {
	defer close(c.done)
	for {
		if ctx.Err() != nil {
			return
		}
		c.drainOnce(ctx)
	}
}

// drainOnce fetches up to batchSize messages, writes them in one IngestBatch, and
// acks/naks. Wrapped so a panic on one batch can't kill the loop.
func (c *Consumer) drainOnce(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			c.logger.Error("[jetstream] consumer panic recovered", "recover", r, "stack", string(debug.Stack()))
		}
	}()

	batch, err := c.consumer.Fetch(c.batchSize, jetstream.FetchMaxWait(fetchMaxWait))
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		c.logger.Warn("[jetstream] fetch failed", "err", err.Error())
		sleep(ctx, idleBackoff)
		return
	}

	var msgs []jetstream.Msg
	var events []domain.MeterEvent
	for msg := range batch.Messages() {
		var e domain.MeterEvent
		if err := json.Unmarshal(msg.Data(), &e); err != nil {
			// Unparseable message can never succeed — terminate it (dead-letter)
			// rather than redeliver forever.
			c.logger.Error("[jetstream] drop unparseable usage event", "err", err.Error())
			_ = msg.Term()
			continue
		}
		msgs = append(msgs, msg)
		events = append(events, e)
	}
	if err := batch.Error(); err != nil && ctx.Err() == nil {
		c.logger.Warn("[jetstream] batch error", "err", err.Error())
	}
	if len(events) == 0 {
		return
	}

	if _, err := c.store.IngestBatch(ctx, events); err != nil {
		// Transient (DB) failure — nak so JetStream redelivers (up to MaxDeliver,
		// then dead-letters). The external_id unique index keeps retries idempotent.
		c.logger.Error("[jetstream] batch ingest failed; nak for redelivery", "count", len(events), "err", err.Error())
		for _, m := range msgs {
			_ = m.Nak()
		}
		return
	}
	for _, m := range msgs {
		_ = m.Ack()
	}
}

func (c *Consumer) Close() error {
	c.cancel()
	select {
	case <-c.done:
	case <-time.After(closeGraceTTL):
		c.logger.Warn("[jetstream] usage consumer did not stop within grace period")
	}
	return nil
}

func sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
