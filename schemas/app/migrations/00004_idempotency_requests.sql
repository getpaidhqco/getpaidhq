-- +goose Up
CREATE TABLE "idempotency_requests" (
    "key"              TEXT        NOT NULL,
    "request_hash"     TEXT        NOT NULL,
    "state"            TEXT        NOT NULL,
    "token"            TEXT        NOT NULL,
    "response_code"    INTEGER,
    "response_headers" BYTEA,
    "response_body"    BYTEA,
    "expires_at"       TIMESTAMP(3) NOT NULL,
    "created_at"       TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at"       TIMESTAMP(3) NOT NULL,

    CONSTRAINT "idempotency_requests_pkey" PRIMARY KEY ("key")
);
CREATE INDEX "idempotency_requests_expires_at" ON "idempotency_requests" ("expires_at");

-- +goose Down
DROP TABLE "idempotency_requests";
