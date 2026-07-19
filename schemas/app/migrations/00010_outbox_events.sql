-- +goose Up
-- Transactional outbox for domain events. No status column — state is derived
-- from published_at/attempts.
CREATE TABLE outbox_events (
    id              BIGSERIAL PRIMARY KEY,
    event_id        TEXT        NOT NULL,
    org_id          TEXT        NOT NULL,
    topic           TEXT        NOT NULL,
    payload         JSONB       NOT NULL,
    attempts        INT         NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ,
    last_error      TEXT,
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX outbox_events_pending_idx ON outbox_events (id) WHERE published_at IS NULL;
-- +goose Down
DROP TABLE outbox_events;
