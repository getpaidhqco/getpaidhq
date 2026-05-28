package postgres

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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

// Claim inserts the idempotency key with ON CONFLICT DO NOTHING. Postgres
// returns RowsAffected=1 when the row was newly created and 0 when an
// existing row pre-empted it. That single round-trip is the entire
// race-free claim — no read-then-write split, no constraint-violation
// 500 leaking back to the PSP.
//
// If a row exists but is past its expires_at, we treat it as a fresh
// claim by upserting on expiry. (Replaying a 24-hour-old webhook should
// be processed again, not silently dropped.) Without that, expired keys
// would block reprocessing forever.
func (r *IdempotencyKeyRepo) Claim(ctx context.Context, key string, expiresAt time.Time) (bool, error) {
	now := time.Now().UTC()
	entity := IdempotencyKeyEntity{Key: key, ExpiresAt: expiresAt}

	// First, sweep any expired rows for this key so the upsert path can
	// claim afresh. This is cheap (PK lookup) and only deletes if past
	// expiry; concurrent claimers race here but the outer upsert is the
	// real arbiter.
	if err := dbFromCtx(ctx, r.db).
		Where("key = ? AND expires_at <= ?", key, now).
		Delete(&IdempotencyKeyEntity{}).Error; err != nil {
		return false, err
	}

	res := dbFromCtx(ctx, r.db).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&entity)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// Release deletes the idempotency row so the PSP's next retry can claim
// it afresh. Caller decides when releasing is appropriate (i.e. only
// after a *transient* failure that the PSP might be able to retry past).
// Idempotent — deleting a missing key is not an error.
func (r *IdempotencyKeyRepo) Release(ctx context.Context, key string) error {
	return dbFromCtx(ctx, r.db).
		Where("key = ?", key).
		Delete(&IdempotencyKeyEntity{}).Error
}
