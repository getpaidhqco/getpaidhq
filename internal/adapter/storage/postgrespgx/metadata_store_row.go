package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// metadataStoreRow is the postgres on-the-wire shape of a MetadataStore. The
// table has a triple primary key (org_id, parent_id, key). parent_type and
// namespace are nullable TEXT columns, so they round-trip through *string with
// strOrEmpty/nilIfEmpty; value is NOT NULL and stays a plain string.
type metadataStoreRow struct {
	OrgId      string
	ParentId   string
	ParentType *string
	Key        string
	Value      string
	Namespace  *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

const metadataStoreColumns = `org_id, parent_id, parent_type, key, value, namespace, created_at, updated_at`

func (r *metadataStoreRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.ParentId, &r.ParentType, &r.Key, &r.Value, &r.Namespace, &r.CreatedAt, &r.UpdatedAt)
}

func (r metadataStoreRow) toDomain() domain.MetadataStore {
	return domain.MetadataStore{
		OrgId:      r.OrgId,
		ParentId:   r.ParentId,
		ParentType: strOrEmpty(r.ParentType),
		Key:        r.Key,
		Value:      r.Value,
		Namespace:  strOrEmpty(r.Namespace),
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}

func metadataStoreRowFromDomain(m domain.MetadataStore) metadataStoreRow {
	return metadataStoreRow{
		OrgId:      m.OrgId,
		ParentId:   m.ParentId,
		ParentType: nilIfEmpty(m.ParentType),
		Key:        m.Key,
		Value:      m.Value,
		Namespace:  nilIfEmpty(m.Namespace),
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}
