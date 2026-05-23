package postgres

import (
	"context"
	"time"

	"gorm.io/gorm"
	"getpaidhq/internal/core/port"
)

type IdempotencyKeyEntity struct {
	Key       string    `gorm:"primaryKey"`
	ExpiresAt time.Time `gorm:"index"`
}

func (IdempotencyKeyEntity) TableName() string {
	return "idempotency_keys"
}

type IdempotencyKeyRepo struct {
	db *gorm.DB
}

func NewIdempotencyKeyRepo(db *gorm.DB) port.IdempotencyKeyRepository {
	return &IdempotencyKeyRepo{db: db}
}

func (r *IdempotencyKeyRepo) Exists(ctx context.Context, key string) (bool, error) {
	var count int64
	err := dbFromCtx(ctx, r.db).
		Model(&IdempotencyKeyEntity{}).
		Where("key = ? AND expires_at > ?", key, time.Now()).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *IdempotencyKeyRepo) Create(ctx context.Context, key string, expiresAt time.Time) error {
	entity := IdempotencyKeyEntity{
		Key:       key,
		ExpiresAt: expiresAt,
	}
	return dbFromCtx(ctx, r.db).Create(&entity).Error
}
