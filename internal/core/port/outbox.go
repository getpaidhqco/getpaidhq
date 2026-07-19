package port

import (
	"context"
	"time"

	"getpaidhq/internal/core/domain"
)

// OutboxRepository persists queued domain events for the transactional outbox.
// Create joins the ambient transaction on ctx (dbFromCtx), so an event
// inserted inside RunInTx commits and rolls back with the business write.
type OutboxRepository interface {
	Create(ctx context.Context, event domain.OutboxEvent) error

	// ClaimPending selects up to limit unpublished rows due for delivery
	// (attempts < maxAttempts and next_attempt_at unset or <= now) in
	// insertion order, locked FOR UPDATE SKIP LOCKED. Must be called inside a
	// RunInTx transaction; the lock is what makes concurrent relays safe.
	ClaimPending(ctx context.Context, limit int, maxAttempts int, now time.Time) ([]domain.OutboxEvent, error)

	// MarkPublished stamps published_at on a delivered row.
	MarkPublished(ctx context.Context, id int64, at time.Time) error

	// RecordFailure increments attempts and stores the error and the backoff
	// deadline for the next delivery attempt.
	RecordFailure(ctx context.Context, id int64, lastError string, nextAttemptAt time.Time) error

	// PurgePublished deletes published rows older than the cutoff and reports
	// how many were removed.
	PurgePublished(ctx context.Context, olderThan time.Time) (int64, error)
}
