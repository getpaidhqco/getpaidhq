package postgresgorm

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// metadataStoreRow is the postgres on-the-wire shape of a MetadataStore.
// Triple primary key (org_id, parent_id, key).
type metadataStoreRow struct {
	OrgId      string    `gorm:"column:org_id;primaryKey"`
	ParentId   string    `gorm:"column:parent_id;primaryKey"`
	ParentType string    `gorm:"column:parent_type"`
	Key        string    `gorm:"column:key;primaryKey"`
	Value      string    `gorm:"column:value"`
	Namespace  string    `gorm:"column:namespace"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (metadataStoreRow) TableName() string { return "metadata_store" }

func (r metadataStoreRow) toDomain() domain.MetadataStore {
	return domain.MetadataStore{
		OrgId:      r.OrgId,
		ParentId:   r.ParentId,
		ParentType: r.ParentType,
		Key:        r.Key,
		Value:      r.Value,
		Namespace:  r.Namespace,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}

func metadataStoreRowFromDomain(m domain.MetadataStore) metadataStoreRow {
	return metadataStoreRow{
		OrgId:      m.OrgId,
		ParentId:   m.ParentId,
		ParentType: m.ParentType,
		Key:        m.Key,
		Value:      m.Value,
		Namespace:  m.Namespace,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}

func metadataStoreRowsToDomain(rows []metadataStoreRow) []domain.MetadataStore {
	out := make([]domain.MetadataStore, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out
}
