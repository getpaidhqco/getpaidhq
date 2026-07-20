package postgrespgx

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/port"
)

// IdempotencyStore is the pgx implementation of port.IdempotencyStore, backing
// the idempo middleware. It persists the captured response (code/headers/body)
// for replay and a fencing token so only the original claimant can complete or
// abandon the in-flight request.
type IdempotencyStore struct {
	pool         *pgxpool.Pool
	lockTTL      time.Duration
	retentionTTL time.Duration
}

func NewIdempotencyStore(pool *pgxpool.Pool, lockTTL, retentionTTL time.Duration) port.IdempotencyStore {
	return &IdempotencyStore{pool: pool, lockTTL: lockTTL, retentionTTL: retentionTTL}
}

// Claim is the race-free single-winner gate. The delete-expired-then-INSERT
// ON CONFLICT DO NOTHING pattern makes RowsAffected the sole arbiter: exactly
// one concurrent caller inserts the pending row and gets IdempotencyNew; the
// rest read the existing row and classify it.
//
// An expired row (pending lock past TTL, or completed past retention) is swept
// first so a stale lock can't wedge the key forever — the next caller reclaims
// it afresh as New.
func (s *IdempotencyStore) Claim(ctx context.Context, key, requestHash, token string) (port.IdempotencyClaim, error) {
	now := time.Now().UTC()
	q := dbFromCtx(ctx, s.pool)

	if _, err := q.Exec(ctx,
		`DELETE FROM idempotency_requests WHERE key = $1 AND expires_at <= $2`,
		key, now); err != nil {
		return port.IdempotencyClaim{}, err
	}

	tag, err := q.Exec(ctx,
		`INSERT INTO idempotency_requests (key, request_hash, state, token, expires_at, updated_at)
		 VALUES ($1, $2, 'pending', $3, $4, $5)
		 ON CONFLICT (key) DO NOTHING`,
		key, requestHash, token, now.Add(s.lockTTL), now)
	if err != nil {
		return port.IdempotencyClaim{}, err
	}
	if tag.RowsAffected() == 1 {
		return port.IdempotencyClaim{Status: port.IdempotencyNew}, nil
	}

	// A row pre-empted us; read it and classify.
	var (
		state   string
		hash    string
		code    *int
		headers []byte
		body    []byte
	)
	err = q.QueryRow(ctx,
		`SELECT state, request_hash, response_code, response_headers, response_body
		 FROM idempotency_requests WHERE key = $1`,
		key).Scan(&state, &hash, &code, &headers, &body)
	if err != nil {
		// The holder vanished via a concurrent TTL sweep between our failed
		// INSERT and this SELECT. Report Pending so the caller doesn't
		// double-run; the next attempt will reclaim it as New.
		if errors.Is(err, pgx.ErrNoRows) {
			return port.IdempotencyClaim{Status: port.IdempotencyPending}, nil
		}
		return port.IdempotencyClaim{}, err
	}

	switch state {
	case string(port.IdempotencyPending):
		return port.IdempotencyClaim{Status: port.IdempotencyPending}, nil
	case string(port.IdempotencyCompleted):
		if hash != requestHash {
			return port.IdempotencyClaim{Status: port.IdempotencyConflict}, nil
		}
		claim := port.IdempotencyClaim{
			Status:  port.IdempotencyCompleted,
			Headers: headers,
			Body:    body,
		}
		if code != nil {
			claim.Code = *code
		}
		return claim, nil
	default:
		return port.IdempotencyClaim{Status: port.IdempotencyConflict}, nil
	}
}

// Complete is a token-fenced, pending-only update that records the captured
// response and flips the row to completed (retained until retentionTTL). A wrong
// token or a non-pending row matches no rows and is a silent no-op — the claimant
// that actually holds the lock is the only one that can complete it.
func (s *IdempotencyStore) Complete(ctx context.Context, key, token string, statusCode int, headers, body []byte) error {
	now := time.Now().UTC()
	_, err := dbFromCtx(ctx, s.pool).Exec(ctx,
		`UPDATE idempotency_requests
		 SET state = 'completed', response_code = $3, response_headers = $4,
		     response_body = $5, expires_at = $6, updated_at = $7
		 WHERE key = $1 AND token = $2 AND state = 'pending'`,
		key, token, statusCode, headers, body, now.Add(s.retentionTTL), now)
	return err
}

// Abandon is a token-fenced, pending-only delete that releases the lock so a
// retry can claim afresh (used when the handler failed before completing). A
// wrong token or non-pending row matches nothing and is a silent no-op.
func (s *IdempotencyStore) Abandon(ctx context.Context, key, token string) error {
	_, err := dbFromCtx(ctx, s.pool).Exec(ctx,
		`DELETE FROM idempotency_requests WHERE key = $1 AND token = $2 AND state = 'pending'`,
		key, token)
	return err
}
