// Package jetstream is the durable asynchronous usage-event ingestion backend. It
// sits behind port.EventIngestor: the Ingestor publishes validated events to a NATS
// JetStream work-queue stream (durable, file-backed), and the Consumer drains that
// stream into the EventStore in batches. Selected by USAGE_INGEST_MODE=jetstream.
package jetstream

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	// StreamName is the durable work queue holding accepted-but-unwritten events.
	StreamName = "USAGE_EVENTS"
	// SubjectIngest is where the Ingestor publishes and the Consumer reads.
	SubjectIngest = "usage.events.ingest"
	// ConsumerName is the single durable consumer that drains the stream.
	ConsumerName = "usage-event-writer"
	// maxStreamBytes bounds on-disk buffering; when full, publishes are rejected
	// (DiscardNew) so a stalled consumer applies visible backpressure instead of
	// filling the disk. Generous default; tune via ops.
	maxStreamBytes = 2 << 30 // 2 GiB
)

// EnsureStream idempotently creates/updates the work-queue stream. WorkQueue
// retention deletes a message once the consumer acks it (this is a pipeline, not a
// log — the EventStore is the system of record). FileStorage makes accepted events
// survive a broker restart. The dedup window collapses resends sharing a Nats-Msg-Id.
func EnsureStream(ctx context.Context, js jetstream.JetStream) error {
	_, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:       StreamName,
		Subjects:   []string{SubjectIngest},
		Retention:  jetstream.WorkQueuePolicy,
		Storage:    jetstream.FileStorage,
		Discard:    jetstream.DiscardNew,
		MaxBytes:   maxStreamBytes,
		Duplicates: dedupWindow,
	})
	if err != nil {
		return fmt.Errorf("jetstream: ensure stream %s: %w", StreamName, err)
	}
	return nil
}
