package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// apiKeyRow is the postgres on-the-wire shape of an ApiKey. Package-internal.
// key_hash carries the HMAC of the raw key (deduped by a unique index); it is
// NOT NULL. name is a nullable TEXT column, so it is held as *string and
// converted at the boundary — SQL NULL round-trips as the domain's empty
// string.
type apiKeyRow struct {
	OrgId     string
	Id        string
	Name      *string
	KeyHash   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

const apiKeyColumns = `org_id, id, name, key_hash, created_at, updated_at`

func (r *apiKeyRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.Name, &r.KeyHash, &r.CreatedAt, &r.UpdatedAt)
}

func (r apiKeyRow) toDomain() domain.ApiKey {
	return domain.ApiKey{
		OrgId:     r.OrgId,
		Id:        r.Id,
		Name:      strOrEmpty(r.Name),
		KeyHash:   r.KeyHash,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func apiKeyRowFromDomain(k domain.ApiKey) apiKeyRow {
	return apiKeyRow{
		OrgId:     k.OrgId,
		Id:        k.Id,
		Name:      nilIfEmpty(k.Name),
		KeyHash:   k.KeyHash,
		CreatedAt: k.CreatedAt,
		UpdatedAt: k.UpdatedAt,
	}
}
