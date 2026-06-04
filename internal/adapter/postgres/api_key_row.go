package postgres

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// apiKeyRow is the postgres on-the-wire shape of an ApiKey. Package-internal.
type apiKeyRow struct {
	OrgId     string    `gorm:"column:org_id;primaryKey"`
	Id        string    `gorm:"column:id;primaryKey"`
	Name      string    `gorm:"column:name"`
	KeyHash   string    `gorm:"column:key_hash;unique"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (apiKeyRow) TableName() string { return "api_keys" }

func (r apiKeyRow) toDomain() domain.ApiKey {
	return domain.ApiKey{
		OrgId:     r.OrgId,
		Id:        r.Id,
		Name:      r.Name,
		KeyHash:   r.KeyHash,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func apiKeyRowFromDomain(k domain.ApiKey) apiKeyRow {
	return apiKeyRow{
		OrgId:     k.OrgId,
		Id:        k.Id,
		Name:      k.Name,
		KeyHash:   k.KeyHash,
		CreatedAt: k.CreatedAt,
		UpdatedAt: k.UpdatedAt,
	}
}
