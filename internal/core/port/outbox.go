package port

import (
	"context"
	"time"

	"getpaidhq/internal/core/domain"
)

// OutboxRepository persists queued domain events. Create joins the ambient
// transaction on ctx, so an event inserted inside RunInTx commits and rolls
// back with the business write.
type OutboxRepository interface {
	Create(ctx context.Context, event domain.OutboxEvent) error

	// ClaimPending selects up to limit due, unpublished rows in insertion
	// order, locked FOR UPDATE SKIP LOCKED. Call inside RunInTx.
	ClaimPending(ctx context.Context, limit int, maxAttempts int, now time.Time) ([]domain.OutboxEvent, error)

	MarkPublished(ctx context.Context, id int64, at time.Time) error

	// RecordFailure increments attempts and stores the error and backoff deadline.
	RecordFailure(ctx context.Context, id int64, lastError string, nextAttemptAt time.Time) error

	// PurgePublished deletes published rows older than the cutoff.
	PurgePublished(ctx context.Context, olderThan time.Time) (int64, error)
}
