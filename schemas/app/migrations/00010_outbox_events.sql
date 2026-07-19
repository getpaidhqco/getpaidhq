-- +goose Up
-- Transactional outbox for domain events: rows are written in the same
-- transaction as the business write and delivered to NATS by an in-process
-- relay. There is no status column — state is derived: pending
-- (published_at IS NULL AND attempts < max), failed / left for inspection
-- (published_at IS NULL AND attempts >= max), published (published_at IS NOT
-- NULL, purged after the retention window).
CREATE TABLE outbox_events (
    id              BIGSERIAL PRIMARY KEY,        -- publish order
    event_id        TEXT        NOT NULL,         -- evt_<id>, stable envelope id
    org_id          TEXT        NOT NULL,
    topic           TEXT        NOT NULL,
    payload         JSONB       NOT NULL,         -- full PubSubPayload envelope
    attempts        INT         NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ,
    last_error      TEXT,
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX outbox_events_pending_idx ON outbox_events (id) WHERE published_at IS NULL;
-- +goose Down
DROP TABLE outbox_events;
