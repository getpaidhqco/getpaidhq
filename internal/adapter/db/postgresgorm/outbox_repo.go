package postgresgorm

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type OutboxRepo struct {
	db *gorm.DB
}

func NewOutboxRepo(db *gorm.DB) port.OutboxRepository {
	return &OutboxRepo{db: db}
}

func (r *OutboxRepo) Create(ctx context.Context, ev domain.OutboxEvent) error {
	row := outboxEventRowFromDomain(ev)
	row.Id = 0 // BIGSERIAL assigns publish order
	return dbFromCtx(ctx, r.db).Create(&row).Error
}

func (r *OutboxRepo) ClaimPending(ctx context.Context, limit int, maxAttempts int, now time.Time) ([]domain.OutboxEvent, error) {
	var rows []outboxEventRow
	err := dbFromCtx(ctx, r.db).
		Where("published_at IS NULL AND attempts < ? AND (next_attempt_at IS NULL OR next_attempt_at <= ?)", maxAttempts, now).
		Order("id").
		Limit(limit).
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.OutboxEvent, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out, nil
}

func (r *OutboxRepo) MarkPublished(ctx context.Context, id int64, at time.Time) error {
	return dbFromCtx(ctx, r.db).
		Model(&outboxEventRow{}).
		Where("id = ?", id).
		Update("published_at", at).Error
}

func (r *OutboxRepo) RecordFailure(ctx context.Context, id int64, lastError string, nextAttemptAt time.Time) error {
	return dbFromCtx(ctx, r.db).
		Model(&outboxEventRow{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"attempts":        gorm.Expr("attempts + 1"),
			"last_error":      lastError,
			"next_attempt_at": nextAttemptAt,
		}).Error
}

func (r *OutboxRepo) PurgePublished(ctx context.Context, olderThan time.Time) (int64, error) {
	res := dbFromCtx(ctx, r.db).
		Where("published_at IS NOT NULL AND published_at < ?", olderThan).
		Delete(&outboxEventRow{})
	return res.RowsAffected, res.Error
}
