package domain

import "time"

// ApiKey is the stored record for an API key. The raw secret is NEVER
// persisted — only the HMAC of it (computed with a server-side pepper)
// is. Callers receive the raw secret exactly once when the key is
// created; after that, only KeyHash is available.
type ApiKey struct {
	OrgId   string `gorm:"column:org_id;primaryKey" json:"org_id" validate:"required"`
	Id      string `gorm:"column:id;primaryKey" json:"id" validate:"required"`
	KeyHash string `gorm:"column:key_hash;unique" json:"-" validate:"required"`

	CreatedAt time.Time `gorm:"column:created_at" json:"created_at" validate:"required"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at" validate:"required"`

	// RawKey is the plaintext key, populated only by the constructor
	// returned to the caller of Create — never read back from the DB
	// and never serialized to logs. Tagged `gorm:"-"` so GORM ignores
	// it on writes.
	RawKey string `gorm:"-" json:"raw_key,omitempty"`
}

func (ApiKey) TableName() string { return "api_keys" }
