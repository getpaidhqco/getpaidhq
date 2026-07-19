package postgresgorm

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"getpaidhq/internal/core/port"
)

// IdempotencyRequestEntity maps to the idempotency_requests table backing the
// idempo middleware. Distinct from idempotency_keys: this one persists the
// captured response (code/headers/body) for replay and a fencing token so only
// the original claimant can complete or abandon the in-flight request.
type IdempotencyRequestEntity struct {
	Key             string    `gorm:"column:key;primaryKey"`
	RequestHash     string    `gorm:"column:request_hash"`
	State           string    `gorm:"column:state"`
	Token           string    `gorm:"column:token"`
	ResponseCode    *int      `gorm:"column:response_code"`
	ResponseHeaders []byte    `gorm:"column:response_headers"`
	ResponseBody    []byte    `gorm:"column:response_body"`
	ExpiresAt       time.Time `gorm:"column:expires_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func (IdempotencyRequestEntity) TableName() string { return "idempotency_requests" }

type IdempotencyStore struct {
	db           *gorm.DB
	lockTTL      time.Duration
	retentionTTL time.Duration
}

func NewIdempotencyStore(db *gorm.DB, lockTTL, retentionTTL time.Duration) port.IdempotencyStore {
	return &IdempotencyStore{db: db, lockTTL: lockTTL, retentionTTL: retentionTTL}
}

var _ port.IdempotencyStore = (*IdempotencyStore)(nil)

// Claim is the race-free single-winner gate. The delete-expired-then-INSERT
// ON CONFLICT DO NOTHING pattern (same as IdempotencyKeyRepo) makes RowsAffected
// the sole arbiter: exactly one concurrent caller inserts the pending row and
// gets IdempotencyNew; the rest read the existing row.
//
// An expired row (pending lock past TTL, or completed past retention) is swept
// first so a stale lock can't wedge the key forever — the next caller reclaims
// it afresh as New.
func (s *IdempotencyStore) Claim(ctx context.Context, key, requestHash, token string) (port.IdempotencyClaim, error) {
	now := time.Now().UTC()

	if err := dbFromCtx(ctx, s.db).
		Where("key = ? AND expires_at <= ?", key, now).
		Delete(&IdempotencyRequestEntity{}).Error; err != nil {
		return port.IdempotencyClaim{}, err
	}

	entity := IdempotencyRequestEntity{
		Key:         key,
		RequestHash: requestHash,
		State:       string(port.IdempotencyPending),
		Token:       token,
		ExpiresAt:   now.Add(s.lockTTL),
	}
	res := dbFromCtx(ctx, s.db).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&entity)
	if res.Error != nil {
		return port.IdempotencyClaim{}, res.Error
	}
	if res.RowsAffected == 1 {
		return port.IdempotencyClaim{Status: port.IdempotencyNew}, nil
	}

	// A row pre-empted us; read it and classify.
	var existing IdempotencyRequestEntity
	if err := dbFromCtx(ctx, s.db).
		Where("key = ?", key).
		First(&existing).Error; err != nil {
		// The row was swept (expired) between our failed insert and this read.
		// Mirror the pgx adapter: report Pending (idempo → 409, client retries)
		// rather than surfacing a 500, keeping the two drivers at parity.
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return port.IdempotencyClaim{Status: port.IdempotencyPending}, nil
		}
		return port.IdempotencyClaim{}, err
	}

	switch existing.State {
	case string(port.IdempotencyPending):
		return port.IdempotencyClaim{Status: port.IdempotencyPending}, nil
	case string(port.IdempotencyCompleted):
		if existing.RequestHash != requestHash {
			return port.IdempotencyClaim{Status: port.IdempotencyConflict}, nil
		}
		claim := port.IdempotencyClaim{
			Status:  port.IdempotencyCompleted,
			Headers: existing.ResponseHeaders,
			Body:    existing.ResponseBody,
		}
		if existing.ResponseCode != nil {
			claim.Code = *existing.ResponseCode
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
	return dbFromCtx(ctx, s.db).
		Model(&IdempotencyRequestEntity{}).
		Where("key = ? AND token = ? AND state = ?", key, token, string(port.IdempotencyPending)).
		Updates(map[string]any{
			"state":            string(port.IdempotencyCompleted),
			"response_code":    statusCode,
			"response_headers": headers,
			"response_body":    body,
			"expires_at":       now.Add(s.retentionTTL),
			"updated_at":       now,
		}).Error
}

// Abandon is a token-fenced, pending-only delete that releases the lock so a
// retry can claim afresh (used when the handler failed before completing). A
// wrong token or non-pending row matches nothing and is a silent no-op.
func (s *IdempotencyStore) Abandon(ctx context.Context, key, token string) error {
	return dbFromCtx(ctx, s.db).
		Where("key = ? AND token = ? AND state = ?", key, token, string(port.IdempotencyPending)).
		Delete(&IdempotencyRequestEntity{}).Error
}
