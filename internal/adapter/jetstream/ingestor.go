package jetstream

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// dedupWindow is how long JetStream remembers a Nats-Msg-Id to drop a resend before
// it ever reaches the consumer/DB.
const dedupWindow = 2 * time.Minute

// Ingestor is the durable write path: it publishes a validated event to JetStream and
// returns immediately once the broker has persisted it. Implements port.EventIngestor.
type Ingestor struct {
	js     jetstream.JetStream
	logger port.Logger
}

func NewIngestor(js jetstream.JetStream, logger port.Logger) *Ingestor {
	return &Ingestor{js: js, logger: logger}
}

var _ port.EventIngestor = (*Ingestor)(nil)

// Ingest publishes the event durably (synchronous publish: it waits for the stream
// ack, so a returned nil error means the event is persisted in JetStream). The write
// into the EventStore happens later in the consumer, so the status is "accepted".
func (i *Ingestor) Ingest(ctx context.Context, e domain.MeterEvent) (port.IngestResult, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return port.IngestResult{}, fmt.Errorf("jetstream: marshal usage event: %w", err)
	}
	if _, err := i.js.Publish(ctx, SubjectIngest, data, jetstream.WithMsgID(msgID(e))); err != nil {
		return port.IngestResult{}, fmt.Errorf("jetstream: publish usage event: %w", err)
	}
	return port.IngestResult{Id: e.Id, Status: port.IngestAccepted}, nil
}

// msgID is the JetStream dedup key: the client's external_id when set (so a client
// resend collapses), else the event's own id (idempotent across publish retries).
// Mirrors the ClickHouse read-time dedup_key.
func msgID(e domain.MeterEvent) string {
	if e.ExternalId != "" {
		return e.ExternalId
	}
	return e.Id
}
